# Capability: 策略执行

## Purpose

策略执行能力通过根据匹配的策略操作允许或丢弃网络数据包来执行安全策略决策。它在 Linux TC（Traffic Control）层使用 eBPF 运行，提供线速执行，延迟开销最小。

## Context

策略执行是微隔离系统的执行层。在匹配策略（或应用默认策略）后，执行引擎必须：
- 根据策略操作允许或丢弃数据包
- 为被拒绝的流量生成审计日志
- 保持高性能（无因开销导致的数据包丢弃）
- 提供准确的统计信息

## Requirements

### Requirement: 操作执行

系统必须(SHALL)支持三种策略执行操作：
- **ALLOW**：将数据包传递到其目的地
- **DENY**：立即丢弃数据包
- **LOG**：传递数据包但生成审计事件

#### Scenario: ALLOW 操作执行

**Given** 数据包匹配 action=ALLOW 的策略
**When** eBPF 程序处理数据包
**Then** 程序必须(SHALL)返回 TC_ACT_OK（0）
**And** 数据包必须(SHALL)转发到其目的地
**And** STATS_ALLOWED_PACKETS 计数器必须(SHALL)递增

#### Scenario: DENY 操作执行

**Given** 数据包匹配 action=DENY 的策略
**When** eBPF 程序处理数据包
**Then** 程序必须(SHALL)返回 TC_ACT_SHOT（2）
**And** 数据包必须(SHALL)立即被丢弃
**And** STATS_DENIED_PACKETS 计数器必须(SHALL)递增
**And** 必须(SHALL)通过环形缓冲区向用户空间发送流事件

#### Scenario: LOG 操作执行

**Given** 数据包匹配 action=LOG 的策略
**When** eBPF 程序处理数据包
**Then** 程序必须(SHALL)返回 TC_ACT_OK（0）
**And** 数据包必须(SHALL)转发到其目的地
**And** STATS_ALLOWED_PACKETS 计数器必须(SHALL)递增
**And** 必须(SHALL)向用户空间发送流事件以进行审计日志记录

### Requirement: TC 集成

系统必须(SHALL)附加到 Linux 流量控制（TC）子系统：
- 附加点：TC ingress hook（TCX）
- 网络接口：可配置（例如，eth0、lo）
- 返回码：TC_ACT_OK（允许）或 TC_ACT_SHOT（丢弃）

#### Scenario: TC 程序附加

**Given** 用户空间代理已启动
**When** 调用 DataPlane.LoadAndAttach()
**Then** eBPF 程序必须(SHALL)附加到指定的网络接口
**And** 附加必须(SHALL)使用 TC ingress（TC_INGRESS 方向）
**And** 接口上的所有传入数据包必须(SHALL)被处理

#### Scenario: 非 IP 数据包处理

**Given** 非 IP 数据包到达（例如，ARP、IPv6）
**When** eBPF 程序处理数据包
**Then** 程序必须(SHALL)返回 TC_ACT_OK
**And** 数据包必须(SHALL)无需策略执行即通过

### Requirement: 快速路径 vs. 慢速路径

执行引擎必须(SHALL)针对常见情况进行优化：
- **快速路径**：具有缓存策略决策的现有会话
- **慢速路径**：需要策略查找的新会话

#### Scenario: 快速路径执行（现有会话）

**Given** 流 192.168.1.10:45678 -> 10.0.0.5:443 存在会话
**And** 会话具有缓存的 policy_action=ALLOW
**When** 此流的后续数据包到达
**Then** 执行必须(SHALL)使用缓存的决策（无策略查找）
**And** 处理时间必须(SHALL) < 1 微秒
**And** 数据包必须(SHALL)被允许通过

#### Scenario: 慢速路径执行（新会话）

**Given** 新流不存在会话
**When** 第一个数据包到达
**Then** 必须(SHALL)执行策略查找
**And** 必须(SHALL)使用匹配的操作创建新会话
**And** 必须(SHALL)应用执行决策
**And** 处理时间必须(SHALL) < 3 微秒

### Requirement: 有状态执行

系统必须(SHALL)以有状态的方式执行策略：
- 第一个数据包：策略查找 + 会话创建
- 后续数据包：使用会话中的缓存决策
- 会话状态驱动执行

#### Scenario: 有状态会话执行

**Given** TCP 连接的第一个数据包被允许
**When** 会话已建立
**Then** 此会话的所有后续数据包必须(SHALL)被允许
**即使** 策略稍后被删除
**直到** 会话到期或显式关闭

### Requirement: 数据包丢弃报告

当数据包被丢弃（DENY 操作）时，系统必须(SHALL)：
- 通过环形缓冲区向用户空间报告事件
- 在事件中包含流 5 元组
- 包含时间戳和原因（action=DENY）
- 启用实时安全监控

#### Scenario: 被拒绝流量事件生成

**Given** 数据包匹配 DENY 策略
**When** 数据包被丢弃
**Then** 必须(SHALL)在环形缓冲区中保留 flow_event
**And** 事件必须(SHALL)包含：
  - 流 5 元组（src_ip、dst_ip、src_port、dst_port、protocol）
  - 时间戳（纳秒）
  - action=DENY
  - event_type=NEW_SESSION
**And** 事件必须(SHALL)提交到环形缓冲区供用户空间消费

### Requirement: 执行性能

执行引擎必须(SHALL)满足严格的性能要求：
- 平均数据包处理延迟：< 10 微秒
- 无因 eBPF 处理开销导致的数据包丢弃
- 支持高数据包速率（每个 CPU 核心 100K+ pps）

#### Scenario: 高吞吐量执行

**Given** 系统处于高负载状态（每秒 100,000 个数据包）
**When** 执行引擎处理数据包
**Then** 不得(SHALL NOT)由于 eBPF 处理开销而丢弃数据包
**And** 平均延迟必须(SHALL)保持在 10 微秒以下
**And** 所有策略决策必须(SHALL)准确执行

### Requirement: 性能优化

执行引擎必须(SHALL)实施以下优化：
1. **条件调试**：编译时标志以在生产中禁用调试日志记录
2. **内联热路径**：为已建立的流内联会话更新逻辑
3. **最小环形缓冲区使用**：仅为 DENY 或 LOG 操作发送事件
4. **缓存决策**：为后续数据包使用会话缓存的策略操作

#### Scenario: 调试模式性能影响

**Given** DEBUG_MODE 设置为 0（禁用）
**When** 处理数据包
**Then** 不得(SHALL NOT)执行 bpf_printk() 调用
**And** 延迟必须(SHALL)最小化

**Given** DEBUG_MODE 设置为 1（启用）
**When** 处理数据包
**Then** 调试日志必须(SHALL)写入 trace_pipe
**And** 延迟可能(MAY)由于日志记录开销而增加

### Requirement: 统计集成

执行引擎必须(SHALL)更新统计计数器：
- STATS_TOTAL_PACKETS：每个处理的数据包
- STATS_ALLOWED_PACKETS：每个允许的数据包
- STATS_DENIED_PACKETS：每个被拒绝的数据包
- STATS_NEW_SESSIONS：每个创建的新会话
- STATS_POLICY_HITS：每次策略匹配
- STATS_POLICY_MISSES：每次策略未命中（默认操作）

#### Scenario: 统计计数器更新

**Given** 数据包被处理并允许
**When** 做出执行决策
**Then** STATS_TOTAL_PACKETS 必须(SHALL)递增 1
**And** STATS_ALLOWED_PACKETS 必须(SHALL)递增 1

**Given** 数据包被处理并拒绝
**When** 做出执行决策
**Then** STATS_TOTAL_PACKETS 必须(SHALL)递增 1
**And** STATS_DENIED_PACKETS 必须(SHALL)递增 1

### Requirement: 环形缓冲区事件报告

系统必须(SHALL)使用 eBPF 环形缓冲区进行事件报告：
- 环形缓冲区大小：256 KB
- 事件类型：NEW_SESSION、UPDATE_SESSION、CLOSE_SESSION
- 交付：尽力而为（内核中不阻塞）

#### Scenario: 环形缓冲区容量处理

**Given** 环形缓冲区已满（消耗 256 KB）
**When** 需要报告新事件
**Then** bpf_ringbuf_reserve() 必须(SHALL)失败
**And** 事件必须(SHALL)被丢弃
**And** 数据包处理必须(SHALL)继续
**And** 不得(SHALL NOT)发生内核恐慌或死锁

#### Scenario: 用户空间事件消费

**Given** 用户空间代理正在运行
**When** 流事件发送到环形缓冲区
**Then** 代理必须(SHALL)通过 MonitorFlowEvents() 读取事件
**And** 事件必须(SHALL)记录或处理以进行分析
**And** 事件必须(SHALL)足够快地消费以防止缓冲区溢出

### Requirement: 优雅降级

系统必须(SHALL)优雅地处理错误条件：
- 环形缓冲区已满：丢弃事件，继续处理
- 映射查找失败：应用默认操作
- 内存分配失败：跳过事件报告

#### Scenario: 映射查找失败处理

**Given** 会话映射查找失败的罕见情况
**When** eBPF 程序尝试查找会话
**Then** 程序必须(SHALL)将其视为新会话
**And** 执行策略查找
**And** 继续处理（无数据包丢弃）

## Data Structures

### 流事件结构

```c
struct flow_event {
    struct flow_key key;  // 流 5 元组
    __u64 timestamp;      // 事件时间戳（纳秒）
    __u64 packets;        // 事件时的数据包计数
    __u64 bytes;          // 事件时的字节计数
    __u8  action;         // 策略操作（ALLOW/DENY/LOG）
    __u8  event_type;     // 事件类型（NEW/UPDATE/CLOSE）
    __u16 pad;            // 填充
} __attribute__((packed));
```

### TC 操作码

```c
#define TC_ACT_OK   0  // 允许数据包（通过）
#define TC_ACT_SHOT 2  // 丢弃数据包（阻止）
```

## Implementation Notes

- eBPF 程序使用 TCX（现代 TC 附加方法）附加到 TC ingress hook
- 非 IP 数据包（ARP、IPv6 等）无需处理即通过
- ICMP 和其他没有端口的协议使用 src_port=0、dst_port=0
- 环形缓冲区事件是尽力而为的（不会阻塞数据包处理）
- 调试日志记录（bpf_printk）有条件地编译以实现生产性能

## Performance Characteristics

- **热路径延迟（现有会话）**：< 1 微秒
- **冷路径延迟（新会话）**：< 3 微秒
- **目标平均延迟**：< 10 微秒
- **吞吐量**：每个 CPU 核心 100K+ 数据包/秒
- **数据包丢弃率**：0%（正常负载下）

## 性能优化详情

### 热路径（>99% 的数据包）
1. 会话查找：~0.3 μs
2. 缓存操作检查：~0.1 μs
3. 会话统计更新：~0.2 μs
4. 返回决策：~0.1 μs
**总计：~0.7 μs**

### 冷路径（<1% 的数据包）
1. 会话查找（未命中）：~0.3 μs
2. 策略查找：~0.5 μs
3. 创建会话：~1.0 μs
4. 环形缓冲区事件：~0.5 μs（如果 DENY/LOG）
5. 返回决策：~0.1 μs
**总计：~2.4 μs**

## Related Capabilities

- **Policy Matching**：提供要执行的策略操作
- **Session Tracking**：为快速路径缓存策略决策
- **Statistics Reporting**：收集执行统计信息
- **Dataplane Performance**：所有优化旨在最小化执行延迟


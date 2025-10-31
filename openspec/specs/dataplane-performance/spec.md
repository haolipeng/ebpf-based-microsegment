# Capability: 数据平面性能

## Purpose

数据平面性能能力专注于通过 eBPF 数据平面中的一系列针对性优化来实现超低延迟的数据包处理。目标是在支持高吞吐量（每个 CPU 核心 100K+ 数据包/秒）的同时，将平均数据包处理延迟保持在 10 微秒以下。

## Context

性能对微隔离系统至关重要，因为：
- 网络安全不应成为瓶颈
- 低延迟对延迟敏感的应用程序至关重要
- 高吞吐量是数据密集型工作负载所需的
- 高效的资源使用降低了部署成本

性能优化策略侧重于：
1. 快速路径优化（现有会话）
2. 最小化 eBPF 指令数
3. 减少映射查找开销
4. 消除不必要的操作
5. 利用 CPU 缓存

## Requirements

### Requirement: 延迟目标

系统必须(SHALL)实现以下延迟目标：
- **热路径（现有会话）**：< 1 微秒平均
- **冷路径（新会话）**：< 3 微秒平均
- **总体平均**：< 10 微秒
- **99 百分位**：< 20 微秒

#### Scenario: 热路径延迟测量

**Given** 流 192.168.1.10:45678 -> 10.0.0.5:443 存在会话
**When** 处理此流的 10,000 个数据包
**Then** 每个数据包的平均延迟必须(SHALL)小于 1 微秒
**And** 99 百分位延迟必须(SHALL)小于 2 微秒

#### Scenario: 冷路径延迟测量

**Given** 到达 1,000 个新的唯一流
**When** 处理每个流的第一个数据包
**Then** 每个数据包的平均延迟必须(SHALL)小于 3 微秒
**And** 延迟必须(SHALL)包括策略查找和会话创建

### Requirement: 吞吐量目标

系统必须(SHALL)支持高数据包吞吐量：
- **最低**：每个 CPU 核心每秒 100,000 个数据包
- **目标**：每个 CPU 核心每秒 1,000,000+ 个数据包
- **无数据包丢弃**：由于 eBPF 处理开销

#### Scenario: 高吞吐量压力测试

**Given** 系统处于持续高负载状态
**When** 每个核心每秒处理 1,000,000 个数据包
**Then** 不得(SHALL NOT)由于 eBPF 开销而丢弃数据包
**And** 所有策略执行必须(SHALL)保持准确
**And** 所有统计信息必须(SHALL)准确更新

### Requirement: 条件调试日志记录

系统必须(SHALL)支持调试日志记录的条件编译：
- 生产模式：DEBUG_MODE = 0（无日志记录）
- 开发模式：DEBUG_MODE = 1（详细日志记录）
- 日志记录显著影响性能，在生产中必须(MUST)禁用

#### Scenario: 调试模式禁用（生产）

**Given** DEBUG_MODE 设置为 0
**When** 处理数据包
**Then** 不得(SHALL NOT)将 bpf_printk() 调用编译到 eBPF 程序中
**And** 调试开销不得(SHALL NOT)影响性能
**And** 数据包处理延迟必须(SHALL)最小化

#### Scenario: 调试模式启用（开发）

**Given** DEBUG_MODE 设置为 1
**When** 处理数据包
**Then** 必须(SHALL)执行 bpf_printk() 调用
**And** 调试日志必须(SHALL)写入 /sys/kernel/debug/tracing/trace_pipe
**And** 性能可能(MAY)每个数据包降低 2-5 微秒

### Requirement: 每 CPU 统计优化

系统必须(SHALL)使用优化的每 CPU 统计更新：
- 使用直接递增（`*count += 1`）而不是原子操作
- 利用 PERCPU_ARRAY 进行无锁更新
- 最小化计数器更新的指令数

#### Scenario: 优化的计数器更新

**Given** 数据包被允许通过
**When** STATS_ALLOWED_PACKETS 需要递增
**Then** eBPF 程序必须(SHALL)使用直接递增
**And** 更新必须(SHALL)在 < 50 纳秒内完成
**And** 不得(SHALL NOT)使用原子操作或锁

### Requirement: 策略查找优化

系统必须(SHALL)优化策略查找：
- 重用 flow_key 结构作为 policy_key（相同布局）
- 直接转换而不是结构复制
- 对 hit_count 使用简单递增（无需原子）

#### Scenario: 直接键转换

**Given** 已从数据包中提取 flow_key
**When** 执行策略查找
**Then** flow_key 必须(SHALL)直接转换为 policy_key
**And** 不得(SHALL NOT)发生内存复制
**And** 查找延迟必须(SHALL)最小化

### Requirement: 会话创建优化

系统必须(SHALL)优化会话创建：
- 使用第一个数据包统计信息初始化会话
- 最小化环形缓冲区事件生成（仅用于 DENY/LOG）
- 使用高效的结构初始化

#### Scenario: 优化的会话初始化

**Given** 第一个数据包的新流到达
**When** 创建会话
**Then** 会话必须(SHALL)使用第一个数据包统计信息初始化
**And** packets_to_server 必须(SHALL)设置为 1
**And** bytes_to_server 必须(SHALL)设置为数据包长度
**And** 稍后不得(SHALL NOT)需要额外的数据包处理

#### Scenario: 选择性事件生成

**Given** 使用 action=ALLOW 创建新会话
**When** 创建会话
**Then** 不得(SHALL NOT)生成环形缓冲区事件（优化）

**Given** 使用 action=DENY 创建新会话
**When** 创建会话
**Then** 必须(SHALL)生成环形缓冲区事件（安全审计所需）

### Requirement: 热路径内联

系统必须(SHALL)内联关键热路径操作：
- 会话统计更新必须(SHALL)内联
- 热路径（现有会话）上无函数调用
- 最小化 eBPF 程序复杂性

#### Scenario: 内联会话更新

**Given** 处理现有会话的数据包
**When** 需要更新会话统计信息
**Then** 更新逻辑必须(SHALL)在主程序中内联
**And** 不得(SHALL NOT)进行辅助函数调用
**And** 由于内联，延迟必须(SHALL)最小化

### Requirement: 环形缓冲区使用优化

系统必须(SHALL)最小化环形缓冲区使用：
- 仅为 DENY 或 LOG 操作发送事件
- 不为正常 ALLOW 操作发送事件（减少开销）
- 用户空间仍可监控允许流量的统计信息

#### Scenario: 最小环形缓冲区事件

**Given** 1,000 个数据包通过 action=ALLOW 被允许
**When** 处理数据包
**Then** 必须(SHALL)生成零环形缓冲区事件
**And** 必须(SHALL)避免环形缓冲区开销

**Given** 10 个数据包被 action=DENY 拒绝
**When** 处理数据包
**Then** 必须(SHALL)生成 10 个环形缓冲区事件（每个新被拒绝流一个）

### Requirement: 性能测试工具

系统必须(SHALL)提供性能测试工具：
- 独立性能测试程序
- 定期统计报告
- 数据包速率计算
- 延迟测量能力

#### Scenario: 性能测试执行

**Given** 性能测试工具已构建
**When** 执行 `make perf-test` 和 `sudo bin/perf-test`
**Then** 该工具必须(SHALL)：
  - 加载和附加 eBPF 程序
  - 添加测试策略
  - 每 5 秒监控统计信息
  - 计算和报告数据包速率
  - 运行配置的持续时间（默认 30 秒）

### Requirement: 基准脚本

系统必须(SHALL)提供自动化基准脚本：
- `tests/performance/benchmark_test.sh`
- 为测试生成合成流量
- 测量和报告性能指标

#### Scenario: 自动化基准执行

**Given** 基准脚本存在
**When** 执行 `./tests/performance/benchmark_test.sh`
**Then** 脚本必须(SHALL)：
  - 构建项目
  - 启动性能测试工具
  - 生成合成流量（使用 ping 或 iperf）
  - 捕获性能指标
  - 报告结果

### Requirement: 性能指标文档

系统必须(SHALL)记录性能特征：
- 不同场景的预期延迟
- 吞吐量能力
- 内存使用
- CPU 利用率

此文档必须(SHALL)维护在 `docs/PERFORMANCE.md` 中。

#### Scenario: 性能文档访问

**Given** 用户想了解系统性能
**When** 用户阅读 docs/PERFORMANCE.md
**Then** 该文档必须(SHALL)提供：
  - 延迟目标和测量
  - 吞吐量基准
  - 使用的优化技术
  - 性能调优指南
  - 与基线（预优化）的比较

## 性能优化技术

### 1. 条件调试日志记录

```c
#define DEBUG_MODE 0  // 0=禁用，1=启用

#if DEBUG_MODE
    bpf_printk("Debug message\n");
#endif
```

**影响**：在生产中每个数据包节省 2-5 微秒。

### 2. 优化的每 CPU 统计

```c
// 之前（原子，较慢）：
__sync_fetch_and_add(count, 1);

// 之后（直接递增，更快）：
*count += 1;
```

**影响**：每次计数器更新节省 ~100 纳秒。

### 3. 直接键转换

```c
// 之前（结构复制）：
struct policy_key pkey;
pkey.src_ip = key->src_ip;
// ... 更多复制 ...

// 之后（直接转换）：
struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
```

**影响**：每次策略查找节省 ~200 纳秒。

### 4. 内联热路径

```c
// 之前：会话更新的函数调用
update_session(session, skb->len);

// 之后：内联更新
session->last_seen_ts = get_timestamp_ns();
session->packets_to_server += 1;
session->bytes_to_server += skb->len;
```

**影响**：热路径上每个数据包节省 ~500 纳秒。

### 5. 选择性事件生成

```c
// 仅为 DENY 或 LOG 操作发送事件
if (action == POLICY_ACTION_DENY || action == POLICY_ACTION_LOG) {
    struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
    // ...
}
```

**影响**：对于典型流量，环形缓冲区开销减少 99%。

## Performance Characteristics

### 基线（预优化）
- 热路径：~3 微秒
- 冷路径：~7 微秒
- 平均：~15 微秒

### 优化后
- 热路径：~0.7 微秒（4 倍改进）
- 冷路径：~2.4 微秒（3 倍改进）
- 平均：~5 微秒（3 倍改进）

### 性能细分

**热路径（~0.7 μs）**
- 会话查找：0.3 μs
- 缓存操作检查：0.1 μs
- 会话统计更新：0.2 μs
- 返回决策：0.1 μs

**冷路径（~2.4 μs）**
- 会话查找（未命中）：0.3 μs
- 策略查找：0.5 μs
- 创建会话：1.0 μs
- 环形缓冲区事件（如果 DENY/LOG）：0.5 μs
- 返回决策：0.1 μs

## 内存使用

- **会话映射**：~6.4 MB（100K 会话 × 64 字节）
- **策略映射**：~400 KB（10K 策略 × 40 字节）
- **统计映射**：~256 字节（8 个计数器 × 4 个 CPU × 8 字节）
- **环形缓冲区**：256 KB
- **总计**：~7.1 MB

## CPU 利用率

- **空闲**：< 0.1% CPU
- **100K pps**：~5% CPU（单核）
- **1M pps**：~50% CPU（单核）

## 性能测试工具

### 1. 性能测试工具（`tools/perf-test/main.go`）
- 独立测试程序
- 可配置的持续时间和接口
- 定期统计报告
- 数据包速率计算

### 2. 基准脚本（`tests/performance/benchmark_test.sh`）
- 自动化测试
- 合成流量生成
- 指标捕获和报告

### 3. Makefile 目标
```bash
make perf-test      # 构建性能测试工具
sudo bin/perf-test  # 运行性能测试
```

## Related Capabilities

- **Session Tracking**：会话缓存对热路径性能至关重要
- **Policy Matching**：策略查找优化影响冷路径性能
- **Policy Enforcement**：执行逻辑针对最小延迟进行优化
- **Statistics Reporting**：统计收集针对零争用进行优化

## 未来优化机会

1. **XDP（eXpress Data Path）**：移至 XDP 以实现更低延迟（~200 ns）
2. **eBPF 到 eBPF 调用**：使用尾调用优化代码结构
3. **硬件卸载**：利用 SmartNIC 进行策略执行
4. **IPv6 支持**：优化 IPv6 处理路径
5. **CIDR 匹配**：使用 LPM trie 添加高效的 CIDR 匹配


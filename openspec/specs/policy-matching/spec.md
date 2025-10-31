# Capability: 策略匹配

## Purpose

策略匹配能力使用 eBPF 哈希映射提供高性能的网络安全策略查找和匹配。它支持基于 5 元组的精确匹配，具有 O(1) 查找复杂度，实现线速策略执行以进行微隔离。

## Context

策略匹配是微隔离系统的核心决策组件。它根据网络流的 5 元组特征确定应对其应用哪个操作（ALLOW、DENY、LOG）。匹配引擎必须是：
- 极快（< 1 微秒查找时间）
- 可扩展（支持 10,000+ 策略）
- 准确（精确的 5 元组匹配）
- 高效（最小的内存占用）

## Requirements

### Requirement: 5 元组精确匹配

系统必须(SHALL)支持基于 5 元组的精确匹配：
- 源 IP 地址（IPv4）
- 目标 IP 地址（IPv4）
- 源端口（0-65535）
- 目标端口（0-65535）
- 协议（TCP、UDP、ICMP、ANY）

#### Scenario: SSH 流量的精确匹配

**Given** 存在策略：src=192.168.1.0、dst=10.0.0.5、sport=ANY、dport=22、proto=TCP
**When** 到达数据包：src=192.168.1.100、dst=10.0.0.5、sport=45678、dport=22、proto=TCP
**Then** 如果 src 和 dst 完全匹配（或是通配符），策略查找必须(SHALL)匹配

#### Scenario: 协议特定匹配

**Given** 存在策略：dst=10.0.0.10、dport=80、proto=TCP，action=ALLOW
**And** 存在另一个策略：dst=10.0.0.10、dport=80、proto=UDP，action=DENY
**When** TCP 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)匹配 TCP 策略（ALLOW）
**When** UDP 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)匹配 UDP 策略（DENY）

### Requirement: 策略存储

系统必须(SHALL)在具有以下特征的 eBPF HASH 映射中存储策略：
- 最大条目数：10,000 个策略
- 键：策略 5 元组（struct policy_key）
- 值：策略操作和元数据（struct policy_value）
- O(1) 查找复杂度

#### Scenario: 策略映射容量

**Given** 策略映射的容量为 10,000 个条目
**When** 系统尝试添加第 10,001 个策略
**Then** 操作必须(SHALL)失败并返回错误
**And** 用户空间代理必须(SHALL)收到通知

#### Scenario: 策略插入

**Given** 具有 5 元组和操作的新策略规则
**When** 用户空间代理调用 AddPolicy()
**Then** 策略必须(SHALL)插入到 eBPF 策略映射中
**And** 策略必须(SHALL)立即对新流生效

### Requirement: 策略操作

系统必须(SHALL)支持三种策略操作：
- **ALLOW**：允许流量（TC_ACT_OK）
- **DENY**：丢弃流量（TC_ACT_SHOT）
- **LOG**：允许流量但生成审计日志

#### Scenario: ALLOW 策略应用

**Given** dst=10.0.0.5:443、proto=TCP 的策略，action=ALLOW
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被允许通过（TC_ACT_OK）
**And** 统计计数器 STATS_ALLOWED_PACKETS 必须(SHALL)增加

#### Scenario: DENY 策略应用

**Given** dst=192.168.1.100:22、proto=TCP 的策略，action=DENY
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被丢弃（TC_ACT_SHOT）
**And** 统计计数器 STATS_DENIED_PACKETS 必须(SHALL)增加
**And** 必须(SHALL)通过环形缓冲区向用户空间发送流事件

#### Scenario: LOG 策略应用

**Given** dst=10.0.0.10:8080、proto=TCP 的策略，action=LOG
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被允许通过（TC_ACT_OK）
**And** 必须(SHALL)向用户空间发送流事件以进行审计日志记录

### Requirement: 策略优先级

每个策略必须(SHALL)有一个优先级字段（0-65535，数字越小优先级越高）。
- 目前用于元数据和未来的规则排序
- 在当前实现（精确匹配）中，第一个匹配获胜

#### Scenario: 策略优先级元数据

**Given** 两个策略具有相同的 5 元组但不同的优先级
**When** 插入策略
**Then** 系统必须(SHALL)执行精确匹配语义
**And** 优先级必须(SHALL)可用于未来的分层匹配

### Requirement: 策略命中计数

系统必须(SHALL)为每个策略维护命中计数器以跟踪使用情况：
- 每次策略匹配时计数器递增
- 计数器存储在 policy_value.hit_count 中
- 用户空间可以查询命中计数以进行分析

#### Scenario: 策略命中计数器更新

**Given** 策略的 hit_count = 100
**When** 匹配的数据包到达
**Then** hit_count 必须(SHALL)递增到 101
**And** STATS_POLICY_HITS 全局计数器必须(SHALL)递增

#### Scenario: 策略使用分析

**Given** 安装了多个策略
**When** 用户空间代理查询策略统计信息
**Then** 它必须(SHALL)能够识别：
- 最常匹配的策略（高 hit_count）
- 未使用的策略（hit_count = 0）
- 随时间推移的策略有效性

### Requirement: 策略查找性能

系统必须(SHALL)提供高性能的策略查找：
- 平均查找延迟：< 1 微秒
- 哈希映射 O(1) 复杂度
- 最小的 CPU 开销
- 读取无锁

#### Scenario: 热路径策略查找

**Given** 包含 10,000 个条目的策略映射
**When** 新流需要策略查找
**Then** 查找必须(SHALL)在 < 1 微秒内完成
**And** 不得(SHALL NOT)由于查找开销而发生数据包丢弃

### Requirement: 新会话的策略匹配

系统必须(SHALL)仅对新会话执行策略查找：
- 流的第一个数据包触发策略查找
- 匹配的策略操作缓存在会话状态中
- 后续数据包使用缓存的决策（无需查找）

此优化显著提高了已建立流的性能。

#### Scenario: 策略查找优化

**Given** 新流到达
**When** 处理第一个数据包
**Then** 必须(SHALL)执行策略查找
**And** 匹配的操作必须(SHALL)存储在 session_value.policy_action 中

**When** 同一流的后续数据包到达
**Then** 不得(SHALL NOT)执行策略查找
**And** 必须(SHALL)直接使用缓存的 policy_action

### Requirement: 策略 CRUD 操作

用户空间代理必须(SHALL)支持对策略的完整 CRUD 操作：
- **Create**：向 eBPF 映射添加新策略
- **Read**：从 eBPF 映射列出所有策略
- **Update**：修改现有策略
- **Delete**：从 eBPF 映射删除策略

#### Scenario: 通过 API 添加策略

**Given** 用户空间代理正在运行
**When** 通过 PolicyManager.AddPolicy() 添加新策略
**Then** 策略必须(SHALL)插入到 eBPF policy_map 中
**And** 操作必须(SHALL)返回成功

#### Scenario: 删除策略

**Given** 存在具有特定 5 元组的策略
**When** 使用匹配的 5 元组调用 PolicyManager.DeletePolicy()
**Then** 策略必须(SHALL)从 eBPF policy_map 中删除
**And** 未来的流必须(SHALL NOT)匹配此策略

#### Scenario: 列出所有策略

**Given** 策略映射中存在多个策略
**When** 调用 PolicyManager.ListPolicies()
**Then** 必须(SHALL)返回所有策略的列表
**And** 每个策略必须(SHALL)包括其 5 元组、操作、优先级和 hit_count

### Requirement: 默认策略行为

系统必须(SHALL)在没有策略匹配时应用默认操作：
- 默认操作：ALLOW（开发期间为安全起见而失败打开）
- 可配置用于生产部署（通常失败关闭）

#### Scenario: 无匹配策略

**Given** 特定 5 元组不存在策略
**When** 具有该 5 元组的数据包到达
**Then** 必须(SHALL)应用默认操作（ALLOW）
**And** STATS_POLICY_MISSES 计数器必须(SHALL)递增
**And** 必须(SHALL)创建会话，policy_action=ALLOW

### Requirement: 策略规则 ID 跟踪

每个策略必须(SHALL)有一个唯一的 rule_id 用于跟踪和审计：
- 用户分配的 32 位无符号整数
- 用于日志和事件中的关联
- 帮助识别哪个策略触发了操作

#### Scenario: 事件中的策略规则 ID

**Given** rule_id=1001 和 action=DENY 的策略
**When** 匹配的数据包到达
**Then** 发送到用户空间的流事件必须(SHALL)包含 rule_id=1001
**And** 审计日志必须(SHALL)引用 rule_id=1001

## Data Structures

### 策略键结构

```c
struct policy_key {
    __u32 src_ip;      // 源 IP 地址（ANY 为 0.0.0.0）
    __u32 dst_ip;      // 目标 IP 地址
    __u16 src_port;    // 源端口（ANY 为 0）
    __u16 dst_port;    // 目标端口
    __u8  protocol;    // 协议（6=TCP、17=UDP、0=ANY）
    __u8  pad[3];      // 对齐填充
} __attribute__((packed));
```

### 策略值结构

```c
struct policy_value {
    __u8  action;        // 策略操作（ALLOW/DENY/LOG）
    __u8  log_enabled;   // 启用日志记录（0=禁用、1=启用）
    __u16 priority;      // 策略优先级（数字越小 = 优先级越高）
    __u32 rule_id;       // 用于跟踪的规则 ID
    __u64 hit_count;     // 策略匹配次数
};
```

### 策略操作枚举

```c
enum policy_action {
    POLICY_ACTION_ALLOW = 0,
    POLICY_ACTION_DENY  = 1,
    POLICY_ACTION_LOG   = 2,
};
```

## Implementation Notes

- 策略映射实现为 `BPF_MAP_TYPE_HASH`（标准哈希映射）
- 最大容量：10,000 个策略（可通过 `MAX_ENTRIES_POLICY` 配置）
- 策略匹配仅在第一个数据包（新会话）时发生
- 后续数据包使用 session_value 中的缓存决策
- 当前实现仅支持 IPv4；计划支持 IPv6
- 仅精确匹配；计划在未来版本中支持 CIDR/范围匹配

## Performance Characteristics

- **查找延迟**：< 1 微秒（平均）
- **查找复杂度**：O(1) 哈希映射查找
- **每个策略的内存**：40 字节
- **总内存**：~400 KB（10,000 个策略）
- **并发访问**：读取无锁

## 性能优化策略

策略匹配引擎针对常见情况进行了优化：
1. **第一个数据包**：完整策略查找（较慢路径）
2. **后续数据包**：会话中的缓存决策（快速路径）

这意味着：
- 新会话创建：~2-3 微秒（包括策略查找）
- 已建立会话处理：< 1 微秒（无策略查找）

对于具有长期连接的典型工作负载，>99% 的数据包使用快速路径。

## Related Capabilities

- **Session Tracking**：在会话状态中缓存策略决策
- **Policy Enforcement**：执行匹配的策略操作
- **Statistics Reporting**：跟踪策略命中和未命中
- **Dataplane Performance**：策略缓存对性能至关重要


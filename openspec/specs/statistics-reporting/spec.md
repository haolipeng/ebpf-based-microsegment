# Capability: 统计报告

## Purpose

统计报告能力通过收集、聚合和公开性能和操作指标来提供对微隔离系统操作的全面可见性。它使用 eBPF 每 CPU 数组在内核空间进行无锁统计收集，并提供用户空间 API 用于查询聚合指标。

## Context

统计信息对以下方面至关重要：
- 性能监控和故障排查
- 安全审计和合规性
- 容量规划和资源优化
- 检测异常和攻击
- 验证策略有效性

统计系统必须是：
- 高性能（对数据包处理无影响）
- 准确（无丢失的计数器更新）
- 全面（涵盖所有关键指标）
- 高效（最小的内存占用）

## Requirements

### Requirement: 核心统计指标

系统必须(SHALL)收集以下核心指标：
- **STATS_TOTAL_PACKETS**：处理的总数据包数
- **STATS_ALLOWED_PACKETS**：被策略允许的数据包数
- **STATS_DENIED_PACKETS**：被策略拒绝的数据包数
- **STATS_NEW_SESSIONS**：创建的新会话数
- **STATS_CLOSED_SESSIONS**：关闭/过期的会话数
- **STATS_ACTIVE_SESSIONS**：当前活跃的会话数
- **STATS_POLICY_HITS**：策略匹配数
- **STATS_POLICY_MISSES**：无策略匹配（默认操作）

#### Scenario: 数据包处理统计

**Given** eBPF 程序处理 1000 个数据包
**And** 950 个数据包被允许
**And** 50 个数据包被拒绝
**When** 用户空间查询统计信息
**Then** STATS_TOTAL_PACKETS 必须(SHALL)等于 1000
**And** STATS_ALLOWED_PACKETS 必须(SHALL)等于 950
**And** STATS_DENIED_PACKETS 必须(SHALL)等于 50

#### Scenario: 会话生命周期统计

**Given** 建立了 100 个新的 TCP 连接
**And** 关闭了 30 个连接
**When** 用户空间查询统计信息
**Then** STATS_NEW_SESSIONS 必须(SHALL)等于 100
**And** STATS_CLOSED_SESSIONS 必须(SHALL)等于 30
**And** STATS_ACTIVE_SESSIONS 必须(SHALL)等于 70

#### Scenario: 策略有效性统计

**Given** 800 个流匹配显式策略
**And** 200 个流使用默认策略
**When** 用户空间查询统计信息
**Then** STATS_POLICY_HITS 必须(SHALL)等于 800
**And** STATS_POLICY_MISSES 必须(SHALL)等于 200

### Requirement: 每 CPU 统计收集

系统必须(SHALL)为统计信息使用 eBPF PERCPU_ARRAY 映射：
- 无锁计数器更新（无需原子操作）
- CPU 核心之间无争用
- 用户空间聚合每 CPU 值

#### Scenario: 多 CPU 计数器更新

**Given** 系统有 4 个 CPU 核心
**And** 每个核心处理 1000 个数据包
**When** 并行处理数据包
**Then** 不得(SHALL NOT)发生锁争用
**And** 每个 CPU 核心必须(SHALL)独立更新其自己的计数器
**And** 总 STATS_TOTAL_PACKETS（聚合）必须(SHALL)等于 4000

### Requirement: 无锁计数器更新

系统必须(SHALL)为每 CPU 计数器使用直接递增操作：
- 无需原子操作（__sync_fetch_and_add）
- 简单递增：`*count += 1`
- 最大性能，最小延迟

#### Scenario: 热路径计数器更新性能

**Given** 在热路径上处理数据包
**When** STATS_ALLOWED_PACKETS 需要更新
**Then** 更新必须(SHALL)使用简单递增操作
**And** 更新必须(SHALL)在 < 50 纳秒内完成
**And** 不得(SHALL NOT)发生 CPU 缓存争用

### Requirement: 用户空间统计聚合

用户空间代理必须(SHALL)聚合每 CPU 统计信息：
- 读取每个计数器的所有每 CPU 值
- 求和值以获得总数
- 通过 API 提供聚合值

#### Scenario: 统计查询 API

**Given** 用户空间代理正在运行
**When** 调用 DataPlane.GetStatistics()
**Then** 该方法必须(SHALL)返回 Statistics 结构
**And** 每个字段必须(SHALL)包含所有 CPU 的聚合值

#### Scenario: 每 CPU 统计聚合

**Given** STATS_TOTAL_PACKETS 在 4 个 CPU 上的值为 [250, 300, 275, 275]
**When** 用户空间读取和聚合统计信息
**Then** 返回的 STATS_TOTAL_PACKETS 必须(SHALL)等于 1100（所有 CPU 值的总和）

### Requirement: 实时统计更新

统计信息必须(SHALL)实时更新：
- 计数器更新在数据包处理期间发生
- 无批处理或延迟更新
- 用户空间可以随时查询当前值

#### Scenario: 实时监控

**Given** 监控工具每秒查询统计信息
**When** 持续处理数据包
**Then** 每次查询必须(SHALL)返回最新的计数器值
**And** 这些值必须(SHALL)反映到那时为止处理的所有数据包

### Requirement: 统计查询性能

用户空间统计查询必须(SHALL)高效：
- 查询延迟：< 1 毫秒
- 无阻塞数据包处理
- 对 eBPF 映射的只读访问

#### Scenario: 负载下的统计查询

**Given** 系统每秒处理 100,000 个数据包
**When** 用户空间查询统计信息
**Then** 查询必须(SHALL)在 < 1 毫秒内完成
**And** 不得(SHALL NOT)阻塞或延迟任何数据包处理
**And** 返回的统计信息必须(SHALL)准确

### Requirement: 统计重置

系统必须(SHALL)支持为测试和基准测试目的重置统计信息：
- 将所有计数器清零
- 用于测试和基准测试
- 生产使用的可选功能

#### Scenario: 为基准测试重置统计

**Given** 系统已处理数据包并累积了统计信息
**When** 基准测试需要从干净的计数器开始
**Then** 用户空间代理可以(MAY)将所有统计信息重置为零
**And** 后续数据包处理必须(SHALL)从零开始递增

### Requirement: 统计导出

用户空间代理必须(SHALL)通过以下方式公开统计信息：
- 直接 API（GetStatistics() 方法）
- 日志记录（定期统计转储）
- 未来：Prometheus/Grafana 指标导出

#### Scenario: 定期统计日志记录

**Given** 用户空间代理配置为每 5 秒记录统计信息
**When** 5 秒过去
**Then** 代理必须(SHALL)将所有统计信息记录到 stdout 或日志文件
**And** 日志条目必须(SHALL)包括：
  - 时间戳
  - 所有计数器值
  - 计算的速率（每秒数据包数）

### Requirement: 派生指标计算

用户空间代理必须(SHALL)计算派生指标：
- **数据包速率**：每秒数据包数（增量/时间间隔）
- **拒绝率**：被拒绝数据包的百分比
- **策略命中率**：策略命中的百分比
- **平均会话持续时间**：基于新会话 vs. 关闭会话

#### Scenario: 数据包速率计算

**Given** STATS_TOTAL_PACKETS 在 5 秒内从 1000 增加到 2000
**When** 用户空间计算数据包速率
**Then** 速率必须(SHALL)为（2000 - 1000）/ 5 = 200 数据包/秒

#### Scenario: 拒绝率计算

**Given** STATS_TOTAL_PACKETS = 1000
**And** STATS_DENIED_PACKETS = 50
**When** 用户空间计算拒绝率
**Then** 拒绝率必须(SHALL)为（50 / 1000）* 100 = 5%

### Requirement: 统计持久化

系统必须(SHALL)仅在内核内存中维护统计信息，具有以下特征：
- 卸载 eBPF 程序时计数器重置
- 用户空间代理可以将历史数据持久化到时间序列数据库
- 无内核端持久化

#### Scenario: eBPF 程序重新加载

**Given** 系统已累积统计信息
**When** 卸载并重新加载 eBPF 程序
**Then** 所有统计计数器必须(SHALL)重置为零
**And** 用户空间代理可能(MAY)在重新加载之前保存了历史数据

### Requirement: 统计映射结构

统计映射必须(SHALL)定义为：
- 类型：BPF_MAP_TYPE_PERCPU_ARRAY
- 最大条目数：STATS_MAX（当前 8 个计数器）
- 键：枚举计数器 ID（0-7）
- 值：64 位无符号整数

#### Scenario: 统计映射查找

**Given** 用户空间代理想要读取 STATS_TOTAL_PACKETS（key=0）
**When** 它使用 key=0 执行映射查找
**Then** 它必须(SHALL)接收 uint64 值数组（每个 CPU 核心一个）
**And** 它必须(SHALL)将这些值相加以获得总数

## Data Structures

### 统计枚举

```c
enum stats_key {
    STATS_TOTAL_PACKETS = 0,
    STATS_ALLOWED_PACKETS,
    STATS_DENIED_PACKETS,
    STATS_NEW_SESSIONS,
    STATS_CLOSED_SESSIONS,
    STATS_ACTIVE_SESSIONS,
    STATS_POLICY_HITS,
    STATS_POLICY_MISSES,
    STATS_MAX,  // 必须最后（定义数组大小）
};
```

### 用户空间统计结构

```go
type Statistics struct {
    TotalPackets   uint64
    AllowedPackets uint64
    DeniedPackets  uint64
    NewSessions    uint64
    ClosedSessions uint64
    ActiveSessions uint64
    PolicyHits     uint64
    PolicyMisses   uint64
}
```

### eBPF 映射定义

```c
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");
```

## Implementation Notes

- 统计更新使用直接递增（`*count += 1`）以获得最大性能
- 每 CPU 数组消除 CPU 核心之间的锁争用
- 用户空间聚合仅在查询统计信息时发生
- 统计信息在数据包处理期间内联更新
- 计数器更新无错误检查（为速度优化）

## Performance Characteristics

- **计数器更新延迟**：< 50 纳秒
- **用户空间查询延迟**：< 1 毫秒
- **每个计数器的内存**：8 字节 * CPU 核心数
- **总内存**：~256 字节（8 个计数器 * 4 个 CPU * 8 字节）
- **对数据包处理的开销**：< 0.5%

## 性能优化详情

统计系统设计为零争用更新：

1. **每 CPU 数组**：每个 CPU 核心都有自己的计数器副本
2. **无锁定**：无原子的直接递增
3. **无同步**：更新在每个 CPU 上独立
4. **延迟聚合**：仅在查询时求和（不在更新时）

此设计确保统计收集对数据包处理性能的影响可以忽略不计。

## 统计使用示例

### 示例：实时监控

```go
stats := dp.GetStatistics()
fmt.Printf("Packets: %d (Allowed: %d, Denied: %d)\n",
    stats.TotalPackets,
    stats.AllowedPackets,
    stats.DeniedPackets)
```

### 示例：速率计算

```go
prevStats := dp.GetStatistics()
time.Sleep(5 * time.Second)
currStats := dp.GetStatistics()

deltaPackets := currStats.TotalPackets - prevStats.TotalPackets
pps := float64(deltaPackets) / 5.0
fmt.Printf("Packet rate: %.2f pps\n", pps)
```

### 示例：策略有效性

```go
stats := dp.GetStatistics()
hitRate := float64(stats.PolicyHits) / float64(stats.PolicyHits + stats.PolicyMisses) * 100
fmt.Printf("Policy hit rate: %.2f%%\n", hitRate)
```

## Related Capabilities

- **Session Tracking**：贡献 NEW_SESSIONS、CLOSED_SESSIONS 指标
- **Policy Matching**：贡献 POLICY_HITS、POLICY_MISSES 指标
- **Policy Enforcement**：贡献 ALLOWED/DENIED_PACKETS 指标
- **Dataplane Performance**：统计收集优化为最小开销


# Session Timeout Management

## 概述

会话超时管理机制负责定期扫描会话表，检测并清理超时的会话。这对于防止会话表溢出、释放资源以及及时通知上层应用连接状态变化至关重要。

## 架构设计

### 混合方案
- **eBPF 层**: 提供超时配置 Map (`timeout_config_map`)
- **用户态 Go 层**: 实现定期扫描和清理逻辑
- **事件通知**: 通过日志记录超时事件（未来可扩展为 Ring Buffer 事件）

### 优势
- ✅ 简单可靠，不依赖特定内核版本（BPF Timer 需要 5.15+）
- ✅ 灵活配置超时策略（TCP/UDP 不同超时时间）
- ✅ 不影响数据平面性能（扫描在控制平面进行）
- ✅ 可精确控制扫描频率和批量大小

## 配置参数

### SessionTimeoutConfig 结构体

```go
type SessionTimeoutConfig struct {
    // TCP 会话空闲超时时间（无数据包）
    TCPIdleTimeout time.Duration

    // TCP 会话关闭超时时间（收到 FIN/RST 后）
    TCPCloseTimeout time.Duration

    // UDP 会话空闲超时时间
    UDPIdleTimeout time.Duration

    // 最大会话持续时间（防止无限期会话）
    MaxSessionDuration time.Duration

    // 扫描间隔（多久扫描一次会话表）
    ScanInterval time.Duration
}
```

### 推荐配置

| 参数 | 生产环境 | 开发环境 | 说明 |
|------|----------|----------|------|
| TCPIdleTimeout | 5-10分钟 | 1分钟 | TCP 连接无活动的超时时间 |
| TCPCloseTimeout | 30秒 | 10秒 | TCP 连接关闭后的保留时间 |
| UDPIdleTimeout | 1-2分钟 | 30秒 | UDP 会话无活动的超时时间 |
| MaxSessionDuration | 1小时 | 10分钟 | 防止异常长时间会话 |
| ScanInterval | 30-60秒 | 10秒 | 扫描频率，影响资源消耗 |

## 超时检测逻辑

### 超时条件（按优先级）

1. **TCP 关闭超时** (`tcp_close`)
   - 条件: TCP 状态为 CLOSING/CLOSED/FIN_WAIT 等
   - 超时: 自会话创建起超过 `TCPCloseTimeout`
   - 目的: 快速清理已关闭的 TCP 连接

2. **最大持续时间超时** (`max_duration`)
   - 条件: 所有协议
   - 超时: 自会话创建起超过 `MaxSessionDuration`
   - 目的: 防止异常长时间会话占用资源

3. **TCP 空闲超时** (`tcp_idle`)
   - 条件: TCP 协议
   - 超时: 自最后一个数据包起超过 `TCPIdleTimeout`
   - 目的: 清理不活跃的 TCP 连接

4. **UDP 空闲超时** (`udp_idle`)
   - 条件: UDP 协议
   - 超时: 自最后一个数据包起超过 `UDPIdleTimeout`
   - 目的: 清理不活跃的 UDP 会话

## 使用方法

### 1. 基本用法

```go
package main

import (
    "time"
    "github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
    "github.com/haolipeng/ebpf-based-microsegment/pkg/session"
)

func main() {
    // 创建数据平面
    dp, err := dataplane.New("eth0")
    if err != nil {
        panic(err)
    }
    defer dp.Close()

    // 配置超时参数
    config := session.SessionTimeoutConfig{
        TCPIdleTimeout:     5 * time.Minute,
        TCPCloseTimeout:    30 * time.Second,
        UDPIdleTimeout:     1 * time.Minute,
        MaxSessionDuration: 1 * time.Hour,
        ScanInterval:       30 * time.Second,
    }

    // 启用超时管理
    if err := dp.EnableSessionTimeout(config); err != nil {
        panic(err)
    }

    // 超时管理器现在在后台运行...
}
```

### 2. 获取统计信息

```go
// 获取超时统计
stats := dp.GetTimeoutStats()

fmt.Printf("Total Scans: %d\n", stats.TotalScans)
fmt.Printf("Sessions Scanned: %d\n", stats.TotalSessionsScanned)
fmt.Printf("Total Timed Out: %d\n", stats.TotalTimedOut)
fmt.Printf("  TCP Idle: %d\n", stats.TCPIdleTimeouts)
fmt.Printf("  TCP Close: %d\n", stats.TCPCloseTimeouts)
fmt.Printf("  UDP Idle: %d\n", stats.UDPIdleTimeouts)
fmt.Printf("  Max Duration: %d\n", stats.MaxDurationTimeouts)
fmt.Printf("Last Scan: %s (took %v)\n",
    stats.LastScanTime.Format("15:04:05"),
    stats.LastScanDuration)
```

### 3. 运行示例程序

```bash
# 编译示例
cd /home/work/ebpf-based-microsegment/src/agent
go build -o bin/session_timeout_example ./examples/session_timeout_example.go

# 运行（需要 root 权限）
sudo ./bin/session_timeout_example
```

## 实现细节

### eBPF 层

#### 1. 超时配置 Map

```c
// Timeout configuration map (shared between TC and XDP)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, TIMEOUT_CONFIG_MAX);
    __type(key, __u32);  // enum timeout_config_key
    __type(value, __u64); // timeout in nanoseconds
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} timeout_config_map SEC(".maps");
```

#### 2. 超时配置键

```c
enum timeout_config_key {
    TIMEOUT_CONFIG_TCP = 0,    // TCP session timeout
    TIMEOUT_CONFIG_UDP = 1,    // UDP session timeout
    TIMEOUT_CONFIG_ICMP = 2,   // ICMP session timeout
    TIMEOUT_CONFIG_OTHER = 3,  // Other protocol timeout
    TIMEOUT_CONFIG_MAX,
};
```

#### 3. 默认超时值

```c
#define DEFAULT_TCP_TIMEOUT_NS      (300ULL * 1000000000ULL)  // 5 minutes
#define DEFAULT_UDP_TIMEOUT_NS      (60ULL * 1000000000ULL)   // 1 minute
#define DEFAULT_ICMP_TIMEOUT_NS     (30ULL * 1000000000ULL)   // 30 seconds
#define DEFAULT_OTHER_TIMEOUT_NS    (60ULL * 1000000000ULL)   // 1 minute
```

### Go 用户态层

#### 1. TimeoutManager 结构

```go
type TimeoutManager struct {
    sessionMap *ebpf.Map                 // 会话表
    config     SessionTimeoutConfig      // 超时配置
    ctx        context.Context           // 用于优雅关闭
    cancel     context.CancelFunc
    wg         sync.WaitGroup
    stats      SessionTimeoutStats       // 统计信息
    statsMutex sync.RWMutex
}
```

#### 2. 扫描循环

```go
func (tm *TimeoutManager) scanLoop() {
    ticker := time.NewTicker(tm.config.ScanInterval)
    defer ticker.Stop()

    for {
        select {
        case <-tm.ctx.Done():
            return
        case <-ticker.C:
            if err := tm.runScan(); err != nil {
                log.Errorf("Scan failed: %v", err)
            }
        }
    }
}
```

#### 3. 扫描实现

扫描过程：
1. 遍历所有会话（使用 `sessionMap.Iterate()`）
2. 计算空闲时间和创建时间
3. 根据超时条件判断是否超时
4. 收集超时会话列表
5. 删除超时会话
6. 记录详细日志
7. 更新统计信息

## 性能考虑

### 扫描性能

- **时间复杂度**: O(n)，其中 n 是会话数量
- **空间复杂度**: O(m)，其中 m 是超时会话数量
- **扫描时间**:
  - 10,000 会话: ~10-50ms
  - 100,000 会话: ~100-500ms

### 优化建议

1. **调整扫描间隔**
   - 会话较少时: 可以缩短间隔（如 10 秒）
   - 会话较多时: 延长间隔（如 60 秒）

2. **批量处理**
   - 当前实现已经批量删除会话
   - 未来可以考虑分批扫描（每次只扫描部分会话）

3. **早停优化**
   - LRU_HASH Map 自动淘汰最旧会话
   - 扫描器主要处理异常超时情况

## 日志示例

### 启动日志

```
INFO[0000] [Session Timeout] Starting session timeout manager...
INFO[0000] [Session Timeout] Timeout manager started (scan interval: 30s)
INFO[0005] [Session Timeout] Scan loop started (TCP idle: 5m, TCP close: 30s, UDP idle: 1m, max: 1h)
```

### 超时事件日志

```
INFO[0035] [FLOW TIMEOUT] 192.168.1.100:45678 -> 10.0.0.50:80 proto=TCP reason=tcp_idle packets=150 bytes=45000
INFO[0035] [FLOW TIMEOUT] 192.168.1.101:53210 -> 8.8.8.8:53 proto=UDP reason=udp_idle packets=2 bytes=140
```

### 扫描统计日志

```
INFO[0035] [Session Timeout] Scan completed: scanned 1523 sessions, deleted 12 (TCP idle: 8, TCP close: 2, UDP idle: 2, max: 0, duration: 15.2ms)
DEBUG[0065] [Session Timeout] Scan completed: scanned 1511 sessions, no timeouts (duration: 14.8ms)
```

## 未来增强

### 1. Ring Buffer 事件推送

当前实现仅记录日志，未来可以推送 `FLOW_EVENT_TIMEOUT` 事件到 Ring Buffer：

```c
// In timeout detection logic
struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
if (event) {
    // Populate event with session data
    event->event_type = FLOW_EVENT_TIMEOUT;
    // ... other fields
    bpf_ringbuf_submit(event, 0);
}
```

### 2. eBPF Timer 支持（内核 5.15+）

使用 BPF Timer 在内核态实现超时检测：

```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct flow_key);
    __type(value, struct session_value_with_timer);
} session_map_with_timer SEC(".maps");

struct session_value_with_timer {
    struct session_value value;
    struct bpf_timer timer;
};

static int timeout_callback(void *map, struct flow_key *key, struct session_value_with_timer *session)
{
    // Check if session timed out
    // Delete if needed
    // Push FLOW_EVENT_TIMEOUT
    return 0;
}
```

### 3. 自适应扫描间隔

根据会话数量动态调整扫描间隔：
- 会话 < 1000: 每 10 秒扫描
- 会话 1000-10000: 每 30 秒扫描
- 会话 > 10000: 每 60 秒扫描

### 4. 分批扫描

对于大量会话（>50K），分批扫描避免单次扫描时间过长：

```go
const batchSize = 10000
for i := 0; i < totalSessions; i += batchSize {
    scanBatch(i, min(i+batchSize, totalSessions))
    time.Sleep(100 * time.Millisecond) // 避免 CPU 占用过高
}
```

## 故障排查

### 问题: 超时管理器未启动

**症状**: 没有超时日志，统计信息为空

**解决**:
```go
// 确保调用了 EnableSessionTimeout
if err := dp.EnableSessionTimeout(config); err != nil {
    log.Fatal(err)
}

// 检查是否有错误日志
```

### 问题: 会话未被清理

**症状**: 会话表持续增长

**解决**:
1. 检查超时配置是否合理
2. 查看扫描统计日志
3. 确认 LRU_HASH Map 大小足够

### 问题: 扫描时间过长

**症状**: `LastScanDuration` 超过 1 秒

**解决**:
1. 增加扫描间隔
2. 减少 MAX_ENTRIES_SESSION
3. 考虑实现分批扫描

## 相关文件

- `src/bpf/headers/common_types.h` - 超时配置定义
- `src/bpf/tc_microsegment.bpf.c` - TC 程序 timeout_config_map
- `src/bpf/xdp_microsegment.bpf.c` - XDP 程序 timeout_config_map
- `src/agent/pkg/session/types.go` - 超时配置和统计类型
- `src/agent/pkg/session/timeout_manager.go` - 超时管理器实现
- `src/agent/pkg/dataplane/dataplane.go` - 数据平面集成
- `src/agent/examples/session_timeout_example.go` - 使用示例

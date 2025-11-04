# Flow Collection API - 快速开始指南

本指南帮助开发者快速理解和使用 Flow Collection API 功能。

---

## 📖 目录

1. [概述](#概述)
2. [核心概念](#核心概念)
3. [架构概览](#架构概览)
4. [快速集成](#快速集成)
5. [API 使用示例](#api-使用示例)
6. [常见问题](#常见问题)
7. [下一步](#下一步)

---

## 概述

Flow Collection API 为 eBPF 微隔离系统添加了网络流数据收集、存储和分析能力。主要功能：

- ✅ **实时流收集**: 从 eBPF 数据平面收集网络流事件
- ✅ **持久化存储**: 使用 SQLite 存储历史流数据
- ✅ **强大查询**: 支持多维度过滤、分页、排序
- ✅ **聚合分析**: 流量摘要、依赖关系、Top Talkers
- ⏳ **实时推送**: WebSocket 实时流事件（Phase 4）

**当前版本**: Phase 1-3 已完成 (60%)

---

## 核心概念

### 什么是 Flow？

**Flow (网络流)** 是一组具有相同 5-tuple 的网络数据包序列：
- 源 IP 地址
- 目标 IP 地址
- 源端口
- 目标端口
- 协议 (TCP/UDP/ICMP)

### Flow 生命周期

```
1. NEW      → 新连接建立，eBPF 推送 FLOW_NEW 事件
2. ACTIVE   → 连接活跃，定期更新统计信息
3. CLOSED   → 连接关闭 (FIN/RST)，推送 FLOW_CLOSED 事件
4. TIMEOUT  → 连接超时 (默认 5 分钟无活动)
```

### 数据流转

```
数据包到达
    ↓
eBPF TC Hook 检测新连接
    ↓
push_flow_event() → Ring Buffer (256KB)
    ↓
Go Collector.collectLoop() 读取事件
    ↓
ParseFlowEvent() 解析 48 字节结构
    ↓
enrichWithLabels() 查询 WorkloadManager
    ↓
storage.SaveFlow() → SQLite 数据库
    ↓
API 查询 → 返回 JSON
```

---

## 架构概览

### 三层架构

```
┌────────────────────────────────────────┐
│         REST API 层                     │
│  /api/v1/flows - 查询、过滤、分页       │
│  /api/v1/flows/summary - 聚合统计      │
│  /api/v1/flows/dependencies - 依赖分析 │
└───────────────┬────────────────────────┘
                │
┌───────────────▼────────────────────────┐
│       Go 控制平面                       │
│  Collector → Storage → Aggregator      │
│  活跃流缓存 + 标签丰富                  │
└───────────────┬────────────────────────┘
                │
┌───────────────▼────────────────────────┐
│       eBPF 数据平面                     │
│  TC Hook → push_flow_event() → RingBuf │
└────────────────────────────────────────┘
```

### 关键组件

| 组件 | 文件 | 功能 |
|-----|------|------|
| **Flow Event** | `bpf/headers/common_types.h` | 48 字节流事件结构 |
| **Ring Buffer** | `bpf/tc_microsegment.bpf.c` | 256KB 事件缓冲区 |
| **Collector** | `pkg/flow/collector.go` | 读取 Ring Buffer，标签丰富 |
| **Storage** | `pkg/flow/storage.go` | SQLite 持久化层 |
| **Aggregator** | `pkg/flow/aggregator.go` | 聚合分析逻辑 |
| **API Handler** | `pkg/api/handlers/flow.go` | REST API 端点 |

---

## 快速集成

### 前置条件

```bash
# 1. eBPF 程序已加载
# 2. Agent 正在运行
# 3. SQLite 已安装
```

### 步骤 1: 初始化 Flow Storage

```go
package main

import (
    "github.com/ebpf-microsegment/src/agent/pkg/flow"
    "log"
)

func initFlowStorage() (*flow.SQLiteStorage, error) {
    // 创建 SQLite 存储
    storage, err := flow.NewSQLiteStorage("/var/lib/microsegment/flows.db")
    if err != nil {
        return nil, err
    }

    log.Println("✓ Flow storage initialized")
    return storage, nil
}
```

### 步骤 2: 创建 Flow Collector

```go
import (
    "github.com/cilium/ebpf/ringbuf"
)

func initFlowCollector(
    storage flow.Storage,
    workloadMgr flow.WorkloadManager,
) (*flow.Collector, error) {
    // 打开 Ring Buffer Reader (需要 DataPlane 暴露)
    ringBufReader, err := ebpf.OpenRingBuffer(/* ... */)
    if err != nil {
        return nil, err
    }

    // 创建 Collector 配置
    config := flow.DefaultCollectorConfig()
    config.FlowTimeout = 5 * time.Minute
    config.EnableEnrichment = true
    config.EnablePersistence = true

    // 创建 Collector
    collector := flow.NewCollector(ringBufReader, storage, workloadMgr, config)

    // 启动收集
    if err := collector.Start(); err != nil {
        return nil, err
    }

    log.Println("✓ Flow collector started")
    return collector, nil
}
```

### 步骤 3: 集成到 API Server

```go
func main() {
    // 初始化存储
    flowStorage, err := initFlowStorage()
    if err != nil {
        log.Fatal(err)
    }
    defer flowStorage.Close()

    // 初始化 Collector
    flowCollector, err := initFlowCollector(flowStorage, workloadManager)
    if err != nil {
        log.Fatal(err)
    }
    defer flowCollector.Stop()

    // 创建 API Server
    apiServer, err := api.NewAPIServer(config, dataPlane, policyManager)
    if err != nil {
        log.Fatal(err)
    }

    // 注入 Flow 组件
    apiServer.SetFlowComponents(flowCollector, flowStorage)

    // 启动 API Server
    if err := apiServer.Start(); err != nil {
        log.Fatal(err)
    }

    log.Println("✓ API server with Flow endpoints started")

    // 等待退出信号
    <-exitSignal
}
```

### 步骤 4: 验证

```bash
# 检查 Flow API 端点
curl http://localhost:8080/api/v1/flows/metrics

# 预期输出:
# {
#   "events_processed": 150,
#   "events_dropped": 0,
#   "active_flows": 12,
#   "drop_rate_percent": 0.0
# }
```

---

## API 使用示例

### 示例 1: 查询所有流

```bash
curl "http://localhost:8080/api/v1/flows?limit=10"
```

**响应**:
```json
{
  "flows": [
    {
      "id": "3232235876-167772161-12345-80-6",
      "source_ip": "192.168.1.100",
      "source_port": 12345,
      "dest_ip": "10.0.0.1",
      "dest_port": 80,
      "protocol": "TCP",
      "packet_count": 150,
      "byte_count": 102400,
      "start_time": "2025-11-04T10:15:30Z",
      "state": "ACTIVE"
    }
  ],
  "count": 10,
  "query": {
    "limit": 10,
    "offset": 0,
    "sort_by": "start_time",
    "sort_order": "desc"
  }
}
```

### 示例 2: 按时间范围过滤

```bash
curl "http://localhost:8080/api/v1/flows?\
start_time=2025-11-04T10:00:00Z&\
end_time=2025-11-04T11:00:00Z&\
limit=50"
```

### 示例 3: 多条件过滤

```bash
# 查询所有被拒绝的 TCP 流
curl "http://localhost:8080/api/v1/flows?\
protocol=TCP&\
policy_action=DENY&\
limit=100"
```

### 示例 4: 获取流量摘要

```bash
curl "http://localhost:8080/api/v1/flows/summary"
```

**响应**:
```json
{
  "total_flows": 1250,
  "active_flows": 45,
  "closed_flows": 1205,
  "total_packets": 150000,
  "total_bytes": 102400000,
  "allowed_flows": 1200,
  "denied_flows": 50,
  "top_protocols": [
    {
      "protocol": "TCP",
      "flow_count": 1000,
      "packet_count": 120000,
      "byte_count": 80000000
    }
  ]
}
```

### 示例 5: 应用依赖关系

```bash
curl "http://localhost:8080/api/v1/flows/dependencies?min_flows=5"
```

**响应**:
```json
{
  "dependencies": [
    {
      "source_labels": {
        "app": "nginx",
        "env": "prod"
      },
      "dest_labels": {
        "app": "redis",
        "env": "prod"
      },
      "flow_count": 150,
      "packet_count": 25000,
      "byte_count": 15000000,
      "protocols": ["TCP"],
      "last_seen": "2025-11-04T10:30:00Z"
    }
  ],
  "count": 1
}
```

### 示例 6: Top Talkers

```bash
curl "http://localhost:8080/api/v1/flows/top-talkers?limit=5"
```

**响应**:
```json
{
  "top_talkers": [
    {
      "ip": "192.168.1.100",
      "flow_count": 300,
      "packet_count": 50000,
      "byte_count": 30000000
    }
  ],
  "count": 5
}
```

### 示例 7: Collector 指标

```bash
curl "http://localhost:8080/api/v1/flows/metrics"
```

**响应**:
```json
{
  "events_processed": 125000,
  "events_dropped": 50,
  "active_flows": 45,
  "drop_rate_percent": 0.04
}
```

### 示例 8: 活跃流

```bash
curl "http://localhost:8080/api/v1/flows/active"
```

---

## Go 代码示例

### 查询流

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type FlowListResponse struct {
    Flows []Flow `json:"flows"`
    Count int    `json:"count"`
}

type Flow struct {
    ID         string    `json:"id"`
    SourceIP   string    `json:"source_ip"`
    DestIP     string    `json:"dest_ip"`
    Protocol   string    `json:"protocol"`
    ByteCount  uint64    `json:"byte_count"`
    StartTime  time.Time `json:"start_time"`
}

func queryFlows() error {
    // 构建查询 URL
    url := "http://localhost:8080/api/v1/flows?protocol=TCP&limit=10"

    // 发送请求
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // 解析响应
    var result FlowListResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    // 打印结果
    fmt.Printf("Found %d flows:\n", result.Count)
    for _, flow := range result.Flows {
        fmt.Printf("  %s -> %s (%s): %d bytes\n",
            flow.SourceIP, flow.DestIP, flow.Protocol, flow.ByteCount)
    }

    return nil
}
```

### 监控 Collector

```go
func monitorCollector() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        resp, err := http.Get("http://localhost:8080/api/v1/flows/metrics")
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }

        var metrics struct {
            EventsProcessed uint64  `json:"events_processed"`
            EventsDropped   uint64  `json:"events_dropped"`
            ActiveFlows     int     `json:"active_flows"`
            DropRate        float64 `json:"drop_rate_percent"`
        }

        if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
            log.Printf("Error: %v", err)
            resp.Body.Close()
            continue
        }
        resp.Body.Close()

        log.Printf("Processed: %d, Dropped: %d, Active: %d, Drop Rate: %.2f%%",
            metrics.EventsProcessed,
            metrics.EventsDropped,
            metrics.ActiveFlows,
            metrics.DropRate)
    }
}
```

---

## 常见问题

### Q1: 如何启用 Flow 收集？

**A**: Flow 收集需要三个组件：
1. eBPF 程序加载（自动）
2. Flow Storage 初始化
3. Flow Collector 启动
4. API Server 注入组件

参考上面的[快速集成](#快速集成)章节。

### Q2: Flow 数据会占用多少磁盘空间？

**A**: 取决于流量大小，估算公式：
```
每个 Flow 约 500 字节（包含 JSON 标签）
10,000 flows/hour = ~5 MB/hour = ~120 MB/day
```

**建议**:
- 配置自动清理（默认保留 7 天）
- 监控数据库大小：`SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size();`

### Q3: 查询性能如何优化？

**A**: 已优化措施：
- ✅ 8 个索引覆盖常见查询
- ✅ WAL 模式支持并发读
- ✅ 64MB 缓存

**未来优化**:
- ⏳ 查询结果缓存（LRU Cache）
- ⏳ 预聚合统计表
- ⏳ InfluxDB 替代 SQLite（时序数据）

### Q4: Ring Buffer 满了会怎样？

**A**:
- Ring Buffer 满时，`push_flow_event()` 返回 -1
- 事件**静默丢弃**，不影响数据平面性能
- 通过 `/api/v1/flows/metrics` 监控 `events_dropped`

**缓解措施**:
- 增大 Ring Buffer 大小（当前 256KB）
- 加快 Collector 读取速度
- 降低事件推送频率（仅 NEW/CLOSED）

### Q5: 如何查询带标签的流？

**A**: 当前使用 JSON LIKE 查询（较慢）：
```bash
# 查询 source_labels 中包含 "app=nginx" 的流
curl "http://localhost:8080/api/v1/flows?source_ip=192.168.1.100"
```

**注意**: 标签过滤功能需要 WorkloadManager 集成（待完成）。

**未来优化**: 单独的标签表 + JOIN 查询。

### Q6: 如何清理旧数据？

**A**: 使用 Storage.DeleteOldFlows()：
```go
// 删除 7 天前的数据
deleted, err := storage.DeleteOldFlows(7 * 24 * time.Hour)
if err != nil {
    log.Printf("Error: %v", err)
} else {
    log.Printf("Deleted %d old flows", deleted)
}
```

**自动清理**: 添加定时任务（待实现）。

### Q7: 支持 IPv6 吗？

**A**:
- ❌ 当前版本仅支持 IPv4
- ⏳ IPv6 支持计划在 Phase 5

### Q8: 如何禁用 Flow 收集？

**A**: 不调用 `apiServer.SetFlowComponents()`，Flow API 端点不会注册。

---

## 下一步

### 学习更多

- 📖 [完整实施文档](./flow-collection-implementation-summary.md) - 32,000 字技术详解
- 📊 [实施进度报告](./flow-implementation-progress.md) - 当前状态和计划
- 📋 [OpenSpec 设计文档](../openspec/changes/add-flow-collection-api/design.md)
- 📝 [OpenSpec 任务清单](../openspec/changes/add-flow-collection-api/tasks.md)

### 待完成功能

- ⏳ **Phase 4**: WebSocket 实时流推送
- ⏳ **DataPlane 集成**: Ring Buffer Reader
- ⏳ **WorkloadManager 集成**: 标签查询
- ⏳ **性能测试**: 10K flows/s 验证

### 贡献

发现问题或有改进建议？
1. 查看 [tasks.md](../openspec/changes/add-flow-collection-api/tasks.md)
2. 创建 Issue
3. 提交 Pull Request

---

## 附录: 查询参数完整列表

### GET /api/v1/flows

| 参数 | 类型 | 说明 | 示例 |
|-----|------|------|------|
| `start_time` | RFC3339 | 开始时间 | `2025-11-04T10:00:00Z` |
| `end_time` | RFC3339 | 结束时间 | `2025-11-04T11:00:00Z` |
| `source_ip` | String | 源 IP | `192.168.1.100` |
| `dest_ip` | String | 目标 IP | `10.0.0.1` |
| `protocol` | String | 协议 | `TCP`, `UDP`, `ICMP` |
| `state` | String | 状态 | `ACTIVE`, `CLOSED`, `TIMEOUT` |
| `direction` | String | 方向 | `INGRESS`, `EGRESS` |
| `policy_action` | String | 策略动作 | `ALLOW`, `DENY`, `LOG` |
| `limit` | Int | 返回数量 | `1-1000` (默认: 100) |
| `offset` | Int | 分页偏移 | `0+` (默认: 0) |
| `sort_by` | String | 排序字段 | `start_time`, `byte_count` |
| `sort_order` | String | 排序方向 | `asc`, `desc` (默认: desc) |

### GET /api/v1/flows/summary

| 参数 | 类型 | 说明 | 默认值 |
|-----|------|------|--------|
| `start_time` | RFC3339 | 开始时间 | 1 小时前 |
| `end_time` | RFC3339 | 结束时间 | 现在 |

### GET /api/v1/flows/dependencies

| 参数 | 类型 | 说明 | 默认值 |
|-----|------|------|--------|
| `start_time` | RFC3339 | 开始时间 | 1 小时前 |
| `end_time` | RFC3339 | 结束时间 | 现在 |
| `min_flows` | Int | 最小流数 | 1 |

### GET /api/v1/flows/top-talkers

| 参数 | 类型 | 说明 | 默认值 |
|-----|------|------|--------|
| `start_time` | RFC3339 | 开始时间 | 1 小时前 |
| `end_time` | RFC3339 | 结束时间 | 现在 |
| `limit` | Int | Top N | 10 (最大: 100) |

---

**最后更新**: 2025-11-04
**版本**: 1.0 (Phase 1-3)

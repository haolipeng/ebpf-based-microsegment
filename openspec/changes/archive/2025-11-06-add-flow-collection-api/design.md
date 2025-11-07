# Flow 数据收集 API - 详细设计文档 (Agent-Server 架构)

## 0. 架构变更说明

> **重要**：本文档已从单节点架构更新为 **Agent-Server 架构**。

### 0.1 架构演进

**旧架构（单节点）**：
- Agent 本地收集 Flow → 存储到本地 SQLite → Agent 提供 HTTP API
- 问题：数据孤岛、无全局视图、无法跨节点查询

**新架构（Agent-Server）**：
- Agent 收集 Flow → gRPC 上报到 Server → Server 集中存储（PostgreSQL/TimescaleDB） → Server 提供全局 HTTP API
- 优势：集中管理、全局查询、可扩展至 10000+ 节点

### 0.2 文档阅读指南

本文档分为两大部分：

**第 2 节 - Agent 端设计（保留）**：
- eBPF 数据平面设计（不变）
- Agent Flow Collector 设计（修改：增加 gRPC Reporter）
- 这部分设计仍然适用，但增加了 gRPC 上报逻辑

**第 3-7 节 - Server 端设计（需结合 `add-server-component` 阅读）**：
- ~~原本的 SQLite 存储设计~~ → 改为 PostgreSQL/TimescaleDB（见 `add-server-component/design.md`）
- ~~原本的 Agent HTTP API~~ → 改为 Server HTTP API
- ~~原本的 WebSocket Hub~~ → 改为 Server WebSocket Hub

**建议阅读顺序**：
1. 本文档第 2 节（Agent 端 eBPF 和 Collector）
2. `add-server-component/design.md`（Server 端 gRPC、存储、API）
3. 本文档第 5 节（WebSocket 实时推送，移至 Server 端）

### 0.3 关键变更总结

| 组件 | 旧架构位置 | 新架构位置 | 说明 |
|------|-----------|-----------|------|
| **eBPF Ring Buffer** | Agent | Agent | 不变 |
| **Flow Collector** | Agent | Agent | 保留，增加 gRPC Reporter |
| **Flow 存储** | Agent (SQLite) | Server (PostgreSQL) | **移动到 Server** |
| **Query API** | Agent (HTTP) | Server (HTTP) | **移动到 Server** |
| **WebSocket Hub** | Agent | Server | **移动到 Server** |
| **Aggregator** | Agent | Server | **移动到 Server** |

---

## 1. 系统架构概览

### 1.1 整体架构

Flow 数据收集系统采用 **Agent-Server 四层架构**设计，从 eBPF 内核态到 Agent 用户态，通过 gRPC 上报到 Server 控制平面，Server 提供全局 API 供前端访问，实现跨节点的网络流量可观测性。

**核心设计原则**：
1. **数据平面（Agent）**：轻量级，仅负责收集和上报
2. **控制平面（Server）**：集中存储、聚合、查询
3. **解耦设计**：Agent 和 Server 通过 gRPC 松耦合
4. **可扩展性**：支持 10-10000 节点

```
┌─────────────────────────────────────────────────────────────────┐
│                      Frontend (React + D3.js)                    │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │ Flow Table     │  │ ADM Topology   │  │ Real-time      │   │
│  │ (Query API)    │  │ (Dependencies) │  │ Stream (WS)    │   │
│  └────────────────┘  └────────────────┘  └────────────────┘   │
└──────────────┬────────────────┬────────────────┬───────────────┘
               │ REST API       │ REST API       │ WebSocket
               │ (Query)        │ (Aggregate)    │ (Push)
┌──────────────▼────────────────▼────────────────▼───────────────┐
│                  Go Agent - API Layer (Gin)                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Flow API Handlers                                         │  │
│  │ - GET /api/v1/flows (Query + Filter + Pagination)       │  │
│  │ - GET /api/v1/flows/:id (Single Flow)                   │  │
│  │ - GET /api/v1/flows/summary (Statistics)                │  │
│  │ - GET /api/v1/flows/dependencies (ADM Data)             │  │
│  │ - GET /api/v1/flows/top-talkers (Top N Analysis)        │  │
│  │ - WS  /api/v1/flows/stream (Real-time Push)             │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────┬──────────────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────────────────────┐
│            Go Agent - Management Layer (pkg/flow/)               │
│  ┌──────────────────────┐  ┌──────────────────────────────┐    │
│  │  Flow Collector      │  │  Flow Aggregator             │    │
│  │  - Read Ring Buffer  │  │  - Group by Labels           │    │
│  │  - Parse Events      │  │  - Calculate Dependencies    │    │
│  │  - Enrich Labels     │  │  - Top Talkers Analysis      │    │
│  │  - Save to Storage   │  └──────────────────────────────┘    │
│  │  - Push to WebSocket │                                       │
│  └──────────────────────┘                                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Flow Storage Interface                                   │  │
│  │  - SaveFlow() / GetFlow() / ListFlows()                 │  │
│  │  - GetSummary() / GetDependencies()                      │  │
│  │  - Implementation: SQLiteStorage / InfluxDBStorage      │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────┬──────────────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────────────────────┐
│           Go Agent - Storage Layer                               │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  SQLite Database                                          │  │
│  │  Table: flows                                             │  │
│  │  - id, source_ip, dest_ip, source_port, dest_port       │  │
│  │  - protocol, packet_count, byte_count, duration          │  │
│  │  - start_time, end_time, last_seen                       │  │
│  │  - source_labels (JSON), dest_labels (JSON)              │  │
│  │  - policy_id, policy_action, state, direction            │  │
│  │  Indexes: (start_time), (source_ip), (protocol)          │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────┬──────────────────────────────────────────────────┘
               │ Read Ring Buffer (cilium/ebpf)
┌──────────────▼──────────────────────────────────────────────────┐
│              eBPF Data Plane (Kernel Space)                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  TC eBPF Program (tc_microsegment.bpf.c)                │  │
│  │  - Extract 5-tuple from packets                          │  │
│  │  - Track sessions in session_map (LRU_HASH)             │  │
│  │  - Update packet/byte counters                           │  │
│  │  - Detect connection events (NEW/CLOSED)                 │  │
│  │  - Push events to flow_events Ring Buffer                │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  eBPF Maps                                                │  │
│  │  - session_map (LRU_HASH, 100k entries)                 │  │
│  │  - flow_events (RING_BUFFER, 256KB)  ← NEW              │  │
│  │  - policy_map, stats_map (existing)                      │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 数据流图

```
Packet Arrival (TC Ingress/Egress)
    │
    ▼
[TC eBPF Program]
    │
    ├──> Extract 5-tuple (src_ip, dst_ip, src_port, dst_port, proto)
    │
    ├──> Lookup session_map
    │    │
    │    ├──> NEW Connection → Create session entry
    │    │                    Push FLOW_NEW event to Ring Buffer
    │    │
    │    ├──> EXISTING Connection → Update counters (packets, bytes)
    │    │                          (Optional: periodic FLOW_UPDATE)
    │    │
    │    └──> CLOSED Connection (FIN/RST) → Mark session closed
    │                                        Push FLOW_CLOSED event
    │
    └──> Policy Check & Action (ALLOW/DENY)
         Return TC_ACT_OK / TC_ACT_SHOT

Ring Buffer Event (struct flow_event)
    │
    ▼
[Flow Collector - Go Agent]
    │
    ├──> Read Ring Buffer (cilium/ebpf RingBufferReader)
    │
    ├──> Parse binary event → Flow struct
    │
    ├──> Enrich with Workload Labels
    │    └──> Query WorkloadManager.GetLabelsByIP()
    │
    ├──> Save to Storage (SQLite)
    │    └──> INSERT INTO flows (...) VALUES (...)
    │
    └──> Broadcast to WebSocket Clients
         └──> wsHub.Broadcast(flow)

Query Request (Frontend → API)
    │
    ▼
[API Handler]
    │
    ├──> Parse query parameters (filters, pagination)
    │
    ├──> Call Storage.ListFlows(query)
    │    │
    │    └──> SQL: SELECT * FROM flows WHERE ...
    │         LIMIT 100 OFFSET 0
    │         ORDER BY start_time DESC
    │
    └──> Return JSON Response
         {
           "flows": [...],
           "total": 12345,
           "page": 1
         }

Dependencies Request (Frontend → API)
    │
    ▼
[Aggregator]
    │
    ├──> Query Storage.ListFlows(time_range)
    │
    ├──> Group by (source_labels, dest_labels)
    │
    ├──> Calculate aggregates:
    │    - Total bytes per workload pair
    │    - Flow count per workload pair
    │
    └──> Return Dependency Graph Data
         {
           "nodes": [
             {"id": "app=nginx", "labels": {...}},
             {"id": "app=mysql", "labels": {...}}
           ],
           "edges": [
             {"source": "app=nginx", "target": "app=mysql",
              "bytes": 1234567, "flows": 42}
           ]
         }
```

---

## 2. eBPF 数据平面设计

### 2.1 Flow Event 数据结构

定义在 `src/bpf/headers/flow_types.h`：

```c
#ifndef __FLOW_TYPES_H
#define __FLOW_TYPES_H

#include <linux/types.h>

// Flow event types
enum flow_event_type {
    FLOW_NEW    = 0,  // New connection established
    FLOW_UPDATE = 1,  // Periodic update (optional)
    FLOW_CLOSED = 2,  // Connection closed (FIN/RST)
};

// Flow event structure (pushed to Ring Buffer)
// Size: 48 bytes (packed)
struct flow_event {
    // 5-tuple identification
    __u32 src_ip;        // Source IP (IPv4, network byte order)
    __u32 dst_ip;        // Destination IP
    __u16 src_port;      // Source port
    __u16 dst_port;      // Destination port
    __u8  protocol;      // IPPROTO_TCP/UDP/ICMP
    __u8  event_type;    // enum flow_event_type
    __u8  direction;     // 0=INGRESS, 1=EGRESS
    __u8  padding;       // Alignment padding

    // Traffic statistics
    __u64 packet_count;  // Total packets
    __u64 byte_count;    // Total bytes

    // Timestamps
    __u64 timestamp_ns;  // Event timestamp (nanoseconds since boot)

    // Policy context
    __u32 policy_id;     // Matched policy ID (0 if no policy)
    __u8  policy_action; // 0=ALLOW, 1=DENY
    __u8  state;         // TCP state (ESTABLISHED, FIN_WAIT, etc.)
    __u16 reserved;      // Reserved for future use
} __attribute__((packed));

#endif // __FLOW_TYPES_H
```

### 2.2 Ring Buffer Map 定义

在 `src/bpf/tc_microsegment.bpf.c` 中添加：

```c
#include "headers/flow_types.h"

// Ring Buffer for pushing flow events to userspace
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);  // 256KB buffer
} flow_events SEC(".maps");

// Statistics for dropped events
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 1);
} flow_events_dropped SEC(".maps");
```

### 2.3 Flow 事件推送逻辑

#### 2.3.1 辅助函数：push_flow_event

```c
// Push flow event to Ring Buffer
// Returns 0 on success, negative on error
static __always_inline int push_flow_event(
    struct flow_key *key,
    struct session_value *session,
    enum flow_event_type event_type,
    __u8 direction)
{
    struct flow_event *event;

    // Reserve space in Ring Buffer
    event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
    if (!event) {
        // Ring Buffer full, increment dropped counter
        __u32 key_zero = 0;
        __u64 *dropped = bpf_map_lookup_elem(&flow_events_dropped, &key_zero);
        if (dropped) {
            __sync_fetch_and_add(dropped, 1);
        }
        return -1;
    }

    // Fill event structure
    event->src_ip = key->src_ip;
    event->dst_ip = key->dst_ip;
    event->src_port = key->src_port;
    event->dst_port = key->dst_port;
    event->protocol = key->protocol;
    event->event_type = event_type;
    event->direction = direction;
    event->padding = 0;

    event->packet_count = session->packet_count;
    event->byte_count = session->byte_count;
    event->timestamp_ns = bpf_ktime_get_ns();

    event->policy_id = session->policy_id;
    event->policy_action = session->action;
    event->state = session->state;
    event->reserved = 0;

    // Submit event to Ring Buffer
    bpf_ringbuf_submit(event, 0);

    return 0;
}
```

#### 2.3.2 集成到 TC 程序主函数

修改 `tc_microsegment.bpf.c` 中的主 TC 程序：

```c
SEC("tc")
int tc_microsegment(struct __sk_buff *skb)
{
    // ... existing packet parsing code ...

    struct flow_key key = {0};
    // ... extract 5-tuple into key ...

    // Lookup session
    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);

    if (!session) {
        // NEW CONNECTION
        struct session_value new_session = {0};
        new_session.state = TCP_ESTABLISHED;  // Or appropriate state
        new_session.packet_count = 1;
        new_session.byte_count = skb->len;
        new_session.last_seen = bpf_ktime_get_ns();

        // ... policy check ...
        new_session.policy_id = matched_policy_id;
        new_session.action = policy_action;

        // Insert into session_map
        bpf_map_update_elem(&session_map, &key, &new_session, BPF_NOEXIST);

        // Push FLOW_NEW event
        __u8 direction = (skb->ingress_ifindex != 0) ? 0 : 1;
        push_flow_event(&key, &new_session, FLOW_NEW, direction);

    } else {
        // EXISTING CONNECTION
        session->packet_count++;
        session->byte_count += skb->len;
        session->last_seen = bpf_ktime_get_ns();

        // Check for connection close (TCP FIN/RST)
        if (is_tcp_close(tcp_flags)) {
            session->state = TCP_CLOSED;

            // Push FLOW_CLOSED event
            __u8 direction = (skb->ingress_ifindex != 0) ? 0 : 1;
            push_flow_event(&key, session, FLOW_CLOSED, direction);

            // Optional: delete session from map
            // bpf_map_delete_elem(&session_map, &key);
        }
    }

    // ... rest of policy enforcement ...

    return action;
}
```

### 2.4 性能考虑

#### Ring Buffer 大小选择
- **256KB 容量**：假设每个事件 48 字节，可存储约 5,400 个事件
- **高负载场景**（10,000 flows/s）：
  - 每秒产生约 480KB 事件（48 字节 × 10,000）
  - 需要用户空间以 < 50ms 间隔读取，避免溢出
- **缓解策略**：
  - 只在连接建立/关闭时推送（不是每个数据包）
  - 监控 `flow_events_dropped` 指标
  - 可配置 Ring Buffer 大小（通过宏定义）

#### eBPF 验证器复杂度
- **边界检查**：Ring Buffer 操作自动通过验证器
- **栈使用**：`struct flow_event` 在堆上分配（Ring Buffer），不占用栈
- **指令限制**：`push_flow_event` 约 50 条指令，不会超限

---

## 3. Go 控制平面设计

### 3.1 包结构

```
src/agent/pkg/flow/
├── types.go           # Flow 数据结构定义
├── collector.go       # Flow Collector 实现
├── storage.go         # Storage 接口定义
├── sqlite_storage.go  # SQLite 实现
├── aggregator.go      # 聚合和依赖分析
├── websocket.go       # WebSocket Hub
├── collector_test.go
├── storage_test.go
└── aggregator_test.go
```

### 3.2 核心数据结构 (types.go)

```go
package flow

import (
    "encoding/json"
    "time"
)

// Flow represents a network flow (aggregation of packets with same 5-tuple)
type Flow struct {
    // Primary key
    ID string `json:"id" db:"id"` // UUID

    // 5-tuple identification
    SourceIP   string `json:"source_ip" db:"source_ip"`
    SourcePort uint16 `json:"source_port" db:"source_port"`
    DestIP     string `json:"dest_ip" db:"dest_ip"`
    DestPort   uint16 `json:"dest_port" db:"dest_port"`
    Protocol   string `json:"protocol" db:"protocol"` // TCP/UDP/ICMP

    // Traffic statistics
    PacketCount uint64 `json:"packet_count" db:"packet_count"`
    ByteCount   uint64 `json:"byte_count" db:"byte_count"`
    Duration    int64  `json:"duration_ms" db:"duration"` // milliseconds

    // Timestamps
    StartTime time.Time  `json:"start_time" db:"start_time"`
    EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"` // NULL if still active
    LastSeen  time.Time  `json:"last_seen" db:"last_seen"`

    // Workload enrichment
    SourceLabels map[string]string `json:"source_labels,omitempty" db:"source_labels"` // JSON
    DestLabels   map[string]string `json:"dest_labels,omitempty" db:"dest_labels"`     // JSON

    // Policy context
    PolicyID     int    `json:"policy_id,omitempty" db:"policy_id"`
    PolicyAction string `json:"policy_action" db:"policy_action"` // ALLOW/DENY

    // State
    State     string `json:"state" db:"state"`         // ACTIVE/CLOSED/TIMEOUT
    Direction string `json:"direction" db:"direction"` // INGRESS/EGRESS
}

// FlowEvent represents a flow event from eBPF (matches C struct)
type FlowEvent struct {
    SrcIP       uint32 // Network byte order
    DstIP       uint32
    SrcPort     uint16
    DstPort     uint16
    Protocol    uint8
    EventType   uint8 // 0=NEW, 1=UPDATE, 2=CLOSED
    Direction   uint8 // 0=INGRESS, 1=EGRESS
    Padding     uint8
    PacketCount uint64
    ByteCount   uint64
    TimestampNs uint64
    PolicyID    uint32
    PolicyAction uint8
    State       uint8
    Reserved    uint16
}

// FlowQuery represents query parameters for listing flows
type FlowQuery struct {
    // Time range
    StartTime *time.Time
    EndTime   *time.Time

    // 5-tuple filters
    SourceIP   *string
    DestIP     *string
    SourcePort *uint16
    DestPort   *uint16
    Protocol   *string

    // Label filters (JSON query)
    SourceLabels map[string]string
    DestLabels   map[string]string

    // State filters
    State     *string
    Direction *string

    // Pagination
    Limit  int
    Offset int

    // Sorting
    OrderBy string // "start_time", "byte_count", "packet_count"
    Order   string // "asc", "desc"
}

// FlowSummary represents aggregated statistics
type FlowSummary struct {
    TotalFlows      int64             `json:"total_flows"`
    TotalPackets    uint64            `json:"total_packets"`
    TotalBytes      uint64            `json:"total_bytes"`
    ActiveFlows     int64             `json:"active_flows"`
    ClosedFlows     int64             `json:"closed_flows"`
    FlowsByProtocol map[string]int64  `json:"flows_by_protocol"`
    FlowsByAction   map[string]int64  `json:"flows_by_action"`
}

// Dependency represents a communication relationship between workloads
type Dependency struct {
    SourceLabels map[string]string `json:"source_labels"`
    DestLabels   map[string]string `json:"dest_labels"`
    FlowCount    int64             `json:"flow_count"`
    TotalBytes   uint64            `json:"total_bytes"`
    TotalPackets uint64            `json:"total_packets"`
}
```

### 3.3 Flow Collector 实现 (collector.go)

```go
package flow

import (
    "bytes"
    "context"
    "encoding/binary"
    "fmt"
    "net"
    "time"

    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/ringbuf"
    "github.com/google/uuid"
    log "github.com/sirupsen/logrus"
)

// Collector reads flow events from eBPF Ring Buffer
type Collector struct {
    ringBuf      *ringbuf.Reader
    storage      Storage
    workloadMgr  WorkloadManager // For label enrichment
    wsHub        *WebSocketHub   // For real-time push
    ctx          context.Context
    cancel       context.CancelFunc
    droppedCount uint64
}

// WorkloadManager interface for label enrichment
type WorkloadManager interface {
    GetLabelsByIP(ip string) (map[string]string, error)
}

// NewCollector creates a new Flow Collector
func NewCollector(
    flowEventsMap *ebpf.Map,
    storage Storage,
    workloadMgr WorkloadManager,
    wsHub *WebSocketHub,
) (*Collector, error) {
    reader, err := ringbuf.NewReader(flowEventsMap)
    if err != nil {
        return nil, fmt.Errorf("failed to create ring buffer reader: %w", err)
    }

    ctx, cancel := context.WithCancel(context.Background())

    return &Collector{
        ringBuf:     reader,
        storage:     storage,
        workloadMgr: workloadMgr,
        wsHub:       wsHub,
        ctx:         ctx,
        cancel:      cancel,
    }, nil
}

// Start begins collecting flow events
func (c *Collector) Start() {
    go c.collectLoop()
}

// Stop stops the collector and closes resources
func (c *Collector) Stop() error {
    c.cancel()
    return c.ringBuf.Close()
}

// collectLoop is the main event collection loop
func (c *Collector) collectLoop() {
    log.Info("Flow Collector started")

    for {
        select {
        case <-c.ctx.Done():
            log.Info("Flow Collector stopped")
            return
        default:
        }

        // Read event from Ring Buffer (blocking with timeout)
        record, err := c.ringBuf.Read()
        if err != nil {
            if err == ringbuf.ErrClosed {
                return
            }
            log.Errorf("Failed to read from ring buffer: %v", err)
            continue
        }

        // Parse event
        event, err := c.parseFlowEvent(record.RawSample)
        if err != nil {
            log.Errorf("Failed to parse flow event: %v", err)
            continue
        }

        // Convert to Flow struct
        flow, err := c.eventToFlow(event)
        if err != nil {
            log.Errorf("Failed to convert event to flow: %v", err)
            continue
        }

        // Enrich with workload labels
        c.enrichWithLabels(flow)

        // Save to storage
        if err := c.storage.SaveFlow(flow); err != nil {
            log.Errorf("Failed to save flow: %v", err)
            continue
        }

        // Push to WebSocket clients
        if c.wsHub != nil {
            c.wsHub.Broadcast(flow)
        }

        log.Debugf("Collected flow: %s:%d -> %s:%d (%s, %d bytes)",
            flow.SourceIP, flow.SourcePort,
            flow.DestIP, flow.DestPort,
            flow.Protocol, flow.ByteCount)
    }
}

// parseFlowEvent parses binary event data from Ring Buffer
func (c *Collector) parseFlowEvent(data []byte) (*FlowEvent, error) {
    if len(data) < 48 {
        return nil, fmt.Errorf("invalid event size: %d", len(data))
    }

    event := &FlowEvent{}
    buf := bytes.NewReader(data)

    if err := binary.Read(buf, binary.LittleEndian, event); err != nil {
        return nil, fmt.Errorf("failed to decode event: %w", err)
    }

    return event, nil
}

// eventToFlow converts FlowEvent to Flow struct
func (c *Collector) eventToFlow(event *FlowEvent) (*Flow, error) {
    flow := &Flow{
        ID:           uuid.New().String(),
        SourceIP:     intToIP(event.SrcIP),
        SourcePort:   event.SrcPort,
        DestIP:       intToIP(event.DstIP),
        DestPort:     event.DstPort,
        Protocol:     protocolToString(event.Protocol),
        PacketCount:  event.PacketCount,
        ByteCount:    event.ByteCount,
        StartTime:    time.Now(), // Will be refined based on event type
        LastSeen:     time.Now(),
        PolicyID:     int(event.PolicyID),
        PolicyAction: actionToString(event.PolicyAction),
        Direction:    directionToString(event.Direction),
    }

    // Set state based on event type
    switch event.EventType {
    case 0: // FLOW_NEW
        flow.State = "ACTIVE"
    case 1: // FLOW_UPDATE
        flow.State = "ACTIVE"
    case 2: // FLOW_CLOSED
        flow.State = "CLOSED"
        now := time.Now()
        flow.EndTime = &now
    }

    return flow, nil
}

// enrichWithLabels adds workload labels to flow
func (c *Collector) enrichWithLabels(flow *Flow) {
    // Get source labels
    if srcLabels, err := c.workloadMgr.GetLabelsByIP(flow.SourceIP); err == nil {
        flow.SourceLabels = srcLabels
    }

    // Get destination labels
    if dstLabels, err := c.workloadMgr.GetLabelsByIP(flow.DestIP); err == nil {
        flow.DestLabels = dstLabels
    }
}

// Helper functions
func intToIP(ip uint32) string {
    return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24)).String()
}

func protocolToString(proto uint8) string {
    switch proto {
    case 6:
        return "TCP"
    case 17:
        return "UDP"
    case 1:
        return "ICMP"
    default:
        return fmt.Sprintf("PROTO_%d", proto)
    }
}

func actionToString(action uint8) string {
    if action == 0 {
        return "ALLOW"
    }
    return "DENY"
}

func directionToString(dir uint8) string {
    if dir == 0 {
        return "INGRESS"
    }
    return "EGRESS"
}
```

### 3.4 gRPC Reporter 实现 (reporter.go) - **新增**

> **架构变更**：Agent 不再存储 Flow 到本地 SQLite，而是通过 gRPC 批量上报到 Server。

```go
package flow

import (
    "context"
    "time"

    pb "github.com/ebpf-microsegment/src/proto/flow"
    "google.golang.org/grpc"
    log "github.com/sirupsen/logrus"
)

// Reporter sends flow events to Server via gRPC
type Reporter struct {
    client      pb.FlowServiceClient
    conn        *grpc.ClientConn
    batchSize   int
    batchTimeout time.Duration
    buffer      []*Flow
    bufferMutex sync.Mutex
    ctx         context.Context
    cancel      context.CancelFunc
    wg          sync.WaitGroup
}

// ReporterConfig holds Reporter configuration
type ReporterConfig struct {
    ServerAddr   string        // Server gRPC address (e.g., "server:9090")
    BatchSize    int           // Max flows per batch (default: 1000)
    BatchTimeout time.Duration // Max wait time before sending (default: 1s)
    EnableRetry  bool          // Enable retry on failure
}

// NewReporter creates a new gRPC Reporter
func NewReporter(config ReporterConfig) (*Reporter, error) {
    // Connect to Server
    conn, err := grpc.Dial(config.ServerAddr, grpc.WithInsecure())
    if err != nil {
        return nil, fmt.Errorf("failed to connect to server: %w", err)
    }

    client := pb.NewFlowServiceClient(conn)
    ctx, cancel := context.WithCancel(context.Background())

    return &Reporter{
        client:       client,
        conn:         conn,
        batchSize:    config.BatchSize,
        batchTimeout: config.BatchTimeout,
        buffer:       make([]*Flow, 0, config.BatchSize),
        ctx:          ctx,
        cancel:       cancel,
    }, nil
}

// Start begins the reporting loop
func (r *Reporter) Start() {
    r.wg.Add(1)
    go r.reportLoop()
}

// Stop stops the reporter and flushes remaining flows
func (r *Reporter) Stop() error {
    r.cancel()
    r.wg.Wait()

    // Flush remaining flows
    if len(r.buffer) > 0 {
        r.sendBatch(r.buffer)
    }

    return r.conn.Close()
}

// Report adds a flow to the buffer and sends when batch is full
func (r *Reporter) Report(flow *Flow) {
    r.bufferMutex.Lock()
    r.buffer = append(r.buffer, flow)
    shouldSend := len(r.buffer) >= r.batchSize
    r.bufferMutex.Unlock()

    if shouldSend {
        r.flushBuffer()
    }
}

// reportLoop periodically flushes the buffer based on timeout
func (r *Reporter) reportLoop() {
    defer r.wg.Done()

    ticker := time.NewTicker(r.batchTimeout)
    defer ticker.Stop()

    for {
        select {
        case <-r.ctx.Done():
            return
        case <-ticker.C:
            r.flushBuffer()
        }
    }
}

// flushBuffer sends all buffered flows to Server
func (r *Reporter) flushBuffer() {
    r.bufferMutex.Lock()
    if len(r.buffer) == 0 {
        r.bufferMutex.Unlock()
        return
    }

    batch := r.buffer
    r.buffer = make([]*Flow, 0, r.batchSize)
    r.bufferMutex.Unlock()

    r.sendBatch(batch)
}

// sendBatch sends a batch of flows via gRPC streaming
func (r *Reporter) sendBatch(batch []*Flow) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Open client stream
    stream, err := r.client.ReportFlowEvents(ctx)
    if err != nil {
        log.Errorf("Failed to open flow report stream: %v", err)
        return
    }

    // Send all flows in batch
    for _, flow := range batch {
        event := r.flowToProto(flow)
        if err := stream.Send(event); err != nil {
            log.Errorf("Failed to send flow event: %v", err)
            break
        }
    }

    // Close stream and get response
    resp, err := stream.CloseAndRecv()
    if err != nil {
        log.Errorf("Failed to close flow report stream: %v", err)
        return
    }

    if resp.Success {
        log.Debugf("Reported %d flows to server (accepted: %d, rejected: %d)",
            len(batch), resp.AcceptedCount, resp.RejectedCount)
    } else {
        log.Errorf("Server rejected flow batch: %s", resp.Message)
    }
}

// flowToProto converts Flow struct to protobuf FlowEvent
func (r *Reporter) flowToProto(flow *Flow) *pb.FlowEvent {
    event := &pb.FlowEvent{
        Id:           flow.ID,
        AgentId:      r.agentID, // Set in NewReporter
        SourceIp:     flow.SourceIP,
        SourcePort:   uint32(flow.SourcePort),
        DestIp:       flow.DestIP,
        DestPort:     uint32(flow.DestPort),
        Protocol:     flow.Protocol,
        PacketCount:  flow.PacketCount,
        ByteCount:    flow.ByteCount,
        StartTime:    flow.StartTime.Unix(),
        LastSeen:     flow.LastSeen.Unix(),
        PolicyId:     uint32(flow.PolicyID),
        PolicyAction: flow.PolicyAction,
        State:        flow.State,
        Direction:    flow.Direction,
        SourceLabels: flow.SourceLabels,
        DestLabels:   flow.DestLabels,
    }

    if flow.EndTime != nil {
        event.EndTime = flow.EndTime.Unix()
    }

    return event
}
```

**集成到 Collector**：

```go
// 修改 Collector 结构体，移除 storage，添加 reporter
type Collector struct {
    ringBuf      *ringbuf.Reader
    reporter     *Reporter        // 替代 storage
    workloadMgr  WorkloadManager
    ctx          context.Context
    cancel       context.CancelFunc
}

// 修改 collectLoop，将 SaveFlow 改为 Report
func (c *Collector) collectLoop() {
    for {
        // ... 读取和解析事件 ...

        flow, err := c.eventToFlow(event)
        if err != nil {
            continue
        }

        c.enrichWithLabels(flow)

        // 通过 gRPC 上报到 Server（替代本地存储）
        c.reporter.Report(flow)

        log.Debugf("Reported flow: %s:%d -> %s:%d",
            flow.SourceIP, flow.SourcePort,
            flow.DestIP, flow.DestPort)
    }
}
```

---

### 3.5 Storage 接口 (storage.go) - **已废弃，移至 Server 端**

> **架构变更**：Storage 接口现在在 Server 端实现（见 `add-server-component/design.md`），Agent 不再需要本地存储。

以下是 Server 端的 Storage 接口定义（位于 `src/server/pkg/storage/`）：

```go
package flow

// Storage defines the interface for flow persistence
type Storage interface {
    // SaveFlow persists a flow record
    SaveFlow(flow *Flow) error

    // GetFlow retrieves a flow by ID
    GetFlow(id string) (*Flow, error)

    // ListFlows queries flows with filters and pagination
    ListFlows(query *FlowQuery) ([]*Flow, int64, error) // (flows, total, error)

    // GetSummary returns aggregated statistics
    GetSummary(startTime, endTime time.Time) (*FlowSummary, error)

    // GetDependencies returns workload communication dependencies
    GetDependencies(startTime, endTime time.Time) ([]*Dependency, error)

    // GetTopTalkers returns top N sources/destinations by bytes
    GetTopTalkers(startTime, endTime time.Time, n int) ([]TopTalker, error)

    // DeleteOldFlows removes flows older than the specified time
    DeleteOldFlows(olderThan time.Time) (int64, error)

    // Close closes the storage connection
    Close() error
}

type TopTalker struct {
    IP          string            `json:"ip"`
    Labels      map[string]string `json:"labels"`
    TotalBytes  uint64            `json:"total_bytes"`
    TotalFlows  int64             `json:"total_flows"`
}
```

### 3.5 SQLite 存储实现 (sqlite_storage.go)

```go
package flow

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
    db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    storage := &SQLiteStorage{db: db}

    if err := storage.initSchema(); err != nil {
        return nil, err
    }

    return storage, nil
}

func (s *SQLiteStorage) initSchema() error {
    schema := `
    CREATE TABLE IF NOT EXISTS flows (
        id TEXT PRIMARY KEY,
        source_ip TEXT NOT NULL,
        source_port INTEGER NOT NULL,
        dest_ip TEXT NOT NULL,
        dest_port INTEGER NOT NULL,
        protocol TEXT NOT NULL,
        packet_count INTEGER NOT NULL,
        byte_count INTEGER NOT NULL,
        duration INTEGER,
        start_time DATETIME NOT NULL,
        end_time DATETIME,
        last_seen DATETIME NOT NULL,
        source_labels TEXT, -- JSON
        dest_labels TEXT,   -- JSON
        policy_id INTEGER,
        policy_action TEXT NOT NULL,
        state TEXT NOT NULL,
        direction TEXT NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_flows_start_time ON flows(start_time);
    CREATE INDEX IF NOT EXISTS idx_flows_source_ip ON flows(source_ip);
    CREATE INDEX IF NOT EXISTS idx_flows_dest_ip ON flows(dest_ip);
    CREATE INDEX IF NOT EXISTS idx_flows_protocol ON flows(protocol);
    CREATE INDEX IF NOT EXISTS idx_flows_state ON flows(state);
    `

    _, err := s.db.Exec(schema)
    return err
}

func (s *SQLiteStorage) SaveFlow(flow *Flow) error {
    srcLabelsJSON, _ := json.Marshal(flow.SourceLabels)
    dstLabelsJSON, _ := json.Marshal(flow.DestLabels)

    _, err := s.db.Exec(`
        INSERT INTO flows (
            id, source_ip, source_port, dest_ip, dest_port, protocol,
            packet_count, byte_count, duration, start_time, end_time, last_seen,
            source_labels, dest_labels, policy_id, policy_action, state, direction
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        flow.ID, flow.SourceIP, flow.SourcePort, flow.DestIP, flow.DestPort, flow.Protocol,
        flow.PacketCount, flow.ByteCount, flow.Duration, flow.StartTime, flow.EndTime, flow.LastSeen,
        string(srcLabelsJSON), string(dstLabelsJSON), flow.PolicyID, flow.PolicyAction, flow.State, flow.Direction,
    )

    return err
}

func (s *SQLiteStorage) ListFlows(query *FlowQuery) ([]*Flow, int64, error) {
    // Build WHERE clause
    where := []string{"1=1"}
    args := []interface{}{}

    if query.StartTime != nil {
        where = append(where, "start_time >= ?")
        args = append(args, query.StartTime)
    }
    if query.EndTime != nil {
        where = append(where, "start_time <= ?")
        args = append(args, query.EndTime)
    }
    if query.SourceIP != nil {
        where = append(where, "source_ip = ?")
        args = append(args, *query.SourceIP)
    }
    if query.Protocol != nil {
        where = append(where, "protocol = ?")
        args = append(args, *query.Protocol)
    }
    // ... more filters ...

    whereClause := strings.Join(where, " AND ")

    // Count total
    var total int64
    err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM flows WHERE %s", whereClause), args...).Scan(&total)
    if err != nil {
        return nil, 0, err
    }

    // Query flows
    orderBy := query.OrderBy
    if orderBy == "" {
        orderBy = "start_time"
    }
    order := query.Order
    if order == "" {
        order = "DESC"
    }

    rows, err := s.db.Query(fmt.Sprintf(`
        SELECT * FROM flows WHERE %s ORDER BY %s %s LIMIT ? OFFSET ?
    `, whereClause, orderBy, order), append(args, query.Limit, query.Offset)...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    flows := []*Flow{}
    for rows.Next() {
        flow := &Flow{}
        var srcLabelsJSON, dstLabelsJSON string
        // Scan all fields...
        err := rows.Scan(&flow.ID, &flow.SourceIP, &flow.SourcePort, ...)
        if err != nil {
            return nil, 0, err
        }
        json.Unmarshal([]byte(srcLabelsJSON), &flow.SourceLabels)
        json.Unmarshal([]byte(dstLabelsJSON), &flow.DestLabels)
        flows = append(flows, flow)
    }

    return flows, total, nil
}

// ... other methods (GetSummary, GetDependencies, etc.) ...
```

*(完整实现见代码库)*

---

---

## **架构变更说明：Server 端设计**

> 从第 4 节开始，以下章节（API 设计、WebSocket 推送、性能优化等）的实现已**从 Agent 移至 Server**。
>
> **重要**：这些章节中的代码示例和设计仍然有效，但需要理解为**在 Server 端实现**，而不是 Agent 端。
>
> **建议**：结合阅读 `add-server-component/design.md` 以获取完整的 Server 端架构。

**关键变更**：
- ~~Agent 提供 HTTP API~~ → **Server 提供 HTTP API**（端口 :8080）
- ~~Agent 存储到 SQLite~~ → **Server 存储到 PostgreSQL/TimescaleDB**
- ~~Agent 运行 WebSocket Hub~~ → **Server 运行 WebSocket Hub**
- ~~Agent 本地聚合~~ → **Server 跨节点全局聚合**

以下章节的代码路径更新：
- `src/agent/pkg/api/` → `src/server/pkg/api/`
- `src/agent/pkg/flow/storage.go` → `src/server/pkg/storage/flow_storage.go`
- `src/agent/pkg/flow/aggregator.go` → `src/server/pkg/aggregator/`

---

## 4. API 设计 (**Server 端实现**)

### 4.1 REST API 端点

#### 4.1.1 GET /api/v1/flows - 查询流列表

**请求参数**：
```
start_time:     ISO8601 时间戳 (optional, default: 1 hour ago)
end_time:       ISO8601 时间戳 (optional, default: now)
source_ip:      string (optional)
dest_ip:        string (optional)
protocol:       TCP/UDP/ICMP (optional)
state:          ACTIVE/CLOSED (optional)
direction:      INGRESS/EGRESS (optional)
source_labels:  JSON object (optional, e.g., {"app":"nginx"})
dest_labels:    JSON object (optional)
limit:          int (default: 100, max: 1000)
offset:         int (default: 0)
order_by:       start_time/byte_count/packet_count (default: start_time)
order:          asc/desc (default: desc)
```

**响应示例**：
```json
{
  "flows": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "source_ip": "10.0.1.10",
      "source_port": 45678,
      "dest_ip": "10.0.2.20",
      "dest_port": 3306,
      "protocol": "TCP",
      "packet_count": 1234,
      "byte_count": 567890,
      "duration_ms": 5000,
      "start_time": "2025-11-04T10:30:00Z",
      "end_time": "2025-11-04T10:30:05Z",
      "last_seen": "2025-11-04T10:30:05Z",
      "source_labels": {
        "app": "nginx",
        "role": "frontend",
        "env": "production"
      },
      "dest_labels": {
        "app": "mysql",
        "role": "database",
        "env": "production"
      },
      "policy_id": 42,
      "policy_action": "ALLOW",
      "state": "CLOSED",
      "direction": "EGRESS"
    }
  ],
  "total": 12345,
  "page": 1,
  "limit": 100
}
```

#### 4.1.2 GET /api/v1/flows/:id - 获取单条流记录

**响应示例**：
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "source_ip": "10.0.1.10",
  ...
}
```

#### 4.1.3 GET /api/v1/flows/summary - 流量统计摘要

**请求参数**：
```
start_time: ISO8601 时间戳 (optional)
end_time:   ISO8601 时间戳 (optional)
```

**响应示例**：
```json
{
  "total_flows": 12345,
  "total_packets": 9876543210,
  "total_bytes": 123456789012,
  "active_flows": 234,
  "closed_flows": 12111,
  "flows_by_protocol": {
    "TCP": 10000,
    "UDP": 2000,
    "ICMP": 345
  },
  "flows_by_action": {
    "ALLOW": 12000,
    "DENY": 345
  }
}
```

#### 4.1.4 GET /api/v1/flows/dependencies - 应用依赖关系

**请求参数**：
```
start_time: ISO8601 时间戳 (optional, default: 1 hour ago)
end_time:   ISO8601 时间戳 (optional, default: now)
group_by:   label keys (comma-separated, default: "app,role")
```

**响应示例**：
```json
{
  "nodes": [
    {
      "id": "nginx-frontend",
      "labels": {
        "app": "nginx",
        "role": "frontend"
      }
    },
    {
      "id": "mysql-database",
      "labels": {
        "app": "mysql",
        "role": "database"
      }
    }
  ],
  "edges": [
    {
      "source": "nginx-frontend",
      "target": "mysql-database",
      "flow_count": 42,
      "total_bytes": 1234567,
      "total_packets": 5678
    }
  ]
}
```

#### 4.1.5 GET /api/v1/flows/top-talkers - Top N 分析

**请求参数**：
```
start_time: ISO8601 时间戳 (optional)
end_time:   ISO8601 时间戳 (optional)
n:          int (default: 10, max: 100)
by:         source/destination (default: source)
```

**响应示例**：
```json
{
  "top_talkers": [
    {
      "ip": "10.0.1.10",
      "labels": {
        "app": "nginx"
      },
      "total_bytes": 987654321,
      "total_flows": 1234
    }
  ]
}
```

#### 4.1.6 WS /api/v1/flows/stream - 实时流推送 (WebSocket)

**连接 URL**：
```
ws://localhost:8080/api/v1/flows/stream
```

**订阅过滤 (发送 JSON 消息)**：
```json
{
  "action": "subscribe",
  "filters": {
    "protocol": "TCP",
    "source_labels": {
      "app": "nginx"
    }
  }
}
```

**接收消息格式**：
```json
{
  "type": "flow",
  "data": {
    "id": "...",
    "source_ip": "10.0.1.10",
    ...
  }
}
```

---

## 5. 实时推送设计 (WebSocket)

### 5.1 WebSocket Hub 实现

```go
package flow

import (
    "encoding/json"
    "sync"

    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
)

type WebSocketHub struct {
    clients    map[*Client]bool
    broadcast  chan *Flow
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

type Client struct {
    hub     *WebSocketHub
    conn    *websocket.Conn
    send    chan *Flow
    filters *FlowQuery // Subscription filters
}

func NewWebSocketHub() *WebSocketHub {
    return &WebSocketHub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan *Flow, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *WebSocketHub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            log.Infof("WebSocket client connected (total: %d)", len(h.clients))

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
            log.Infof("WebSocket client disconnected (total: %d)", len(h.clients))

        case flow := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                // Check if flow matches client filters
                if client.matchesFilters(flow) {
                    select {
                    case client.send <- flow:
                    default:
                        // Client send buffer full, close connection
                        close(client.send)
                        delete(h.clients, client)
                    }
                }
            }
            h.mu.RUnlock()
        }
    }
}

func (h *WebSocketHub) Broadcast(flow *Flow) {
    select {
    case h.broadcast <- flow:
    default:
        // Broadcast buffer full, drop flow
        log.Warn("WebSocket broadcast buffer full, dropping flow")
    }
}

func (c *Client) matchesFilters(flow *Flow) bool {
    if c.filters == nil {
        return true // No filters, accept all
    }

    // Check protocol filter
    if c.filters.Protocol != nil && *c.filters.Protocol != flow.Protocol {
        return false
    }

    // Check label filters
    // (implement label matching logic)

    return true
}

func (c *Client) writePump() {
    defer func() {
        c.conn.Close()
    }()

    for flow := range c.send {
        data, err := json.Marshal(map[string]interface{}{
            "type": "flow",
            "data": flow,
        })
        if err != nil {
            log.Errorf("Failed to marshal flow: %v", err)
            continue
        }

        if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
            log.Errorf("Failed to write message: %v", err)
            return
        }
    }
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }

        // Parse subscription filters
        var req struct {
            Action  string     `json:"action"`
            Filters *FlowQuery `json:"filters"`
        }
        if err := json.Unmarshal(message, &req); err != nil {
            log.Errorf("Invalid subscription message: %v", err)
            continue
        }

        if req.Action == "subscribe" {
            c.filters = req.Filters
            log.Infof("Client subscribed with filters: %+v", req.Filters)
        }
    }
}
```

---

## 6. 性能优化

### 6.1 数据库优化

#### 索引策略
- **时间查询**：`idx_flows_start_time` (start_time)
- **IP 查询**：`idx_flows_source_ip`, `idx_flows_dest_ip`
- **协议查询**：`idx_flows_protocol`
- **复合索引**：考虑 `(start_time, protocol, state)` 用于常见查询模式

#### 批量插入
```go
func (s *SQLiteStorage) SaveFlowBatch(flows []*Flow) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare("INSERT INTO flows (...) VALUES (?, ...)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, flow := range flows {
        if _, err := stmt.Exec(...); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

#### 查询结果缓存
```go
import "github.com/hashicorp/golang-lru"

type CachedStorage struct {
    storage Storage
    cache   *lru.Cache // LRU cache for query results
}

func (s *CachedStorage) ListFlows(query *FlowQuery) ([]*Flow, int64, error) {
    cacheKey := query.CacheKey()
    if cached, ok := s.cache.Get(cacheKey); ok {
        return cached.([]*Flow), cachedTotal, nil
    }

    flows, total, err := s.storage.ListFlows(query)
    if err == nil {
        s.cache.Add(cacheKey, flows)
    }

    return flows, total, err
}
```

### 6.2 Ring Buffer 调优

#### 动态调整大小
```c
// In tc_microsegment.bpf.c
#ifndef FLOW_RINGBUF_SIZE
#define FLOW_RINGBUF_SIZE (256 * 1024)  // Default: 256KB
#endif

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, FLOW_RINGBUF_SIZE);
} flow_events SEC(".maps");
```

编译时指定：
```bash
clang -O2 -target bpf -c -DFLOW_RINGBUF_SIZE=$((512*1024)) tc_microsegment.bpf.c
```

#### 用户空间读取优化
```go
func (c *Collector) collectLoop() {
    // Batch read multiple events
    batch := make([]ringbuf.Record, 0, 100)

    for {
        record, err := c.ringBuf.Read()
        if err != nil {
            continue
        }

        batch = append(batch, record)

        // Process batch when full or timeout
        if len(batch) >= 100 {
            c.processBatch(batch)
            batch = batch[:0]
        }
    }
}
```

### 6.3 WebSocket 优化

#### 消息批处理
```go
func (c *Client) writePumpBatched() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    batch := []*Flow{}

    for {
        select {
        case flow := <-c.send:
            batch = append(batch, flow)

        case <-ticker.C:
            if len(batch) > 0 {
                data, _ := json.Marshal(map[string]interface{}{
                    "type": "flow_batch",
                    "data": batch,
                })
                c.conn.WriteMessage(websocket.TextMessage, data)
                batch = batch[:0]
            }
        }
    }
}
```

---

## 7. 监控和可观测性

### 7.1 Prometheus 指标

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    flowEventsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flow_events_total",
            Help: "Total number of flow events processed",
        },
        []string{"event_type", "protocol"},
    )

    flowCollectorErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flow_collector_errors_total",
            Help: "Total number of flow collector errors",
        },
        []string{"error_type"},
    )

    flowStorageLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "flow_storage_latency_seconds",
            Help:    "Flow storage operation latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation"},
    )

    websocketClientsConnected = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "websocket_clients_connected",
            Help: "Number of connected WebSocket clients",
        },
    )

    ringBufferEventsDropped = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "ring_buffer_events_dropped_total",
            Help: "Total number of events dropped due to ring buffer full",
        },
    )
)

func init() {
    prometheus.MustRegister(
        flowEventsTotal,
        flowCollectorErrors,
        flowStorageLatency,
        websocketClientsConnected,
        ringBufferEventsDropped,
    )
}
```

### 7.2 结构化日志

```go
import log "github.com/sirupsen/logrus"

func (c *Collector) collectLoop() {
    log.WithFields(log.Fields{
        "component": "flow_collector",
    }).Info("Starting flow collection")

    // ...

    log.WithFields(log.Fields{
        "flow_id":     flow.ID,
        "source_ip":   flow.SourceIP,
        "dest_ip":     flow.DestIP,
        "protocol":    flow.Protocol,
        "byte_count":  flow.ByteCount,
    }).Debug("Flow collected")
}
```

---

## 8. 测试策略

### 8.1 eBPF 程序测试

使用 `bpftool` 和 `tc` 进行手工测试：

```bash
# 加载 eBPF 程序
sudo ./loader lo

# 查看 Ring Buffer
sudo bpftool map show name flow_events
sudo bpftool map dump name flow_events

# 生成测试流量
curl http://localhost:8080

# 查看丢弃计数
sudo bpftool map dump name flow_events_dropped
```

### 8.2 Go 单元测试

```go
// collector_test.go
func TestFlowEventParsing(t *testing.T) {
    data := []byte{/* 48 bytes raw event */}
    collector := &Collector{}

    event, err := collector.parseFlowEvent(data)
    assert.NoError(t, err)
    assert.Equal(t, uint32(0x0A000101), event.SrcIP)
    assert.Equal(t, uint16(12345), event.SrcPort)
}

// storage_test.go
func TestSQLiteStorage_SaveAndGet(t *testing.T) {
    storage, _ := NewSQLiteStorage(":memory:")
    defer storage.Close()

    flow := &Flow{
        ID: "test-id",
        SourceIP: "10.0.1.1",
        // ...
    }

    err := storage.SaveFlow(flow)
    assert.NoError(t, err)

    retrieved, err := storage.GetFlow("test-id")
    assert.NoError(t, err)
    assert.Equal(t, flow.SourceIP, retrieved.SourceIP)
}
```

### 8.3 API 集成测试

```go
func TestFlowAPI_ListFlows(t *testing.T) {
    // Setup test server
    router := gin.Default()
    handler := NewFlowHandler(mockStorage)
    router.GET("/api/v1/flows", handler.ListFlows)

    // Test request
    req := httptest.NewRequest("GET", "/api/v1/flows?protocol=TCP&limit=10", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.NotNil(t, resp["flows"])
}
```

### 8.4 性能基准测试

```go
func BenchmarkFlowCollector_ParseEvent(b *testing.B) {
    collector := &Collector{}
    data := []byte{/* 48 bytes */}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        collector.parseFlowEvent(data)
    }
}

func BenchmarkSQLiteStorage_SaveFlow(b *testing.B) {
    storage, _ := NewSQLiteStorage(":memory:")
    defer storage.Close()

    flow := &Flow{/* ... */}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        storage.SaveFlow(flow)
    }
}
```

---

## 9. 生产部署考虑

### 9.1 数据生命周期管理

```go
// Cron job to clean old flows
func (s *SQLiteStorage) StartCleanupJob(ctx context.Context, retention time.Duration) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            cutoff := time.Now().Add(-retention)
            deleted, err := s.DeleteOldFlows(cutoff)
            if err != nil {
                log.Errorf("Failed to cleanup old flows: %v", err)
            } else {
                log.Infof("Deleted %d old flows (older than %v)", deleted, cutoff)
            }

        case <-ctx.Done():
            return
        }
    }
}
```

### 9.2 配置参数

```go
type FlowConfig struct {
    RingBufferSize     int           `yaml:"ring_buffer_size" default:"262144"`
    CollectorWorkers   int           `yaml:"collector_workers" default:"1"`
    StoragePath        string        `yaml:"storage_path" default:"/var/lib/ebpf-microsegment/flows.db"`
    RetentionDays      int           `yaml:"retention_days" default:"7"`
    EnableWebSocket    bool          `yaml:"enable_websocket" default:"true"`
    WebSocketBatchSize int           `yaml:"websocket_batch_size" default:"100"`
    QueryCacheSize     int           `yaml:"query_cache_size" default:"1000"`
}
```

### 9.3 错误处理和恢复

```go
func (c *Collector) collectLoopWithRetry() {
    retries := 0
    maxRetries := 5

    for {
        err := c.collectLoop()
        if err == nil {
            return
        }

        retries++
        if retries >= maxRetries {
            log.Fatalf("Flow collector failed after %d retries: %v", maxRetries, err)
        }

        backoff := time.Duration(retries) * time.Second
        log.Warnf("Flow collector error: %v. Retrying in %v...", err, backoff)
        time.Sleep(backoff)
    }
}
```

---

## 10. 未来增强功能

### 10.1 InfluxDB 集成

InfluxDB 是时序数据库，更适合存储大规模 Flow 数据：

```go
import influxdb2 "github.com/influxdata/influxdb-client-go/v2"

type InfluxDBStorage struct {
    client influxdb2.Client
    org    string
    bucket string
}

func (s *InfluxDBStorage) SaveFlow(flow *Flow) error {
    writeAPI := s.client.WriteAPIBlocking(s.org, s.bucket)

    point := influxdb2.NewPoint(
        "network_flow",
        map[string]string{
            "source_ip":   flow.SourceIP,
            "dest_ip":     flow.DestIP,
            "protocol":    flow.Protocol,
            "state":       flow.State,
        },
        map[string]interface{}{
            "byte_count":   flow.ByteCount,
            "packet_count": flow.PacketCount,
            "duration":     flow.Duration,
        },
        flow.StartTime,
    )

    return writeAPI.WritePoint(context.Background(), point)
}
```

### 10.2 异常检测

基于历史基线的异常流量检测：

```go
type AnomalyDetector struct {
    baseline map[string]*Baseline // Key: workload pair
}

type Baseline struct {
    MeanBytes   float64
    StdDevBytes float64
    MeanFlows   float64
    StdDevFlows float64
}

func (d *AnomalyDetector) Detect(flow *Flow) bool {
    key := fmt.Sprintf("%v-%v", flow.SourceLabels, flow.DestLabels)
    baseline := d.baseline[key]

    if baseline == nil {
        return false // No baseline yet
    }

    // Z-score anomaly detection
    zScore := (float64(flow.ByteCount) - baseline.MeanBytes) / baseline.StdDevBytes
    return math.Abs(zScore) > 3.0 // 3-sigma threshold
}
```

### 10.3 NetFlow/IPFIX 导出

支持标准网络流协议用于 SIEM 集成：

```go
type NetFlowExporter struct {
    conn net.Conn
}

func (e *NetFlowExporter) Export(flow *Flow) error {
    // Convert to NetFlow v9 format
    record := &netflow.Record{
        SrcAddr:   flow.SourceIP,
        DstAddr:   flow.DestIP,
        SrcPort:   flow.SourcePort,
        DstPort:   flow.DestPort,
        Protocol:  flow.Protocol,
        Bytes:     flow.ByteCount,
        Packets:   flow.PacketCount,
        StartTime: flow.StartTime,
        EndTime:   *flow.EndTime,
    }

    return e.conn.Write(record.Encode())
}
```

---

## 11. 总结

本设计文档详细描述了 Flow 数据收集 API 的完整架构和实现方案，涵盖：

- **eBPF 数据平面**：Ring Buffer 事件推送机制
- **Go 控制平面**：Flow Collector、Storage、Aggregator
- **API 层**：REST API 和 WebSocket 实时推送
- **性能优化**：数据库索引、批处理、缓存
- **可观测性**：Prometheus 指标和结构化日志
- **测试策略**：单元测试、集成测试、性能基准测试
- **生产部署**：数据生命周期管理、错误恢复、配置化

该系统设计能够满足以下需求：
- ✅ 处理 10,000 flows/s
- ✅ API 查询响应时间 < 100ms
- ✅ WebSocket 推送延迟 < 500ms
- ✅ 支持前端 ADM 可视化
- ✅ 支持标签过滤和聚合分析

**下一步**：开始实施 Phase 1（eBPF Flow 收集）。

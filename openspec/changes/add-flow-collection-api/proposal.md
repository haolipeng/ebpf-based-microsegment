# Proposal: Flow 数据收集 API

## 概述

为 eBPF 微隔离系统添加网络流（Flow）数据收集和查询 API，用于支持前端应用依赖地图（Application Dependency Mapping, ADM）可视化和流量分析功能。

## 动机

### 当前状态
- ✅ 系统已实现基于 eBPF 的数据包级策略执行
- ✅ 实现了 Session 跟踪（session_map）用于连接状态管理
- ✅ 实现了 Workload 管理和标签系统
- ✅ 实现了基本的统计信息收集（stats_map）

### 存在的问题
- ❌ **缺乏流级别的数据聚合**：当前只有数据包级别的统计，无法提供会话级别的流量视图
- ❌ **无法追溯历史流量**：session_map 是内存态，系统重启后丢失，无历史数据
- ❌ **缺少工作负载关系可视化**：前端无法展示 "哪些工作负载正在互相通信"
- ❌ **缺少流量趋势分析**：无法回答 "过去 1 小时的 top talkers 是谁" 等问题
- ❌ **缺少应用依赖地图**：无法生成服务依赖拓扑图

### 目标
1. **实现 Flow 数据收集**：从 eBPF 数据平面聚合流级别的流量数据
2. **持久化 Flow 数据**：存储历史流量记录，支持时间范围查询
3. **提供 Flow 查询 API**：为前端提供流量查询、过滤、聚合的 REST API
4. **支持实时流推送**：通过 WebSocket/SSE 推送新流事件到前端
5. **支持应用依赖分析**：提供聚合后的工作负载间依赖关系数据

### 成功标准
- [ ] 能够查询过去 24 小时的所有流记录
- [ ] API 响应时间 < 100ms（查询 1000 条流记录）
- [ ] 实时流推送延迟 < 500ms
- [ ] 前端能够基于 Flow API 渲染应用依赖地图
- [ ] 支持标签过滤查询（如："查询 app=nginx 的所有出站流量"）

## 影响范围

### 新增组件

**1. eBPF 数据平面**
- 新增 `flow_events` Ring Buffer：用于向用户空间推送流事件

**2. Go 控制平面**
- 新增 `pkg/flow/` 包：
  - `types.go` - Flow 数据结构定义
  - `collector.go` - Flow 收集器（从 Ring Buffer 读取）
  - `storage.go` - Flow 持久化接口
  - `aggregator.go` - Flow 聚合和依赖分析

**3. API 层**
- 新增 Flow API 端点：
  - `GET /api/v1/flows` - 查询流列表
  - `GET /api/v1/flows/:id` - 获取单条流记录
  - `GET /api/v1/flows/summary` - 流量统计摘要
  - `GET /api/v1/flows/dependencies` - 应用依赖关系
  - `WS /api/v1/flows/stream` - 实时流推送

**4. 存储层**
- 选项 1：扩展 SQLite（短期方案）
- 选项 2：引入 InfluxDB（长期方案，更适合时序数据）

### 修改的组件

**1. eBPF 程序 (`tc_microsegment.bpf.c`)**
- 修改：在连接建立/关闭时通过 Ring Buffer 推送流事件
- 影响：轻微性能影响（需测试）

**2. Agent 主程序 (`cmd/main.go`)**
- 修改：启动 Flow Collector 和 API Server
- 影响：增加内存和 CPU 开销

**3. Workload Manager (`pkg/workload/`)**
- 修改：Flow Collector 需要查询工作负载标签
- 影响：需要提供高效的 IP → Labels 查询接口

### 不修改的组件
- ✅ Policy 管理（`pkg/policy/`）
- ✅ 现有的 Session 跟踪逻辑
- ✅ 现有的统计信息收集（`stats_map`）

## 架构概览

```
┌──────────────────────────────────────────────────────────┐
│                    Frontend (React)                       │
│  - Application Dependency Map (ADM)                       │
│  - Traffic Flow Table                                     │
│  - Real-time Flow Stream                                  │
└────────────────────┬─────────────────────────────────────┘
                     │ REST API / WebSocket
┌────────────────────▼─────────────────────────────────────┐
│              Go Agent - API Layer                         │
│  /api/v1/flows (Query, Filter, Pagination)               │
│  /api/v1/flows/dependencies (Aggregated Dependencies)    │
│  /api/v1/flows/stream (WebSocket Push)                   │
└────────────────────┬─────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────┐
│         Flow Collector (pkg/flow/collector.go)            │
│  - Read from Ring Buffer (flow_events)                   │
│  - Enrich with Workload Labels                           │
│  - Persist to Storage                                     │
│  - Push to WebSocket clients                             │
└────────┬───────────────────────────┬─────────────────────┘
         │                           │
         │                           ▼
         │                    ┌──────────────────┐
         │                    │  Flow Storage    │
         │                    │  (SQLite/InfluxDB)│
         │                    └──────────────────┘
         │
┌────────▼─────────────────────────────────────────────────┐
│          eBPF Data Plane (tc_microsegment.bpf.c)         │
│  - Track Sessions (session_map)                          │
│  - On New Flow: bpf_ringbuf_output(flow_events)         │
│  - On Flow End: bpf_ringbuf_output(flow_events)         │
└──────────────────────────────────────────────────────────┘
```

## 核心数据结构

### Flow 结构（Go）
```go
type Flow struct {
    // Identification (5-tuple)
    SourceIP      string    `json:"source_ip"`
    SourcePort    uint16    `json:"source_port"`
    DestIP        string    `json:"dest_ip"`
    DestPort      uint16    `json:"dest_port"`
    Protocol      string    `json:"protocol"` // TCP/UDP/ICMP

    // Traffic Statistics
    PacketCount   uint64    `json:"packet_count"`
    ByteCount     uint64    `json:"byte_count"`
    Duration      int64     `json:"duration_ms"` // milliseconds

    // Timestamps
    StartTime     time.Time `json:"start_time"`
    EndTime       time.Time `json:"end_time,omitempty"`
    LastSeen      time.Time `json:"last_seen"`

    // Workload Enrichment
    SourceLabels  map[string]string `json:"source_labels,omitempty"`
    DestLabels    map[string]string `json:"dest_labels,omitempty"`

    // Policy Context
    PolicyID      int       `json:"policy_id,omitempty"`
    PolicyAction  string    `json:"policy_action"` // ALLOW/DENY

    // State
    State         string    `json:"state"` // ACTIVE/CLOSED/TIMEOUT
    Direction     string    `json:"direction"` // INGRESS/EGRESS
}
```

### Flow Event（eBPF → Userspace）
```c
struct flow_event {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  event_type;  // 0=NEW, 1=UPDATE, 2=CLOSED
    __u64 packet_count;
    __u64 byte_count;
    __u64 timestamp_ns;
};
```

## 实施阶段

### Phase 1: 基础 Flow 收集（Week 1）
- [ ] 定义 Flow 数据结构（eBPF + Go）
- [ ] 在 eBPF 中实现 Ring Buffer 推送逻辑
- [ ] 实现 Go Flow Collector（读取 Ring Buffer）
- [ ] 基于 SQLite 实现基本的 Flow 存储

**验收标准**：能够从 eBPF 收集流事件并存储到 SQLite

### Phase 2: Flow 查询 API（Week 2）
- [ ] 实现 Flow 查询 API（分页、过滤）
- [ ] 实现标签过滤查询
- [ ] 实现时间范围查询
- [ ] 添加单元测试和集成测试

**验收标准**：前端能够通过 API 查询和过滤流记录

### Phase 3: 实时流推送（Week 3）
- [ ] 实现 WebSocket/SSE 实时推送
- [ ] 实现流事件缓冲和批处理
- [ ] 添加客户端重连逻辑

**验收标准**：前端能够实时接收新流事件（延迟 < 500ms）

### Phase 4: 应用依赖分析（Week 4）
- [ ] 实现 Flow 聚合逻辑（按标签分组）
- [ ] 实现应用依赖关系 API
- [ ] 实现 Top Talkers 分析
- [ ] 优化性能（缓存、索引）

**验收标准**：前端能够渲染应用依赖地图（ADM）

## 风险与缓解

### 风险 1: 性能影响
- **问题**：Ring Buffer 推送可能影响 eBPF 数据平面性能
- **缓解**：
  - 只在连接建立/关闭时推送（不是每个数据包）
  - 使用 Ring Buffer 而非 Perf Buffer（更高效）
  - 配置可调的批处理参数

### 风险 2: 存储容量
- **问题**：Flow 数据快速增长可能耗尽磁盘空间
- **缓解**：
  - 实现自动清理策略（保留 7 天）
  - 提供聚合存储（每小时统计）
  - 长期考虑 InfluxDB 的数据压缩

### 风险 3: 查询性能
- **问题**：大量 Flow 记录可能导致查询缓慢
- **缓解**：
  - 在 SQLite 上创建适当索引
  - 实现查询结果缓存
  - 限制查询时间范围（默认 1 小时）

## 替代方案

### 方案 A: 直接使用现有 session_map（不采纳）
- ❌ 无法持久化
- ❌ 无法查询历史数据
- ❌ 容量有限（max_entries）

### 方案 B: 使用 Perf Buffer 而非 Ring Buffer（不采纳）
- ❌ Ring Buffer 是更现代、更高效的选择
- ❌ Perf Buffer 在高负载下容易丢失事件

### 方案 C: 不实现实时推送，仅提供轮询 API（不采纳）
- ❌ 前端需要频繁轮询，浪费资源
- ❌ 延迟较高，用户体验差

## 测试策略

### 单元测试
- Flow Collector 的事件解析逻辑
- Flow Storage 的 CRUD 操作
- API 查询过滤逻辑

### 集成测试
- eBPF → Go Collector → Storage 的端到端流程
- API 查询准确性
- WebSocket 推送可靠性

### 性能测试
- 模拟 10,000 flows/s 的收集能力
- 查询响应时间（1000 条记录 < 100ms）
- 内存和磁盘使用量

## 相关文档

- `docs/microsegmentation-mvp-implementation-plan.md` - Phase 4 前端可视化需求
- `openspec/project.md` - 项目架构和命名约定
- `src/agent/pkg/workload/` - 工作负载管理（用于标签查询）
- `src/agent/pkg/policy/` - 策略管理（用于 PolicyID 关联）

## 批准状态

- [ ] 技术审查通过
- [ ] 架构设计审查通过
- [ ] 准备开始实施

# Proposal: Flow 数据收集 API (Agent-Server 架构)

## 概述

为 eBPF 微隔离系统添加网络流（Flow）数据收集和查询 API，基于 **Agent-Server 架构**实现跨节点流量聚合和全局可视化。Agent 负责从 eBPF 收集流数据并通过 gRPC 上报到 Server，Server 负责集中存储、聚合分析并提供查询 API，支持前端应用依赖地图（Application Dependency Mapping, ADM）可视化和流量分析功能。

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
- ❌ **缺少跨节点全局视图**：每个 Agent 数据孤立，无法查看整个集群的流量全貌

### 目标
1. **Agent 端：实现 Flow 数据收集**：从 eBPF Ring Buffer 读取流事件并通过 gRPC 上报到 Server
2. **Server 端：集中存储 Flow 数据**：使用 PostgreSQL/TimescaleDB 存储来自所有 Agent 的流量记录
3. **Server 端：提供全局查询 API**：为前端提供跨节点流量查询、过滤、聚合的 REST API
4. **Server 端：实时流推送**：通过 WebSocket 推送新流事件到前端
5. **Server 端：全局依赖分析**：提供跨节点的工作负载间依赖关系数据

### 成功标准
- [ ] Agent 能够稳定地从 eBPF 收集流事件（无丢失）
- [ ] Agent 通过 gRPC 批量上报到 Server（延迟 < 1s）
- [ ] Server 能够查询过去 24 小时的所有节点流记录
- [ ] Server API 响应时间 < 100ms（查询 1000 条流记录）
- [ ] Server 实时流推送延迟 < 500ms
- [ ] 前端能够基于 Server API 渲染跨节点应用依赖地图
- [ ] 支持标签过滤查询（如："查询所有节点上 app=nginx 的出站流量"）

## 影响范围

### 新增组件

**1. Agent 端 - eBPF 数据平面**
- 新增 `flow_events` Ring Buffer：用于向用户空间推送流事件

**2. Agent 端 - Go 控制平面**
- 新增 `pkg/flow/` 包：
  - `types.go` - Flow 数据结构定义
  - `collector.go` - Flow 收集器（从 Ring Buffer 读取）
  - `reporter.go` - gRPC Reporter（批量上报到 Server）

**3. Server 端 - gRPC 服务**
- 扩展 `FlowService`：
  - `ReportFlowEvents()` - 接收 Agent 的流式上报（客户端流）
  - `QueryFlows()` - 查询流记录（可选，也可以通过 HTTP API）

**4. Server 端 - 存储层**
- 新增 PostgreSQL/TimescaleDB 存储：
  - `flows` 表（TimescaleDB Hypertable，时序优化）
  - 索引优化（时间、IP、标签等）

**5. Server 端 - HTTP API 层**
- 新增 Flow API 端点：
  - `GET /api/v1/flows` - 查询流列表（全局）
  - `GET /api/v1/flows/:id` - 获取单条流记录
  - `GET /api/v1/flows/summary` - 流量统计摘要（全局）
  - `GET /api/v1/flows/dependencies` - 应用依赖关系（全局）
  - `WS /api/v1/flows/stream` - 实时流推送

**6. Server 端 - 聚合分析**
- 新增 `pkg/aggregator/` 包：
  - `flow_aggregator.go` - 跨节点流量聚合
  - `dependency_analyzer.go` - 全局依赖关系分析

### 修改的组件

**1. Agent 端 - eBPF 程序 (`tc_microsegment.bpf.c`)**
- 修改：在连接建立/关闭时通过 Ring Buffer 推送流事件
- 影响：轻微性能影响（需测试）

**2. Agent 端 - Agent 主程序 (`src/agent/cmd/main.go`)**
- 修改：启动 Flow Collector 和 gRPC Reporter
- 影响：增加内存和 CPU 开销（Ring Buffer 读取 + gRPC 上报）

**3. Agent 端 - Workload Manager (`pkg/workload/`)**
- 修改：Flow Collector 需要查询工作负载标签
- 影响：需要提供高效的 IP → Labels 查询接口

**4. Server 端 - gRPC Server (`src/server/pkg/grpc/`)**
- 修改：实现 FlowService 接收流事件上报
- 影响：增加 gRPC 服务端负载

**5. Server 端 - Server 主程序 (`src/server/cmd/main.go`)**
- 修改：初始化 Flow 存储和聚合器
- 影响：增加 Server 初始化复杂度

### 不修改的组件
- ✅ Policy 管理（`pkg/policy/`）
- ✅ 现有的 Session 跟踪逻辑
- ✅ 现有的统计信息收集（`stats_map`）

## 架构概览 (Agent-Server 架构)

```
┌──────────────────────────────────────────────────────────────────┐
│                       Frontend (React)                            │
│  - Application Dependency Map (ADM) - 全局拓扑                   │
│  - Traffic Flow Table - 跨节点流量查询                           │
│  - Real-time Flow Stream - 实时流推送                            │
└────────────────────┬─────────────────────────────────────────────┘
                     │ REST API / WebSocket
                     │
┌────────────────────▼─────────────────────────────────────────────┐
│            microsegment-server (控制平面)                         │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  HTTP API Layer (Gin) - :8080                              │ │
│  │  - GET /api/v1/flows (全局查询)                            │ │
│  │  - GET /api/v1/flows/dependencies (全局依赖)               │ │
│  │  - WS  /api/v1/flows/stream (实时推送)                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  gRPC Server - :9090                                        │ │
│  │  - FlowService.ReportFlowEvents() (接收 Agent 上报)        │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Storage (PostgreSQL + TimescaleDB)                        │ │
│  │  - flows 表 (Hypertable, 时序优化)                         │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Aggregator (跨节点聚合)                                   │ │
│  │  - Flow Aggregator, Dependency Analyzer                   │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────▲───────────────────────────────────────────┘
                       │ gRPC (流式上报)
          ┌────────────┼────────────┬─────────────┐
          │            │            │             │
┌─────────▼───┐  ┌────▼────────┐  ┌▼────────┐  ┌─▼─────────┐
│  Agent 1    │  │  Agent 2    │  │ Agent 3 │  │  Agent N  │
│  (Node 1)   │  │  (Node 2)   │  │ (Node 3)│  │  (Node N) │
│ ┌─────────┐ │  │ ┌─────────┐ │  │┌───────┐│  │┌─────────┐│
│ │Flow     │ │  │ │Flow     │ │  ││Flow   ││  ││Flow     ││
│ │Collector│ │  │ │Collector│ │  ││Collect││  ││Collector││
│ │         │ │  │ │         │ │  ││or     ││  ││         ││
│ │+        │ │  │ │+        │ │  ││       ││  ││+        ││
│ │gRPC     │ │  │ │gRPC     │ │  ││+gRPC  ││  ││gRPC     ││
│ │Reporter │ │  │ │Reporter │ │  ││Report ││  ││Reporter ││
│ └────▲────┘ │  │ └────▲────┘ │  │└───▲───┘│  │└────▲────┘│
│      │      │  │      │      │  │    │    │  │     │     │
│ ┌────┴────┐ │  │ ┌────┴────┐ │  │┌───┴───┐│  │┌────┴────┐│
│ │eBPF Ring│ │  │ │eBPF Ring│ │  ││eBPF   ││  ││eBPF Ring││
│ │Buffer   │ │  │ │Buffer   │ │  ││Ring   ││  ││Buffer   ││
│ │(Kernel) │ │  │ │(Kernel) │ │  ││Buffer ││  ││(Kernel) ││
│ └─────────┘ │  │ └─────────┘ │  │└───────┘│  │└─────────┘│
└─────────────┘  └─────────────┘  └─────────┘  └───────────┘
```

**数据流向**：
1. 各节点 eBPF 捕获流事件 → Ring Buffer
2. Agent Flow Collector 读取 Ring Buffer → 丰富标签 → gRPC Reporter
3. gRPC Reporter 批量上报到 Server (流式 gRPC)
4. Server 接收流事件 → 存储到 PostgreSQL/TimescaleDB
5. Server 提供 HTTP API 供前端查询全局流量
6. Server 通过 WebSocket 实时推送新流事件到前端

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

### Phase 1: Agent 端 - 基础 Flow 收集（Week 1）
- [ ] 定义 Flow 数据结构（eBPF + Go）
- [ ] 在 eBPF 中实现 Ring Buffer 推送逻辑
- [ ] 实现 Agent Flow Collector（读取 Ring Buffer）
- [ ] 实现 Agent gRPC Reporter（批量上报到 Server）

**验收标准**：Agent 能够从 eBPF 收集流事件并通过 gRPC 上报到 Server

### Phase 2: Server 端 - Flow 接收和存储（Week 2）
- [ ] 实现 Server FlowService gRPC 接口
- [ ] 实现 PostgreSQL/TimescaleDB 存储层
- [ ] 创建 flows 表 schema 和索引
- [ ] 实现批量写入优化

**验收标准**：Server 能够接收 Agent 上报并持久化到数据库

### Phase 3: Server 端 - Flow 查询 API（Week 3）
- [ ] 实现 HTTP Flow 查询 API（分页、过滤）
- [ ] 实现标签过滤查询（JSONB 查询）
- [ ] 实现时间范围查询（TimescaleDB 优化）
- [ ] 添加单元测试和集成测试

**验收标准**：前端能够通过 Server API 查询全局流记录

### Phase 4: Server 端 - 实时流推送（Week 4）
- [ ] 实现 WebSocket Hub 和实时推送
- [ ] 实现流事件缓冲和批处理
- [ ] 添加客户端重连逻辑
- [ ] 实现订阅过滤机制

**验收标准**：前端能够实时接收新流事件（延迟 < 500ms）

### Phase 5: Server 端 - 全局依赖分析（Week 5）
- [ ] 实现跨节点 Flow 聚合逻辑（按标签分组）
- [ ] 实现全局应用依赖关系 API
- [ ] 实现 Top Talkers 分析
- [ ] 优化性能（缓存、索引、物化视图）

**验收标准**：前端能够渲染全局应用依赖地图（ADM）

### Phase 6: 端到端集成测试（Week 6）
- [ ] 多节点 Agent → Server 流量上报测试
- [ ] 全局查询 API 压力测试
- [ ] WebSocket 并发连接测试
- [ ] 数据一致性验证

**验收标准**：完整的 Agent-Server 流量收集和查询系统正常工作

## 风险与缓解

### 风险 1: Agent 性能影响
- **问题**：Ring Buffer 推送和 gRPC 上报可能影响 Agent 性能
- **缓解**：
  - 只在连接建立/关闭时推送（不是每个数据包）
  - 使用 Ring Buffer 而非 Perf Buffer（更高效）
  - gRPC 批量上报（减少网络开销）
  - 配置可调的批处理参数

### 风险 2: gRPC 网络开销
- **问题**：大量 Agent 同时上报可能导致 Server 过载
- **缓解**：
  - 使用流式 gRPC（客户端流）减少连接数
  - Agent 端批量缓冲（1000 条/批或 1 秒超时）
  - Server 端限流和背压机制
  - 监控 gRPC 连接和延迟指标

### 风险 3: Server 存储容量
- **问题**：Flow 数据快速增长可能耗尽磁盘空间
- **缓解**：
  - TimescaleDB 数据保留策略（默认 30 天）
  - 提供聚合存储（每小时统计物化视图）
  - TimescaleDB 数据压缩
  - 监控磁盘使用率告警

### 风险 4: Server 查询性能
- **问题**：大量 Flow 记录可能导致查询缓慢
- **缓解**：
  - TimescaleDB Hypertable 时序优化
  - 在关键字段上创建索引（时间、IP、标签）
  - 实现查询结果缓存
  - 限制查询时间范围（默认 1 小时）
  - 使用物化视图加速聚合查询

### 风险 5: 数据一致性
- **问题**：Agent 断线或 Server 故障可能导致数据丢失
- **缓解**：
  - Agent 本地缓冲（短期内存缓存）
  - gRPC 重连和重试机制
  - Server 端事务写入保证原子性
  - 监控 Agent 上报成功率

## 替代方案

### 方案 A: Agent 本地 SQLite 存储（不采纳）
- ❌ 数据孤岛：每个 Agent 数据独立，无法跨节点查询
- ❌ 无全局视图：前端无法查看整个集群的流量
- ❌ 扩展性差：需要逐个查询每个 Agent 再聚合
- ❌ 管理复杂：需要管理多个 SQLite 数据库

**采纳 Agent-Server 架构的原因**：
- ✅ 集中存储：所有流量数据存储在 Server 的 PostgreSQL
- ✅ 全局查询：一次查询获取所有节点数据
- ✅ 扩展性好：支持 10-10000 节点
- ✅ 统一管理：单一数据源，易于备份和维护

### 方案 B: Agent 提供 HTTP API，前端聚合（不采纳）
- ❌ 前端负担：需要查询所有 Agent 并在前端聚合
- ❌ 性能差：N 个 Agent = N 次 HTTP 请求
- ❌ 网络开销：前端到每个 Agent 的连接
- ❌ 一致性难：部分 Agent 失败时处理复杂

**采纳 Server 集中 API 的原因**：
- ✅ 前端简单：单一 API 入口
- ✅ 性能好：Server 端聚合更高效
- ✅ 网络优化：Agent → Server 内网 gRPC
- ✅ 一致性好：Server 提供统一的数据视图

### 方案 C: 使用 Perf Buffer 而非 Ring Buffer（不采纳）
- ❌ Ring Buffer 是更现代、更高效的选择
- ❌ Perf Buffer 在高负载下容易丢失事件

### 方案 D: 不实现实时推送，仅提供轮询 API（不采纳）
- ❌ 前端需要频繁轮询，浪费资源
- ❌ 延迟较高，用户体验差

## 测试策略

### Agent 端单元测试
- Flow Collector 的事件解析逻辑
- gRPC Reporter 的批量上报逻辑
- 标签丰富化逻辑

### Server 端单元测试
- FlowService gRPC 接收逻辑
- PostgreSQL 存储的 CRUD 操作
- HTTP API 查询过滤逻辑
- WebSocket Hub 推送逻辑

### Agent-Server 集成测试
- eBPF → Agent Collector → gRPC Reporter → Server 的端到端流程
- 多 Agent 同时上报到 Server
- Server API 查询准确性
- WebSocket 推送可靠性

### 性能测试
- **Agent 端**：模拟 10,000 flows/s 的收集和上报能力
- **Server 端**：
  - 接收 100 个 Agent 同时上报（10,000 flows/s × 100 = 1M flows/s）
  - 查询响应时间（1000 条记录 < 100ms）
  - PostgreSQL 写入性能和磁盘使用
  - WebSocket 并发连接（1000+ clients）

### 端到端场景测试
- 3 节点 Agent → 1 Server → 前端查询全局流量
- Agent 断线重连和数据恢复
- Server 故障和数据一致性

## 相关文档

- `openspec/changes/add-server-component/` - Server 组件实现（必需前置依赖）
- `openspec/changes/add-grpc-protocol-definitions/` - gRPC 协议定义（已完成）
- `docs/archive/mvp-plan.md` - Phase 4 前端可视化需求
- `openspec/project.md` - 项目架构和命名约定
- `src/agent/pkg/workload/` - 工作负载管理（用于标签查询）
- `src/agent/pkg/policy/` - 策略管理（用于 PolicyID 关联）
- `src/server/pkg/grpc/` - Server 端 gRPC 服务
- `src/server/pkg/storage/` - Server 端存储层

## 依赖关系

**前置依赖**：
- ✅ `add-grpc-protocol-definitions` - gRPC proto 定义已完成
- ⏳ `add-server-component` - Server 组件实现（并行开发）

**后续依赖**：
- `add-deployment-configurations` - 部署配置会使用 Flow 数据

## 批准状态

- [ ] 技术审查通过
- [ ] 架构设计审查通过（Agent-Server 架构）
- [ ] 准备开始实施

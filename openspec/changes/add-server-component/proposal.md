# 提案: 添加 Server 组件

**变更 ID**: `add-server-component`
**提案日期**: 2025-11-04
**状态**: 提案中
**优先级**: P0 (关键路径)
**预估工作量**: 7-10 天

---

## 📋 概述

实现独立的 Server 组件（第二个可执行文件 `microsegment-server`）,作为 Agent-Server 架构的中心控制节点。Server 负责接收所有 Agent 的流事件上报、策略分发、全局数据聚合和统一 API 服务。

## 🎯 目标

1. **实现第二个可执行文件** - `microsegment-server` (独立于 `microsegment-agent`)
2. **实现 gRPC 服务端** - FlowService, PolicyService, AgentService
3. **实现中心存储** - PostgreSQL + TimescaleDB 替代 SQLite
4. **实现全局 API** - 跨节点流量查询、依赖分析、Agent 管理
5. **实现数据聚合** - 全局流量统计、Top Talkers、依赖关系图

## 💡 动机

### 当前架构限制
- **单节点数据孤岛** - 每个 Agent 数据存储在本地 SQLite,无法跨节点查询
- **策略配置分散** - 需要逐节点配置策略,容易不一致
- **无全局视图** - 无法查看整个集群的流量和依赖关系

### Agent-Server 架构优势
- ✅ **集中管理** - 统一策略下发,保证一致性
- ✅ **全局视图** - 跨节点流量分析和依赖关系图
- ✅ **可扩展** - 支持 10-10000 节点
- ✅ **统一监控** - 集中告警和可观测性

## 🏗️ 核心设计

### 架构图

```
                          microsegment-server
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  HTTP API Server (Gin) - :8080                     │ │
│  │  ├─ GET  /api/v1/flows (全局流量)                  │ │
│  │  ├─ GET  /api/v1/dependencies (全局依赖)           │ │
│  │  ├─ GET  /api/v1/agents (Agent 列表)               │ │
│  │  └─ POST /api/v1/policies (策略下发)               │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  gRPC Server - :9090                                │ │
│  │  ├─ FlowService (接收流事件)                        │ │
│  │  ├─ PolicyService (策略分发)                        │ │
│  │  └─ AgentService (Agent 管理)                       │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Central Storage (PostgreSQL + TimescaleDB)        │ │
│  │  ├─ flows (时序表)                                  │ │
│  │  ├─ policies (关系表)                               │ │
│  │  ├─ agents (关系表)                                 │ │
│  │  └─ events (审计日志)                               │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
└──────────────────────────────────────────────────────────┘
              ↑ gRPC (流事件上报、策略同步、心跳)
    ┌─────────┼─────────┬─────────┐
    │         │         │         │
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│Agent 1│ │Agent 2│ │Agent 3│ │Agent N│
└───────┘ └───────┘ └───────┘ └───────┘
```

### 目录结构

```
src/server/                     # 新增 Server 组件
├── cmd/
│   └── main.go                # Server 入口
├── pkg/
│   ├── grpc/                  # gRPC 服务实现
│   │   ├── flow_service.go
│   │   ├── policy_service.go
│   │   └── agent_service.go
│   ├── storage/               # 中心存储
│   │   ├── postgres.go
│   │   ├── flow_storage.go
│   │   ├── policy_storage.go
│   │   └── agent_storage.go
│   ├── api/                   # HTTP API
│   │   ├── server.go
│   │   ├── router.go
│   │   └── handlers/
│   │       ├─ flow.go
│   │       ├─ policy.go
│   │       └─ agent.go
│   ├── aggregator/            # 数据聚合
│   │   ├── flow_aggregator.go
│   │   └── dependency.go
│   └── config/                # Server 配置
│       └── config.go
├── migrations/                # 数据库迁移脚本
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── go.mod
└── go.sum
```

## 📂 核心组件

### 1. gRPC 服务实现

#### FlowService
```go
type flowService struct {
    storage storage.FlowStorage
    pb.UnimplementedFlowServiceServer
}

func (s *flowService) ReportFlowEvents(stream pb.FlowService_ReportFlowEventsServer) error {
    // 批量接收流事件
    // 保存到 PostgreSQL/TimescaleDB
    // 返回确认
}
```

#### PolicyService
```go
type policyService struct {
    storage storage.PolicyStorage
    pb.UnimplementedPolicyServiceServer
}

func (s *policyService) SyncPolicies(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
    // 查询策略列表
    // 返回给 Agent
}

func (s *policyService) SubscribePolicies(req *pb.SubscribeRequest, stream pb.PolicyService_SubscribePoliciesServer) error {
    // 推送策略更新
}
```

#### AgentService
```go
type agentService struct {
    storage storage.AgentStorage
    pb.UnimplementedAgentServiceServer
}

func (s *agentService) RegisterAgent(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // 注册 Agent
    // 返回配置
}

func (s *agentService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    // 更新 Agent 状态
    // 检查健康
}
```

### 2. PostgreSQL 数据库设计

#### flows 表 (TimescaleDB 时序表)
```sql
CREATE TABLE flows (
    time          TIMESTAMPTZ NOT NULL,
    id            TEXT NOT NULL,
    agent_id      TEXT NOT NULL,
    source_ip     INET NOT NULL,
    source_port   INTEGER,
    dest_ip       INET NOT NULL,
    dest_port     INTEGER,
    protocol      TEXT,
    packet_count  BIGINT,
    byte_count    BIGINT,
    duration_ms   BIGINT,
    state         TEXT,
    policy_action TEXT,
    source_labels JSONB,
    dest_labels   JSONB
);

-- 转换为时序表
SELECT create_hypertable('flows', 'time');

-- 索引
CREATE INDEX ON flows (agent_id, time DESC);
CREATE INDEX ON flows (source_ip, time DESC);
CREATE INDEX ON flows (dest_ip, time DESC);
CREATE INDEX ON flows USING GIN (source_labels);
CREATE INDEX ON flows USING GIN (dest_labels);
```

#### agents 表
```sql
CREATE TABLE agents (
    agent_id         TEXT PRIMARY KEY,
    hostname         TEXT,
    version          TEXT,
    interface        TEXT,
    ip_addresses     TEXT[],
    status           TEXT,
    last_heartbeat   TIMESTAMPTZ,
    registered_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
```

#### policies 表
```sql
CREATE TABLE policies (
    rule_id       INTEGER PRIMARY KEY,
    src_ip        CIDR,
    dst_ip        CIDR,
    src_port      INTEGER,
    dst_port      INTEGER,
    protocol      TEXT,
    action        TEXT,
    priority      INTEGER,
    version       BIGINT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. HTTP API 端点

```go
// 全局流量查询
GET /api/v1/flows?start_time=...&end_time=...&agent_id=...

// 全局依赖分析
GET /api/v1/dependencies?start_time=...&end_time=...

// Agent 列表
GET /api/v1/agents

// Agent 详情
GET /api/v1/agents/:id

// 策略管理
POST /api/v1/policies
GET  /api/v1/policies
PUT  /api/v1/policies/:id
DELETE /api/v1/policies/:id
```

## ✅ 验收标准

- [ ] Server 编译生成独立可执行文件 `microsegment-server`
- [ ] gRPC Server 监听 :9090 端口
- [ ] HTTP API Server 监听 :8080 端口
- [ ] PostgreSQL 数据库初始化成功
- [ ] 所有 gRPC 服务实现并可调用
- [ ] 全局流量查询 API 正常工作
- [ ] Agent 注册和心跳正常
- [ ] 策略分发机制正常

## 🔗 依赖

**前置依赖**: add-grpc-protocol-definitions

**后续依赖**: refactor-agent-for-remote-reporting

---

**提案人**: Claude Code
**下一步**: 创建 design.md 和 tasks.md

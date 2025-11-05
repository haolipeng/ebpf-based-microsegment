# 设计文档: Server 组件实现

**变更 ID**: `add-server-component`
**设计日期**: 2025-11-04
**版本**: 1.0

---

## 📐 架构设计

### 1. 总体架构

```
┌────────────────────────────────────────────────────────────────┐
│               microsegment-server                              │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Entrypoint Layer                                        │ │
│  │  ├─ cmd/main.go (启动入口)                                │ │
│  │  ├─ 配置加载 (YAML/ENV)                                   │ │
│  │  └─ 组件初始化                                            │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  API Layer (双协议支持)                                   │ │
│  │  ├─ gRPC Server (:9090)                                  │ │
│  │  │  ├─ FlowService                                       │ │
│  │  │  ├─ PolicyService                                     │ │
│  │  │  └─ AgentService                                      │ │
│  │  └─ HTTP Server (:8080)                                  │ │
│  │     ├─ GET  /api/v1/flows (全局查询)                     │ │
│  │     ├─ GET  /api/v1/dependencies                         │ │
│  │     ├─ GET  /api/v1/agents                               │ │
│  │     └─ POST /api/v1/policies                             │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Business Logic Layer                                    │ │
│  │  ├─ Agent Manager (注册、健康检查)                        │ │
│  │  ├─ Flow Aggregator (全局聚合)                           │ │
│  │  ├─ Policy Distributor (策略分发)                        │ │
│  │  └─ Dependency Analyzer (依赖分析)                       │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Storage Layer                                           │ │
│  │  ├─ FlowStorage (TimescaleDB)                           │ │
│  │  ├─ PolicyStorage (PostgreSQL)                          │ │
│  │  ├─ AgentStorage (PostgreSQL)                           │ │
│  │  └─ EventStorage (PostgreSQL)                           │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Database (PostgreSQL + TimescaleDB)                     │ │
│  │  ├─ flows (Hypertable, 时序数据)                         │ │
│  │  ├─ policies (关系表)                                     │ │
│  │  ├─ agents (关系表)                                       │ │
│  │  └─ events (审计日志)                                     │ │
│  └──────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
                    ↑ gRPC (:9090)
         ┌──────────┼──────────┬──────────┐
         │          │          │          │
    ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
    │Agent 1 │ │Agent 2 │ │Agent 3 │ │Agent N │
    └────────┘ └────────┘ └────────┘ └────────┘
```

---

## 📂 目录结构详细设计

```
src/server/
├── cmd/
│   └── main.go                      # Server 入口点
│
├── pkg/
│   ├── grpc/                        # gRPC 服务层
│   │   ├── flow_service.go          # FlowService 实现
│   │   ├── policy_service.go        # PolicyService 实现
│   │   ├── agent_service.go         # AgentService 实现
│   │   └── interceptor.go           # gRPC 拦截器 (日志、认证)
│   │
│   ├── api/                         # HTTP API 层
│   │   ├── server.go                # HTTP 服务器
│   │   ├── router.go                # 路由配置
│   │   ├── middleware.go            # 中间件 (CORS、认证)
│   │   └── handlers/
│   │       ├── flow.go              # 流量查询 Handler
│   │       ├── policy.go            # 策略管理 Handler
│   │       ├── agent.go             # Agent 管理 Handler
│   │       └── health.go            # 健康检查 Handler
│   │
│   ├── storage/                     # 存储层
│   │   ├── postgres.go              # PostgreSQL 连接管理
│   │   ├── flow_storage.go          # Flow 存储实现
│   │   ├── policy_storage.go        # Policy 存储实现
│   │   ├── agent_storage.go         # Agent 存储实现
│   │   └── event_storage.go         # Event 审计日志存储
│   │
│   ├── manager/                     # 业务逻辑层
│   │   ├── agent_manager.go         # Agent 生命周期管理
│   │   ├── policy_distributor.go    # 策略分发管理
│   │   └── health_monitor.go        # Agent 健康监控
│   │
│   ├── aggregator/                  # 数据聚合层
│   │   ├── flow_aggregator.go       # 流量聚合
│   │   ├── dependency_analyzer.go   # 依赖关系分析
│   │   └── statistics.go            # 统计数据计算
│   │
│   └── config/                      # 配置管理
│       ├── config.go                # 配置结构体
│       └── loader.go                # 配置加载器
│
├── migrations/                      # 数据库迁移脚本
│   ├── 001_initial_schema.up.sql   # 初始化 schema
│   ├── 001_initial_schema.down.sql # 回滚脚本
│   ├── 002_add_indexes.up.sql      # 索引优化
│   └── 002_add_indexes.down.sql
│
├── config/                          # 配置文件示例
│   ├── server.yaml.example          # Server 配置模板
│   └── database.yaml.example        # 数据库配置模板
│
├── go.mod
├── go.sum
└── README.md
```

---

## 💾 数据库设计

### 1. flows 表 (TimescaleDB Hypertable)

```sql
-- 流量数据表 (时序表)
CREATE TABLE flows (
    -- 时间维度 (TimescaleDB 必需)
    time              TIMESTAMPTZ NOT NULL,

    -- 流标识
    id                TEXT NOT NULL,
    agent_id          TEXT NOT NULL,

    -- 5-tuple
    source_ip         INET NOT NULL,
    source_port       INTEGER,
    dest_ip           INET NOT NULL,
    dest_port         INTEGER,
    protocol          TEXT,

    -- 流量统计
    packet_count      BIGINT DEFAULT 0,
    byte_count        BIGINT DEFAULT 0,
    duration_ms       BIGINT,

    -- 时间戳
    start_time        TIMESTAMPTZ,
    end_time          TIMESTAMPTZ,
    last_seen         TIMESTAMPTZ,

    -- 状态
    state             TEXT,           -- ACTIVE/CLOSED/TIMEOUT
    direction         TEXT,           -- INGRESS/EGRESS

    -- 策略
    policy_id         INTEGER,
    policy_action     TEXT,           -- ALLOW/DENY/LOG

    -- 标签 (JSONB 用于灵活查询)
    source_labels     JSONB,
    dest_labels       JSONB,

    -- 约束
    PRIMARY KEY (time, id)
);

-- 转换为 TimescaleDB Hypertable (按时间分区)
SELECT create_hypertable('flows', 'time', chunk_time_interval => INTERVAL '1 day');

-- 数据保留策略 (保留 30 天)
SELECT add_retention_policy('flows', INTERVAL '30 days');

-- 索引优化
CREATE INDEX idx_flows_agent_id_time ON flows (agent_id, time DESC);
CREATE INDEX idx_flows_source_ip_time ON flows (source_ip, time DESC);
CREATE INDEX idx_flows_dest_ip_time ON flows (dest_ip, time DESC);
CREATE INDEX idx_flows_policy_id ON flows (policy_id);
CREATE INDEX idx_flows_state ON flows (state);

-- JSONB 标签索引 (GIN 索引用于高效 JSON 查询)
CREATE INDEX idx_flows_source_labels ON flows USING GIN (source_labels);
CREATE INDEX idx_flows_dest_labels ON flows USING GIN (dest_labels);

-- 物化视图: 每小时流量汇总
CREATE MATERIALIZED VIEW flows_hourly_summary
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS hour,
    agent_id,
    protocol,
    policy_action,
    COUNT(*) AS flow_count,
    SUM(packet_count) AS total_packets,
    SUM(byte_count) AS total_bytes
FROM flows
GROUP BY hour, agent_id, protocol, policy_action;

-- 自动刷新物化视图
SELECT add_continuous_aggregate_policy('flows_hourly_summary',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');
```

### 2. policies 表

```sql
CREATE TABLE policies (
    -- 主键
    rule_id           INTEGER PRIMARY KEY,

    -- 规则定义
    src_ip            CIDR,           -- CIDR 格式 (支持网段)
    dst_ip            CIDR,
    src_port          INTEGER,        -- 0 表示任意端口
    dst_port          INTEGER,
    protocol          TEXT,           -- TCP/UDP/ICMP/ANY
    action            TEXT NOT NULL,  -- ALLOW/DENY/LOG
    priority          INTEGER DEFAULT 0,

    -- 版本控制
    version           BIGINT NOT NULL DEFAULT 0,

    -- 元数据
    description       TEXT,
    metadata          JSONB,          -- 扩展字段

    -- 审计
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    created_by        TEXT,

    -- 约束
    CHECK (action IN ('ALLOW', 'DENY', 'LOG')),
    CHECK (protocol IN ('TCP', 'UDP', 'ICMP', 'ANY'))
);

-- 索引
CREATE INDEX idx_policies_version ON policies (version);
CREATE INDEX idx_policies_priority ON policies (priority DESC);
CREATE INDEX idx_policies_updated_at ON policies (updated_at DESC);

-- 触发器: 自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 触发器: 自动递增版本号
CREATE OR REPLACE FUNCTION increment_policy_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.version = OLD.version + 1;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER increment_policy_version_trigger BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION increment_policy_version();
```

### 3. agents 表

```sql
CREATE TABLE agents (
    -- 主键
    agent_id          TEXT PRIMARY KEY,

    -- 基本信息
    hostname          TEXT NOT NULL,
    version           TEXT,
    interface         TEXT,

    -- 网络信息
    ip_addresses      TEXT[],         -- 数组类型

    -- 系统信息
    os                TEXT,
    kernel_version    TEXT,

    -- 能力标识
    capabilities      JSONB,          -- 支持的功能

    -- 状态
    status            TEXT NOT NULL DEFAULT 'UNKNOWN',
    last_heartbeat    TIMESTAMPTZ,

    -- 时间戳
    registered_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),

    -- 约束
    CHECK (status IN ('HEALTHY', 'DEGRADED', 'UNHEALTHY', 'STOPPING', 'UNKNOWN'))
);

-- 索引
CREATE INDEX idx_agents_status ON agents (status);
CREATE INDEX idx_agents_last_heartbeat ON agents (last_heartbeat DESC);
CREATE INDEX idx_agents_registered_at ON agents (registered_at DESC);

-- 触发器: 自动更新 updated_at
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 视图: 在线 Agents
CREATE VIEW online_agents AS
SELECT *
FROM agents
WHERE last_heartbeat > NOW() - INTERVAL '30 seconds'
  AND status IN ('HEALTHY', 'DEGRADED');
```

### 4. events 表 (审计日志)

```sql
CREATE TABLE events (
    -- 主键
    id                BIGSERIAL PRIMARY KEY,

    -- 事件信息
    event_type        TEXT NOT NULL,  -- AGENT_REGISTERED, POLICY_CREATED, ...
    event_source      TEXT NOT NULL,  -- AGENT, API, SYSTEM
    agent_id          TEXT,

    -- 事件内容
    message           TEXT,
    metadata          JSONB,

    -- 严重级别
    severity          TEXT NOT NULL DEFAULT 'INFO',

    -- 时间戳
    created_at        TIMESTAMPTZ DEFAULT NOW(),

    -- 约束
    CHECK (severity IN ('DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE SET NULL
);

-- 索引
CREATE INDEX idx_events_event_type ON events (event_type);
CREATE INDEX idx_events_agent_id ON events (agent_id);
CREATE INDEX idx_events_severity ON events (severity);
CREATE INDEX idx_events_created_at ON events (created_at DESC);

-- 数据保留: 仅保留 90 天日志
CREATE OR REPLACE FUNCTION delete_old_events()
RETURNS void AS $$
BEGIN
    DELETE FROM events WHERE created_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- 定时任务: 每天清理旧日志
-- (需要配合 pg_cron 扩展)
```

---

## 🔧 核心组件实现

### 1. gRPC 服务实现

#### FlowService 实现

```go
package grpc

import (
    "context"
    "io"
    "log"

    pb "github.com/ebpf-microsegment/src/proto/flow"
    "github.com/ebpf-microsegment/src/server/pkg/storage"
)

type FlowService struct {
    pb.UnimplementedFlowServiceServer
    storage storage.FlowStorage
}

func NewFlowService(storage storage.FlowStorage) *FlowService {
    return &FlowService{storage: storage}
}

// ReportFlowEvents - 批量接收流事件 (客户端流)
func (s *FlowService) ReportFlowEvents(stream pb.FlowService_ReportFlowEventsServer) error {
    var (
        acceptedCount uint64 = 0
        rejectedCount uint64 = 0
        batch         []*pb.FlowEvent
    )

    // 批量接收
    for {
        event, err := stream.Recv()
        if err == io.EOF {
            // 客户端关闭流,保存最后一批
            if len(batch) > 0 {
                if err := s.storage.BatchSaveFlowEvents(batch); err != nil {
                    log.Printf("Failed to save batch: %v", err)
                    rejectedCount += uint64(len(batch))
                } else {
                    acceptedCount += uint64(len(batch))
                }
            }

            // 返回响应
            return stream.SendAndClose(&pb.ReportResponse{
                Success:       true,
                Message:       "Flow events received",
                AcceptedCount: acceptedCount,
                RejectedCount: rejectedCount,
            })
        }

        if err != nil {
            log.Printf("Error receiving flow event: %v", err)
            return err
        }

        // 添加到批次
        batch = append(batch, event)

        // 批次大小达到阈值,执行写入
        if len(batch) >= 1000 {
            if err := s.storage.BatchSaveFlowEvents(batch); err != nil {
                log.Printf("Failed to save batch: %v", err)
                rejectedCount += uint64(len(batch))
            } else {
                acceptedCount += uint64(len(batch))
            }
            batch = batch[:0] // 清空批次
        }
    }
}

// QueryFlows - 查询流量
func (s *FlowService) QueryFlows(ctx context.Context, req *pb.FlowQuery) (*pb.FlowQueryResponse, error) {
    flows, totalCount, err := s.storage.QueryFlows(ctx, req)
    if err != nil {
        return nil, err
    }

    return &pb.FlowQueryResponse{
        Flows:      flows,
        TotalCount: uint32(totalCount),
        QueryInfo: &pb.FlowQueryInfo{
            Limit:     req.Limit,
            Offset:    req.Offset,
            SortBy:    req.SortBy,
            SortOrder: req.SortOrder,
        },
    }, nil
}

// GetFlowSummary - 获取流量汇总
func (s *FlowService) GetFlowSummary(ctx context.Context, req *pb.FlowSummaryRequest) (*pb.FlowSummary, error) {
    return s.storage.GetFlowSummary(ctx, req)
}
```

#### PolicyService 实现

```go
package grpc

import (
    "context"
    "log"
    "sync"

    pb "github.com/ebpf-microsegment/src/proto/policy"
    "github.com/ebpf-microsegment/src/server/pkg/storage"
)

type PolicyService struct {
    pb.UnimplementedPolicyServiceServer
    storage     storage.PolicyStorage
    subscribers map[string]chan *pb.PolicyUpdate  // agent_id -> channel
    subMutex    sync.RWMutex
}

func NewPolicyService(storage storage.PolicyStorage) *PolicyService {
    return &PolicyService{
        storage:     storage,
        subscribers: make(map[string]chan *pb.PolicyUpdate),
    }
}

// SyncPolicies - 同步完整策略列表
func (s *PolicyService) SyncPolicies(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
    policies, version, err := s.storage.GetAllPolicies(ctx)
    if err != nil {
        return nil, err
    }

    return &pb.SyncResponse{
        Policies:      policies,
        PolicyVersion: version,
        ServerTime:    time.Now().Unix(),
    }, nil
}

// SubscribePolicies - 订阅策略更新 (服务器流)
func (s *PolicyService) SubscribePolicies(req *pb.SubscribeRequest, stream pb.PolicyService_SubscribePoliciesServer) error {
    agentID := req.AgentId

    // 创建订阅 channel
    updateChan := make(chan *pb.PolicyUpdate, 100)

    s.subMutex.Lock()
    s.subscribers[agentID] = updateChan
    s.subMutex.Unlock()

    defer func() {
        s.subMutex.Lock()
        delete(s.subscribers, agentID)
        close(updateChan)
        s.subMutex.Unlock()
    }()

    log.Printf("Agent %s subscribed to policy updates", agentID)

    // 首先发送当前所有策略
    policies, version, err := s.storage.GetAllPolicies(stream.Context())
    if err != nil {
        return err
    }

    for _, policy := range policies {
        if err := stream.Send(&pb.PolicyUpdate{
            UpdateType:    pb.PolicyUpdateType_POLICY_UPDATE_ADD,
            Policy:        policy,
            PolicyVersion: version,
        }); err != nil {
            return err
        }
    }

    // 持续监听更新
    for {
        select {
        case <-stream.Context().Done():
            return nil
        case update := <-updateChan:
            if err := stream.Send(update); err != nil {
                return err
            }
        }
    }
}

// BroadcastPolicyUpdate - 广播策略更新 (被 PolicyStorage 调用)
func (s *PolicyService) BroadcastPolicyUpdate(update *pb.PolicyUpdate) {
    s.subMutex.RLock()
    defer s.subMutex.RUnlock()

    for agentID, ch := range s.subscribers {
        select {
        case ch <- update:
            log.Printf("Sent policy update to agent %s", agentID)
        default:
            log.Printf("Warning: channel full for agent %s", agentID)
        }
    }
}

// ReportPolicyStats - 接收策略统计
func (s *PolicyService) ReportPolicyStats(ctx context.Context, req *pb.PolicyStatsReport) (*pb.ReportResponse, error) {
    if err := s.storage.SavePolicyStats(ctx, req); err != nil {
        return &pb.ReportResponse{
            Success: false,
            Message: err.Error(),
        }, err
    }

    return &pb.ReportResponse{
        Success:       true,
        Message:       "Policy stats received",
        AcceptedCount: uint64(len(req.Stats)),
    }, nil
}
```

#### AgentService 实现

```go
package grpc

import (
    "context"
    "log"
    "time"

    pb "github.com/ebpf-microsegment/src/proto/agent"
    "github.com/ebpf-microsegment/src/server/pkg/manager"
    "github.com/ebpf-microsegment/src/server/pkg/storage"
)

type AgentService struct {
    pb.UnimplementedAgentServiceServer
    storage storage.AgentStorage
    manager *manager.AgentManager
}

func NewAgentService(storage storage.AgentStorage, mgr *manager.AgentManager) *AgentService {
    return &AgentService{
        storage: storage,
        manager: mgr,
    }
}

// RegisterAgent - Agent 注册
func (s *AgentService) RegisterAgent(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    log.Printf("Agent registration: %s (version: %s)", req.AgentId, req.Version)

    // 保存 Agent 信息
    if err := s.storage.RegisterAgent(ctx, req); err != nil {
        return &pb.RegisterResponse{
            Success: false,
            Message: err.Error(),
        }, err
    }

    // 通知 AgentManager
    s.manager.OnAgentRegistered(req.AgentId)

    // 返回配置
    return &pb.RegisterResponse{
        Success:       true,
        Message:       "Agent registered successfully",
        ServerVersion: "1.0.0",
        ServerTime:    time.Now().Unix(),
        Config: &pb.AgentConfig{
            HeartbeatInterval: 10,   // 10 秒心跳
            StatsInterval:     60,   // 60 秒统计上报
            FlowBatchSize:     1000, // 批量大小
            FlowBatchTimeout:  1,    // 1 秒超时
        },
    }, nil
}

// Heartbeat - 心跳处理
func (s *AgentService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    // 更新最后心跳时间
    if err := s.storage.UpdateHeartbeat(ctx, req.AgentId, req.Metrics); err != nil {
        return &pb.HeartbeatResponse{
            Healthy: false,
            Message: err.Error(),
        }, err
    }

    // 检查是否有待下发的命令
    commands := s.manager.GetPendingCommands(req.AgentId)

    return &pb.HeartbeatResponse{
        Healthy:  true,
        Message:  "OK",
        Commands: commands,
    }, nil
}

// ReportStatus - 状态上报
func (s *AgentService) ReportStatus(ctx context.Context, req *pb.StatusReport) (*pb.StatusResponse, error) {
    log.Printf("Agent %s status: %s - %s", req.AgentId, req.Status.String(), req.Message)

    if err := s.storage.UpdateAgentStatus(ctx, req); err != nil {
        return &pb.StatusResponse{
            Acknowledged: false,
            Message:      err.Error(),
        }, err
    }

    return &pb.StatusResponse{
        Acknowledged: true,
        Message:      "Status received",
    }, nil
}

// UnregisterAgent - Agent 注销
func (s *AgentService) UnregisterAgent(ctx context.Context, req *pb.UnregisterRequest) (*pb.UnregisterResponse, error) {
    log.Printf("Agent unregistration: %s (reason: %s)", req.AgentId, req.Reason)

    if err := s.storage.UnregisterAgent(ctx, req.AgentId); err != nil {
        return &pb.UnregisterResponse{
            Success: false,
            Message: err.Error(),
        }, err
    }

    s.manager.OnAgentUnregistered(req.AgentId)

    return &pb.UnregisterResponse{
        Success: true,
        Message: "Agent unregistered successfully",
    }, nil
}
```

### 2. HTTP API 实现

#### Flow Handler

```go
package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ebpf-microsegment/src/server/pkg/storage"
)

type FlowHandler struct {
    storage storage.FlowStorage
}

func NewFlowHandler(storage storage.FlowStorage) *FlowHandler {
    return &FlowHandler{storage: storage}
}

// ListFlows - GET /api/v1/flows
func (h *FlowHandler) ListFlows(c *gin.Context) {
    // 解析查询参数
    var query storage.FlowQueryParams
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 查询数据库
    flows, total, err := h.storage.ListFlows(c.Request.Context(), &query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "flows": flows,
        "total": total,
        "query": query,
    })
}

// GetDependencies - GET /api/v1/dependencies
func (h *FlowHandler) GetDependencies(c *gin.Context) {
    startTimeStr := c.Query("start_time")
    endTimeStr := c.Query("end_time")

    startTime, _ := time.Parse(time.RFC3339, startTimeStr)
    endTime, _ := time.Parse(time.RFC3339, endTimeStr)

    dependencies, err := h.storage.GetDependencies(c.Request.Context(), startTime, endTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "dependencies": dependencies,
        "time_range": gin.H{
            "start_time": startTime,
            "end_time":   endTime,
        },
    })
}
```

### 3. 配置文件设计

```yaml
# config/server.yaml
server:
  # gRPC Server 配置
  grpc:
    host: "0.0.0.0"
    port: 9090
    max_recv_msg_size: 10485760  # 10MB
    max_send_msg_size: 10485760
    connection_timeout: 10s

  # HTTP Server 配置
  http:
    host: "0.0.0.0"
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    enable_cors: true

# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "microsegment"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300s

# 日志配置
logging:
  level: "info"  # debug, info, warn, error
  format: "json"
  output: "stdout"

# 数据保留策略
retention:
  flows_days: 30
  events_days: 90
```

---

## 📊 测试计划

### 单元测试
- [ ] gRPC 服务测试 (mock storage)
- [ ] HTTP API 测试
- [ ] Storage 层测试 (使用 testcontainers)
- [ ] Business Logic 测试

### 集成测试
- [ ] Agent-Server 通信测试
- [ ] 数据库事务测试
- [ ] 并发写入测试

### 性能测试
- [ ] 流事件批量写入 (目标: 10K/s)
- [ ] 全局流量查询 (目标: <100ms for 1000 records)
- [ ] 并发 Agent 连接 (目标: 1000+ agents)

---

**设计完成**: ✅
**下一步**: 创建 tasks.md

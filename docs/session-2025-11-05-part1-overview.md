# 会话记录：Agent 双模式架构实现 - Part 1: 概览

**日期**: 2025-11-05
**主题**: 实现 Agent 双模式架构（Standalone + Agent-Server）
**OpenSpec 提案**: `refactor-agent-for-remote-reporting`

---

## 会话概述

本次会话完成了微分段 Agent 的重大重构，从单一的 standalone 模式扩展为支持双模式运行：
1. **Standalone 模式**: 本地 SQLite 存储（原有行为）
2. **Agent-Server 模式**: 通过 gRPC 向中心服务器报告流事件（新功能）

### 会话流程

1. **继续之前的工作** - 从上一个会话恢复，完成文档编写
2. **创建核心组件**:
   - Reporter 接口和实现（LocalReporter, GRPCReporter）
   - AgentClient（服务器通信管理）
   - 配置系统（YAML + 环境变量）
3. **更新主入口点** - 支持模式选择
4. **编写文档** - 用户指南、技术总结、设计文档
5. **归档提案** - 完成核心实现后归档

---

## 主要成果

### 1. 架构设计

#### Standalone 模式架构
```
┌─────────────────────┐
│  eBPF Flow Collector │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│   LocalReporter     │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│  SQLite Storage     │
└─────────────────────┘
```

**特点**:
- 简单单节点部署
- 本地 SQLite 数据库
- 无外部依赖
- 适合开发/测试

#### Agent-Server 模式架构
```
┌─────────────────────┐
│  eBPF Flow Collector │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│   GRPCReporter      │ ────► Batch Queue (100 flows, 5s timeout)
└──────────┬───────────┘
           │
           ▼
┌─────────────────────┐
│   AgentClient       │
│   - Register        │
│   - Heartbeat (30s) │
│   - Policy Sync     │
└──────────┬───────────┘
           │
           ▼ gRPC (port 9090)
┌──────────────────────┐
│ Microsegment Server  │
│ - PostgreSQL         │
│ - HTTP API (8080)    │
│ - gRPC API (9090)    │
└──────────────────────┘
```

**特点**:
- 多节点分布式架构
- 中心化 PostgreSQL 存储
- 从服务器同步策略
- Agent 健康监控
- 可扩展到数千节点

### 2. 核心组件实现

#### Reporter 接口
```go
// src/agent/pkg/reporter/reporter.go
type Reporter interface {
    Report(ctx context.Context, flow *flow.Flow) error
    ReportBatch(ctx context.Context, flows []*flow.Flow) error
    Start() error
    Stop() error
}
```

**设计理念**:
- 可插拔架构
- 关注点分离
- 易于测试（可 mock）
- 未来可扩展（Kafka reporter 等）

#### LocalReporter 实现
```go
// src/agent/pkg/reporter/local_reporter.go
type LocalReporter struct {
    storage storage.Storage
}

func (r *LocalReporter) Report(ctx context.Context, f *flow.Flow) error {
    return r.storage.SaveFlow(f)
}
```

**特点**:
- 包装现有 SQLite 存储
- 零性能开销
- 保持向后兼容

#### GRPCReporter 实现
```go
// src/agent/pkg/reporter/grpc_reporter.go
type GRPCReporter struct {
    serverAddr string
    agentID    string
    conn       *grpc.ClientConn
    client     flowpb.FlowServiceClient
    batchSize  int
    batchQueue chan *flowpb.FlowEvent
    stopCh     chan struct{}
}
```

**关键特性**:
- **批处理**: 累积 100 个流或 5 秒超时
- **异步发送**: 独立 goroutine，非阻塞
- **队列管理**: 缓冲通道（2x 批大小）防止阻塞
- **协议转换**: 内部 Flow 结构 → protobuf FlowEvent
- **优雅关闭**: 退出时刷新待发送批次

**性能优化**:
```go
func (r *GRPCReporter) batchSender() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    batch := make([]*flowpb.FlowEvent, 0, r.batchSize)

    for {
        select {
        case event := <-r.batchQueue:
            batch = append(batch, event)
            if len(batch) >= r.batchSize {
                r.sendBatchAsync(batch)
                batch = make([]*flowpb.FlowEvent, 0, r.batchSize)
            }
        case <-ticker.C:
            if len(batch) > 0 {
                r.sendBatchAsync(batch)
                batch = make([]*flowpb.FlowEvent, 0, r.batchSize)
            }
        case <-r.stopCh:
            if len(batch) > 0 {
                r.sendBatchAsync(batch)
            }
            return
        }
    }
}
```

**协议转换示例**:
```go
func (r *GRPCReporter) flowToProto(f *flow.Flow) *flowpb.FlowEvent {
    return &flowpb.FlowEvent{
        SrcIp:        ipToUint32(f.SourceIP),
        DstIp:        ipToUint32(f.DestIP),
        SrcPort:      uint32(f.SourcePort),
        DstPort:      uint32(f.DestPort),
        Protocol:     protocolStringToEnum(f.Protocol),
        EventType:    eventTypeStringToEnum(f.EventType),
        Direction:    directionStringToEnum(f.Direction),
        PacketCount:  f.PacketCount,
        ByteCount:    f.ByteCount,
        TimestampNs:  uint64(f.StartTime.UnixNano()),
        PolicyId:     f.PolicyID,
        PolicyAction: policyActionStringToEnum(f.PolicyAction),
        State:        stateStringToEnum(f.State),
        AgentId:      r.agentID,
        SourceLabels: f.SourceLabels,
        DestLabels:   f.DestLabels,
    }
}
```

#### AgentClient 实现
```go
// src/agent/pkg/client/agent_client.go
type AgentClient struct {
    agentID       string
    hostname      string
    version       string
    serverAddr    string

    conn          *grpc.ClientConn
    agentService  agentpb.AgentServiceClient
    policyService policypb.PolicyServiceClient

    heartbeatInterval int
    statsInterval     int
    stopCh            chan struct{}

    flowCount   uint64
    policyCount uint32
}
```

**关键功能**:

1. **注册 (Registration)**:
```go
func (c *AgentClient) registerAgent() error {
    req := &agentpb.RegisterRequest{
        AgentId:     c.agentID,
        Hostname:    c.hostname,
        Version:     c.version,
        IpAddresses: getLocalIPs(),
        OsInfo:      getOSInfo(),
        Metadata:    map[string]string{},
    }

    resp, err := c.agentService.RegisterAgent(context.Background(), req)
    if err != nil {
        return fmt.Errorf("registration RPC failed: %w", err)
    }

    c.heartbeatInterval = int(resp.Config.HeartbeatInterval)
    c.statsInterval = int(resp.Config.StatsInterval)

    return nil
}
```

2. **心跳 (Heartbeat)**:
```go
func (c *AgentClient) StartHeartbeat() {
    ticker := time.NewTicker(time.Duration(c.heartbeatInterval) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := c.sendHeartbeat(); err != nil {
                logrus.Errorf("Heartbeat failed: %v", err)
            }
        case <-c.stopCh:
            return
        }
    }
}

func (c *AgentClient) sendHeartbeat() error {
    req := &agentpb.HeartbeatRequest{
        AgentId:   c.agentID,
        Timestamp: uint64(time.Now().Unix()),
        Metrics: &agentpb.AgentMetrics{
            CpuUsage:    getCPUUsage(),
            MemoryUsage: getMemoryUsage(),
            FlowCount:   c.flowCount,
            PolicyCount: c.policyCount,
        },
    }

    resp, err := c.agentService.Heartbeat(context.Background(), req)
    // ...
}
```

3. **策略同步 (Policy Sync)**:
```go
func (c *AgentClient) SyncPolicies(currentVersion uint64) ([]*policypb.Policy, uint64, error) {
    req := &policypb.SyncRequest{
        AgentId:              c.agentID,
        CurrentPolicyVersion: currentVersion,
        Capabilities:         []string{"label-based-policy", "ip-based-policy"},
    }

    resp, err := c.policyService.SyncPolicies(context.Background(), req)
    if err != nil {
        return nil, 0, fmt.Errorf("policy sync failed: %w", err)
    }

    return resp.Policies, resp.PolicyVersion, nil
}
```

### 3. 配置系统

#### 配置结构
```go
// src/agent/pkg/config/config.go
type Config struct {
    Mode          string        `mapstructure:"mode"`
    Interface     string        `mapstructure:"interface"`
    LogLevel      string        `mapstructure:"log_level"`
    StatsInterval int           `mapstructure:"stats_interval"`
    Storage       StorageConfig `mapstructure:"storage"`
    API           APIConfig     `mapstructure:"api"`
    AgentServer   *AgentServerConfig `mapstructure:"agent_server,omitempty"`
}

type AgentServerConfig struct {
    Enabled           bool          `mapstructure:"enabled"`
    ServerAddr        string        `mapstructure:"server_addr"`
    AgentID           string        `mapstructure:"agent_id"`
    BatchSize         int           `mapstructure:"batch_size"`
    BatchTimeout      time.Duration `mapstructure:"batch_timeout"`
    ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
}
```

#### 配置验证
```go
func (c *Config) Validate() error {
    // 验证模式
    if c.Mode != "standalone" && c.Mode != "agent-server" {
        return fmt.Errorf("invalid mode: %s", c.Mode)
    }

    // Agent-Server 模式验证
    if c.Mode == "agent-server" {
        if c.AgentServer == nil {
            return fmt.Errorf("agent_server config required")
        }
        if c.AgentServer.ServerAddr == "" {
            return fmt.Errorf("server_addr is required")
        }
        // 自动生成 agent ID
        if c.AgentServer.AgentID == "" {
            c.AgentServer.AgentID = generateAgentID()
        }
        // 设置默认值
        if c.AgentServer.BatchSize == 0 {
            c.AgentServer.BatchSize = 100
        }
    }

    return nil
}
```

#### 配置加载
```go
func LoadConfig(configPath string) (*Config, error) {
    v := viper.New()
    setDefaults(v)

    if configPath != "" {
        v.SetConfigFile(configPath)
        if err := v.ReadInConfig(); err != nil {
            return nil, err
        }
    }

    // 环境变量覆盖
    v.AutomaticEnv()
    v.SetEnvPrefix("MICROSEGMENT")

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, cfg.Validate()
}
```

---

## 下一部分

继续查看：
- [Part 2: 配置示例和主入口点实现](session-2025-11-05-part2-implementation.md)
- [Part 3: 文档和总结](session-2025-11-05-part3-documentation.md)

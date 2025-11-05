# 设计文档: Agent 远程上报重构

**变更 ID**: `refactor-agent-for-remote-reporting`
**设计日期**: 2025-11-05
**版本**: 1.0
**依赖**: add-grpc-protocol-definitions, add-server-component

---

## 📋 设计概述

重构 Agent 组件以支持向中心 Server 远程上报流事件和同步策略，同时保持对现有 standalone 模式的完全向后兼容。

### 设计目标

1. **双模式支持**: Agent 可运行在 standalone 或 agent-server 模式
2. **零侵入性**: 现有 standalone 部署不受影响
3. **可插拔架构**: Reporter 和 PolicySyncer 接口化
4. **高性能**: gRPC 批量上报，减少网络开销
5. **容错性**: 支持本地缓存和自动重连

---

## 🏗️ 架构设计

### 整体架构对比

#### Standalone 模式（现有）
```
┌─────────────────────────────────────┐
│   microsegment-agent (单体)         │
│                                     │
│  ┌──────────┐      ┌─────────────┐ │
│  │FlowCollec│─────>│SQLite Storage│ │
│  │  tor     │      └─────────────┘ │
│  └──────────┘                       │
│       ↑                             │
│  ┌──────────┐                       │
│  │  eBPF    │                       │
│  │ Dataplane│                       │
│  └──────────┘                       │
│                                     │
│  ┌──────────┐                       │
│  │ HTTP API │ :8080                 │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

#### Agent-Server 模式（新增）
```
┌─────────────────────────────────────┐
│   microsegment-agent                │
│                                     │
│  ┌──────────┐      ┌─────────────┐ │
│  │FlowCollec│─────>│GRPCReporter │─┼──> gRPC
│  │  tor     │      │  (batched)  │ │     ↓
│  └──────────┘      └─────────────┘ │  Server
│       ↑                             │
│  ┌──────────┐                       │
│  │  eBPF    │                       │
│  │ Dataplane│                       │
│  └──────────┘                       │
│                                     │
│  ┌──────────┐      ┌─────────────┐ │
│  │ Policy   │<─────│GRPCSyncer   │<┼─── gRPC
│  │ Manager  │      │             │ │     ↑
│  └──────────┘      └─────────────┘ │  Server
│                                     │
│  ┌──────────┐                       │
│  │AgentClien│─────────────────────┼──> gRPC
│  │  t       │  Register/Heartbeat │     ↓
│  └──────────┘                       │  Server
└─────────────────────────────────────┘
```

---

## 📐 核心组件设计

### 1. Reporter 接口

**目的**: 抽象流数据上报，支持本地或远程

```go
// pkg/reporter/reporter.go
package reporter

import (
    "context"
    "github.com/ebpf-microsegment/src/agent/pkg/flow"
)

// Reporter 接口定义数据上报行为
type Reporter interface {
    // Report 上报单个流事件
    Report(ctx context.Context, flow *flow.Flow) error

    // ReportBatch 批量上报多个流事件（效率更高）
    ReportBatch(ctx context.Context, flows []*flow.Flow) error

    // Start 初始化 Reporter
    Start() error

    // Stop 优雅关闭 Reporter
    Stop() error
}
```

### 2. LocalReporter 实现

**目的**: Standalone 模式，保持现有行为

```go
// pkg/reporter/local_reporter.go
type LocalReporter struct {
    storage storage.Storage  // 现有 SQLite 存储
}

func NewLocalReporter(storage storage.Storage) *LocalReporter {
    return &LocalReporter{storage: storage}
}

func (r *LocalReporter) Report(ctx context.Context, f *flow.Flow) error {
    // 直接写入 SQLite
    return r.storage.SaveFlow(ctx, f)
}

func (r *LocalReporter) Start() error {
    // 无需初始化
    return nil
}

func (r *LocalReporter) Stop() error {
    // 无需清理
    return nil
}
```

**特点**:
- ✅ 零性能开销
- ✅ 无网络依赖
- ✅ 完全向后兼容

### 3. GRPCReporter 实现

**目的**: Agent-Server 模式，高效远程上报

```go
// pkg/reporter/grpc_reporter.go
type GRPCReporter struct {
    serverAddr string
    agentID    string
    conn       *grpc.ClientConn
    client     flowpb.FlowServiceClient

    // Batching
    batchSize  int
    batchQueue chan *flowpb.FlowEvent
    stopCh     chan struct{}
}

func NewGRPCReporter(serverAddr, agentID string, batchSize int) *GRPCReporter {
    return &GRPCReporter{
        serverAddr: serverAddr,
        agentID:    agentID,
        batchSize:  batchSize,
        batchQueue: make(chan *flowpb.FlowEvent, batchSize*2),
        stopCh:     make(chan struct{}),
    }
}

func (r *GRPCReporter) Start() error {
    // 1. 连接 Server
    conn, err := grpc.Dial(r.serverAddr, grpc.WithInsecure())
    if err != nil {
        return err
    }
    r.conn = conn
    r.client = flowpb.NewFlowServiceClient(conn)

    // 2. 启动批量发送 goroutine
    go r.batchSender()

    return nil
}

func (r *GRPCReporter) Report(ctx context.Context, f *flow.Flow) error {
    // 转换为 protobuf 并加入队列
    event := r.flowToProto(f)
    select {
    case r.batchQueue <- event:
        return nil
    default:
        return fmt.Errorf("queue full")
    }
}

func (r *GRPCReporter) batchSender() {
    ticker := time.NewTicker(5 * time.Second)
    batch := make([]*flowpb.FlowEvent, 0, r.batchSize)

    for {
        select {
        case event := <-r.batchQueue:
            batch = append(batch, event)
            if len(batch) >= r.batchSize {
                r.sendBatch(batch)
                batch = batch[:0]
            }

        case <-ticker.C:
            if len(batch) > 0 {
                r.sendBatch(batch)
                batch = batch[:0]
            }

        case <-r.stopCh:
            return
        }
    }
}

func (r *GRPCReporter) sendBatch(events []*flowpb.FlowEvent) error {
    stream, err := r.client.ReportFlowEvents(context.Background())
    if err != nil {
        return err
    }

    for _, event := range events {
        if err := stream.Send(event); err != nil {
            return err
        }
    }

    resp, err := stream.CloseAndRecv()
    if err != nil {
        return err
    }

    log.Debugf("Sent %d events, server accepted %d",
        len(events), resp.AcceptedCount)
    return nil
}
```

**特点**:
- ✅ 批量发送（默认 100 events 或 5 秒触发）
- ✅ 异步队列，不阻塞数据平面
- ✅ 自动重连（gRPC 内置）
- ✅ 支持背压（队列满时丢弃）

### 4. AgentClient

**目的**: 封装 Agent 与 Server 的生命周期交互

```go
// pkg/agent/client.go
package agent

import (
    "context"
    "time"

    agentpb "github.com/ebpf-microsegment/src/proto/agent"
    "google.golang.org/grpc"
)

type Client struct {
    serverAddr string
    agentID    string
    conn       *grpc.ClientConn
    client     agentpb.AgentServiceClient
    stopCh     chan struct{}
}

func NewClient(serverAddr, agentID string) *Client {
    return &Client{
        serverAddr: serverAddr,
        agentID:    agentID,
        stopCh:     make(chan struct{}),
    }
}

func (c *Client) Connect() error {
    conn, err := grpc.Dial(c.serverAddr, grpc.WithInsecure())
    if err != nil {
        return err
    }
    c.conn = conn
    c.client = agentpb.NewAgentServiceClient(conn)
    return nil
}

func (c *Client) Register(info *AgentInfo) error {
    req := &agentpb.RegisterRequest{
        AgentId:       c.agentID,
        Hostname:      info.Hostname,
        Version:       info.Version,
        Interface:     info.Interface,
        IpAddresses:   info.IPAddresses,
        Os:            info.OS,
        KernelVersion: info.KernelVersion,
        StartTime:     time.Now().UnixNano(),
    }

    resp, err := c.client.RegisterAgent(context.Background(), req)
    if err != nil {
        return err
    }

    if !resp.Success {
        return fmt.Errorf("registration failed: %s", resp.Message)
    }

    log.Infof("Registered with server, version=%s", resp.ServerVersion)
    return nil
}

func (c *Client) StartHeartbeat(interval time.Duration) {
    go c.heartbeatLoop(interval)
}

func (c *Client) heartbeatLoop(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            req := &agentpb.HeartbeatRequest{
                AgentId:   c.agentID,
                Timestamp: time.Now().UnixNano(),
                Metrics:   c.collectMetrics(),
            }

            resp, err := c.client.Heartbeat(context.Background(), req)
            if err != nil {
                log.Errorf("Heartbeat failed: %v", err)
                continue
            }

            if !resp.Healthy {
                log.Warnf("Server reports unhealthy: %s", resp.Message)
            }

        case <-c.stopCh:
            return
        }
    }
}

func (c *Client) collectMetrics() *agentpb.AgentMetrics {
    // TODO: 收集实际指标
    return &agentpb.AgentMetrics{
        CpuUsage:     0.0,
        MemoryUsage:  0,
        PacketsProcessed: 0,
    }
}

func (c *Client) Close() error {
    close(c.stopCh)
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}
```

### 5. 配置扩展

```go
// pkg/config/config.go
type Config struct {
    // 运行模式
    Mode string `yaml:"mode"` // "standalone" 或 "agent-server"

    // Standalone 配置（现有）
    Standalone StandaloneConfig `yaml:"standalone"`

    // Agent-Server 配置（新增）
    AgentServer AgentServerConfig `yaml:"agent_server"`
}

type AgentServerConfig struct {
    ServerAddr string `yaml:"server_addr"` // "server.example.com:9090"
    AgentID    string `yaml:"agent_id"`    // "node-01"

    // 批量发送配置
    BatchSize    int           `yaml:"batch_size"`     // 默认 100
    BatchTimeout time.Duration `yaml:"batch_timeout"`  // 默认 5s

    // 心跳配置
    HeartbeatInterval time.Duration `yaml:"heartbeat_interval"` // 默认 30s

    // TLS 配置（可选）
    TLS TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
    Enabled  bool   `yaml:"enabled"`
    CertFile string `yaml:"cert_file"`
    KeyFile  string `yaml:"key_file"`
    CAFile   string `yaml:"ca_file"`
}
```

**配置文件示例**:

**Standalone 模式** (`config-standalone.yaml`):
```yaml
mode: standalone

standalone:
  storage:
    path: /var/lib/ebpf-agent/agent.db
  api:
    enabled: true
    port: 8080
```

**Agent-Server 模式** (`config-agent-server.yaml`):
```yaml
mode: agent-server

agent_server:
  server_addr: "server.example.com:9090"
  agent_id: "node-01"
  batch_size: 100
  batch_timeout: 5s
  heartbeat_interval: 30s
  tls:
    enabled: false
```

### 6. FlowCollector 改造

**改动**: 使用 Reporter 接口替代直接写 SQLite

```go
// pkg/flow/collector.go (改造前)
type Collector struct {
    storage     storage.Storage  // 直接依赖 SQLite
    ringBuf     *ringbuf.Reader
    workloadMgr *workload.Manager
}

func (c *Collector) processEvent(event *Event) {
    flow := c.buildFlow(event)

    // 直接保存到 SQLite
    c.storage.SaveFlow(context.Background(), flow)
}
```

```go
// pkg/flow/collector.go (改造后)
type Collector struct {
    reporter    reporter.Reporter  // 使用接口
    ringBuf     *ringbuf.Reader
    workloadMgr *workload.Manager
}

func (c *Collector) processEvent(event *Event) {
    flow := c.buildFlow(event)

    // 通过 Reporter 接口上报（可能是本地或远程）
    if err := c.reporter.Report(context.Background(), flow); err != nil {
        log.Errorf("Failed to report flow: %v", err)
    }
}
```

**优点**:
- ✅ 单一职责：Collector 只负责处理事件，不关心存储方式
- ✅ 可测试性：可以 mock Reporter 进行单元测试
- ✅ 灵活性：未来可以添加其他 Reporter（Kafka, Redis 等）

### 7. main.go 改造

```go
// cmd/main.go
func runAgent(cmd *cobra.Command, args []string) {
    // 1. 加载配置
    cfg, err := config.Load(configFile)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. 创建数据平面
    dp, err := dataplane.New(iface)
    if err != nil {
        log.Fatalf("Failed to create dataplane: %v", err)
    }
    defer dp.Close()

    // 3. 创建 Workload Manager
    workloadMgr := workload.NewManager()

    // 4. 创建 Reporter（根据模式）
    var rep reporter.Reporter
    if cfg.Mode == "agent-server" {
        // Agent-Server 模式
        log.Info("Starting in agent-server mode")

        rep = reporter.NewGRPCReporter(
            cfg.AgentServer.ServerAddr,
            cfg.AgentServer.AgentID,
            cfg.AgentServer.BatchSize,
        )

        // 启动 Reporter
        if err := rep.Start(); err != nil {
            log.Fatalf("Failed to start gRPC reporter: %v", err)
        }
        defer rep.Stop()

        // 注册 Agent
        agentClient := agent.NewClient(
            cfg.AgentServer.ServerAddr,
            cfg.AgentServer.AgentID,
        )
        if err := agentClient.Connect(); err != nil {
            log.Fatalf("Failed to connect to server: %v", err)
        }
        defer agentClient.Close()

        if err := agentClient.Register(getAgentInfo()); err != nil {
            log.Fatalf("Failed to register: %v", err)
        }

        // 启动心跳
        agentClient.StartHeartbeat(cfg.AgentServer.HeartbeatInterval)

    } else {
        // Standalone 模式（默认）
        log.Info("Starting in standalone mode")

        storage, err := storage.NewSQLite(cfg.Standalone.Storage.Path)
        if err != nil {
            log.Fatalf("Failed to create storage: %v", err)
        }
        defer storage.Close()

        rep = reporter.NewLocalReporter(storage)
        rep.Start()
        defer rep.Stop()

        // 启动 HTTP API
        apiServer := api.NewServer(storage, cfg.Standalone.API.Port)
        go apiServer.Start()
        defer apiServer.Stop()
    }

    // 5. 创建 FlowCollector (两种模式共用)
    collector := flow.NewCollector(dp, rep, workloadMgr)
    if err := collector.Start(); err != nil {
        log.Fatalf("Failed to start collector: %v", err)
    }

    // 6. 等待信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    log.Info("Shutting down...")
}

func getAgentInfo() *agent.AgentInfo {
    hostname, _ := os.Hostname()
    return &agent.AgentInfo{
        Hostname:      hostname,
        Version:       "1.0.0",
        Interface:     iface,
        IPAddresses:   getLocalIPs(),
        OS:            runtime.GOOS,
        KernelVersion: getKernelVersion(),
    }
}
```

---

## 🔄 数据流设计

### Standalone 模式数据流

```
eBPF Ring Buffer
    ↓
FlowCollector.processEvent()
    ↓
LocalReporter.Report()
    ↓
SQLite.SaveFlow()
    ↓
Done
```

### Agent-Server 模式数据流

```
eBPF Ring Buffer
    ↓
FlowCollector.processEvent()
    ↓
GRPCReporter.Report()
    ↓
batchQueue (channel)
    ↓
batchSender() goroutine
    ↓  (every 100 events or 5s)
gRPC Stream: ReportFlowEvents()
    ↓
Server.FlowService
    ↓
PostgreSQL
    ↓
Done
```

---

## 🧪 测试策略

### 单元测试

```go
// pkg/reporter/grpc_reporter_test.go
func TestGRPCReporter_Batching(t *testing.T) {
    // Mock gRPC server
    server := startMockServer(t)
    defer server.Stop()

    reporter := NewGRPCReporter(server.Addr(), "test-agent", 10)
    reporter.Start()
    defer reporter.Stop()

    // Send 15 flows
    for i := 0; i < 15; i++ {
        flow := &flow.Flow{ /* ... */ }
        reporter.Report(context.Background(), flow)
    }

    // Wait for batches
    time.Sleep(6 * time.Second)

    // Verify: 2 batches (10 + 5)
    assert.Equal(t, 2, server.ReceivedBatches())
}
```

### 集成测试

```bash
#!/bin/bash
# tests/integration/test_agent_server.sh

# 1. Start server
./bin/microsegment-server --config test-server.yaml &
SERVER_PID=$!

# 2. Wait for server ready
sleep 2

# 3. Start agent in agent-server mode
./bin/microsegment-agent --config test-agent-server.yaml &
AGENT_PID=$!

# 4. Wait for flows
sleep 10

# 5. Query server API
curl http://localhost:8080/api/v1/flows | jq '.flows | length'

# 6. Cleanup
kill $AGENT_PID $SERVER_PID
```

---

## 📊 性能考量

### 内存使用

| 模式 | Reporter 开销 | 说明 |
|-----|-------------|------|
| Standalone | ~0 MB | 无额外开销 |
| Agent-Server | ~2-5 MB | 批量队列 + gRPC 连接 |

### CPU 使用

| 操作 | 开销 | 优化 |
|-----|------|-----|
| Flow → Protobuf 转换 | ~100 ns/event | 使用 fixed32/fixed64 |
| 批量序列化 | ~10 µs/batch | gRPC 内置优化 |
| 网络发送 | ~1 ms/batch | 批量减少 RPC 次数 |

### 吞吐量

| 指标 | 预期值 |
|-----|-------|
| 单 Agent 上报速率 | ~10K events/s |
| 批量大小 | 100 events |
| 批量间隔 | 5 秒 |
| 峰值内存 | ~5 MB |

---

## 🚨 错误处理

### gRPC 连接失败

```go
func (r *GRPCReporter) sendBatch(events []*flowpb.FlowEvent) error {
    err := r.doSend(events)
    if err != nil {
        log.Errorf("Failed to send batch: %v", err)

        // 可选: 写入本地缓存
        if r.fallbackStorage != nil {
            r.fallbackStorage.SaveBatch(events)
        }

        return err
    }
    return nil
}
```

### Server 不可用

**策略**:
1. gRPC 自动重连（exponential backoff）
2. 队列背压：丢弃最旧的事件
3. 本地缓存（可选）：写入 SQLite 后续上传

---

## 🔐 安全考量

### TLS 支持（可选）

```go
func (r *GRPCReporter) Start() error {
    var opts []grpc.DialOption

    if r.tlsEnabled {
        creds, err := credentials.NewClientTLSFromFile(
            r.tlsConfig.CertFile,
            r.tlsConfig.ServerName,
        )
        if err != nil {
            return err
        }
        opts = append(opts, grpc.WithTransportCredentials(creds))
    } else {
        opts = append(opts, grpc.WithInsecure())
    }

    conn, err := grpc.Dial(r.serverAddr, opts...)
    // ...
}
```

### 认证（可选）

```go
// Token-based auth
type tokenAuth struct {
    token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    return map[string]string{
        "authorization": "Bearer " + t.token,
    }, nil
}
```

---

## 📋 验收标准

### 功能验收
- [ ] Agent 可在 standalone 模式运行（完全向后兼容）
- [ ] Agent 可在 agent-server 模式连接 Server
- [ ] 流事件正确上报到 Server 并存入 PostgreSQL
- [ ] Agent 心跳正常，Server 可查看 Agent 列表
- [ ] 配置文件支持两种模式切换

### 性能验收
- [ ] Standalone 模式性能无退化
- [ ] Agent-Server 模式吞吐量 ≥ 5K events/s
- [ ] 内存增加 < 10 MB
- [ ] gRPC 批量发送延迟 < 10ms

### 质量验收
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 代码无 lint 错误
- [ ] 文档完整（README, 配置示例）

---

## 🔄 回滚计划

如果 agent-server 模式出现问题：

1. **立即回退**: 使用 standalone 配置重启 Agent
2. **数据不丢失**: Standalone 模式继续写本地 SQLite
3. **零停机**: 修改配置文件，无需重新编译

---

**设计审批**: 待审批
**下一步**: 创建 tasks.md 实施清单

# 提案: 重构 Agent 支持远程上报

**变更 ID**: `refactor-agent-for-remote-reporting`
**提案日期**: 2025-11-04
**状态**: 提案中
**优先级**: P0
**预估工作量**: 5-7 天

---

## 📋 概述

重构 Agent 组件,支持向中心 Server 上报流事件和同步策略,同时保持向后兼容性（支持 standalone 单体模式）。

## 🎯 目标

1. **支持两种运行模式** - standalone (单体) 和 agent-server (分布式)
2. **实现 gRPC Client** - 连接 Server,上报流事件、同步策略、发送心跳
3. **重构 FlowCollector** - 支持可插拔 Reporter 接口
4. **重构 PolicyManager** - 支持从 Server 拉取/订阅策略
5. **添加心跳机制** - 定期向 Server 报告健康状态
6. **保持向后兼容** - 不影响现有 standalone 部署

## 🏗️ 核心设计

### 运行模式选择

```yaml
# config.yaml
mode: agent-server  # 或 standalone

# standalone 模式 (默认,保持现有行为)
standalone:
  storage: sqlite
  api:
    enabled: true
    port: 8080

# agent-server 模式 (新增)
agent-server:
  server_url: grpc://server.example.com:9090
  agent_id: node-01
  tls:
    enabled: true
    cert_file: /etc/agent/cert.pem
  local_cache:
    enabled: true  # 断网时本地缓存
    max_size: 1000
```

### Reporter 接口重构

```go
// Reporter 接口 (抽象数据上报)
type Reporter interface {
    Report(flow *Flow) error
    Start() error
    Stop() error
}

// LocalReporter - SQLite 实现 (standalone 模式)
type LocalReporter struct {
    storage Storage
}

// GRPCReporter - gRPC 实现 (agent-server 模式)
type GRPCReporter struct {
    client   pb.FlowServiceClient
    queue    chan *Flow
    batchSize int
    batchTimeout time.Duration
}
```

### FlowCollector 改造

```go
type Collector struct {
    ringBuf     *ringbuf.Reader
    reporter    Reporter      // 可插拔
    workloadMgr WorkloadManager
    activeFlows map[string]*Flow
    flowsMutex  sync.RWMutex
}

func (c *Collector) processFlowEvent(event *FlowEvent) {
    flow := c.enrichWithLabels(event)

    // 上报到 Reporter (可能是 SQLite 或 gRPC)
    if err := c.reporter.Report(flow); err != nil {
        log.Errorf("Failed to report flow: %v", err)
    }
}
```

### PolicyManager 改造

```go
type PolicyManager struct {
    dataPlane  *dataplane.DataPlane
    storage    Storage  // 本地缓存
    syncer     PolicySyncer  // 可选,agent-server 模式使用
}

// PolicySyncer 接口
type PolicySyncer interface {
    Sync() ([]*Policy, error)
    Subscribe(callback func(*PolicyUpdate)) error
}

// GRPCPolicySyncer 实现
type GRPCPolicySyncer struct {
    client pb.PolicyServiceClient
}
```

### main.go 改造

```go
func runAgent(cmd *cobra.Command, args []string) {
    // 加载配置
    cfg := loadConfig(configFile)

    // 创建数据平面
    dp, err := dataplane.New(iface)

    // 创建 Reporter (根据模式)
    var reporter flow.Reporter
    if cfg.Mode == "agent-server" {
        reporter, err = createGRPCReporter(cfg.ServerURL, cfg.AgentID)
    } else {
        reporter, err = createLocalReporter(cfg.Storage.Path)
    }

    // 创建 Collector
    collector := flow.NewCollector(dp.RingBuf(), reporter, workloadMgr)

    // Agent-Server 模式: 注册和心跳
    if cfg.Mode == "agent-server" {
        agentClient := agent.NewClient(cfg.ServerURL)
        agentClient.Register(agentInfo)
        go agentClient.HeartbeatLoop(10 * time.Second)
    }

    // 启动 Collector
    collector.Start()
}
```

## ✅ 验收标准

- [ ] 支持 `--mode standalone` 和 `--mode agent-server` 参数
- [ ] Agent-Server 模式能成功连接 Server
- [ ] 流事件通过 gRPC 正常上报
- [ ] 策略从 Server 同步成功
- [ ] 心跳机制正常工作
- [ ] Standalone 模式保持原有功能不变
- [ ] 断网时本地缓存正常工作

## 🔗 依赖

**前置依赖**:
- add-grpc-protocol-definitions
- add-server-component

---

**提案人**: Claude Code

# eBPF 微隔离项目 - 代码架构与学习指南

> **目标读者**: 需要快速理解代码逻辑和执行流程的开发者
> **阅读时间**: 主线 1 小时，深入学习 3-4 小时
> **最后更新**: 2025-11-22

---

## 目录

1. [项目架构概览](#1-项目架构概览)
2. [快速入门 - 5 分钟理解核心](#2-快速入门---5-分钟理解核心)
3. [Agent 模块详解](#3-agent-模块详解)
4. [Server 模块详解](#4-server-模块详解)
5. [Web 前端模块详解](#5-web-前端模块详解)
6. [eBPF 数据平面](#6-ebpf-数据平面)
7. [关键数据结构](#7-关键数据结构)
8. [学习路径建议](#8-学习路径建议)

---

## 1. 项目架构概览

### 1.1 整体架构（三层设计）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Web UI (React + TypeScript)                      │
│   Dashboard | Topology | Flows | Policies | Agents                       │
│   文件: web/src/                                                         │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │ REST API / WebSocket
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Server (Go - Central Management)                      │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  REST API (Gin)          │  gRPC Services        │  WebSocket Hub  │ │
│  │  /api/v1/flows           │  FlowService          │  Real-time      │ │
│  │  /api/v1/policies        │  PolicyService        │  Updates        │ │
│  │  /api/v1/agents          │  AgentService         │                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Storage Layer           │  PubSub              │  Aggregator      │ │
│  │  PostgreSQL (GORM)       │  Policy Updates      │  Flow Stats      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│   文件: src/server/                                                      │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │ gRPC
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Agent (Go - Per Node)                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Local API (Gin)         │  gRPC Reporter       │  Policy Manager  │ │
│  │  /api/v1/policies        │  Flow Events         │  Indexed Match   │ │
│  │  /api/v1/workloads       │  Statistics          │  Label-based     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  DataPlane Manager       │  Session Manager     │  Labels/Groups   │ │
│  │  TC/XDP eBPF Loader      │  Timeout Handling    │  K8s Syncer      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│   文件: src/agent/                                                       │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │ eBPF Maps
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    eBPF Programs (Kernel Space)                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  TC Program: tc_microsegment.bpf.c                               │   │
│  │  XDP Program: xdp_microsegment.bpf.c                             │   │
│  │  Process Monitor: process_monitor.bpf.c                          │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  eBPF Maps:                                                       │   │
│  │  - session_map (LRU_HASH)      - 会话跟踪                        │   │
│  │  - policy_map (HASH)           - 精确匹配策略                    │   │
│  │  - indexed_policy_* (HASH)     - 索引策略匹配                    │   │
│  │  - flow_events (RINGBUF)       - 流事件上报                      │   │
│  │  - stats_map (PERCPU_ARRAY)    - 统计计数器                      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│   文件: src/bpf/                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 目录结构详解

```
ebpf-based-microsegment/
├── api/proto/                      # Protocol Buffers 定义
│   ├── common/common.proto         # 公共类型定义
│   ├── flow/flow.proto             # 流事件定义
│   └── policy/policy.proto         # 策略定义
│
├── src/
│   ├── agent/                      # Agent 用户态程序
│   │   ├── cmd/main.go             # Agent 入口
│   │   ├── pkg/
│   │   │   ├── api/                # 本地 REST API
│   │   │   │   ├── server.go       # HTTP 服务器
│   │   │   │   └── handlers/       # API 处理器
│   │   │   │       ├── policy.go   # 策略 CRUD
│   │   │   │       ├── workload.go # 工作负载管理
│   │   │   │       └── health.go   # 健康检查
│   │   │   │
│   │   │   ├── dataplane/          # eBPF 数据平面管理
│   │   │   │   ├── dataplane.go    # 主逻辑
│   │   │   │   ├── manager.go      # 统一管理器
│   │   │   │   ├── tc_loader.go    # TC 程序加载
│   │   │   │   ├── xdp_loader.go   # XDP 程序加载
│   │   │   │   ├── nat.go          # NAT 支持
│   │   │   │   └── fragment.go     # 分片处理
│   │   │   │
│   │   │   ├── policy/             # 策略管理
│   │   │   │   ├── policy.go       # 策略 CRUD
│   │   │   │   ├── indexed_policy_manager.go  # 索引策略
│   │   │   │   ├── compiler.go     # 策略编译
│   │   │   │   └── storage.go      # SQLite 持久化
│   │   │   │
│   │   │   ├── reporter/           # 流事件上报
│   │   │   │   ├── reporter.go     # 接口定义
│   │   │   │   └── grpc_reporter.go # gRPC 上报实现
│   │   │   │
│   │   │   ├── flow/               # 流事件处理
│   │   │   │   └── types.go        # 流数据类型
│   │   │   │
│   │   │   ├── workload/           # 工作负载管理
│   │   │   │   ├── types.go        # 工作负载类型
│   │   │   │   ├── manager.go      # 管理器
│   │   │   │   └── storage.go      # 存储
│   │   │   │
│   │   │   ├── labels/             # 标签系统
│   │   │   │   ├── types.go        # 标签类型
│   │   │   │   ├── merger.go       # 标签合并
│   │   │   │   └── autotagger.go   # 自动标签
│   │   │   │
│   │   │   ├── groups/             # 安全组
│   │   │   │   ├── types.go        # 组类型
│   │   │   │   ├── manager.go      # 组管理
│   │   │   │   └── selector.go     # 标签选择器
│   │   │   │
│   │   │   ├── session/            # 会话管理
│   │   │   │   ├── types.go        # 会话类型
│   │   │   │   └── timeout_manager.go # 超时管理
│   │   │   │
│   │   │   ├── k8s/                # Kubernetes 集成
│   │   │   │   ├── syncer.go       # 资源同步
│   │   │   │   └── informer.go     # 事件监听
│   │   │   │
│   │   │   └── config/             # 配置管理
│   │   │       └── config.go       # 配置结构
│   │   │
│   │   └── test/                   # 测试
│   │       ├── e2e/                # 端到端测试
│   │       └── benchmark/          # 性能基准
│   │
│   ├── server/                     # Server 中央管理
│   │   ├── cmd/main.go             # Server 入口
│   │   ├── pkg/
│   │   │   ├── grpc/               # gRPC 服务
│   │   │   │   ├── flow_service.go # 流服务
│   │   │   │   ├── policy_service.go # 策略服务
│   │   │   │   └── agent_service.go # Agent 服务
│   │   │   │
│   │   │   ├── api/                # REST API
│   │   │   │   ├── handlers/       # API 处理器
│   │   │   │   │   ├── flow.go     # 流查询
│   │   │   │   │   ├── policy.go   # 策略管理
│   │   │   │   │   ├── agent.go    # Agent 管理
│   │   │   │   │   └── alert.go    # 告警管理
│   │   │   │   └── middleware/     # 中间件
│   │   │   │
│   │   │   ├── storage/            # 数据存储
│   │   │   │   ├── postgres.go     # PostgreSQL 连接
│   │   │   │   ├── gorm_db.go      # GORM 配置
│   │   │   │   ├── models.go       # 数据模型
│   │   │   │   ├── flow_storage.go # 流存储
│   │   │   │   ├── policy_storage.go # 策略存储
│   │   │   │   └── agent_storage.go # Agent 存储
│   │   │   │
│   │   │   ├── pubsub/             # 发布订阅
│   │   │   │   └── policy_pubsub.go # 策略更新推送
│   │   │   │
│   │   │   ├── aggregator/         # 数据聚合
│   │   │   │   ├── flow_aggregator.go # 流聚合
│   │   │   │   └── types.go        # 聚合类型
│   │   │   │
│   │   │   └── websocket/          # WebSocket
│   │   │       └── hub.go          # 实时推送
│   │   │
│   │   └── migrations/             # 数据库迁移
│   │
│   └── bpf/                        # eBPF 程序 (C)
│       ├── tc_microsegment.bpf.c   # TC eBPF 主程序
│       ├── xdp_microsegment.bpf.c  # XDP eBPF 程序
│       ├── process_monitor.bpf.c   # 进程监控
│       └── headers/
│           ├── common_types.h      # 共享数据结构
│           ├── policy_match.h      # 策略匹配
│           ├── indexed_policy_match.h # 索引策略
│           ├── tcp_state_machine.h # TCP 状态机
│           ├── nat_support.h       # NAT 支持
│           └── fragment_handler.h  # 分片处理
│
├── web/                            # Web 前端 (React + TypeScript)
│   └── src/
│       ├── api/                    # API 客户端
│       │   ├── client.ts           # HTTP 客户端
│       │   ├── flows.ts            # 流 API
│       │   ├── policies.ts         # 策略 API
│       │   └── agents.ts           # Agent API
│       │
│       ├── pages/                  # 页面组件
│       │   ├── Dashboard/          # 仪表板
│       │   ├── Topology/           # 拓扑图
│       │   ├── Flows/              # 流查询
│       │   ├── Policies/           # 策略管理
│       │   └── Agents/             # Agent 管理
│       │
│       ├── components/             # UI 组件
│       │   ├── topology/           # 拓扑组件
│       │   ├── flows/              # 流组件
│       │   ├── policies/           # 策略组件
│       │   └── visualization/      # 可视化
│       │
│       ├── hooks/                  # React Hooks
│       │   ├── useFlows.ts         # 流数据
│       │   ├── useTopology.ts      # 拓扑数据
│       │   └── usePolicies.ts      # 策略数据
│       │
│       ├── lib/graph/              # 图数据库
│       │   ├── Graph.ts            # 图结构
│       │   └── algorithms.ts       # 图算法
│       │
│       └── utils/                  # 工具函数
│           └── topologyUtils.ts    # 拓扑工具
│
├── pkg/                            # 共享包
│   ├── netutil/                    # 网络工具
│   │   ├── ip.go                   # IP 处理
│   │   └── validation.go           # 参数验证
│   └── constants/                  # 常量定义
│       └── policy.go               # 策略常量
│
├── config/                         # 配置文件
├── deploy/                         # 部署配置
│   ├── docker/                     # Docker 配置
│   ├── kubernetes/                 # K8s 配置
│   └── systemd/                    # Systemd 服务
│
└── openspec/                       # OpenSpec 变更管理
    ├── specs/                      # 规格定义
    └── changes/                    # 变更提案
```

---

## 2. 快速入门 - 5 分钟理解核心

### 2.1 这是什么？

**eBPF 微隔离系统**：基于 eBPF 的高性能网络访问控制系统

- **Agent**: 部署在每个节点，运行 eBPF 程序进行网络过滤
- **Server**: 中央管理服务器，存储流数据和策略
- **Web UI**: 可视化管理界面

### 2.2 核心数据流

```
网络包 → TC/XDP eBPF → 策略匹配 → 允许/拒绝
                ↓
           Ring Buffer
                ↓
         Agent (Go)
                ↓
         gRPC Reporter
                ↓
         Server (Go)
                ↓
         PostgreSQL + WebSocket
                ↓
         Web UI (React)
```

### 2.3 快速定位代码

| 功能 | 文件位置 |
|------|----------|
| Agent 入口 | `src/agent/cmd/main.go` |
| Server 入口 | `src/server/cmd/main.go` |
| eBPF TC 程序 | `src/bpf/tc_microsegment.bpf.c` |
| 策略管理 | `src/agent/pkg/policy/policy.go` |
| 流上报 | `src/agent/pkg/reporter/grpc_reporter.go` |
| 流存储 | `src/server/pkg/storage/flow_storage.go` |
| 拓扑可视化 | `web/src/utils/topologyUtils.ts` |

---

## 3. Agent 模块详解

### 3.1 启动流程

```go
// src/agent/cmd/main.go
func main() {
    // 1. 加载配置
    cfg := config.Load()

    // 2. 初始化 DataPlane (加载 eBPF 程序)
    dp := dataplane.NewManager(cfg)
    dp.Start()

    // 3. 初始化策略管理器
    pm := policy.NewIndexedPolicyManager(dp)

    // 4. 初始化 gRPC Reporter
    reporter := reporter.NewGRPCReporter(cfg.ServerAddr)

    // 5. 启动流事件监控
    go dp.MonitorFlowEvents(reporter)

    // 6. 启动 API Server
    api.NewServer(cfg, dp, pm).Start()
}
```

### 3.2 DataPlane 模块

```
src/agent/pkg/dataplane/
├── manager.go          # 统一管理 TC/XDP
├── tc_loader.go        # TC eBPF 加载
├── xdp_loader.go       # XDP eBPF 加载
├── dataplane.go        # 核心逻辑
├── nat.go              # NAT 处理
└── fragment.go         # 分片处理
```

**关键函数**:
- `NewManager()`: 创建数据平面管理器
- `Start()`: 加载 eBPF 程序并附加到网卡
- `MonitorFlowEvents()`: 监控 Ring Buffer 中的流事件
- `UpdatePolicy()`: 更新 eBPF map 中的策略

### 3.3 Policy 模块

```
src/agent/pkg/policy/
├── policy.go                   # 基础策略 CRUD
├── indexed_policy_manager.go   # 索引策略管理 (高性能)
├── compiler.go                 # 策略编译器
├── storage.go                  # SQLite 持久化
└── errors.go                   # 错误定义
```

**策略匹配优先级**:
1. 精确匹配 (5-tuple exact match)
2. 索引策略匹配 (label-based)
3. 通配符策略匹配

---

## 4. Server 模块详解

### 4.1 启动流程

```go
// src/server/cmd/main.go
func main() {
    // 1. 加载配置
    cfg := config.Load()

    // 2. 初始化数据库
    db := storage.NewPostgresDB(cfg.DatabaseURL)
    storage.InitSchema(db)

    // 3. 初始化存储层
    flowStorage := storage.NewFlowStorage(db)
    policyStorage := storage.NewPolicyStorage(db)
    agentStorage := storage.NewAgentStorage(db)

    // 4. 初始化 PubSub
    pubsub := pubsub.NewPolicyPubSub()

    // 5. 启动 gRPC Server
    grpcServer := grpc.NewServer(flowStorage, policyStorage, pubsub)
    go grpcServer.Start()

    // 6. 启动 REST API Server
    apiServer := api.NewServer(flowStorage, policyStorage, agentStorage)
    apiServer.Start()
}
```

### 4.2 gRPC 服务

```
src/server/pkg/grpc/
├── flow_service.go      # FlowService - 接收流事件
├── policy_service.go    # PolicyService - 策略同步
└── agent_service.go     # AgentService - Agent 注册/心跳
```

**FlowService**:
- `ReportFlowEvents()`: 接收 Agent 上报的流事件
- 批量写入 PostgreSQL

**PolicyService**:
- `SyncPolicies()`: Agent 启动时同步策略
- `SubscribePolicies()`: 订阅策略更新 (streaming)

### 4.3 存储层

```
src/server/pkg/storage/
├── postgres.go          # PostgreSQL 连接
├── gorm_db.go           # GORM 配置
├── models.go            # 数据模型
├── flow_storage.go      # 流存储 (GORM)
├── policy_storage.go    # 策略存储
└── agent_storage.go     # Agent 存储
```

---

## 5. Web 前端模块详解

### 5.1 技术栈

- **框架**: React 18 + TypeScript
- **构建**: Vite
- **UI**: Tailwind CSS
- **图表**: D3.js / Recharts
- **状态管理**: React Query
- **测试**: Vitest

### 5.2 页面结构

```
web/src/pages/
├── Dashboard/           # 仪表板 - 概览统计
├── Topology/            # 拓扑图 - 网络可视化
├── Flows/               # 流查询 - 流量分析
├── Policies/            # 策略管理 - CRUD
└── Agents/              # Agent 管理 - 状态监控
```

### 5.3 拓扑可视化

```typescript
// web/src/utils/topologyUtils.ts
// 核心函数:

// 将流数据聚合为拓扑数据
aggregateFlowsToTopology(flows: Flow[], viewMode: ViewMode): TopologyData

// 增量更新拓扑
mergeTopologyUpdate(existing: TopologyData, newFlow: Flow, viewMode: ViewMode): TopologyData

// 计算节点大小
calculateNodeSize(metrics: TrafficMetrics): number

// 计算边宽度
calculateEdgeWidth(metrics: TrafficMetrics): number
```

---

## 6. eBPF 数据平面

### 6.1 程序文件

```
src/bpf/
├── tc_microsegment.bpf.c       # TC eBPF 主程序
├── xdp_microsegment.bpf.c      # XDP eBPF 程序
├── process_monitor.bpf.c       # 进程监控
└── headers/
    ├── common_types.h          # 共享数据结构
    ├── policy_match.h          # 策略匹配逻辑
    ├── indexed_policy_match.h  # 索引策略匹配
    ├── tcp_state_machine.h     # TCP 状态跟踪
    ├── nat_support.h           # NAT 处理
    └── fragment_handler.h      # IP 分片处理
```

### 6.2 eBPF Maps

| Map 名称 | 类型 | 用途 |
|----------|------|------|
| `session_map` | LRU_HASH | 会话缓存，热路径优化 |
| `policy_map` | HASH | 精确匹配策略 |
| `indexed_policy_*` | HASH | 索引策略 (按 IP/端口) |
| `flow_events` | RINGBUF | 流事件上报 |
| `stats_map` | PERCPU_ARRAY | 统计计数器 |

### 6.3 数据包处理流程

```
TC Ingress Hook
      │
      ▼
提取 5-tuple (src_ip, dst_ip, src_port, dst_port, protocol)
      │
      ▼
查找 session_map (会话缓存)
      │
      ├─ 命中 → 直接返回缓存的决策
      │
      └─ 未命中 →
           │
           ▼
      查找 policy_map (精确匹配)
           │
           ├─ 命中 → 缓存决策到 session_map
           │
           └─ 未命中 →
                │
                ▼
           查找索引策略 (label-based)
                │
                ▼
           执行决策 (TC_ACT_OK / TC_ACT_SHOT)
                │
                ▼
           发送流事件到 Ring Buffer
```

---

## 7. 关键数据结构

### 7.1 Flow Event (Protobuf)

```protobuf
// api/proto/flow/flow.proto
message FlowEvent {
    uint64 timestamp_ns = 1;
    uint32 src_ip = 2;      // IPv4，已扩展支持 IPv6
    uint32 dst_ip = 3;
    uint32 src_port = 4;
    uint32 dst_port = 5;
    Protocol protocol = 6;
    Direction direction = 7;
    uint64 packet_count = 8;
    uint64 byte_count = 9;
    string agent_id = 10;
    map<string, string> source_labels = 11;
    map<string, string> dest_labels = 12;
}
```

### 7.2 Policy (Protobuf)

```protobuf
// api/proto/policy/policy.proto
message Policy {
    uint32 rule_id = 1;
    string src_ip = 2;
    string dst_ip = 3;
    uint32 src_port = 4;
    uint32 dst_port = 5;
    Protocol protocol = 6;
    PolicyAction action = 7;
    uint32 priority = 8;
    map<string, string> source_labels = 9;
    map<string, string> dest_labels = 10;
    string description = 11;
    string process_name = 12;    // 进程名匹配
    string process_path = 13;    // 进程路径匹配
    ProcessMatchMode match_mode = 14;
}
```

### 7.3 eBPF 结构 (C)

```c
// src/bpf/headers/common_types.h

// 流键 (用于 session_map)
struct flow_key {
    __u32 src_ip[4];    // IPv6 ready
    __u32 dst_ip[4];
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
};

// 会话值
struct session_value {
    __u8 action;        // ALLOW/DENY/LOG
    __u32 policy_id;
    __u64 last_seen;
    __u64 packet_count;
    __u64 byte_count;
};

// 流事件
struct flow_event {
    __u64 timestamp_ns;
    struct flow_key key;
    __u8 action;
    __u8 direction;
    __u32 policy_id;
    __u64 packet_count;
    __u64 byte_count;
};
```

---

## 8. 学习路径建议

### 8.1 第一天：理解架构

1. 阅读本文档的架构概览部分
2. 运行 `make build` 编译项目
3. 阅读 `src/agent/cmd/main.go` 理解启动流程
4. 阅读 `src/server/cmd/main.go` 理解服务端

### 8.2 第二天：深入 Agent

1. 阅读 `src/agent/pkg/dataplane/` 理解 eBPF 加载
2. 阅读 `src/agent/pkg/policy/` 理解策略管理
3. 阅读 `src/agent/pkg/reporter/` 理解流上报

### 8.3 第三天：深入 Server

1. 阅读 `src/server/pkg/grpc/` 理解 gRPC 服务
2. 阅读 `src/server/pkg/storage/` 理解数据存储
3. 运行测试 `go test ./...`

### 8.4 第四天：深入 eBPF

1. 阅读 `src/bpf/tc_microsegment.bpf.c` 理解包处理
2. 阅读 `src/bpf/headers/` 理解数据结构
3. 使用 `bpftool` 调试 eBPF 程序

### 8.5 第五天：前端开发

1. 运行 `cd web && npm install && npm run dev`
2. 阅读 `web/src/utils/topologyUtils.ts` 理解拓扑
3. 阅读 `web/src/lib/graph/` 理解图数据库

---

## 代码搜索速查

```bash
# 查找函数定义
grep -Ern "^func " --include="*.go" src/

# 查找结构体
grep -Ern "^type .* struct" --include="*.go" src/

# 查找 gRPC 服务
grep -rn "func.*Service" --include="*.go" src/server/pkg/grpc/

# 查找 eBPF map 定义
grep -rn "struct.*__type" --include="*.h" src/bpf/

# 查找 TypeScript 组件
grep -rn "export.*function\|export.*const" --include="*.ts" web/src/
```

---

**最后更新**: 2025-11-22
**维护者**: Claude Code Assistant
**版本**: 2.0.0

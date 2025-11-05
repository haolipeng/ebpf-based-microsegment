# 架构对比分析：当前实现 vs Agent-Server 架构

**文档日期**: 2025-11-04
**作者**: Claude Code
**版本**: 1.0

---

## 执行摘要

**当前项目架构**: **单体架构（Single Binary）**
**是否为 Agent-Server 架构**: **否**

当前项目编译后只生成 **一个可执行文件** `microsegment-agent`，它同时包含：
- eBPF 数据平面（内核级数据包过滤）
- Go 控制平面（策略管理、工作负载管理）
- REST API 服务器（可选，默认启用）
- SQLite 本地数据库（策略、工作负载、流量数据持久化）

这是一个 **一体化的本地部署方案**，适合单节点部署和测试环境，但不是传统的分布式 Agent-Server 架构。

---

## 目录

1. [当前架构详细分析](#1-当前架构详细分析)
2. [传统 Agent-Server 架构](#2-传统-agent-server-架构)
3. [架构对比表](#3-架构对比表)
4. [数据流对比](#4-数据流对比)
5. [适用场景分析](#5-适用场景分析)
6. [迁移方案](#6-迁移方案)
7. [推荐建议](#7-推荐建议)

---

## 1. 当前架构详细分析

### 1.1 架构类型

**单体架构（Monolithic Single-Binary Architecture）**

```
┌─────────────────────────────────────────────────────────────────┐
│                    microsegment-agent (单个可执行文件)           │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  REST API Server (Gin) - http://127.0.0.1:8080             │ │
│  │  ├─ GET  /api/v1/policies                                  │ │
│  │  ├─ GET  /api/v1/flows                                     │ │
│  │  ├─ GET  /api/v1/stats                                     │ │
│  │  └─ GET  /api/v1/health                                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           ↕                                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Control Plane (Go)                                        │ │
│  │  ├─ PolicyManager                                          │ │
│  │  ├─ WorkloadManager                                        │ │
│  │  ├─ GroupManager                                           │ │
│  │  ├─ FlowCollector                                          │ │
│  │  └─ LabelValidator                                         │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           ↕                                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Storage Layer (SQLite)                                    │ │
│  │  ├─ policies.db                                            │ │
│  │  ├─ workloads.db                                           │ │
│  │  └─ flows.db                                               │ │
│  │  (本地文件: /var/lib/microsegment/*.db)                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           ↕                                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  DataPlane Driver (Go + cilium/ebpf)                       │ │
│  │  - eBPF 程序加载/卸载                                       │ │
│  │  - eBPF Maps 访问 (Ring Buffer 读取)                       │ │
│  │  - 统计数据收集                                             │ │
│  └────────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────────┘
                            ↓ (eBPF 系统调用)
┌─────────────────────────────────────────────────────────────────┐
│                   Linux Kernel Space                             │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  TC eBPF Program (tc_microsegment.bpf.c)                   │ │
│  │  - 数据包过滤 (5-tuple matching)                            │ │
│  │  - 会话跟踪 (LRU Hash Maps)                                │ │
│  │  - 策略匹配 (精确 + 通配符)                                 │ │
│  │  - 流事件推送 (Ring Buffer)                                │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  eBPF Maps                                                 │ │
│  │  ├─ session_map (LRU_HASH, 100k entries)                  │ │
│  │  ├─ policy_map (HASH)                                     │ │
│  │  ├─ wildcard_policy_map (HASH)                            │ │
│  │  ├─ stats_map (ARRAY)                                     │ │
│  │  └─ flow_events (RINGBUF, 256KB)                          │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 运行方式

```bash
# 编译
cd src/agent
go build -o microsegment-agent ./cmd

# 运行（单个进程）
sudo ./microsegment-agent \
  --interface lo \
  --enable-api \
  --api-host 127.0.0.1 \
  --api-port 8080
```

**启动过程**（见 [src/agent/cmd/main.go](../src/agent/cmd/main.go:43-106)）：

```go
func runAgent(cmd *cobra.Command, args []string) {
    // 1. 初始化数据平面 (加载 eBPF 程序到内核)
    dp, err := dataplane.New(iface)

    // 2. 创建策略管理器 (本地管理)
    pm := policy.NewManager(dp)

    // 3. 启动 API 服务器 (可选，默认启用)
    if enableAPI {
        apiServer, err = api.NewAPIServer(apiConfig, dp, pm)
        apiServer.Start()  // http://127.0.0.1:8080
    }

    // 4. 启动流事件监控 (本地 Ring Buffer 读取)
    go dp.MonitorFlowEvents()

    // 5. 定期打印统计信息 (本地日志)
    go printStats(statsInterval)

    // 6. 等待信号终止
    <-sig
}
```

### 1.3 数据流

**策略配置流程**:
```
用户 → curl http://localhost:8080/api/v1/policies
     → API Handler → PolicyManager → eBPF Maps (内核)
     → 策略立即生效
```

**流量数据收集流程**:
```
网络数据包 → TC Hook → eBPF 程序 → Ring Buffer 事件
          → Go Collector (collectLoop) → SQLite 本地数据库
          → API 查询返回 → 用户
```

**关键特点**：
- ✅ **本地存储**: SQLite 数据库文件在本地
- ✅ **本地 API**: 监听 127.0.0.1:8080，仅本机访问
- ✅ **单进程**: 所有组件运行在一个进程中
- ✅ **直接访问**: eBPF Maps 通过系统调用直接访问
- ❌ **无远程上报**: 不向中心服务器发送数据

### 1.4 当前架构的优缺点

#### 优点

1. **部署简单**
   - 单个二进制文件，无需安装多个组件
   - 无需配置分布式通信（无 gRPC、无消息队列）
   - 配置简单，仅需本地参数

2. **性能高效**
   - 无网络 I/O 开销（数据无需跨节点传输）
   - 低延迟（本地 Ring Buffer 读取）
   - 直接访问 eBPF Maps

3. **开发调试友好**
   - 日志集中（单进程日志）
   - 调试简单（单进程 gdb/dlv）
   - 无分布式系统复杂性

4. **资源消耗低**
   - 单进程内存占用小
   - 无需额外的服务器节点

#### 缺点

1. **无集中管理**
   - 每个节点独立运行，无法统一查看所有节点状态
   - 策略配置需要逐节点执行
   - 无全局视图

2. **无跨节点分析**
   - 流量数据分散在各节点 SQLite 数据库
   - 无法分析跨节点的应用依赖关系
   - 无全局 Top Talkers 统计

3. **可扩展性差**
   - 不适合大规模部署（100+ 节点）
   - 无法实现集中策略下发
   - 难以实现一致性策略

4. **可观测性弱**
   - 需要逐节点查询 API
   - 无统一监控面板
   - 告警分散

---

## 2. 传统 Agent-Server 架构

### 2.1 架构设计

**两个独立的可执行文件**：

```
┌────────────────────────────────────────────────────────────────┐
│                  Server (中心控制服务器)                        │
│                  可执行文件: microsegment-server                │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐│
│  │  Web API Server (Gin) - http://0.0.0.0:8080                ││
│  │  ├─ GET  /api/v1/policies (全局策略查询)                    ││
│  │  ├─ GET  /api/v1/flows (全局流量查询)                       ││
│  │  ├─ GET  /api/v1/agents (Agent 列表)                        ││
│  │  ├─ GET  /api/v1/dependencies (全局依赖图)                  ││
│  │  └─ POST /api/v1/policies (策略下发)                        ││
│  └────────────────────────────────────────────────────────────┘│
│  ┌────────────────────────────────────────────────────────────┐│
│  │  gRPC Server (监听 Agent 上报)                              ││
│  │  ├─ FlowEventService (接收流事件)                           ││
│  │  ├─ PolicySyncService (策略同步)                            ││
│  │  └─ HeartbeatService (Agent 健康检查)                       ││
│  └────────────────────────────────────────────────────────────┘│
│  ┌────────────────────────────────────────────────────────────┐│
│  │  Central Storage (PostgreSQL / TimescaleDB)                ││
│  │  ├─ flows 表 (所有节点的流量数据)                           ││
│  │  ├─ policies 表 (全局策略)                                  ││
│  │  ├─ agents 表 (Agent 注册信息)                              ││
│  │  └─ events 表 (审计日志)                                    ││
│  └────────────────────────────────────────────────────────────┘│
│  ┌────────────────────────────────────────────────────────────┐│
│  │  Analysis Engine                                           ││
│  │  ├─ Global Dependency Mapping                             ││
│  │  ├─ Anomaly Detection                                     ││
│  │  └─ Policy Recommendation                                 ││
│  └────────────────────────────────────────────────────────────┘│
└───────────────────────────┬────────────────────────────────────┘
                            ↓ gRPC / HTTP
┌─────────────────────────────────────────────────────────────────┐
│               Agent 1 (节点 192.168.1.10)                        │
│               可执行文件: microsegment-agent                     │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  gRPC Client (上报到 Server)                                │ │
│  │  ├─ 流事件上报 (每秒批量)                                    │ │
│  │  ├─ 统计数据上报 (每分钟)                                    │ │
│  │  ├─ 心跳上报 (每10秒)                                        │ │
│  │  └─ 策略拉取 (轮询/订阅)                                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Local Control Plane                                       │ │
│  │  ├─ PolicyManager (从 Server 同步)                          │ │
│  │  ├─ FlowCollector (本地收集 + 上报)                         │ │
│  │  └─ Local Cache (SQLite, 可选)                             │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  eBPF DataPlane (同当前实现)                                │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│               Agent 2 (节点 192.168.1.11)                        │
│               (架构同 Agent 1)                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│               Agent N (节点 192.168.1.N)                         │
│               (架构同 Agent 1)                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 运行方式

```bash
# 编译（生成两个二进制文件）
cd src/server
go build -o microsegment-server ./cmd

cd src/agent
go build -o microsegment-agent ./cmd

# 运行 Server (中心节点)
./microsegment-server \
  --grpc-port 9090 \
  --http-port 8080 \
  --db-url "postgresql://user:pass@localhost:5432/microsegment"

# 运行 Agent (每个客户节点)
sudo ./microsegment-agent \
  --interface eth0 \
  --server-url grpc://server.example.com:9090 \
  --agent-id node-01
```

### 2.3 数据流

**策略配置流程**:
```
管理员 → POST http://server:8080/api/v1/policies (创建策略)
       → Server 保存到 PostgreSQL
       → Server 推送策略到所有 Agents (gRPC)
       → Agent 更新 eBPF Maps
       → 策略在所有节点生效
```

**流量数据收集流程**:
```
网络数据包 (节点 1) → eBPF 程序 → Ring Buffer
                   → Agent Collector → gRPC 批量上报
                   → Server → PostgreSQL (flows 表)

网络数据包 (节点 2) → eBPF 程序 → Ring Buffer
                   → Agent Collector → gRPC 批量上报
                   → Server → PostgreSQL (flows 表)

用户 → GET http://server:8080/api/v1/flows?source_ip=192.168.1.10
     → Server 查询 PostgreSQL (所有节点数据)
     → 返回全局流量视图
```

### 2.4 Agent-Server 架构的优缺点

#### 优点

1. **集中管理**
   - 统一策略下发（一次配置，全网生效）
   - 集中监控面板
   - 全局视图

2. **跨节点分析**
   - 全局应用依赖关系图
   - 跨节点流量分析
   - 异常检测和安全事件关联

3. **可扩展性强**
   - 支持成百上千个节点
   - 水平扩展 Server 组件
   - 负载均衡

4. **企业级功能**
   - 多租户隔离
   - 审计日志
   - RBAC 权限管理
   - 合规性报告

#### 缺点

1. **部署复杂**
   - 需要独立的 Server 节点
   - 需要配置数据库（PostgreSQL/TimescaleDB）
   - 需要配置网络通信（gRPC、防火墙）

2. **网络依赖**
   - Agent 依赖 Server 可达性
   - 网络故障影响数据上报
   - 需要考虑网络分区问题

3. **资源消耗高**
   - Server 需要独立硬件资源
   - 数据库存储成本
   - 网络带宽消耗

4. **开发复杂度高**
   - 分布式系统调试困难
   - 需要处理时钟同步、数据一致性
   - gRPC 接口设计和版本管理

---

## 3. 架构对比表

| 维度 | 当前架构（单体） | Agent-Server 架构 |
|-----|----------------|-------------------|
| **可执行文件数量** | 1 个 (`microsegment-agent`) | 2 个 (`microsegment-agent` + `microsegment-server`) |
| **部署位置** | 每个节点独立运行 | Agent 在节点，Server 在中心 |
| **数据存储** | 本地 SQLite | 中心 PostgreSQL/TimescaleDB |
| **API 访问** | 本地 `127.0.0.1:8080` | 全局 `server:8080` |
| **策略管理** | 逐节点配置 | 中心下发 |
| **流量数据** | 本地查询 | 全局聚合查询 |
| **通信方式** | 无网络通信 | gRPC/HTTP (Agent → Server) |
| **适用规模** | 1-10 节点 | 10-10000 节点 |
| **部署复杂度** | 低（单文件） | 高（多组件） |
| **开发复杂度** | 低（单进程） | 高（分布式） |
| **性能开销** | 低（无网络 I/O） | 中（网络上报） |
| **可观测性** | 弱（分散） | 强（集中） |
| **故障域** | 单节点 | 全局（Server SPOF） |
| **数据一致性** | 无问题（本地） | 需要考虑（最终一致性） |
| **跨节点分析** | ❌ 不支持 | ✅ 支持 |
| **策略一致性** | ❌ 难以保证 | ✅ 保证 |
| **实时监控** | ❌ 需逐节点查询 | ✅ 统一面板 |
| **成本** | 低（无额外硬件） | 高（Server + 数据库） |

---

## 4. 数据流对比

### 4.1 策略配置流程对比

#### 当前架构
```
┌─────────┐      HTTP POST        ┌──────────────┐
│  管理员  │ ───────────────────> │ Agent 1 API  │
│         │  /api/v1/policies    │ (localhost:  │
└─────────┘                       │  8080)       │
                                  └──────┬───────┘
                                         ↓
                                  ┌──────────────┐
                                  │ eBPF Maps    │
                                  │ (节点 1)      │
                                  └──────────────┘

┌─────────┐      HTTP POST        ┌──────────────┐
│  管理员  │ ───────────────────> │ Agent 2 API  │
│         │  /api/v1/policies    │ (节点 2:8080)│
└─────────┘                       └──────┬───────┘
                                         ↓
                                  ┌──────────────┐
                                  │ eBPF Maps    │
                                  │ (节点 2)      │
                                  └──────────────┘

⚠️ 问题: 需要手动配置每个节点，策略不一致风险
```

#### Agent-Server 架构
```
┌─────────┐      HTTP POST        ┌──────────────┐
│  管理员  │ ───────────────────> │ Server API   │
│         │  /api/v1/policies    │ (中心)        │
└─────────┘                       └──────┬───────┘
                                         ↓
                                  ┌──────────────┐
                                  │ PostgreSQL   │
                                  │ (持久化)      │
                                  └──────┬───────┘
                                         ↓ gRPC Push
                        ┌────────────────┼────────────────┐
                        ↓                ↓                ↓
                 ┌──────────┐     ┌──────────┐     ┌──────────┐
                 │ Agent 1  │     │ Agent 2  │     │ Agent N  │
                 │ eBPF Maps│     │ eBPF Maps│     │ eBPF Maps│
                 └──────────┘     └──────────┘     └──────────┘

✅ 优点: 一次配置，全网生效，策略一致性保证
```

### 4.2 流量数据查询流程对比

#### 当前架构
```
┌─────────┐    GET /api/v1/flows   ┌──────────────┐
│  用户    │ ─────────────────────> │ Agent 1 API  │
└─────────┘    (localhost:8080)    └──────┬───────┘
                                          ↓
                                   ┌──────────────┐
                                   │ SQLite       │
                                   │ (节点 1 数据) │
                                   └──────────────┘

返回: 仅节点 1 的流量数据

⚠️ 问题: 无法查询全局流量，需要手动汇总
```

#### Agent-Server 架构
```
┌─────────┐    GET /api/v1/flows   ┌──────────────┐
│  用户    │ ─────────────────────> │ Server API   │
└─────────┘    (server:8080)       └──────┬───────┘
                                          ↓
                                   ┌──────────────┐
                                   │ PostgreSQL   │
                                   │ (全局数据)    │
                                   │ - 节点 1 flows│
                                   │ - 节点 2 flows│
                                   │ - 节点 N flows│
                                   └──────────────┘

返回: 所有节点的流量数据，支持全局过滤

✅ 优点: 全局视图，支持跨节点分析
```

### 4.3 流事件上报流程对比

#### 当前架构
```
网络数据包 (节点 1) → eBPF → Ring Buffer → Collector → SQLite (节点 1)
网络数据包 (节点 2) → eBPF → Ring Buffer → Collector → SQLite (节点 2)
网络数据包 (节点 N) → eBPF → Ring Buffer → Collector → SQLite (节点 N)

数据分散在各节点，无法聚合分析
```

#### Agent-Server 架构
```
网络数据包 (节点 1) → eBPF → Ring Buffer → Collector → gRPC Client ──┐
网络数据包 (节点 2) → eBPF → Ring Buffer → Collector → gRPC Client ──┤
网络数据包 (节点 N) → eBPF → Ring Buffer → Collector → gRPC Client ──┤
                                                                     ↓
                                                              ┌──────────────┐
                                                              │ Server gRPC  │
                                                              │   Receiver   │
                                                              └──────┬───────┘
                                                                     ↓
                                                              ┌──────────────┐
                                                              │ PostgreSQL   │
                                                              │ (全局流表)    │
                                                              └──────────────┘

数据集中存储，支持全局分析和关联查询
```

---

## 5. 适用场景分析

### 5.1 当前架构适用场景

✅ **适合**:
1. **单节点部署**
   - 开发测试环境
   - 个人学习和实验
   - PoC (Proof of Concept)

2. **小规模部署 (1-10 节点)**
   - 小型企业内网
   - 独立的微服务集群
   - 对集中管理无强需求

3. **边缘计算场景**
   - 边缘节点独立运行
   - 网络连接不稳定
   - 低延迟要求

4. **学习和教学**
   - eBPF 学习项目（如当前）
   - 简单的演示环境
   - 快速原型验证

❌ **不适合**:
1. **大规模部署 (100+ 节点)**
2. **需要全局视图和跨节点分析**
3. **统一策略管理需求**
4. **企业级安全合规要求**

---

### 5.2 Agent-Server 架构适用场景

✅ **适合**:
1. **大规模部署 (10-10000 节点)**
   - Kubernetes 集群 (多节点)
   - 数据中心网络
   - 云环境多租户

2. **需要集中管理**
   - 统一策略下发
   - 集中监控和告警
   - 审计和合规

3. **跨节点分析需求**
   - 应用依赖关系图
   - 全局流量分析
   - 异常检测和威胁情报

4. **企业生产环境**
   - SLA 保证
   - 高可用性需求
   - RBAC 权限管理

❌ **不适合**:
1. **单节点或小规模部署** (过度设计)
2. **网络隔离环境** (Agent 无法连接 Server)
3. **资源受限环境** (无法部署独立 Server)
4. **快速原型和学习项目** (复杂度高)

---

## 6. 迁移方案

如果未来需要从当前单体架构迁移到 Agent-Server 架构，建议的迁移路径：

### 6.1 Phase 1: 设计和规划 (1-2 周)

**任务**:
1. 定义 gRPC 接口 (protobuf 定义)
   ```protobuf
   // proto/flow.proto
   service FlowService {
     rpc ReportFlowEvents(stream FlowEvent) returns (ReportResponse);
     rpc SyncPolicies(SyncRequest) returns (stream Policy);
     rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
   }
   ```

2. 设计 Server 架构
   - 选择数据库 (PostgreSQL + TimescaleDB)
   - 设计表结构
   - 定义 API 接口

3. 定义数据迁移策略
   - SQLite → PostgreSQL 迁移工具
   - 数据格式转换

### 6.2 Phase 2: Server 实现 (3-4 周)

**目录结构**:
```
src/
├── agent/                # 现有 Agent 代码
├── server/               # 新增 Server 代码
│   ├── cmd/
│   │   └── main.go      # Server 入口
│   ├── pkg/
│   │   ├── api/         # HTTP API
│   │   ├── grpc/        # gRPC 服务
│   │   ├── storage/     # PostgreSQL 存储
│   │   ├── aggregator/  # 数据聚合
│   │   └── policy/      # 策略分发
│   └── go.mod
└── proto/                # gRPC protobuf 定义
    ├── flow.proto
    ├── policy.proto
    └── agent.proto
```

**实现步骤**:
1. 实现 gRPC Server 框架
2. 实现 PostgreSQL 存储层
3. 实现流事件接收和持久化
4. 实现策略同步服务
5. 实现 HTTP API (查询接口)

### 6.3 Phase 3: Agent 改造 (2-3 周)

**修改清单**:

1. **添加 gRPC Client** ([src/agent/pkg/reporter/grpc_client.go](../src/agent/pkg/reporter/grpc_client.go))
   ```go
   package reporter

   type GRPCReporter struct {
       client   pb.FlowServiceClient
       conn     *grpc.ClientConn
       queue    chan *flow.FlowEvent
       batchSize int
   }

   func (r *GRPCReporter) Start() error {
       go r.reportLoop()  // 批量上报
   }

   func (r *GRPCReporter) reportLoop() {
       ticker := time.NewTicker(1 * time.Second)
       batch := make([]*pb.FlowEvent, 0, r.batchSize)

       for {
           select {
           case event := <-r.queue:
               batch = append(batch, convertToProto(event))
               if len(batch) >= r.batchSize {
                   r.sendBatch(batch)
                   batch = batch[:0]
               }
           case <-ticker.C:
               if len(batch) > 0 {
                   r.sendBatch(batch)
                   batch = batch[:0]
               }
           }
       }
   }
   ```

2. **修改 FlowCollector** ([src/agent/pkg/flow/collector.go](../src/agent/pkg/flow/collector.go:100-150))
   ```go
   type Collector struct {
       // ... 现有字段
       reporter Reporter  // 新增: 上报接口 (本地 SQLite 或 gRPC)
   }

   func (c *Collector) processFlowEvent(event *FlowEvent) {
       // 丰富标签
       flow := c.enrichWithLabels(event)

       // 本地存储 (可选，用于缓存)
       if c.config.EnableLocalCache {
           c.storage.SaveFlow(flow)
       }

       // 上报到 Server
       if c.reporter != nil {
           c.reporter.Report(flow)
       }
   }
   ```

3. **修改 main.go** ([src/agent/cmd/main.go](../src/agent/cmd/main.go:43-106))
   ```go
   var (
       // ... 现有参数
       serverURL      string  // 新增: Server 地址
       enableReporter bool    // 新增: 是否启用上报
   )

   func runAgent(cmd *cobra.Command, args []string) {
       // ... 现有初始化

       // 创建上报器
       var reporter reporter.Reporter
       if enableReporter {
           reporter, err = reporter.NewGRPCReporter(serverURL)
           if err != nil {
               log.Fatalf("Failed to create reporter: %v", err)
           }
           reporter.Start()
       }

       // 创建 Collector 时注入 reporter
       collector := flow.NewCollector(dp.RingBuf(), storage, reporter)
       collector.Start()

       // ... 其余逻辑
   }
   ```

### 6.4 Phase 4: 集成测试 (1-2 周)

**测试场景**:
1. **单 Agent 测试**
   - Agent 启动并注册到 Server
   - 流事件正常上报
   - 策略下发生效

2. **多 Agent 测试**
   - 3-5 个 Agent 同时上报
   - 数据正确聚合
   - 策略一致性

3. **故障测试**
   - Server 宕机恢复
   - Agent 重连机制
   - 数据丢失处理

### 6.5 Phase 5: 部署和迁移 (1-2 周)

**迁移步骤**:
1. 部署 Server (中心节点)
2. 启动 PostgreSQL 数据库
3. 逐步升级 Agent (灰度发布)
4. 迁移历史数据 (SQLite → PostgreSQL)
5. 切换流量到新 API
6. 监控和优化

**总时间**: **8-13 周** (2-3 个月)

---

## 7. 推荐建议

### 7.1 短期建议（当前项目）

基于项目当前状态（学习和 PoC 阶段），**建议保持当前单体架构**：

**理由**:
1. ✅ **符合学习目标**: 当前架构简单易懂，适合 eBPF 学习
2. ✅ **开发效率高**: 单进程开发和调试简单
3. ✅ **功能完整**: 已实现核心功能（策略、工作负载、流量收集）
4. ✅ **文档完善**: 32,000 字实施文档支持

**优化方向**（保持单体架构前提下）:
1. **改进 API 访问控制**
   ```go
   // 支持远程访问 (可选)
   rootCmd.Flags().StringVar(&apiHost, "api-host", "0.0.0.0", "API server host")
   ```
   - 允许 `--api-host 0.0.0.0` 支持远程访问
   - 添加 API 认证 (Token/Basic Auth)

2. **添加简单的聚合脚本**
   ```bash
   #!/bin/bash
   # scripts/aggregate_flows.sh
   # 从多个 Agent 收集数据并聚合

   AGENTS=("192.168.1.10:8080" "192.168.1.11:8080" "192.168.1.12:8080")

   for agent in "${AGENTS[@]}"; do
       curl "http://$agent/api/v1/flows" >> all_flows.json
   done

   jq -s 'add' all_flows.json > aggregated.json
   ```

3. **完成集成**
   - 完成 DataPlane Ring Buffer 集成
   - 完成 WorkloadManager 标签丰富
   - 完成 Phase 4 WebSocket 实时推送

### 7.2 中期建议（如需扩展）

如果项目需要支持 **10+ 节点** 或 **生产环境部署**，建议：

1. **评估实际需求**
   - 节点数量 > 10? → 考虑 Agent-Server
   - 需要全局视图? → 考虑 Agent-Server
   - 需要集中策略? → 考虑 Agent-Server

2. **渐进式迁移**
   - 先实现 gRPC 上报接口（Agent 改造）
   - 再实现 Server 接收和存储
   - 最后迁移数据和切换流量

3. **借鉴成熟方案**
   - 参考 Cilium Hubble (Agent-Server 架构)
   - 参考 Prometheus (Pull 模型 vs Push 模型)
   - 参考 Elastic APM (分布式追踪)

### 7.3 长期建议（生产化）

如果项目最终要生产化，建议架构演进路径：

```
Phase 1: 单体架构 (当前)
         ↓
Phase 2: 混合架构 (可选上报)
         - Agent 支持本地 + 远程上报
         - Server 可选部署
         ↓
Phase 3: Agent-Server 架构
         - 完整的分布式系统
         - 高可用性
         ↓
Phase 4: 云原生架构
         - Kubernetes Operator
         - Helm Charts
         - 自动扩缩容
```

---

## 8. 总结

### 8.1 核心结论

| 问题 | 答案 |
|-----|------|
| **当前项目是 Agent-Server 架构吗?** | ❌ **不是**，是单体架构 |
| **编译后是两个程序吗?** | ❌ **不是**，只有一个 `microsegment-agent` |
| **Agent 会上报事件到 Server 吗?** | ❌ **不会**，数据存储在本地 SQLite |
| **是否适合当前学习目标?** | ✅ **是**，简单易懂适合学习 |
| **是否需要改造为 Agent-Server?** | ⏸️ **暂时不需要**，除非需要大规模部署 |

### 8.2 关键差异总结

**当前架构 (单体)**:
- 📦 **1 个可执行文件**: `microsegment-agent`
- 💾 **本地存储**: SQLite 数据库
- 🌐 **本地 API**: `127.0.0.1:8080`
- 🎯 **适用场景**: 单节点、学习、PoC
- ⚡ **优点**: 简单、高效、低成本
- ⚠️ **缺点**: 无集中管理、难扩展

**Agent-Server 架构 (分布式)**:
- 📦 **2 个可执行文件**: `microsegment-agent` + `microsegment-server`
- 💾 **中心存储**: PostgreSQL/TimescaleDB
- 🌐 **全局 API**: `server:8080`
- 🎯 **适用场景**: 多节点、生产环境
- ⚡ **优点**: 集中管理、可扩展、全局视图
- ⚠️ **缺点**: 复杂、网络依赖、成本高

### 8.3 最终建议

**当前阶段（学习和开发）**:
- ✅ **保持单体架构**
- ✅ **完成 Phase 1-5 实施**（Flow Collection API）
- ✅ **补充集成测试和性能测试**
- ✅ **完善文档和示例**

**未来扩展（如需大规模部署）**:
- ⏳ **评估实际需求**（节点数、管理需求）
- ⏳ **设计 gRPC 接口**（3-6 个月后）
- ⏳ **渐进式迁移到 Agent-Server**（分阶段实施）

---

**文档版本**: 1.0
**最后更新**: 2025-11-04
**下次审查**: 项目 Phase 5 完成后

**相关文档**:
- [项目架构](../openspec/project.md)
- [Flow Collection API 实施总结](./flow-collection-implementation-summary.md)
- [Flow 实施进度](./flow-implementation-progress.md)

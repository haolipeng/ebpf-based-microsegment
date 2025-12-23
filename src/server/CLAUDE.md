[根目录](../../CLAUDE.md) > **src/server**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# Server 模块

eBPF 微隔离系统的中心化控制平面，负责策略管理、Agent 协调和数据聚合。

## 核心职责

| 职责 | 说明 | 核心包 |
|------|------|--------|
| 策略管理 | 中心化存储和分发策略规则 | [policy](pkg/policy/CLAUDE.md), [storage](pkg/storage/CLAUDE.md) |
| Agent 管理 | 跟踪状态、心跳检测 | [grpc](pkg/grpc/CLAUDE.md) |
| 流数据聚合 | 收集和聚合多 Agent 流事件 | [aggregator](pkg/aggregator/CLAUDE.md) |
| 拓扑构建 | 基于流数据构建网络拓扑 | [topology](pkg/topology/CLAUDE.md), [graph](pkg/graph/CLAUDE.md) |
| HTTP API | RESTful API 给 Web UI | [api](pkg/api/CLAUDE.md) |
| gRPC 服务 | 供 Agent 调用的接口 | [grpc](pkg/grpc/CLAUDE.md) |

## 快速开始

```bash
# 构建
cd src/server && go build -o ../../bin/microsegment-server ./cmd

# 运行
./bin/microsegment-server --config ../../config/server.yaml

# 测试
go test -v ./...
```

## 入口文件

- **主程序**: `cmd/main.go`
- **配置文件**: `../../config/server.yaml`

## 对外接口

### HTTP REST API (`localhost:8080/api/v1`)

| 分类 | 端点 | 方法 | 描述 |
|------|------|------|------|
| 策略 | `/policies` | GET/POST | 策略列表/创建 |
| 策略 | `/policies/:id` | GET/PUT/DELETE | 单个策略操作 |
| 流 | `/flows` | GET | 查询流事件 |
| 流 | `/flows/aggregated` | GET | 聚合数据 |
| Agent | `/agents` | GET | Agent 列表 |
| 拓扑 | `/topology` | GET | 网络拓扑 |
| 告警 | `/alerts` | GET | 告警列表 |

### gRPC 服务 (`localhost:50051`)

| 服务 | 方法 | 描述 |
|------|------|------|
| FlowService | `ReportFlows` | Agent 上报流事件 |
| FlowService | `StreamFlows` | 流事件订阅 |
| PolicyService | `GetPolicies` | 获取策略 |
| PolicyService | `SubscribePolicies` | 订阅策略变更 |
| AgentService | `RegisterAgent` | Agent 注册 |
| AgentService | `Heartbeat` | 心跳 |

## 子目录索引 (pkg/)

| 子目录 | 职责 | 文档 |
|--------|------|------|
| **storage** | PostgreSQL 数据访问层 (Bun ORM) | [→](pkg/storage/CLAUDE.md) |
| **grpc** | gRPC 服务实现 | [→](pkg/grpc/CLAUDE.md) |
| **api** | HTTP API 处理器 | [→](pkg/api/CLAUDE.md) |
| **aggregator** | 流数据聚合 | [→](pkg/aggregator/CLAUDE.md) |
| **topology** | 网络拓扑构建 | [→](pkg/topology/CLAUDE.md) |
| **policy** | 策略压缩存储 | [→](pkg/policy/CLAUDE.md) |
| **graph** | 图算法工具 | [→](pkg/graph/CLAUDE.md) |
| **pubsub** | 策略发布/订阅 | [→](pkg/pubsub/CLAUDE.md) |
| **identity** | 身份管理 | [→](pkg/identity/CLAUDE.md) |
| **config** | 配置管理 | [→](pkg/config/CLAUDE.md) |
| **testutil** | 测试工具 | [→](pkg/testutil/CLAUDE.md) |

## 关键依赖

| 依赖 | 用途 |
|------|------|
| `gin-gonic/gin` | HTTP API |
| `grpc` | gRPC 服务 |
| `uptrace/bun` | PostgreSQL ORM |
| `golang-migrate` | 数据库迁移 |
| `testcontainers-go` | 集成测试 |

## 数据库

- **PostgreSQL 14+**
- 主要表: `policies`, `flows`, `agents`, `alerts`
- 迁移脚本: `scripts/migrate.sh`

---

**最后更新**: 2025-12-23

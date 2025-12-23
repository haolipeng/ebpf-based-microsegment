[根目录](../../CLAUDE.md) > **src/agent**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# Agent 模块

eBPF 微隔离系统的边缘代理，负责数据平面管理、策略执行和流量监控。

## 核心职责

| 职责 | 说明 | 核心包 |
|------|------|--------|
| 数据平面 | 加载 eBPF 程序，管理 TC/XDP Hook | [dataplane](pkg/dataplane/CLAUDE.md) |
| 策略执行 | 编译策略规则，下发到 eBPF Maps | [policy](pkg/policy/CLAUDE.md) |
| 工作负载发现 | 自动发现容器/Pod，采集标签 | [workload](pkg/workload/CLAUDE.md) |
| 流量监控 | 收集流事件，上报到 Server | [flow](pkg/flow/CLAUDE.md), [reporter](pkg/reporter/CLAUDE.md) |
| 本地 API | RESTful API 用于管理和调试 | [api](pkg/api/CLAUDE.md) |

## 快速开始

```bash
# 构建
cd src/agent && go build -o ../../bin/microsegment-agent ./cmd

# 运行（需要 root）
sudo ./bin/microsegment-agent --config config/agent-server.yaml  # Agent-Server 模式
sudo ./bin/microsegment-agent --config config/standalone.yaml    # Standalone 模式

# 测试
go test -v ./...                    # 单元测试
sudo go test -v ./test/e2e/...      # E2E 测试
```

## 入口文件

- **主程序**: `cmd/main.go`
- **配置文件**: `config/agent-server.yaml` (生产), `config/standalone.yaml` (调试)

## 对外接口

### RESTful API (`localhost:8081/api/v1`)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/policies` | GET/POST | 策略管理 |
| `/policies/:id` | GET/PUT/DELETE | 单个策略 |
| `/workloads` | GET | 工作负载 |
| `/groups` | GET/POST | 分组管理 |
| `/flows` | GET | 流事件 |
| `/statistics` | GET | 统计数据 |

### gRPC 客户端 (连接 Server)

- **FlowService**: 上报流事件
- **PolicyService**: 订阅策略变更
- **AgentService**: 心跳和状态

## 子目录索引 (pkg/)

| 子目录 | 职责 | 文档 |
|--------|------|------|
| **dataplane** | eBPF 程序加载和管理 | [→](pkg/dataplane/CLAUDE.md) |
| **policy** | 策略存储与编译 | [→](pkg/policy/CLAUDE.md) |
| **flow** | 流事件收集与聚合 | [→](pkg/flow/CLAUDE.md) |
| **workload** | 工作负载发现 | [→](pkg/workload/CLAUDE.md) |
| **groups** | 分组和选择器 | [→](pkg/groups/CLAUDE.md) |
| **reporter** | gRPC 上报客户端 | [→](pkg/reporter/CLAUDE.md) |
| **api** | RESTful API 服务器 | [→](pkg/api/CLAUDE.md) |
| **k8s** | Kubernetes 集成 | [→](pkg/k8s/CLAUDE.md) |
| **runtime** | 容器运行时适配 | [→](pkg/runtime/CLAUDE.md) |
| **labels** | 标签验证与合并 | [→](pkg/labels/CLAUDE.md) |
| **conntrack** | 连接跟踪/NAT 同步 | [→](pkg/conntrack/CLAUDE.md) |
| **fragment** | IP 分片处理 | [→](pkg/fragment/CLAUDE.md) |
| **session** | 会话管理 | [→](pkg/session/CLAUDE.md) |
| **process** | 进程监控 | [→](pkg/process/CLAUDE.md) |
| **identity** | 身份识别 | [→](pkg/identity/CLAUDE.md) |
| **ipcache** | IP 缓存管理 | [→](pkg/ipcache/CLAUDE.md) |
| **config** | 配置管理 | [→](pkg/config/CLAUDE.md) |
| **client** | gRPC 客户端封装 | [→](pkg/client/CLAUDE.md) |
| **benchmark** | 性能基准测试 | [→](pkg/benchmark/CLAUDE.md) |
| **testutil** | 测试工具 | [→](pkg/testutil/CLAUDE.md) |

## 关键依赖

| 依赖 | 用途 |
|------|------|
| `cilium/ebpf` | eBPF 程序加载 |
| `vishvananda/netlink` | 网络操作 |
| `gin-gonic/gin` | HTTP API |
| `grpc` | gRPC 通信 |
| `k8s.io/client-go` | K8s 集成 |

## 调试技巧

```bash
# 查看 eBPF 程序
sudo bpftool prog show

# 查看 Maps
sudo bpftool map dump name policy_map

# 查看 API
curl http://localhost:8081/api/v1/health
```

---

**最后更新**: 2025-12-23

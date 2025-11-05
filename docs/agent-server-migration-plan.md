# Agent-Server 架构迁移计划

**文档日期**: 2025-11-04
**版本**: 1.0
**状态**: 提案阶段

---

## 📋 总体概述

本文档汇总了从当前单体架构迁移到 Agent-Server 分布式架构所需的 **5 个 OpenSpec 提案**。

### 当前架构

**单体架构 (Monolithic)**:
- 1 个可执行文件: `microsegment-agent`
- 本地 SQLite 存储
- 本地 API 服务器 (127.0.0.1:8080)
- 适用: 1-10 节点

### 目标架构

**Agent-Server 架构 (Distributed)**:
- 2 个可执行文件: `microsegment-agent` + `microsegment-server`
- 中心 PostgreSQL + TimescaleDB 存储
- gRPC 通信 (Agent → Server)
- 全局 API 服务器 (server:8080)
- 适用: 10-10000 节点

---

## 🎯 OpenSpec 提案清单

### 提案 1: add-grpc-protocol-definitions ⭐
**优先级**: P0 (必须先完成)
**工作量**: 3-5 天
**依赖**: 无

**目标**: 定义 Agent-Server 通信的 gRPC 接口

**核心内容**:
- ✅ Protocol Buffers 定义 (common.proto, flow.proto, policy.proto, agent.proto)
- ✅ gRPC 服务定义 (FlowService, PolicyService, AgentService)
- ✅ 代码生成脚本 (Makefile + shell 脚本)
- ✅ Go 依赖包配置

**文件位置**: `openspec/changes/add-grpc-protocol-definitions/`

**关键产出**:
```
proto/
├── common.proto      # 通用类型和枚举
├── flow.proto        # 流事件相关定义
├── policy.proto      # 策略相关定义
└── agent.proto       # Agent 管理定义

src/proto/            # 生成的 Go 代码
└── ...
```

---

### 提案 2: add-server-component ⭐
**优先级**: P0
**工作量**: 7-10 天
**依赖**: add-grpc-protocol-definitions

**目标**: 实现独立的 Server 组件 (第二个可执行文件)

**核心内容**:
- ✅ gRPC 服务端实现 (FlowService, PolicyService, AgentService)
- ✅ PostgreSQL + TimescaleDB 中心存储
- ✅ HTTP API (全局流量查询、Agent 管理、策略下发)
- ✅ 数据聚合引擎 (全局依赖分析、Top Talkers)

**文件位置**: `openspec/changes/add-server-component/`

**关键产出**:
```
src/server/
├── cmd/main.go                    # Server 入口
├── pkg/grpc/                      # gRPC 服务实现
├── pkg/storage/                   # PostgreSQL 存储
├── pkg/api/                       # HTTP API
└── migrations/                    # 数据库迁移脚本
```

**编译产物**: `microsegment-server` 可执行文件

---

### 提案 3: refactor-agent-for-remote-reporting ⭐
**优先级**: P0
**工作量**: 5-7 天
**依赖**: add-grpc-protocol-definitions, add-server-component

**目标**: 改造 Agent 支持远程上报,保持向后兼容

**核心内容**:
- ✅ 支持两种模式 (standalone 和 agent-server)
- ✅ gRPC Client 实现 (流事件上报、策略同步)
- ✅ Reporter 接口重构 (LocalReporter 和 GRPCReporter)
- ✅ PolicyManager 改造 (支持 Server 同步)
- ✅ 心跳机制实现
- ✅ 配置文件支持

**文件位置**: `openspec/changes/refactor-agent-for-remote-reporting/`

**关键改动**:
```
src/agent/pkg/
├── reporter/
│   ├── reporter.go               # Reporter 接口
│   ├── local_reporter.go         # SQLite 实现
│   └── grpc_reporter.go          # gRPC 实现
├── flow/collector.go             # 支持可插拔 Reporter
├── policy/manager.go             # 支持 Server 同步
└── agent/client.go               # gRPC Client (注册、心跳)
```

---

### 提案 4: add-database-migration-tools
**优先级**: P1
**工作量**: 2-3 天
**依赖**: add-server-component

**目标**: 提供从 SQLite 到 PostgreSQL 的数据迁移工具

**核心内容**:
- ✅ SQLite 数据读取器
- ✅ PostgreSQL 数据写入器
- ✅ 数据验证和一致性检查
- ✅ 批量迁移 (1000+ flows/batch)
- ✅ 进度显示和日志

**文件位置**: `openspec/changes/add-database-migration-tools/`

**关键产出**:
```
src/tools/migrate/
├── main.go                       # 迁移工具入口
├── sqlite_reader.go              # SQLite 读取
├── postgres_writer.go            # PostgreSQL 写入
└── validator.go                  # 数据验证
```

**使用示例**:
```bash
./migrate \
  --source sqlite:///var/lib/agent/flows.db \
  --target postgres://user:pass@localhost:5432/microsegment \
  --batch-size 1000
```

---

### 提案 5: add-deployment-configurations
**优先级**: P1
**工作量**: 3-4 天
**依赖**: add-server-component, refactor-agent-for-remote-reporting

**目标**: 提供生产级部署配置和文档

**核心内容**:
- ✅ Docker 部署 (Dockerfile + docker-compose.yml)
- ✅ Kubernetes 部署 (DaemonSet + Deployment + Service)
- ✅ Systemd 服务配置
- ✅ Prometheus 监控配置
- ✅ Grafana 仪表板
- ✅ 部署文档 (单节点、小规模、大规模)

**文件位置**: `openspec/changes/add-deployment-configurations/`

**关键产出**:
```
deployments/
├── docker/
│   ├── Dockerfile.agent
│   ├── Dockerfile.server
│   └── docker-compose.yml
├── k8s/
│   ├── agent-daemonset.yaml
│   ├── server-deployment.yaml
│   ├── server-service.yaml
│   └── postgres-statefulset.yaml
├── systemd/
│   ├── microsegment-agent.service
│   └── microsegment-server.service
└── monitoring/
    ├── prometheus.yml
    └── grafana-dashboard.json
```

---

## 📊 实施时间线

### 阶段划分

```
┌─────────────────────────────────────────────────────────────┐
│ 阶段 1: 基础设施 (3-5 天)                                   │
│ ├─ add-grpc-protocol-definitions                           │
│ └─ 输出: proto 定义 + Go 代码生成                           │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ 阶段 2: 核心组件 (12-17 天,可部分并行)                      │
│ ├─ add-server-component (7-10 天)                          │
│ └─ refactor-agent-for-remote-reporting (5-7 天)            │
│    (Server 完成 gRPC 接口后可开始 Agent 改造)               │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ 阶段 3: 工具和部署 (5-7 天,可并行)                          │
│ ├─ add-database-migration-tools (2-3 天)                   │
│ └─ add-deployment-configurations (3-4 天)                  │
└─────────────────────────────────────────────────────────────┘
```

### 时间估算

| 实施方式 | 工作日 | 日历周 | 说明 |
|---------|--------|--------|------|
| **串行实施** | 20-29 天 | 4-6 周 | 按依赖顺序逐个完成 |
| **并行实施** | 20-22 天 | 4-5 周 | 阶段 2 和阶段 3 部分并行 |

**推荐**: 并行实施,可在 **4-5 周**内完成。

---

## 🔄 提案依赖关系图

```
add-grpc-protocol-definitions (P0, 3-5 天)
         │
         ├──────────────────────────┐
         ↓                          ↓
add-server-component      refactor-agent-for-remote-reporting
    (P0, 7-10 天)              (P0, 5-7 天)
         │                          │
         │   ┌──────────────────────┤
         ↓   ↓                      ↓
add-database-migration-tools   add-deployment-configurations
       (P1, 2-3 天)                 (P1, 3-4 天)
```

**关键路径**:
1 → 2 → 3 → 5 (总计 18-26 天)

**可并行**:
- 提案 2 和提案 3 (在 gRPC 定义完成后)
- 提案 4 和提案 5 (在核心组件完成后)

---

## 🎯 实施建议

### 选项 A: 立即开始迁移 (适合生产部署需求)

**适用场景**:
- 节点数 ≥ 10
- 需要集中管理和全局视图
- 生产环境部署

**实施步骤**:
1. **Week 1**: 创建提案 1 的详细 design.md 和 tasks.md,开始实施
2. **Week 2**: 完成 gRPC 定义,开始提案 2 (Server) 和提案 3 (Agent)
3. **Week 3-4**: 完成 Server 和 Agent 改造
4. **Week 5**: 实施提案 4 和提案 5,完成部署

**预期产出**: 完整的 Agent-Server 架构,支持大规模部署

---

### 选项 B: 渐进式演进 (推荐,适合当前学习阶段)

**适用场景**:
- 当前是学习项目
- 节点数 < 10
- 希望保持架构简单

**实施步骤**:
1. **先完成当前进行中的功能**
   - ✅ add-label-based-policy (已提案)
   - 🔄 add-flow-collection-api (已提案)

2. **可选: 添加轻量级 gRPC 支持**
   - 仅实施提案 1 的一部分 (Reporter 接口抽象)
   - 不强制使用 Server
   - 保持单体部署

3. **未来需要时再实施完整迁移**
   - 按需创建和实施 5 个完整提案

**预期产出**: 保持简单,为未来迁移预留接口

---

### 选项 C: 先实现前端 Web 界面 (快速演示)

**适用场景**:
- 需要快速展示功能
- PoC 或演示环境
- 暂不需要大规模部署

**实施步骤**:
1. 在当前单体架构基础上添加前端 Web 界面
2. 实现 Dashboard、Policy、Flow 等页面
3. 完成 eBPF → Agent → Web 的完整数据流
4. 未来需要时再迁移 Agent-Server

**预期产出**: 可视化界面,快速演示

---

## 📝 提案状态总结

| 提案 ID | 标题 | 优先级 | 工作量 | proposal.md | design.md | tasks.md | 实施状态 |
|---------|------|--------|--------|-------------|-----------|----------|---------|
| add-grpc-protocol-definitions | gRPC 协议定义 | P0 | 3-5 天 | ✅ | ✅ | ✅ | ⏸️ 待批准 |
| add-server-component | Server 组件 | P0 | 7-10 天 | ✅ | ⏸️ | ⏸️ | ⏸️ 待批准 |
| refactor-agent-for-remote-reporting | Agent 改造 | P0 | 5-7 天 | ✅ | ⏸️ | ⏸️ | ⏸️ 待批准 |
| add-database-migration-tools | 数据迁移 | P1 | 2-3 天 | ✅ | ⏸️ | ⏸️ | ⏸️ 待批准 |
| add-deployment-configurations | 部署配置 | P1 | 3-4 天 | ✅ | ⏸️ | ⏸️ | ⏸️ 待批准 |

**已完成**: 5/5 提案的 proposal.md
**进行中**: 1/5 提案的完整文档 (add-grpc-protocol-definitions)
**待完成**: 4/5 提案的 design.md 和 tasks.md

---

## 🚦 下一步行动

### 立即行动 (如选择迁移)
1. ✅ 审批提案 1: add-grpc-protocol-definitions
2. ⏸️ 补充提案 2-5 的 design.md 和 tasks.md
3. ⏸️ 开始实施提案 1 (创建 proto 文件)

### 延迟行动 (如选择暂缓)
1. ✅ 完成当前 add-label-based-policy 提案
2. ✅ 完成当前 add-flow-collection-api 提案
3. ✅ 实现前端 Web 界面 (可选)
4. ⏸️ 未来需要时再启动 Agent-Server 迁移

---

## 📚 相关文档

- [架构对比分析](./architecture-comparison.md)
- [OpenSpec 工作流程](../openspec/AGENTS.md)
- [项目架构](../openspec/project.md)
- [Flow Collection API 实施总结](./flow-collection-implementation-summary.md)

---

**文档维护**: 随提案实施进度更新
**下次审查**: 提案批准后

# OpenSpec 变更清单

本目录包含所有 OpenSpec 变更提案,包括已归档和进行中的提案。

---

## 🚧 进行中的提案

### 1. add-label-based-policy ⭐
**状态**: 🔄 实施中
**优先级**: P0
**工作量**: 3-4 天

**目标**: 实现基于标签的策略系统

**进度**:
- ✅ proposal.md
- ✅ design.md
- ✅ tasks.md
- 🔄 实施中 (Task 1.2-1.4)

---

### 2. add-flow-collection-api ⭐
**状态**: 🔄 实施中
**优先级**: P0
**工作量**: 2-3 天

**目标**: 实现流量数据收集 API

**进度**:
- ✅ proposal.md
- ✅ design.md
- ✅ tasks.md
- ✅ Phase 1-3 完成 (60%)
- ⏸️ Phase 4-5 待完成

---

## 📋 Agent-Server 架构迁移提案 (待批准)

**总体说明**: 5 个提案用于从单体架构迁移到 Agent-Server 分布式架构

**参考文档**: [docs/agent-server-migration-plan.md](../../docs/agent-server-migration-plan.md)

---

### 3. add-grpc-protocol-definitions
**状态**: ⏸️ 提案阶段
**优先级**: P0 (Agent-Server 迁移基础)
**工作量**: 3-5 天
**依赖**: 无

**目标**: 定义 Agent-Server 通信的 gRPC 接口

**进度**:
- ✅ proposal.md
- ✅ design.md
- ✅ tasks.md
- ⏸️ 待批准

**产出**:
- proto/ (Protocol Buffers 定义)
- src/proto/ (生成的 Go 代码)
- Makefile (代码生成脚本)

---

### 4. add-server-component
**状态**: ⏸️ 提案阶段
**优先级**: P0
**工作量**: 7-10 天
**依赖**: add-grpc-protocol-definitions

**目标**: 实现独立的 Server 组件 (第二个可执行文件)

**进度**:
- ✅ proposal.md
- ⏸️ design.md (待创建)
- ⏸️ tasks.md (待创建)
- ⏸️ 待批准

**产出**:
- src/server/ (Server 代码)
- microsegment-server (可执行文件)
- PostgreSQL + TimescaleDB schema

---

### 5. refactor-agent-for-remote-reporting
**状态**: ⏸️ 提案阶段
**优先级**: P0
**工作量**: 5-7 天
**依赖**: add-grpc-protocol-definitions, add-server-component

**目标**: 改造 Agent 支持远程上报,保持向后兼容

**进度**:
- ✅ proposal.md
- ⏸️ design.md (待创建)
- ⏸️ tasks.md (待创建)
- ⏸️ 待批准

**产出**:
- src/agent/pkg/reporter/ (Reporter 接口和实现)
- 支持 standalone 和 agent-server 两种模式
- 配置文件支持

---

### 6. add-database-migration-tools
**状态**: ⏸️ 提案阶段
**优先级**: P1
**工作量**: 2-3 天
**依赖**: add-server-component

**目标**: SQLite → PostgreSQL 数据迁移工具

**进度**:
- ✅ proposal.md
- ⏸️ design.md (待创建)
- ⏸️ tasks.md (待创建)
- ⏸️ 待批准

**产出**:
- src/tools/migrate/ (迁移工具)

---

### 7. add-deployment-configurations
**状态**: ⏸️ 提案阶段
**优先级**: P1
**工作量**: 3-4 天
**依赖**: add-server-component, refactor-agent-for-remote-reporting

**目标**: 生产级部署配置 (Docker, Kubernetes, Systemd)

**进度**:
- ✅ proposal.md
- ⏸️ design.md (待创建)
- ⏸️ tasks.md (待创建)
- ⏸️ 待批准

**产出**:
- deployments/docker/
- deployments/k8s/
- deployments/systemd/
- deployments/monitoring/

---

## 📁 已归档的提案

已完成并归档的提案存储在 `archive/` 目录:

- [add-control-plane-api](archive/add-control-plane-api/) - 控制平面 API 实现 (已完成)
- [add-config-endpoints](archive/add-config-endpoints/) - 配置端点实现 (已完成)
- 更多归档提案...

查看归档提案详情: [archive/README.md](archive/README.md)

---

## 📊 提案统计

| 状态 | 数量 |
|-----|------|
| 🔄 实施中 | 2 |
| ⏸️ 提案阶段 (待批准) | 5 |
| ✅ 已归档 | 5+ |

---

## 🚀 如何创建新提案

参考 [OpenSpec AGENTS 指南](../AGENTS.md)

**基本步骤**:
1. 创建变更目录: `openspec/changes/{change-id}/`
2. 编写 proposal.md (变更概述)
3. 编写 design.md (详细设计)
4. 编写 tasks.md (实施任务)
5. 创建 spec/ 目录存放规范增量
6. 开始实施

---

## 📚 相关文档

- [OpenSpec 工作流程](../AGENTS.md)
- [项目上下文](../project.md)
- [架构对比分析](../../docs/architecture-comparison.md)
- [Agent-Server 迁移计划](../../docs/agent-server-migration-plan.md)

---

**最后更新**: 2025-11-04
**维护**: 随提案变更更新

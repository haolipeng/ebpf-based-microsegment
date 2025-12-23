[上级索引](../CLAUDE.md) > **storage**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# storage

## 架构定位

PostgreSQL 数据访问层 | 输入: 业务对象（Policy、Flow、Agent、Alert） | 输出: 数据库操作结果、查询数据

## 子目录索引

| 子目录 | 职责 |
|--------|------|
| **migrations** | 数据库迁移脚本 |

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| postgres.go | PostgreSQL 连接管理 | `NewPostgresDB()` |
| bun_models.go | Bun ORM 模型定义 | `BunFlow`, `BunSecurityAlert`, `BunPolicy` |
| policy_storage.go | 策略 CRUD 操作 | `PolicyStorage`, `CreatePolicy()`, `GetAllPolicies()` |
| flow_storage.go | 流事件存储 | `FlowStorage`, `BatchSaveFlowEvents()`, `QueryFlows()` |
| agent_storage.go | Agent 状态存储 | `AgentStorage`, `RegisterAgent()`, `UpdateHeartbeat()` |
| alert_storage.go | 告警存储 | `AlertStorage`, `CreateAlert()`, `ListAlerts()` |
| migrate.go | 数据库迁移执行器 | `RunMigrations()` |

## 核心功能

- **Bun ORM**: 使用 uptrace/bun 进行 PostgreSQL 操作
- **JSONB 支持**: 标签、元数据使用原生 JSONB 类型
- **批量操作**: 批量插入流事件优化性能
- **连接池**: 配置化的连接池管理

## 关键表

| 表名 | 用途 |
|------|------|
| policies | 策略规则 |
| flows | 流事件记录 |
| agents | Agent 注册和状态 |
| security_alerts | 安全告警 |


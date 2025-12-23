[上级索引](../CLAUDE.md) > **api**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# api

## 架构定位

HTTP RESTful API 层 | 输入: HTTP 请求（Gin 框架）、Storage 层依赖 | 输出: JSON 响应、标准化错误格式

## 子目录索引

| 子目录 | 职责 |
|--------|------|
| **handlers** | 各资源的 HTTP 处理器 |
| **middleware** | 错误处理、日志等中间件 |

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| errors.go | 标准化错误响应格式 | `RespondWithError()`, `RespondWithSuccess()`, `ErrorResponse` |
| handlers/policy.go | 策略 CRUD API | `PolicyHandler`, `ListPolicies()`, `CreatePolicy()` |
| handlers/flow.go | 流事件查询 API | `FlowHandler`, `ListFlows()`, `GetFlowStats()` |
| handlers/agent.go | Agent 管理 API | `AgentHandler`, `ListAgents()`, `GetAgentStatus()` |
| handlers/topology.go | 网络拓扑 API | `TopologyHandler`, `GetTopology()` |
| handlers/alert.go | 告警管理 API | `AlertHandler`, `ListAlerts()` |
| handlers/aggregator.go | 聚合数据 API | `AggregatorHandler`, `GetDependencies()` |
| middleware/error_handler.go | 全局错误处理中间件 | `ErrorHandler()` |

## 核心功能

- **RESTful 端点**: 策略、流、Agent、拓扑、告警资源的 CRUD 操作
- **标准响应格式**: 统一 success/error 结构
- **参数验证**: Gin binding 验证
- **错误处理**: 全局错误捕获和标准化


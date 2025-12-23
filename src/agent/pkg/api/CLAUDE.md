[上级索引](../CLAUDE.md) > **api**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# api

## 架构定位

RESTful HTTP API 服务器 | 输入: HTTP 请求（策略 CRUD、流查询、配置更新、健康检查） | 输出: JSON 响应（策略数据、流事件、统计信息、健康状态）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| server.go | API 服务器主体，初始化 Gin 引擎和 HTTP 服务器 | `NewAPIServer()`, `Start()`, `Stop()` |
| router.go | 路由注册，定义所有 API 端点和中间件 | `SetupRoutes()` |
| middleware.go | HTTP 中间件（日志、CORS、认证等） | `LoggingMiddleware()`, `CORSMiddleware()` |
| config.go | API 服务器配置（监听地址、超时等） | `DefaultConfig()` |
| handlers/policy.go | 策略管理接口（CRUD） | `ListPolicies()`, `CreatePolicy()`, `DeletePolicy()` |
| handlers/policy_rule.go | 策略规则详细操作 | `GetPolicyRule()`, `UpdatePolicyRule()` |
| handlers/workload.go | 工作负载查询接口 | `ListWorkloads()`, `GetWorkload()` |
| handlers/group.go | 分组管理接口 | `ListGroups()`, `CreateGroup()` |
| handlers/flow.go | 流事件查询接口 | `GetFlows()`, `GetFlowByID()` |
| handlers/statistics.go | 统计数据查询（会话、策略命中等） | `GetStatistics()` |
| handlers/config.go | 配置管理接口 | `GetConfig()`, `UpdateConfig()` |
| handlers/health.go | 健康检查和系统状态 | `HealthCheck()`, `ReadinessCheck()` |
| handlers/lifecycle.go | 生命周期管理（优雅关闭等） | `Shutdown()` |
| models/policy.go | 策略数据模型 | `Policy`, `PolicyRule` |
| models/workload.go | 工作负载数据模型 | `Workload` |
| models/group.go | 分组数据模型 | `Group` |
| models/flow.go | 流事件数据模型 | `Flow`, `FlowEvent` |
| models/statistics.go | 统计数据模型 | `Statistics`, `PolicyStats` |
| models/config.go | 配置数据模型 | `APIConfig` |
| models/health.go | 健康状态模型 | `HealthStatus` |
| models/error.go | 错误响应模型 | `ErrorResponse` |

## API 端点总览

**基础路径**: `/api/v1`

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/policies` | GET/POST | 策略列表/创建 |
| `/policies/:id` | GET/PUT/DELETE | 策略详情/更新/删除 |
| `/workloads` | GET | 工作负载列表 |
| `/groups` | GET/POST | 分组列表/创建 |
| `/flows` | GET | 流事件查询 |
| `/statistics` | GET | 统计数据 |
| `/config` | GET/PUT | 配置查询/更新 |

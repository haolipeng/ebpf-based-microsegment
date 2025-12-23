[上级索引](../CLAUDE.md) > **client**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# client

## 架构定位

提供 gRPC 客户端，用于 Agent 与 Server 通信（心跳、策略同步、流上报）。
**输入**: 本地流事件、Agent 状态、策略变更订阅
**输出**: gRPC 请求到 Server（FlowService、PolicyService、AgentService）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| agent_client.go | gRPC 客户端封装，管理连接和服务调用 | `NewAgentClient()`, `Connect()`, `StartHeartbeat()`, `SyncPolicies()`, `ReportFlow()` |

## 核心功能

- **连接管理**: 建立和维护到 Server 的 gRPC 连接
- **心跳机制**: 定期发送 Agent 状态和指标
- **策略同步**: 订阅 Server 策略变更并应用到本地
- **流上报**: 批量上报流事件到 Server
- **重连机制**: 自动处理连接断开和重连

## 依赖服务

- **AgentService**: Agent 注册和心跳
- **PolicyService**: 策略订阅和同步
- **FlowService**: 流事件上报

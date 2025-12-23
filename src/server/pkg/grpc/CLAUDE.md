[上级索引](../CLAUDE.md) > **grpc**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# grpc

## 架构定位

gRPC 服务实现 | 输入: Agent 发起的 gRPC 请求（流事件、策略订阅、心跳） | 输出: 策略配置、确认响应、状态更新

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| flow_service.go | 流事件接收和处理 | `FlowServiceServer`, `ReportFlowEvents()` |
| policy_service.go | 策略分发和订阅 | `PolicyServiceServer`, `GetPolicies()`, `SubscribePolicies()` |
| agent_service.go | Agent 注册和心跳 | `AgentServiceServer`, `RegisterAgent()`, `Heartbeat()` |

## 核心功能

- **流式接收**: Agent 流事件的流式上报处理
- **批量入库**: 按 maxFlowBatchSize 批量写入数据库
- **策略推送**: 策略变更的实时推送
- **拓扑更新**: 流事件触发实时拓扑构建

## 协议定义

- 详见 [api/proto/flow](../../../../api/proto/flow/CLAUDE.md)
- 详见 [api/proto/policy](../../../../api/proto/policy/CLAUDE.md)
- 详见 [api/proto/agent](../../../../api/proto/agent/CLAUDE.md)


[上级索引](../CLAUDE.md) > **pubsub**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# pubsub

## 架构定位

策略发布订阅 | 输入: 策略更新事件 | 输出: 向已订阅 Agent 推送策略变更

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| policy_pubsub.go | 策略更新的发布/订阅机制 | `PolicyPubSub`, `Subscribe()`, `Unsubscribe()`, `Publish()` |

## 核心功能

- **Agent 订阅**: Agent 连接时注册订阅通道
- **非阻塞推送**: 通道满时跳过该 Agent（避免慢 Agent 阻塞）
- **自动替换**: 重复订阅时关闭旧通道
- **线程安全**: RWMutex 保护的订阅者列表

## 工作流程

1. Agent 通过 gRPC 调用 SubscribePolicies
2. gRPC 服务调用 pubsub.Subscribe() 获取更新通道
3. 策略变更时调用 pubsub.Publish() 广播
4. gRPC 服务从通道读取并推送给 Agent

## 应用场景

- **策略实时同步**: 策略变更后立即推送到所有 Agent
- **多 Agent 广播**: 一对多的策略分发


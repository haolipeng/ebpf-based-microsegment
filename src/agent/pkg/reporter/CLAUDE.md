[上级索引](../CLAUDE.md) > **reporter**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# reporter

## 架构定位

流事件上报器 | 输入: 本地流事件、Agent 指标 | 输出: gRPC 流上报到 Server（FlowService.ReportFlow）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| grpc_reporter.go | gRPC 上报客户端，批量发送流事件 | `NewGRPCReporter()`, `Start()`, `Report()`, `ReportBatch()` |
| reporter.go | Reporter 接口定义 | `Reporter` interface |

## 核心功能

- **批量上报**: 聚合多个流事件后批量发送，减少 gRPC 调用
- **异步队列**: 使用 channel 缓冲流事件，避免阻塞数据平面
- **重试机制**: 上报失败时自动重试（指数退避）
- **指标统计**: 记录上报成功/失败/重试次数

## 配置参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| BatchSize | 批量大小 | 100 |
| MaxRetries | 最大重试次数 | 3 |
| RetryBaseDelay | 重试基础延迟 | 1s |
| RetryMaxDelay | 最大重试延迟 | 30s |
| QueueSize | 队列大小 | 1000 |

## 应用场景

- **Agent-Server 模式**: Agent 上报流到 Server 进行集中分析
- **多 Agent 聚合**: Server 聚合多个 Agent 的流数据
- **审计存储**: Server 持久化流事件到数据库

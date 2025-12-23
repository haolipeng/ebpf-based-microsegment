[上级索引](../CLAUDE.md) > **flow**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# flow

## 架构定位

流事件收集器 | 输入: eBPF Ring Buffer 流事件（新连接、拒绝、超时） | 输出: 聚合后的流记录、本地存储、gRPC 上报到 Server

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| collector.go | 从 Ring Buffer 收集流事件 | `NewCollector()`, `Start()`, `Stop()`, `ReadEvents()` |
| aggregator.go | 流事件聚合（相同五元组合并统计） | `NewAggregator()`, `Aggregate()`, `FlushAggregated()` |
| storage.go | 流数据持久化接口（SQLite/PostgreSQL） | `SaveFlow()`, `UpdateFlow()`, `QueryFlows()` |
| lifecycle.go | 流生命周期管理（创建、更新、关闭） | `OnFlowCreated()`, `OnFlowClosed()` |
| types.go | 流数据类型定义 | `Flow`, `FlowEvent`, `FlowKey` |

## 核心功能

- **实时收集**: 从 eBPF Ring Buffer 读取流事件
- **智能聚合**: 合并相同五元组的流，减少数据量
- **标签注入**: 从 WorkloadManager 查询 IP 对应的标签
- **优雅关闭**: 确保所有流事件被处理后再退出
- **批量上报**: 聚合多个流事件后批量发送到 Server

## 流事件类型

- **NEW_CONNECTION**: 新建连接（首次包）
- **CONNECTION_REJECTED**: 策略拒绝
- **CONNECTION_CLOSED**: 连接关闭（TCP FIN/RST）
- **CONNECTION_TIMEOUT**: 会话超时清理

## 性能优化

- Ring Buffer 批量读取，减少系统调用
- Per-CPU 统计，无锁数据收集
- 聚合窗口可配置（默认 10 秒）

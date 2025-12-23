[上级索引](../CLAUDE.md) > **workload**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# workload

## 架构定位

工作负载管理器 | 输入: 工作负载元数据（ID、名称、标签、IP 地址） | 输出: 工作负载存储、标签索引、IP-标签映射

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| manager.go | 工作负载管理器，提供高层操作 | `NewManager()`, `CreateWorkload()`, `UpdateWorkload()`, `DeleteWorkload()`, `GetLabelsByIP()` |
| storage.go | 工作负载持久化接口（SQLite） | `CreateWorkload()`, `GetWorkload()`, `ListWorkloads()`, `DeleteWorkload()` |
| types.go | 工作负载数据类型定义 | `Workload`, `WorkloadRuntime` |

## 核心功能

- **工作负载 CRUD**: 创建、查询、更新、删除工作负载
- **标签管理**: 维护工作负载标签，支持标签更新
- **IP 索引**: 支持通过 IP 地址查询工作负载和标签
- **运行时标识**: 区分 K8s、Docker、Containerd 来源
- **验证**: 确保工作负载 ID、IP 唯一性

## 索引结构

- **By ID**: 快速查找特定工作负载
- **By IP**: 流事件关联工作负载（查询 IP 的标签）
- **By Labels**: 分组成员解析（查询匹配标签的工作负载）
- **By Namespace**: K8s 命名空间过滤

## 应用场景

- **策略编译**: 将基于标签的策略编译为基于 IP 的策略
- **流事件注解**: 为流事件添加源/目标工作负载信息
- **分组成员**: 解析分组包含的工作负载
- **审计**: 记录哪个工作负载发起了连接

[上级索引](../CLAUDE.md) > **topology**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# topology

## 架构定位

网络拓扑构建 | 输入: 流事件数据 | 输出: 节点和边的拓扑图、节点属性、连接统计

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| manager.go | 拓扑图管理器 | `Manager`, `NewManager()`, `UpdateNode()`, `UpdateEdge()`, `GetTopology()` |
| builder.go | 流事件到拓扑的转换 | `Builder`, `ProcessFlowEvents()` |
| types.go | 拓扑节点和边的类型定义 | `NodeAttr`, `EdgeAttr`, `Topology` |

## 核心功能

- **实时拓扑**: 从流事件实时构建网络拓扑
- **节点属性**: 名称、标签、最后活跃时间
- **边属性**: 流量统计、协议分布
- **过期清理**: 可配置的 TTL 清理机制

## 工作流程

1. FlowService 接收 Agent 上报的流事件
2. 调用 Builder.ProcessFlowEvents() 处理
3. Builder 更新 Manager 中的节点和边
4. TopologyHandler 从 Manager 查询拓扑数据

## 与 graph 包的关系

- `topology` 是业务层，管理拓扑语义
- `graph` 是基础设施层，提供图数据结构


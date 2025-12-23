[上级索引](../CLAUDE.md) > **aggregator**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# aggregator

## 架构定位

流数据聚合器 | 输入: 原始流事件数据（PostgreSQL flows 表） | 输出: 聚合后的依赖关系、统计数据、时间序列分析结果

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| flow_aggregator.go | 流数据聚合和分析逻辑 | `NewFlowAggregator()`, `GetDependencies()`, `GetTimeSeries()` |
| types.go | 聚合查询和结果的类型定义 | `AggregationQuery`, `FlowDependency`, `TimeSeries` |

## 核心功能

- **依赖关系分析**: 基于标签分组的源-目标依赖提取
- **时间序列聚合**: 按时间窗口聚合流量统计
- **SQL 查询构建**: 动态构建聚合查询语句
- **协议统计**: 多协议流量汇总分析

## 应用场景

- **网络拓扑可视化**: 提供节点间依赖关系
- **流量趋势分析**: 历史流量变化追踪
- **容量规划**: 基于流量模式的资源预测


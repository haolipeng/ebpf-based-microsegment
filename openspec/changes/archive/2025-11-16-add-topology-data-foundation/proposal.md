# 提案：Topology Data Foundation

## 为什么

当前的网络流量数据以扁平记录形式存储，无法直接可视化为拓扑图。运维团队需要通过查看主机（基于 IP）或服务（基于标签）之间的关系来理解网络通信模式。

**当前痛点**：
- Flow 数据缺少图数据模型（节点、边）
- 没有将 Flow 聚合为拓扑结构的逻辑
- 同时缺失 IP 视图和标签视图的支持
- 缺少实时拓扑刷新机制

**业务需求**：
- 将 Flow 数组聚合为节点（主机/服务）和边（连接）
- 支持 IP 视图（按源/目的 IP）与标签视图（按服务标签）
- 计算节点与边的流量指标
- 支持通过 WebSocket 的实时拓扑刷新

## 变更内容

**TypeScript 类型定义**（`web/src/types/topology.ts`）：
- `TopologyNode` —— 表示网络节点（IP 或服务）
- `TopologyEdge` —— 表示节点之间的连接
- `TopologyData` —— 完整拓扑图结构
- `TopologyViewMode` —— 视图模式枚举（IP | LABEL）
- `TopologyFilters` —— 拓扑查询的筛选条件

**数据聚合工具**（`web/src/utils/topologyUtils.ts`）：
- `aggregateFlowsToTopology()` —— 将 Flow 数组转换为拓扑数据
- `calculateNodeSize()` —— 根据流量计算节点大小
- `calculateEdgeWidth()` —— 根据流量计算边宽
- `mergeTopologyUpdate()` —— 将实时 Flow 数据合并进现有拓扑
- `getNodeLabel()` —— 获取节点展示标签

**数据获取 Hook**（`web/src/hooks/useTopology.ts`）：
- `useTopology()` —— 自定义 Hook，用于获取和管理拓扑数据
- 与 `useFlows`、`useFlowStream` 集成
- 对实时更新做 500ms 防抖
- 返回拓扑数据、加载状态和错误信息

## 影响

**新增文件**：
- `web/src/types/topology.ts`（类型定义）
- `web/src/utils/topologyUtils.ts`（数据转换逻辑）
- `web/src/hooks/useTopology.ts`（React Hook 集成）

**依赖**：
- 复用现有 `Flow` 类型、`useFlows`、`useFlowStream`
- 不新增 npm 依赖

**测试要求**：
- Flow 聚合逻辑单测（IP/标签两种模式）
- 计算函数单测（节点大小、边宽）
- 实时合并逻辑单测
- 覆盖率目标 ≥ 80%

**Breaking Changes**：无（纯新增）

**影响的能力**：
- `topology-visualization`（新增能力）


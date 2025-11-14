# Topology Visualization 能力 —— 数据基础层

## ADDED Requirements

### Requirement: 拓扑数据模型
系统 MUST 提供用于拓扑图节点、边及聚合统计的 TypeScript 接口。

#### Scenario: IP 视图的节点表示
- **WHEN** 以 IP 视图表示网络主机
- **THEN** `TopologyNode` MUST 包含 id、label、type='IP' 以及流量指标（flowCount、packetCount、byteCount、activeFlows）

#### Scenario: 服务视图的节点表示
- **WHEN** 以标签视图表示服务
- **THEN** `TopologyNode` MUST 包含 id、label、type='SERVICE'、流量指标以及标签元数据

#### Scenario: 节点之间的边表示
- **WHEN** 表示两个节点之间的连接
- **THEN** `TopologyEdge` MUST 包含 source、target、流量指标、协议列表以及方向（INGRESS/EGRESS/BIDIRECTIONAL）

#### Scenario: 完整拓扑数据结构
- **WHEN** 提供完整拓扑数据
- **THEN** `TopologyData` MUST 包含 nodes 列表、edges 列表以及 stats（totalNodes、totalEdges、totalFlows）

### Requirement: Flow 聚合为拓扑
系统 MUST 将扁平的 Flow 记录聚合为同时支持 IP 与标签视图的拓扑结构。

#### Scenario: 按 IP 聚合 Flow
- **WHEN** 调用 `aggregateFlowsToTopology()` 且 viewMode='IP'
- **THEN** Flow MUST 按 sourceIp/destIp 分组生成节点
- **AND** IP 之间的连接 MUST 生成带聚合指标的边

#### Scenario: 按服务标签聚合 Flow
- **WHEN** 调用 `aggregateFlowsToTopology()` 且 viewMode='LABEL'
- **THEN** Flow MUST 按 sourceLabels/destLabels 分组合并为服务节点
- **AND** 系统 MUST 优先使用 `app` 标签，否则使用首个 key-value 作为节点标识

#### Scenario: 标签视图下缺少标签的 Flow
- **WHEN** 标签视图聚合时发现 Flow 无标签
- **THEN** 该 Flow MUST 被跳过（不得创建无标签节点）

#### Scenario: 节点数量上限
- **WHEN** `aggregateFlowsToTopology()` 传入 maxNodes
- **THEN** MUST 仅保留流量排名前 N 的节点
- **AND** 边 MUST 过滤为只连接被保留节点

#### Scenario: 双向连接计算
- **WHEN** 两个节点间存在双向 Flow
- **THEN** MUST 仅创建一个 direction='BIDIRECTIONAL' 的边
- **AND** 其指标 MUST 合并两个方向的流量

### Requirement: 视觉指标计算
系统 MUST 提供基于对数缩放的节点大小与边宽度计算函数。

#### Scenario: 低流量的节点大小
- **WHEN** `calculateNodeSize()` 输入 flowCount ≤ 1
- **THEN** MUST 返回最小尺寸 20px

#### Scenario: 高流量的节点大小
- **WHEN** `calculateNodeSize()` 输入较大的 flowCount
- **THEN** MUST 返回 20-80px 的对数缩放结果

#### Scenario: 极低流量的边宽
- **WHEN** `calculateEdgeWidth()` 输入 byteCount < 1KB
- **THEN** MUST 返回最小边宽 1px

#### Scenario: 大流量的边宽
- **WHEN** `calculateEdgeWidth()` 输入较大的 byteCount
- **THEN** MUST 依据 KB 单位返回 1-10px 的对数缩放结果

### Requirement: 实时拓扑更新
系统 MUST 支持将实时 Flow 更新合并到既有拓扑数据。

#### Scenario: 新 Flow 更新既有节点
- **WHEN** `mergeTopologyUpdate()` 收到针对已存在节点的新 Flow
- **THEN** MUST 通过自增方式更新节点指标
- **AND** MUST 更新或创建聚合指标的边

#### Scenario: 新节点的 Flow
- **WHEN** `mergeTopologyUpdate()` 收到包含全新 source 或 target 的 Flow
- **THEN** MUST 创建带初始化指标的新节点
- **AND** MUST 创建连接这些节点的新边

#### Scenario: 更新边的协议列表
- **WHEN** 合并带有新协议的 Flow
- **THEN** MUST 将该协议加入 edge.metrics.protocols

### Requirement: React Hook 集成
系统 MUST 提供 `useTopology()` Hook 用于获取并实时管理拓扑数据。

#### Scenario: 带筛选条件拉取拓扑数据
- **WHEN** 使用带 viewMode、timeRange、maxNodes 的 `useTopology()`
- **THEN** MUST 通过 `useFlows()` 获取 Flow
- **AND** MUST 使用 `aggregateFlowsToTopology()` 聚合
- **AND** MUST 返回含 loading、error 状态的 `TopologyData`

#### Scenario: 启用实时更新
- **WHEN** `useTopology()` 入参 `enableRealtime=true`
- **THEN** MUST 通过 `useFlowStream()` 建立实时连接
- **AND** MUST 使用 `mergeTopologyUpdate()`（含 500ms 防抖）合并新 Flow
- **AND** MUST 返回 WebSocket `isConnected` 状态

#### Scenario: 防抖快速更新
- **WHEN** 500ms 内收到多条 Flow 更新
- **THEN** MUST 批量合并并在防抖结束后重算拓扑一次
- **AND** MUST 避免过度重渲染

#### Scenario: 组件卸载清理
- **WHEN** 使用 `useTopology()` 的组件卸载
- **THEN** MUST 清理防抖定时器
- **AND** MUST 由 `useFlowStream()` 关闭 WebSocket

### Requirement: 方向处理
系统 MUST 正确处理 Flow 方向枚举并映射到拓扑边。

#### Scenario: 将 UNKNOWN 映射为 EGRESS
- **WHEN** Flow 方向为 'UNKNOWN' 且需创建边
- **THEN** MUST 将边方向设为 'EGRESS'
- **AND** MUST 保证 `TopologyEdge` 类型中不出现 'UNKNOWN'


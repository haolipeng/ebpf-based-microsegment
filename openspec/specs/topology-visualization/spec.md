# topology-visualization Specification

## Purpose
TBD - created by archiving change add-topology-data-foundation. Update Purpose after archive.
## Requirements
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

### Requirement: 交互式图形渲染
系统 MUST 使用 ECharts 将拓扑数据渲染为可交互的力导向图，并支持 100+ 节点。

#### Scenario: 使用力导向布局渲染拓扑
- **WHEN** TopologyGraph 组件收到 `TopologyData`
- **THEN** MUST 使用 ECharts Graph 系列绘制节点和边
- **AND** MUST 套用可配置的力导向参数（repulsion=300、gravity=0.1）

#### Scenario: 处理空拓扑数据
- **WHEN** TopologyGraph 收到空或 null 数据
- **THEN** MUST 显示 Ant Design Empty，提示 “No topology data”

#### Scenario: 显示加载状态
- **WHEN** `loading` 为 true
- **THEN** MUST 显示 Ant Design Spin，提示 “Loading topology data...”

#### Scenario: 支持灵活高度
- **WHEN** `height` 属性为数字或字符串
- **THEN** 图表容器 MUST 根据值调整高度
- **AND** 未传入时默认 600px

### Requirement: 节点视觉编码
系统 MUST 通过尺寸、颜色与样式编码节点流量指标。

#### Scenario: 按流量设置节点尺寸
- **WHEN** 渲染节点
- **THEN** 节点 `symbolSize` MUST 使用 `calculateNodeSize(flowCount)`
- **AND** MUST 保持在 20-80px 范围

#### Scenario: 按类型着色节点
- **WHEN** 渲染 IP 节点（type='IP'）
- **THEN** MUST 使用蓝色 (#5470c6)
- **WHEN** 渲染 Service 节点（type='SERVICE'）
- **THEN** MUST 使用绿色 (#91cc75)

#### Scenario: 使用透明度标识活跃流
- **WHEN** 节点 activeFlows > 0
- **THEN** MUST 设 opacity=1.0
- **WHEN** activeFlows = 0
- **THEN** MUST 设 opacity=0.6

#### Scenario: 节点描边与阴影
- **WHEN** 渲染任意节点
- **THEN** MUST 添加 2px 白色描边与阴影（shadowBlur=10，shadowColor=rgba(0,0,0,0.3)）

### Requirement: 边视觉编码
系统 MUST 依据协议与流量，使用宽度、颜色、样式对边进行编码。

#### Scenario: 按流量设置边宽
- **WHEN** 渲染边
- **THEN** MUST 使用 `calculateEdgeWidth(byteCount)` 设置 `lineStyle.width`
- **AND** 宽度 MUST 位于 1-10px

#### Scenario: 按协议着色边
- **WHEN** 协议包含 TCP
- **THEN** MUST 用蓝色 (#5470c6)
- **WHEN** 包含 UDP
- **THEN** MUST 用绿色 (#91cc75)
- **WHEN** 包含 ICMP
- **THEN** MUST 用橙色 (#fac858)
- **WHEN** 其他协议
- **THEN** MUST 用灰色 (#999)

#### Scenario: 边曲率与透明度
- **WHEN** 渲染边
- **THEN** `curveness` MUST 设为 0.1
- **AND** `opacity` MUST 设为 0.6

### Requirement: 节点交互行为
系统 MUST 支持节点的点击、悬停、拖拽、聚焦等交互。

#### Scenario: 点击节点触发回调
- **WHEN** 用户点击节点
- **THEN** MUST 调用 `onNodeClick` 并传入完整 `TopologyNode`

#### Scenario: 悬停节点显示 tooltip
- **WHEN** 用户悬停节点
- **THEN** tooltip MUST 展示类型、流数、包量、流量、标签等信息

#### Scenario: 拖拽节点重新定位
- **WHEN** 用户拖拽节点
- **THEN** 节点 MUST 跟随移动，力导向布局实时调整

#### Scenario: 双击节点聚焦
- **WHEN** 用户双击节点
- **THEN** 系统 MUST 高亮该节点及其相邻连接

#### Scenario: 悬停时强调相邻连接
- **WHEN** 用户悬停节点
- **THEN** 系统 MUST 使用 focus='adjacency' 高亮节点及边
- **AND** 边宽 MUST 增加至 5px 并添加阴影

### Requirement: 画布交互
系统 MUST 支持画布层面的导航交互。

#### Scenario: 滚轮缩放
- **WHEN** 用户在图上滚动鼠标
- **THEN** MUST 保持中心点进行缩放

#### Scenario: 拖拽背景平移
- **WHEN** 用户拖拽空白处
- **THEN** MUST 平移整个拓扑

#### Scenario: 自由漫游
- **WHEN** roam 启用
- **THEN** 用户 MUST 可自由缩放与平移

### Requirement: Tooltip 信息展示
系统 MUST 在悬停节点和边时显示详尽 tooltip。

#### Scenario: 节点 tooltip 内容
- **WHEN** 展示节点 tooltip
- **THEN** MUST 包含图标（🖥️/⚙️）、label、类型、流数、活跃流、包数、格式化流量、标签信息

#### Scenario: 边 tooltip 内容
- **WHEN** 展示边 tooltip
- **THEN** MUST 包含 source→target、方向（Inbound/Outbound/Bidirectional）、流数、包数、格式化流量、协议列表

#### Scenario: 指标格式化
- **WHEN** 在 tooltip 中显示数值
- **THEN** 数字 MUST 使用千分位
- **AND** 字节 MUST 使用单位（如 1.2 MB、340 KB）

### Requirement: 视觉图例
系统 MUST 提供图例组件说明视觉编码与交互。

#### Scenario: 节点类型图例
- **WHEN** 展示图例
- **THEN** MUST 用蓝/绿圆点区分 IP 与 Service
- **AND** 文案 MUST 随 viewMode 变化

#### Scenario: 节点尺寸图例
- **WHEN** 展示图例
- **THEN** MUST 显示 20/40/60px 三种尺寸，对应 Low/Medium/High Traffic

#### Scenario: 协议颜色图例
- **WHEN** 展示图例
- **THEN** MUST 显示 TCP/UDP/ICMP/Other 的颜色示例

#### Scenario: 边宽图例
- **WHEN** 展示图例
- **THEN** MUST 显示 1/3/6px 线宽，对应 Small/Medium/Large

#### Scenario: 展示交互提示
- **WHEN** 展示图例
- **THEN** MUST 包含悬停/点击/拖拽/滚轮/双击等提示

#### Scenario: 图例位置
- **WHEN** 渲染图例
- **THEN** MUST 绝对定位在右上角偏移 20px
- **AND** MUST 设 z-index=10
- **AND** MUST 使用带阴影的卡片样式

### Requirement: 性能优化
系统 MUST 在大规模拓扑下保持流畅。

#### Scenario: 100+ 节点渲染时延
- **WHEN** 渲染 100+ 节点
- **THEN** 首次渲染 MUST 在 2 秒内完成

#### Scenario: 交互期间帧率
- **WHEN** 用户拖拽/缩放/平移
- **THEN** 帧率 MUST ≥ 30FPS

#### Scenario: 使用 canvas 渲染器
- **WHEN** 初始化 ECharts
- **THEN** renderer MUST 设为 'canvas'

#### Scenario: 启用 lazyUpdate
- **WHEN** 配置 ReactECharts
- **THEN** `lazyUpdate` MUST 设为 true

#### Scenario: 全量数据替换
- **WHEN** 拓扑数据整体变化
- **THEN** `notMerge` MUST 设为 true 以高效重建

### Requirement: 响应式设计
系统 MUST 针对不同屏幕尺寸调整可视化布局。

#### Scenario: 移动端图高度
- **WHEN** 视口 < 768px
- **THEN** 图高度 MUST 降至 400px

#### Scenario: 小屏幕下的图例
- **WHEN** 视口 < 768px
- **THEN** 图例宽度 MUST 收缩以适配
- **AND** MUST 保持可读

#### Scenario: 触屏交互
- **WHEN** 用户在触屏设备上操作
- **THEN** MUST 支持拖动、双指缩放、点按节点

### Requirement: 拓扑页面路由
系统 MUST 提供独立的 `/topology` 路由供用户访问拓扑功能。

#### Scenario: 导航至拓扑页面
- **WHEN** 用户访问 `/topology`
- **THEN** MUST 渲染 Topology 页面，并显示标题 "Network Topology" 及描述

#### Scenario: 通过侧边菜单进入拓扑
- **WHEN** 用户点击导航菜单中的 "Topology"
- **THEN** MUST 跳转至 `/topology` 且高亮该菜单项

### Requirement: 视图模式控制
系统 MUST 提供视图模式切换控件以在 IP 与 Label 视图间切换。

#### Scenario: 切换至 IP 视图
- **WHEN** 用户选择 "IP View"
- **THEN** `viewMode` MUST 置为 'IP' 且清空选中节点

#### Scenario: 切换至 Service 视图
- **WHEN** 用户选择 "Service View"
- **THEN** `viewMode` MUST 置为 'LABEL' 且清空选中节点

### Requirement: 时间范围筛选
系统 MUST 提供默认 7 天范围的时间筛选控件。

#### Scenario: 选择自定义时间范围
- **WHEN** 用户在 RangePicker 中设定起止时间
- **THEN** filters.startTime/endTime MUST 更新并触发数据刷新

### Requirement: Flow 筛选控件
系统 MUST 提供协议、状态、动作、最大节点等筛选能力。

#### Scenario: 协议筛选
- **WHEN** 选择协议（TCP/UDP/ICMP/All）
- **THEN** 仅展示匹配协议的 Flow

#### Scenario: 状态筛选
- **WHEN** 选择状态（Active/Closed/Timeout/All）
- **THEN** 仅展示匹配状态的 Flow

#### Scenario: 动作筛选
- **WHEN** 选择动作（Allow/Deny/Log/All）
- **THEN** 仅展示匹配动作的 Flow

#### Scenario: 最大节点控制
- **WHEN** 调整 maxNodes (10-200)
- **THEN** 拓扑 MUST 仅保留前 N 个节点并过滤边

### Requirement: 实时更新开关
系统 MUST 提供实时开关以启用/关闭 WebSocket 更新。

#### Scenario: 启用实时模式
- **WHEN** 用户开启实时开关
- **THEN** MUST 连接 Flow 流并展示成功 Alert

#### Scenario: 连接断开提示
- **WHEN** 实时模式开启但 WebSocket 断开
- **THEN** MUST 展示警告 Alert

### Requirement: 手动刷新与重置
系统 MUST 提供刷新与重置按钮。

#### Scenario: 手动刷新
- **WHEN** 点击 "Refresh"
- **THEN** MUST 调用 `refetch()` 重新获取数据

#### Scenario: 重置筛选
- **WHEN** 点击 "Reset"
- **THEN** MUST 恢复默认筛选条件

### Requirement: 节点详情面板
系统 MUST 支持点击节点查看详情。

#### Scenario: 打开节点详情
- **WHEN** 用户点击节点
- **THEN** MUST 打开右侧 Drawer 显示节点信息

#### Scenario: 展示基础信息
- **WHEN** Drawer 打开
- **THEN** MUST 展示节点 ID、类型、标签

#### Scenario: 展示流量统计
- **WHEN** Drawer 打开
- **THEN** MUST 展示总流数、活跃流、总包数、总字节数（格式化）

#### Scenario: 展示相关 Flow
- **WHEN** Drawer 打开
- **THEN** MUST 在表格中展示相关 Flow 并支持分页

#### Scenario: 展示连接统计
- **WHEN** Drawer 打开
- **THEN** MUST 展示入站/出站连接数量

### Requirement: 页面布局与响应式
系统 MUST 提供自适应布局以覆盖不同屏幕。

#### Scenario: 桌面布局
- **WHEN** 视口 ≥ 768px
- **THEN** MUST 使用 24px padding、横向控件、图高度 `calc(100vh - 64px)`

#### Scenario: 移动布局
- **WHEN** 视口 < 768px
- **THEN** MUST 控件纵向排列，图高度 400px，图例保持可读

#### Scenario: 移动端详情面板
- **WHEN** 移动端打开 Drawer
- **THEN** MUST 占满宽度并允许表格横向滚动

### Requirement: 错误与空状态提示
系统 MUST 提供明确的加载、错误、空状态提示。

#### Scenario: 加载状态
- **WHEN** 数据加载中
- **THEN** TopologyGraph MUST 显示 “Loading topology data...”

#### Scenario: 错误状态
- **WHEN** 获取数据失败
- **THEN** MUST 显示错误 Alert（"Failed to load data"）

#### Scenario: 空数据
- **WHEN** 当前筛选无任何 Flow
- **THEN** MUST 显示 "No topology data" 空状态

### Requirement: 国际化准备
系统 MUST 全局使用英文文案并统一格式化。

#### Scenario: 文案统一
- **WHEN** 渲染任何文本
- **THEN** MUST 使用英文术语

#### Scenario: 指标格式化
- **WHEN** 展示数值与字节
- **THEN** MUST 使用千分位与标准单位（KB/MB/GB）


# Topology Visualization 能力 —— 可视化核心

## ADDED Requirements

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


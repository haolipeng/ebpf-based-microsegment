# 提案：Topology Visualization Core

## 为什么

在已有拓扑数据结构的基础上，需要构建核心可视化组件，用以渲染可交互的网络拓扑图。运维团队需要通过直观的视觉编码（节点大小、边宽、颜色）和丰富的交互（缩放、平移、拖拽、悬停、点击）来理解网络关系。

**当前痛点**：
- 缺少拓扑图可视化组件
- 需要支撑 100+ 节点的渲染性能
- 需要丰富的交互体验（缩放、平移、拖拽节点、查看详情）
- 需要通过视觉编码快速呈现流量指标

**业务需求**：
- 使用 ECharts 将拓扑渲染为交互式力导向图
- 使用节点大小、边宽、颜色编码流量指标
- 支持缩放、平移、拖拽、点击、悬停等交互
- 展示图例解释视觉编码规则
- 在 100+ 节点场景下保持流畅

## 变更内容

**Topology Graph 组件**（`web/src/components/topology/TopologyGraph.tsx`）：
- 基于 echarts-for-react 的 React 组件
- 可配置物理参数的力导向布局
- 节点视觉映射（大小、颜色、图标随类型与指标变化）
- 边视觉映射（宽度、颜色、方向随协议与流量变化）
- 点击、双击、悬停等事件处理
- 空数据与加载状态处理

**ECharts 配置**（`web/src/components/topology/topologyConfig.ts`）：
- `getTopologyChartOption()` —— 生成完整图表配置
- `getNodeStyle()` —— 计算节点视觉样式
- `getEdgeStyle()` —— 计算边线样式
- `formatNodeTooltip()` —— 格式化节点悬停提示
- `formatEdgeTooltip()` —— 格式化边悬停提示

**图例组件**（`web/src/components/topology/TopologyLegend.tsx`）：
- 展示节点类型（IP / Service）
- 展示节点尺寸映射（低/中/高流量）
- 展示协议颜色（TCP/UDP/ICMP/Other）
- 展示边宽映射（小/中/大流量）
- 提供交互提示（悬停、点击、拖拽、缩放）

**样式**（`web/src/styles/topology.css`）：
- 图表容器样式
- 节点高亮动画
- 边流动动画（可选）
- 加载与空状态样式
- 响应式布局规则

## 影响

**新增文件**：
- `web/src/components/topology/TopologyGraph.tsx`（主图组件）
- `web/src/components/topology/TopologyLegend.tsx`（图例组件）
- `web/src/components/topology/topologyConfig.ts`（ECharts 配置）
- `web/src/styles/topology.css`（样式文件）

**依赖**：
- `echarts`：^6.0.0（已安装）
- `echarts-for-react`：^3.0.5（已安装）
- 依赖 `add-topology-data-foundation`（数据类型与工具）

**测试要求**：
- 不同节点数量的可视化测试（10、50、100+）
- 交互测试（点击、拖拽、缩放）
- 性能测试（首次渲染时间、100+ 节点帧率）
- 响应式测试（移动端、平板、桌面）

**Breaking Changes**：无（纯新增）

**影响的能力**：
- `topology-visualization`（扩展现有能力）


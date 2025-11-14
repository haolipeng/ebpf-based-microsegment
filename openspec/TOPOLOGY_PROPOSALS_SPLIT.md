# 网络流量拓扑图功能 - OpenSpec提案拆分方案

## 概述

将网络流量拓扑图功能拆分为3个独立的OpenSpec change提案，每个提案职责清晰、可独立交付和测试。

---

## 拆分策略

### 提案1: add-topology-data-foundation
**数据基础层** - 类型、工具、Hook

### 提案2: add-topology-visualization-core  
**可视化核心层** - 图表组件、交互、图例

### 提案3: add-topology-page-integration
**页面集成层** - 完整页面、控制栏、详情面板

---

## 提案1: add-topology-data-foundation

### Change ID
`add-topology-data-foundation`

### 为什么 (Why)

需要建立拓扑图的数据基础设施，将网络流量数据转换为可视化所需的拓扑图数据结构。

**当前痛点**:
- Flow数据是扁平的记录列表，无法直接用于拓扑图渲染
- 缺少节点和边的数据模型
- 需要支持IP视图和标签视图两种聚合方式
- 需要实时更新机制

**业务需求**:
- 将Flow数组聚合为节点(Node)和边(Edge)
- 支持按IP或标签聚合
- 计算节点和边的流量指标
- 支持实时WebSocket数据更新

### 变更内容 (What Changes)

#### 1. 类型定义 (`web/src/types/topology.ts`)

```typescript
// 节点定义
export interface TopologyNode {
  id: string
  label: string
  type: 'IP' | 'SERVICE' // IP地址或服务标签
  metrics: {
    flowCount: number
    packetCount: number
    byteCount: number
    activeFlows: number
  }
  labels?: Record<string, string> // 标签视图时的元数据
  position?: { x: number; y: number } // 可选的固定位置
}

// 边定义
export interface TopologyEdge {
  id: string
  source: string // 源节点ID
  target: string // 目标节点ID
  metrics: {
    flowCount: number
    packetCount: number
    byteCount: number
    protocols: string[] // TCP, UDP, ICMP等
  }
  direction: 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL'
}

// 拓扑数据
export interface TopologyData {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
  stats: {
    totalNodes: number
    totalEdges: number
    totalFlows: number
  }
}

// 视图模式
export type TopologyViewMode = 'IP' | 'LABEL'

// 筛选条件
export interface TopologyFilters extends FlowQuery {
  viewMode: TopologyViewMode
  maxNodes?: number // 最多显示N个节点(Top N)
}
```

#### 2. 数据聚合工具 (`web/src/utils/topologyUtils.ts`)

```typescript
/**
 * 将Flow数组聚合为拓扑图数据
 */
export function aggregateFlowsToTopology(
  flows: Flow[],
  viewMode: TopologyViewMode,
  maxNodes?: number
): TopologyData

/**
 * 计算节点大小(用于可视化)
 */
export function calculateNodeSize(flowCount: number): number

/**
 * 计算边宽度(用于可视化)
 */
export function calculateEdgeWidth(byteCount: number): number

/**
 * 合并实时更新的Flow到现有拓扑数据
 */
export function mergeTopologyUpdate(
  existing: TopologyData,
  newFlow: Flow,
  viewMode: TopologyViewMode
): TopologyData

/**
 * 获取节点的显示标签
 */
export function getNodeLabel(node: TopologyNode): string
```

#### 3. 拓扑数据Hook (`web/src/hooks/useTopology.ts`)

```typescript
/**
 * 获取拓扑图数据的自定义Hook
 * 
 * @param filters - 筛选条件(包含viewMode)
 * @returns 拓扑数据和状态
 */
export function useTopology(filters: TopologyFilters) {
  return {
    data: TopologyData | undefined
    isLoading: boolean
    error: Error | null
    refetch: () => void
  }
}
```

功能:
- 调用useFlows获取流数据
- 使用topologyUtils聚合为拓扑数据
- 集成useFlowStream实时更新
- 支持时间范围筛选
- 自动防抖更新(避免频繁重新计算)

### 成功标准

- [ ] TypeScript编译无错误
- [ ] 所有类型定义完整且合理
- [ ] aggregateFlowsToTopology正确将Flow转换为拓扑数据
  - [ ] IP模式: 按sourceIp/destIp聚合
  - [ ] 标签模式: 按sourceLabels/destLabels聚合
- [ ] 计算函数(节点大小、边宽度)返回合理值
- [ ] mergeTopologyUpdate正确合并实时数据
- [ ] useTopology Hook正确返回数据
- [ ] 实时WebSocket更新正常工作
- [ ] 单元测试覆盖率>80%

### 依赖

- 已有: `Flow`类型、`useFlows`、`useFlowStream`
- 无外部新增npm依赖

### 影响范围

**新增文件**:
- `web/src/types/topology.ts`
- `web/src/utils/topologyUtils.ts`
- `web/src/hooks/useTopology.ts`

**不影响现有代码**

### 预计工作量

2-3天

---

## 提案2: add-topology-visualization-core

### Change ID
`add-topology-visualization-core`

### 为什么 (Why)

需要实现拓扑图的核心可视化能力，使用ECharts渲染交互式网络图。

**当前痛点**:
- 缺少网络拓扑图可视化组件
- 需要支持大规模节点渲染(100+节点)
- 需要丰富的交互能力(缩放、拖拽、点击)

**业务需求**:
- 可视化节点和边
- 节点大小/颜色反映流量指标
- 边粗细/颜色反映流量大小和协议
- 支持交互操作(缩放、平移、拖拽、点击)
- 显示图例说明视觉编码规则

### 变更内容 (What Changes)

#### 1. 拓扑图主组件 (`web/src/components/topology/TopologyGraph.tsx`)

```typescript
interface TopologyGraphProps {
  data: TopologyData
  viewMode: TopologyViewMode
  onNodeClick?: (node: TopologyNode) => void
  onEdgeClick?: (edge: TopologyEdge) => void
  height?: number | string
  loading?: boolean
}

export default function TopologyGraph(props: TopologyGraphProps)
```

功能:
- 使用echarts-for-react渲染Graph图表
- 力导向布局(force-directed layout)
- 节点视觉映射:
  - 大小 = calculateNodeSize(flowCount)
  - 颜色 = 节点类型/状态
  - 图标 = IP/服务图标
- 边视觉映射:
  - 粗细 = calculateEdgeWidth(byteCount)
  - 颜色 = 协议类型(TCP=蓝色, UDP=绿色, ICMP=橙色)
  - 箭头 = 流量方向
- 交互能力:
  - 鼠标悬停显示tooltip(节点/边详情)
  - 点击节点触发onNodeClick事件
  - 拖拽节点重新布局
  - 缩放和平移画布
  - 双击节点聚焦(高亮相关连接)

#### 2. ECharts配置 (`web/src/components/topology/topologyConfig.ts`)

```typescript
/**
 * 生成ECharts Graph配置
 */
export function getTopologyChartOption(
  data: TopologyData,
  viewMode: TopologyViewMode
): EChartsOption

/**
 * 节点样式配置
 */
export function getNodeStyle(node: TopologyNode): ItemStyleOption

/**
 * 边样式配置
 */
export function getEdgeStyle(edge: TopologyEdge): LineStyleOption

/**
 * Tooltip格式化
 */
export function formatTooltip(
  type: 'node' | 'edge',
  data: TopologyNode | TopologyEdge
): string
```

#### 3. 图例组件 (`web/src/components/topology/TopologyLegend.tsx`)

```typescript
interface TopologyLegendProps {
  viewMode: TopologyViewMode
}

export default function TopologyLegend(props: TopologyLegendProps)
```

显示:
- 节点大小说明(小/中/大 = 流量少/中/多)
- 节点颜色说明(IP节点=蓝色, 服务节点=绿色)
- 边粗细说明(细/粗 = 流量少/多)
- 协议颜色说明(TCP/UDP/ICMP)
- 流量方向箭头说明

#### 4. 样式文件 (`web/src/styles/topology.css`)

```css
/* 图表容器 */
.topology-graph-container { ... }

/* 节点高亮效果 */
.topology-node-highlight { ... }

/* 边动画效果 */
.topology-edge-animate { ... }

/* Loading状态 */
.topology-loading { ... }
```

### 成功标准

- [ ] TopologyGraph正确渲染nodes和edges
- [ ] 节点视觉映射正确(大小、颜色、图标)
- [ ] 边视觉映射正确(粗细、颜色、箭头)
- [ ] 交互功能正常:
  - [ ] 缩放和平移
  - [ ] 拖拽节点
  - [ ] 点击节点触发事件
  - [ ] 悬停显示tooltip
- [ ] 图例正确显示说明
- [ ] 响应式布局(支持不同屏幕尺寸)
- [ ] 性能流畅(100+节点不卡顿)
- [ ] 无TypeScript错误
- [ ] 无ESLint警告

### 依赖

- `echarts`: ^6.0.0 (已安装)
- `echarts-for-react`: ^3.0.5 (已安装)
- 提案1: add-topology-data-foundation

### 影响范围

**新增文件**:
- `web/src/components/topology/TopologyGraph.tsx`
- `web/src/components/topology/TopologyLegend.tsx`
- `web/src/components/topology/topologyConfig.ts`
- `web/src/styles/topology.css`

**不影响现有代码**

### 预计工作量

2-3天

---

## 提案3: add-topology-page-integration

### Change ID
`add-topology-page-integration`

### 为什么 (Why)

将拓扑图数据层和可视化层整合为完整的页面功能，提供用户可访问的网络拓扑视图。

**当前痛点**:
- 用户无法访问拓扑图功能
- 缺少控制界面切换视图模式和筛选条件
- 缺少节点详情查看能力

**业务需求**:
- 创建独立的Topology页面(/topology)
- 提供视图模式切换(IP/标签)
- 提供时间范围筛选
- 提供实时更新开关
- 点击节点查看详情
- 导航菜单添加入口

### 变更内容 (What Changes)

#### 1. Topology页面 (`web/src/pages/Topology/index.tsx`)

```typescript
export default function Topology()
```

页面结构:
```
<div className="topology-page">
  <header>
    <Title>Network Topology</Title>
    <TopologyControls />
  </header>
  
  <main>
    <TopologyGraph />
    <TopologyLegend />
  </main>
  
  <NodeDetailPanel />
</div>
```

状态管理:
```typescript
- selectedNode: TopologyNode | null
- viewMode: TopologyViewMode
- filters: TopologyFilters
- realtimeEnabled: boolean
```

#### 2. 控制栏组件 (`web/src/components/topology/TopologyControls.tsx`)

```typescript
interface TopologyControlsProps {
  viewMode: TopologyViewMode
  onViewModeChange: (mode: TopologyViewMode) => void
  filters: TopologyFilters
  onFiltersChange: (filters: TopologyFilters) => void
  realtimeEnabled: boolean
  onRealtimeToggle: (enabled: boolean) => void
}

export default function TopologyControls(props: TopologyControlsProps)
```

包含:
- 视图模式切换: `<Segmented options={['IP', 'LABEL']} />`
- 时间范围: `<DatePicker.RangePicker />`
- 实时更新: `<Switch checked={realtimeEnabled} />`
- 协议筛选: `<Select options={protocols} />`
- 状态筛选: `<Select options={states} />`
- 最大节点数: `<InputNumber max={200} />`
- 重置按钮: `<Button>Reset</Button>`

#### 3. 节点详情面板 (`web/src/components/topology/NodeDetailPanel.tsx`)

```typescript
interface NodeDetailPanelProps {
  node: TopologyNode | null
  flows: Flow[] // 与该节点相关的流
  onClose: () => void
}

export default function NodeDetailPanel(props: NodeDetailPanelProps)
```

使用Ant Design Drawer显示:
- **基本信息**:
  - 节点ID/标签
  - 节点类型(IP/服务)
  - 标签信息(标签视图)
- **流量统计**:
  - 总流数
  - 总包数
  - 总字节数(格式化)
  - 活跃流数
- **相关流列表**:
  - 显示与该节点相关的Flow记录
  - 支持排序和分页
- **策略信息**:
  - 应用的策略(如果有)

#### 4. 路由集成 (`web/src/router.tsx`)

添加新路由:
```typescript
{
  path: '/topology',
  element: <Topology />,
}
```

#### 5. 导航菜单更新

在主布局的菜单中添加:
```typescript
{
  key: 'topology',
  icon: <ShareAltOutlined />,
  label: 'Topology',
  path: '/topology',
}
```

### 成功标准

- [ ] Topology页面可访问(/topology)
- [ ] 双视图模式切换正常(IP ↔ 标签)
- [ ] 时间范围筛选正常工作
- [ ] 实时更新开关正常工作
- [ ] 控制栏所有筛选器正常工作
- [ ] 点击节点打开详情面板
- [ ] 详情面板正确显示节点信息和相关流
- [ ] 导航菜单入口可见且可点击
- [ ] 页面响应式布局(移动端/平板/桌面)
- [ ] Loading和Error状态友好
- [ ] 性能流畅(100+节点)
- [ ] 无TypeScript错误
- [ ] 无ESLint警告

### 依赖

- Ant Design组件库(已安装)
- 提案1: add-topology-data-foundation
- 提案2: add-topology-visualization-core

### 影响范围

**新增文件**:
- `web/src/pages/Topology/index.tsx`
- `web/src/components/topology/TopologyControls.tsx`
- `web/src/components/topology/NodeDetailPanel.tsx`

**修改文件**:
- `web/src/router.tsx` - 添加路由
- 主布局组件 - 添加菜单项

### 预计工作量

1-2天

---

## 实施计划

### 阶段1: 数据基础层 (2-3天)
**提案**: add-topology-data-foundation

**可交付成果**:
- ✅ 类型定义完整
- ✅ 工具函数实现并测试
- ✅ useTopology Hook可用
- ✅ 单元测试通过

**验收方式**:
```typescript
// 可以这样使用
const { data, isLoading } = useTopology({
  viewMode: 'IP',
  maxNodes: 100,
})
// data = { nodes: [...], edges: [...], stats: {...} }
```

### 阶段2: 可视化核心层 (2-3天)
**提案**: add-topology-visualization-core

**可交付成果**:
- ✅ TopologyGraph组件渲染正常
- ✅ 交互功能可用
- ✅ 图例组件显示正确
- ✅ 样式美观

**验收方式**:
```tsx
// 可以这样使用
<TopologyGraph
  data={topologyData}
  viewMode="IP"
  onNodeClick={handleNodeClick}
/>
<TopologyLegend viewMode="IP" />
```

**并行开发提示**:
- 可以使用mock TopologyData进行开发
- 与阶段1部分并行

### 阶段3: 页面集成层 (1-2天)
**提案**: add-topology-page-integration

**可交付成果**:
- ✅ Topology完整页面可用
- ✅ 所有控制功能正常
- ✅ 详情面板可用
- ✅ 路由和导航正常

**验收方式**:
- 访问 http://localhost:3000/topology
- 切换视图模式
- 筛选数据
- 点击节点查看详情

### 总工期: 5-8天

---

## 拆分优势

1. **职责清晰**: 每个提案有明确的边界和可交付成果
2. **独立测试**: 每层可独立测试和验证
3. **并行开发**: 阶段2可使用mock数据部分并行开发
4. **风险控制**: 失败影响范围小，易于回滚
5. **Review友好**: 每个提案代码量适中(300-500行)，易于review
6. **复用性好**: 数据层和可视化层可复用到其他场景(如Dashboard的小型拓扑图)
7. **增量交付**: 每个阶段完成后都有可演示的成果

---

## OpenSpec操作命令

### 创建提案目录

```bash
# 提案1
mkdir -p openspec/changes/add-topology-data-foundation/specs/topology-visualization
touch openspec/changes/add-topology-data-foundation/{proposal.md,tasks.md}

# 提案2  
mkdir -p openspec/changes/add-topology-visualization-core/specs/topology-visualization
touch openspec/changes/add-topology-visualization-core/{proposal.md,tasks.md}

# 提案3
mkdir -p openspec/changes/add-topology-page-integration/specs/topology-visualization
touch openspec/changes/add-topology-page-integration/{proposal.md,tasks.md}
```

### 验证提案

```bash
openspec validate add-topology-data-foundation --strict
openspec validate add-topology-visualization-core --strict
openspec validate add-topology-page-integration --strict
```

### 查看提案

```bash
openspec show add-topology-data-foundation
openspec show add-topology-visualization-core
openspec show add-topology-page-integration
```

---

## 风险和缓解措施

### 风险1: 性能问题(大规模节点)
**场景**: 节点数>100时渲染卡顿

**缓解**:
- 提案1中限制maxNodes参数(默认100)
- 提案2中使用ECharts的渐进式渲染
- 考虑虚拟化或LOD(Level of Detail)

### 风险2: 实时更新频率过高
**场景**: WebSocket推送过快导致频繁重新渲染

**缓解**:
- 提案1中实现防抖机制(500ms聚合更新)
- 批量更新而非逐条更新

### 风险3: 标签视图数据不足
**场景**: 部分Flow缺少sourceLabels/destLabels

**缓解**:
- 提案1中处理null/undefined情况
- 显示为"Unlabeled"节点

---

## 参考资料

- [Illumio Application Dependency Map](https://docs.illumio.com/core/21.5/Content/Guides/security-policy/application-dependency-map/application-dependency-map.htm)
- [ECharts Graph文档](https://echarts.apache.org/zh/option.html#series-graph)
- [Force-Directed Layout算法](https://en.wikipedia.org/wiki/Force-directed_graph_drawing)
- 项目现有代码: `web/src/pages/Flows/index.tsx` (实时更新参考)


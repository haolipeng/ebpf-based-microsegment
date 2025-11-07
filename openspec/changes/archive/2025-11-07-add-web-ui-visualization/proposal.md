# Proposal: 添加 Web UI 可视化功能 (Web UI Visualization)

## Why (为什么)

当前 Web UI 缺少高级可视化功能,用户无法直观地分析网络流量趋势、流量分布和网络拓扑关系。虽然基础的统计卡片和表格已经实现,但缺少:

1. **流量趋势图表** - 无法查看时间序列数据,难以发现流量模式和异常
2. **Top Talkers 分析** - 无法快速识别最活跃的源/目标 IP
3. **协议分布饼图** - 无法直观看到不同协议的流量占比
4. **网络拓扑图** - 无法可视化工作负载之间的通信关系

实现可视化功能将使用户能够:
- 快速发现流量异常和攻击模式
- 识别网络瓶颈和热点
- 理解微隔离策略的影响
- 优化网络安全配置

## What Changes (变更内容)

### 1. 流量趋势图表 (Traffic Trend Charts)

**组件**: `FlowTrendChart.tsx`
- 时间序列折线图 (流数量随时间变化)
- 字节数面积图 (流量大小趋势)
- 按协议分组的多折线图 (TCP/UDP/ICMP)
- 时间粒度选择 (1分钟/5分钟/1小时)
- 实时数据更新
- 可缩放和拖动
- Tooltip 显示详细数据

**技术栈**: Apache ECharts 或 Recharts

### 2. Top Talkers 可视化 (Top Talkers Visualization)

**组件**: `TopTalkersChart.tsx`
- Top 10 源 IP 横向柱状图
- Top 10 目标 IP 横向柱状图
- 显示流数量和字节数
- 按流量或字节数排序切换
- 点击 IP 跳转到相关流量列表
- 显示标签信息 (如果有)

### 3. 协议分布图表 (Protocol Distribution Charts)

**组件**: `ProtocolDistributionChart.tsx`
- 饼图显示协议占比 (TCP/UDP/ICMP)
- 环形图显示协议流量字节数分布
- 鼠标悬停显示百分比和具体数值
- 颜色编码: TCP=蓝色、UDP=绿色、ICMP=橙色

### 4. 策略效果可视化 (Policy Effectiveness Visualization)

**组件**: `PolicyEffectivenessChart.tsx`
- 堆叠柱状图: Allow vs Deny vs Log 策略命中数
- 按时间显示策略命中趋势
- Top 10 最常命中的策略
- 策略命中率百分比

### 5. 网络拓扑图 (Network Topology Graph)

**组件**: `NetworkTopologyGraph.tsx`
- 力导向图显示工作负载通信关系
- 节点代表工作负载 (按标签分组)
- 边代表流量 (粗细表示流量大小)
- 节点颜色表示策略动作 (绿=allow, 红=deny)
- 可拖动节点
- 缩放和平移
- 点击节点显示详细信息

**技术栈**: D3.js 或 Cytoscape.js

### 6. 仪表盘聚合视图 (Dashboard Aggregation)

**页面**: `/dashboard` 增强
- 集成所有关键可视化组件
- 响应式网格布局
- 可自定义图表位置和大小
- 导出图表为图片功能

### React Hooks

- `useFlowTrend(timeRange, granularity)` - 获取流量趋势数据
- `useTopTalkers(type, limit)` - 获取 Top Talkers 数据
- `useProtocolDistribution(timeRange)` - 获取协议分布
- `usePolicyEffectiveness(timeRange)` - 获取策略效果统计
- `useNetworkTopology()` - 获取网络拓扑数据

### API 需求

需要后端提供以下 API (如果尚未实现):
- `GET /api/v1/flows/trends` - 流量趋势时间序列
- `GET /api/v1/flows/top-talkers` - Top Talkers 统计
- `GET /api/v1/flows/protocol-distribution` - 协议分布
- `GET /api/v1/policies/effectiveness` - 策略效果统计
- `GET /api/v1/topology/graph` - 网络拓扑图数据

## Success Criteria (成功标准)

1. **功能完整性**
   - ✅ 流量趋势图表正确显示时间序列数据
   - ✅ Top Talkers 正确排序和显示
   - ✅ 协议分布饼图显示准确百分比
   - ✅ 策略效果图表正确统计命中数
   - ✅ 网络拓扑图正确显示节点和边

2. **交互性**
   - ✅ 图表支持缩放、拖动、选择
   - ✅ Tooltip 显示详细信息
   - ✅ 点击节点/柱状图可以钻取到详细数据
   - ✅ 时间范围和粒度可以动态调整

3. **性能**
   - ✅ 大数据量渲染流畅 (1000+ 数据点)
   - ✅ 图表渲染时间 <1 秒
   - ✅ 实时更新不卡顿
   - ✅ 使用数据聚合和采样优化性能

4. **响应式设计**
   - ✅ 图表在不同屏幕尺寸下自适应
   - ✅ 移动端显示简化版图表
   - ✅ 图例和标签不重叠

5. **代码质量**
   - ✅ TypeScript 类型检查通过
   - ✅ ESLint 检查无错误
   - ✅ 图表组件可复用
   - ✅ 遵循现有代码风格

## Dependencies (依赖项)

### 前端依赖
- **ECharts** 或 **Recharts** - 图表库 (需要安装)
- **D3.js** 或 **Cytoscape.js** - 网络拓扑图 (需要安装)
- Ant Design - 布局和 UI 组件
- React Query - 数据获取

### 后端 API
- 流量趋势 API
- Top Talkers API
- 协议分布 API
- 策略效果 API
- 网络拓扑 API

### 已实现功能
- Web UI Foundation ✅
- Flow Analytics UI (基础部分) ✅
- Agent Management UI ✅

## 实施顺序

建议按以下顺序实施,以便逐步交付价值:

1. **Phase 1: 基础图表** (最高优先级)
   - 流量趋势折线图
   - Top Talkers 柱状图
   - 协议分布饼图

2. **Phase 2: 高级分析**
   - 策略效果可视化
   - 多维度对比图表

3. **Phase 3: 网络拓扑** (最复杂)
   - 网络拓扑力导向图
   - 交互式节点操作

## 技术选型建议

- **图表库**: 推荐 **Apache ECharts**
  - 功能强大,支持多种图表类型
  - 性能优秀,支持大数据量
  - 文档完善,社区活跃
  - TypeScript 支持良好

- **拓扑图**: 推荐 **Cytoscape.js**
  - 专为网络图设计
  - 丰富的布局算法
  - 良好的性能
  - 易于定制

# Tasks: 添加 Web UI 可视化功能

**状态: ✅ 核心功能已完成 (Core Features Completed)**

## 完成情况总结

### 已完成的 Phases
- ✅ Phase 1: 基础设施和依赖
- ✅ Phase 2: 流量趋势图表
- ✅ Phase 3: Top Talkers 可视化
- ✅ Phase 4: 协议分布图表
- ✅ Phase 5: 策略效果可视化
- ✅ Phase 6: 仪表盘集成
- ⏭️ Phase 7: 网络拓扑图 (可选 - 跳过)
- ✅ Phase 8: 优化和测试

### 实现的图表组件 (8个)
1. FlowTrendChart - 流量趋势折线图
2. BytesTrendChart - 字节数趋势面积图
3. ProtocolTrendChart - 协议多折线图
4. TopTalkersChart - Top Talkers 横向柱状图
5. ProtocolPieChart - 协议分布饼图
6. ProtocolDonutChart - 协议字节数环形图
7. PolicyEffectivenessChart - 策略效果堆叠柱状图
8. TopPoliciesChart - Top 策略横向柱状图

### 性能优化
- ✅ 数据采样 (MAX_CHART_POINTS = 200)
- ✅ React Query 缓存和自动刷新
- ✅ useMemo 优化渲染
- ✅ 图表懒加载

### 验证结果
- ✅ TypeScript 检查通过
- ✅ ESLint 通过 (1个无关警告)
- ✅ 开发服务器运行正常
- ✅ HMR 热更新成功

---

## Phase 1: 基础设施和依赖 (Infrastructure & Dependencies)

### Task 1.1: 安装图表库依赖
- 安装 Apache ECharts: `npm install echarts echarts-for-react`
- 安装类型定义: `npm install @types/echarts --save-dev`
- 验证安装成功
- 更新 package.json

### Task 1.2: 创建图表工具函数
- 文件: `web/src/utils/chartHelpers.ts`
- 通用图表配置
- 颜色主题定义
- 时间格式化函数
- 数据聚合函数
- 验证: TypeScript 检查

### Task 1.3: 创建可视化 Hooks
- 文件: `web/src/hooks/useVisualization.ts`
- 实现 `useFlowTrend()` - 获取流量趋势
- 实现 `useTopTalkers()` - 获取 Top Talkers
- 实现 `useProtocolDistribution()` - 获取协议分布
- 使用 React Query 缓存
- 验证: TypeScript 检查和 ESLint

## Phase 2: 流量趋势图表 (Traffic Trend Charts)

### Task 2.1: 创建流量趋势折线图组件
- 文件: `web/src/components/visualization/FlowTrendChart.tsx`
- 使用 ECharts 折线图
- X 轴: 时间
- Y 轴: 流数量
- 时间粒度选择器 (1分钟/5分钟/1小时)
- Tooltip 显示详细数据
- 响应式尺寸
- 验证: TypeScript 检查和 ESLint

### Task 2.2: 添加字节数趋势面积图
- 文件: `web/src/components/visualization/BytesTrendChart.tsx`
- 使用 ECharts 面积图
- X 轴: 时间
- Y 轴: 字节数 (格式化为 KB/MB/GB)
- 平滑曲线
- 渐变填充
- 验证: TypeScript 检查和 ESLint

### Task 2.3: 创建按协议分组的多折线图
- 文件: `web/src/components/visualization/ProtocolTrendChart.tsx`
- 多条折线 (TCP/UDP/ICMP)
- 颜色编码: TCP=蓝色、UDP=绿色、ICMP=橙色
- 图例可点击切换显示
- 堆叠模式可选
- 验证: TypeScript 检查和 ESLint

## Phase 3: Top Talkers 可视化 (Top Talkers Visualization)

### Task 3.1: 创建 Top Talkers 柱状图组件
- 文件: `web/src/components/visualization/TopTalkersChart.tsx`
- 横向柱状图
- Top 10 源 IP 和 Top 10 目标 IP
- 显示流数量和字节数
- 排序切换按钮 (按流量/按字节)
- 点击柱状图跳转到流量列表
- 验证: TypeScript 检查和 ESLint

### Task 3.2: 添加标签信息显示
- 在 Top Talkers 柱状图中显示 IP 标签
- 如果 IP 有标签,显示在柱状图旁边
- 无标签时显示 "No labels"
- 验证: TypeScript 检查和 ESLint

## Phase 4: 协议分布图表 (Protocol Distribution Charts)

### Task 4.1: 创建协议分布饼图组件
- 文件: `web/src/components/visualization/ProtocolPieChart.tsx`
- 使用 ECharts 饼图
- 显示 TCP/UDP/ICMP 占比
- 百分比标签
- 颜色编码一致
- Tooltip 显示流数量和百分比
- 验证: TypeScript 检查和 ESLint

### Task 4.2: 创建协议流量环形图
- 文件: `web/src/components/visualization/ProtocolDonutChart.tsx`
- 使用 ECharts 环形图
- 显示字节数分布
- 中心显示总字节数
- 鼠标悬停高亮
- 验证: TypeScript 检查和 ESLint

## Phase 5: 策略效果可视化 (Policy Effectiveness Visualization)

### Task 5.1: 创建策略命中堆叠柱状图
- 文件: `web/src/components/visualization/PolicyEffectivenessChart.tsx`
- 堆叠柱状图: Allow/Deny/Log
- 按时间显示趋势
- 颜色: Allow=绿色、Deny=红色、Log=蓝色
- 时间范围选择器
- 验证: TypeScript 检查和 ESLint

### Task 5.2: 添加 Top 10 最常命中策略图表
- 文件: `web/src/components/visualization/TopPoliciesChart.tsx`
- 横向柱状图
- 显示策略 Rule ID 和命中次数
- 点击跳转到策略详情
- 验证: TypeScript 检查和 ESLint

## Phase 6: 仪表盘集成 (Dashboard Integration)

### Task 6.1: 更新仪表盘页面
- 文件: `web/src/pages/Dashboard/index.tsx`
- 集成流量趋势图表
- 集成 Top Talkers 图表
- 集成协议分布图表
- 响应式网格布局 (Ant Design Row/Col)
- 验证: TypeScript 检查和 ESLint

### Task 6.2: 添加图表刷新和导出功能
- 每个图表添加刷新按钮
- 图表导出为 PNG 图片功能
- 统一的加载状态
- 统一的错误处理
- 验证: TypeScript 检查和 ESLint

## Phase 7: 网络拓扑图 (Network Topology - Optional)

### Task 7.1: 安装拓扑图库
- 安装 Cytoscape.js: `npm install cytoscape`
- 安装类型定义: `npm install @types/cytoscape --save-dev`
- 安装布局插件: `npm install cytoscape-cose-bilkent`
- 验证安装

### Task 7.2: 创建网络拓扑图组件
- 文件: `web/src/components/visualization/NetworkTopologyGraph.tsx`
- 力导向图布局
- 节点代表工作负载
- 边代表流量
- 可拖动节点
- 缩放和平移
- 验证: TypeScript 检查和 ESLint

### Task 7.3: 添加拓扑图交互功能
- 点击节点显示详细信息
- 高亮连接的节点和边
- 节点颜色表示策略动作
- 边粗细表示流量大小
- 验证: TypeScript 检查和 ESLint

## Phase 8: 优化和测试 (Optimization & Testing)

### Task 8.1: 性能优化
- 大数据量时使用数据采样
- 图表懒加载
- 虚拟化长列表
- Debounce 实时更新
- 验证: 性能测试

### Task 8.2: 响应式设计优化
- 移动端图表简化
- 图表自适应容器大小
- 图例和标签不重叠
- 验证: 不同屏幕尺寸测试

### Task 8.3: 错误处理和空状态
- API 请求失败时显示错误
- 数据为空时显示空状态提示
- 加载状态骨架屏
- 重试机制
- 验证: 手动测试各种错误场景

### Task 8.4: 最终验证
- 运行 TypeScript 检查: `cd web && npx tsc --noEmit`
- 运行 ESLint: `cd web && npm run lint`
- 手动测试所有图表交互
- 测试数据刷新功能
- 测试导出功能
- 更新 tasks.md 标记所有任务完成

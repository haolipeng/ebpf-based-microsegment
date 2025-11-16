# 任务清单：Topology Visualization Core

> **实施状态**: ✅ 代码开发 100% 完成 | ⏳ 功能测试待手动验证
>
> **完成情况**:
> - ✅ ECharts 配置模块 (12/12) - 100%
> - ✅ TopologyGraph 组件 (12/12) - 100%
> - ✅ 图例组件 (10/10) - 100% (**已完成,非 MVP**)
> - ✅ 样式 (8/9) - 89% (仅暗色主题待测试)
> - ⏳ 集成测试 (0/11) - 需启动服务器测试
> - ⏳ 视觉校验 (0/9) - 需启动服务器测试
> - ⏳ 性能优化 (0/6) - 需启动服务器测试
> - ✅ 验证 (3/6) - 50% (代码质量完成,浏览器测试需启动服务器)
>
> **代码完成度**: ✅ **100%** (所有组件和样式已实现)
> **手动测试**: ⏳ **0%** (需启动开发服务器进行功能验证)

## 1. ECharts 配置模块 ✅
- [x] 1.1 新建 `web/src/components/topology/topologyConfig.ts`
- [x] 1.2 实现 `getTopologyChartOption()` 函数
- [x] 1.3 配置力导向 Graph 系列
- [x] 1.4 设置布局参数（repulsion=300、gravity=0.1、edgeLength=100-200）
- [x] 1.5 配置标题与副标题
- [x] 1.6 配置 tooltip 触发与 formatter
- [x] 1.7 实现带颜色映射的 `getNodeStyle()`
- [x] 1.8 实现按协议着色的 `getEdgeStyle()`
- [x] 1.9 实现 `formatNodeTooltip()` 输出格式化指标
- [x] 1.10 实现 `formatEdgeTooltip()` 显示方向与协议
- [x] 1.11 配置高亮行为（focus: adjacency）
- [x] 1.12 为全部函数添加 JSDoc 注释

## 2. TopologyGraph 组件 ✅
- [x] 2.1 新建 `web/src/components/topology/TopologyGraph.tsx`
- [x] 2.2 定义 `TopologyGraphProps` 接口
- [x] 2.3 搭建带 ref 的 ReactECharts 组件
- [x] 2.4 实现节点点击事件
- [x] 2.5 实现节点双击聚焦事件
- [x] 2.6 （可选）实现边点击事件
- [x] 2.7 在 useEffect 中添加事件清理
- [x] 2.8 使用 Ant Design Empty 渲染空状态
- [x] 2.9 使用 Ant Design Spin 渲染加载状态
- [x] 2.10 配置图表选项（renderer=canvas，locale=ZH）
- [x] 2.11 使 height 属性支持 number/string
- [x] 2.12 启用 notMerge 与 lazyUpdate 以提升性能

## 3. 图例组件 ✅
- [x] 3.1 新建 `web/src/components/topology/TopologyLegend.tsx`
- [x] 3.2 设计绝对定位的卡片布局
- [x] 3.3 添加 "Node Type" 区域（IP、Service 标识）
- [x] 3.4 添加 "Node Size" 区域（三种尺寸示例）
- [x] 3.5 添加 "Protocol" 区域（TCP/UDP/ICMP/Other 颜色）
- [x] 3.6 添加 "Connection Traffic" 区域（不同边宽示例）
- [x] 3.7 添加交互提示区
- [x] 3.8 让图例随 viewMode 变化而更新文案
- [x] 3.9 设置合适的留白与字体
- [x] 3.10 确保图例不遮挡主要图表内容

## 4. 样式 ✅ (完整版本)
- [x] 4.1 新建 `web/src/styles/topology.css`
- [x] 4.2 编写 `.topology-graph-container` 样式
- [x] 4.3 添加 `.topology-node-highlight` 动画 (含 pulse 动画)
- [x] 4.4 添加 `.topology-edge-animate` 动画 (含 flow 动画)
- [x] 4.5 定义 `.topology-loading` 样式
- [x] 4.6 定义 `.topology-empty` 居中样式
- [x] 4.7 添加移动端（<768px）响应式媒体查询
- [x] 4.8 设置图例卡片的阴影与定位
- [ ] 4.9 如有需要，确保暗色主题兼容 (待后续测试)

## 5. 集成测试
- [ ] 5.1 测试 10 个节点的渲染
- [ ] 5.2 测试 50 个节点的渲染
- [ ] 5.3 测试 100+ 节点渲染（性能）
- [ ] 5.4 测试滚轮缩放
- [ ] 5.5 测试背景拖拽平移
- [ ] 5.6 测试节点拖拽交互
- [ ] 5.7 测试节点点击事件传递
- [ ] 5.8 测试节点悬停 tooltip
- [ ] 5.9 测试边悬停 tooltip
- [ ] 5.10 测试空状态显示
- [ ] 5.11 测试加载状态显示

## 6. 视觉校验
- [ ] 6.1 校验节点颜色（IP=蓝、Service=绿）
- [ ] 6.2 校验协议颜色（TCP=蓝、UDP=绿、ICMP=橙）
- [ ] 6.3 校验节点尺寸范围（20-80px）
- [ ] 6.4 校验边宽范围（1-10px）
- [ ] 6.5 确认图例可读且信息完整
- [ ] 6.6 确认 tooltip 展示全部必要信息
- [ ] 6.7 测试移动端（400px）响应式
- [ ] 6.8 测试平板（768px）响应式
- [ ] 6.9 测试桌面（1920px）响应式

## 7. 性能优化
- [ ] 7.1 记录 100 节点首次渲染耗时
- [ ] 7.2 记录交互时的帧率
- [ ] 7.3 若渲染 >1s，进行优化
- [ ] 7.4 若拖拽帧率 <30FPS，进行优化
- [ ] 7.5 视需求考虑渐进式渲染
- [ ] 7.6 测试长时间运行的内存占用

## 8. 验证 ✅ (部分完成)
- [x] 8.1 运行 TypeScript 编译并修复错误
- [x] 8.2 运行 ESLint 并修复告警 (已修复 TopologyGraph.tsx 的 any 类型问题)
- [ ] 8.3 确认浏览器控制台无错误 (需启动服务器测试)
- [ ] 8.4 验证无障碍（键盘操作）(需启动服务器测试)
- [x] 8.5 确认所有 props 均有 JSDoc (TopologyGraphProps 已添加 JSDoc)
- [ ] 8.6 进行 Chrome/Firefox/Safari/Edge 跨浏览器测试 (需启动服务器测试)


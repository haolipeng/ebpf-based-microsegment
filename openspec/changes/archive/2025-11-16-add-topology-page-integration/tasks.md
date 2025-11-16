# 任务清单：Topology Page Integration

> **实施状态**: ✅ MVP 基础集成已完成 | ⏳ 高级交互组件待实现
>
> **完成情况**:
> - ⏳ TopologyControls 组件 (0/14) - MVP 暂不包含控制面板
> - ⏳ NodeDetailPanel 组件 (0/15) - MVP 暂不包含详情面板
> - ✅ Topology 页面组件 (8/18) - MVP 基础版本完成
> - ✅ 路由集成 (4/4)
> - ✅ 导航菜单集成 (5/6) - 基础集成完成，跳转测试需启动服务器
> - ✅ 样式引入 (3/3)
> - ⏳ 相关 Flow 逻辑 (0/5) - MVP 暂不包含
> - ⏳ 集成测试 (0/15) - 需启动服务器测试
> - ⏳ 响应式测试 (0/7) - 需启动服务器测试
> - ⏳ 错误处理测试 (0/5) - 需启动服务器测试
> - ✅ 验证 (4/7) - 代码质量检查完成，浏览器测试需启动服务器

## 1. TopologyControls 组件
- [ ] 1.1 新建 `web/src/components/topology/TopologyControls.tsx`
- [ ] 1.2 定义 `TopologyControlsProps` 接口
- [ ] 1.3 实现视图模式 Segmented（IP View / Service View）
- [ ] 1.4 实现默认 7 天的时间范围选择器
- [ ] 1.5 实现协议下拉框（TCP/UDP/ICMP/All）
- [ ] 1.6 实现状态下拉框（Active/Closed/Timeout/All）
- [ ] 1.7 实现动作下拉框（Allow/Deny/Log/All）
- [ ] 1.8 实现最大节点数输入框（10-200，步长 10）
- [ ] 1.9 实现带 On/Off 标签的实时开关
- [ ] 1.10 实现带 ReloadOutlined 图标的刷新按钮
- [ ] 1.11 实现重置按钮及处理逻辑
- [ ] 1.12 实现带 DownloadOutlined 图标的导出按钮（禁用）
- [ ] 1.13 使用 Card 包裹所有控件并设置间距
- [ ] 1.14 为每个控件添加标签（字体 12，颜色 #666）

## 2. NodeDetailPanel 组件
- [ ] 2.1 新建 `web/src/components/topology/NodeDetailPanel.tsx`
- [ ] 2.2 定义 `NodeDetailPanelProps` 接口
- [ ] 2.3 实现 Ant Design Drawer（宽 800px，右侧）
- [ ] 2.4 添加包含节点图标与 “Node Details” 的标题
- [ ] 2.5 创建 “Basic Information” Descriptions 卡片
- [ ] 2.6 展示节点 ID、类型、标签
- [ ] 2.7 创建 “Traffic Statistics” 双列 Descriptions
- [ ] 2.8 展示总流数、活跃流（绿色文本）、总包数、总字节数（格式化）
- [ ] 2.9 创建 “Related Flows” 表格
- [ ] 2.10 定义表头（Source/Dest IP/Port、Protocol、State、Action、Bytes）
- [ ] 2.11 使用 `protocolName()` 格式化协议
- [ ] 2.12 为 State/Action 应用颜色标签
- [ ] 2.13 添加分页（pageSize=10，显示总数）
- [ ] 2.14 创建 “Connection Statistics” Descriptions
- [ ] 2.15 展示入站/出站连接数量

## 3. Topology 页面组件 ✅ (MVP 基础版本)
- [x] 3.1 新建 `web/src/pages/Topology/index.tsx`
- [x] 3.2 管理 viewMode、realtimeEnabled、selectedNode、filters 状态 (MVP: 仅管理基础状态)
- [x] 3.3 初始化默认筛选（viewMode='IP'、maxNodes=100、近 7 天）
- [x] 3.4 集成 `useTopology()` 并传入筛选与实时标记
- [ ] 3.5 实现 `handleViewModeChange` 更新状态并清空选中节点 (MVP 暂不包含控制面板)
- [ ] 3.6 实现 `handleFiltersChange` 合并部分筛选变更 (MVP 暂不包含控制面板)
- [ ] 3.7 实现 `handleRealtimeToggle` (MVP 暂不包含控制面板)
- [ ] 3.8 实现 `handleNodeClick` 记录选中节点 (MVP 暂不包含详情面板)
- [ ] 3.9 实现 `handleDetailClose` 清除选中节点 (MVP 暂不包含详情面板)
- [x] 3.10 渲染页面标题 "Network Topology" 与描述
- [ ] 3.11 渲染 TopologyControls 并传入处理函数 (MVP 暂不包含控制面板)
- [ ] 3.12 启用实时模式时渲染状态 Alert (MVP 暂不包含)
- [x] 3.13 加载失败时渲染错误 Alert
- [x] 3.14 渲染带数据与 onNodeClick 的 TopologyGraph
- [ ] 3.15 渲染 TopologyLegend 并传入 viewMode (MVP 暂不包含图例)
- [ ] 3.16 渲染 NodeDetailPanel 并传递 selectedNode/flows (MVP 暂不包含详情面板)
- [ ] 3.17 计算 selectedNode 对应的 relatedFlows (MVP 暂不包含)
- [x] 3.18 应用响应式布局样式（padding:24px，height:calc(100vh-64px)）

## 4. 路由集成 ✅
- [x] 4.1 打开 `web/src/router.tsx`
- [x] 4.2 引入 Topology 页面组件
- [x] 4.3 添加 `{ path: '/topology', element: <Topology /> }`
- [x] 4.4 确认位于 MainLayout 子路由中

## 5. 导航菜单集成 ✅
- [x] 5.1 打开 `web/src/components/layout/Sidebar.tsx`
- [x] 5.2 引入 `ShareAltOutlined`
- [x] 5.3 向 `menuItems` 添加菜单项
- [x] 5.4 设置 key='/topology'、icon=<ShareAltOutlined />、label='Topology'
- [x] 5.5 放置在 Flows 与 Policies 之间
- [ ] 5.6 验证点击可正常跳转 (需启动服务器测试)

## 6. 样式引入 ✅
- [x] 6.1 打开 `web/src/main.tsx`
- [x] 6.2 添加 `import './styles/topology.css'`
- [x] 6.3 确认在 `index.css` 之后引入

## 7. 相关 Flow 逻辑
- [ ] 7.1 在 Topology 页面使用 useMemo 计算 `relatedFlows`
- [ ] 7.2 IP 视图：筛选 sourceIp/destIp = selectedNode.id 的 Flow
- [ ] 7.3 标签视图：筛选 sourceLabels.app/destLabels.app = selectedNode.label 的 Flow
- [ ] 7.4 selectedNode 为 null 时返回空数组
- [ ] 7.5 （当前实现返回空数组，作为占位说明）

## 8. 集成测试
- [ ] 8.1 浏览器访问 /topology
- [ ] 8.2 验证默认状态渲染正常
- [ ] 8.3 切换 IP/Service 视图并验证
- [ ] 8.4 更改时间范围并观察更新
- [ ] 8.5 切换协议筛选并验证
- [ ] 8.6 切换状态筛选并验证
- [ ] 8.7 切换动作筛选并验证
- [ ] 8.8 调整 max nodes 并确认节点数量受控
- [ ] 8.9 切换实时开关并验证 WebSocket 状态
- [ ] 8.10 点击 Refresh 并确认重新加载
- [ ] 8.11 点击 Reset 并确认筛选重置
- [ ] 8.12 点击节点并确认详情面板出现
- [ ] 8.13 验证详情信息正确
- [ ] 8.14 关闭详情面板并确认消失
- [ ] 8.15 确认处于 /topology 时菜单高亮

## 9. 响应式测试
- [ ] 9.1 移动端（400px）测试页面
- [ ] 9.2 确认控件纵向堆叠
- [ ] 9.3 确认图表高度适配移动端
- [ ] 9.4 确认图例可见且紧凑
- [ ] 9.5 平板（768px）测试
- [ ] 9.6 桌面（1920px）测试
- [ ] 9.7 确认详情 Drawer 在所有尺寸可用

## 10. 错误处理测试
- [ ] 10.1 无 Flow 数据时的空状态
- [ ] 10.2 API 报错时是否显示错误 Alert
- [ ] 10.3 WebSocket 断开时是否显示警告
- [ ] 10.4 数据加载过程是否显示 loading 状态
- [ ] 10.5 错误文案是否友好

## 11. 验证 ✅ (部分完成)
- [x] 11.1 运行 TypeScript 编译并修复错误
- [x] 11.2 运行 ESLint 并修复告警 (已修复 MVP 新增代码的告警)
- [ ] 11.3 确认浏览器控制台无错误 (需启动服务器测试)
- [x] 11.4 确认所有文案为英文（便于国际化）
- [x] 11.5 确认组件类型定义完整 (MVP 基础版本已完成)
- [ ] 11.6 确认交互元素可键盘操作 (需启动服务器测试)
- [ ] 11.7 在 Chrome/Firefox/Safari/Edge 进行跨浏览器测试 (需启动服务器测试)


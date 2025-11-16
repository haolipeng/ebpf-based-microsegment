# 提案：Topology Page Integration

## 为什么

数据层与可视化核心已经就绪，需要将其整合为一个可供用户访问的拓扑页面，包含完整的控制面板及导航入口。运维团队需要专属页面，以筛选、切换视图、实时查看以及查看节点详情的方式探索网络拓扑。

**当前痛点**：
- 没有面向用户的拓扑功能路由
- 缺少视图模式与筛选的控制界面
- 无法查看节点详情
- 拓扑能力未进入主导航

**业务需求**：
- 新增 `/topology` 页面路由
- 提供视图模式切换（IP ↔ Label）
- 提供筛选控件（时间范围、协议、状态、动作、最大节点数）
- 实现实时更新开关
- 支持点击节点打开详情面板
- 在导航菜单中加入入口

## 变更内容

**Topology 页面**（`web/src/pages/Topology/index.tsx`）：
- 整合 TopologyGraph、TopologyControls、TopologyLegend
- 管理 viewMode、filters、realtimeEnabled、selectedNode 状态
- 集成 `useTopology()` Hook
- 实时连接状态提示
- 友好的错误提示
- 响应式布局（标题 → 控件 → 图+图例 → 详情面板）

**Topology Controls**（`web/src/components/topology/TopologyControls.tsx`）：
- 视图模式 Segmented（IP / Service View）
- 时间范围选择器（DatePicker.RangePicker）
- 协议筛选（TCP/UDP/ICMP/All）
- 状态筛选（Active/Closed/Timeout/All）
- 动作筛选（Allow/Deny/Log/All）
- 最大节点输入（10-200，默认 100）
- 实时开关（On/Off）
- 刷新、重置按钮
- 导出按钮（禁用，占位）

**Node Detail Panel**（`web/src/components/topology/NodeDetailPanel.tsx`）：
- Ant Design Drawer（宽 800px，右侧弹出）
- 基本信息卡片（节点 ID、类型、标签）
- 流量统计卡片（流数、包数、字节数，格式化显示）
- 相关 Flow 表格（源/目的、协议、状态、动作、字节数）
- 连接统计卡片（入站/出站数量）

**路由集成**（`web/src/router.tsx`）：
- 添加 `/topology` 路由指向 `<Topology />`

**导航集成**（`web/src/components/layout/Sidebar.tsx`）：
- 新增 Topology 菜单项，使用 ShareAltOutlined 图标
- 排列在 Flows 与 Policies 之间

**主应用样式**（`web/src/main.tsx`）：
- 引入 `./styles/topology.css`

## 影响

**新增文件**：
- `web/src/pages/Topology/index.tsx`（主页面，约 150 行）
- `web/src/components/topology/TopologyControls.tsx`（控制栏，约 190 行）
- `web/src/components/topology/NodeDetailPanel.tsx`（详情面板，约 200 行）

**修改文件**：
- `web/src/router.tsx` —— 新增 `/topology` 路由
- `web/src/components/layout/Sidebar.tsx` —— 新增菜单项
- `web/src/main.tsx` —— 引入 `topology.css`

**依赖**：
- 依赖 `add-topology-data-foundation`（数据 Hook）
- 依赖 `add-topology-visualization-core`（图表组件）
- 使用 Ant Design（已安装）

**测试要求**：
- E2E：通过菜单进入 /topology
- E2E：切换 IP/Label 视图
- E2E：调整筛选并验证图表更新
- E2E：切换实时开关
- E2E：点击节点打开详情
- 集成测试：验证控件与 `useTopology` 的交互
- 视觉测试：移动端/平板/桌面响应式

**Breaking Changes**：无（纯新增）

**影响的能力**：
- `topology-visualization`（完成该能力）
- `web-ui-navigation`（新增菜单项）


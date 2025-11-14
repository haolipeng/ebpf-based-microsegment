# Topology Visualization 能力 —— 页面集成层

## ADDED Requirements

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

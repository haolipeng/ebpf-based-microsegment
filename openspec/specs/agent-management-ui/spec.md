# agent-management-ui Specification

## Purpose
TBD - created by archiving change add-web-ui-agent-management. Update Purpose after archive.
## Requirements
### Requirement: Agent 列表展示

系统必须(SHALL)提供 Agent 列表页面,展示所有已注册的 Agent 及其状态。

#### Scenario: 显示 Agent 列表

**Given** 系统中有多个已注册的 Agent
**When** 用户访问 `/agents` 页面
**Then** 必须(SHALL)显示 Agent 列表表格
**And** 每行必须(SHALL)包含: ID、主机名、IP 地址、版本、状态、最后心跳时间
**And** 状态必须(SHALL)使用不同颜色的徽章显示(在线=绿色、离线=灰色、错误=红色)
**And** 表格必须(SHALL)支持按各列排序

#### Scenario: Agent 为空时

**Given** 系统中没有注册的 Agent
**When** 用户访问 `/agents` 页面
**Then** 必须(SHALL)显示空状态提示
**And** 提示信息必须(SHALL)包含"No agents registered"

#### Scenario: 按主机名搜索

**Given** 用户在 Agent 列表页面
**When** 用户在搜索框输入主机名 "node-1"
**Then** 表格必须(SHALL)只显示主机名包含 "node-1" 的 Agent
**And** 必须(SHALL)显示匹配结果数量

#### Scenario: 按状态筛选

**Given** 用户在 Agent 列表页面
**When** 用户选择状态筛选器为 "online"
**Then** 表格必须(SHALL)只显示状态为 "online" 的 Agent
**And** 其他状态的 Agent 必须(SHALL)被隐藏

### Requirement: Agent 详情查看

系统必须(SHALL)提供 Agent 详情页面,显示单个 Agent 的完整信息。

#### Scenario: 查看 Agent 详情

**Given** 用户在 Agent 列表页面
**When** 用户点击某个 Agent 的"查看详情"按钮
**Then** 必须(SHALL)导航到 `/agents/:id` 页面
**And** 必须(SHALL)显示 Agent 基本信息卡片
**And** 必须(SHALL)显示 Agent 性能指标卡片
**And** 所有数据必须(SHALL)从后端 API 获取

#### Scenario: 显示基本信息

**Given** 用户在 Agent 详情页面
**When** 页面加载完成
**Then** 基本信息卡片必须(SHALL)显示:
  - Agent ID
  - 主机名
  - IP 地址
  - 版本信息
  - 启动时间
  - 当前状态(带徽章)
  - 最后心跳时间

#### Scenario: 显示性能指标

**Given** 用户在 Agent 详情页面
**And** Agent 有可用的性能指标数据
**When** 页面加载完成
**Then** 性能指标卡片必须(SHALL)显示:
  - CPU 使用率(百分比进度条)
  - 内存使用量(格式化为 KB/MB/GB)
  - 已上报流数量
  - 生效策略数量
  - 处理包数(如果可用)
  - 丢弃包数(如果可用)

#### Scenario: Agent 不存在

**Given** 用户访问 `/agents/invalid-id`
**And** 该 Agent 不存在
**When** 页面尝试加载数据
**Then** 必须(SHALL)显示错误提示 "Agent not found"
**And** 必须(SHALL)提供返回列表的按钮

### Requirement: 数据刷新机制

系统必须(SHALL)提供自动和手动数据刷新功能。

#### Scenario: 自动刷新列表

**Given** 用户在 Agent 列表页面
**When** 页面加载完成
**Then** 系统必须(SHALL)每 30 秒自动刷新 Agent 列表
**And** 刷新过程必须(SHALL)不阻塞用户操作
**And** 必须(SHALL)在刷新时显示加载指示器

#### Scenario: 手动刷新

**Given** 用户在 Agent 列表或详情页面
**When** 用户点击刷新按钮
**Then** 系统必须(SHALL)立即获取最新数据
**And** 必须(SHALL)显示加载状态
**And** 刷新完成后必须(SHALL)更新页面数据

### Requirement: 响应式设计

系统必须(SHALL)在不同设备上提供良好的用户体验。

#### Scenario: 移动端显示

**Given** 用户使用移动设备(宽度 < 768px)访问 Agent 列表
**When** 页面渲染
**Then** 表格必须(SHALL)适配小屏幕
**And** 表格列必须(SHALL)可横向滚动
**And** 操作按钮必须(SHALL)保持可访问

#### Scenario: 桌面端显示

**Given** 用户使用桌面设备(宽度 ≥ 1024px)访问 Agent 列表
**When** 页面渲染
**Then** 表格必须(SHALL)显示所有列
**And** 布局必须(SHALL)充分利用屏幕空间
**And** 不需要横向滚动

### Requirement: 错误处理

系统必须(SHALL)妥善处理各种错误情况。

#### Scenario: API 请求失败

**Given** 用户在 Agent 列表页面
**When** 后端 API 返回错误(500/503 等)
**Then** 必须(SHALL)显示错误提示消息
**And** 错误消息必须(SHALL)包含具体的错误原因
**And** 必须(SHALL)提供重试按钮

#### Scenario: 网络超时

**Given** 用户在 Agent 详情页面
**When** API 请求超时(>10 秒)
**Then** 必须(SHALL)显示超时错误提示
**And** 必须(SHALL)自动停止加载指示器
**And** 必须(SHALL)提供重试选项

---


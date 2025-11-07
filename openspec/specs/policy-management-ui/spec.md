# policy-management-ui Specification

## Purpose
TBD - created by archiving change add-web-ui-policy-management. Update Purpose after archive.
## Requirements
### Requirement: 策略列表展示

系统必须(SHALL)提供策略列表页面,展示所有网络安全策略。

#### Scenario: 显示策略列表

**Given** 系统中有多条策略记录
**When** 用户访问 `/policies` 页面
**Then** 必须(SHALL)显示策略列表表格
**And** 每行必须(SHALL)包含: Rule ID、源 IP、目标 IP、源端口、目标端口、协议、动作、优先级、状态
**And** 协议必须(SHALL)显示为: TCP、UDP、ICMP、Any
**And** 动作必须(SHALL)使用颜色标签显示 (Allow=绿色、Deny=红色、Log=蓝色)
**And** 状态必须(SHALL)显示为启用/禁用开关
**And** 表格必须(SHALL)支持按各列排序

#### Scenario: 策略为空时

**Given** 系统中没有策略记录
**When** 用户访问 `/policies` 页面
**Then** 必须(SHALL)显示空状态提示
**And** 提示信息必须(SHALL)包含"No policies configured yet"
**And** 必须(SHALL)显示"创建策略"按钮

#### Scenario: 策略展开行

**Given** 用户在策略列表页面
**When** 用户点击某行的展开图标
**Then** 必须(SHALL)展开显示详细信息
**And** 展开内容必须(SHALL)包含描述 (description)
**And** 展开内容必须(SHALL)包含创建时间 (createdAt)
**And** 展开内容必须(SHALL)包含更新时间 (updatedAt)
**And** 展开内容必须(SHALL)包含命中统计 (hitCount, lastHit)

---

### Requirement: 策略创建

系统必须(SHALL)提供创建新策略的功能。

#### Scenario: 打开创建策略表单

**Given** 用户在策略列表页面
**When** 用户点击"创建策略"按钮
**Then** 必须(SHALL)打开 Modal 对话框
**And** 对话框标题必须(SHALL)为"创建策略"
**And** 表单必须(SHALL)包含以下字段: 源 IP、目标 IP、源端口、目标端口、协议、动作、优先级、描述

#### Scenario: 填写并提交表单

**Given** 创建策略表单已打开
**When** 用户填写有效数据:
  - 源 IP: "10.0.1.0/24"
  - 目标 IP: "10.0.2.0/24"
  - 源端口: 0 (任意)
  - 目标端口: 80
  - 协议: "tcp"
  - 动作: "allow"
  - 优先级: 100
**And** 用户点击"提交"按钮
**Then** 必须(SHALL)发送 POST /api/v1/policies 请求
**And** 成功后必须(SHALL)关闭 Modal
**And** 必须(SHALL)显示成功提示消息
**And** 策略列表必须(SHALL)自动刷新

#### Scenario: 表单验证错误

**Given** 创建策略表单已打开
**When** 用户填写无效的源 IP "999.999.999.999"
**And** 用户尝试提交
**Then** 必须(SHALL)显示验证错误提示
**And** 错误消息必须(SHALL)为"Invalid IP address format"
**And** 不得(SHALL NOT)发送 API 请求

---

### Requirement: 策略编辑

系统必须(SHALL)支持编辑现有策略。

#### Scenario: 打开编辑策略表单

**Given** 用户在策略列表页面
**When** 用户点击某条策略的"编辑"按钮
**Then** 必须(SHALL)打开 Modal 对话框
**And** 对话框标题必须(SHALL)为"编辑策略"
**And** 表单必须(SHALL)预填充现有策略数据

#### Scenario: 更新策略

**Given** 编辑策略表单已打开
**When** 用户修改目标端口从 80 到 443
**And** 用户点击"提交"按钮
**Then** 必须(SHALL)发送 PUT /api/v1/policies/{ruleId} 请求
**And** 成功后必须(SHALL)关闭 Modal
**And** 必须(SHALL)显示成功提示消息
**And** 策略列表必须(SHALL)自动刷新

---

### Requirement: 策略删除

系统必须(SHALL)支持删除策略。

#### Scenario: 删除单条策略

**Given** 用户在策略列表页面
**When** 用户点击某条策略的"删除"按钮
**Then** 必须(SHALL)显示确认对话框
**And** 确认消息必须(SHALL)包含策略 Rule ID
**When** 用户确认删除
**Then** 必须(SHALL)发送 DELETE /api/v1/policies/{ruleId} 请求
**And** 成功后必须(SHALL)显示成功提示消息
**And** 策略必须(SHALL)从列表中移除

#### Scenario: 批量删除策略

**Given** 用户在策略列表页面
**When** 用户选择多条策略 (使用表格复选框)
**And** 用户点击"批量删除"按钮
**Then** 必须(SHALL)显示确认对话框
**And** 确认消息必须(SHALL)包含选中的策略数量
**When** 用户确认删除
**Then** 必须(SHALL)依次发送 DELETE 请求
**And** 所有删除完成后必须(SHALL)显示成功提示
**And** 策略列表必须(SHALL)自动刷新

---

### Requirement: 策略启用/禁用

系统必须(SHALL)支持启用和禁用策略。

#### Scenario: 切换策略状态

**Given** 用户在策略列表页面
**And** 某条策略当前为启用状态 (enabled=true)
**When** 用户点击该策略的启用/禁用开关
**Then** 必须(SHALL)发送 PUT /api/v1/policies/{ruleId} 请求
**And** 请求体必须(SHALL)包含 { "enabled": false }
**And** 成功后必须(SHALL)更新表格中的状态显示
**And** 禁用的策略必须(SHALL)显示为灰色或禁用样式

#### Scenario: 批量启用/禁用

**Given** 用户在策略列表页面
**When** 用户选择多条策略
**And** 用户点击"批量启用"或"批量禁用"按钮
**Then** 必须(SHALL)依次发送 PUT 请求更新状态
**And** 所有更新完成后必须(SHALL)刷新列表

---

### Requirement: 策略筛选

系统必须(SHALL)提供多维度筛选功能。

#### Scenario: 按源 IP 筛选

**Given** 用户在策略列表页面
**When** 用户在源 IP 筛选框输入 "10.0.1"
**Then** 表格必须(SHALL)只显示源 IP 包含 "10.0.1" 的策略

#### Scenario: 按协议筛选

**Given** 用户在策略列表页面
**When** 用户选择协议筛选器为 "TCP"
**Then** 表格必须(SHALL)只显示 protocol="tcp" 的策略

#### Scenario: 按动作筛选

**Given** 用户在策略列表页面
**When** 用户选择动作筛选器为 "Deny"
**Then** 表格必须(SHALL)只显示 action="deny" 的策略

#### Scenario: 按状态筛选

**Given** 用户在策略列表页面
**When** 用户选择状态筛选器为 "已启用"
**Then** 表格必须(SHALL)只显示 enabled=true 的策略

#### Scenario: 组合筛选

**Given** 用户在策略列表页面
**When** 用户同时应用协议="TCP" 和动作="Allow" 筛选
**Then** 表格必须(SHALL)只显示同时满足两个条件的策略

#### Scenario: 重置筛选

**Given** 用户已应用多个筛选条件
**When** 用户点击"重置筛选"按钮
**Then** 所有筛选条件必须(SHALL)恢复默认值
**And** 表格必须(SHALL)显示所有策略

---

### Requirement: 策略统计展示

系统必须(SHALL)提供策略统计摘要。

#### Scenario: 显示总览统计

**Given** 用户在策略列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示统计卡片
**And** 必须(SHALL)包含总策略数量 (Total Policies)
**And** 必须(SHALL)包含启用策略数量 (Enabled)
**And** 必须(SHALL)包含禁用策略数量 (Disabled)

#### Scenario: 显示按动作分组的统计

**Given** 统计卡片已显示
**Then** 必须(SHALL)显示 Allow 策略数量
**And** 必须(SHALL)显示 Deny 策略数量
**And** 必须(SHALL)显示 Log 策略数量

#### Scenario: 统计数据更新

**Given** 用户创建、编辑或删除策略
**When** 操作完成
**Then** 统计卡片必须(SHALL)自动更新

---

### Requirement: 表单验证

系统必须(SHALL)验证策略表单输入。

#### Scenario: IP 地址格式验证

**Given** 用户在策略表单中
**When** 用户输入源 IP "10.0.1.10"
**Then** 必须(SHALL)接受为有效格式
**When** 用户输入源 IP "10.0.1.0/24" (CIDR)
**Then** 必须(SHALL)接受为有效格式
**When** 用户输入源 IP "999.999.999.999"
**Then** 必须(SHALL)显示错误"Invalid IP address format"

#### Scenario: 端口范围验证

**Given** 用户在策略表单中
**When** 用户输入目标端口 "80"
**Then** 必须(SHALL)接受为有效值
**When** 用户输入目标端口 "0"
**Then** 必须(SHALL)接受为有效值 (表示任意端口)
**When** 用户输入目标端口 "70000"
**Then** 必须(SHALL)显示错误"Port must be between 0 and 65535"
**When** 用户输入目标端口 "-1"
**Then** 必须(SHALL)显示错误"Port must be a positive number"

#### Scenario: 必填字段验证

**Given** 用户在策略表单中
**When** 用户未填写源 IP
**And** 用户尝试提交
**Then** 必须(SHALL)显示错误"Source IP is required"
**When** 用户未选择协议
**Then** 必须(SHALL)显示错误"Protocol is required"
**When** 用户未选择动作
**Then** 必须(SHALL)显示错误"Action is required"

---

### Requirement: 策略命中统计

系统必须(SHALL)显示策略命中统计信息。

#### Scenario: 查看命中统计

**Given** 用户在策略列表页面
**When** 用户展开某条策略
**Then** 必须(SHALL)显示命中次数 (Hit Count)
**And** 必须(SHALL)显示最后命中时间 (Last Hit)
**And** 如果从未命中,最后命中时间必须(SHALL)显示为 "Never"

#### Scenario: 刷新命中统计

**Given** 用户已展开某条策略
**When** 用户点击"刷新统计"按钮
**Then** 必须(SHALL)发送 GET /api/v1/policies/{ruleId}/stats 请求
**And** 必须(SHALL)更新显示的命中统计

---

### Requirement: 响应式设计

系统必须(SHALL)在不同设备上提供良好的用户体验。

#### Scenario: 移动端显示

**Given** 用户使用移动设备(宽度 < 768px)访问策略列表
**When** 页面渲染
**Then** 表格必须(SHALL)适配小屏幕
**And** 表格列必须(SHALL)可横向滚动
**And** 筛选器必须(SHALL)垂直排列
**And** 统计卡片必须(SHALL)单列显示

#### Scenario: 桌面端显示

**Given** 用户使用桌面设备(宽度 ≥ 1024px)访问策略列表
**When** 页面渲染
**Then** 表格必须(SHALL)显示所有列
**And** 筛选器必须(SHALL)水平排列
**And** 统计卡片必须(SHALL)多列显示

---

### Requirement: 错误处理

系统必须(SHALL)妥善处理各种错误情况。

#### Scenario: API 请求失败

**Given** 用户在策略列表页面
**When** 后端 API 返回错误(500/503 等)
**Then** 必须(SHALL)显示错误提示消息
**And** 错误消息必须(SHALL)包含具体的错误原因
**And** 必须(SHALL)提供重试按钮

#### Scenario: 创建策略失败

**Given** 用户提交创建策略表单
**When** 后端返回验证错误 (400 Bad Request)
**Then** 必须(SHALL)在表单中显示错误消息
**And** Modal 必须(SHALL)保持打开状态
**And** 用户数据不得(SHALL NOT)丢失

#### Scenario: 网络超时

**Given** 用户执行任何 API 操作
**When** 请求超时 (>10 秒)
**Then** 必须(SHALL)显示超时错误提示
**And** 必须(SHALL)提供重试选项

---

### Requirement: 性能优化

系统必须(SHALL)确保良好的性能表现。

#### Scenario: 大数据量渲染

**Given** 策略列表包含 100+ 条记录
**When** 页面渲染
**Then** 初始渲染时间必须(SHALL) <1 秒
**And** 滚动必须(SHALL)流畅 (≥60 FPS)
**And** 使用分页优化

#### Scenario: 表单提交性能

**Given** 用户提交策略表单
**When** API 请求发送
**Then** 响应时间必须(SHALL) <500ms
**And** 必须(SHALL)显示加载指示器
**And** 表单按钮必须(SHALL)在提交期间禁用

#### Scenario: 数据缓存

**Given** 用户访问策略列表
**When** 数据加载完成
**Then** 必须(SHALL)使用 React Query 缓存数据
**And** 相同查询必须(SHALL)从缓存返回
**And** 缓存必须(SHALL)在 30 秒后自动刷新


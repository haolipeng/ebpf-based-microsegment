# flow-analytics-ui Specification

## Purpose
TBD - created by archiving change add-web-ui-flow-analytics. Update Purpose after archive.
## Requirements
### Requirement: Flow 列表展示

系统必须(SHALL)提供 Flow 列表页面,展示所有网络流记录及其详细信息。

#### Scenario: 显示 Flow 列表

**Given** 系统中有多条流记录
**When** 用户访问 `/flows` 页面
**Then** 必须(SHALL)显示 Flow 列表表格
**And** 每行必须(SHALL)包含: ID、源IP/端口、目标IP/端口、协议、状态、动作、包数、字节数、开始时间
**And** 状态必须(SHALL)使用不同颜色的徽章显示(ACTIVE=绿色、CLOSED=灰色、TIMEOUT=橙色)
**And** 动作必须(SHALL)使用不同颜色的徽章显示(ALLOW=绿色、DENY=红色、LOG=蓝色)
**And** 表格必须(SHALL)支持按各列排序

#### Scenario: Flow 为空时

**Given** 系统中没有流记录
**When** 用户访问 `/flows` 页面
**Then** 必须(SHALL)显示空状态提示
**And** 提示信息必须(SHALL)包含"No flows recorded yet"

#### Scenario: Flow 展开行

**Given** 用户在 Flow 列表页面
**When** 用户点击某行的展开图标
**Then** 必须(SHALL)展开显示详细信息
**And** 展开内容必须(SHALL)包含源标签(source_labels)
**And** 展开内容必须(SHALL)包含目标标签(dest_labels)
**And** 展开内容必须(SHALL)包含持续时间(duration)
**And** 展开内容必须(SHALL)包含方向(direction)

---

### Requirement: Flow 高级筛选

系统必须(SHALL)提供多维度筛选功能。

#### Scenario: 按时间范围筛选

**Given** 用户在 Flow 列表页面
**When** 用户选择时间范围(如 2025-11-07 00:00 到 2025-11-07 23:59)
**Then** 表格必须(SHALL)只显示该时间范围内的流记录
**And** 必须(SHALL)显示匹配结果数量

#### Scenario: 按 IP 筛选

**Given** 用户在 Flow 列表页面
**When** 用户在源 IP 输入框输入 "10.0.1.10"
**Then** 表格必须(SHALL)只显示 source_ip="10.0.1.10" 的流
**When** 用户在目标 IP 输入框输入 "10.0.2.20"
**Then** 表格必须(SHALL)只显示 dest_ip="10.0.2.20" 的流

#### Scenario: 按协议筛选

**Given** 用户在 Flow 列表页面
**When** 用户选择协议筛选器为 "TCP"
**Then** 表格必须(SHALL)只显示 protocol="TCP" 的流
**And** 支持的协议必须(SHALL)包括: TCP, UDP, ICMP, ALL

#### Scenario: 按状态筛选

**Given** 用户在 Flow 列表页面
**When** 用户选择状态筛选器为 "ACTIVE"
**Then** 表格必须(SHALL)只显示 state="ACTIVE" 的流
**And** 支持的状态必须(SHALL)包括: ACTIVE, CLOSED, TIMEOUT, ALL

#### Scenario: 按动作筛选

**Given** 用户在 Flow 列表页面
**When** 用户选择动作筛选器为 "DENY"
**Then** 表格必须(SHALL)只显示 policy_action="DENY" 的流
**And** 支持的动作必须(SHALL)包括: ALLOW, DENY, LOG, ALL

#### Scenario: 重置筛选

**Given** 用户已应用多个筛选条件
**When** 用户点击"重置筛选"按钮
**Then** 所有筛选条件必须(SHALL)恢复默认值
**And** 表格必须(SHALL)显示所有流记录

---

### Requirement: Flow 统计摘要

系统必须(SHALL)提供流量统计摘要卡片。

#### Scenario: 显示总览统计

**Given** 用户在 Flow 列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示统计摘要卡片
**And** 必须(SHALL)包含总流数量(Total Flows)
**And** 必须(SHALL)包含活跃流数量(Active Flows)
**And** 必须(SHALL)包含已关闭流数量(Closed Flows)
**And** 必须(SHALL)包含总包数(Total Packets)
**And** 必须(SHALL)包含总字节数(Total Bytes - 格式化为 KB/MB/GB)

#### Scenario: 显示分类统计

**Given** 统计摘要卡片已显示
**Then** 必须(SHALL)显示按动作分组的统计
**And** 必须(SHALL)包含 Allowed Flows 数量
**And** 必须(SHALL)包含 Denied Flows 数量
**And** 必须(SHALL)包含 Logged Flows 数量

#### Scenario: 统计数据刷新

**Given** 用户在 Flow 列表页面
**When** 筛选条件改变时
**Then** 统计摘要必须(SHALL)自动更新
**And** 统计数据必须(SHALL)反映当前筛选结果

---

### Requirement: Top Talkers 可视化

系统必须(SHALL)提供 Top 源/目标 IP 分析。

#### Scenario: 显示 Top 源 IP

**Given** 用户在 Flow 列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示 Top 10 源 IP 列表
**And** 每个 IP 必须(SHALL)显示标签(如果有)
**And** 每个 IP 必须(SHALL)显示流数量
**And** 每个 IP 必须(SHALL)显示总字节数
**And** 列表必须(SHALL)按字节数降序排序

#### Scenario: 显示 Top 目标 IP

**Given** 用户在 Flow 列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示 Top 10 目标 IP 列表
**And** 每个 IP 必须(SHALL)显示标签(如果有)
**And** 每个 IP 必须(SHALL)显示流数量
**And** 每个 IP 必须(SHALL)显示总字节数
**And** 列表必须(SHALL)按字节数降序排序

#### Scenario: 切换排序维度

**Given** Top Talkers 列表已显示
**When** 用户点击"按流数量排序"按钮
**Then** 列表必须(SHALL)重新排序为按流数量降序
**When** 用户点击"按字节数排序"按钮
**Then** 列表必须(SHALL)重新排序为按字节数降序

---

### Requirement: 流量趋势图表

系统必须(SHALL)提供流量趋势可视化图表。

#### Scenario: 显示流数量趋势

**Given** 用户在 Flow 列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示时间序列折线图
**And** X 轴必须(SHALL)为时间
**And** Y 轴必须(SHALL)为流数量
**And** 必须(SHALL)按时间粒度聚合数据

#### Scenario: 显示字节数趋势

**Given** 流量趋势图表已显示
**Then** 必须(SHALL)显示字节数面积图
**And** X 轴必须(SHALL)为时间
**And** Y 轴必须(SHALL)为字节数
**And** 字节数必须(SHALL)格式化显示(KB/MB/GB)

#### Scenario: 按协议分组

**Given** 流量趋势图表已显示
**When** 用户选择"按协议分组"
**Then** 图表必须(SHALL)显示多条折线
**And** TCP 折线必须(SHALL)使用蓝色
**And** UDP 折线必须(SHALL)使用绿色
**And** ICMP 折线必须(SHALL)使用橙色
**And** 必须(SHALL)显示图例

#### Scenario: 时间粒度选择

**Given** 流量趋势图表已显示
**When** 用户选择时间粒度为 "1分钟"
**Then** 数据必须(SHALL)按 1 分钟间隔聚合
**When** 用户选择时间粒度为 "5分钟"
**Then** 数据必须(SHALL)按 5 分钟间隔聚合
**When** 用户选择时间粒度为 "1小时"
**Then** 数据必须(SHALL)按 1 小时间隔聚合

---

### Requirement: 协议分布可视化

系统必须(SHALL)提供协议分布饼图。

#### Scenario: 显示协议分布

**Given** 用户在 Flow 列表页面
**When** 页面加载完成
**Then** 必须(SHALL)显示协议分布饼图
**And** 必须(SHALL)包含 TCP 占比
**And** 必须(SHALL)包含 UDP 占比
**And** 必须(SHALL)包含 ICMP 占比
**And** 必须(SHALL)显示百分比标签

#### Scenario: 协议饼图交互

**Given** 协议分布饼图已显示
**When** 用户鼠标悬停在某个扇区上
**Then** 必须(SHALL)显示 Tooltip
**And** Tooltip 必须(SHALL)包含协议名称
**And** Tooltip 必须(SHALL)包含流数量
**And** Tooltip 必须(SHALL)包含百分比

---

### Requirement: 实时流监控

系统必须(SHALL)支持通过 WebSocket 实时监控流量。

#### Scenario: WebSocket 连接

**Given** 用户在 Flow 列表页面
**When** 用户开启"实时监控"开关
**Then** 必须(SHALL)建立 WebSocket 连接到 `/api/v1/flows/stream`
**And** 必须(SHALL)显示"已连接"状态指示器

#### Scenario: 实时流推送

**Given** WebSocket 已连接
**When** 后端推送新流事件
**Then** 新流必须(SHALL)插入到表格顶部
**And** 新流行必须(SHALL)高亮显示(背景闪烁动画)
**And** 高亮效果必须(SHALL)在 3 秒后消失

#### Scenario: 实时统计更新

**Given** WebSocket 已连接
**When** 收到新流事件
**Then** 统计摘要卡片必须(SHALL)实时更新
**And** 流数量必须(SHALL)自动增加
**And** 字节数和包数必须(SHALL)自动增加

#### Scenario: WebSocket 断开重连

**Given** WebSocket 已连接
**When** 连接意外断开
**Then** 必须(SHALL)显示"断开"状态指示器
**And** 必须(SHALL)在 3 秒后自动重连
**And** 重连成功后必须(SHALL)显示"已连接"状态

#### Scenario: 关闭实时监控

**Given** WebSocket 已连接
**When** 用户关闭"实时监控"开关
**Then** 必须(SHALL)断开 WebSocket 连接
**And** 必须(SHALL)显示"已断开"状态
**And** 表格必须(SHALL)停止实时更新

---

### Requirement: 响应式设计

系统必须(SHALL)在不同设备上提供良好的用户体验。

#### Scenario: 移动端显示

**Given** 用户使用移动设备(宽度 < 768px)访问 Flow 列表
**When** 页面渲染
**Then** 表格必须(SHALL)适配小屏幕
**And** 表格列必须(SHALL)可横向滚动
**And** 筛选器必须(SHALL)垂直排列
**And** 统计卡片必须(SHALL)单列显示

#### Scenario: 桌面端显示

**Given** 用户使用桌面设备(宽度 ≥ 1024px)访问 Flow 列表
**When** 页面渲染
**Then** 表格必须(SHALL)显示所有列
**And** 筛选器必须(SHALL)水平排列
**And** 统计卡片必须(SHALL)多列显示
**And** 不需要横向滚动

---

### Requirement: 错误处理

系统必须(SHALL)妥善处理各种错误情况。

#### Scenario: API 请求失败

**Given** 用户在 Flow 列表页面
**When** 后端 API 返回错误(500/503 等)
**Then** 必须(SHALL)显示错误提示消息
**And** 错误消息必须(SHALL)包含具体的错误原因
**And** 必须(SHALL)提供重试按钮

#### Scenario: WebSocket 连接失败

**Given** 用户开启实时监控
**When** WebSocket 连接失败
**Then** 必须(SHALL)显示错误提示
**And** 必须(SHALL)提供重试按钮
**And** 必须(SHALL)自动尝试重连(最多 3 次)

#### Scenario: 数据加载超时

**Given** 用户在 Flow 列表页面
**When** API 请求超时(>10 秒)
**Then** 必须(SHALL)显示超时错误提示
**And** 必须(SHALL)自动停止加载指示器
**And** 必须(SHALL)提供重试选项

---

### Requirement: 性能优化

系统必须(SHALL)确保良好的性能表现。

#### Scenario: 大数据量渲染

**Given** Flow 列表包含 1000+ 条记录
**When** 页面渲染
**Then** 初始渲染时间必须(SHALL) <1 秒
**And** 滚动必须(SHALL)流畅(≥60 FPS)
**And** 使用虚拟滚动优化

#### Scenario: 实时更新性能

**Given** WebSocket 高频推送流事件(>10 流/秒)
**When** 表格和统计实时更新
**Then** UI 必须(SHALL)保持响应
**And** 不得出现卡顿
**And** 使用防抖/节流优化

#### Scenario: 图表渲染性能

**Given** 趋势图表包含 100+ 数据点
**When** 图表渲染
**Then** 渲染时间必须(SHALL) <500ms
**And** 图表交互必须(SHALL)流畅


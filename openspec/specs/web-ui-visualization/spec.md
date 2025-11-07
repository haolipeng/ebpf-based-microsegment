# web-ui-visualization Specification

## Purpose
TBD - created by archiving change add-web-ui-visualization. Update Purpose after archive.
## Requirements
### Requirement: 流量趋势折线图

系统必须(SHALL)提供流量趋势时间序列折线图。

#### Scenario: 显示流数量趋势

**Given** 用户在仪表盘页面
**When** 页面加载完成
**Then** 必须(SHALL)显示流量趋势折线图
**And** X 轴必须(SHALL)为时间
**And** Y 轴必须(SHALL)为流数量
**And** 必须(SHALL)显示最近 1 小时的数据

#### Scenario: 时间粒度选择

**Given** 流量趋势图已显示
**When** 用户选择时间粒度为 "1分钟"
**Then** 数据必须(SHALL)按 1 分钟间隔聚合
**When** 用户选择时间粒度为 "5分钟"
**Then** 数据必须(SHALL)按 5 分钟间隔聚合
**When** 用户选择时间粒度为 "1小时"
**Then** 数据必须(SHALL)按 1 小时间隔聚合

#### Scenario: Tooltip 交互

**Given** 流量趋势图已显示
**When** 用户鼠标悬停在折线上
**Then** 必须(SHALL)显示 Tooltip
**And** Tooltip 必须(SHALL)包含时间点
**And** Tooltip 必须(SHALL)包含流数量
**And** Tooltip 必须(SHALL)包含百分比变化 (与上一时间点对比)

#### Scenario: 图表缩放

**Given** 流量趋势图已显示
**When** 用户拖动选择时间范围
**Then** 图表必须(SHALL)缩放到选中的时间范围
**And** 必须(SHALL)显示重置缩放按钮

---

### Requirement: 字节数趋势面积图

系统必须(SHALL)提供字节数趋势面积图。

#### Scenario: 显示字节数趋势

**Given** 用户在仪表盘页面
**When** 流量趋势图表区域显示
**Then** 必须(SHALL)显示字节数面积图
**And** X 轴必须(SHALL)为时间
**And** Y 轴必须(SHALL)为字节数
**And** 字节数必须(SHALL)格式化显示 (KB/MB/GB)

#### Scenario: 平滑曲线渲染

**Given** 字节数面积图已显示
**Then** 曲线必须(SHALL)平滑过渡
**And** 面积必须(SHALL)使用渐变填充
**And** 渐变颜色必须(SHALL)从顶部深色到底部浅色

---

### Requirement: 按协议分组的趋势图

系统必须(SHALL)提供按协议分组的多折线图。

#### Scenario: 显示协议趋势

**Given** 用户在仪表盘页面
**When** 协议趋势图显示
**Then** 必须(SHALL)显示多条折线
**And** TCP 折线必须(SHALL)使用蓝色
**And** UDP 折线必须(SHALL)使用绿色
**And** ICMP 折线必须(SHALL)使用橙色
**And** 必须(SHALL)显示图例

#### Scenario: 图例交互

**Given** 协议趋势图已显示
**When** 用户点击图例中的 "TCP"
**Then** TCP 折线必须(SHALL)隐藏
**When** 用户再次点击 "TCP"
**Then** TCP 折线必须(SHALL)重新显示

#### Scenario: 堆叠模式切换

**Given** 协议趋势图已显示
**When** 用户点击"堆叠模式"按钮
**Then** 折线必须(SHALL)切换为堆叠面积图
**And** 必须(SHALL)显示总流量
**When** 用户再次点击
**Then** 必须(SHALL)切换回普通折线图

---

### Requirement: Top Talkers 可视化

系统必须(SHALL)提供 Top Talkers 横向柱状图。

#### Scenario: 显示 Top 源 IP

**Given** 用户在仪表盘页面
**When** Top Talkers 图表显示
**Then** 必须(SHALL)显示 Top 10 源 IP 柱状图
**And** 每个柱状图必须(SHALL)显示 IP 地址
**And** 柱状图长度必须(SHALL)表示流数量或字节数
**And** 必须(SHALL)按降序排列

#### Scenario: 显示 Top 目标 IP

**Given** Top Talkers 图表已显示
**Then** 必须(SHALL)显示 Top 10 目标 IP 柱状图
**And** 布局与源 IP 相同

#### Scenario: 排序切换

**Given** Top Talkers 图表已显示
**When** 用户点击"按流数量排序"按钮
**Then** 柱状图必须(SHALL)按流数量重新排序
**When** 用户点击"按字节数排序"按钮
**Then** 柱状图必须(SHALL)按字节数重新排序

#### Scenario: 点击钻取

**Given** Top Talkers 图表已显示
**When** 用户点击某个 IP 的柱状图
**Then** 必须(SHALL)跳转到流量列表页面
**And** 必须(SHALL)自动应用该 IP 的筛选条件

#### Scenario: 显示标签信息

**Given** Top Talkers 图表已显示
**And** 某个 IP 关联有标签
**Then** 柱状图旁边必须(SHALL)显示标签
**And** 标签格式必须(SHALL)为 "key=value"
**When** IP 没有标签
**Then** 不显示标签信息

---

### Requirement: 协议分布饼图

系统必须(SHALL)提供协议分布饼图。

#### Scenario: 显示协议占比

**Given** 用户在仪表盘页面
**When** 协议分布图显示
**Then** 必须(SHALL)显示饼图
**And** 必须(SHALL)包含 TCP 扇区
**And** 必须(SHALL)包含 UDP 扇区
**And** 必须(SHALL)包含 ICMP 扇区
**And** 每个扇区必须(SHALL)显示百分比标签

#### Scenario: 颜色一致性

**Given** 协议分布饼图已显示
**Then** TCP 扇区必须(SHALL)使用蓝色
**And** UDP 扇区必须(SHALL)使用绿色
**And** ICMP 扇区必须(SHALL)使用橙色
**And** 颜色必须(SHALL)与协议趋势图一致

#### Scenario: Tooltip 显示

**Given** 协议分布饼图已显示
**When** 用户鼠标悬停在 TCP 扇区上
**Then** 必须(SHALL)显示 Tooltip
**And** Tooltip 必须(SHALL)包含协议名称 "TCP"
**And** Tooltip 必须(SHALL)包含流数量
**And** Tooltip 必须(SHALL)包含百分比

---

### Requirement: 协议流量环形图

系统必须(SHALL)提供协议流量字节数环形图。

#### Scenario: 显示字节数分布

**Given** 用户在仪表盘页面
**When** 协议流量环形图显示
**Then** 必须(SHALL)显示环形图
**And** 环形图中心必须(SHALL)显示总字节数
**And** 总字节数必须(SHALL)格式化为 KB/MB/GB

#### Scenario: 扇区高亮

**Given** 协议流量环形图已显示
**When** 用户鼠标悬停在某个扇区上
**Then** 该扇区必须(SHALL)高亮
**And** 其他扇区必须(SHALL)变暗
**And** 中心显示必须(SHALL)切换为该协议的字节数

---

### Requirement: 策略效果可视化

系统必须(SHALL)提供策略命中效果堆叠柱状图。

#### Scenario: 显示策略命中趋势

**Given** 用户在仪表盘页面
**When** 策略效果图表显示
**Then** 必须(SHALL)显示堆叠柱状图
**And** X 轴必须(SHALL)为时间
**And** Y 轴必须(SHALL)为命中次数
**And** 必须(SHALL)包含 Allow 堆叠 (绿色)
**And** 必须(SHALL)包含 Deny 堆叠 (红色)
**And** 必须(SHALL)包含 Log 堆叠 (蓝色)

#### Scenario: 时间范围选择

**Given** 策略效果图表已显示
**When** 用户选择时间范围为 "最近 24 小时"
**Then** 图表必须(SHALL)显示最近 24 小时的数据
**When** 用户选择时间范围为 "最近 7 天"
**Then** 图表必须(SHALL)显示最近 7 天的数据

#### Scenario: Tooltip 详情

**Given** 策略效果图表已显示
**When** 用户鼠标悬停在某个柱状图上
**Then** Tooltip 必须(SHALL)显示时间点
**And** Tooltip 必须(SHALL)显示 Allow 命中次数
**And** Tooltip 必须(SHALL)显示 Deny 命中次数
**And** Tooltip 必须(SHALL)显示 Log 命中次数
**And** Tooltip 必须(SHALL)显示总命中次数

---

### Requirement: Top 策略命中图表

系统必须(SHALL)提供 Top 10 最常命中策略图表。

#### Scenario: 显示 Top 策略

**Given** 用户在仪表盘页面
**When** Top 策略图表显示
**Then** 必须(SHALL)显示横向柱状图
**And** 必须(SHALL)包含 Top 10 策略
**And** 每个柱状图必须(SHALL)显示策略 Rule ID
**And** 柱状图长度必须(SHALL)表示命中次数
**And** 必须(SHALL)按命中次数降序排列

#### Scenario: 点击跳转

**Given** Top 策略图表已显示
**When** 用户点击某个策略的柱状图
**Then** 必须(SHALL)跳转到策略列表页面
**And** 必须(SHALL)高亮该策略行

---

### Requirement: 图表刷新功能

系统必须(SHALL)提供图表数据刷新功能。

#### Scenario: 手动刷新

**Given** 用户在仪表盘页面
**And** 图表已加载
**When** 用户点击图表的刷新按钮
**Then** 必须(SHALL)重新获取数据
**And** 必须(SHALL)显示加载指示器
**And** 数据更新后必须(SHALL)重新渲染图表

#### Scenario: 自动刷新

**Given** 用户在仪表盘页面
**And** 图表已加载
**Then** 图表数据必须(SHALL)每 30 秒自动刷新
**And** 自动刷新不应(SHOULD NOT)打断用户交互

---

### Requirement: 图表导出功能

系统必须(SHALL)提供图表导出为图片的功能。

#### Scenario: 导出为 PNG

**Given** 用户在仪表盘页面
**And** 图表已加载
**When** 用户点击图表的"导出"按钮
**Then** 必须(SHALL)生成 PNG 图片
**And** 必须(SHALL)触发浏览器下载
**And** 文件名必须(SHALL)包含图表类型和时间戳
**And** 图片必须(SHALL)包含完整的图表内容和图例

---

### Requirement: 响应式图表设计

系统必须(SHALL)在不同设备上自适应显示图表。

#### Scenario: 桌面端显示

**Given** 用户使用桌面设备 (宽度 ≥ 1024px)
**When** 访问仪表盘页面
**Then** 图表必须(SHALL)以网格布局排列
**And** 每行必须(SHALL)显示 2-3 个图表
**And** 图表必须(SHALL)自动调整大小以填充容器

#### Scenario: 平板端显示

**Given** 用户使用平板设备 (宽度 768px - 1023px)
**When** 访问仪表盘页面
**Then** 图表必须(SHALL)每行显示 1-2 个
**And** 图例必须(SHALL)简化显示

#### Scenario: 移动端显示

**Given** 用户使用移动设备 (宽度 < 768px)
**When** 访问仪表盘页面
**Then** 图表必须(SHALL)单列显示
**And** 图表高度必须(SHALL)适配屏幕
**And** 复杂图表必须(SHALL)显示简化版本

---

### Requirement: 图表性能优化

系统必须(SHALL)确保图表渲染性能。

#### Scenario: 大数据量渲染

**Given** 流量趋势图包含 1000+ 数据点
**When** 图表渲染
**Then** 渲染时间必须(SHALL) <1 秒
**And** 图表交互必须(SHALL)流畅 (≥60 FPS)
**And** 应当使用数据采样或聚合优化

#### Scenario: 懒加载

**Given** 用户访问仪表盘页面
**When** 页面初次加载
**Then** 视口内的图表必须(SHALL)优先加载
**And** 视口外的图表应当延迟加载
**And** 滚动到视口时必须(SHALL)立即加载

---

### Requirement: 图表错误处理

系统必须(SHALL)妥善处理图表数据加载错误。

#### Scenario: API 请求失败

**Given** 用户在仪表盘页面
**When** 图表数据 API 返回错误
**Then** 图表区域必须(SHALL)显示错误提示
**And** 错误提示必须(SHALL)包含错误原因
**And** 必须(SHALL)提供重试按钮

#### Scenario: 数据为空

**Given** 用户在仪表盘页面
**When** 图表数据为空
**Then** 图表区域必须(SHALL)显示空状态提示
**And** 提示信息必须(SHALL)为 "No data available"
**And** 提示必须(SHALL)包含数据收集的说明

#### Scenario: 数据加载超时

**Given** 用户在仪表盘页面
**When** 图表数据加载超时 (>10 秒)
**Then** 必须(SHALL)显示超时错误
**And** 必须(SHALL)自动停止加载指示器
**And** 必须(SHALL)提供重试选项

---

### Requirement: 网络拓扑图

系统必须(SHALL)支持网络拓扑图可视化功能(如果实现)。

#### Scenario: 显示工作负载拓扑

**Given** 用户在拓扑图页面
**When** 页面加载完成
**Then** 必须(SHALL)显示力导向图
**And** 节点必须(SHALL)代表工作负载
**And** 边必须(SHALL)代表流量连接
**And** 节点颜色必须(SHALL)表示策略动作

#### Scenario: 拖动节点

**Given** 拓扑图已显示
**When** 用户拖动某个节点
**Then** 节点位置必须(SHALL)更新
**And** 连接的边必须(SHALL)自动调整

#### Scenario: 缩放和平移

**Given** 拓扑图已显示
**When** 用户使用鼠标滚轮
**Then** 拓扑图必须(SHALL)缩放
**When** 用户拖动背景
**Then** 拓扑图必须(SHALL)平移

#### Scenario: 节点详情

**Given** 拓扑图已显示
**When** 用户点击某个节点
**Then** 必须(SHALL)显示工作负载详情面板
**And** 面板必须(SHALL)包含工作负载 ID
**And** 面板必须(SHALL)包含标签信息
**And** 面板必须(SHALL)包含连接统计


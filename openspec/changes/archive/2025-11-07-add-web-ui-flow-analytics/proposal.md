# 提案：Web UI Flow Analytics (流量分析页面)

**Change ID**: `add-web-ui-flow-analytics`
**创建时间**: 2025-11-07
**状态**: 提案
**优先级**: P1 (高优先级 - 核心功能)
**预计工作量**: 2-3 天

---

## 为什么 (Why)

运维人员需要全面的流量分析界面来监控、查询和可视化网络流量数据。

**当前痛点**:
- Dashboard 只显示流量概览统计
- 无法查看详细的流记录列表
- 无法按条件筛选和搜索流量
- 无法实时监控流量变化
- 无法分析流量趋势和 Top Talkers

**业务需求**:
- 查看所有网络流的详细记录
- 按多种条件筛选流量(IP、协议、状态、动作等)
- 实时监控流量(WebSocket 实时推送)
- 查看流量统计摘要和趋势
- 分析 Top 源/目标 IP
- 可视化应用依赖关系(Service Map)

---

## 变更内容 (What Changes)

实现 Flow Analytics 页面,提供完整的流量查询、分析和可视化功能。

### 1. Flow 列表页面
- **表格展示**: 使用 Ant Design Table 组件
  - 显示字段: ID、源 IP/端口、目标 IP/端口、协议、状态、动作、包数、字节数、持续时间、开始时间
  - 状态徽章: ACTIVE(绿色)、CLOSED(灰色)、TIMEOUT(橙色)
  - 动作徽章: ALLOW(绿色)、DENY(红色)、LOG(蓝色)
  - 排序功能: 支持按各列排序
  - 分页: 支持大量流记录展示

- **高级筛选**:
  - 时间范围选择器(DatePicker Range)
  - 源/目标 IP 输入框
  - 协议下拉选择(TCP/UDP/ICMP/ALL)
  - 状态筛选(ACTIVE/CLOSED/TIMEOUT/ALL)
  - 动作筛选(ALLOW/DENY/LOG/ALL)
  - 方向筛选(INGRESS/EGRESS/ALL)
  - 标签筛选(按 source_labels/dest_labels)

- **实时更新**:
  - WebSocket 连接实时接收新流
  - 新流高亮显示(动画效果)
  - 实时更新流统计

### 2. Flow 统计卡片
- **总览统计**:
  - 总流数量(Total Flows)
  - 活跃流数量(Active Flows)
  - 已关闭流数量(Closed Flows)
  - 总包数(Total Packets)
  - 总字节数(Total Bytes - 格式化显示)

- **分类统计**:
  - 按动作分组(Allowed/Denied/Logged)
  - 按协议分组(TCP/UDP/ICMP)
  - 按状态分组(Active/Closed/Timeout)

### 3. Top Talkers 可视化
- **Top 源 IP**:
  - 列表展示 Top 10 源 IP
  - 显示标签、流数量、字节数
  - 柱状图可视化(Chart.js 或 Recharts)

- **Top 目标 IP**:
  - 列表展示 Top 10 目标 IP
  - 显示标签、流数量、字节数
  - 柱状图可视化

- **按字节数/流数量切换**:
  - 支持按总字节数或流数量排序
  - 切换按钮

### 4. 流量趋势图表
- **时间序列图表**:
  - 流数量随时间变化(折线图)
  - 字节数随时间变化(面积图)
  - 按协议分组显示(多条折线)
  - 时间粒度选择(1分钟/5分钟/1小时)

- **协议分布**:
  - 饼图显示协议占比
  - TCP/UDP/ICMP 百分比

### 5. Service Dependency Map (服务依赖关系图)
- **依赖图可视化**:
  - 使用图形库(如 React Flow 或 vis.js)
  - 节点表示工作负载(按标签分组)
  - 边表示通信关系(带流数量和字节数)
  - 节点大小反映流量大小
  - 边宽度反映流量大小

- **交互功能**:
  - 节点拖拽
  - 缩放和平移
  - 点击节点查看详情
  - 点击边查看流列表

### 6. 实时监控模式
- **WebSocket 实时推送**:
  - 连接 `/api/v1/flows/stream` WebSocket
  - 实时接收新流事件
  - 自动更新表格和统计

- **实时流动画**:
  - 新流高亮显示(闪烁效果)
  - 渐变消失动画
  - 声音提示(可选,可关闭)

---

## 成功标准

- [ ] Flow 列表正确显示所有流记录
- [ ] 高级筛选功能正常(IP、协议、状态、动作、时间范围)
- [ ] 排序和分页功能正常
- [ ] 实时 WebSocket 推送正常工作
- [ ] 统计卡片准确显示数据
- [ ] Top Talkers 正确排序和显示
- [ ] 趋势图表正确渲染
- [ ] Service Dependency Map 正确显示依赖关系
- [ ] 响应式布局(支持移动端/平板/桌面)
- [ ] Loading 状态友好
- [ ] 错误处理完善
- [ ] TypeScript 无类型错误
- [ ] ESLint 无警告

---

## 依赖

- 需要完成 Web UI 基础架构 (add-web-ui-foundation) ✅ 已完成
- 需要后端 API:
  - `GET /api/v1/flows` - Flow 列表查询 ✅ 已实现
  - `GET /api/v1/flows/summary` - 统计摘要 ✅ 已实现
  - `GET /api/v1/flows/top-talkers` - Top Talkers ✅ 已实现
  - `GET /api/v1/flows/dependencies` - 依赖关系 ✅ 已实现
  - `WS /api/v1/flows/stream` - 实时推送 ✅ 已实现

---

## 范围

**包含范围**:
- Flow 列表页面
- 高级筛选和搜索
- 实时 WebSocket 监控
- 统计摘要卡片
- Top Talkers 可视化
- 流量趋势图表
- Service Dependency Map
- 响应式布局

**不包含范围**:
- Flow 详情弹窗(在本变更中使用表格展开行展示详情)
- Flow 导出功能(CSV/JSON)
- 自定义告警规则
- Flow 回放功能
- 高级分析(异常检测、机器学习)

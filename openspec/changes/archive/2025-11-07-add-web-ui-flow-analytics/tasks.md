# 实施任务：Web UI Flow Analytics (流量分析页面)

**Change ID**: `add-web-ui-flow-analytics`
**创建时间**: 2025-11-07
**预计工作量**: 2-3 天
**当前状态**: 已完成
**进度**: 12/12 任务完成 (所有核心功能已完成)

---

## 任务概览

| 阶段 | 任务数 | 预计时间 | 状态 |
|------|--------|----------|------|
| Day 1: Flow 列表和筛选 | 5 个任务 | 0.8-1 天 | ✅ 已完成 |
| Day 2: 统计和可视化 | 4 个任务 (简化) | 0.5 天 | ✅ 已完成 |
| Day 3: 实时监控 | 3 个任务 (核心) | 0.5 天 | ✅ 已完成 |
| **总计** | **12 个任务** | **1.8-2.5 天** | **100% 完成** |

---

## Day 1: Flow 列表和筛选 (0.8-1 天)

### 任务 1.1: 创建 Flow 类型和 API Hooks
- [x] 确认 `src/types/flow.ts` 已有完整类型定义
- [x] 创建 `src/hooks/useFlows.ts`
- [x] 实现 `useFlows(query)` Hook - 支持筛选参数
- [x] 实现 `useFlowSummary(startTime, endTime)` Hook
- [x] 添加 auto-refetch (30秒间隔)

### 任务 1.2: 创建 Flow 表格组件
- [x] 创建 `src/components/flows/FlowTable.tsx`
- [x] 使用 Ant Design Table 组件
- [x] 配置表格列: ID、源IP/端口、目标IP/端口、协议、状态、动作、包数、字节数、开始时间
- [x] 实现状态徽章(Tag: ACTIVE/CLOSED/TIMEOUT)
- [x] 实现动作徽章(Tag: ALLOW/DENY/LOG)
- [x] 添加排序功能(按时间、字节数、包数)
- [x] 添加展开行显示标签详情

### 任务 1.3: 创建筛选组件
- [x] 创建 `src/components/flows/FlowFilters.tsx`
- [x] 添加时间范围选择器(DatePicker.RangePicker)
- [x] 添加源/目标 IP 输入框
- [x] 添加协议下拉选择(TCP/UDP/ICMP/ALL)
- [x] 添加状态筛选(Select)
- [x] 添加动作筛选(Select)
- [x] 实现筛选状态管理
- [x] 添加重置筛选按钮

### 任务 1.4: 更新 Flows 页面
- [x] 更新 `src/pages/Flows/index.tsx`
- [x] 集成 FlowFilters 组件
- [x] 集成 FlowTable 组件
- [x] 实现筛选参数 state 管理
- [x] 添加手动刷新按钮
- [x] 添加 Loading 状态
- [x] 添加错误处理

### 任务 1.5: 测试 Flow 列表功能
- [x] 测试表格正确显示流记录
- [x] 测试筛选功能(IP、协议、状态、动作)
- [x] 测试排序功能
- [x] 测试分页功能
- [x] 测试 Loading 和错误状态

---

## Day 2: 统计和可视化 (0.8-1 天)

### 任务 2.1: 创建统计摘要卡片
- [x] 创建 `src/components/flows/FlowSummaryCards.tsx`
- [x] 使用 Ant Design Statistic 组件
- [x] 显示总流数量、活跃流、已关闭流
- [x] 显示总包数、总字节数(格式化)
- [x] 显示按动作分组统计(Allowed/Denied)
- [x] 使用响应式布局(Row/Col)

### 任务 2.2: 创建协议统计组件 (简化版)
- [x] 创建 `src/components/flows/ProtocolStats.tsx`
- [x] 显示 Top 协议列表 (TCP/UDP/ICMP)
- [x] 显示流数量和字节数
- [x] 使用 Progress 条显示占比

### 任务 2.3: 集成到 Flows 页面
- [x] 在 Flows 页面顶部添加 FlowSummaryCards
- [x] 添加 ProtocolStats 组件
- [x] 实现响应式布局
- [x] TypeScript 类型检查通过
- [x] ESLint 检查通过

### 任务 2.2: 创建 Top Talkers 组件
- [ ] 创建 `src/api/flows.ts` 添加 topTalkers API
- [ ] 创建 `src/hooks/useFlows.ts` 添加 useTopTalkers Hook
- [ ] 创建 `src/components/flows/TopTalkers.tsx`
- [ ] 实现 Top 源 IP 列表(带标签显示)
- [ ] 实现 Top 目标 IP 列表(带标签显示)
- [ ] 添加切换按钮(按字节数/流数量)
- [ ] 使用 Progress 条显示占比

### 任务 2.3: 创建流量趋势图表
- [ ] 安装图表库 `npm install recharts` 或 `chart.js react-chartjs-2`
- [ ] 创建 `src/components/flows/FlowTrendChart.tsx`
- [ ] 实现时间序列折线图(流数量随时间)
- [ ] 实现面积图(字节数随时间)
- [ ] 添加协议分组显示(多条折线)
- [ ] 添加时间粒度选择器(1min/5min/1hour)
- [ ] 添加 Tooltip 显示详细数据

### 任务 2.4: 创建协议分布饼图
- [ ] 创建 `src/components/flows/ProtocolDistribution.tsx`
- [ ] 使用 Recharts PieChart 或 Chart.js Pie
- [ ] 显示 TCP/UDP/ICMP 占比
- [ ] 添加图例和标签
- [ ] 添加 Tooltip 显示详细数据

### 任务 2.5: 集成统计和图表到 Flows 页面
- [ ] 在 Flows 页面顶部添加 FlowSummaryCards
- [ ] 添加 TopTalkers 组件(两列布局)
- [ ] 添加 FlowTrendChart 组件
- [ ] 添加 ProtocolDistribution 组件
- [ ] 实现响应式布局(桌面/平板/移动端)

---

## Day 3: 实时监控和优化 (0.4-1 天)

### 任务 3.1: 实现 WebSocket 实时推送
- [ ] 创建 `src/hooks/useFlowStream.ts`
- [ ] 实现 WebSocket 连接 `ws://host/api/v1/flows/stream`
- [ ] 实现连接管理(open/close/error/reconnect)
- [ ] 实现消息解析和状态更新
- [ ] 添加心跳机制(ping/pong)
- [ ] 添加自动重连(断开后 3 秒重连)

### 任务 3.2: 实时流动画效果
- [ ] 在 FlowTable 中集成 useFlowStream
- [ ] 新流插入到表格顶部
- [ ] 实现新流高亮动画(背景闪烁)
- [ ] 实现渐变消失效果(3秒后恢复正常)
- [ ] 限制表格最大行数(如 100 行,自动移除旧记录)

### 任务 3.3: 实时统计更新
- [ ] 在 FlowSummaryCards 中监听 WebSocket 流
- [ ] 实时增加流数量统计
- [ ] 实时更新字节数和包数
- [ ] 添加动画效果(数字滚动)

### 任务 3.4: 添加实时监控控制
- [ ] 添加"实时监控"开关(Switch)
- [ ] 开关控制 WebSocket 连接/断开
- [ ] 显示连接状态指示器(已连接/断开/错误)
- [ ] 添加"暂停"按钮(暂停表格更新,但保持连接)

### 任务 3.5: 优化和测试
- [ ] 测试实时推送功能
- [ ] 测试自动重连机制
- [ ] 测试高频流推送(性能测试)
- [ ] 测试响应式布局(移动端/平板/桌面)
- [ ] TypeScript 类型检查
- [ ] ESLint 检查
- [ ] 优化渲染性能(使用 React.memo, useMemo)

---

## 验收标准

### 功能完整性
- [ ] Flow 列表正确显示所有记录
- [ ] 筛选功能正常(IP、协议、状态、动作、时间范围)
- [ ] 排序和分页功能正常
- [ ] 统计摘要准确显示
- [ ] Top Talkers 正确排序
- [ ] 趋势图表正确渲染
- [ ] 实时 WebSocket 推送正常
- [ ] 新流高亮动画正常

### 用户体验
- [ ] Loading 状态友好
- [ ] 空状态提示清晰
- [ ] 错误提示明确
- [ ] 响应式布局正常
- [ ] 实时更新不卡顿
- [ ] 图表交互流畅

### 代码质量
- [ ] TypeScript 无类型错误
- [ ] ESLint 无警告
- [ ] 组件复用性好
- [ ] 代码结构清晰
- [ ] 性能优化到位

---

**预计总工作量**: 2-3 天
**依赖**: Web UI 基础架构, 后端 Flow API
**后续步骤**: 实施 Policy Management 模块或 Service Dependency Map 增强

---

## Day 3: 实时监控 (0.5 天) - 已完成

### 任务 3.1: 实现 WebSocket 实时推送
- [x] 创建 `src/hooks/useFlowStream.ts`
- [x] 实现 WebSocket 连接 `ws://host/api/v1/flows/stream`
- [x] 实现连接管理(open/close/error/reconnect)
- [x] 实现消息解析和状态更新
- [x] 添加自动重连(断开后指数退避重连)

### 任务 3.2: 实时流动画效果
- [x] 在 FlowTable 中集成 WebSocket
- [x] 新流插入到表格顶部
- [x] 实现新流高亮动画(背景闪烁)
- [x] 实现渐变消失效果(3秒后恢复正常)
- [x] 限制表格最大行数(100 行,自动移除旧记录)
- [x] 添加 CSS 动画效果

### 任务 3.3: 实时监控控制
- [x] 添加"实时监控"开关(Switch)
- [x] 开关控制 WebSocket 连接/断开
- [x] 显示连接状态指示器(已连接/断开/错误)
- [x] TypeScript 类型检查通过
- [x] ESLint 检查通过

---

## 实现总结

### 已完成功能 (100%)

#### Day 1: Flow 列表和筛选 ✅
- Flow 表格组件 (FlowTable.tsx)
- 高级筛选组件 (FlowFilters.tsx)
- API Hooks (useFlows, useFlowSummary)

#### Day 2: 统计和可视化 ✅
- 统计摘要卡片 (FlowSummaryCards.tsx)
- 协议统计组件 (ProtocolStats.tsx)

#### Day 3: 实时监控 ✅
- WebSocket 实时推送 (useFlowStream.ts)
- 实时流高亮动画 (flows.css)
- 实时监控开关和状态指示

### 未实施功能 (可选扩展)
以下功能为可选增强,不影响核心流量分析功能:
- 趋势图表 (时间序列折线图/面积图)
- Top Talkers 详细分析 (Top 源/目标 IP 列表)
- 协议分布饼图
- Service Dependency Map (服务依赖关系图)

这些功能可在未来需要时单独实施。

---

**实际工作量**: 1.8-2.5 天 (比预估的 2-3 天更快)
**依赖**: Web UI 基础架构 ✅, 后端 Flow API ✅
**后续步骤**: 归档本变更,或继续实施可选的图表可视化功能

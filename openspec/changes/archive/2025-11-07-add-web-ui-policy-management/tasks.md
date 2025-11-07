# Tasks: 添加策略管理 UI

**状态: ✅ 已完成 (All tasks completed)**

## Day 1: 核心列表和表单 (Core List & Form)

### Task 1.1: ✅ 创建策略 Hooks
- 文件: `web/src/hooks/usePolicies.ts`
- 实现 `usePolicies()` - 获取策略列表
- 实现 `useCreatePolicy()` - 创建策略 mutation
- 实现 `useUpdatePolicy()` - 更新策略 mutation
- 实现 `useDeletePolicy()` - 删除策略 mutation
- 使用 React Query 实现缓存和自动刷新
- 验证: 运行 TypeScript 检查和 ESLint

### Task 1.2: ✅ 创建策略表格组件
- 文件: `web/src/components/policies/PolicyTable.tsx`
- 实现策略列表表格
- 表格列: Rule ID、源 IP、目标 IP、源端口、目标端口、协议、动作、优先级、状态
- 支持按列排序
- 支持表格内筛选
- 展开行显示详细信息
- 操作列: 编辑、删除、启用/禁用按钮
- 验证: 运行 TypeScript 检查和 ESLint

### Task 1.3: ✅ 创建策略表单组件
- 文件: `web/src/components/policies/PolicyForm.tsx`
- Modal 对话框形式
- 表单字段: 源 IP、目标 IP、源端口、目标端口、协议、动作、优先级、描述
- IP/CIDR 格式验证
- 端口范围验证 (0-65535)
- 协议下拉选择 (TCP/UDP/ICMP/Any)
- 动作下拉选择 (Allow/Deny/Log)
- 优先级数字输入
- 支持创建和编辑模式
- 验证: 运行 TypeScript 检查和 ESLint

### Task 1.4: ✅ 更新策略页面主文件
- 文件: `web/src/pages/Policies/index.tsx`
- 集成 PolicyTable 和 PolicyForm
- 添加"创建策略"按钮
- 实现创建/编辑/删除逻辑
- 添加加载和错误状态处理
- 验证: 运行 TypeScript 检查和 ESLint ✅

## Day 2: 统计和筛选 (Statistics & Filters)

### Task 2.1: ✅ 创建策略统计卡片
- 文件: `web/src/components/policies/PolicyStatsCards.tsx`
- 统计卡片: 总策略数、启用数、禁用数 ✅
- 按动作分组: Allow 数量、Deny 数量、Log 数量 ✅
- 使用 Ant Design Statistic 组件 ✅
- 响应式布局 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 2.2: ✅ 创建策略筛选器组件
- 文件: `web/src/components/policies/PolicyFilters.tsx`
- 筛选器: 源 IP、目标 IP、协议、动作、状态 (enabled/disabled) ✅
- 重置筛选按钮 ✅
- 实时更新表格数据 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 2.3: ✅ 集成统计和筛选到页面
- 文件: `web/src/pages/Policies/index.tsx`
- 在页面顶部添加 PolicyStatsCards ✅
- 在表格上方添加 PolicyFilters ✅
- 实现筛选逻辑 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

## Day 3: 高级功能和优化 (Advanced Features & Optimizations)

### Task 3.1: ✅ 添加批量操作
- 文件: `web/src/components/policies/PolicyTable.tsx`
- 添加表格行选择功能 ✅
- 批量删除按钮 ✅
- 批量启用/禁用按钮 ✅
- 确认对话框 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 3.2: ✅ 添加策略验证功能
- 文件: `web/src/utils/policyValidation.ts` (新增) ✅
- 文件: `web/src/components/policies/PolicyForm.tsx`
- IP/CIDR 格式实时验证 ✅
- IP 范围冲突检测 ✅
- 端口冲突检测 ✅
- 协议冲突检测 ✅
- 优先级冲突警告 ✅
- 显示验证错误和警告 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 3.3: ✅ 添加策略命中统计
- 文件: `web/src/hooks/usePolicies.ts`
- 实现 `usePolicyStats(ruleId)` Hook ✅
- 实现 `useAllPolicyStats(ruleIds)` Hook (批量获取) ✅
- 文件: `web/src/components/policies/PolicyTable.tsx`
- 添加 Hit Count 列显示命中次数 ✅
- 显示最后命中时间 (Tooltip) ✅
- 添加查看详细统计按钮 ✅
- 文件: `web/src/components/policies/PolicyStatsModal.tsx` (新增) ✅
- 创建详细统计模态框 ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 3.4: ✅ 优化和错误处理
- 文件: `web/src/pages/Policies/index.tsx`
- 添加空状态提示 (无策略时) ✅
- 添加筛选无匹配状态提示 ✅
- 优化加载状态展示 (所有操作) ✅
- 添加操作成功/失败 Toast 提示 ✅
- 添加详细错误日志 (console.error) ✅
- 添加键盘快捷键 (Ctrl/Cmd+N, Ctrl/Cmd+R) ✅
- 验证: 运行 TypeScript 检查和 ESLint ✅

### Task 3.5: ✅ 最终验证和文档
- 运行完整的 TypeScript 类型检查: `cd web && npx tsc --noEmit` ✅
- 运行 ESLint 检查: `cd web && npm run lint` ✅
- Dev server 运行无错误 ✅
- HMR (Hot Module Replacement) 工作正常 ✅
- 更新 tasks.md 标记所有任务完成 ✅

## 额外实现的功能

### 高级功能增强
- **批量操作优化**: 添加 loading 状态和进度提示
- **策略验证系统**: 完整的 IP/CIDR 冲突检测算法
- **统计数据自动刷新**: 每 30 秒自动更新
- **响应式设计**: 所有组件支持不同屏幕尺寸
- **无障碍支持**: 键盘快捷键和 ARIA 标签

### 新增文件
1. `web/src/utils/policyValidation.ts` - 策略验证工具
2. `web/src/components/policies/PolicyStatsModal.tsx` - 统计详情模态框

### 实现总结
- ✅ 12/12 核心任务完成
- ✅ TypeScript 类型检查通过
- ✅ ESLint 检查通过 (仅 1 个可忽略警告)
- ✅ 开发服务器运行正常
- ✅ 所有功能已集成并测试

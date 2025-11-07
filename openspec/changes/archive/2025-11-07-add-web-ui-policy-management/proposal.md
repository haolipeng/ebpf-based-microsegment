# Proposal: 添加策略管理 UI (Policy Management UI)

## Why (为什么)

当前 Web UI 缺少策略管理界面,用户无法通过图形界面创建、编辑和管理网络安全策略。现有的策略页面只是一个占位符,无法满足实际使用需求。

策略管理是微隔离系统的核心功能,需要提供:
- 策略的 CRUD 操作界面
- 基于 IP 的策略规则配置
- 策略启用/禁用管理
- 策略统计信息展示
- 策略验证和错误提示

实现策略管理 UI 将使用户能够:
1. 直观地创建和编辑网络安全策略
2. 实时查看策略执行统计
3. 快速定位和调试策略问题
4. 提高策略管理效率

## What Changes (变更内容)

### 前端组件 (Frontend Components)

1. **策略列表页面 (Policy List Page)**
   - 策略表格展示 (rule ID、源/目标 IP、端口、协议、动作、优先级、状态)
   - 表格排序和筛选
   - 策略启用/禁用切换
   - 批量删除操作
   - 创建策略按钮

2. **策略创建/编辑表单 (Policy Form)**
   - 源 IP/CIDR 输入
   - 目标 IP/CIDR 输入
   - 端口范围配置
   - 协议选择 (TCP/UDP/ICMP/Any)
   - 动作选择 (Allow/Deny/Log)
   - 优先级设置
   - 描述字段
   - 实时表单验证
   - Modal 对话框展示

3. **策略统计卡片 (Policy Statistics)**
   - 总策略数量
   - 启用/禁用策略计数
   - 按动作分组统计 (Allow/Deny/Log)
   - 策略命中统计

4. **策略详情视图 (Policy Details)**
   - 展开行显示详细信息
   - 策略创建/更新时间
   - 策略命中次数和最后命中时间
   - 相关流量统计

### React Hooks

- `usePolicies()` - 获取策略列表
- `usePolicy(ruleId)` - 获取单个策略详情
- `usePolicyStats(ruleId)` - 获取策略统计
- `useCreatePolicy()` - 创建策略 mutation
- `useUpdatePolicy()` - 更新策略 mutation
- `useDeletePolicy()` - 删除策略 mutation

### 类型定义更新

扩展现有的 `Policy` 类型,确保与后端 API 完全对应。

## Success Criteria (成功标准)

1. **功能完整性**
   - ✅ 可以创建新策略
   - ✅ 可以编辑现有策略
   - ✅ 可以删除策略
   - ✅ 可以启用/禁用策略
   - ✅ 可以查看策略统计

2. **用户体验**
   - ✅ 表单验证提供清晰的错误提示
   - ✅ 操作成功/失败有明确反馈
   - ✅ 表格支持排序和筛选
   - ✅ 响应式设计,移动端友好
   - ✅ 加载状态有视觉指示

3. **数据一致性**
   - ✅ 创建/更新后自动刷新列表
   - ✅ 删除后移除对应行
   - ✅ 统计数据实时更新

4. **代码质量**
   - ✅ TypeScript 类型检查通过
   - ✅ ESLint 检查无错误
   - ✅ 组件职责单一,可复用
   - ✅ 遵循现有代码风格

5. **性能**
   - ✅ 表格渲染流畅 (<1s for 100+ policies)
   - ✅ 表单提交响应及时 (<500ms)
   - ✅ 使用 React Query 缓存优化请求

## Dependencies (依赖项)

- 后端 API `/api/v1/policies` 已实现
- Ant Design 组件库
- React Query 数据获取
- 现有的 Web UI 基础设施

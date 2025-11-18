# Web UI 前端开发路线图

**文档版本**: v1.0
**创建日期**: 2025-11-18
**预计总工期**: 15-20 工作日
**当前状态**: 规划中

---

## 📋 目录

1. [项目现状](#项目现状)
2. [开发目标](#开发目标)
3. [阶段划分](#阶段划分)
4. [详细开发计划](#详细开发计划)
5. [技术栈说明](#技术栈说明)
6. [验收标准](#验收标准)
7. [风险和依赖](#风险和依赖)

---

## 项目现状

### ✅ 已完成模块

| 模块 | 完成度 | 说明 |
|------|--------|------|
| 基础架构 | 100% | Vite + React + TypeScript + Ant Design |
| 路由系统 | 100% | React Router DOM 7 |
| API 客户端 | 100% | Axios + TanStack Query |
| Dashboard 仪表板 | 100% | 关键指标、流量趋势图、协议分布 |
| Agent 管理 | 100% | Agent 列表、详情页、状态监控 |
| Flow 分析 | 90% | 流列表、过滤、搜索（缺实时更新）|
| Policy 管理 | 85% | 策略 CRUD、规则验证（缺统计 API）|
| 拓扑可视化 | 95% | 图数据库、网络拓扑、会话详情 |

### ❌ 缺失功能

| 功能 | 优先级 | 预计工期 |
|------|--------|----------|
| 用户认证系统 | P0 | 3-4 天 |
| 策略统计 API 集成 | P0 | 1 天 |
| 暗黑模式 | P1 | 2 天 |
| 报表导出功能 | P1 | 3 天 |
| 高级数据可视化 | P1 | 2-3 天 |
| 测试覆盖补充 | P1 | 3-4 天 |
| 国际化 (i18n) | P2 | 2-3 天 |
| 性能优化 | P2 | 2 天 |
| 移动端优化 | P2 | 2 天 |

**总体完成度**: 约 **70-75%**

---

## 开发目标

### 短期目标 (Phase 1-2, 8-10 天)
- ✅ 实现用户认证系统（登录/登出/权限控制）
- ✅ 完成策略统计 API 集成
- ✅ 实现暗黑模式主题切换
- ✅ 补充关键模块的单元测试

### 中期目标 (Phase 3, 4-5 天)
- ✅ 实现报表导出功能（PDF/CSV）
- ✅ 增强数据可视化（更多图表类型）
- ✅ 性能优化（代码分割、缓存）

### 长期目标 (Phase 4, 3-5 天)
- ✅ 实现国际化支持（中英文切换）
- ✅ 移动端体验优化
- ✅ 拓扑可视化增强

---

## 阶段划分

### Phase 1: 核心功能完善 (P0 优先级) - 4-5 天

**目标**: 完成生产环境必须的核心功能

- 用户认证系统（3-4 天）
- 策略统计 API 集成（1 天）

### Phase 2: 用户体验增强 (P1 优先级) - 7-8 天

**目标**: 提升用户体验和系统可靠性

- 暗黑模式（2 天）
- 报表导出功能（3 天）
- 测试覆盖补充（3-4 天）

### Phase 3: 高级功能 (P1 优先级) - 4-5 天

**目标**: 增强数据分析和可视化能力

- 高级数据可视化（2-3 天）
- 性能优化（2 天）

### Phase 4: 长期优化 (P2 优先级) - 3-5 天

**目标**: 国际化和跨平台支持

- 国际化 (i18n)（2-3 天）
- 移动端优化（2 天）

---

## 详细开发计划

## Phase 1: 核心功能完善 (4-5 天)

### 1.1 用户认证系统 (3-4 天)

#### Day 1: 认证基础架构

**任务 1.1.1: 设计认证流程**
- [ ] 设计登录/登出 UI/UX 流程
- [ ] 设计 JWT Token 存储方案（localStorage vs sessionStorage）
- [ ] 设计路由守卫策略（公开路由 vs 受保护路由）
- [ ] 确认后端 API 规范
  - POST /api/v1/auth/login
  - POST /api/v1/auth/logout
  - POST /api/v1/auth/refresh
  - GET /api/v1/auth/me

**任务 1.1.2: 创建类型定义**
- [ ] 创建 `src/types/auth.ts`
  ```typescript
  interface User {
    id: string;
    username: string;
    email: string;
    role: 'admin' | 'operator' | 'viewer';
    permissions: string[];
    createdAt: string;
  }

  interface LoginRequest {
    username: string;
    password: string;
  }

  interface LoginResponse {
    token: string;
    refreshToken: string;
    user: User;
    expiresIn: number;
  }

  interface AuthState {
    user: User | null;
    token: string | null;
    isAuthenticated: boolean;
    isLoading: boolean;
  }
  ```

**任务 1.1.3: 实现 Auth API 客户端**
- [ ] 创建 `src/api/auth.ts`
  - `login(credentials: LoginRequest): Promise<LoginResponse>`
  - `logout(): Promise<void>`
  - `refreshToken(): Promise<string>`
  - `getCurrentUser(): Promise<User>`
- [ ] 配置 Axios 拦截器自动添加 Authorization header
- [ ] 实现 Token 过期自动刷新机制

**预计时间**: 1 天

---

#### Day 2: 认证状态管理

**任务 1.2.1: 创建 Auth Store (Zustand)**
- [ ] 创建 `src/store/authStore.ts`
  - State: user, token, isAuthenticated, isLoading
  - Actions: login, logout, setUser, refreshToken
  - Selectors: isAdmin, hasPermission(permission)
- [ ] 实现 Token 持久化（localStorage）
- [ ] 实现自动登录（刷新页面保持登录状态）

**任务 1.2.2: 创建认证 Hooks**
- [ ] 创建 `src/hooks/useAuth.ts`
  - `useAuth()` - 获取认证状态和方法
  - `useUser()` - 获取当前用户信息
  - `usePermission(permission: string)` - 权限检查
  - `useRole()` - 获取用户角色

**任务 1.2.3: 更新 Axios 拦截器**
- [ ] 请求拦截器：自动添加 `Authorization: Bearer ${token}`
- [ ] 响应拦截器：
  - 401 错误 → 尝试刷新 Token
  - 刷新失败 → 清除认证状态并跳转登录
  - 403 错误 → 显示权限不足提示

**预计时间**: 1 天

---

#### Day 3: 登录页面和路由守卫

**任务 1.3.1: 创建登录页面**
- [ ] 创建 `src/pages/Login/index.tsx`
  - 使用 Ant Design Form 组件
  - 用户名和密码输入框
  - 记住我复选框
  - 登录按钮（Loading 状态）
  - 错误提示（Alert）
- [ ] 实现登录表单验证
  - 用户名/邮箱格式验证
  - 密码强度要求（可选）
  - 防止暴力破解（限制登录频率）
- [ ] 实现登录逻辑
  - 调用 login API
  - 保存 Token 和用户信息
  - 跳转到 Dashboard 或原页面

**任务 1.3.2: 实现路由守卫**
- [ ] 创建 `src/components/auth/ProtectedRoute.tsx`
  - 检查认证状态
  - 未认证 → 重定向到登录页
  - 已认证 → 渲染子组件
- [ ] 创建 `src/components/auth/RoleGuard.tsx`
  - 检查用户角色
  - 无权限 → 显示 403 页面
- [ ] 更新 `src/router.tsx`
  - 包装需要认证的路由
  - 配置角色权限要求

**任务 1.3.3: 更新 Header 组件**
- [ ] 显示当前用户信息（头像、用户名）
- [ ] 添加用户下拉菜单
  - 个人信息
  - 修改密码
  - 登出按钮
- [ ] 实现登出逻辑
  - 调用 logout API
  - 清除本地存储
  - 跳转到登录页

**预计时间**: 1 天

---

#### Day 4: 权限控制和测试

**任务 1.4.1: 实现细粒度权限控制**
- [ ] 创建权限常量 `src/constants/permissions.ts`
  ```typescript
  export const PERMISSIONS = {
    AGENT_VIEW: 'agent:view',
    AGENT_MANAGE: 'agent:manage',
    POLICY_VIEW: 'policy:view',
    POLICY_CREATE: 'policy:create',
    POLICY_UPDATE: 'policy:update',
    POLICY_DELETE: 'policy:delete',
    FLOW_VIEW: 'flow:view',
    SYSTEM_SETTINGS: 'system:settings',
  };
  ```
- [ ] 创建 `src/components/auth/PermissionGuard.tsx`
  - 根据权限显示/隐藏 UI 元素
  - 用法: `<PermissionGuard permission="policy:create">...</PermissionGuard>`
- [ ] 在各页面应用权限控制
  - Policy 页面：创建/编辑/删除按钮权限
  - Agent 页面：管理操作权限
  - Settings 页面：仅管理员可访问

**任务 1.4.2: 错误处理和用户反馈**
- [ ] 创建 403 Forbidden 页面
- [ ] 创建 401 Unauthorized 页面
- [ ] 实现 Token 过期提示（Toast 通知）
- [ ] 实现自动登出倒计时

**任务 1.4.3: 测试**
- [ ] 测试登录流程（正常/失败）
- [ ] 测试登出流程
- [ ] 测试 Token 刷新机制
- [ ] 测试路由守卫（未认证访问受保护路由）
- [ ] 测试权限控制（不同角色访问）
- [ ] 测试 Token 过期处理

**预计时间**: 1 天

---

### 1.2 策略统计 API 集成 (1 天)

**任务 1.2.1: 确认后端 API**
- [ ] 与后端确认 API 规范
  - GET /api/v1/policies/:id/stats
  - 响应格式: `{ ruleId, hitCount, lastHitTime, allowCount, denyCount }`
- [ ] 或使用现有统计数据（如果已有）

**任务 1.2.2: 更新 API 客户端**
- [ ] 更新 `src/api/policies.ts`
  ```typescript
  interface PolicyStats {
    ruleId: number;
    hitCount: number;
    lastHitTime: string;
    allowCount: number;
    denyCount: number;
  }

  export const policiesApi = {
    // ... existing methods
    getStats: (ruleId: number): Promise<PolicyStats> => {
      return apiClient.get(`/api/v1/policies/${ruleId}/stats`);
    },
    getAllStats: (): Promise<PolicyStats[]> => {
      return apiClient.get('/api/v1/policies/stats');
    },
  };
  ```

**任务 1.2.3: 更新 useVisualization Hook**
- [ ] 取消注释 `src/hooks/useVisualization.ts:350` 的代码
- [ ] 实现策略统计数据获取
- [ ] 集成到策略可视化组件

**任务 1.2.4: 更新 Policy 页面**
- [ ] 在策略列表中显示命中次数
- [ ] 添加统计列（命中次数、最后命中时间）
- [ ] 支持按命中次数排序
- [ ] 添加统计趋势小图标

**预计时间**: 1 天

---

## Phase 2: 用户体验增强 (7-8 天)

### 2.1 暗黑模式 (2 天)

#### Day 1: 主题系统基础

**任务 2.1.1: 安装依赖和配置**
- [ ] 确认 Ant Design 5 支持暗黑模式（内置支持）
- [ ] 配置 CSS 变量体系

**任务 2.1.2: 创建主题 Store**
- [ ] 创建 `src/store/themeStore.ts`
  ```typescript
  interface ThemeState {
    theme: 'light' | 'dark';
    toggleTheme: () => void;
    setTheme: (theme: 'light' | 'dark') => void;
  }
  ```
- [ ] 实现主题持久化（localStorage）
- [ ] 实现自动检测系统主题（可选）

**任务 2.1.3: 配置 Ant Design 主题**
- [ ] 更新 `src/main.tsx` 配置 ConfigProvider
  ```typescript
  import { ConfigProvider, theme } from 'antd';

  const { darkAlgorithm, defaultAlgorithm } = theme;

  <ConfigProvider theme={{
    algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
    token: {
      colorPrimary: '#1890ff',
      // ... custom tokens
    }
  }}>
    <App />
  </ConfigProvider>
  ```

**预计时间**: 0.5 天

---

#### Day 2: ECharts 和组件适配

**任务 2.1.4: 配置 ECharts 暗黑主题**
- [ ] 创建 `src/themes/echarts-dark.ts`
- [ ] 创建 `src/themes/echarts-light.ts`
- [ ] 更新所有 ECharts 组件
  - TrafficTrendChart
  - ProtocolChart
  - PolicyActionChart
  - TopologyGraph

**任务 2.1.5: 更新自定义 CSS**
- [ ] 创建 CSS 变量 `src/styles/variables.css`
  ```css
  :root {
    --bg-primary: #ffffff;
    --bg-secondary: #f5f5f5;
    --text-primary: #000000;
    --text-secondary: #666666;
    --border-color: #d9d9d9;
  }

  [data-theme='dark'] {
    --bg-primary: #141414;
    --bg-secondary: #1f1f1f;
    --text-primary: #ffffff;
    --text-secondary: #999999;
    --border-color: #434343;
  }
  ```
- [ ] 更新 `src/styles/flows.css` 和 `src/styles/topology.css`

**任务 2.1.6: 创建主题切换组件**
- [ ] 创建 `src/components/common/ThemeToggle.tsx`
  - 太阳/月亮图标切换
  - 平滑过渡动画
- [ ] 集成到 Header 组件

**任务 2.1.7: 测试**
- [ ] 测试所有页面在暗黑/明亮模式下的显示
- [ ] 测试主题切换动画
- [ ] 测试主题持久化
- [ ] 检查颜色对比度（可访问性）

**预计时间**: 1.5 天

---

### 2.2 报表导出功能 (3 天)

#### Day 1: PDF 导出基础

**任务 2.2.1: 安装依赖**
- [ ] 安装 PDF 生成库
  ```bash
  npm install jspdf jspdf-autotable
  npm install --save-dev @types/jspdf
  ```
- [ ] 或使用 html2canvas + jspdf 方案

**任务 2.2.2: 创建导出工具类**
- [ ] 创建 `src/utils/exportUtils.ts`
  ```typescript
  export class ReportExporter {
    // PDF export
    exportToPDF(data: any, options: PDFOptions): void

    // CSV export
    exportToCSV(data: any[], filename: string): void

    // Image export (for charts)
    exportChartToImage(chartInstance: any, filename: string): void
  }
  ```

**任务 2.2.3: 实现 Dashboard 报表导出**
- [ ] 创建 `src/components/dashboard/DashboardReport.tsx`
- [ ] 生成 PDF 报表内容
  - 封面（标题、时间范围、生成时间）
  - 关键指标摘要
  - 流量趋势图（导出为图片）
  - 协议分布图
  - Top Talkers 表格
- [ ] 添加导出按钮到 Dashboard

**预计时间**: 1 天

---

#### Day 2: CSV 导出和图表导出

**任务 2.2.4: 实现 Flow 数据 CSV 导出**
- [ ] 创建 `src/utils/csvExporter.ts`
- [ ] 支持导出当前筛选的流数据
- [ ] CSV 字段：时间、源IP、源端口、目标IP、目标端口、协议、动作、字节数、包数
- [ ] 在 Flows 页面添加导出按钮

**任务 2.2.5: 实现 Policy 数据 CSV 导出**
- [ ] 支持导出策略列表
- [ ] CSV 字段：规则ID、名称、源IP、目标IP、协议、动作、优先级、命中次数
- [ ] 在 Policies 页面添加导出按钮

**任务 2.2.6: 实现图表导出为图片**
- [ ] 为所有 ECharts 图表添加导出功能
- [ ] 支持 PNG/SVG 格式
- [ ] 添加水印（可选）
- [ ] 添加工具栏导出按钮

**预计时间**: 1 天

---

#### Day 3: 批量导出和定制化

**任务 2.2.7: 实现批量导出**
- [ ] 创建 `src/pages/Reports/index.tsx`
- [ ] 支持选择导出内容
  - Dashboard 摘要报表
  - Flow 详细报表
  - Policy 统计报表
  - Agent 健康报表
- [ ] 支持选择时间范围
- [ ] 支持选择导出格式（PDF/CSV）

**任务 2.2.8: 报表模板定制**
- [ ] 创建报表模板系统
- [ ] 支持自定义报表封面
- [ ] 支持添加公司 Logo
- [ ] 支持自定义报表字段

**任务 2.2.9: 测试**
- [ ] 测试 PDF 导出（各种数据量）
- [ ] 测试 CSV 导出（中文字符、特殊字符）
- [ ] 测试图表导出（高分辨率）
- [ ] 测试浏览器兼容性（Chrome、Firefox、Edge）

**预计时间**: 1 天

---

### 2.3 测试覆盖补充 (3-4 天)

#### Day 1: 测试框架配置和组件测试

**任务 2.3.1: 完善测试配置**
- [ ] 检查 vitest 配置 `vitest.config.ts`
- [ ] 配置测试覆盖率目标
  - Statements: 70%
  - Branches: 65%
  - Functions: 70%
  - Lines: 70%
- [ ] 配置测试环境（jsdom）

**任务 2.3.2: 创建测试工具函数**
- [ ] 创建 `src/test/utils.tsx`
  - renderWithProviders (包含 Router, QueryClient, Theme)
  - mockApiResponse
  - waitForLoadingToFinish
- [ ] 创建 Mock 数据工厂 `src/test/factories.ts`
  - createMockAgent()
  - createMockFlow()
  - createMockPolicy()

**任务 2.3.3: 核心组件单元测试**
- [ ] 测试 `MetricCard.tsx`
  - 渲染正常
  - Loading 状态
  - 数值格式化
- [ ] 测试 `TopTalkersList.tsx`
  - 列表渲染
  - 排序功能
  - 空数据状态
- [ ] 测试 `SessionDetail.tsx`
  - 节点详情显示
  - 边详情显示
  - 统计计算正确性

**预计时间**: 1 天

---

#### Day 2: Hook 和工具函数测试

**任务 2.3.4: API Hooks 测试**
- [ ] 测试 `useAgents.ts`
  - 数据获取成功
  - 错误处理
  - 缓存机制
- [ ] 测试 `useFlows.ts`
- [ ] 测试 `usePolicies.ts`
- [ ] 测试 `useAuth.ts`（如果已实现）

**任务 2.3.5: 工具函数测试**
- [ ] 测试 `src/utils/format.ts`
  - formatBytes()
  - formatNumber()
  - formatTimestamp()
- [ ] 测试 `src/utils/chartHelpers.ts`
- [ ] 测试 `src/utils/policyValidation.ts`
  - CIDR 验证
  - IP 范围验证
  - 端口验证

**任务 2.3.6: 拓扑工具测试**
- [ ] 测试 `src/utils/topologyUtils.ts`
  - IP 聚合逻辑
  - 标签解析
  - 节点分组
- [ ] 测试 `src/utils/topologyAggregator.ts`
  - 流数据聚合
  - 统计计算
  - 连通组件查找

**预计时间**: 1 天

---

#### Day 3-4: 集成测试和 E2E 测试

**任务 2.3.7: 页面集成测试**
- [ ] 测试 Dashboard 页面
  - 数据加载
  - 图表渲染
  - 刷新功能
- [ ] 测试 Agents 页面
  - 列表渲染
  - 详情页导航
  - 筛选功能
- [ ] 测试 Policies 页面
  - CRUD 操作
  - 表单验证
  - 列表分页

**任务 2.3.8: E2E 测试（可选）**
- [ ] 安装 Playwright 或 Cypress
  ```bash
  npm install --save-dev @playwright/test
  ```
- [ ] 编写关键流程测试
  - 登录流程
  - 创建策略流程
  - 查看流详情流程
- [ ] 配置 CI/CD 集成

**任务 2.3.9: 测试覆盖率报告**
- [ ] 运行覆盖率测试 `npm run test:coverage`
- [ ] 分析覆盖率报告
- [ ] 补充低覆盖率区域的测试
- [ ] 生成 HTML 报告

**预计时间**: 1-2 天

---

## Phase 3: 高级功能 (4-5 天)

### 3.1 高级数据可视化 (2-3 天)

#### Day 1: 时间范围选择器和图表增强

**任务 3.1.1: 创建高级时间选择器**
- [ ] 创建 `src/components/common/TimeRangePicker.tsx`
  - 预设时间范围（1h, 6h, 24h, 7d, 30d）
  - 自定义时间范围
  - 快速时间跳转（上一个/下一个周期）
- [ ] 集成到 Dashboard 和 Flows 页面
- [ ] 实现时间范围 URL 同步（可分享链接）

**任务 3.1.2: 流量热力图**
- [ ] 创建 `src/components/visualization/TrafficHeatmap.tsx`
- [ ] 使用 ECharts Heatmap 图表
- [ ] X 轴：小时（0-23）
- [ ] Y 轴：星期（周一到周日）
- [ ] 值：流量字节数或流数量
- [ ] 添加到 Dashboard

**任务 3.1.3: Top Talkers 气泡图**
- [ ] 创建 `src/components/visualization/TopTalkersBubble.tsx`
- [ ] X 轴：入站流量
- [ ] Y 轴：出站流量
- [ ] 气泡大小：总流量
- [ ] 气泡颜色：协议类型
- [ ] 支持点击跳转到详情

**预计时间**: 1 天

---

#### Day 2: 高级图表和仪表板定制

**任务 3.1.4: 策略命中率漏斗图**
- [ ] 创建 `src/components/visualization/PolicyFunnel.tsx`
- [ ] 使用 ECharts Funnel 图表
- [ ] 展示策略匹配流程
  - 总流量
  - 精确匹配
  - 通配符匹配
  - 默认策略
- [ ] 点击查看详情

**任务 3.1.5: 网络流量桑基图**
- [ ] 创建 `src/components/visualization/FlowSankey.tsx`
- [ ] 使用 ECharts Sankey 图表
- [ ] 展示流量流向
  - 源 IP → 目标 IP
  - 源服务 → 目标服务
  - 源区域 → 目标区域
- [ ] 支持节点点击展开

**任务 3.1.6: 协议分布雷达图**
- [ ] 创建 `src/components/visualization/ProtocolRadar.tsx`
- [ ] 使用 ECharts Radar 图表
- [ ] 多维度展示协议特征
  - 流数量
  - 字节数
  - 包数
  - 平均包大小
  - 连接数

**预计时间**: 1 天

---

#### Day 3: 仪表板定制和交互

**任务 3.1.7: 可定制仪表板**
- [ ] 创建 `src/components/dashboard/CustomizableDashboard.tsx`
- [ ] 支持拖拽调整图表位置（react-grid-layout）
  ```bash
  npm install react-grid-layout
  ```
- [ ] 支持添加/删除图表组件
- [ ] 支持调整图表大小
- [ ] 保存用户自定义布局（localStorage）

**任务 3.1.8: 图表联动**
- [ ] 实现图表点击联动
  - 点击协议分布饼图 → 筛选 Flow 列表
  - 点击 Top Talkers → 高亮拓扑节点
  - 点击时间趋势图 → 更新详细数据
- [ ] 创建全局筛选器状态管理

**任务 3.1.9: 数据钻取**
- [ ] 实现数据下钻功能
  - Dashboard → Flows（带筛选条件）
  - Flow 汇总 → Flow 详情
  - 拓扑节点 → Agent/IP 详情
- [ ] 面包屑导航

**预计时间**: 1 天（可选）

---

### 3.2 性能优化 (2 天)

#### Day 1: 代码分割和懒加载

**任务 3.2.1: 路由级代码分割**
- [ ] 更新 `src/router.tsx`
  ```typescript
  const Dashboard = lazy(() => import('./pages/Dashboard'));
  const Agents = lazy(() => import('./pages/Agents'));
  const Flows = lazy(() => import('./pages/Flows'));
  const Policies = lazy(() => import('./pages/Policies'));
  const Topology = lazy(() => import('./pages/Topology'));
  ```
- [ ] 添加 Suspense 加载占位符
- [ ] 测试懒加载效果

**任务 3.2.2: 组件级代码分割**
- [ ] 大型图表组件懒加载
  - TrafficHeatmap
  - FlowSankey
  - TopologyGraph
- [ ] 按需加载第三方库
  - ECharts 按需引入（只引入使用的图表类型）
  - 图标库按需引入

**任务 3.2.3: Bundle 分析和优化**
- [ ] 安装分析工具
  ```bash
  npm install --save-dev rollup-plugin-visualizer
  ```
- [ ] 分析打包产物
  ```bash
  npm run build -- --mode analyze
  ```
- [ ] 识别大体积依赖
  - 考虑替换或按需引入
  - 移除未使用的依赖

**预计时间**: 1 天

---

#### Day 2: 缓存和渲染优化

**任务 3.2.4: TanStack Query 缓存优化**
- [ ] 优化 staleTime 和 cacheTime 配置
  ```typescript
  // Dashboard 数据：30 秒缓存
  useQuery({
    queryKey: ['dashboard'],
    staleTime: 30 * 1000,
    cacheTime: 5 * 60 * 1000,
  });

  // 静态数据：5 分钟缓存
  useQuery({
    queryKey: ['policies'],
    staleTime: 5 * 60 * 1000,
  });
  ```
- [ ] 实现预加载（Prefetching）
  - 鼠标悬停时预加载详情
- [ ] 实现乐观更新（Optimistic Updates）

**任务 3.2.5: 虚拟滚动优化**
- [ ] 安装虚拟滚动库
  ```bash
  npm install react-window
  ```
- [ ] 优化 Flow 列表（支持 1000+ 条数据）
- [ ] 优化 Policy 列表
- [ ] 优化 Agent 列表

**任务 3.2.6: React 渲染优化**
- [ ] 使用 React.memo 优化组件
  - MetricCard
  - FlowTableRow
  - PolicyCard
- [ ] 使用 useMemo 缓存计算结果
  - 图表配置对象
  - 过滤后的数据
- [ ] 使用 useCallback 缓存回调函数
- [ ] 避免匿名函数作为 props

**任务 3.2.7: 图片和资源优化**
- [ ] 图片懒加载（react-lazy-load-image-component）
- [ ] SVG 图标内联（减少请求）
- [ ] 字体优化（仅加载使用的字重）

**任务 3.2.8: 性能监控**
- [ ] 添加性能监控工具（web-vitals）
  ```bash
  npm install web-vitals
  ```
- [ ] 监控关键指标
  - FCP (First Contentful Paint)
  - LCP (Largest Contentful Paint)
  - FID (First Input Delay)
  - CLS (Cumulative Layout Shift)
- [ ] 添加性能报告（可选）

**预计时间**: 1 天

---

## Phase 4: 长期优化 (3-5 天)

### 4.1 国际化 (i18n) (2-3 天)

#### Day 1: i18n 基础架构

**任务 4.1.1: 安装和配置 i18n**
- [ ] 安装依赖
  ```bash
  npm install react-i18next i18next i18next-browser-languagedetector
  ```
- [ ] 创建 i18n 配置 `src/i18n/config.ts`
  ```typescript
  import i18n from 'i18next';
  import { initReactI18next } from 'react-i18next';
  import LanguageDetector from 'i18next-browser-languagedetector';

  import en from './locales/en.json';
  import zh from './locales/zh.json';

  i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      resources: {
        en: { translation: en },
        zh: { translation: zh },
      },
      fallbackLng: 'en',
      interpolation: {
        escapeValue: false,
      },
    });
  ```

**任务 4.1.2: 创建翻译文件结构**
- [ ] 创建 `src/i18n/locales/en.json`
- [ ] 创建 `src/i18n/locales/zh.json`
- [ ] 组织翻译键结构
  ```json
  {
    "common": {
      "save": "Save",
      "cancel": "Cancel",
      "delete": "Delete"
    },
    "dashboard": {
      "title": "Dashboard",
      "activeAgents": "Active Agents"
    },
    "agents": {
      "title": "Agents",
      "list": "Agent List"
    }
  }
  ```

**任务 4.1.3: 创建语言切换组件**
- [ ] 创建 `src/components/common/LanguageSwitcher.tsx`
  - 下拉菜单选择语言
  - 🇺🇸 English / 🇨🇳 简体中文
- [ ] 集成到 Header 组件
- [ ] 保存语言偏好（localStorage）

**预计时间**: 1 天

---

#### Day 2-3: 翻译所有界面文本

**任务 4.1.4: 翻译核心组件**
- [ ] 翻译 Layout 组件
  - Header
  - Sidebar 菜单
  - Footer
- [ ] 翻译 Dashboard 页面
  - 关键指标标题
  - 图表标题和标签
  - 按钮文本
- [ ] 翻译 Agents 页面
- [ ] 翻译 Flows 页面
- [ ] 翻译 Policies 页面
- [ ] 翻译 Topology 页面

**任务 4.1.5: 翻译表单和验证消息**
- [ ] 翻译表单标签
- [ ] 翻译占位符文本
- [ ] 翻译验证错误消息
- [ ] 翻译成功/失败提示

**任务 4.1.6: Ant Design 国际化**
- [ ] 配置 Ant Design LocaleProvider
  ```typescript
  import zhCN from 'antd/locale/zh_CN';
  import enUS from 'antd/locale/en_US';

  <ConfigProvider locale={locale === 'zh' ? zhCN : enUS}>
    <App />
  </ConfigProvider>
  ```
- [ ] 测试日期选择器、分页等组件

**任务 4.1.7: 动态内容国际化**
- [ ] 时间格式本地化
  ```typescript
  import { format } from 'date-fns';
  import { zhCN, enUS } from 'date-fns/locale';

  format(date, 'PPP', { locale: locale === 'zh' ? zhCN : enUS });
  ```
- [ ] 数字格式本地化
  ```typescript
  new Intl.NumberFormat(locale).format(12345.67);
  ```
- [ ] 货币格式本地化（如果有）

**任务 4.1.8: 测试**
- [ ] 测试所有页面的中英文切换
- [ ] 测试 RTL 布局（如果支持阿拉伯语等）
- [ ] 测试缺失翻译的 fallback
- [ ] 检查翻译质量（术语一致性）

**预计时间**: 1-2 天

---

### 4.2 移动端优化 (2 天)

#### Day 1: 响应式布局优化

**任务 4.2.1: 移动端布局调整**
- [ ] 优化 Sidebar
  - 移动端：抽屉模式（Drawer）
  - 平板：可折叠 Sidebar
  - 桌面：固定 Sidebar
- [ ] 优化 Header
  - 移动端：汉堡菜单 + 精简按钮
  - 隐藏次要功能按钮
- [ ] 优化 Dashboard 布局
  - 移动端：单列布局
  - 平板：双列布局
  - 桌面：三列布局

**任务 4.2.2: 表格和列表优化**
- [ ] 移动端表格优化
  - 隐藏次要列
  - 使用卡片视图替代表格
  - 支持横向滚动（关键列固定）
- [ ] 创建 `src/components/common/ResponsiveTable.tsx`
  - 自动检测设备类型
  - 移动端：卡片模式
  - 桌面：表格模式

**任务 4.2.3: 图表响应式**
- [ ] 优化 ECharts 图表
  - 移动端：减少图例、简化标签
  - 自适应图表高度
  - 禁用复杂交互（移动端）
- [ ] 优化拓扑图
  - 移动端：缩放和平移手势
  - 简化节点标签

**预计时间**: 1 天

---

#### Day 2: 移动端交互和性能

**任务 4.2.4: 触摸交互优化**
- [ ] 增大按钮点击区域（最小 44x44px）
- [ ] 优化下拉菜单（移动端全屏）
- [ ] 添加触摸反馈（active 状态）
- [ ] 支持手势操作
  - 左右滑动切换页面
  - 下拉刷新

**任务 4.2.5: 移动端性能优化**
- [ ] 减少移动端首屏加载资源
- [ ] 延迟加载非关键图表
- [ ] 优化图片大小（响应式图片）
- [ ] 减少动画复杂度

**任务 4.2.6: PWA 支持（可选）**
- [ ] 创建 `manifest.json`
  ```json
  {
    "name": "eBPF Microsegmentation",
    "short_name": "eBPF μSeg",
    "start_url": "/",
    "display": "standalone",
    "theme_color": "#1890ff",
    "icons": [...]
  }
  ```
- [ ] 配置 Service Worker
- [ ] 支持离线访问（缓存静态资源）
- [ ] 添加到主屏幕提示

**任务 4.2.7: 移动端测试**
- [ ] 真机测试（iOS Safari, Android Chrome）
- [ ] 测试不同屏幕尺寸
  - 小屏手机（320px）
  - 大屏手机（414px）
  - 平板（768px, 1024px）
- [ ] 测试触摸交互
- [ ] 测试性能（Lighthouse）

**预计时间**: 1 天

---

## 技术栈说明

### 认证系统
- **JWT**: JSON Web Token 认证
- **Zustand**: 轻量级状态管理
- **Axios Interceptors**: 自动 Token 注入和刷新

### 主题系统
- **Ant Design Theme**: 内置暗黑模式支持
- **CSS Variables**: 自定义主题变量
- **ECharts Themes**: 图表主题适配

### 报表导出
- **jsPDF**: PDF 生成库
- **jspdf-autotable**: PDF 表格插件
- **html2canvas**: HTML 转图片（可选）
- **Papa Parse**: CSV 解析和生成（可选）

### 数据可视化
- **ECharts**: 企业级图表库
- **echarts-for-react**: React 封装
- **react-grid-layout**: 可拖拽布局（可选）

### 性能优化
- **React.lazy**: 组件懒加载
- **react-window**: 虚拟滚动
- **rollup-plugin-visualizer**: Bundle 分析
- **web-vitals**: 性能监控

### 国际化
- **react-i18next**: React i18n 解决方案
- **i18next-browser-languagedetector**: 自动语言检测
- **date-fns**: 日期国际化

### 测试
- **Vitest**: 单元测试框架
- **@testing-library/react**: React 组件测试
- **@vitest/ui**: 测试 UI 界面
- **Playwright**: E2E 测试（可选）

---

## 验收标准

### Phase 1 验收标准

**用户认证系统**:
- [ ] 登录/登出功能正常
- [ ] Token 自动刷新机制工作
- [ ] 路由守卫正确拦截未授权访问
- [ ] 角色权限控制生效
- [ ] Token 过期提示友好

**策略统计 API**:
- [ ] 策略命中统计正确显示
- [ ] 统计数据实时更新
- [ ] 可视化图表准确

### Phase 2 验收标准

**暗黑模式**:
- [ ] 主题切换流畅（无闪烁）
- [ ] 所有页面和组件适配
- [ ] 主题持久化正常
- [ ] 颜色对比度符合 WCAG 标准

**报表导出**:
- [ ] PDF 格式正确，内容完整
- [ ] CSV 导出支持中文
- [ ] 图表导出清晰（高分辨率）
- [ ] 大数据量导出不卡顿

**测试覆盖**:
- [ ] 代码覆盖率达到 70% 以上
- [ ] 关键流程有 E2E 测试
- [ ] CI/CD 集成测试通过

### Phase 3 验收标准

**高级可视化**:
- [ ] 新图表类型正常渲染
- [ ] 图表联动功能工作
- [ ] 自定义仪表板可保存
- [ ] 数据钻取路径正确

**性能优化**:
- [ ] 首屏加载时间 < 2 秒
- [ ] 打包体积减小 20% 以上
- [ ] LCP < 2.5 秒
- [ ] 虚拟滚动流畅（60fps）

### Phase 4 验收标准

**国际化**:
- [ ] 中英文切换无遗漏
- [ ] Ant Design 组件已国际化
- [ ] 日期/数字格式本地化
- [ ] 语言偏好持久化

**移动端优化**:
- [ ] 响应式布局无错位
- [ ] 触摸交互流畅
- [ ] 移动端性能良好（Lighthouse > 85）
- [ ] PWA 安装功能正常（可选）

---

## 风险和依赖

### 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 后端 API 不稳定 | 高 | Mock 数据、API 版本管理 |
| 第三方库兼容性 | 中 | 提前验证、保持版本更新 |
| 性能瓶颈 | 中 | 提前性能测试、渐进优化 |
| 浏览器兼容性 | 低 | 使用 polyfill、测试覆盖 |

### 依赖关系

**Phase 1 依赖**:
- 后端必须提供认证 API（/api/v1/auth/*）
- 后端必须提供策略统计 API

**Phase 2 依赖**:
- Phase 1 完成（认证系统可能影响权限控制）

**Phase 3 依赖**:
- 无强依赖，可并行开发

**Phase 4 依赖**:
- 翻译内容需要专业翻译（如有预算）
- 移动端测试需要真机设备

### 人力资源

**推荐配置**:
- 前端开发 1-2 人
- UI/UX 设计师 0.5 人（兼职，主要 Phase 2-3）
- 测试工程师 0.5 人（兼职，主要 Phase 2）
- 翻译 0.2 人（外包，Phase 4）

### 时间估算

| 阶段 | 工期 | 人日 |
|------|------|------|
| Phase 1 | 4-5 天 | 4-5 |
| Phase 2 | 7-8 天 | 7-8 |
| Phase 3 | 4-5 天 | 4-5 |
| Phase 4 | 3-5 天 | 3-5 |
| **总计** | **18-23 天** | **18-23** |

*注：按 1 人全职开发计算*

---

## 优先级调整建议

### 如果时间紧张（10 天内）

**必须完成**（P0）:
- ✅ 用户认证系统（3 天）
- ✅ 策略统计 API 集成（1 天）
- ✅ 暗黑模式（2 天）
- ✅ 核心测试覆盖（2 天）
- ✅ 性能优化（1 天）
- ✅ 报表导出基础功能（1 天）

**延后**（P2）:
- 国际化（除非有明确需求）
- 高级可视化（仅保留基础图表）
- 移动端深度优化（基础响应式即可）

### 如果资源充足（20 天以上）

**额外增强**:
- [ ] 实时协作功能（多用户同时查看）
- [ ] 高级过滤器（保存筛选条件）
- [ ] 通知系统（告警、任务提醒）
- [ ] 审计日志（用户操作记录）
- [ ] 系统设置页面（全局配置）

---

## 后续维护计划

### 代码维护
- 定期更新依赖（每月）
- 修复安全漏洞（CVE）
- 代码重构（技术债务清理）

### 功能迭代
- 收集用户反馈
- 优先级排序新需求
- 每季度一次大版本更新

### 性能监控
- 持续监控 Core Web Vitals
- 定期性能审计（Lighthouse）
- 用户体验调研

---

**文档维护者**: Claude AI Assistant
**最后更新**: 2025-11-18
**版本**: v1.0

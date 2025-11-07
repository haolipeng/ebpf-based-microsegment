# 实施任务：Web UI 基础架构

**Change ID**: `add-web-ui-foundation`
**创建时间**: 2025-11-07
**预计工作量**: 2-3 天
**当前状态**: 已完成
**进度**: 15/15 任务完成

---

## 任务概览

| 阶段 | 任务数 | 预计时间 | 状态 |
|------|--------|----------|------|
| Day 1: 项目初始化 | 5 个任务 | 1 天 | ✅ 已完成 |
| Day 2: 核心架构 | 6 个任务 | 1 天 | ✅ 已完成 |
| Day 3: 完善和验证 | 4 个任务 | 0.5-1 天 | ✅ 已完成 |
| **总计** | **15 个任务** | **2-3 天** | **100% 完成** |

---

## Day 1: 项目初始化 (1 天)

### 任务 1.1: 初始化项目
- [x] 创建 `web/` 目录
- [x] 使用 Vite 初始化 React + TypeScript 项目
- [x] 验证项目可正常启动(`npm run dev`)
- [x] 配置 Git 忽略规则(`.gitignore`)

### 任务 1.2: 安装核心依赖
- [x] 安装 Ant Design: `npm install antd`
- [x] 安装路由: `npm install react-router-dom`
- [x] 安装状态管理: `npm install zustand`
- [x] 安装数据获取: `npm install @tanstack/react-query`
- [x] 安装 HTTP 客户端: `npm install axios`
- [x] 更新 `package.json` 脚本

### 任务 1.3: 配置开发工具
- [x] 配置 ESLint(`.eslintrc.cjs`)
- [x] 配置 Prettier(`.prettierrc`)
- [x] 配置 TypeScript(`tsconfig.json` 严格模式)
- [x] 验证代码检查和格式化正常工作

### 任务 1.4: 创建目录结构
- [x] 创建 `src/api/` 目录
- [x] 创建 `src/components/layout/` 目录
- [x] 创建 `src/components/common/` 目录
- [x] 创建 `src/pages/` 目录(Dashboard/Agents/Flows/Policies)
- [x] 创建 `src/hooks/` 目录
- [x] 创建 `src/store/` 目录
- [x] 创建 `src/types/` 目录
- [x] 创建 `src/utils/` 目录

### 任务 1.5: 配置环境变量
- [x] 创建 `.env.development` 文件
- [x] 配置 `VITE_API_BASE_URL`
- [x] 创建 `.env.production.example` 文件
- [x] 在 `vite.config.ts` 中配置代理(解决开发环境 CORS)

---

## Day 2: 核心架构 (1 天)

### 任务 2.1: 实现 API 客户端
- [x] 创建 `src/api/client.ts`(Axios 实例配置)
- [x] 实现请求拦截器(添加通用 headers)
- [x] 实现响应拦截器(统一错误处理)
- [x] 配置超时和重试机制

### 任务 2.2: 定义 TypeScript 类型
- [x] 创建 `src/types/agent.ts`(Agent 和 AgentMetrics 类型)
- [x] 创建 `src/types/flow.ts`(Flow 类型)
- [x] 创建 `src/types/policy.ts`(Policy 类型)
- [x] 创建 `src/types/common.ts`(通用类型,如 ApiResponse)
- [x] 创建 `src/api/types.ts`(API 响应类型)

### 任务 2.3: 实现布局组件
- [x] 创建 `src/components/layout/MainLayout.tsx`
- [x] 创建 `src/components/layout/Header.tsx`
- [x] 创建 `src/components/layout/Sidebar.tsx`
- [x] 使用 Ant Design Layout 组件
- [x] 实现响应式布局(移动端适配)
- [x] 添加 Logo 和应用标题

### 任务 2.4: 配置路由系统
- [x] 创建 `src/router.tsx`
- [x] 配置 React Router(BrowserRouter)
- [x] 定义路由表(/, /agents, /flows, /policies)
- [x] 实现嵌套路由(MainLayout 作为父路由)
- [x] 在 `src/main.tsx` 中集成路由

### 任务 2.5: 创建页面占位组件
- [x] 创建 `src/pages/Dashboard/index.tsx`(临时占位)
- [x] 创建 `src/pages/Agents/index.tsx`(临时占位)
- [x] 创建 `src/pages/Flows/index.tsx`(临时占位)
- [x] 创建 `src/pages/Policies/index.tsx`(临时占位)
- [x] 每个页面显示标题和简单描述

### 任务 2.6: 实现 API 模块
- [x] 创建 `src/api/agents.ts`
- [x] 实现 `agentsApi.list()` 方法
- [x] 创建 `src/api/flows.ts`(基础结构)
- [x] 创建 `src/api/policies.ts`(基础结构)
- [x] 添加类型注解

---

## Day 3: 完善和验证 (0.5-1 天)

### 任务 3.1: 集成 TanStack Query
- [x] 在 `src/main.tsx` 中配置 QueryClient
- [x] 创建 `src/hooks/useAgents.ts`(使用 useQuery)
- [x] 测试数据获取和缓存
- [x] 配置合理的 staleTime 和 cacheTime

### 任务 3.2: 连接后端 API 测试
- [x] 启动本地后端服务(Server MVP)
- [x] 在 Dashboard 页面调用 Agent API
- [x] 验证数据正常返回并显示
- [x] 处理 Loading 和 Error 状态
- [x] 验证 CORS 配置正确

### 任务 3.3: 构建和部署验证
- [x] 执行 `npm run build` 验证构建成功
- [x] 检查 TypeScript 类型错误
- [x] 检查 ESLint 警告
- [x] 执行 `npm run preview` 预览生产版本
- [x] 验证打包体积合理(<500KB gzip)

### 任务 3.4: 文档和清理
- [x] 更新 `web/README.md`(开发指南)
- [x] 添加启动和构建说明
- [x] 清理未使用的代码
- [x] 提交代码到 Git

---

## 验收标准

### 功能完整性
- [x] 项目可正常启动(无错误)
- [x] 所有路由可访问(Dashboard/Agents/Flows/Policies)
- [x] 布局渲染正常(Header + Sidebar + Content)
- [x] Sidebar 菜单高亮当前路由
- [x] API 客户端可连接后端健康检查端点

### 代码质量
- [x] TypeScript 无类型错误
- [x] ESLint 无警告
- [x] Prettier 格式化一致
- [x] 代码结构清晰,符合规范

### 性能
- [x] 开发服务器启动时间 < 3 秒
- [x] 页面首次渲染时间 < 1 秒
- [x] 生产构建时间 < 30 秒

### 文档
- [x] README 包含完整的开发指南
- [x] 环境变量说明清晰
- [x] API 使用示例完整

---

**预计总工作量**: 2-3 天
**依赖**: Server MVP HTTP API, 后端 CORS 配置
**后续步骤**: 实施 Dashboard 仪表板模块

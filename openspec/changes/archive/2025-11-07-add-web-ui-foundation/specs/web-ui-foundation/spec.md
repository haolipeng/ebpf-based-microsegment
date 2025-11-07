# 规范：Web UI 基础架构

本规范定义前端 Web UI 的基础架构要求。

---

## 需求

### 需求：项目初始化

前端项目必须使用现代化的构建工具和框架。

#### 场景：创建项目

**前提** 系统已安装 Node.js 18+
**当** 执行项目初始化命令
**那么** 必须生成 Vite + React + TypeScript 项目结构
**并且** 项目可以成功启动开发服务器
**并且** 浏览器可以访问 http://localhost:5173

#### 场景：依赖安装

**前提** package.json 已配置
**当** 执行 `npm install`
**那么** 必须成功安装所有依赖
**并且** 包含 React 18+
**并且** 包含 TypeScript 5+
**并且** 包含 Ant Design 5+
**并且** 包含 React Router 6+
**并且** 包含 TanStack Query 5+
**并且** 包含 Axios 1+

### 需求：布局系统

Web UI 必须提供响应式的主布局。

#### 场景：渲染主布局

**前提** 用户访问任意页面
**当** 页面加载完成
**那么** 必须显示顶部 Header
**并且** 必须显示左侧 Sidebar
**并且** 必须显示主内容区 Content
**并且** 布局必须占满整个视口

#### 场景：侧边栏导航

**前提** 用户在任意页面
**当** 用户查看 Sidebar
**那么** 必须显示 Dashboard 菜单项
**并且** 必须显示 Agents 菜单项
**并且** 必须显示 Flows 菜单项
**并且** 必须显示 Policies 菜单项
**并且** 当前路由对应的菜单项必须高亮

#### 场景：响应式布局

**前提** 布局已渲染
**当** 浏览器宽度 < 768px(移动端)
**那么** Sidebar 必须自动折叠
**并且** 可以通过按钮展开/收起
**并且** Content 必须占满屏幕宽度

### 需求：路由系统

Web UI 必须支持客户端路由。

#### 场景：访问根路径

**前提** 应用已启动
**当** 用户访问 `/`
**那么** 必须渲染 Dashboard 页面
**并且** URL 必须为 `/`
**并且** Sidebar 的 Dashboard 菜单高亮

#### 场景：切换页面

**前提** 用户在 Dashboard 页面
**当** 用户点击 Agents 菜单
**那么** 必须导航到 `/agents`
**并且** 必须渲染 Agents 页面
**并且** 页面不刷新(SPA 行为)
**并且** Sidebar 的 Agents 菜单高亮

#### 场景：直接访问子路径

**前提** 应用已启动
**当** 用户直接访问 `/flows`
**那么** 必须渲染 Flows 页面
**并且** 布局正常显示

### 需求：API 客户端

Web UI 必须提供统一的 API 客户端封装。

#### 场景：创建 API 客户端

**前提** Axios 已安装
**当** 初始化 API 客户端
**那么** 必须配置 baseURL 为后端 API 地址
**并且** 必须配置 10 秒超时
**并且** 必须设置 Content-Type 为 application/json

#### 场景：请求拦截

**前提** API 客户端已初始化
**当** 发送任意请求
**那么** 请求拦截器必须执行
**并且** 必须添加通用请求头
**并且** 必须记录请求日志(开发环境)

#### 场景：响应拦截

**前提** API 客户端已初始化
**当** 收到响应
**那么** 响应拦截器必须执行
**并且** 成功响应必须返回 response.data
**并且** 错误响应必须统一处理
**并且** 必须记录错误日志

#### 场景：调用后端健康检查

**前提** 后端服务运行在 localhost:8080
**当** 调用 `GET /health`
**那么** 必须返回 200 状态码
**并且** 响应必须包含 status: "healthy"
**并且** 必须在 10 秒内返回

### 需求：类型系统

Web UI 必须定义完整的 TypeScript 类型。

#### 场景：定义 Agent 类型

**前提** TypeScript 已配置
**当** 定义 Agent 类型
**那么** 必须包含 agent_id(string)
**并且** 必须包含 hostname(string)
**并且** 必须包含 status('active' | 'inactive')
**并且** 必须包含 metrics(AgentMetrics)
**并且** 所有字段必须有明确类型

#### 场景：定义 Flow 类型

**前提** TypeScript 已配置
**当** 定义 Flow 类型
**那么** 必须包含 5-tuple(src_ip, dst_ip, src_port, dst_port, protocol)
**并且** 必须包含 policy_action('ALLOW' | 'DENY' | 'LOG')
**并且** 必须包含 state('ACTIVE' | 'CLOSED' | 'TIMEOUT')
**并且** 必须包含 labels(Record<string, string>)

#### 场景：类型检查

**前提** 所有类型已定义
**当** 执行 `npm run type-check` 或 `tsc --noEmit`
**那么** 必须无类型错误
**并且** 必须启用 strict 模式
**并且** 不能使用 any 类型(除非明确标注)

### 需求：数据获取

Web UI 必须使用 TanStack Query 管理服务端状态。

#### 场景：配置 QueryClient

**前提** TanStack Query 已安装
**当** 应用启动
**那么** 必须创建 QueryClient 实例
**并且** 必须包裹在 QueryClientProvider 中
**并且** 必须配置默认选项(staleTime, cacheTime, retry)

#### 场景：获取 Agent 列表

**前提** QueryClient 已配置
**当** 组件调用 `useAgents()` Hook
**那么** 必须发送 GET /api/v1/agents 请求
**并且** 必须返回 loading 状态
**并且** 成功时必须返回 data 和 isSuccess
**并且** 失败时必须返回 error 和 isError
**并且** 数据必须自动缓存

#### 场景：自动重新获取

**前提** Agent 数据已缓存
**当** 窗口重新获得焦点
**那么** 必须自动重新获取数据(如果数据过期)
**并且** 必须在后台静默更新
**并且** 用户界面不能闪烁

### 需求：开发工具

Web UI 必须配置代码质量工具。

#### 场景：ESLint 检查

**前提** ESLint 已配置
**当** 执行 `npm run lint`
**那么** 必须检查所有 .ts 和 .tsx 文件
**并且** 必须报告代码风格问题
**并且** 必须报告潜在错误

#### 场景：Prettier 格式化

**前提** Prettier 已配置
**当** 执行 `npm run format`
**那么** 必须格式化所有代码文件
**并且** 必须使用 2 空格缩进
**并且** 必须使用单引号
**并且** 必须添加分号

### 需求：构建和部署

Web UI 必须支持生产构建。

#### 场景：生产构建

**前提** 项目已完成开发
**当** 执行 `npm run build`
**那么** 必须成功构建到 dist/ 目录
**并且** 必须生成优化的 JavaScript bundle
**并且** 必须生成 source map
**并且** 构建时间必须 < 1 分钟

#### 场景：预览生产版本

**前提** 生产构建已完成
**当** 执行 `npm run preview`
**那么** 必须启动静态文件服务器
**并且** 应用必须正常运行
**并且** 所有路由必须可访问

#### 场景：打包体积

**前提** 生产构建已完成
**当** 检查 dist/ 目录
**那么** 主 bundle 大小必须 < 500KB(gzip 压缩后)
**并且** 必须生成 vendor chunk(第三方库)
**并且** 必须启用 Tree Shaking

### 需求：CORS 配置

Web UI 必须能够跨域访问后端 API。

#### 场景：开发环境代理

**前提** Vite 配置文件存在
**当** 启动开发服务器
**那么** 必须配置代理到 http://localhost:8080
**并且** `/api` 路径必须代理到后端
**并且** WebSocket 必须支持代理

#### 场景：生产环境 CORS

**前提** 后端服务已配置 CORS
**当** 前端发送跨域请求
**那么** 必须包含 Access-Control-Allow-Origin 响应头
**并且** 必须允许 GET, POST, PUT, DELETE 方法
**并且** 预检请求(OPTIONS)必须成功

---

## 成功指标

- 项目初始化完成,可正常启动
- 布局系统渲染正确,响应式工作正常
- 路由切换流畅,无页面刷新
- API 客户端连接后端成功
- TypeScript 类型定义完整且无错误
- 数据获取和缓存正常工作
- ESLint 和 Prettier 配置正确
- 生产构建成功,打包体积合理
- CORS 配置正确,跨域请求成功

---

## ADDED Requirements

本次变更新增了完整的 Web UI 基础架构,创建了 `web/` 目录及其所有子目录和文件。

### Requirement: Web UI 项目结构

系统必须(SHALL)提供完整的 Web UI 项目结构,包含以下目录和文件:

**配置文件:**
- `package.json` - 项目依赖和脚本配置,包含所有必需的依赖(React 19、TypeScript 5、Ant Design 5、TanStack Query 5 等)
- `tsconfig.json` - TypeScript 编译配置,启用 strict 模式和 verbatimModuleSyntax
- `vite.config.ts` - Vite 构建工具配置,包含代理设置(`/api` 代理到 `localhost:8080`)
- `.prettierrc` - Prettier 代码格式化配置(2 空格缩进,单引号,无分号)
- `.eslintrc.cjs` - ESLint 代码检查配置
- `.env.development` - 开发环境变量配置(`VITE_API_BASE_URL=http://localhost:8080`)
- `.env.production.example` - 生产环境变量示例
- `README.md` - 完整的开发指南和文档,包含启动、构建、测试指令

**源代码目录:**
- `src/api/` - API 客户端封装模块
  - `client.ts` - Axios 实例配置,实现请求/响应拦截器和统一错误处理
  - `agents.ts` - Agent API 端点(list, get, health)
  - `flows.ts` - Flow API 端点(list, get, delete, stats)
  - `policies.ts` - Policy API 端点(list, get, create, update, delete, stats)
- `src/types/` - TypeScript 类型定义
  - `common.ts` - 通用类型(ApiResponse, PaginationParams, TimeRange 等)
  - `agent.ts` - Agent 和 AgentMetrics 类型定义
  - `flow.ts` - Flow 类型定义(包含 5-tuple, state, labels)
  - `policy.ts` - Policy 类型定义(PolicyRule, PolicyAction, PolicyStats)
- `src/components/` - React 组件
  - `layout/MainLayout.tsx` - 主布局容器,使用 Ant Design Layout
  - `layout/Header.tsx` - 顶部导航栏,显示应用标题和 Logo
  - `layout/Sidebar.tsx` - 侧边栏导航菜单,支持路由高亮
- `src/pages/` - 页面组件
  - `Dashboard/index.tsx` - 仪表板页面,集成健康检查和 Agent 统计展示
  - `Agents/index.tsx` - Agent 管理页面(占位)
  - `Flows/index.tsx` - 网络流页面(占位)
  - `Policies/index.tsx` - 安全策略页面(占位)
- `src/hooks/` - 自定义 React Hooks
  - `useAgents.ts` - Agent 数据获取 Hooks(useAgents, useAgent, useHealthCheck)
- `src/router.tsx` - React Router 路由配置,定义嵌套路由结构
- `src/main.tsx` - 应用入口,配置 QueryClient 和路由提供者
- `src/index.css` - 全局样式

#### Scenario: 项目启动和访问

**Given** Web UI 项目已初始化,所有依赖已安装
**When** 开发者执行 `npm run dev`
**Then** Vite 开发服务器必须(SHALL)在 3 秒内启动
**And** 应用必须(SHALL)可通过 http://localhost:3000 访问
**And** 页面必须(SHALL)显示完整的布局(Header + Sidebar + Content)
**And** Dashboard 页面必须(SHALL)成功渲染
**And** 控制台必须(SHALL)无错误信息

#### Scenario: API 客户端连接后端

**Given** 后端服务运行在 http://localhost:8080
**When** Dashboard 页面加载
**Then** 前端必须(SHALL)发送 GET /health 请求
**And** 请求必须(SHALL)通过 Vite 代理转发到后端
**And** 如果后端健康,必须(SHALL)显示绿色健康状态图标
**And** 必须(SHALL)显示 Agent 统计信息(总数、在线、离线)

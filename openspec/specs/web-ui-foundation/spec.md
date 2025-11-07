# web-ui-foundation Specification

## Purpose
TBD - created by archiving change add-web-ui-foundation. Update Purpose after archive.
## Requirements
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


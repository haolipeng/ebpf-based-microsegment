# 设计文档：Web UI 基础架构

**Change ID**: `add-web-ui-foundation`
**创建时间**: 2025-11-07

---

## 技术架构

### 技术栈选型

#### 核心框架
- **React 18.3+**: 主流前端框架,生态成熟
- **TypeScript 5.x**: 类型安全,提升开发体验
- **Vite 5.x**: 极快的开发服务器,优秀的构建性能

#### UI 组件库
- **Ant Design 5.x**:
  - 企业级中后台组件齐全
  - 设计规范完善
  - 中文文档优秀
  - 内置常用布局和表单组件

#### 状态管理
- **Zustand**:
  - 轻量级(< 1KB)
  - API 简洁
  - 无需 Provider 包裹
  - 适合中小型应用

#### 数据获取
- **TanStack Query (React Query)**:
  - 自动缓存和失效
  - 后台数据同步
  - 分页和无限滚动支持
  - 乐观更新

#### 路由
- **React Router v6**:
  - 声明式路由
  - 嵌套路由支持
  - 类型安全(配合 TypeScript)

### 目录结构

```
web/
├── public/                 # 静态资源
│   └── vite.svg
├── src/
│   ├── api/                # API 客户端层
│   │   ├── client.ts       # Axios 实例配置
│   │   ├── agents.ts       # Agent 相关 API
│   │   ├── flows.ts        # Flow 相关 API
│   │   ├── policies.ts     # Policy 相关 API
│   │   └── types.ts        # API 响应类型
│   ├── components/         # 可复用组件
│   │   ├── layout/         # 布局组件
│   │   │   ├── MainLayout.tsx
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── index.ts
│   │   └── common/         # 通用组件
│   │       └── PageHeader.tsx
│   ├── pages/              # 页面组件
│   │   ├── Dashboard/
│   │   │   └── index.tsx
│   │   ├── Agents/
│   │   │   └── index.tsx
│   │   ├── Flows/
│   │   │   └── index.tsx
│   │   └── Policies/
│   │       └── index.tsx
│   ├── hooks/              # 自定义 Hooks
│   │   └── useApi.ts
│   ├── store/              # 状态管理
│   │   └── appStore.ts
│   ├── types/              # TypeScript 类型定义
│   │   ├── agent.ts
│   │   ├── flow.ts
│   │   ├── policy.ts
│   │   └── common.ts
│   ├── utils/              # 工具函数
│   │   ├── format.ts       # 格式化工具
│   │   └── constants.ts    # 常量定义
│   ├── router.tsx          # 路由配置
│   ├── App.tsx             # 根组件
│   ├── main.tsx            # 入口文件
│   └── vite-env.d.ts
├── .eslintrc.cjs           # ESLint 配置
├── .prettierrc             # Prettier 配置
├── tsconfig.json           # TypeScript 配置
├── tsconfig.node.json
├── vite.config.ts          # Vite 配置
├── package.json
└── README.md
```

### 核心组件设计

#### 1. MainLayout (主布局)

```typescript
interface MainLayoutProps {
  children: React.ReactNode;
}

// 布局结构
<Layout>
  <Header />
  <Layout>
    <Sidebar />
    <Content>{children}</Content>
  </Layout>
</Layout>
```

**功能**:
- 固定 Header(顶部导航)
- 可折叠 Sidebar(侧边菜单)
- 自适应 Content(主内容区)

#### 2. Header (顶部导航)

**包含元素**:
- Logo 和应用名称
- 面包屑导航
- 系统状态指示器(API 连接状态)
- 用户信息(未来扩展)

#### 3. Sidebar (侧边菜单)

**菜单项**:
- Dashboard (仪表板)
- Agents (Agent 管理)
- Flows (流量分析)
- Policies (策略管理)
- Analytics (高级分析)

**功能**:
- 支持折叠/展开
- 高亮当前路由
- 图标 + 文字

### API 客户端设计

#### Axios 配置

```typescript
// src/api/client.ts
import axios from 'axios';

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // 添加认证 token(未来扩展)
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    // 统一错误处理
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);
```

#### API 模块化

```typescript
// src/api/agents.ts
import { apiClient } from './client';
import type { Agent, AgentListResponse } from './types';

export const agentsApi = {
  // 获取所有 Agent
  list: () => apiClient.get<AgentListResponse>('/agents'),

  // 获取单个 Agent 详情(待后端实现)
  get: (id: string) => apiClient.get<Agent>(`/agents/${id}`),
};
```

### 路由设计

```typescript
// src/router.tsx
import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './components/layout/MainLayout';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Flows from './pages/Flows';
import Policies from './pages/Policies';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: 'agents', element: <Agents /> },
      { path: 'flows', element: <Flows /> },
      { path: 'policies', element: <Policies /> },
    ],
  },
]);
```

### 类型定义

#### Agent 类型

```typescript
// src/types/agent.ts
export interface Agent {
  agent_id: string;
  hostname: string;
  version: string;
  interface: string;
  ip_addresses: string[];
  os: string;
  kernel_version: string;
  start_time: string;
  last_heartbeat: string;
  status: 'active' | 'inactive';
  metrics: AgentMetrics;
}

export interface AgentMetrics {
  cpu_usage: number;
  memory_usage: number;
  packets_processed: number;
  active_sessions: number;
  flows_reported: number;
  active_policies: number;
}
```

#### Flow 类型

```typescript
// src/types/flow.ts
export interface Flow {
  id: number;
  timestamp_ns: number;
  src_ip: string;
  dst_ip: string;
  src_port: number;
  dst_port: number;
  protocol: number;
  direction: 'INGRESS' | 'EGRESS';
  packet_count: number;
  byte_count: number;
  policy_id?: number;
  policy_action: 'ALLOW' | 'DENY' | 'LOG';
  state: 'ACTIVE' | 'CLOSED' | 'TIMEOUT';
  agent_id: string;
  source_labels?: Record<string, string>;
  dest_labels?: Record<string, string>;
  created_at: string;
  updated_at: string;
}
```

#### Policy 类型

```typescript
// src/types/policy.ts
export interface Policy {
  rule_id: number;
  src_ip?: string;
  dst_ip?: string;
  src_port?: string;
  dst_port?: string;
  protocol?: number;
  action: 'ALLOW' | 'DENY' | 'LOG';
  priority: number;
  source_labels?: Record<string, string>;
  dest_labels?: Record<string, string>;
  description?: string;
  created_at: string;
  updated_at: string;
}
```

### 环境变量配置

```typescript
// .env.development
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/api/v1/flows/stream

// .env.production
VITE_API_BASE_URL=/api/v1
VITE_WS_URL=ws://your-domain.com/api/v1/flows/stream
```

---

## 开发规范

### 代码风格

#### ESLint 配置
```json
{
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react/recommended",
    "plugin:react-hooks/recommended"
  ],
  "rules": {
    "react/react-in-jsx-scope": "off",
    "@typescript-eslint/no-unused-vars": "warn"
  }
}
```

#### Prettier 配置
```json
{
  "semi": true,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5",
  "printWidth": 100
}
```

### 命名规范

- **组件**: PascalCase (如 `MainLayout.tsx`)
- **文件**: camelCase (如 `apiClient.ts`)
- **类型**: PascalCase (如 `Agent`, `Flow`)
- **常量**: UPPER_SNAKE_CASE (如 `API_BASE_URL`)
- **函数**: camelCase (如 `formatTimestamp`)

### Git 提交规范

- `feat`: 新功能
- `fix`: 修复 Bug
- `refactor`: 重构代码
- `style`: 样式调整
- `docs`: 文档更新
- `chore`: 构建配置

---

## 性能优化

### 代码分割
- 使用 React.lazy() 懒加载路由页面
- 动态导入大型库

### 构建优化
- Vite 默认优化(Tree Shaking、Minify)
- 配置 CDN 加速(生产环境)

### 缓存策略
- TanStack Query 自动缓存
- 配置合理的 staleTime 和 cacheTime

---

## 安全考虑

### CORS 配置
后端需要配置允许前端域名访问:
```go
// Server 端 CORS 配置
router.Use(cors.New(cors.Config{
  AllowOrigins: []string{"http://localhost:5173"}, // Vite 默认端口
  AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
  AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
}))
```

### XSS 防护
- 使用 React 自动转义
- 避免 dangerouslySetInnerHTML

### 敏感信息
- API Token 存储在 HttpOnly Cookie
- 避免在前端硬编码密钥

---

## 测试策略

### 单元测试
- 使用 Vitest (Vite 官方测试框架)
- 测试工具函数和 Hooks

### 组件测试
- 使用 React Testing Library
- 测试关键组件交互

### E2E 测试
- 使用 Playwright (后续阶段)
- 测试关键用户流程

---

## 部署方案

### 开发环境
```bash
npm run dev  # 启动 Vite 开发服务器(端口 5173)
```

### 生产构建
```bash
npm run build  # 构建到 dist/ 目录
npm run preview  # 预览生产构建
```

### Docker 部署(后续)
```dockerfile
# 多阶段构建
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

---

## 里程碑

### Milestone 1: 项目初始化 (Day 1)
- ✅ 初始化 Vite + React + TypeScript 项目
- ✅ 安装核心依赖
- ✅ 配置 ESLint 和 Prettier

### Milestone 2: 基础架构 (Day 2)
- ✅ 创建目录结构
- ✅ 实现主布局组件
- ✅ 配置路由系统
- ✅ 实现 API 客户端

### Milestone 3: 类型系统和验证 (Day 3)
- ✅ 定义 TypeScript 类型
- ✅ 连接后端 API 测试
- ✅ 构建验证
- ✅ 文档完善

---

## 后续扩展

此基础架构完成后,可继续开发:
1. Dashboard 仪表板
2. Agent 管理模块
3. Flow 分析模块
4. Policy 管理模块
5. 依赖关系可视化
6. 实时监控模块

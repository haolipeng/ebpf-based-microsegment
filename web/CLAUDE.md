[根目录](../CLAUDE.md) > **web**

---

# Web 模块

## 模块职责

Web 模块提供 eBPF 微隔离系统的管理界面，负责：

1. **可视化展示**: Dashboard、流量图表、拓扑图
2. **策略管理**: 策略 CRUD 操作、批量导入导出
3. **流量监控**: 实时流事件展示、过滤、聚合
4. **Agent 管理**: 查看 Agent 状态、指标、日志
5. **拓扑可视化**: 网络拓扑图、会话详情
6. **告警管理**: 告警查看、确认、过滤

## 入口与启动

### 主入口
- **文件**: `src/main.tsx`
- **开发服务器**: `npm run dev`
- **访问地址**: `http://localhost:3000`

### 启动流程

```
1. Vite 启动开发服务器
   ↓
2. 加载 main.tsx 入口
   ↓
3. 初始化 React Router
   ↓
4. 初始化 React Query Client
   ↓
5. 渲染 MainLayout
   - Sidebar (导航菜单)
   - Header (顶部栏)
   - Content (路由页面)
   ↓
6. 建立 WebSocket 连接 (可选)
   ↓
7. 渲染首页 (Dashboard)
```

## 技术栈

### 核心框架

```json
{
  "react": "^19.1.1",
  "react-dom": "^19.1.1",
  "react-router-dom": "^7.9.5",
  "vite": "^7.1.7",
  "typescript": "~5.9.3"
}
```

### UI 库

```json
{
  "antd": "^5.28.0",              // 组件库
  "echarts": "^6.0.0",            // 图表
  "echarts-for-react": "^3.0.5"   // ECharts React 封装
}
```

### 状态管理

```json
{
  "@tanstack/react-query": "^5.90.7",  // 服务端状态
  "zustand": "^5.0.8"                  // 客户端状态
}
```

### HTTP 客户端

```json
{
  "axios": "^1.13.2"
}
```

## 页面结构

### 路由定义

**src/router.tsx**:
```tsx
import { createBrowserRouter } from 'react-router-dom';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: 'policies', element: <Policies /> },
      { path: 'flows', element: <Flows /> },
      { path: 'agents', element: <Agents /> },
      { path: 'agents/:id', element: <AgentDetail /> },
      { path: 'topology', element: <Topology /> },
    ],
  },
]);
```

### 页面组件

| 路径 | 组件 | 描述 |
|------|------|------|
| `/` | Dashboard | 总览仪表盘 |
| `/policies` | Policies | 策略管理 |
| `/flows` | Flows | 流事件监控 |
| `/agents` | Agents | Agent 列表 |
| `/agents/:id` | AgentDetail | Agent 详情 |
| `/topology` | Topology | 网络拓扑图 |

## 关键目录

```
web/
├── src/
│   ├── components/          # 可复用组件
│   │   ├── layout/         # 布局组件 (Header, Sidebar, MainLayout)
│   │   ├── dashboard/      # Dashboard 组件
│   │   ├── policies/       # 策略相关组件
│   │   ├── flows/          # 流事件组件
│   │   ├── agents/         # Agent 组件
│   │   ├── topology/       # 拓扑图组件
│   │   ├── visualization/  # 图表组件
│   │   └── common/         # 通用组件
│   ├── pages/              # 页面组件
│   │   ├── Dashboard/
│   │   ├── Policies/
│   │   ├── Flows/
│   │   ├── Agents/
│   │   └── Topology/
│   ├── hooks/              # 自定义 Hooks
│   │   ├── useAgents.ts
│   │   ├── usePolicies.ts
│   │   ├── useFlows.ts
│   │   ├── useTopology.ts
│   │   └── useVisualization.ts
│   ├── api/                # API 客户端
│   │   ├── client.ts       # Axios 配置
│   │   ├── policies.ts     # 策略 API
│   │   ├── flows.ts        # 流事件 API
│   │   └── agents.ts       # Agent API
│   ├── types/              # TypeScript 类型定义
│   │   ├── policy.ts
│   │   ├── flow.ts
│   │   ├── agent.ts
│   │   ├── topology.ts
│   │   └── common.ts
│   ├── utils/              # 工具函数
│   │   ├── format.ts       # 格式化函数
│   │   ├── chartHelpers.ts # 图表辅助
│   │   └── topologyUtils.ts
│   ├── lib/                # 第三方库封装
│   │   └── graph/          # 图算法
│   ├── config/             # 配置文件
│   │   └── api.ts          # API 配置
│   ├── main.tsx            # 入口文件
│   └── router.tsx          # 路由配置
├── public/                 # 静态资源
├── package.json
├── tsconfig.json
├── vite.config.ts
└── index.html
```

## 核心功能

### 1. Dashboard

**组件**: `src/pages/Dashboard/index.tsx`

**功能**:
- 关键指标卡片（流量、策略、Agent 状态）
- 流量趋势图
- 协议分布饼图
- Top 对话主机
- 策略命中统计

**使用的 Hooks**:
- `useFlows()` - 获取流数据
- `usePolicies()` - 获取策略统计
- `useAgents()` - 获取 Agent 状态
- `useVisualization()` - 图表数据转换

### 2. 策略管理

**组件**: `src/pages/Policies/index.tsx`

**功能**:
- 策略列表展示（表格）
- 创建/编辑/删除策略
- 策略过滤和搜索
- 批量导入/导出
- 策略统计卡片

**子组件**:
- `PolicyTable` - 策略表格
- `PolicyForm` - 策略表单
- `PolicyFilters` - 过滤器
- `PolicyStatsCards` - 统计卡片

### 3. 流事件监控

**组件**: `src/pages/Flows/index.tsx`

**功能**:
- 实时流事件展示
- 流过滤（协议、Action、时间范围）
- 流聚合视图
- 协议统计
- 进程统计

**子组件**:
- `FlowTable` - 流事件表格
- `FlowFilters` - 过滤器
- `FlowSummaryCards` - 摘要卡片
- `ProtocolStats` - 协议统计
- `ProcessStats` - 进程统计

### 4. 网络拓扑

**组件**: `src/pages/Topology/index.tsx`

**功能**:
- 基于 ECharts Graph 的拓扑可视化
- 节点（工作负载）和边（会话）
- 交互式操作（缩放、拖拽、选择）
- 节点详情面板
- 会话详情面板
- 拓扑过滤和聚合

**子组件**:
- `TopologyGraph` - 拓扑图主体
- `TopologyControls` - 控制面板
- `TopologyLegend` - 图例
- `NodeDetailPanel` - 节点详情
- `SessionDetail` - 会话详情

**核心逻辑**:
```typescript
// lib/graph/Graph.ts
export class Graph {
  addNode(node: Node): void;
  addEdge(edge: Edge): void;
  getShortestPath(from: string, to: string): string[];
  getCriticalNodes(): string[];
  // ... 更多图算法
}
```

### 5. Agent 管理

**组件**: `src/pages/Agents/index.tsx`

**功能**:
- Agent 列表
- Agent 状态监控
- Agent 详情页（指标、配置）
- 心跳检测

**子组件**:
- `AgentTable` - Agent 表格
- `AgentInfoCard` - 信息卡片
- `AgentMetricsCard` - 指标卡片

## API 集成

### API 客户端配置

**src/api/client.ts**:
```typescript
import axios from 'axios';
import { API_BASE_URL } from '../config/api';

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
apiClient.interceptors.request.use((config) => {
  // 添加认证 token 等
  return config;
});

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    // 错误处理
    return Promise.reject(error);
  }
);
```

### API 调用示例

**src/api/policies.ts**:
```typescript
import { apiClient } from './client';
import { Policy, CreatePolicyRequest } from '../types/policy';

export const policiesApi = {
  getAll: async (): Promise<Policy[]> => {
    return apiClient.get('/policies');
  },

  getById: async (id: number): Promise<Policy> => {
    return apiClient.get(`/policies/${id}`);
  },

  create: async (data: CreatePolicyRequest): Promise<Policy> => {
    return apiClient.post('/policies', data);
  },

  update: async (id: number, data: Partial<Policy>): Promise<Policy> => {
    return apiClient.put(`/policies/${id}`, data);
  },

  delete: async (id: number): Promise<void> => {
    return apiClient.delete(`/policies/${id}`);
  },
};
```

### 使用 React Query

**src/hooks/usePolicies.ts**:
```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { policiesApi } from '../api/policies';

export function usePolicies() {
  const queryClient = useQueryClient();

  const policiesQuery = useQuery({
    queryKey: ['policies'],
    queryFn: policiesApi.getAll,
    refetchInterval: 5000,  // 5秒刷新
  });

  const createMutation = useMutation({
    mutationFn: policiesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });

  return {
    policies: policiesQuery.data || [],
    isLoading: policiesQuery.isLoading,
    createPolicy: createMutation.mutate,
  };
}
```

## 类型定义

### Policy 类型

**src/types/policy.ts**:
```typescript
export interface Policy {
  id: number;
  rule_id: number;
  src_ip: string;
  dst_ip: string;
  dst_port: number;
  protocol: 'tcp' | 'udp' | 'icmp';
  action: 'allow' | 'deny';
  direction: 'ingress' | 'egress' | 'bidirectional';
  priority: number;
  labels?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyRequest {
  rule_id: number;
  src_ip: string;
  dst_ip: string;
  dst_port: number;
  protocol: string;
  action: string;
  direction: string;
  priority: number;
}
```

### Flow 类型

**src/types/flow.ts**:
```typescript
export interface Flow {
  id: number;
  agent_id: string;
  src_ip: string;
  dst_ip: string;
  src_port: number;
  dst_port: number;
  protocol: string;
  bytes: number;
  packets: number;
  action: 'allow' | 'deny';
  timestamp: string;
  rule_id?: number;
}

export interface FlowAggregated {
  key: string;
  total_bytes: number;
  total_packets: number;
  flow_count: number;
  protocols: Record<string, number>;
}
```

## 测试

### 单元测试

```bash
# 运行单元测试
npm run test

# 运行测试并生成覆盖率
npm run test:coverage

# UI 模式
npm run test:ui
```

### 测试示例

**src/lib/graph/__tests__/Graph.test.ts**:
```typescript
import { Graph } from '../Graph';

describe('Graph', () => {
  it('should add nodes and edges', () => {
    const graph = new Graph();
    graph.addNode({ id: 'A', name: 'Node A' });
    graph.addNode({ id: 'B', name: 'Node B' });
    graph.addEdge({ from: 'A', to: 'B', weight: 1 });

    expect(graph.getNodes()).toHaveLength(2);
    expect(graph.getEdges()).toHaveLength(1);
  });

  it('should find shortest path', () => {
    const graph = new Graph();
    // ... 添加节点和边
    const path = graph.getShortestPath('A', 'C');
    expect(path).toEqual(['A', 'B', 'C']);
  });
});
```

## 开发与构建

### 开发模式

```bash
cd web
npm install
npm run dev
```

### 生产构建

```bash
npm run build
npm run preview  # 预览构建结果
```

### 代码质量

```bash
# ESLint 检查
npm run lint

# Prettier 格式化
npm run format

# TypeScript 类型检查
tsc --noEmit
```

## 配置文件

### Vite 配置

**vite.config.ts**:
```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
```

### TypeScript 配置

**tsconfig.json**:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "jsx": "react-jsx",
    "strict": true,
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "esModuleInterop": true
  }
}
```

## 常见问题 (FAQ)

### Q1: 如何配置 API 地址？

修改 `src/config/api.ts`:
```typescript
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';
```

### Q2: 如何添加新页面？

1. 在 `src/pages/` 创建新组件
2. 在 `src/router.tsx` 添加路由
3. 在 `src/components/layout/Sidebar.tsx` 添加菜单项

### Q3: 图表不显示？

检查：
1. ECharts 数据格式是否正确
2. 使用 `SafeECharts` 组件（错误边界）
3. 查看浏览器控制台错误信息

### Q4: 如何调试 API 请求？

打开浏览器开发者工具 → Network 标签页，查看 XHR 请求

## 变更记录 (Changelog)

### [初始化] - 2025-11-27 00:02:00

- 创建 Web 模块文档
- 记录组件结构、API 集成、类型定义
- 扫描覆盖：components, pages, hooks, api, types
- 覆盖率：100%

---

**最后更新**: 2025-11-27 00:02:00
**维护者**: ebpf-based-microsegment team

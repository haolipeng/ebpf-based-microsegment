# Frontend Architecture Design Document

## Overview

This document describes the frontend architecture for the eBPF-based Microsegmentation Web UI. The frontend is a modern single-page application (SPA) built with React 19 and TypeScript, designed to visualize network flows, manage security policies, and display real-time traffic topology for Kubernetes and Docker environments.

## Technology Stack

### Core Framework

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 19.x | UI framework with hooks-based architecture |
| TypeScript | 5.9.x | Type-safe JavaScript |
| Vite | 7.x | Build tool and dev server |
| React Router | 7.x | Client-side routing |

### State & Data Management

| Technology | Version | Purpose |
|------------|---------|---------|
| TanStack Query | 5.x | Server state management, caching, and synchronization |
| Zustand | 5.x | Client-side state management |
| Axios | 1.x | HTTP client for REST API |

### UI & Visualization

| Technology | Version | Purpose |
|------------|---------|---------|
| Ant Design | 5.x | UI component library |
| ECharts | 6.x | Data visualization (charts, topology graph) |
| echarts-for-react | 3.x | React wrapper for ECharts |

### Development Tools

| Technology | Purpose |
|------------|---------|
| ESLint 9 | Code linting |
| Prettier | Code formatting |
| Vitest | Unit testing |
| Testing Library | Component testing |

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Web Browser                                    │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                        React Application                            │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │                      Pages (Routes)                           │  │ │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ │  │ │
│  │  │  │Dashboard│ │ Flows   │ │Topology │ │Policies │ │ Agents  │ │  │ │
│  │  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ │  │ │
│  │  └───────┼──────────┼──────────┼──────────┼──────────┼─────────┘  │ │
│  │          │          │          │          │          │            │ │
│  │  ┌───────┴──────────┴──────────┴──────────┴──────────┴─────────┐  │ │
│  │  │                       Components                             │  │ │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐               │  │ │
│  │  │  │  Layout    │ │   Charts   │ │  Topology  │               │  │ │
│  │  │  │ (Header,   │ │(Trend,Pie, │ │ (Graph,    │               │  │ │
│  │  │  │  Sidebar)  │ │ Bar,Donut) │ │  Legend,   │               │  │ │
│  │  │  │            │ │            │ │  Controls) │               │  │ │
│  │  │  └────────────┘ └────────────┘ └────────────┘               │  │ │
│  │  └─────────────────────────┬─────────────────────────────────────┘  │ │
│  │                            │                                        │ │
│  │  ┌─────────────────────────┴─────────────────────────────────────┐  │ │
│  │  │                        Hooks Layer                             │  │ │
│  │  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────────┐  │  │ │
│  │  │  │ useFlows  │ │useTopology│ │usePolicies│ │useFlowStream  │  │  │ │
│  │  │  │           │ │           │ │           │ │  (WebSocket)  │  │  │ │
│  │  │  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └───────┬───────┘  │  │ │
│  │  └────────┼─────────────┼─────────────┼───────────────┼──────────┘  │ │
│  │           │             │             │               │             │ │
│  │  ┌────────┴─────────────┴─────────────┴───────────────┴──────────┐  │ │
│  │  │                     TanStack Query                             │  │ │
│  │  │            (Server State, Caching, Refetching)                 │  │ │
│  │  └────────────────────────────┬──────────────────────────────────┘  │ │
│  │                               │                                     │ │
│  │  ┌────────────────────────────┴──────────────────────────────────┐  │ │
│  │  │                      API Client (Axios)                        │  │ │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │  │ │
│  │  │  │ flows.ts │ │policies.ts│ │agents.ts │ │   WebSocket      │  │  │ │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │  │ │
│  │  └────────────────────────────┬──────────────────────────────────┘  │ │
│  └───────────────────────────────┼─────────────────────────────────────┘ │
└──────────────────────────────────┼───────────────────────────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │    Backend Server (Go)      │
                    │    REST API + WebSocket     │
                    │    http://server:8080/api   │
                    └─────────────────────────────┘
```

## Directory Structure

```
web/src/
├── api/                    # API layer
│   ├── client.ts          # Axios instance with interceptors
│   ├── flows.ts           # Flow API endpoints
│   ├── policies.ts        # Policy API endpoints
│   └── agents.ts          # Agent API endpoints
│
├── components/            # Reusable UI components
│   ├── common/           # Shared components
│   │   └── SafeECharts.tsx
│   │
│   ├── layout/           # App shell components
│   │   ├── Header.tsx
│   │   ├── Sidebar.tsx
│   │   └── MainLayout.tsx
│   │
│   ├── topology/         # Network topology visualization
│   │   ├── TopologyGraph.tsx      # Main ECharts graph
│   │   ├── TopologyControls.tsx   # Filter & view mode controls
│   │   ├── TopologyLegend.tsx     # Dynamic legend
│   │   ├── NodeDetailPanel.tsx    # Node detail drawer
│   │   ├── SessionDetail.tsx      # Session details
│   │   └── topologyConfig.ts      # ECharts configuration
│   │
│   ├── flows/            # Flow list & filtering
│   │   ├── FlowTable.tsx
│   │   ├── FlowFilters.tsx
│   │   ├── FlowSummaryCards.tsx
│   │   ├── ProcessStats.tsx
│   │   └── ProtocolStats.tsx
│   │
│   ├── policies/         # Policy management
│   │   ├── PolicyTable.tsx
│   │   ├── PolicyForm.tsx
│   │   ├── PolicyFilters.tsx
│   │   ├── PolicyStatsCards.tsx
│   │   └── PolicyStatsModal.tsx
│   │
│   ├── dashboard/        # Dashboard widgets
│   │   ├── MetricCard.tsx
│   │   ├── TrafficTrendChart.tsx
│   │   ├── PolicyActionChart.tsx
│   │   ├── ProtocolChart.tsx
│   │   └── TopTalkersList.tsx
│   │
│   ├── agents/           # Agent management
│   │   ├── AgentTable.tsx
│   │   ├── AgentInfoCard.tsx
│   │   └── AgentMetricsCard.tsx
│   │
│   └── visualization/    # Chart components
│       ├── FlowTrendChart.tsx
│       ├── BytesTrendChart.tsx
│       ├── ProtocolPieChart.tsx
│       ├── ProtocolDonutChart.tsx
│       └── TopTalkersChart.tsx
│
├── config/               # Application configuration
│   └── api.ts           # API endpoint configuration
│
├── hooks/               # Custom React hooks
│   ├── useFlows.ts     # Flow data fetching
│   ├── useFlowStream.ts # WebSocket real-time flows
│   ├── useTopology.ts  # Topology aggregation
│   ├── usePolicies.ts  # Policy CRUD operations
│   ├── useAgents.ts    # Agent data fetching
│   └── useVisualization.ts
│
├── lib/                 # Utility libraries
│   └── graph/          # Graph algorithms
│       ├── Graph.ts    # Graph data structure
│       ├── algorithms.ts # Layout algorithms
│       └── types.ts
│
├── pages/              # Route pages
│   ├── Dashboard/
│   ├── Flows/
│   ├── Topology/
│   ├── Policies/
│   └── Agents/
│
├── types/              # TypeScript type definitions
│   ├── flow.ts        # Flow & process types
│   ├── topology.ts    # Topology node/edge types
│   ├── policy.ts      # Policy types
│   ├── agent.ts       # Agent types
│   └── common.ts      # Shared types
│
├── utils/              # Utility functions
│   ├── topologyUtils.ts  # Topology aggregation
│   ├── chartHelpers.ts   # Chart formatting
│   └── format.ts         # Data formatting
│
└── styles/             # CSS styles
    ├── topology.css
    └── flows.css
```

## Core Modules

### 1. Network Topology System

The topology system visualizes network traffic relationships between Kubernetes/Docker entities at multiple abstraction levels.

#### View Modes

| Mode | Description | Node Types Shown |
|------|-------------|------------------|
| NAMESPACE | Group traffic by K8s namespace | NAMESPACE, EXTERNAL |
| SERVICE | Group by service (app label) | SERVICE, IP, EXTERNAL |
| POD | Show individual pods | POD, IP, EXTERNAL |
| CONTAINER | Show containers within pods | CONTAINER, POD, EXTERNAL |
| PROCESS | Show processes within containers | PROCESS, CONTAINER, EXTERNAL |
| IP | Raw IP address view | IP, EXTERNAL |

#### Node Types

```typescript
type TopologyNodeType =
  | 'NAMESPACE'   // Kubernetes namespace
  | 'SERVICE'     // K8s service or service group
  | 'POD'         // Kubernetes pod
  | 'CONTAINER'   // Docker/K8s container
  | 'PROCESS'     // Process within container
  | 'IP'          // Raw IP address
  | 'EXTERNAL'    // External endpoint
```

#### Key Components

- **TopologyGraph.tsx**: Main ECharts force-directed graph component
- **TopologyControls.tsx**: View mode selector, filters, time range picker
- **TopologyLegend.tsx**: Dynamic legend based on current view mode
- **topologyConfig.ts**: ECharts configuration generator

#### Data Flow

```
Flow API Data
     │
     ▼
┌────────────────────┐
│   useTopology()    │  ← Aggregates flows by view mode
│                    │
│  - extractK8sInfo  │  ← Extracts namespace/pod/container from labels
│  - generateNodeId  │  ← Creates unique node IDs
│  - aggregateFlows  │  ← Groups flows into topology nodes/edges
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│  TopologyGraph     │  ← Renders ECharts graph
│                    │
│  - Force layout    │  ← Node positioning
│  - Node symbols    │  ← Different shapes per type
│  - Edge rendering  │  ← Width = traffic volume
│  - Tooltips        │  ← Detailed metrics on hover
└────────────────────┘
```

#### Node Visual Encoding

| Node Type | Shape | Color | Meaning |
|-----------|-------|-------|---------|
| NAMESPACE | Diamond | #722ED1 (Purple) | K8s namespace grouping |
| SERVICE | Circle | #1890FF (Blue) | Service/app group |
| POD | Rounded Rect | #52C41A (Green) | Individual pod |
| CONTAINER | Square | #13C2C2 (Cyan) | Container instance |
| PROCESS | Circle | #FA8C16 (Orange) | Running process |
| IP | Circle | #8C8C8C (Gray) | Raw IP address |
| EXTERNAL | Triangle | #F5222D (Red) | External endpoint |

#### Edge Visual Encoding

| Status | Color | Meaning |
|--------|-------|---------|
| Normal | #91CAFF | Standard connection |
| Bidirectional | #95DE64 | Two-way traffic |
| Denied | #FF7875 | Blocked by policy |
| Warning | #FFC069 | Security alert |

### 2. Flow Management System

The flow system provides detailed network flow inspection with filtering, searching, and real-time updates.

#### Flow Data Model

```typescript
interface Flow {
  id: string
  sourceIp: string
  sourcePort: number
  destIp: string
  destPort: number
  protocol: 'TCP' | 'UDP' | 'ICMP'
  packetCount: number
  byteCount: number
  policyAction: 'ALLOW' | 'DENY' | 'LOG'
  state: 'ACTIVE' | 'CLOSED' | 'TIMEOUT'
  direction: 'INGRESS' | 'EGRESS'
  processInfo?: ProcessInfo  // Process-level visibility
  sourceLabels?: Record<string, string>  // K8s labels
  destLabels?: Record<string, string>
}
```

#### Process Visibility

Flows include process information extracted from eBPF programs:

```typescript
interface ProcessInfo {
  pid: number
  comm: string        // Process name (16 chars max)
  exePath?: string    // Full executable path
  cmdline?: string    // Command line arguments
  containerId?: string
  isSuspicious?: boolean
}
```

#### Real-time Updates

WebSocket integration for live flow updates:

```typescript
const { isConnected, error } = useFlowStream({
  enabled: realtimeEnabled,
  onFlow: handleNewFlow,
})
```

### 3. Policy Management System

Security policy CRUD operations with label-based matching.

#### Policy Types

```typescript
interface Policy {
  id: number
  name: string
  action: 'ALLOW' | 'DENY' | 'LOG'
  priority: number
  sourceLabelSelector: string  // label-based matching
  destLabelSelector: string
  protocol?: string
  port?: number
  enabled: boolean
}
```

### 4. API Layer

#### Configuration

API endpoints are configured via environment variables:

```typescript
// config/api.ts
export const apiConfig = {
  baseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: Number(import.meta.env.VITE_API_TIMEOUT) || 30000,
  wsUrl: import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws',
}
```

#### Environment Files

```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_TIMEOUT=30000

# .env.production
VITE_API_BASE_URL=/api
VITE_WS_URL=/ws
VITE_API_TIMEOUT=30000
```

#### API Client

Axios instance with request/response interceptors:

```typescript
// api/client.ts
const apiClient = axios.create({
  baseURL: apiConfig.baseUrl,
  timeout: apiConfig.timeout,
  headers: { 'Content-Type': 'application/json' },
})

// Request interceptor - add timestamp
apiClient.interceptors.request.use(config => {
  config.headers['X-Request-Time'] = new Date().toISOString()
  return config
})

// Response interceptor - error handling
apiClient.interceptors.response.use(
  response => response,
  error => handleApiError(error)
)
```

### 5. State Management

#### Server State (TanStack Query)

Used for API data with caching, background refetching, and optimistic updates:

```typescript
// useFlows.ts
export function useFlows(query: FlowQuery) {
  return useQuery({
    queryKey: ['flows', query],
    queryFn: () => flowsApi.getFlows(query),
    refetchInterval: 30000,  // Auto-refresh every 30s
  })
}

// usePolicies.ts - with mutations
export function usePolicies() {
  const queryClient = useQueryClient()

  const createPolicy = useMutation({
    mutationFn: policiesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })

  return { policies, createPolicy, updatePolicy, deletePolicy }
}
```

#### Client State (Zustand)

Used for UI state that doesn't need server synchronization:

```typescript
// Example: topology view preferences
const useTopologyStore = create((set) => ({
  viewMode: 'SERVICE',
  selectedNode: null,
  setViewMode: (mode) => set({ viewMode: mode }),
  setSelectedNode: (node) => set({ selectedNode: node }),
}))
```

## Routing

```typescript
// Main routes
<Routes>
  <Route path="/" element={<MainLayout />}>
    <Route index element={<Navigate to="/dashboard" />} />
    <Route path="dashboard" element={<Dashboard />} />
    <Route path="flows" element={<Flows />} />
    <Route path="topology" element={<Topology />} />
    <Route path="policies" element={<Policies />} />
    <Route path="agents" element={<Agents />} />
    <Route path="agents/:id" element={<AgentDetail />} />
  </Route>
</Routes>
```

## Development Workflow

### Scripts

```bash
# Development server with HMR
npm run dev

# Type checking
npx tsc --noEmit

# Linting
npm run lint

# Formatting
npm run format

# Unit tests
npm run test

# Production build
npm run build
```

### Code Quality

- **TypeScript**: Strict mode enabled, no implicit any
- **ESLint**: React hooks rules, React Refresh rules
- **Prettier**: Consistent code formatting

## Performance Optimizations

### 1. Topology Rendering

- Debounced real-time updates (500ms aggregation window)
- Virtualized node limit (max 100-500 nodes)
- Progressive loading for large datasets

### 2. Data Caching

- TanStack Query automatic caching
- Stale-while-revalidate strategy
- Background refetching

### 3. Bundle Optimization

- Vite code splitting
- Tree shaking unused code
- Lazy loading routes

## Testing Strategy

### Unit Tests

- Component rendering tests
- Hook behavior tests
- Utility function tests

### Integration Tests

- API mocking with MSW
- Component interaction tests

### E2E Tests

- User flow testing (future)

## Security Considerations

### 1. API Security

- No sensitive data in frontend code
- Environment-based API URLs
- Error messages don't leak internals

### 2. Input Validation

- Filter inputs sanitized
- SQL injection prevention in API
- XSS prevention via React

### 3. Authentication

- JWT token support (planned)
- Session management (planned)

## Future Roadmap

### Short-term

- [ ] Export topology as image/JSON
- [ ] Saved filter presets
- [ ] Policy simulation mode

### Medium-term

- [ ] Multi-cluster support
- [ ] Role-based access control
- [ ] Audit logging

### Long-term

- [ ] AI-powered policy recommendations
- [ ] Anomaly detection visualization
- [ ] Custom dashboard builder

## Conclusion

The frontend architecture is designed for scalability, maintainability, and performance. The modular component structure, typed data models, and clear separation of concerns enable rapid development while maintaining code quality. The topology visualization system provides deep visibility into Kubernetes and Docker network traffic, helping operators understand and secure their container infrastructure.

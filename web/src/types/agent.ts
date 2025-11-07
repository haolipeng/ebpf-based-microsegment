// Agent information
export interface Agent {
  id: string
  hostname: string
  ipAddress: string
  version: string
  status: 'online' | 'offline' | 'error'
  lastHeartbeat: string
  startTime: string
  metrics?: AgentMetrics
}

// Agent metrics
export interface AgentMetrics {
  cpuUsage: number // percentage
  memoryUsage: number // bytes
  flowsReported: number
  activePolicies: number
  packetsProcessed?: number
  packetsDropped?: number
}

// Agent list response
export interface AgentListResponse {
  agents: Agent[]
  total: number
}

// Agent health check
export interface AgentHealthCheck {
  healthy: boolean
  message?: string
  timestamp: string
}

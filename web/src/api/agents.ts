import apiClient from './client'
import type { Agent, AgentListResponse } from '../types/agent'

// Server response type (Go struct with capitalized fields)
interface ServerAgent {
  AgentID: string
  Hostname: string
  Version: string
  Interface: string
  IPAddresses: string[]
  OS: string
  KernelVersion: string
  StartTime: number
  LastHeartbeat: number
  Status: string
  CPUUsage: number
  MemoryUsage: number
  PacketsProcessed: number
  ActiveSessions: number
  FlowsReported: number
  ActivePolicies: number
}

// Transform server agent to frontend agent
function transformAgent(serverAgent: ServerAgent): Agent {
  return {
    id: serverAgent.AgentID,
    hostname: serverAgent.Hostname,
    ipAddress: serverAgent.IPAddresses[0] || '',
    version: serverAgent.Version,
    status: serverAgent.Status === 'active' ? 'online' : 'offline',
    lastHeartbeat: new Date(serverAgent.LastHeartbeat / 1000000).toISOString(),
    startTime: new Date(serverAgent.StartTime / 1000000).toISOString(),
    metrics: {
      cpuUsage: serverAgent.CPUUsage,
      memoryUsage: serverAgent.MemoryUsage,
      flowsReported: serverAgent.FlowsReported,
      activePolicies: serverAgent.ActivePolicies,
      packetsProcessed: serverAgent.PacketsProcessed,
    }
  }
}

export const agentsApi = {
  // List all agents
  list: async (): Promise<AgentListResponse> => {
    const response = await apiClient.get<ServerAgent[]>('/v1/agents')
    // Server returns array directly, transform to expected format
    const agents = response.data.map(transformAgent)
    return {
      agents,
      total: agents.length
    }
  },

  // Get agent by ID
  get: async (id: string): Promise<Agent> => {
    const response = await apiClient.get<Agent>(`/v1/agents/${id}`)
    return response.data
  },

  // Get agent health status
  health: async (): Promise<{ healthy: boolean }> => {
    // Health endpoint is at root, not under /api
    const response = await apiClient.get<{ status: string; service: string; version: string }>('http://10.107.12.201:8080/health')
    // Transform server response to expected format
    return {
      healthy: response.data.status === 'healthy'
    }
  },
}

import apiClient from './client'
import type { Agent, AgentListResponse } from '../types/agent'

export const agentsApi = {
  // List all agents
  list: async (): Promise<AgentListResponse> => {
    const response = await apiClient.get<AgentListResponse>('/agents')
    return response.data
  },

  // Get agent by ID
  get: async (id: string): Promise<Agent> => {
    const response = await apiClient.get<Agent>(`/agents/${id}`)
    return response.data
  },

  // Get agent health status
  health: async (): Promise<{ healthy: boolean }> => {
    const response = await apiClient.get<{ healthy: boolean }>('/health')
    return response.data
  },
}

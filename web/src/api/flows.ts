import apiClient from './client'
import type { Flow, FlowQuery, FlowSummary } from '../types/flow'

export const flowsApi = {
  // Query flows with filters
  query: async (params: FlowQuery): Promise<Flow[]> => {
    const response = await apiClient.get<Flow[]>('/flows', { params })
    return response.data
  },

  // Get flow summary statistics
  summary: async (startTime?: string, endTime?: string): Promise<FlowSummary> => {
    const response = await apiClient.get<FlowSummary>('/flows/summary', {
      params: { start_time: startTime, end_time: endTime },
    })
    return response.data
  },

  // Get active flows
  active: async (): Promise<Flow[]> => {
    const response = await apiClient.get<Flow[]>('/flows/active')
    return response.data
  },
}

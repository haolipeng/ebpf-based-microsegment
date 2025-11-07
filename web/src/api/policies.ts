import apiClient from './client'
import type { Policy, PolicyListResponse, PolicyStats } from '../types/policy'

export const policiesApi = {
  // List all policies
  list: async (): Promise<PolicyListResponse> => {
    const response = await apiClient.get<PolicyListResponse>('/policies')
    return response.data
  },

  // Get policy by ID
  get: async (ruleId: number): Promise<Policy> => {
    const response = await apiClient.get<Policy>(`/policies/${ruleId}`)
    return response.data
  },

  // Create new policy
  create: async (policy: Omit<Policy, 'ruleId'>): Promise<Policy> => {
    const response = await apiClient.post<Policy>('/policies', policy)
    return response.data
  },

  // Update existing policy
  update: async (ruleId: number, policy: Partial<Policy>): Promise<Policy> => {
    const response = await apiClient.put<Policy>(`/policies/${ruleId}`, policy)
    return response.data
  },

  // Delete policy
  delete: async (ruleId: number): Promise<void> => {
    await apiClient.delete(`/policies/${ruleId}`)
  },

  // Get policy statistics
  stats: async (ruleId: number): Promise<PolicyStats> => {
    const response = await apiClient.get<PolicyStats>(`/policies/${ruleId}/stats`)
    return response.data
  },
}

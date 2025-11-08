import apiClient from './client'
import type { Policy, PolicyListResponse, PolicyStats } from '../types/policy'

// Server response type
interface ServerPolicy {
  rule_id?: number
  src_ip: string
  dst_ip: string
  src_port?: number
  dst_port?: number
  protocol: number
  action?: string
  priority?: number
  created_at: number
  updated_at: number
  description?: string
}

interface ServerPolicyListResponse {
  policies: ServerPolicy[]
  version: number
}

// Protocol number to name mapping
function protocolNumberToName(protocol: number): 'tcp' | 'udp' | 'icmp' | 'any' {
  switch (protocol) {
    case 6: return 'tcp'
    case 17: return 'udp'
    case 1: return 'icmp'
    default: return 'any'
  }
}

// Transform server policy to frontend policy
function transformPolicy(serverPolicy: ServerPolicy, index: number): Policy {
  return {
    ruleId: serverPolicy.rule_id || index,
    srcIp: serverPolicy.src_ip,
    dstIp: serverPolicy.dst_ip,
    srcPort: serverPolicy.src_port || 0,
    dstPort: serverPolicy.dst_port || 0,
    protocol: protocolNumberToName(serverPolicy.protocol),
    action: (serverPolicy.action || 'allow') as 'allow' | 'deny' | 'log',
    priority: serverPolicy.priority || 0,
    description: serverPolicy.description,
    createdAt: new Date(serverPolicy.created_at / 1000000).toISOString(),
    updatedAt: new Date(serverPolicy.updated_at / 1000000).toISOString(),
  }
}

export const policiesApi = {
  // List all policies
  list: async (): Promise<PolicyListResponse> => {
    const response = await apiClient.get<ServerPolicyListResponse>('/v1/policies')
    const policies = response.data.policies.map(transformPolicy)
    return {
      policies,
      total: policies.length
    }
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

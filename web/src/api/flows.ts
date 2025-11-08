import apiClient from './client'
import type { Flow, FlowQuery, FlowSummary } from '../types/flow'

// Server response type for flow summary (snake_case from Go)
interface ServerFlowSummary {
  total_flows: number
  total_packets: number
  total_bytes: number
  unique_source_ips: number
  unique_dest_ips: number
  avg_duration_ms: number
  top_protocols: Array<{
    protocol: number
    count: number
    bytes: number
  }> | null
}

// Transform server flow summary to frontend format
function transformFlowSummary(serverSummary: ServerFlowSummary): FlowSummary {
  // Protocol number to name mapping
  const protocolName = (proto: number): string => {
    switch (proto) {
      case 6: return 'TCP'
      case 17: return 'UDP'
      case 1: return 'ICMP'
      default: return `Protocol ${proto}`
    }
  }

  return {
    totalFlows: serverSummary.total_flows || 0,
    activeFlows: 0, // Not provided by current API
    closedFlows: 0, // Not provided by current API
    totalPackets: serverSummary.total_packets || 0,
    totalBytes: serverSummary.total_bytes || 0,
    allowedFlows: 0, // Not provided by current API
    deniedFlows: 0, // Not provided by current API
    topProtocols: (serverSummary.top_protocols || []).map(p => ({
      protocol: protocolName(p.protocol),
      flowCount: p.count,
      packetCount: 0, // Not provided
      byteCount: p.bytes,
    })),
    topSourceIps: [], // Not provided by current API
    topDestIps: [], // Not provided by current API
  }
}

// Server response type for flow query
interface ServerFlowQueryResponse {
  flows: any[] // Server returns flows array
  has_more: boolean
  limit: number
  offset: number
  total_count: number
}

export const flowsApi = {
  // Query flows with filters
  query: async (params: FlowQuery): Promise<Flow[]> => {
    const response = await apiClient.get<ServerFlowQueryResponse>('/v1/flows', { params })
    // Server returns object with flows array, extract it
    return response.data.flows || []
  },

  // Get flow summary statistics
  summary: async (startTime?: string, endTime?: string): Promise<FlowSummary> => {
    const response = await apiClient.get<ServerFlowSummary>('/v1/flows/summary', {
      params: { start_time: startTime, end_time: endTime },
    })
    return transformFlowSummary(response.data)
  },

  // Get active flows
  active: async (): Promise<Flow[]> => {
    const response = await apiClient.get<Flow[]>('/v1/flows/active')
    return response.data
  },
}

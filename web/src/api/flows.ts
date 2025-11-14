import apiClient from './client'
import type { Flow, FlowQuery, FlowSummary } from '../types/flow'

// Server response type for flow summary (snake_case from Go)
interface ServerFlowSummary {
  total_flows: number
  active_flows: number
  closed_flows: number
  total_packets: number
  total_bytes: number
  allowed_flows: number
  denied_flows: number
  unique_source_ips: number
  unique_dest_ips: number
  avg_duration_ms: number
  top_protocols: Array<{
    protocol: string // Protocol number as string from backend
    count: number
    bytes: number
  }> | null
}

// Protocol number to name mapping (IANA protocol numbers)
export const protocolName = (proto: string | number): string => {
  const protoNum = typeof proto === 'string' ? parseInt(proto, 10) : proto
  switch (protoNum) {
    case 1: return 'ICMP'
    case 2: return 'IGMP'
    case 6: return 'TCP'
    case 17: return 'UDP'
    case 41: return 'IPv6'
    case 47: return 'GRE'
    case 50: return 'ESP'
    case 51: return 'AH'
    case 58: return 'ICMPv6'
    case 89: return 'OSPF'
    case 132: return 'SCTP'
    default: return isNaN(protoNum) ? 'Unknown' : `Protocol ${protoNum}`
  }
}

// Transform server flow summary to frontend format
function transformFlowSummary(serverSummary: ServerFlowSummary): FlowSummary {

  return {
    totalFlows: serverSummary.total_flows || 0,
    activeFlows: serverSummary.active_flows || 0,
    closedFlows: serverSummary.closed_flows || 0,
    totalPackets: serverSummary.total_packets || 0,
    totalBytes: serverSummary.total_bytes || 0,
    allowedFlows: serverSummary.allowed_flows || 0,
    deniedFlows: serverSummary.denied_flows || 0,
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

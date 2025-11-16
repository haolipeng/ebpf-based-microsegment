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

// Server flow type (snake_case from backend)
interface ServerFlow {
  id: number | string
  src_ip: string
  src_port: number
  dst_ip: string
  dst_port: number
  protocol: number | string
  direction: number | string
  packet_count: number
  byte_count: number
  start_time?: number | string
  end_time?: number | string
  last_seen?: number | string
  source_labels?: Record<string, string>
  dest_labels?: Record<string, string>
  policy_id?: number
  policy_action?: number | string
  state?: number | string
  agent_id?: string
}

// Transform server flow to frontend flow format
function transformFlow(serverFlow: ServerFlow): Flow {
  // Convert protocol number to name using the protocolName helper
  const protocolNum = typeof serverFlow.protocol === 'string' 
    ? parseInt(serverFlow.protocol, 10) 
    : serverFlow.protocol
  const protocol = protocolName(protocolNum)

  // Convert state number to string
  const stateMap: Record<number, Flow['state']> = {
    1: 'ACTIVE',
    2: 'CLOSED',
    3: 'TIMEOUT',
  }
  const state = typeof serverFlow.state === 'number' 
    ? (stateMap[serverFlow.state] || 'CLOSED')
    : (serverFlow.state as Flow['state'] || 'CLOSED')

  // Convert policy action number to string
  const actionMap: Record<number, Flow['policyAction']> = {
    1: 'ALLOW',
    2: 'DENY',
    3: 'LOG',
  }
  const policyAction = typeof serverFlow.policy_action === 'number'
    ? (actionMap[serverFlow.policy_action] || 'ALLOW')
    : (serverFlow.policy_action as Flow['policyAction'] || 'ALLOW')

  // Convert direction number to string
  const directionMap: Record<number, Flow['direction']> = {
    1: 'INGRESS',
    2: 'EGRESS',
    0: 'UNKNOWN',
  }
  const direction = typeof serverFlow.direction === 'number'
    ? (directionMap[serverFlow.direction] || 'UNKNOWN')
    : (serverFlow.direction as Flow['direction'] || 'UNKNOWN')

  // Convert timestamps (backend returns nanoseconds, divide by 1e9 to get milliseconds)
  const startTime = serverFlow.start_time 
    ? (typeof serverFlow.start_time === 'number' 
        ? new Date(serverFlow.start_time / 1000000).toISOString() // Already in microseconds from backend
        : serverFlow.start_time)
    : new Date().toISOString()
  
  const endTime = serverFlow.end_time
    ? (typeof serverFlow.end_time === 'number'
        ? new Date(serverFlow.end_time / 1000000).toISOString()
        : serverFlow.end_time)
    : undefined

  const lastSeen = serverFlow.last_seen
    ? (typeof serverFlow.last_seen === 'number'
        ? new Date(serverFlow.last_seen / 1000000).toISOString()
        : serverFlow.last_seen)
    : startTime

  return {
    id: String(serverFlow.id),
    sourceIp: serverFlow.src_ip,
    sourcePort: serverFlow.src_port,
    destIp: serverFlow.dst_ip,
    destPort: serverFlow.dst_port,
    protocol: protocol,
    packetCount: serverFlow.packet_count,
    byteCount: serverFlow.byte_count,
    durationMs: endTime ? (new Date(endTime).getTime() - new Date(startTime).getTime()) : 0,
    startTime,
    endTime,
    lastSeen,
    sourceLabels: serverFlow.source_labels,
    destLabels: serverFlow.dest_labels,
    policyId: serverFlow.policy_id,
    policyAction,
    state,
    direction,
    eventType: 'NEW', // Default, not provided by backend
  }
}

export const flowsApi = {
  // Query flows with filters
  query: async (params: FlowQuery): Promise<Flow[]> => {
    // Transform camelCase params to snake_case for backend API
    const apiParams: Record<string, any> = {}
    if (params.startTime) apiParams.start_time = params.startTime
    if (params.endTime) apiParams.end_time = params.endTime
    if (params.sourceIp) apiParams.source_ip = params.sourceIp
    if (params.destIp) apiParams.dest_ip = params.destIp
    if (params.protocol) apiParams.protocol = params.protocol
    if (params.state) apiParams.state = params.state
    if (params.direction) apiParams.direction = params.direction
    if (params.policyAction) apiParams.policy_action = params.policyAction
    // Set default limit if not provided
    apiParams.limit = params.limit || 1000  // Default to 1000 for topology to get more data
    if (params.offset) apiParams.offset = params.offset
    // Ignore viewMode and maxNodes as they are frontend-only filters
    
    const response = await apiClient.get<ServerFlowQueryResponse>('/v1/flows', { params: apiParams })
    // Server returns object with flows array, transform each flow
    const serverFlows = response.data.flows || []
    return serverFlows.map(transformFlow)
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

// Network flow data
export interface Flow {
  id: string
  sourceIp: string
  sourcePort: number
  destIp: string
  destPort: number
  protocol: 'TCP' | 'UDP' | 'ICMP' | string
  packetCount: number
  byteCount: number
  durationMs: number
  startTime: string
  endTime?: string
  lastSeen: string
  sourceLabels?: Record<string, string>
  destLabels?: Record<string, string>
  policyId?: number
  policyAction: 'ALLOW' | 'DENY' | 'LOG'
  state: 'ACTIVE' | 'CLOSED' | 'TIMEOUT'
  direction: 'INGRESS' | 'EGRESS' | 'UNKNOWN'
  eventType: 'NEW' | 'UPDATE' | 'CLOSED' | 'TIMEOUT'
}

// Flow query parameters
export interface FlowQuery {
  startTime?: string
  endTime?: string
  sourceIp?: string
  destIp?: string
  protocol?: string
  state?: string
  direction?: string
  policyAction?: string
  limit?: number
  offset?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

// Flow summary statistics
export interface FlowSummary {
  totalFlows: number
  activeFlows: number
  closedFlows: number
  totalPackets: number
  totalBytes: number
  allowedFlows: number
  deniedFlows: number
  topProtocols: ProtocolStats[]
  topSourceIps: IpStats[]
  topDestIps: IpStats[]
}

// Protocol statistics
export interface ProtocolStats {
  protocol: string
  flowCount: number
  packetCount: number
  byteCount: number
}

// IP address statistics
export interface IpStats {
  ip: string
  flowCount: number
  packetCount: number
  byteCount: number
}

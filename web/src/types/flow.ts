// Process information extracted from eBPF programs
export interface ProcessInfo {
  pid: number
  ppid?: number
  uid?: number
  gid?: number
  comm: string // Process command name (16 bytes, kernel truncated)
  exePath?: string // Full executable path from /proc/<pid>/exe
  cmdline?: string // Command line arguments
  startTime?: number // Process start timestamp (Unix nanoseconds)
  isSuspicious?: boolean // Marked by security validator
  containerId?: string // Container ID from cgroup path
}

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
  processInfo?: ProcessInfo // Process that generated this flow
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
  // Process filtering fields (Issue #53)
  pid?: number
  processName?: string // Filter by process command name
  processPath?: string // Filter by process executable path
  containerId?: string // Filter by container ID
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

// Process statistics (Issue #53)
export interface ProcessStats {
  processName: string // Process command name
  processPath?: string // Full executable path
  flowCount: number
  connectionCount: number
  packetCount: number
  byteCount: number
  containerCount?: number // Number of unique containers
  isSuspicious?: boolean
}

// Container statistics
export interface ContainerStats {
  containerId: string
  processCount: number
  flowCount: number
  byteCount: number
}

// Network policy rule
export interface Policy {
  ruleId: number
  srcIp: string // CIDR notation
  dstIp: string // CIDR notation
  srcPort: number // 0 for any
  dstPort: number // 0 for any
  protocol: 'tcp' | 'udp' | 'icmp' | 'any'
  action: 'allow' | 'deny' | 'log'
  priority: number
  enabled?: boolean
  description?: string
  createdAt?: string
  updatedAt?: string
}

// Policy statistics
export interface PolicyStats {
  ruleId: number
  hitCount: number
  lastHit?: string
}

// Policy list response
export interface PolicyListResponse {
  policies: Policy[]
  total: number
}

// Policy validation result
export interface PolicyValidation {
  valid: boolean
  errors?: string[]
  warnings?: string[]
}

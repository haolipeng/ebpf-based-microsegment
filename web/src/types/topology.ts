import type { Flow, FlowQuery, ProcessInfo } from './flow'

/**
 * Topology node types supporting K8s/Docker concepts
 */
export type TopologyNodeType =
  | 'NAMESPACE'   // Kubernetes namespace
  | 'SERVICE'     // Kubernetes service or service group
  | 'POD'         // Kubernetes pod
  | 'CONTAINER'   // Docker/K8s container
  | 'PROCESS'     // Process within container
  | 'IP'          // Raw IP address (external or unknown)
  | 'EXTERNAL'    // External endpoint (outside cluster)

/**
 * Topology view modes for different aggregation levels
 */
export type TopologyViewMode =
  | 'NAMESPACE'   // Group by namespace
  | 'SERVICE'     // Group by service (labels.app)
  | 'POD'         // Show individual pods
  | 'CONTAINER'   // Show containers within pods
  | 'PROCESS'     // Show processes within containers
  | 'IP'          // Raw IP view (legacy)

/**
 * Container runtime type
 */
export type ContainerRuntime = 'docker' | 'containerd' | 'cri-o' | 'unknown'

/**
 * Kubernetes metadata for a node
 */
export interface K8sMetadata {
  /** Kubernetes namespace */
  namespace?: string
  /** Pod name */
  podName?: string
  /** Pod UID */
  podUid?: string
  /** Container name */
  containerName?: string
  /** Container ID (short form) */
  containerId?: string
  /** Container runtime */
  containerRuntime?: ContainerRuntime
  /** Service name (from labels.app or labels.service) */
  serviceName?: string
  /** Deployment/StatefulSet/DaemonSet name */
  workloadName?: string
  /** Workload type */
  workloadType?: 'Deployment' | 'StatefulSet' | 'DaemonSet' | 'ReplicaSet' | 'Job' | 'CronJob'
  /** Node name (K8s node, not topology node) */
  nodeName?: string
  /** All labels from the resource */
  labels?: Record<string, string>
  /** Annotations */
  annotations?: Record<string, string>
}

/**
 * Traffic metrics for nodes and edges
 */
export interface TrafficMetrics {
  /** Number of flows */
  flowCount: number
  /** Number of active flows */
  activeFlows: number
  /** Total packets */
  packetCount: number
  /** Total bytes */
  byteCount: number
  /** Bytes per second (calculated) */
  bytesPerSecond?: number
  /** Packets per second (calculated) */
  packetsPerSecond?: number
  /** Connection count (unique src-dst pairs) */
  connectionCount?: number
}

/**
 * Security status for a node
 */
export interface SecurityStatus {
  /** Number of allowed flows */
  allowedFlows: number
  /** Number of denied flows */
  deniedFlows: number
  /** Number of logged flows */
  loggedFlows: number
  /** Has suspicious processes */
  hasSuspiciousProcess?: boolean
  /** Policy violation count */
  policyViolations?: number
}

/**
 * Topology node definition with K8s/Docker support
 */
export interface TopologyNode {
  /** Unique node identifier */
  id: string
  /** Display label */
  label: string
  /** Node type */
  type: TopologyNodeType
  /** Traffic metrics */
  metrics: TrafficMetrics
  /** Security status */
  security?: SecurityStatus
  /** K8s/Docker metadata */
  k8s?: K8sMetadata
  /** Process info (for PROCESS type nodes) */
  processInfo?: ProcessInfo
  /** IP address (for IP/EXTERNAL type or as additional info) */
  ipAddress?: string
  /** Port (for service endpoints) */
  port?: number
  /** Parent node ID (for hierarchical views) */
  parentId?: string
  /** Child node IDs */
  childIds?: string[]
  /** Is this node a group/cluster */
  isGroup?: boolean
  /** Number of children (for collapsed groups) */
  childCount?: number
  /** Health status */
  health?: 'healthy' | 'warning' | 'critical' | 'unknown'
  /** Optional fixed position */
  position?: { x: number; y: number }
  /** Is node expanded (for groups) */
  expanded?: boolean
}

/**
 * Edge protocol information
 */
export interface EdgeProtocol {
  /** Protocol name */
  name: string
  /** Port number */
  port: number
  /** Flow count for this protocol/port */
  flowCount: number
  /** Bytes for this protocol/port */
  byteCount: number
}

/**
 * Topology edge definition
 */
export interface TopologyEdge {
  /** Unique edge identifier */
  id: string
  /** Source node ID */
  source: string
  /** Target node ID */
  target: string
  /** Traffic metrics */
  metrics: TrafficMetrics
  /** Security status */
  security?: SecurityStatus
  /** Protocol details */
  protocols: EdgeProtocol[]
  /** Primary direction */
  direction: 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL'
  /** Is bidirectional traffic detected */
  isBidirectional?: boolean
  /** Edge style (for visualization) */
  style?: 'solid' | 'dashed' | 'dotted'
  /** Edge color hint based on security */
  colorHint?: 'normal' | 'allowed' | 'denied' | 'warning'
}

/**
 * Node group for hierarchical display
 */
export interface TopologyGroup {
  /** Group ID */
  id: string
  /** Group label */
  label: string
  /** Group type (same as node type) */
  type: TopologyNodeType
  /** Contained node IDs */
  nodeIds: string[]
  /** K8s metadata */
  k8s?: K8sMetadata
  /** Aggregated metrics */
  metrics: TrafficMetrics
  /** Is expanded */
  expanded: boolean
  /** Parent group ID (for nested groups) */
  parentId?: string
}

/**
 * Complete topology data structure
 */
export interface TopologyData {
  /** All nodes */
  nodes: TopologyNode[]
  /** All edges */
  edges: TopologyEdge[]
  /** Groups for hierarchical view */
  groups?: TopologyGroup[]
  /** Statistics */
  stats: TopologyStats
  /** Current view mode */
  viewMode: TopologyViewMode
  /** Data timestamp */
  timestamp?: string
}

/**
 * Topology statistics
 */
export interface TopologyStats {
  /** Total node count */
  totalNodes: number
  /** Total edge count */
  totalEdges: number
  /** Total flow count */
  totalFlows: number
  /** Active flow count */
  activeFlows: number
  /** Total bytes */
  totalBytes: number
  /** Namespace count */
  namespaceCount?: number
  /** Service count */
  serviceCount?: number
  /** Pod count */
  podCount?: number
  /** Container count */
  containerCount?: number
  /** Process count */
  processCount?: number
  /** External endpoint count */
  externalCount?: number
}

/**
 * Topology filter options
 */
export interface TopologyFilters extends FlowQuery {
  /** View mode */
  viewMode: TopologyViewMode
  /** Max nodes to display */
  maxNodes?: number
  /** Filter by namespace */
  namespace?: string
  /** Filter by service name */
  service?: string
  /** Filter by pod name pattern */
  podPattern?: string
  /** Show external connections */
  showExternal?: boolean
  /** Show only suspicious */
  onlySuspicious?: boolean
  /** Min flow count threshold */
  minFlowCount?: number
  /** Time range in minutes */
  timeRangeMinutes?: number
}

/**
 * Node detail information
 */
export interface NodeDetail {
  /** Node information */
  node: TopologyNode
  /** Related flows */
  flows: Flow[]
  /** Inbound connection count */
  inboundConnections: number
  /** Outbound connection count */
  outboundConnections: number
  /** Connected node IDs (inbound) */
  inboundNodeIds?: string[]
  /** Connected node IDs (outbound) */
  outboundNodeIds?: string[]
  /** Child nodes (for groups) */
  children?: TopologyNode[]
  /** Timeline data */
  timeline?: {
    timestamp: string
    flowCount: number
    byteCount: number
  }[]
}

/**
 * Layout configuration
 */
export interface TopologyLayoutConfig {
  /** Layout algorithm */
  algorithm: 'force' | 'dagre' | 'circular' | 'grid' | 'hierarchical'
  /** Group layout mode */
  groupLayout?: 'nested' | 'swimlane' | 'clustered'
  /** Force layout parameters */
  force?: {
    repulsion: number
    gravity: number
    edgeLength: number
    friction: number
  }
  /** Spacing between groups */
  groupSpacing?: number
  /** Spacing between nodes in same group */
  nodeSpacing?: number
}

/**
 * Real-time update event
 */
export interface TopologyUpdateEvent {
  /** Event type */
  type: 'NODE_ADD' | 'NODE_UPDATE' | 'NODE_REMOVE' | 'EDGE_ADD' | 'EDGE_UPDATE' | 'EDGE_REMOVE'
  /** Affected node/edge ID */
  id: string
  /** New/updated data */
  data?: TopologyNode | TopologyEdge
  /** Timestamp */
  timestamp: string
}

/**
 * Helper type: Extract K8s info from flow labels
 */
export interface ExtractedK8sInfo {
  namespace: string
  serviceName: string
  podName: string
  containerName?: string
  containerId?: string
  workloadName?: string
  workloadType?: string
}

/**
 * Parse container ID to extract runtime and short ID
 */
export function parseContainerId(fullId: string): { runtime: ContainerRuntime; shortId: string } {
  if (!fullId) {
    return { runtime: 'unknown', shortId: '' }
  }

  // Format: docker://abc123... or containerd://abc123...
  const match = fullId.match(/^(docker|containerd|cri-o):\/\/(.+)$/)
  if (match) {
    return {
      runtime: match[1] as ContainerRuntime,
      shortId: match[2].substring(0, 12),
    }
  }

  // Just a raw ID
  return {
    runtime: 'unknown',
    shortId: fullId.substring(0, 12),
  }
}

/**
 * Extract K8s info from flow labels
 */
export function extractK8sInfo(labels?: Record<string, string>): ExtractedK8sInfo | null {
  if (!labels) {
    return null
  }

  const namespace = labels['kubernetes.io/namespace'] || labels['namespace'] || labels['k8s.namespace'] || ''
  const serviceName = labels['app'] || labels['app.kubernetes.io/name'] || labels['service'] || ''
  const podName = labels['pod'] || labels['kubernetes.io/pod-name'] || labels['k8s.pod.name'] || ''
  const containerName = labels['container'] || labels['io.kubernetes.container.name'] || ''
  const containerId = labels['containerId'] || labels['io.kubernetes.docker.type'] || ''
  const workloadName = labels['deployment'] || labels['statefulset'] || labels['daemonset'] || ''
  const workloadType = labels['deployment'] ? 'Deployment' : labels['statefulset'] ? 'StatefulSet' : labels['daemonset'] ? 'DaemonSet' : ''

  if (!namespace && !serviceName && !podName) {
    return null
  }

  return {
    namespace,
    serviceName,
    podName,
    containerName,
    containerId,
    workloadName,
    workloadType,
  }
}

/**
 * Generate unique node ID based on type and identifiers
 */
export function generateNodeId(type: TopologyNodeType, identifiers: Record<string, string>): string {
  switch (type) {
    case 'NAMESPACE':
      return `ns:${identifiers.namespace || 'default'}`
    case 'SERVICE':
      return `svc:${identifiers.namespace || 'default'}/${identifiers.service || 'unknown'}`
    case 'POD':
      return `pod:${identifiers.namespace || 'default'}/${identifiers.pod || 'unknown'}`
    case 'CONTAINER':
      return `container:${identifiers.containerId || identifiers.containerName || 'unknown'}`
    case 'PROCESS':
      return `proc:${identifiers.containerId || 'host'}/${identifiers.pid || '0'}`
    case 'IP':
      return `ip:${identifiers.ip || '0.0.0.0'}`
    case 'EXTERNAL':
      return `ext:${identifiers.ip || 'unknown'}`
    default:
      return `unknown:${Object.values(identifiers).join('-')}`
  }
}

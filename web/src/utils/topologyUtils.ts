import type { Flow, ProcessInfo } from '../types/flow'
import type {
  TopologyData,
  TopologyNode,
  TopologyEdge,
  TopologyViewMode,
  TopologyFilters,
  TopologyStats,
  TopologyGroup,
  TopologyNodeType,
  TrafficMetrics,
  SecurityStatus,
  K8sMetadata,
  EdgeProtocol,
} from '../types/topology'
import { extractK8sInfo, generateNodeId, parseContainerId } from '../types/topology'

/**
 * Default empty metrics
 */
function createEmptyMetrics(): TrafficMetrics {
  return {
    flowCount: 0,
    activeFlows: 0,
    packetCount: 0,
    byteCount: 0,
    connectionCount: 0,
  }
}

/**
 * Default empty security status
 */
function createEmptySecurity(): SecurityStatus {
  return {
    allowedFlows: 0,
    deniedFlows: 0,
    loggedFlows: 0,
  }
}

/**
 * Merge metrics from a flow into existing metrics
 */
function mergeFlowMetrics(metrics: TrafficMetrics, flow: Flow): void {
  metrics.flowCount++
  metrics.packetCount += flow.packetCount
  metrics.byteCount += flow.byteCount
  if (flow.state === 'ACTIVE') {
    metrics.activeFlows++
  }
}

/**
 * Update security status from a flow
 */
function updateSecurity(security: SecurityStatus, flow: Flow): void {
  switch (flow.policyAction) {
    case 'ALLOW':
      security.allowedFlows++
      break
    case 'DENY':
      security.deniedFlows++
      break
    case 'LOG':
      security.loggedFlows++
      break
  }
  if (flow.processInfo?.isSuspicious) {
    security.hasSuspiciousProcess = true
  }
}

/**
 * Determine if an IP is external (not in private ranges)
 */
function isExternalIP(ip: string): boolean {
  if (!ip) return false

  // Check for private IP ranges
  if (ip.startsWith('10.') ||
      ip.startsWith('192.168.') ||
      ip.match(/^172\.(1[6-9]|2[0-9]|3[0-1])\./) ||
      ip.startsWith('127.') ||
      ip === '0.0.0.0') {
    return false
  }
  return true
}

/**
 * Extract endpoint info from flow (source or destination side)
 */
interface EndpointInfo {
  nodeType: TopologyNodeType
  nodeId: string
  label: string
  k8s?: K8sMetadata
  processInfo?: ProcessInfo
  ip: string
}

function extractEndpointInfo(
  ip: string,
  labels: Record<string, string> | undefined,
  processInfo: ProcessInfo | undefined,
  viewMode: TopologyViewMode
): EndpointInfo {
  const k8sInfo = extractK8sInfo(labels)
  const containerId = processInfo?.containerId

  // Determine node type based on view mode and available data
  let nodeType: TopologyNodeType
  let nodeId: string
  let label: string
  let k8s: K8sMetadata | undefined

  switch (viewMode) {
    case 'NAMESPACE':
      if (k8sInfo?.namespace) {
        nodeType = 'NAMESPACE'
        nodeId = generateNodeId('NAMESPACE', { namespace: k8sInfo.namespace })
        label = k8sInfo.namespace
        k8s = { namespace: k8sInfo.namespace, labels }
      } else if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = `External (${ip})`
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break

    case 'SERVICE':
      if (k8sInfo?.serviceName) {
        nodeType = 'SERVICE'
        nodeId = generateNodeId('SERVICE', {
          namespace: k8sInfo.namespace,
          service: k8sInfo.serviceName
        })
        label = k8sInfo.serviceName
        k8s = {
          namespace: k8sInfo.namespace,
          serviceName: k8sInfo.serviceName,
          workloadName: k8sInfo.workloadName,
          labels,
        }
      } else if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = `External (${ip})`
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break

    case 'POD':
      if (k8sInfo?.podName) {
        nodeType = 'POD'
        nodeId = generateNodeId('POD', {
          namespace: k8sInfo.namespace,
          pod: k8sInfo.podName
        })
        label = k8sInfo.podName
        k8s = {
          namespace: k8sInfo.namespace,
          podName: k8sInfo.podName,
          serviceName: k8sInfo.serviceName,
          labels,
        }
      } else if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = `External (${ip})`
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break

    case 'CONTAINER':
      if (containerId) {
        const { runtime, shortId } = parseContainerId(containerId)
        nodeType = 'CONTAINER'
        nodeId = generateNodeId('CONTAINER', { containerId: shortId })
        label = k8sInfo?.containerName || shortId
        k8s = {
          namespace: k8sInfo?.namespace,
          podName: k8sInfo?.podName,
          containerName: k8sInfo?.containerName,
          containerId: shortId,
          containerRuntime: runtime,
          labels,
        }
      } else if (k8sInfo?.podName) {
        // Fallback to pod if no container ID
        nodeType = 'POD'
        nodeId = generateNodeId('POD', {
          namespace: k8sInfo.namespace,
          pod: k8sInfo.podName
        })
        label = k8sInfo.podName
        k8s = { namespace: k8sInfo.namespace, podName: k8sInfo.podName, labels }
      } else if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = `External (${ip})`
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break

    case 'PROCESS':
      if (processInfo && processInfo.pid > 0) {
        nodeType = 'PROCESS'
        nodeId = generateNodeId('PROCESS', {
          containerId: containerId || 'host',
          pid: String(processInfo.pid)
        })
        label = processInfo.comm || `PID ${processInfo.pid}`
        k8s = containerId ? {
          containerId: parseContainerId(containerId).shortId,
          namespace: k8sInfo?.namespace,
          podName: k8sInfo?.podName,
          labels,
        } : undefined
      } else if (containerId) {
        const { shortId } = parseContainerId(containerId)
        nodeType = 'CONTAINER'
        nodeId = generateNodeId('CONTAINER', { containerId: shortId })
        label = shortId
        k8s = { containerId: shortId, labels }
      } else if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = `External (${ip})`
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break

    case 'IP':
    default:
      if (isExternalIP(ip)) {
        nodeType = 'EXTERNAL'
        nodeId = generateNodeId('EXTERNAL', { ip })
        label = ip
      } else {
        nodeType = 'IP'
        nodeId = generateNodeId('IP', { ip })
        label = ip
      }
      break
  }

  return {
    nodeType,
    nodeId,
    label,
    k8s,
    processInfo,
    ip,
  }
}

/**
 * Main aggregation function - aggregates flows to topology based on view mode
 */
export function aggregateFlowsToTopology(
  flows: Flow[],
  viewMode: TopologyViewMode,
  maxNodes?: number,
  filters?: Partial<TopologyFilters>
): TopologyData {
  const nodeMap = new Map<string, TopologyNode>()
  const edgeMap = new Map<string, TopologyEdge>()
  const connectionSet = new Set<string>() // Track unique connections

  // Process each flow
  for (const flow of flows) {
    // Apply filters
    if (filters?.namespace) {
      const srcK8s = extractK8sInfo(flow.sourceLabels)
      const dstK8s = extractK8sInfo(flow.destLabels)
      if (srcK8s?.namespace !== filters.namespace && dstK8s?.namespace !== filters.namespace) {
        continue
      }
    }
    if (filters?.minFlowCount && flow.packetCount < filters.minFlowCount) {
      continue
    }
    if (filters?.onlySuspicious && !flow.processInfo?.isSuspicious) {
      continue
    }

    // Extract source and destination endpoint info
    const srcInfo = extractEndpointInfo(
      flow.sourceIp,
      flow.sourceLabels,
      flow.processInfo,
      viewMode
    )
    const dstInfo = extractEndpointInfo(
      flow.destIp,
      flow.destLabels,
      undefined, // Destination doesn't have process info
      viewMode
    )

    // Skip external-only connections if filter set
    if (filters?.showExternal === false) {
      if (srcInfo.nodeType === 'EXTERNAL' && dstInfo.nodeType === 'EXTERNAL') {
        continue
      }
    }

    // Create or update source node
    if (!nodeMap.has(srcInfo.nodeId)) {
      nodeMap.set(srcInfo.nodeId, {
        id: srcInfo.nodeId,
        label: srcInfo.label,
        type: srcInfo.nodeType,
        metrics: createEmptyMetrics(),
        security: createEmptySecurity(),
        k8s: srcInfo.k8s,
        processInfo: srcInfo.processInfo,
        ipAddress: srcInfo.ip,
      })
    }
    const srcNode = nodeMap.get(srcInfo.nodeId)!
    mergeFlowMetrics(srcNode.metrics, flow)
    updateSecurity(srcNode.security!, flow)

    // Create or update destination node
    if (!nodeMap.has(dstInfo.nodeId)) {
      nodeMap.set(dstInfo.nodeId, {
        id: dstInfo.nodeId,
        label: dstInfo.label,
        type: dstInfo.nodeType,
        metrics: createEmptyMetrics(),
        security: createEmptySecurity(),
        k8s: dstInfo.k8s,
        ipAddress: dstInfo.ip,
      })
    }
    const dstNode = nodeMap.get(dstInfo.nodeId)!
    mergeFlowMetrics(dstNode.metrics, flow)
    updateSecurity(dstNode.security!, flow)

    // Track connections for both nodes
    const connKey = `${srcInfo.nodeId}->${dstInfo.nodeId}`
    if (!connectionSet.has(connKey)) {
      connectionSet.add(connKey)
      srcNode.metrics.connectionCount = (srcNode.metrics.connectionCount || 0) + 1
      dstNode.metrics.connectionCount = (dstNode.metrics.connectionCount || 0) + 1
    }

    // Create or update edge
    const edgeId = `${srcInfo.nodeId}->${dstInfo.nodeId}`
    const reverseEdgeId = `${dstInfo.nodeId}->${srcInfo.nodeId}`

    let edge = edgeMap.get(edgeId) || edgeMap.get(reverseEdgeId)
    const isReverse = edgeMap.has(reverseEdgeId)

    if (!edge) {
      edge = {
        id: edgeId,
        source: srcInfo.nodeId,
        target: dstInfo.nodeId,
        metrics: createEmptyMetrics(),
        security: createEmptySecurity(),
        protocols: [],
        direction: flow.direction === 'UNKNOWN' ? 'EGRESS' : flow.direction,
        isBidirectional: false,
      }
      edgeMap.set(edgeId, edge)
    }

    // Update edge metrics
    mergeFlowMetrics(edge.metrics, flow)
    updateSecurity(edge.security!, flow)

    // Track bidirectional
    if (isReverse) {
      edge.isBidirectional = true
      edge.direction = 'BIDIRECTIONAL'
    }

    // Update protocol info
    const protocolKey = `${flow.protocol}:${flow.destPort}`
    let protocolInfo = edge.protocols.find(p => p.name === flow.protocol && p.port === flow.destPort)
    if (!protocolInfo) {
      protocolInfo = {
        name: flow.protocol,
        port: flow.destPort,
        flowCount: 0,
        byteCount: 0,
      }
      edge.protocols.push(protocolInfo)
    }
    protocolInfo.flowCount++
    protocolInfo.byteCount += flow.byteCount

    // Set edge color hint based on security
    if (flow.policyAction === 'DENY') {
      edge.colorHint = 'denied'
    } else if (edge.colorHint !== 'denied') {
      edge.colorHint = flow.policyAction === 'ALLOW' ? 'allowed' : 'normal'
    }
  }

  // Convert to arrays
  let nodes = Array.from(nodeMap.values())
  let edges = Array.from(edgeMap.values())

  // Calculate node health
  nodes.forEach(node => {
    if (node.security) {
      if (node.security.deniedFlows > 0 || node.security.hasSuspiciousProcess) {
        node.health = 'critical'
      } else if (node.security.loggedFlows > 0) {
        node.health = 'warning'
      } else {
        node.health = 'healthy'
      }
    }
  })

  // Limit nodes if needed (Top N by traffic)
  if (maxNodes && nodes.length > maxNodes) {
    nodes = nodes
      .sort((a, b) => b.metrics.byteCount - a.metrics.byteCount)
      .slice(0, maxNodes)

    const nodeIds = new Set(nodes.map(n => n.id))
    edges = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target))
  }

  // Build groups for hierarchical views
  const groups = buildGroups(nodes, viewMode)

  // Calculate stats
  const stats = calculateStats(nodes, edges, flows)

  return {
    nodes,
    edges,
    groups,
    stats,
    viewMode,
    timestamp: new Date().toISOString(),
  }
}

/**
 * Build node groups for hierarchical display
 */
function buildGroups(nodes: TopologyNode[], viewMode: TopologyViewMode): TopologyGroup[] {
  const groups: TopologyGroup[] = []

  if (viewMode === 'SERVICE' || viewMode === 'POD' || viewMode === 'CONTAINER') {
    // Group by namespace
    const namespaceMap = new Map<string, TopologyNode[]>()

    nodes.forEach(node => {
      const ns = node.k8s?.namespace || 'default'
      if (!namespaceMap.has(ns)) {
        namespaceMap.set(ns, [])
      }
      namespaceMap.get(ns)!.push(node)
    })

    namespaceMap.forEach((nsNodes, namespace) => {
      if (nsNodes.length > 0) {
        const metrics = createEmptyMetrics()
        nsNodes.forEach(n => {
          metrics.flowCount += n.metrics.flowCount
          metrics.packetCount += n.metrics.packetCount
          metrics.byteCount += n.metrics.byteCount
          metrics.activeFlows += n.metrics.activeFlows
        })

        groups.push({
          id: `ns:${namespace}`,
          label: namespace,
          type: 'NAMESPACE',
          nodeIds: nsNodes.map(n => n.id),
          k8s: { namespace },
          metrics,
          expanded: true,
        })
      }
    })
  }

  if (viewMode === 'CONTAINER' || viewMode === 'PROCESS') {
    // Group by pod within namespace groups
    const podMap = new Map<string, TopologyNode[]>()

    nodes.forEach(node => {
      if (node.k8s?.podName) {
        const podKey = `${node.k8s.namespace || 'default'}/${node.k8s.podName}`
        if (!podMap.has(podKey)) {
          podMap.set(podKey, [])
        }
        podMap.get(podKey)!.push(node)
      }
    })

    podMap.forEach((podNodes, podKey) => {
      const [namespace, podName] = podKey.split('/')
      const metrics = createEmptyMetrics()
      podNodes.forEach(n => {
        metrics.flowCount += n.metrics.flowCount
        metrics.packetCount += n.metrics.packetCount
        metrics.byteCount += n.metrics.byteCount
        metrics.activeFlows += n.metrics.activeFlows
      })

      groups.push({
        id: `pod:${podKey}`,
        label: podName,
        type: 'POD',
        nodeIds: podNodes.map(n => n.id),
        k8s: { namespace, podName },
        metrics,
        expanded: true,
        parentId: `ns:${namespace}`,
      })
    })
  }

  return groups
}

/**
 * Calculate topology statistics
 */
function calculateStats(
  nodes: TopologyNode[],
  edges: TopologyEdge[],
  flows: Flow[]
): TopologyStats {
  const stats: TopologyStats = {
    totalNodes: nodes.length,
    totalEdges: edges.length,
    totalFlows: flows.length,
    activeFlows: flows.filter(f => f.state === 'ACTIVE').length,
    totalBytes: flows.reduce((sum, f) => sum + f.byteCount, 0),
    namespaceCount: 0,
    serviceCount: 0,
    podCount: 0,
    containerCount: 0,
    processCount: 0,
    externalCount: 0,
  }

  // Count by type
  const namespaces = new Set<string>()
  const services = new Set<string>()
  const pods = new Set<string>()

  nodes.forEach(node => {
    switch (node.type) {
      case 'NAMESPACE':
        stats.namespaceCount!++
        break
      case 'SERVICE':
        stats.serviceCount!++
        if (node.k8s?.namespace) namespaces.add(node.k8s.namespace)
        if (node.k8s?.serviceName) services.add(node.k8s.serviceName)
        break
      case 'POD':
        stats.podCount!++
        if (node.k8s?.namespace) namespaces.add(node.k8s.namespace)
        if (node.k8s?.podName) pods.add(node.k8s.podName)
        break
      case 'CONTAINER':
        stats.containerCount!++
        if (node.k8s?.namespace) namespaces.add(node.k8s.namespace)
        if (node.k8s?.podName) pods.add(node.k8s.podName)
        break
      case 'PROCESS':
        stats.processCount!++
        break
      case 'EXTERNAL':
        stats.externalCount!++
        break
    }
  })

  // Update counts from sets for unique values
  if (stats.namespaceCount === 0) stats.namespaceCount = namespaces.size
  if (stats.serviceCount === 0) stats.serviceCount = services.size
  if (stats.podCount === 0) stats.podCount = pods.size

  return stats
}

/**
 * Calculate node size for visualization (logarithmic scale)
 */
export function calculateNodeSize(metrics: TrafficMetrics, type: TopologyNodeType): number {
  const minSize = 24
  const maxSize = 80

  // Different base sizes for different types
  const baseSizes: Record<TopologyNodeType, number> = {
    NAMESPACE: 60,
    SERVICE: 50,
    POD: 40,
    CONTAINER: 35,
    PROCESS: 30,
    IP: 30,
    EXTERNAL: 35,
  }

  const baseSize = baseSizes[type] || 30

  if (metrics.flowCount <= 1) return Math.max(minSize, baseSize - 10)

  const size = baseSize + Math.log10(metrics.flowCount) * 8
  return Math.min(Math.max(size, minSize), maxSize)
}

/**
 * Calculate edge width for visualization (logarithmic scale)
 */
export function calculateEdgeWidth(metrics: TrafficMetrics): number {
  const minWidth = 1
  const maxWidth = 12
  const baseWidth = 2

  if (metrics.byteCount <= 1024) return minWidth

  const width = baseWidth + Math.log10(metrics.byteCount / 1024) * 1.5
  return Math.min(Math.max(width, minWidth), maxWidth)
}

/**
 * Get color for node type
 */
export function getNodeTypeColor(type: TopologyNodeType): string {
  const colors: Record<TopologyNodeType, string> = {
    NAMESPACE: '#722ed1',  // Purple
    SERVICE: '#1890ff',    // Blue
    POD: '#52c41a',        // Green
    CONTAINER: '#13c2c2',  // Cyan
    PROCESS: '#fa8c16',    // Orange
    IP: '#8c8c8c',         // Gray
    EXTERNAL: '#f5222d',   // Red
  }
  return colors[type] || '#8c8c8c'
}

/**
 * Get icon for node type (for ECharts symbol)
 */
export function getNodeTypeSymbol(type: TopologyNodeType): string {
  const symbols: Record<TopologyNodeType, string> = {
    NAMESPACE: 'roundRect',
    SERVICE: 'circle',
    POD: 'diamond',
    CONTAINER: 'rect',
    PROCESS: 'triangle',
    IP: 'circle',
    EXTERNAL: 'pin',
  }
  return symbols[type] || 'circle'
}

/**
 * Merge real-time flow update into existing topology
 */
export function mergeTopologyUpdate(
  existing: TopologyData,
  newFlow: Flow,
  viewMode: TopologyViewMode
): TopologyData {
  // Clone existing data
  const nodes = [...existing.nodes]
  const edges = [...existing.edges]
  const groups = existing.groups ? [...existing.groups] : []

  // Extract endpoint info
  const srcInfo = extractEndpointInfo(
    newFlow.sourceIp,
    newFlow.sourceLabels,
    newFlow.processInfo,
    viewMode
  )
  const dstInfo = extractEndpointInfo(
    newFlow.destIp,
    newFlow.destLabels,
    undefined,
    viewMode
  )

  // Update or create source node
  let srcNodeIndex = nodes.findIndex(n => n.id === srcInfo.nodeId)
  if (srcNodeIndex === -1) {
    nodes.push({
      id: srcInfo.nodeId,
      label: srcInfo.label,
      type: srcInfo.nodeType,
      metrics: createEmptyMetrics(),
      security: createEmptySecurity(),
      k8s: srcInfo.k8s,
      processInfo: srcInfo.processInfo,
      ipAddress: srcInfo.ip,
    })
    srcNodeIndex = nodes.length - 1
  }
  mergeFlowMetrics(nodes[srcNodeIndex].metrics, newFlow)
  updateSecurity(nodes[srcNodeIndex].security!, newFlow)

  // Update or create destination node
  let dstNodeIndex = nodes.findIndex(n => n.id === dstInfo.nodeId)
  if (dstNodeIndex === -1) {
    nodes.push({
      id: dstInfo.nodeId,
      label: dstInfo.label,
      type: dstInfo.nodeType,
      metrics: createEmptyMetrics(),
      security: createEmptySecurity(),
      k8s: dstInfo.k8s,
      ipAddress: dstInfo.ip,
    })
    dstNodeIndex = nodes.length - 1
  }
  mergeFlowMetrics(nodes[dstNodeIndex].metrics, newFlow)
  updateSecurity(nodes[dstNodeIndex].security!, newFlow)

  // Update or create edge
  const edgeId = `${srcInfo.nodeId}->${dstInfo.nodeId}`
  let edgeIndex = edges.findIndex(e =>
    e.id === edgeId || e.id === `${dstInfo.nodeId}->${srcInfo.nodeId}`
  )

  if (edgeIndex === -1) {
    edges.push({
      id: edgeId,
      source: srcInfo.nodeId,
      target: dstInfo.nodeId,
      metrics: createEmptyMetrics(),
      security: createEmptySecurity(),
      protocols: [],
      direction: newFlow.direction === 'UNKNOWN' ? 'EGRESS' : newFlow.direction,
    })
    edgeIndex = edges.length - 1
  }

  mergeFlowMetrics(edges[edgeIndex].metrics, newFlow)
  updateSecurity(edges[edgeIndex].security!, newFlow)

  // Update protocol info
  const edge = edges[edgeIndex]
  let protocolInfo = edge.protocols.find(
    p => p.name === newFlow.protocol && p.port === newFlow.destPort
  )
  if (!protocolInfo) {
    protocolInfo = {
      name: newFlow.protocol,
      port: newFlow.destPort,
      flowCount: 0,
      byteCount: 0,
    }
    edge.protocols.push(protocolInfo)
  }
  protocolInfo.flowCount++
  protocolInfo.byteCount += newFlow.byteCount

  return {
    nodes,
    edges,
    groups,
    stats: {
      ...existing.stats,
      totalNodes: nodes.length,
      totalEdges: edges.length,
      totalFlows: existing.stats.totalFlows + 1,
    },
    viewMode,
    timestamp: new Date().toISOString(),
  }
}

/**
 * Get display label for a node
 */
export function getNodeLabel(node: TopologyNode): string {
  switch (node.type) {
    case 'NAMESPACE':
      return node.k8s?.namespace || node.label
    case 'SERVICE':
      return node.k8s?.serviceName || node.label
    case 'POD':
      return node.k8s?.podName || node.label
    case 'CONTAINER':
      return node.k8s?.containerName || node.k8s?.containerId || node.label
    case 'PROCESS':
      return node.processInfo?.comm || node.label
    case 'EXTERNAL':
      return `External: ${node.ipAddress || node.label}`
    default:
      return node.label
  }
}

/**
 * Format bytes to human readable
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

/**
 * Format number with K/M/B suffix
 */
export function formatNumber(num: number): string {
  if (num >= 1e9) return `${(num / 1e9).toFixed(1)}B`
  if (num >= 1e6) return `${(num / 1e6).toFixed(1)}M`
  if (num >= 1e3) return `${(num / 1e3).toFixed(1)}K`
  return String(num)
}

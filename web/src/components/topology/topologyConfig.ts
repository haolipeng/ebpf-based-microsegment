import type { EChartsOption } from 'echarts'
import type {
  TopologyData,
  TopologyNode,
  TopologyEdge,
  TopologyViewMode,
  TopologyNodeType,
  TopologyGroup,
} from '../../types/topology'
import {
  calculateNodeSize,
  calculateEdgeWidth,
  getNodeTypeColor,
  getNodeTypeSymbol,
  formatBytes,
  formatNumber,
} from '../../utils/topologyUtils'

/**
 * View mode display names
 */
const VIEW_MODE_NAMES: Record<TopologyViewMode, string> = {
  NAMESPACE: 'Namespace View',
  SERVICE: 'Service View',
  POD: 'Pod View',
  CONTAINER: 'Container View',
  PROCESS: 'Process View',
  IP: 'IP View',
}

/**
 * Node type icons for tooltips
 */
const NODE_TYPE_ICONS: Record<TopologyNodeType, string> = {
  NAMESPACE: '📦',
  SERVICE: '⚙️',
  POD: '🎯',
  CONTAINER: '📦',
  PROCESS: '⚡',
  IP: '🖥️',
  EXTERNAL: '🌐',
}

/**
 * Calculate optimal force layout parameters based on node count and view mode
 */
function calculateForceParams(nodeCount: number, viewMode: TopologyViewMode) {
  // Base parameters
  let repulsion = 800
  let gravity = 0.08
  let edgeLength: [number, number] = [120, 250]
  let friction = 0.6

  // Adjust based on node count
  if (nodeCount <= 10) {
    repulsion = 800
    gravity = 0.08
    edgeLength = [120, 250]
  } else if (nodeCount <= 30) {
    repulsion = 1000
    gravity = 0.05
    edgeLength = [150, 300]
  } else if (nodeCount <= 50) {
    repulsion = 1200
    gravity = 0.04
    edgeLength = [180, 350]
  } else if (nodeCount <= 100) {
    repulsion = 1500
    gravity = 0.03
    edgeLength = [200, 400]
  } else {
    repulsion = 2000
    gravity = 0.02
    edgeLength = [250, 500]
    friction = 0.4
  }

  // Adjust based on view mode (more detailed views need more spacing)
  if (viewMode === 'PROCESS' || viewMode === 'CONTAINER') {
    repulsion *= 1.2
    edgeLength = [edgeLength[0] * 1.2, edgeLength[1] * 1.2]
  } else if (viewMode === 'NAMESPACE') {
    repulsion *= 1.5
    edgeLength = [edgeLength[0] * 1.5, edgeLength[1] * 1.5]
  }

  return { repulsion, gravity, edgeLength, friction }
}

/**
 * Get node categories for legend
 */
function getCategories(viewMode: TopologyViewMode): { name: string; itemStyle: { color: string } }[] {
  const categories: { name: string; itemStyle: { color: string } }[] = []

  // Add categories based on view mode
  switch (viewMode) {
    case 'NAMESPACE':
      categories.push({ name: 'Namespace', itemStyle: { color: getNodeTypeColor('NAMESPACE') } })
      break
    case 'SERVICE':
      categories.push({ name: 'Service', itemStyle: { color: getNodeTypeColor('SERVICE') } })
      break
    case 'POD':
      categories.push({ name: 'Pod', itemStyle: { color: getNodeTypeColor('POD') } })
      break
    case 'CONTAINER':
      categories.push({ name: 'Container', itemStyle: { color: getNodeTypeColor('CONTAINER') } })
      categories.push({ name: 'Pod', itemStyle: { color: getNodeTypeColor('POD') } })
      break
    case 'PROCESS':
      categories.push({ name: 'Process', itemStyle: { color: getNodeTypeColor('PROCESS') } })
      categories.push({ name: 'Container', itemStyle: { color: getNodeTypeColor('CONTAINER') } })
      break
    case 'IP':
      categories.push({ name: 'IP', itemStyle: { color: getNodeTypeColor('IP') } })
      break
  }

  // Always add external
  categories.push({ name: 'External', itemStyle: { color: getNodeTypeColor('EXTERNAL') } })

  return categories
}

/**
 * Get category index for a node
 */
function getCategoryIndex(node: TopologyNode, categories: { name: string }[]): number {
  const typeName = node.type === 'IP' ? 'IP' :
    node.type === 'EXTERNAL' ? 'External' :
    node.type.charAt(0) + node.type.slice(1).toLowerCase()

  const index = categories.findIndex(c => c.name === typeName)
  return index >= 0 ? index : categories.length - 1
}

/**
 * Generate ECharts Graph configuration with K8s/Docker support
 */
export function getTopologyChartOption(
  data: TopologyData,
  viewMode: TopologyViewMode
): EChartsOption {
  const forceParams = calculateForceParams(data.nodes.length, viewMode)
  const categories = getCategories(viewMode)

  // Build subtitle with stats
  const subtextParts = [`${data.stats.totalNodes} nodes`, `${data.stats.totalEdges} edges`]
  if (data.stats.namespaceCount) subtextParts.push(`${data.stats.namespaceCount} namespaces`)
  if (data.stats.serviceCount) subtextParts.push(`${data.stats.serviceCount} services`)
  if (data.stats.podCount) subtextParts.push(`${data.stats.podCount} pods`)
  if (data.stats.containerCount) subtextParts.push(`${data.stats.containerCount} containers`)
  if (data.stats.externalCount) subtextParts.push(`${data.stats.externalCount} external`)

  return {
    title: {
      text: `Network Topology (${VIEW_MODE_NAMES[viewMode]})`,
      subtext: subtextParts.join(' | '),
      left: 'center',
      top: 10,
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold',
      },
      subtextStyle: {
        fontSize: 12,
        color: '#666',
      },
    },
    tooltip: {
      trigger: 'item',
      confine: true,
      formatter: (params: unknown) => {
        const p = params as { dataType: string; data: TopologyNode | TopologyEdge }
        if (p.dataType === 'node') {
          return formatNodeTooltip(p.data as TopologyNode)
        } else if (p.dataType === 'edge') {
          return formatEdgeTooltip(p.data as TopologyEdge)
        }
        return ''
      },
    },
    legend: {
      show: true,
      type: 'scroll',
      orient: 'horizontal',
      bottom: 10,
      data: categories.map(c => c.name),
    },
    grid: {
      left: '5%',
      right: '5%',
      top: '80px',
      bottom: '60px',
      containLabel: true,
    },
    series: [
      {
        type: 'graph',
        layout: 'force',
        categories,
        data: data.nodes.map(node => ({
          id: node.id,
          name: node.label,
          value: node.metrics.byteCount,
          symbol: getNodeTypeSymbol(node.type),
          symbolSize: calculateNodeSize(node.metrics, node.type),
          category: getCategoryIndex(node, categories),
          itemStyle: getNodeStyle(node),
          label: {
            show: true,
            position: 'bottom',
            formatter: '{b}',
            fontSize: 10,
            color: '#333',
            textShadowColor: 'rgba(255, 255, 255, 0.9)',
            textShadowBlur: 3,
          },
          // Store full node data for click events
          ...node,
        })),
        edges: data.edges.map(edge => ({
          source: edge.source,
          target: edge.target,
          value: edge.metrics.byteCount,
          lineStyle: getEdgeStyle(edge),
          label: {
            show: false,
          },
          // Store full edge data for click events
          ...edge,
        })),
        roam: true,
        draggable: true,
        force: {
          repulsion: forceParams.repulsion,
          gravity: forceParams.gravity,
          edgeLength: forceParams.edgeLength,
          layoutAnimation: true,
          friction: forceParams.friction,
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: {
            width: 5,
          },
          itemStyle: {
            shadowBlur: 15,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
        blur: {
          itemStyle: {
            opacity: 0.3,
          },
          lineStyle: {
            opacity: 0.1,
          },
        },
      },
    ],
  }
}

/**
 * Get node style based on type and health
 */
export function getNodeStyle(node: TopologyNode) {
  const baseColor = getNodeTypeColor(node.type)

  // Determine color based on health status
  let color = baseColor
  let borderColor = baseColor
  let borderWidth = 2

  if (node.health === 'critical') {
    borderColor = '#f5222d'
    borderWidth = 3
  } else if (node.health === 'warning') {
    borderColor = '#faad14'
    borderWidth = 3
  }

  // Adjust opacity based on active flows
  const opacity = node.metrics.activeFlows > 0 ? 0.95 : 0.7

  return {
    color,
    opacity,
    borderColor,
    borderWidth,
    shadowBlur: 8,
    shadowColor: `${baseColor}40`,
  }
}

/**
 * Get edge style based on security status and traffic
 */
export function getEdgeStyle(edge: TopologyEdge) {
  // Determine color based on security
  let color = '#91caff' // Default soft blue

  if (edge.colorHint === 'denied') {
    color = '#ff7875' // Red for denied
  } else if (edge.colorHint === 'warning') {
    color = '#ffc069' // Orange for warning
  } else if (edge.isBidirectional) {
    color = '#95de64' // Green for bidirectional
  }

  const width = Math.max(1, calculateEdgeWidth(edge.metrics))

  return {
    color,
    width,
    curveness: edge.isBidirectional ? 0.15 : 0.05,
    opacity: 0.6,
    type: edge.style || ('solid' as const),
  }
}

/**
 * Format node tooltip with K8s metadata
 */
export function formatNodeTooltip(node: TopologyNode): string {
  const icon = NODE_TYPE_ICONS[node.type] || '📍'
  const lines: string[] = []

  lines.push(`<div style="font-weight: bold; margin-bottom: 8px; font-size: 14px;">`)
  lines.push(`${icon} ${node.label}`)
  lines.push(`</div>`)

  lines.push(`<div style="font-size: 12px;">`)

  // Type info
  lines.push(`<div><b>Type:</b> ${node.type}</div>`)

  // K8s metadata
  if (node.k8s) {
    if (node.k8s.namespace) {
      lines.push(`<div><b>Namespace:</b> ${node.k8s.namespace}</div>`)
    }
    if (node.k8s.serviceName) {
      lines.push(`<div><b>Service:</b> ${node.k8s.serviceName}</div>`)
    }
    if (node.k8s.podName) {
      lines.push(`<div><b>Pod:</b> ${node.k8s.podName}</div>`)
    }
    if (node.k8s.containerName) {
      lines.push(`<div><b>Container:</b> ${node.k8s.containerName}</div>`)
    }
    if (node.k8s.containerId) {
      lines.push(`<div><b>Container ID:</b> ${node.k8s.containerId}</div>`)
    }
    if (node.k8s.workloadName) {
      lines.push(`<div><b>Workload:</b> ${node.k8s.workloadName} (${node.k8s.workloadType || 'unknown'})</div>`)
    }
  }

  // Process info
  if (node.processInfo) {
    lines.push(`<div style="margin-top: 4px; border-top: 1px solid #eee; padding-top: 4px;">`)
    lines.push(`<div><b>Process:</b> ${node.processInfo.comm} (PID: ${node.processInfo.pid})</div>`)
    if (node.processInfo.exePath) {
      lines.push(`<div><b>Path:</b> ${node.processInfo.exePath}</div>`)
    }
    if (node.processInfo.isSuspicious) {
      lines.push(`<div style="color: #f5222d;"><b>⚠️ Suspicious Process</b></div>`)
    }
    lines.push(`</div>`)
  }

  // IP address
  if (node.ipAddress) {
    lines.push(`<div><b>IP:</b> ${node.ipAddress}</div>`)
  }

  // Metrics
  lines.push(`<div style="margin-top: 4px; border-top: 1px solid #eee; padding-top: 4px;">`)
  lines.push(`<div><b>Flows:</b> ${formatNumber(node.metrics.flowCount)} (${formatNumber(node.metrics.activeFlows)} active)</div>`)
  lines.push(`<div><b>Packets:</b> ${formatNumber(node.metrics.packetCount)}</div>`)
  lines.push(`<div><b>Traffic:</b> ${formatBytes(node.metrics.byteCount)}</div>`)
  if (node.metrics.connectionCount) {
    lines.push(`<div><b>Connections:</b> ${formatNumber(node.metrics.connectionCount)}</div>`)
  }
  lines.push(`</div>`)

  // Security status
  if (node.security) {
    const { allowedFlows, deniedFlows, loggedFlows } = node.security
    if (deniedFlows > 0 || loggedFlows > 0) {
      lines.push(`<div style="margin-top: 4px; border-top: 1px solid #eee; padding-top: 4px;">`)
      lines.push(`<div><b>Security:</b></div>`)
      lines.push(`<div style="color: #52c41a;">✓ Allowed: ${formatNumber(allowedFlows)}</div>`)
      if (deniedFlows > 0) {
        lines.push(`<div style="color: #f5222d;">✗ Denied: ${formatNumber(deniedFlows)}</div>`)
      }
      if (loggedFlows > 0) {
        lines.push(`<div style="color: #faad14;">⚠ Logged: ${formatNumber(loggedFlows)}</div>`)
      }
      lines.push(`</div>`)
    }
  }

  // Health indicator
  if (node.health && node.health !== 'healthy') {
    const healthColors = { critical: '#f5222d', warning: '#faad14', unknown: '#8c8c8c' }
    const healthLabels = { critical: '⚠️ Critical', warning: '⚠️ Warning', unknown: '❓ Unknown' }
    lines.push(`<div style="margin-top: 4px; color: ${healthColors[node.health]}; font-weight: bold;">`)
    lines.push(healthLabels[node.health])
    lines.push(`</div>`)
  }

  lines.push(`</div>`)

  return lines.join('')
}

/**
 * Format edge tooltip with protocol details
 */
export function formatEdgeTooltip(edge: TopologyEdge): string {
  const lines: string[] = []

  // Source -> Target
  const directionIcon = edge.isBidirectional ? '↔️' : '→'
  lines.push(`<div style="font-weight: bold; margin-bottom: 8px;">`)
  lines.push(`${edge.source} ${directionIcon} ${edge.target}`)
  lines.push(`</div>`)

  lines.push(`<div style="font-size: 12px;">`)

  // Direction
  const directionLabels = {
    INGRESS: '⬇️ Inbound',
    EGRESS: '⬆️ Outbound',
    BIDIRECTIONAL: '↕️ Bidirectional',
  }
  lines.push(`<div><b>Direction:</b> ${directionLabels[edge.direction]}</div>`)

  // Metrics
  lines.push(`<div><b>Flows:</b> ${formatNumber(edge.metrics.flowCount)} (${formatNumber(edge.metrics.activeFlows)} active)</div>`)
  lines.push(`<div><b>Packets:</b> ${formatNumber(edge.metrics.packetCount)}</div>`)
  lines.push(`<div><b>Traffic:</b> ${formatBytes(edge.metrics.byteCount)}</div>`)

  // Protocols
  if (edge.protocols && edge.protocols.length > 0) {
    lines.push(`<div style="margin-top: 4px; border-top: 1px solid #eee; padding-top: 4px;">`)
    lines.push(`<div><b>Protocols:</b></div>`)
    edge.protocols.slice(0, 5).forEach(p => {
      lines.push(`<div>• ${p.name}:${p.port} (${formatNumber(p.flowCount)} flows, ${formatBytes(p.byteCount)})</div>`)
    })
    if (edge.protocols.length > 5) {
      lines.push(`<div style="color: #666;">... and ${edge.protocols.length - 5} more</div>`)
    }
    lines.push(`</div>`)
  }

  // Security
  if (edge.security) {
    const { allowedFlows, deniedFlows } = edge.security
    if (deniedFlows > 0) {
      lines.push(`<div style="margin-top: 4px; border-top: 1px solid #eee; padding-top: 4px;">`)
      lines.push(`<div style="color: #52c41a;">✓ Allowed: ${formatNumber(allowedFlows)}</div>`)
      lines.push(`<div style="color: #f5222d;">✗ Denied: ${formatNumber(deniedFlows)}</div>`)
      lines.push(`</div>`)
    }
  }

  lines.push(`</div>`)

  return lines.join('')
}

/**
 * Generate group configuration for hierarchical display (future use)
 */
export function getGroupConfig(groups: TopologyGroup[]): unknown[] {
  return groups.map(group => ({
    id: group.id,
    name: group.label,
    type: group.type,
    nodeIds: group.nodeIds,
    expanded: group.expanded,
    itemStyle: {
      color: getNodeTypeColor(group.type),
      opacity: 0.1,
      borderColor: getNodeTypeColor(group.type),
      borderWidth: 2,
      borderType: 'dashed',
    },
  }))
}

import type { EChartsOption } from 'echarts'
import type { TopologyData, TopologyNode, TopologyEdge, TopologyViewMode } from '../../types/topology'
import { calculateNodeSize, calculateEdgeWidth } from '../../utils/topologyUtils'
import { formatBytes, formatNumber } from '../../utils/format'

/**
 * Get node symbol based on type
 * Using simple circles with different visual styles to match the reference design
 */
function getNodeSymbol(_node: TopologyNode): string {
  // Use circle for all nodes to match the reference design
  // The visual differentiation will come from color and size
  return 'circle'
}

/**
 * Calculate optimal force layout parameters based on node count
 */
function calculateForceParams(nodeCount: number) {
  // Adjust parameters based on number of nodes
  if (nodeCount <= 10) {
    return {
      repulsion: 800,
      gravity: 0.08,
      edgeLength: [120, 250],
    }
  } else if (nodeCount <= 30) {
    return {
      repulsion: 1000,
      gravity: 0.05,
      edgeLength: [150, 300],
    }
  } else if (nodeCount <= 50) {
    return {
      repulsion: 1200,
      gravity: 0.04,
      edgeLength: [180, 350],
    }
  } else {
    return {
      repulsion: 1500,
      gravity: 0.03,
      edgeLength: [200, 400],
    }
  }
}

/**
 * 生成ECharts Graph配置
 */
export function getTopologyChartOption(
  data: TopologyData,
  viewMode: TopologyViewMode
): EChartsOption {
  const forceParams = calculateForceParams(data.nodes.length)
  return {
    title: {
      text: `Network Topology (${viewMode === 'IP' ? 'IP View' : 'Service View'})`,
      subtext: `${data.stats.totalNodes} nodes, ${data.stats.totalEdges} connections`,
      left: 'center',
      top: 10,
    },
    tooltip: {
      trigger: 'item',
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      formatter: (params: any) => {
        if (params.dataType === 'node') {
          return formatNodeTooltip(params.data as TopologyNode)
        } else if (params.dataType === 'edge') {
          return formatEdgeTooltip(params.data as TopologyEdge)
        }
        return ''
      },
    },
    legend: {
      show: false,
    },
    // Add grid configuration to give more space for topology
    grid: {
      left: '5%',
      right: '5%',
      top: '80px',
      bottom: '5%',
      containLabel: true,
    },
    series: [
      {
        type: 'graph',
        layout: 'force',
        data: data.nodes.map(node => ({
          id: node.id,
          name: node.label,
          value: node.metrics.byteCount,
          symbol: getNodeSymbol(node),
          symbolSize: calculateNodeSize(node.metrics.flowCount),
          itemStyle: getNodeStyle(node),
          label: {
            show: true,
            position: 'bottom',
            formatter: '{b}',
            fontSize: 10,
            color: '#5f6368', // Neutral gray for text
            // Remove background for cleaner look to match reference design
            textShadowColor: 'rgba(255, 255, 255, 0.8)',
            textShadowBlur: 3,
            textShadowOffsetX: 0,
            textShadowOffsetY: 1,
          },
        })),
        edges: data.edges.map(edge => ({
          source: edge.source,
          target: edge.target,
          lineStyle: getEdgeStyle(edge),
          label: {
            show: false,
          },
        })),
        roam: true, // 允许缩放和平移
        draggable: true, // 允许拖拽节点
        force: {
          repulsion: forceParams.repulsion, // 节点之间的斥力 - 根据节点数量动态调整
          gravity: forceParams.gravity, // 节点受到的向中心的引力 - 根据节点数量动态调整
          edgeLength: forceParams.edgeLength, // 边的两个节点之间的距离 - 根据节点数量动态调整
          layoutAnimation: true,
          friction: 0.6, // 摩擦力 - 控制节点移动的阻尼
        },
        emphasis: {
          focus: 'adjacency', // 高亮相邻节点和边
          lineStyle: {
            width: 5,
          },
          itemStyle: {
            shadowBlur: 10,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  }
}

/**
 * 获取节点样式 - 使用浅蓝色系匹配目标设计
 */
export function getNodeStyle(node: TopologyNode) {
  // Use softer blue colors to match the reference design
  let color = '#A8C7FA' // Soft blue for IP nodes (similar to reference)

  if (node.type === 'SERVICE') {
    color = '#8AB4F8' // Slightly deeper blue for service nodes
  }

  // Adjust opacity based on active flows
  const opacity = node.metrics.activeFlows > 0 ? 0.9 : 0.6

  return {
    color,
    opacity,
    borderColor: '#5E97F6', // Slightly darker blue border
    borderWidth: 2,
    shadowBlur: 8,
    shadowColor: 'rgba(94, 151, 246, 0.3)', // Blue-tinted shadow
  }
}

/**
 * 获取边样式 - 使用浅蓝色细线匹配目标设计
 */
export function getEdgeStyle(edge: TopologyEdge) {
  // Use soft blue color for all edges to match the reference design
  const color = '#A8C7FA' // Soft blue, consistent with nodes

  // Use thinner lines to match the reference design
  const width = Math.max(1, calculateEdgeWidth(edge.metrics.byteCount) * 0.7)

  return {
    color,
    width,
    curveness: 0.05, // Minimal curve for cleaner look
    opacity: 0.5, // More transparent for a lighter appearance
    type: 'solid' as const,
  }
}

/**
 * 格式化节点Tooltip
 */
export function formatNodeTooltip(node: TopologyNode): string {
  return `
    <div style="padding: 8px;">
      <div style="font-weight: bold; margin-bottom: 8px;">
        ${node.type === 'IP' ? '🖥️' : '⚙️'} ${node.label}
      </div>
      <div style="font-size: 12px;">
        <div>Type: ${node.type === 'IP' ? 'IP Address' : 'Service'}</div>
        <div>Flows: ${formatNumber(node.metrics.flowCount)}</div>
        <div>Active: ${formatNumber(node.metrics.activeFlows)}</div>
        <div>Packets: ${formatNumber(node.metrics.packetCount)}</div>
        <div>Traffic: ${formatBytes(node.metrics.byteCount)}</div>
        ${node.labels ? `<div style="margin-top: 4px; color: #666;">Labels: ${JSON.stringify(node.labels)}</div>` : ''}
      </div>
    </div>
  `
}

/**
 * 格式化边Tooltip
 */
export function formatEdgeTooltip(edge: TopologyEdge): string {
  return `
    <div style="padding: 8px;">
      <div style="font-weight: bold; margin-bottom: 8px;">
        ${edge.source} → ${edge.target}
      </div>
      <div style="font-size: 12px;">
        <div>Direction: ${
          edge.direction === 'INGRESS' ? 'Inbound'
          : edge.direction === 'EGRESS' ? 'Outbound'
          : 'Bidirectional'
        }</div>
        <div>Flows: ${formatNumber(edge.metrics.flowCount)}</div>
        <div>Packets: ${formatNumber(edge.metrics.packetCount)}</div>
        <div>Traffic: ${formatBytes(edge.metrics.byteCount)}</div>
        <div>Protocols: ${edge.metrics.protocols.join(', ')}</div>
      </div>
    </div>
  `
}


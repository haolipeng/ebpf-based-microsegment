import type { EChartsOption } from 'echarts'
import type { TopologyData, TopologyNode, TopologyEdge, TopologyViewMode } from '../../types/topology'
import { calculateNodeSize, calculateEdgeWidth } from '../../utils/topologyUtils'
import { formatBytes, formatNumber } from '../../utils/format'

/**
 * 生成ECharts Graph配置
 */
export function getTopologyChartOption(
  data: TopologyData,
  viewMode: TopologyViewMode
): EChartsOption {
  return {
    title: {
      text: `Network Topology (${viewMode === 'IP' ? 'IP View' : 'Service View'})`,
      subtext: `${data.stats.totalNodes} nodes, ${data.stats.totalEdges} connections`,
      left: 'center',
      top: 10,
    },
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        if (params.dataType === 'node') {
          return formatNodeTooltip(params.data)
        } else if (params.dataType === 'edge') {
          return formatEdgeTooltip(params.data)
        }
        return ''
      },
    },
    legend: {
      show: false,
    },
    series: [
      {
        type: 'graph',
        layout: 'force',
        data: data.nodes.map(node => ({
          id: node.id,
          name: node.label,
          value: node.metrics.byteCount,
          symbolSize: calculateNodeSize(node.metrics.flowCount),
          itemStyle: getNodeStyle(node),
          label: {
            show: true,
            position: 'bottom',
            formatter: '{b}',
            fontSize: 10,
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
          repulsion: 300, // 节点之间的斥力
          gravity: 0.1, // 节点受到的向中心的引力
          edgeLength: [100, 200], // 边的两个节点之间的距离
          layoutAnimation: true,
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
 * 获取节点样式
 */
export function getNodeStyle(node: TopologyNode) {
  let color = '#5470c6' // 默认蓝色

  if (node.type === 'SERVICE') {
    color = '#91cc75' // 服务节点-绿色
  }

  // 根据活跃流数量调整颜色深度
  const opacity = node.metrics.activeFlows > 0 ? 1 : 0.6

  return {
    color,
    opacity,
    borderColor: '#fff',
    borderWidth: 2,
    shadowBlur: 10,
    shadowColor: 'rgba(0, 0, 0, 0.3)',
  }
}

/**
 * 获取边样式
 */
export function getEdgeStyle(edge: TopologyEdge) {
  // 根据协议设置颜色
  let color = '#999' // 默认灰色

  if (edge.metrics.protocols.includes('TCP')) {
    color = '#5470c6' // TCP-蓝色
  } else if (edge.metrics.protocols.includes('UDP')) {
    color = '#91cc75' // UDP-绿色
  } else if (edge.metrics.protocols.includes('ICMP')) {
    color = '#fac858' // ICMP-橙色
  }

  const width = calculateEdgeWidth(edge.metrics.byteCount)

  return {
    color,
    width,
    curveness: 0.1, // 边的曲度
    opacity: 0.6,
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


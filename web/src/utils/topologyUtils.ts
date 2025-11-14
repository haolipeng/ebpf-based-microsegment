import type { Flow } from '../types/flow'
import type {
  TopologyData,
  TopologyNode,
  TopologyEdge,
  TopologyViewMode,
} from '../types/topology'

/**
 * 将Flow数组聚合为拓扑图数据
 * 
 * @param flows - 流量记录数组
 * @param viewMode - 视图模式（IP或标签）
 * @param maxNodes - 最多显示的节点数（Top N）
 * @returns 拓扑图数据
 */
export function aggregateFlowsToTopology(
  flows: Flow[],
  viewMode: TopologyViewMode,
  maxNodes?: number
): TopologyData {
  if (viewMode === 'IP') {
    return aggregateByIP(flows, maxNodes)
  } else {
    return aggregateByLabel(flows, maxNodes)
  }
}

/**
 * 按IP地址聚合流量
 */
function aggregateByIP(flows: Flow[], maxNodes?: number): TopologyData {
  const nodeMap = new Map<string, TopologyNode>()
  const edgeMap = new Map<string, TopologyEdge>()

  // 聚合节点和边
  flows.forEach(flow => {
    const sourceId = flow.sourceIp
    const targetId = flow.destIp

    // 创建或更新源节点
    if (!nodeMap.has(sourceId)) {
      nodeMap.set(sourceId, {
        id: sourceId,
        label: sourceId,
        type: 'IP',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
      })
    }

    // 创建或更新目标节点
    if (!nodeMap.has(targetId)) {
      nodeMap.set(targetId, {
        id: targetId,
        label: targetId,
        type: 'IP',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
      })
    }

    // 更新节点指标
    const sourceNode = nodeMap.get(sourceId)!
    const targetNode = nodeMap.get(targetId)!

    sourceNode.metrics.flowCount++
    sourceNode.metrics.packetCount += flow.packetCount
    sourceNode.metrics.byteCount += flow.byteCount
    if (flow.state === 'ACTIVE') {
      sourceNode.metrics.activeFlows++
    }

    targetNode.metrics.flowCount++
    targetNode.metrics.packetCount += flow.packetCount
    targetNode.metrics.byteCount += flow.byteCount
    if (flow.state === 'ACTIVE') {
      targetNode.metrics.activeFlows++
    }

    // 创建或更新边
    const edgeId = `${sourceId}->${targetId}`
    const reverseEdgeId = `${targetId}->${sourceId}`

    if (!edgeMap.has(edgeId) && !edgeMap.has(reverseEdgeId)) {
      // 将UNKNOWN方向映射为EGRESS
      const direction = flow.direction === 'UNKNOWN' ? 'EGRESS' : (flow.direction as 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL')
      edgeMap.set(edgeId, {
        id: edgeId,
        source: sourceId,
        target: targetId,
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          protocols: [],
        },
        direction,
      })
    }

    const edge = edgeMap.get(edgeId) || edgeMap.get(reverseEdgeId)!

    // 更新边指标
    edge.metrics.flowCount++
    edge.metrics.packetCount += flow.packetCount
    edge.metrics.byteCount += flow.byteCount

    // 添加协议（去重）
    if (!edge.metrics.protocols.includes(flow.protocol)) {
      edge.metrics.protocols.push(flow.protocol)
    }

    // 更新方向（如果有双向流量）
    if (edgeMap.has(reverseEdgeId)) {
      edge.direction = 'BIDIRECTIONAL'
    }
  })

  // 转换为数组并按流量排序
  let nodes = Array.from(nodeMap.values())
  const edges = Array.from(edgeMap.values())

  // 限制节点数量（Top N）
  if (maxNodes && nodes.length > maxNodes) {
    nodes = nodes
      .sort((a, b) => b.metrics.byteCount - a.metrics.byteCount)
      .slice(0, maxNodes)

    // 只保留涉及这些节点的边
    const nodeIds = new Set(nodes.map(n => n.id))
    const filteredEdges = edges.filter(
      e => nodeIds.has(e.source) && nodeIds.has(e.target)
    )

    return {
      nodes,
      edges: filteredEdges,
      stats: {
        totalNodes: nodes.length,
        totalEdges: filteredEdges.length,
        totalFlows: flows.length,
      },
    }
  }

  return {
    nodes,
    edges,
    stats: {
      totalNodes: nodes.length,
      totalEdges: edges.length,
      totalFlows: flows.length,
    },
  }
}

/**
 * 按标签聚合流量
 */
function aggregateByLabel(flows: Flow[], maxNodes?: number): TopologyData {
  const nodeMap = new Map<string, TopologyNode>()
  const edgeMap = new Map<string, TopologyEdge>()

  // 聚合节点和边
  flows.forEach(flow => {
    // 获取标签键（如果有多个标签，使用app标签或第一个标签）
    const sourceLabel = getServiceLabel(flow.sourceLabels)
    const targetLabel = getServiceLabel(flow.destLabels)

    // 跳过没有标签的流
    if (!sourceLabel || !targetLabel) {
      return
    }

    const sourceId = sourceLabel
    const targetId = targetLabel

    // 创建或更新源节点
    if (!nodeMap.has(sourceId)) {
      nodeMap.set(sourceId, {
        id: sourceId,
        label: sourceId,
        type: 'SERVICE',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
        labels: flow.sourceLabels,
      })
    }

    // 创建或更新目标节点
    if (!nodeMap.has(targetId)) {
      nodeMap.set(targetId, {
        id: targetId,
        label: targetId,
        type: 'SERVICE',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
        labels: flow.destLabels,
      })
    }

    // 更新节点指标
    const sourceNode = nodeMap.get(sourceId)!
    const targetNode = nodeMap.get(targetId)!

    sourceNode.metrics.flowCount++
    sourceNode.metrics.packetCount += flow.packetCount
    sourceNode.metrics.byteCount += flow.byteCount
    if (flow.state === 'ACTIVE') {
      sourceNode.metrics.activeFlows++
    }

    targetNode.metrics.flowCount++
    targetNode.metrics.packetCount += flow.packetCount
    targetNode.metrics.byteCount += flow.byteCount
    if (flow.state === 'ACTIVE') {
      targetNode.metrics.activeFlows++
    }

    // 创建或更新边
    const edgeId = `${sourceId}->${targetId}`
    const reverseEdgeId = `${targetId}->${sourceId}`

    if (!edgeMap.has(edgeId) && !edgeMap.has(reverseEdgeId)) {
      // 将UNKNOWN方向映射为EGRESS
      const direction = flow.direction === 'UNKNOWN' ? 'EGRESS' : (flow.direction as 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL')
      edgeMap.set(edgeId, {
        id: edgeId,
        source: sourceId,
        target: targetId,
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          protocols: [],
        },
        direction,
      })
    }

    const edge = edgeMap.get(edgeId) || edgeMap.get(reverseEdgeId)!

    // 更新边指标
    edge.metrics.flowCount++
    edge.metrics.packetCount += flow.packetCount
    edge.metrics.byteCount += flow.byteCount

    // 添加协议（去重）
    if (!edge.metrics.protocols.includes(flow.protocol)) {
      edge.metrics.protocols.push(flow.protocol)
    }

    // 更新方向
    if (edgeMap.has(reverseEdgeId)) {
      edge.direction = 'BIDIRECTIONAL'
    }
  })

  // 转换为数组并按流量排序
  let nodes = Array.from(nodeMap.values())
  const edges = Array.from(edgeMap.values())

  // 限制节点数量（Top N）
  if (maxNodes && nodes.length > maxNodes) {
    nodes = nodes
      .sort((a, b) => b.metrics.byteCount - a.metrics.byteCount)
      .slice(0, maxNodes)

    // 只保留涉及这些节点的边
    const nodeIds = new Set(nodes.map(n => n.id))
    const filteredEdges = edges.filter(
      e => nodeIds.has(e.source) && nodeIds.has(e.target)
    )

    return {
      nodes,
      edges: filteredEdges,
      stats: {
        totalNodes: nodes.length,
        totalEdges: filteredEdges.length,
        totalFlows: flows.length,
      },
    }
  }

  return {
    nodes,
    edges,
    stats: {
      totalNodes: nodes.length,
      totalEdges: edges.length,
      totalFlows: flows.length,
    },
  }
}

/**
 * 从标签中获取服务名称
 * 优先使用app标签，否则使用第一个标签的值
 */
function getServiceLabel(labels?: Record<string, string>): string | null {
  if (!labels || Object.keys(labels).length === 0) {
    return null
  }

  // 优先使用app标签
  if (labels.app) {
    return labels.app
  }

  // 使用第一个标签
  const firstKey = Object.keys(labels)[0]
  return `${firstKey}:${labels[firstKey]}`
}

/**
 * 计算节点大小（用于可视化）
 * 根据流量数量返回合适的节点大小
 * 
 * @param flowCount - 流数量
 * @returns 节点大小（像素）
 */
export function calculateNodeSize(flowCount: number): number {
  // 最小20，最大80
  const minSize = 20
  const maxSize = 80
  const baseSize = 30

  // 对数缩放
  if (flowCount <= 1) return minSize

  const size = baseSize + Math.log10(flowCount) * 15
  return Math.min(Math.max(size, minSize), maxSize)
}

/**
 * 计算边宽度（用于可视化）
 * 根据流量大小返回合适的边宽度
 * 
 * @param byteCount - 字节数
 * @returns 边宽度（像素）
 */
export function calculateEdgeWidth(byteCount: number): number {
  // 最小1，最大10
  const minWidth = 1
  const maxWidth = 10
  const baseWidth = 2

  // 对数缩放
  if (byteCount <= 1024) return minWidth // < 1KB

  const width = baseWidth + Math.log10(byteCount / 1024) * 1.5 // 以KB为单位
  return Math.min(Math.max(width, minWidth), maxWidth)
}

/**
 * 合并实时更新的Flow到现有拓扑数据
 * 
 * @param existing - 现有拓扑数据
 * @param newFlow - 新的流记录
 * @param viewMode - 视图模式
 * @returns 更新后的拓扑数据
 */
export function mergeTopologyUpdate(
  existing: TopologyData,
  newFlow: Flow,
  viewMode: TopologyViewMode
): TopologyData {
  // 简单实现：将新flow添加到数组并重新聚合
  // 更高效的实现应该直接更新nodes和edges
  const nodes = [...existing.nodes]
  const edges = [...existing.edges]

  if (viewMode === 'IP') {
    const sourceId = newFlow.sourceIp
    const targetId = newFlow.destIp

    // 更新或创建源节点
    let sourceNode = nodes.find(n => n.id === sourceId)
    if (!sourceNode) {
      sourceNode = {
        id: sourceId,
        label: sourceId,
        type: 'IP',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
      }
      nodes.push(sourceNode)
    }

    // 更新或创建目标节点
    let targetNode = nodes.find(n => n.id === targetId)
    if (!targetNode) {
      targetNode = {
        id: targetId,
        label: targetId,
        type: 'IP',
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          activeFlows: 0,
        },
      }
      nodes.push(targetNode)
    }

    // 更新节点指标
    sourceNode.metrics.flowCount++
    sourceNode.metrics.packetCount += newFlow.packetCount
    sourceNode.metrics.byteCount += newFlow.byteCount
    if (newFlow.state === 'ACTIVE') {
      sourceNode.metrics.activeFlows++
    }

    targetNode.metrics.flowCount++
    targetNode.metrics.packetCount += newFlow.packetCount
    targetNode.metrics.byteCount += newFlow.byteCount
    if (newFlow.state === 'ACTIVE') {
      targetNode.metrics.activeFlows++
    }

    // 更新或创建边
    const edgeId = `${sourceId}->${targetId}`
    let edge = edges.find(e => e.id === edgeId || e.id === `${targetId}->${sourceId}`)

    if (!edge) {
      // 将UNKNOWN方向映射为EGRESS
      const direction = newFlow.direction === 'UNKNOWN' ? 'EGRESS' : (newFlow.direction as 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL')
      edge = {
        id: edgeId,
        source: sourceId,
        target: targetId,
        metrics: {
          flowCount: 0,
          packetCount: 0,
          byteCount: 0,
          protocols: [],
        },
        direction,
      }
      edges.push(edge)
    }

    // 更新边指标
    edge.metrics.flowCount++
    edge.metrics.packetCount += newFlow.packetCount
    edge.metrics.byteCount += newFlow.byteCount

    if (!edge.metrics.protocols.includes(newFlow.protocol)) {
      edge.metrics.protocols.push(newFlow.protocol)
    }
  }

  return {
    nodes,
    edges,
    stats: {
      totalNodes: nodes.length,
      totalEdges: edges.length,
      totalFlows: existing.stats.totalFlows + 1,
    },
  }
}

/**
 * 获取节点的显示标签
 * 
 * @param node - 拓扑节点
 * @returns 显示标签
 */
export function getNodeLabel(node: TopologyNode): string {
  if (node.type === 'IP') {
    return node.label
  }

  // 服务节点，显示服务名称
  if (node.labels?.app) {
    return node.labels.app
  }

  return node.label
}


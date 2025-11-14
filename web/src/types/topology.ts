import type { Flow, FlowQuery } from './flow'

/**
 * 拓扑图节点定义
 */
export interface TopologyNode {
  /** 节点唯一标识 */
  id: string
  /** 节点显示标签 */
  label: string
  /** 节点类型 */
  type: 'IP' | 'SERVICE'
  /** 流量指标 */
  metrics: {
    /** 流数量 */
    flowCount: number
    /** 包数量 */
    packetCount: number
    /** 字节数 */
    byteCount: number
    /** 活跃流数量 */
    activeFlows: number
  }
  /** 标签元数据（标签视图时使用） */
  labels?: Record<string, string>
  /** 可选的固定位置 */
  position?: {
    x: number
    y: number
  }
}

/**
 * 拓扑图边定义
 */
export interface TopologyEdge {
  /** 边唯一标识 */
  id: string
  /** 源节点ID */
  source: string
  /** 目标节点ID */
  target: string
  /** 流量指标 */
  metrics: {
    /** 流数量 */
    flowCount: number
    /** 包数量 */
    packetCount: number
    /** 字节数 */
    byteCount: number
    /** 协议列表 */
    protocols: string[]
  }
  /** 流量方向 */
  direction: 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL'
}

/**
 * 完整拓扑图数据
 */
export interface TopologyData {
  /** 节点列表 */
  nodes: TopologyNode[]
  /** 边列表 */
  edges: TopologyEdge[]
  /** 统计信息 */
  stats: {
    /** 总节点数 */
    totalNodes: number
    /** 总边数 */
    totalEdges: number
    /** 总流数 */
    totalFlows: number
  }
}

/**
 * 拓扑图视图模式
 */
export type TopologyViewMode = 'IP' | 'LABEL'

/**
 * 拓扑图筛选条件
 */
export interface TopologyFilters extends FlowQuery {
  /** 视图模式 */
  viewMode: TopologyViewMode
  /** 最多显示节点数（Top N） */
  maxNodes?: number
}

/**
 * 节点详情信息
 */
export interface NodeDetail {
  /** 节点信息 */
  node: TopologyNode
  /** 相关的流列表 */
  flows: Flow[]
  /** 入站连接 */
  inboundConnections: number
  /** 出站连接 */
  outboundConnections: number
}


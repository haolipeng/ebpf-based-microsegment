/**
 * Topology Aggregator
 *
 * Aggregates network flow data into topology graph structure
 * Supports both IP-level and label-based aggregation
 */

import { Graph } from '@/lib/graph';
import type { Flow } from '@/types/flow';
import type {
  TopologyNode,
  TopologyEdge,
  TopologyData,
  TopologyViewMode,
  TopologyFilters,
} from '@/types/topology';

/**
 * Flow statistics for aggregation
 */
interface FlowStats {
  flowCount: number;
  packetCount: number;
  byteCount: number;
  protocols: Set<string>;
}

/**
 * Edge attribute in the graph
 */
interface EdgeAttr extends FlowStats {
  direction: 'INGRESS' | 'EGRESS' | 'BIDIRECTIONAL';
}

/**
 * Aggregate flows into topology graph using our graph database
 */
export class TopologyAggregator {
  private graph: Graph<EdgeAttr>;
  private nodeMetrics: Map<string, FlowStats>;

  constructor() {
    this.graph = new Graph<EdgeAttr>();
    this.nodeMetrics = new Map();
  }

  /**
   * Aggregate flows into topology data
   */
  aggregate(flows: Flow[], filters: TopologyFilters): TopologyData {
    // Reset graph and metrics
    this.graph.clear();
    this.nodeMetrics.clear();

    // Build graph from flows
    for (const flow of flows) {
      this.addFlowToGraph(flow, filters.viewMode);
    }

    // Convert graph to topology data
    const nodes = this.buildNodes(filters.viewMode);
    const edges = this.buildEdges();

    // Apply filters
    const filteredData = this.applyFilters({ nodes, edges }, filters);

    // Calculate statistics
    const stats = {
      totalNodes: filteredData.nodes.length,
      totalEdges: filteredData.edges.length,
      totalFlows: flows.length,
    };

    return {
      nodes: filteredData.nodes,
      edges: filteredData.edges,
      stats,
    };
  }

  /**
   * Add a single flow to the graph
   */
  private addFlowToGraph(flow: Flow, viewMode: TopologyViewMode): void {
    const srcNode = this.getNodeId(flow, 'source', viewMode);
    const dstNode = this.getNodeId(flow, 'dest', viewMode);

    // Skip if node IDs are invalid
    if (!srcNode || !dstNode) {
      return;
    }

    // Update node metrics
    this.updateNodeMetrics(srcNode, flow);
    this.updateNodeMetrics(dstNode, flow);

    // Update edge in graph
    this.updateEdgeInGraph(srcNode, dstNode, flow);
  }

  /**
   * Get node ID based on view mode
   */
  private getNodeId(
    flow: Flow,
    endpoint: 'source' | 'dest',
    viewMode: TopologyViewMode
  ): string | null {
    if (viewMode === 'IP') {
      return endpoint === 'source' ? flow.sourceIp : flow.destIp;
    }

    // Label-based view mode
    const labels = endpoint === 'source' ? flow.sourceLabels : flow.destLabels;
    if (!labels || Object.keys(labels).length === 0) {
      // Fallback to IP if no labels
      return endpoint === 'source' ? flow.sourceIp : flow.destIp;
    }

    // Create a label-based identifier (e.g., "app=frontend,env=prod")
    return this.labelsToString(labels);
  }

  /**
   * Convert labels to string identifier
   */
  private labelsToString(labels: Record<string, string>): string {
    return Object.entries(labels)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([k, v]) => `${k}=${v}`)
      .join(',');
  }

  /**
   * Update node metrics
   */
  private updateNodeMetrics(nodeId: string, flow: Flow): void {
    let metrics = this.nodeMetrics.get(nodeId);
    if (!metrics) {
      metrics = {
        flowCount: 0,
        packetCount: 0,
        byteCount: 0,
        protocols: new Set(),
      };
      this.nodeMetrics.set(nodeId, metrics);
    }

    metrics.flowCount += 1;
    metrics.packetCount += flow.packetCount;
    metrics.byteCount += flow.byteCount;
    metrics.protocols.add(flow.protocol);
  }

  /**
   * Update edge in graph
   */
  private updateEdgeInGraph(src: string, dst: string, flow: Flow): void {
    const linkType = 'traffic';

    // Get existing edge attribute
    const existingAttr = this.graph.attr(src, linkType, dst);

    if (existingAttr) {
      // Update existing edge
      existingAttr.flowCount += 1;
      existingAttr.packetCount += flow.packetCount;
      existingAttr.byteCount += flow.byteCount;
      existingAttr.protocols.add(flow.protocol);

      // Update direction to bidirectional if necessary
      if (existingAttr.direction !== flow.direction) {
        existingAttr.direction = 'BIDIRECTIONAL';
      }

      this.graph.addLink(src, linkType, dst, existingAttr);
    } else {
      // Create new edge
      const newAttr: EdgeAttr = {
        flowCount: 1,
        packetCount: flow.packetCount,
        byteCount: flow.byteCount,
        protocols: new Set([flow.protocol]),
        direction: flow.direction,
      };

      this.graph.addLink(src, linkType, dst, newAttr);
    }
  }

  /**
   * Build topology nodes from graph
   */
  private buildNodes(viewMode: TopologyViewMode): TopologyNode[] {
    const nodes: TopologyNode[] = [];

    for (const nodeId of this.graph.all()) {
      const metrics = this.nodeMetrics.get(nodeId);
      if (!metrics) {
        continue;
      }

      const node: TopologyNode = {
        id: nodeId,
        label: this.getNodeLabel(nodeId, viewMode),
        type: viewMode === 'IP' ? 'IP' : 'SERVICE',
        metrics: {
          flowCount: metrics.flowCount,
          packetCount: metrics.packetCount,
          byteCount: metrics.byteCount,
          activeFlows: metrics.flowCount, // Simplified for now
        },
      };

      // Add labels if in label view mode
      if (viewMode === 'LABEL' && nodeId.includes('=')) {
        node.labels = this.stringToLabels(nodeId);
      }

      nodes.push(node);
    }

    return nodes;
  }

  /**
   * Get node display label
   */
  private getNodeLabel(nodeId: string, viewMode: TopologyViewMode): string {
    if (viewMode === 'IP') {
      return nodeId;
    }

    // For label-based view, extract primary label (e.g., "app=frontend")
    const labels = this.stringToLabels(nodeId);
    const primaryLabel = labels.app || labels.service || labels.name;
    return primaryLabel || nodeId;
  }

  /**
   * Convert string identifier back to labels
   */
  private stringToLabels(str: string): Record<string, string> {
    const labels: Record<string, string> = {};
    const pairs = str.split(',');

    for (const pair of pairs) {
      const [key, value] = pair.split('=');
      if (key && value) {
        labels[key] = value;
      }
    }

    return labels;
  }

  /**
   * Build topology edges from graph
   */
  private buildEdges(): TopologyEdge[] {
    const edges: TopologyEdge[] = [];
    const linkType = 'traffic';

    for (const src of this.graph.all()) {
      const neighbors = this.graph.outs(src, linkType);

      for (const dst of neighbors) {
        const attr = this.graph.attr(src, linkType, dst);
        if (!attr) {
          continue;
        }

        const edge: TopologyEdge = {
          id: `${src}-${dst}`,
          source: src,
          target: dst,
          metrics: {
            flowCount: attr.flowCount,
            packetCount: attr.packetCount,
            byteCount: attr.byteCount,
            protocols: Array.from(attr.protocols),
          },
          direction: attr.direction,
        };

        edges.push(edge);
      }
    }

    return edges;
  }

  /**
   * Apply filters to topology data
   */
  private applyFilters(
    data: { nodes: TopologyNode[]; edges: TopologyEdge[] },
    filters: TopologyFilters
  ): { nodes: TopologyNode[]; edges: TopologyEdge[] } {
    let { nodes, edges } = data;

    // Apply maxNodes filter (show top N nodes by traffic)
    if (filters.maxNodes && nodes.length > filters.maxNodes) {
      // Sort nodes by byte count descending
      const sortedNodes = [...nodes].sort(
        (a, b) => b.metrics.byteCount - a.metrics.byteCount
      );

      // Keep top N nodes
      const topNodes = sortedNodes.slice(0, filters.maxNodes);
      const topNodeIds = new Set(topNodes.map(n => n.id));

      // Filter edges to only include those between top nodes
      edges = edges.filter(
        e => topNodeIds.has(e.source) && topNodeIds.has(e.target)
      );

      nodes = topNodes;
    }

    return { nodes, edges };
  }

  /**
   * Get the underlying graph for advanced queries
   */
  getGraph(): Graph<EdgeAttr> {
    return this.graph;
  }

  /**
   * Find connected components in the topology
   */
  findConnectedComponents(): Set<string>[] {
    const { findConnectedComponents } = require('@/lib/graph/algorithms');
    return findConnectedComponents(this.graph, 'traffic');
  }

  /**
   * Find nodes with highest traffic (hubs)
   */
  findTrafficHubs(topN: number = 10): Array<[string, number]> {
    const { findHubs } = require('@/lib/graph/algorithms');
    return findHubs(this.graph, topN, 'traffic', 'both');
  }

  /**
   * Find nodes within N hops of a given node
   */
  findNeighborhood(
    nodeId: string,
    maxHops: number
  ): Map<string, number> {
    const { findNodesWithinHops } = require('@/lib/graph/algorithms');
    return findNodesWithinHops(this.graph, nodeId, maxHops, 'traffic');
  }

  /**
   * Get node detail information
   */
  getNodeDetail(nodeId: string): {
    inboundConnections: number;
    outboundConnections: number;
    totalTraffic: number;
  } | null {
    const metrics = this.nodeMetrics.get(nodeId);
    if (!metrics) {
      return null;
    }

    const inboundConnections = this.graph.ins(nodeId, 'traffic').length;
    const outboundConnections = this.graph.outs(nodeId, 'traffic').length;

    return {
      inboundConnections,
      outboundConnections,
      totalTraffic: metrics.byteCount,
    };
  }
}

/**
 * Create a topology aggregator instance
 */
export function createTopologyAggregator(): TopologyAggregator {
  return new TopologyAggregator();
}

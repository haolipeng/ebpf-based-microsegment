/**
 * Enhanced Topology Graph Hook
 *
 * Provides advanced topology analysis and interaction capabilities
 * using the graph database
 */

import { useRef, useMemo, useCallback } from 'react';
import { TopologyAggregator, createTopologyAggregator } from '@/utils/topologyAggregator';
import type { Flow } from '@/types/flow';
import type { TopologyData, TopologyFilters, TopologyNode } from '@/types/topology';
import { Graph } from '@/lib/graph';

interface UseTopologyGraphOptions {
  /** Flow data source */
  flows: Flow[];
  /** Topology filters */
  filters: TopologyFilters;
  /** Enable auto-refresh */
  autoRefresh?: boolean;
  /** Refresh interval in milliseconds */
  refreshInterval?: number;
}

interface TopologyGraphState {
  /** Current topology data */
  data: TopologyData;
  /** Underlying graph database */
  graph: Graph<any>;
  /** Topology aggregator instance */
  aggregator: TopologyAggregator;
}

interface TopologyGraphActions {
  /** Find nodes connected to a given node */
  findConnectedNodes: (nodeId: string, maxHops?: number) => Set<string>;
  /** Find traffic hubs (high-degree nodes) */
  findTrafficHubs: (topN?: number) => Array<[string, number]>;
  /** Find connected components */
  findConnectedComponents: () => Set<string>[];
  /** Get detailed node information */
  getNodeDetail: (nodeId: string) => {
    inboundConnections: number;
    outboundConnections: number;
    totalTraffic: number;
  } | null;
  /** Get neighbors of a node */
  getNeighbors: (nodeId: string, direction?: 'in' | 'out' | 'both') => string[];
  /** Refresh topology data */
  refresh: () => void;
}

/**
 * Enhanced topology graph hook with graph database integration
 */
export function useTopologyGraph(
  options: UseTopologyGraphOptions
): TopologyGraphState & TopologyGraphActions {
  const { flows, filters } = options;

  // Create aggregator instance (reuse across renders)
  const aggregatorRef = useRef<TopologyAggregator | null>(null);
  if (!aggregatorRef.current) {
    aggregatorRef.current = createTopologyAggregator();
  }
  const aggregator = aggregatorRef.current;

  // Compute topology data
  const data = useMemo(() => {
    return aggregator.aggregate(flows, filters);
  }, [flows, filters, aggregator]);

  // Get graph instance
  const graph = useMemo(() => {
    return aggregator.getGraph();
  }, [aggregator]);

  // Find connected nodes
  const findConnectedNodes = useCallback(
    (nodeId: string, maxHops: number = 2): Set<string> => {
      const neighborhood = aggregator.findNeighborhood(nodeId, maxHops);
      return new Set(neighborhood.keys());
    },
    [aggregator]
  );

  // Find traffic hubs
  const findTrafficHubs = useCallback(
    (topN: number = 10): Array<[string, number]> => {
      return aggregator.findTrafficHubs(topN);
    },
    [aggregator]
  );

  // Find connected components
  const findConnectedComponents = useCallback((): Set<string>[] => {
    return aggregator.findConnectedComponents();
  }, [aggregator]);

  // Get node detail
  const getNodeDetail = useCallback(
    (nodeId: string) => {
      return aggregator.getNodeDetail(nodeId);
    },
    [aggregator]
  );

  // Get neighbors
  const getNeighbors = useCallback(
    (nodeId: string, direction: 'in' | 'out' | 'both' = 'both'): string[] => {
      if (direction === 'in') {
        return graph.ins(nodeId, 'traffic');
      } else if (direction === 'out') {
        return graph.outs(nodeId, 'traffic');
      } else {
        return graph.both(nodeId, 'traffic');
      }
    },
    [graph]
  );

  // Refresh (re-aggregate)
  const refresh = useCallback(() => {
    // This will trigger re-computation via useMemo
    aggregator.aggregate(flows, filters);
  }, [aggregator, flows, filters]);

  return {
    data,
    graph,
    aggregator,
    findConnectedNodes,
    findTrafficHubs,
    findConnectedComponents,
    getNodeDetail,
    getNeighbors,
    refresh,
  };
}

/**
 * Hook for node selection and focus
 */
interface UseNodeFocusOptions {
  /** Initial selected node */
  initialNode?: TopologyNode | null;
  /** Callback when selection changes */
  onSelectionChange?: (node: TopologyNode | null) => void;
}

export function useNodeFocus(options: UseNodeFocusOptions = {}) {
  const { initialNode = null, onSelectionChange } = options;
  const [selectedNode, setSelectedNode] = React.useState<TopologyNode | null>(initialNode);
  const [focusedNeighbors, setFocusedNeighbors] = React.useState<Set<string>>(new Set());

  const selectNode = useCallback(
    (node: TopologyNode | null) => {
      setSelectedNode(node);
      onSelectionChange?.(node);
    },
    [onSelectionChange]
  );

  const focusOnNode = useCallback((node: TopologyNode, neighbors: string[]) => {
    setSelectedNode(node);
    setFocusedNeighbors(new Set(neighbors));
  }, []);

  const clearFocus = useCallback(() => {
    setSelectedNode(null);
    setFocusedNeighbors(new Set());
  }, []);

  const isNodeFocused = useCallback(
    (nodeId: string): boolean => {
      if (!selectedNode) return false;
      return selectedNode.id === nodeId || focusedNeighbors.has(nodeId);
    },
    [selectedNode, focusedNeighbors]
  );

  return {
    selectedNode,
    focusedNeighbors,
    selectNode,
    focusOnNode,
    clearFocus,
    isNodeFocused,
  };
}

/**
 * Hook for topology statistics
 */
export function useTopologyStats(data: TopologyData) {
  const stats = useMemo(() => {
    const { nodes, edges } = data;

    // Calculate degree distribution
    const degreeMap = new Map<string, number>();
    for (const edge of edges) {
      degreeMap.set(edge.source, (degreeMap.get(edge.source) || 0) + 1);
      degreeMap.set(edge.target, (degreeMap.get(edge.target) || 0) + 1);
    }

    const degrees = Array.from(degreeMap.values());
    const avgDegree = degrees.length > 0 ? degrees.reduce((a, b) => a + b, 0) / degrees.length : 0;
    const maxDegree = degrees.length > 0 ? Math.max(...degrees) : 0;

    // Calculate traffic distribution
    const totalBytes = nodes.reduce((sum, node) => sum + node.metrics.byteCount, 0);
    const avgBytes = nodes.length > 0 ? totalBytes / nodes.length : 0;

    // Find isolated nodes (no connections)
    const connectedNodes = new Set<string>();
    for (const edge of edges) {
      connectedNodes.add(edge.source);
      connectedNodes.add(edge.target);
    }
    const isolatedCount = nodes.filter(n => !connectedNodes.has(n.id)).length;

    return {
      avgDegree,
      maxDegree,
      totalBytes,
      avgBytes,
      isolatedCount,
      density: nodes.length > 1 ? edges.length / (nodes.length * (nodes.length - 1)) : 0,
    };
  }, [data]);

  return stats;
}

// Re-export React for the useNodeFocus hook
import React from 'react';

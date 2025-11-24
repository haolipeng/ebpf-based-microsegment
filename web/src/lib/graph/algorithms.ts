/**
 * Graph Algorithms
 *
 * Provides graph traversal and analysis algorithms
 * Inspired by NeuVector's connected component analysis
 */

import { Graph } from './Graph';
import type { ConnectedNodeCallback } from './types';

/**
 * Find all nodes connected to the start node via BFS traversal
 * Supports optional filtering and link type restrictions
 *
 * @param graph - The graph to traverse
 * @param start - Starting node
 * @param linkType - Optional link type filter
 * @param filter - Optional callback to filter connected nodes
 * @returns Set of connected node names
 */
export function findConnectedNodes<TAttr = any>(
  graph: Graph<TAttr>,
  start: string,
  linkType?: string,
  filter?: ConnectedNodeCallback
): Set<string> {
  if (!graph.hasNode(start)) {
    return new Set();
  }

  const visited = new Set<string>();
  const queue: string[] = [start];
  visited.add(start);

  while (queue.length > 0) {
    const current = queue.shift()!;

    // Get all neighbors (both incoming and outgoing)
    const neighbors = graph.both(current, linkType);

    for (const neighbor of neighbors) {
      if (visited.has(neighbor)) {
        continue;
      }

      // Apply filter if provided
      if (filter && !filter(neighbor)) {
        continue;
      }

      visited.add(neighbor);
      queue.push(neighbor);
    }
  }

  return visited;
}

/**
 * Find all connected components in the graph
 * Uses BFS to identify disjoint subgraphs
 *
 * @param graph - The graph to analyze
 * @param linkType - Optional link type filter
 * @returns Array of connected components (each is a set of node names)
 */
export function findConnectedComponents<TAttr = any>(
  graph: Graph<TAttr>,
  linkType?: string
): Set<string>[] {
  const allNodes = graph.all();
  const visited = new Set<string>();
  const components: Set<string>[] = [];

  for (const node of allNodes) {
    if (visited.has(node)) {
      continue;
    }

    // Find all nodes connected to this node
    const component = findConnectedNodes(graph, node, linkType);

    // Mark all nodes in this component as visited
    for (const n of component) {
      visited.add(n);
    }

    components.push(component);
  }

  return components;
}

/**
 * Find shortest path between two nodes using BFS
 *
 * @param graph - The graph to search
 * @param start - Start node
 * @param end - End node
 * @param linkType - Optional link type filter
 * @returns Array of node names representing the path, or empty array if no path exists
 */
export function findShortestPath<TAttr = any>(
  graph: Graph<TAttr>,
  start: string,
  end: string,
  linkType?: string
): string[] {
  if (!graph.hasNode(start) || !graph.hasNode(end)) {
    return [];
  }

  if (start === end) {
    return [start];
  }

  const visited = new Set<string>();
  const queue: Array<{ node: string; path: string[] }> = [{ node: start, path: [start] }];
  visited.add(start);

  while (queue.length > 0) {
    const { node: current, path } = queue.shift()!;

    // Get outgoing neighbors only (directed path)
    const neighbors = graph.outs(current, linkType);

    for (const neighbor of neighbors) {
      if (visited.has(neighbor)) {
        continue;
      }

      const newPath = [...path, neighbor];

      if (neighbor === end) {
        return newPath;
      }

      visited.add(neighbor);
      queue.push({ node: neighbor, path: newPath });
    }
  }

  return [];
}

/**
 * Calculate degree centrality for all nodes
 * Returns a map of node -> degree count
 *
 * @param graph - The graph to analyze
 * @param linkType - Optional link type filter
 * @param direction - 'in' for in-degree, 'out' for out-degree, 'both' for total degree
 */
export function calculateDegreeCentrality<TAttr = any>(
  graph: Graph<TAttr>,
  linkType?: string,
  direction: 'in' | 'out' | 'both' = 'both'
): Map<string, number> {
  const centrality = new Map<string, number>();

  for (const node of graph.all()) {
    let degree = 0;

    if (direction === 'in' || direction === 'both') {
      degree += graph.ins(node, linkType).length;
    }

    if (direction === 'out' || direction === 'both') {
      degree += graph.outs(node, linkType).length;
    }

    centrality.set(node, degree);
  }

  return centrality;
}

/**
 * Find nodes with the highest degree centrality
 *
 * @param graph - The graph to analyze
 * @param topN - Number of top nodes to return
 * @param linkType - Optional link type filter
 * @param direction - Direction to calculate degree
 * @returns Array of [node, degree] tuples sorted by degree descending
 */
export function findHubs<TAttr = any>(
  graph: Graph<TAttr>,
  topN: number = 10,
  linkType?: string,
  direction: 'in' | 'out' | 'both' = 'both'
): Array<[string, number]> {
  const centrality = calculateDegreeCentrality(graph, linkType, direction);

  return Array.from(centrality.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, topN);
}

/**
 * Check if the graph contains a cycle using DFS
 *
 * @param graph - The graph to check
 * @param linkType - Optional link type filter
 * @returns true if the graph contains at least one cycle
 */
export function hasCycle<TAttr = any>(
  graph: Graph<TAttr>,
  linkType?: string
): boolean {
  const visited = new Set<string>();
  const recursionStack = new Set<string>();

  function dfs(node: string): boolean {
    visited.add(node);
    recursionStack.add(node);

    const neighbors = graph.outs(node, linkType);

    for (const neighbor of neighbors) {
      if (!visited.has(neighbor)) {
        if (dfs(neighbor)) {
          return true;
        }
      } else if (recursionStack.has(neighbor)) {
        // Found a back edge (cycle)
        return true;
      }
    }

    recursionStack.delete(node);
    return false;
  }

  for (const node of graph.all()) {
    if (!visited.has(node)) {
      if (dfs(node)) {
        return true;
      }
    }
  }

  return false;
}

/**
 * Perform topological sort on the graph
 * Only works for DAGs (Directed Acyclic Graphs)
 *
 * @param graph - The graph to sort
 * @param linkType - Optional link type filter
 * @returns Topologically sorted array of nodes, or empty array if graph has cycles
 */
export function topologicalSort<TAttr = any>(
  graph: Graph<TAttr>,
  linkType?: string
): string[] {
  if (hasCycle(graph, linkType)) {
    return [];
  }

  const visited = new Set<string>();
  const stack: string[] = [];

  function dfs(node: string): void {
    visited.add(node);

    const neighbors = graph.outs(node, linkType);
    for (const neighbor of neighbors) {
      if (!visited.has(neighbor)) {
        dfs(neighbor);
      }
    }

    stack.push(node);
  }

  for (const node of graph.all()) {
    if (!visited.has(node)) {
      dfs(node);
    }
  }

  return stack.reverse();
}

/**
 * Find all nodes within N hops of the start node
 *
 * @param graph - The graph to search
 * @param start - Starting node
 * @param maxHops - Maximum number of hops
 * @param linkType - Optional link type filter
 * @returns Map of node -> hop distance
 */
export function findNodesWithinHops<TAttr = any>(
  graph: Graph<TAttr>,
  start: string,
  maxHops: number,
  linkType?: string
): Map<string, number> {
  if (!graph.hasNode(start)) {
    return new Map();
  }

  const distances = new Map<string, number>();
  const queue: Array<{ node: string; distance: number }> = [{ node: start, distance: 0 }];
  distances.set(start, 0);

  while (queue.length > 0) {
    const { node: current, distance } = queue.shift()!;

    if (distance >= maxHops) {
      continue;
    }

    const neighbors = graph.both(current, linkType);

    for (const neighbor of neighbors) {
      if (distances.has(neighbor)) {
        continue;
      }

      const newDistance = distance + 1;
      distances.set(neighbor, newDistance);
      queue.push({ node: neighbor, distance: newDistance });
    }
  }

  return distances;
}

/**
 * Aggregate graph data by grouping nodes based on a key function
 *
 * @param graph - The graph to aggregate
 * @param keyFn - Function to extract group key from node name
 * @param linkType - Optional link type filter
 * @returns New graph with aggregated nodes
 */
export function aggregateGraph<TAttr = any>(
  graph: Graph<TAttr>,
  keyFn: (node: string) => string,
  linkType?: string
): Graph<TAttr> {
  const aggregated = new Graph<TAttr>();
  const nodeToGroup = new Map<string, string>();

  // First pass: assign nodes to groups
  for (const node of graph.all()) {
    const groupKey = keyFn(node);
    nodeToGroup.set(node, groupKey);
  }

  // Second pass: create aggregated edges
  for (const node of graph.all()) {
    const srcGroup = nodeToGroup.get(node)!;
    const neighbors = graph.outs(node, linkType);

    for (const neighbor of neighbors) {
      const dstGroup = nodeToGroup.get(neighbor)!;

      // Skip self-loops in aggregated graph
      if (srcGroup === dstGroup) {
        continue;
      }

      const attr = graph.attr(node, linkType || 'default', neighbor);
      if (attr !== undefined) {
        aggregated.addLink(srcGroup, linkType || 'default', dstGroup, attr);
      }
    }
  }

  return aggregated;
}

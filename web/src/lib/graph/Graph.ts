/**
 * Graph Database Implementation
 *
 * Inspired by NeuVector's in-memory graph database
 * Provides a lightweight directed multigraph with O(1) operations
 */

import {
  GraphNode,
  GraphLink,
  NewLinkCallback,
  DelNodeCallback,
  DelLinkCallback,
  UpdateLinkAttrCallback,
  ConnectedNodeCallback,
  GraphStats,
  GraphExportOptions,
  D3GraphData,
  CytoscapeElement,
} from './types';

/**
 * Main Graph class - manages nodes and edges with callbacks
 */
export class Graph<TAttr = any> {
  private nodes: Map<string, GraphNode<TAttr>> = new Map();
  private cbNewLink?: NewLinkCallback;
  private cbDelNode?: DelNodeCallback;
  private cbDelLink?: DelLinkCallback;
  private cbUpdateLinkAttr?: UpdateLinkAttrCallback;

  constructor() {
    this.nodes = new Map();
  }

  /**
   * Register callback for new link creation
   */
  registerNewLinkCallback(cb: NewLinkCallback): void {
    this.cbNewLink = cb;
  }

  /**
   * Register callback for node deletion
   */
  registerDelNodeCallback(cb: DelNodeCallback): void {
    this.cbDelNode = cb;
  }

  /**
   * Register callback for link deletion
   */
  registerDelLinkCallback(cb: DelLinkCallback): void {
    this.cbDelLink = cb;
  }

  /**
   * Register callback for link attribute update
   */
  registerUpdateLinkAttrCallback(cb: UpdateLinkAttrCallback): void {
    this.cbUpdateLinkAttr = cb;
  }

  /**
   * Add a directed edge from src to dst with given link type and attribute
   * O(1) operation
   * Creates nodes automatically if they don't exist
   */
  addLink(src: string, linkType: string, dst: string, attr: TAttr): void {
    // Ensure source node exists
    let srcNode = this.nodes.get(src);
    if (!srcNode) {
      srcNode = new GraphNode<TAttr>();
      this.nodes.set(src, srcNode);
    }

    // Ensure destination node exists
    let dstNode = this.nodes.get(dst);
    if (!dstNode) {
      dstNode = new GraphNode<TAttr>();
      this.nodes.set(dst, dstNode);
    }

    // Add outgoing link from src
    let outLink = srcNode.outs.get(linkType);
    if (!outLink) {
      outLink = new GraphLink<TAttr>();
      srcNode.outs.set(linkType, outLink);
    }

    const isNewLink = !outLink.ends.has(dst);
    outLink.ends.set(dst, attr);

    // Add incoming link to dst
    let inLink = dstNode.ins.get(linkType);
    if (!inLink) {
      inLink = new GraphLink<TAttr>();
      dstNode.ins.set(linkType, inLink);
    }
    inLink.ends.set(src, attr);

    // Trigger callbacks
    if (isNewLink && this.cbNewLink) {
      this.cbNewLink(src, linkType, dst);
    } else if (!isNewLink && this.cbUpdateLinkAttr) {
      this.cbUpdateLinkAttr(src, linkType, dst);
    }
  }

  /**
   * Delete a specific directed edge
   * O(1) operation
   */
  deleteLink(src: string, linkType: string, dst: string): void {
    const srcNode = this.nodes.get(src);
    const dstNode = this.nodes.get(dst);

    if (!srcNode || !dstNode) {
      return;
    }

    // Remove from source node's outgoing links
    const outLink = srcNode.outs.get(linkType);
    if (outLink) {
      outLink.ends.delete(dst);
      if (outLink.ends.size === 0) {
        srcNode.outs.delete(linkType);
      }
    }

    // Remove from destination node's incoming links
    const inLink = dstNode.ins.get(linkType);
    if (inLink) {
      inLink.ends.delete(src);
      if (inLink.ends.size === 0) {
        dstNode.ins.delete(linkType);
      }
    }

    // Trigger callback
    if (this.cbDelLink) {
      this.cbDelLink(src, linkType, dst);
    }
  }

  /**
   * Delete a node and all its associated edges
   * Cascades to remove all incoming and outgoing links
   */
  deleteNode(node: string): void {
    const nodeData = this.nodes.get(node);
    if (!nodeData) {
      return;
    }

    // Delete all outgoing links
    for (const [linkType, link] of nodeData.outs.entries()) {
      for (const dst of link.ends.keys()) {
        this.deleteLink(node, linkType, dst);
      }
    }

    // Delete all incoming links
    for (const [linkType, link] of nodeData.ins.entries()) {
      for (const src of link.ends.keys()) {
        this.deleteLink(src, linkType, node);
      }
    }

    // Remove the node itself
    this.nodes.delete(node);

    // Trigger callback
    if (this.cbDelNode) {
      this.cbDelNode(node);
    }
  }

  /**
   * Get the attribute of a specific edge
   * Returns undefined if edge doesn't exist
   */
  attr(src: string, linkType: string, dst: string): TAttr | undefined {
    const srcNode = this.nodes.get(src);
    if (!srcNode) {
      return undefined;
    }

    const outLink = srcNode.outs.get(linkType);
    if (!outLink) {
      return undefined;
    }

    return outLink.ends.get(dst);
  }

  /**
   * Get all incoming neighbors of a node for a specific link type
   * If linkType is not specified, returns all incoming neighbors across all link types
   * O(1) for specific link type, O(k) where k is number of link types otherwise
   */
  ins(node: string, linkType?: string): string[] {
    const nodeData = this.nodes.get(node);
    if (!nodeData) {
      return [];
    }

    if (linkType) {
      const link = nodeData.ins.get(linkType);
      return link ? Array.from(link.ends.keys()) : [];
    }

    // Get all incoming neighbors across all link types
    const neighbors = new Set<string>();
    for (const link of nodeData.ins.values()) {
      for (const neighbor of link.ends.keys()) {
        neighbors.add(neighbor);
      }
    }
    return Array.from(neighbors);
  }

  /**
   * Get all outgoing neighbors of a node for a specific link type
   * If linkType is not specified, returns all outgoing neighbors across all link types
   * O(1) for specific link type, O(k) where k is number of link types otherwise
   */
  outs(node: string, linkType?: string): string[] {
    const nodeData = this.nodes.get(node);
    if (!nodeData) {
      return [];
    }

    if (linkType) {
      const link = nodeData.outs.get(linkType);
      return link ? Array.from(link.ends.keys()) : [];
    }

    // Get all outgoing neighbors across all link types
    const neighbors = new Set<string>();
    for (const link of nodeData.outs.values()) {
      for (const neighbor of link.ends.keys()) {
        neighbors.add(neighbor);
      }
    }
    return Array.from(neighbors);
  }

  /**
   * Get all neighbors (both incoming and outgoing) of a node
   * If linkType is not specified, returns all neighbors across all link types
   */
  both(node: string, linkType?: string): string[] {
    const inNeighbors = this.ins(node, linkType);
    const outNeighbors = this.outs(node, linkType);

    const neighbors = new Set<string>([...inNeighbors, ...outNeighbors]);
    return Array.from(neighbors);
  }

  /**
   * Get all nodes in the graph
   */
  all(): string[] {
    return Array.from(this.nodes.keys());
  }

  /**
   * Check if a node exists in the graph
   */
  hasNode(node: string): boolean {
    return this.nodes.has(node);
  }

  /**
   * Get the number of nodes in the graph
   */
  getNodeCount(): number {
    return this.nodes.size;
  }

  /**
   * Get the total number of edges in the graph
   */
  getEdgeCount(): number {
    let count = 0;
    for (const node of this.nodes.values()) {
      for (const link of node.outs.values()) {
        count += link.ends.size;
      }
    }
    return count;
  }

  /**
   * Get all unique link types in the graph
   */
  getLinkTypes(): string[] {
    const linkTypes = new Set<string>();
    for (const node of this.nodes.values()) {
      for (const linkType of node.outs.keys()) {
        linkTypes.add(linkType);
      }
    }
    return Array.from(linkTypes);
  }

  /**
   * Get graph statistics
   */
  getStats(): GraphStats {
    const nodeCount = this.getNodeCount();
    const edgeCount = this.getEdgeCount();
    const linkTypes = this.getLinkTypes();
    const averageDegree = nodeCount > 0 ? (edgeCount * 2) / nodeCount : 0;

    return {
      nodeCount,
      edgeCount,
      linkTypes,
      averageDegree,
    };
  }

  /**
   * Clear all nodes and edges from the graph
   */
  clear(): void {
    this.nodes.clear();
  }

  /**
   * Export graph data in various formats
   */
  export(options: GraphExportOptions): any {
    const { format, includeAttributes = false, linkTypes } = options;

    switch (format) {
      case 'json':
        return this.exportJSON(includeAttributes, linkTypes);
      case 'd3':
        return this.exportD3(includeAttributes, linkTypes);
      case 'cytoscape':
        return this.exportCytoscape(includeAttributes, linkTypes);
      default:
        throw new Error(`Unsupported export format: ${format}`);
    }
  }

  /**
   * Export as raw JSON structure
   */
  private exportJSON(includeAttributes: boolean, linkTypes?: string[]): any {
    const result: any = {
      nodes: this.all(),
      edges: [],
    };

    for (const [src, srcNode] of this.nodes.entries()) {
      for (const [linkType, link] of srcNode.outs.entries()) {
        if (linkTypes && !linkTypes.includes(linkType)) {
          continue;
        }

        for (const [dst, attr] of link.ends.entries()) {
          const edge: any = {
            source: src,
            target: dst,
            type: linkType,
          };

          if (includeAttributes && attr !== undefined) {
            edge.attributes = attr;
          }

          result.edges.push(edge);
        }
      }
    }

    return result;
  }

  /**
   * Export in D3.js force graph format
   */
  private exportD3(includeAttributes: boolean, linkTypes?: string[]): D3GraphData {
    const nodes = this.all().map(id => ({ id }));
    const links: D3GraphData['links'] = [];

    for (const [src, srcNode] of this.nodes.entries()) {
      for (const [linkType, link] of srcNode.outs.entries()) {
        if (linkTypes && !linkTypes.includes(linkType)) {
          continue;
        }

        for (const [dst, attr] of link.ends.entries()) {
          const linkData: any = {
            source: src,
            target: dst,
            type: linkType,
          };

          if (includeAttributes && attr !== undefined) {
            Object.assign(linkData, attr);
          }

          links.push(linkData);
        }
      }
    }

    return { nodes, links };
  }

  /**
   * Export in Cytoscape.js format
   */
  private exportCytoscape(includeAttributes: boolean, linkTypes?: string[]): CytoscapeElement[] {
    const elements: CytoscapeElement[] = [];

    // Add nodes
    for (const node of this.all()) {
      elements.push({
        data: { id: node },
      });
    }

    // Add edges
    for (const [src, srcNode] of this.nodes.entries()) {
      for (const [linkType, link] of srcNode.outs.entries()) {
        if (linkTypes && !linkTypes.includes(linkType)) {
          continue;
        }

        for (const [dst, attr] of link.ends.entries()) {
          const edgeData: any = {
            id: `${src}-${linkType}-${dst}`,
            source: src,
            target: dst,
            type: linkType,
          };

          if (includeAttributes && attr !== undefined) {
            Object.assign(edgeData, attr);
          }

          elements.push({ data: edgeData });
        }
      }
    }

    return elements;
  }
}

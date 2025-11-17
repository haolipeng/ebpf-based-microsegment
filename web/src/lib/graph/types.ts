/**
 * Graph Database Types
 *
 * Inspired by NeuVector's graph implementation
 * Provides a lightweight in-memory graph database for network topology
 */

/**
 * Callback type for new link creation
 */
export type NewLinkCallback = (src: string, linkType: string, dst: string) => void;

/**
 * Callback type for node deletion
 */
export type DelNodeCallback = (node: string) => void;

/**
 * Callback type for link deletion
 */
export type DelLinkCallback = (src: string, linkType: string, dst: string) => void;

/**
 * Callback type for link attribute update
 */
export type UpdateLinkAttrCallback = (src: string, linkType: string, dst: string) => void;

/**
 * Callback type for connected node filtering
 */
export type ConnectedNodeCallback = (node: string) => boolean;

/**
 * Graph link - stores connections to multiple end nodes
 * Key: target node name
 * Value: link attribute (any data associated with the edge)
 */
export class GraphLink<TAttr = any> {
  ends: Map<string, TAttr> = new Map();

  constructor() {
    this.ends = new Map();
  }
}

/**
 * Graph node - maintains incoming and outgoing links
 * Each link type (e.g., "traffic", "policy", "attr") has separate storage
 */
export class GraphNode<TAttr = any> {
  ins: Map<string, GraphLink<TAttr>> = new Map();   // Incoming links by link type
  outs: Map<string, GraphLink<TAttr>> = new Map();  // Outgoing links by link type

  constructor() {
    this.ins = new Map();
    this.outs = new Map();
  }
}

/**
 * Graph statistics
 */
export interface GraphStats {
  nodeCount: number;
  edgeCount: number;
  linkTypes: string[];
  averageDegree: number;
}

/**
 * Options for graph export
 */
export interface GraphExportOptions {
  format: 'json' | 'd3' | 'cytoscape';
  includeAttributes?: boolean;
  linkTypes?: string[];
}

/**
 * D3.js compatible format
 */
export interface D3GraphData {
  nodes: Array<{ id: string; [key: string]: any }>;
  links: Array<{
    source: string;
    target: string;
    type: string;
    [key: string]: any;
  }>;
}

/**
 * Cytoscape.js compatible format
 */
export interface CytoscapeElement {
  data: {
    id: string;
    source?: string;
    target?: string;
    [key: string]: any;
  };
}

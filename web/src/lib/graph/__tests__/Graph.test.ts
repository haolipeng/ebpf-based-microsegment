/**
 * Graph Database Tests
 *
 * Unit tests for the graph database implementation
 */

import { vi } from 'vitest';
import { Graph } from '../Graph';
import {
  findConnectedNodes,
  findConnectedComponents,
  findShortestPath,
  calculateDegreeCentrality,
  findHubs,
  hasCycle,
  topologicalSort,
} from '../algorithms';

describe('Graph', () => {
  let graph: Graph<string>;

  beforeEach(() => {
    graph = new Graph<string>();
  });

  describe('Basic Operations', () => {
    test('should add nodes and edges', () => {
      graph.addLink('A', 'traffic', 'B', 'edge1');
      graph.addLink('B', 'traffic', 'C', 'edge2');

      expect(graph.hasNode('A')).toBe(true);
      expect(graph.hasNode('B')).toBe(true);
      expect(graph.hasNode('C')).toBe(true);
      expect(graph.getNodeCount()).toBe(3);
      expect(graph.getEdgeCount()).toBe(2);
    });

    test('should get edge attributes', () => {
      graph.addLink('A', 'traffic', 'B', 'attr1');

      expect(graph.attr('A', 'traffic', 'B')).toBe('attr1');
      expect(graph.attr('B', 'traffic', 'A')).toBeUndefined();
    });

    test('should update edge attributes', () => {
      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.addLink('A', 'traffic', 'B', 'attr2');

      expect(graph.attr('A', 'traffic', 'B')).toBe('attr2');
    });

    test('should delete edges', () => {
      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.deleteLink('A', 'traffic', 'B');

      expect(graph.attr('A', 'traffic', 'B')).toBeUndefined();
      expect(graph.hasNode('A')).toBe(true); // Nodes should still exist
      expect(graph.hasNode('B')).toBe(true);
    });

    test('should delete nodes and cascade edges', () => {
      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.addLink('B', 'traffic', 'C', 'attr2');
      graph.addLink('C', 'traffic', 'A', 'attr3');

      graph.deleteNode('B');

      expect(graph.hasNode('B')).toBe(false);
      expect(graph.attr('A', 'traffic', 'B')).toBeUndefined();
      expect(graph.attr('B', 'traffic', 'C')).toBeUndefined();
      expect(graph.getNodeCount()).toBe(2);
      expect(graph.getEdgeCount()).toBe(1);
    });
  });

  describe('Neighbor Queries', () => {
    beforeEach(() => {
      // Build a small graph: A -> B -> C
      //                       A -> D
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('A', 'traffic', 'D', 'e3');
    });

    test('should get outgoing neighbors', () => {
      const neighbors = graph.outs('A', 'traffic');
      expect(neighbors).toHaveLength(2);
      expect(neighbors).toContain('B');
      expect(neighbors).toContain('D');
    });

    test('should get incoming neighbors', () => {
      const neighbors = graph.ins('B', 'traffic');
      expect(neighbors).toHaveLength(1);
      expect(neighbors).toContain('A');
    });

    test('should get all neighbors', () => {
      const neighbors = graph.both('B', 'traffic');
      expect(neighbors).toHaveLength(2);
      expect(neighbors).toContain('A');
      expect(neighbors).toContain('C');
    });

    test('should handle non-existent nodes', () => {
      expect(graph.outs('X', 'traffic')).toEqual([]);
      expect(graph.ins('X', 'traffic')).toEqual([]);
      expect(graph.both('X', 'traffic')).toEqual([]);
    });
  });

  describe('Multiple Link Types', () => {
    test('should support multiple link types between same nodes', () => {
      graph.addLink('A', 'traffic', 'B', 'traffic-attr');
      graph.addLink('A', 'policy', 'B', 'policy-attr');

      expect(graph.attr('A', 'traffic', 'B')).toBe('traffic-attr');
      expect(graph.attr('A', 'policy', 'B')).toBe('policy-attr');
    });

    test('should filter neighbors by link type', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('A', 'policy', 'C', 'e2');

      expect(graph.outs('A', 'traffic')).toEqual(['B']);
      expect(graph.outs('A', 'policy')).toEqual(['C']);
    });

    test('should get all neighbors across link types', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('A', 'policy', 'C', 'e2');

      const neighbors = graph.outs('A');
      expect(neighbors).toHaveLength(2);
      expect(neighbors).toContain('B');
      expect(neighbors).toContain('C');
    });
  });

  describe('Statistics', () => {
    beforeEach(() => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('C', 'traffic', 'A', 'e3');
      graph.addLink('A', 'policy', 'D', 'e4');
    });

    test('should get correct node count', () => {
      expect(graph.getNodeCount()).toBe(4);
    });

    test('should get correct edge count', () => {
      expect(graph.getEdgeCount()).toBe(4);
    });

    test('should get all link types', () => {
      const linkTypes = graph.getLinkTypes();
      expect(linkTypes).toHaveLength(2);
      expect(linkTypes).toContain('traffic');
      expect(linkTypes).toContain('policy');
    });

    test('should get graph statistics', () => {
      const stats = graph.getStats();
      expect(stats.nodeCount).toBe(4);
      expect(stats.edgeCount).toBe(4);
      expect(stats.linkTypes).toHaveLength(2);
      expect(stats.averageDegree).toBe(2); // 8 total degree / 4 nodes
    });
  });

  describe('Export', () => {
    beforeEach(() => {
      graph.addLink('A', 'traffic', 'B', { weight: 10 });
      graph.addLink('B', 'traffic', 'C', { weight: 20 });
    });

    test('should export as JSON', () => {
      const data = graph.export({ format: 'json', includeAttributes: true });

      expect(data.nodes).toHaveLength(3);
      expect(data.edges).toHaveLength(2);
      expect(data.edges[0].attributes).toEqual({ weight: 10 });
    });

    test('should export as D3 format', () => {
      const data = graph.export({ format: 'd3', includeAttributes: false });

      expect(data.nodes).toHaveLength(3);
      expect(data.links).toHaveLength(2);
      expect(data.links[0].source).toBe('A');
      expect(data.links[0].target).toBe('B');
    });

    test('should export as Cytoscape format', () => {
      const elements = graph.export({ format: 'cytoscape', includeAttributes: true });

      // 3 nodes + 2 edges = 5 elements
      expect(elements).toHaveLength(5);

      const nodes = elements.filter(e => !e.data.source);
      const edges = elements.filter(e => e.data.source);

      expect(nodes).toHaveLength(3);
      expect(edges).toHaveLength(2);
    });
  });

  describe('Callbacks', () => {
    test('should trigger new link callback', () => {
      const callback = vi.fn();
      graph.registerNewLinkCallback(callback);

      graph.addLink('A', 'traffic', 'B', 'attr1');
      expect(callback).toHaveBeenCalledWith('A', 'traffic', 'B');
    });

    test('should trigger update link attribute callback', () => {
      const callback = vi.fn();
      graph.registerUpdateLinkAttrCallback(callback);

      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.addLink('A', 'traffic', 'B', 'attr2'); // Update

      expect(callback).toHaveBeenCalledWith('A', 'traffic', 'B');
    });

    test('should trigger delete link callback', () => {
      const callback = vi.fn();
      graph.registerDelLinkCallback(callback);

      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.deleteLink('A', 'traffic', 'B');

      expect(callback).toHaveBeenCalledWith('A', 'traffic', 'B');
    });

    test('should trigger delete node callback', () => {
      const callback = vi.fn();
      graph.registerDelNodeCallback(callback);

      graph.addLink('A', 'traffic', 'B', 'attr1');
      graph.deleteNode('A');

      expect(callback).toHaveBeenCalledWith('A');
    });
  });
});

describe('Graph Algorithms', () => {
  let graph: Graph<string>;

  beforeEach(() => {
    graph = new Graph<string>();
  });

  describe('Connected Components', () => {
    test('should find connected nodes', () => {
      // Build: A - B - C    D - E
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('D', 'traffic', 'E', 'e3');

      const connected = findConnectedNodes(graph, 'A', 'traffic');
      expect(connected.size).toBe(3);
      expect(connected.has('A')).toBe(true);
      expect(connected.has('B')).toBe(true);
      expect(connected.has('C')).toBe(true);
      expect(connected.has('D')).toBe(false);
    });

    test('should find all connected components', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('D', 'traffic', 'E', 'e3');

      const components = findConnectedComponents(graph, 'traffic');
      expect(components).toHaveLength(2);

      const sizes = components.map(c => c.size).sort();
      expect(sizes).toEqual([2, 3]);
    });
  });

  describe('Shortest Path', () => {
    test('should find shortest path', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('A', 'traffic', 'D', 'e3');
      graph.addLink('D', 'traffic', 'C', 'e4');

      const path = findShortestPath(graph, 'A', 'C', 'traffic');
      expect(path).toEqual(['A', 'B', 'C']);
    });

    test('should return empty array for unreachable nodes', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('C', 'traffic', 'D', 'e2');

      const path = findShortestPath(graph, 'A', 'D', 'traffic');
      expect(path).toEqual([]);
    });
  });

  describe('Centrality', () => {
    test('should calculate degree centrality', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('A', 'traffic', 'C', 'e2');
      graph.addLink('B', 'traffic', 'C', 'e3');

      const centrality = calculateDegreeCentrality(graph, 'traffic', 'both');

      expect(centrality.get('A')).toBe(2); // 2 outgoing
      expect(centrality.get('B')).toBe(2); // 1 in + 1 out
      expect(centrality.get('C')).toBe(2); // 2 incoming
    });

    test('should find traffic hubs', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('A', 'traffic', 'C', 'e2');
      graph.addLink('A', 'traffic', 'D', 'e3');
      graph.addLink('B', 'traffic', 'C', 'e4');

      const hubs = findHubs(graph, 3, 'traffic', 'both');

      expect(hubs).toHaveLength(3);
      expect(hubs[0][0]).toBe('A'); // A has highest degree
      expect(hubs[0][1]).toBe(3);
    });
  });

  describe('Cycle Detection', () => {
    test('should detect cycles', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('C', 'traffic', 'A', 'e3'); // Creates cycle

      expect(hasCycle(graph, 'traffic')).toBe(true);
    });

    test('should detect no cycle in DAG', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'C', 'e2');
      graph.addLink('A', 'traffic', 'C', 'e3');

      expect(hasCycle(graph, 'traffic')).toBe(false);
    });
  });

  describe('Topological Sort', () => {
    test('should perform topological sort on DAG', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('A', 'traffic', 'C', 'e2');
      graph.addLink('B', 'traffic', 'D', 'e3');
      graph.addLink('C', 'traffic', 'D', 'e4');

      const sorted = topologicalSort(graph, 'traffic');

      expect(sorted).toHaveLength(4);
      expect(sorted.indexOf('A')).toBeLessThan(sorted.indexOf('B'));
      expect(sorted.indexOf('A')).toBeLessThan(sorted.indexOf('C'));
      expect(sorted.indexOf('B')).toBeLessThan(sorted.indexOf('D'));
      expect(sorted.indexOf('C')).toBeLessThan(sorted.indexOf('D'));
    });

    test('should return empty array for cyclic graph', () => {
      graph.addLink('A', 'traffic', 'B', 'e1');
      graph.addLink('B', 'traffic', 'A', 'e2');

      const sorted = topologicalSort(graph, 'traffic');
      expect(sorted).toEqual([]);
    });
  });
});

import { describe, it, expect } from 'vitest'
import {
  aggregateFlowsToTopology,
  calculateNodeSize,
  calculateEdgeWidth,
  mergeTopologyUpdate,
  getNodeLabel,
} from './topologyUtils'
import type { Flow } from '../types/flow'
import type { TopologyData, TopologyNode } from '../types/topology'

// Mock Flow data factory
const createMockFlow = (overrides?: Partial<Flow>): Flow => ({
  id: 'flow-1',
  sourceIp: '192.168.1.1',
  destIp: '192.168.1.2',
  sourcePort: 8080,
  destPort: 80,
  protocol: 'TCP',
  state: 'ACTIVE',
  direction: 'EGRESS',
  packetCount: 100,
  byteCount: 10000,
  startTime: new Date().toISOString(),
  lastSeen: new Date().toISOString(),
  durationMs: 0,
  policyAction: 'ALLOW',
  eventType: 'NEW',
  sourceLabels: { app: 'web', env: 'prod' },
  destLabels: { app: 'db', env: 'prod' },
  ...overrides,
})

describe('topologyUtils', () => {
  describe('aggregateFlowsToTopology - IP View', () => {
    it('should aggregate single flow into topology data', () => {
      const flows = [createMockFlow()]
      const result = aggregateFlowsToTopology(flows, 'IP')

      expect(result.nodes).toHaveLength(2)
      expect(result.edges).toHaveLength(1)
      expect(result.stats.totalFlows).toBe(1)
    })

    it('should create nodes with correct IDs and types', () => {
      const flows = [createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2' })]
      const result = aggregateFlowsToTopology(flows, 'IP')

      const sourceNode = result.nodes.find(n => n.id === '10.0.0.1')
      const targetNode = result.nodes.find(n => n.id === '10.0.0.2')

      expect(sourceNode).toBeDefined()
      expect(targetNode).toBeDefined()
      expect(sourceNode?.type).toBe('IP')
      expect(targetNode?.type).toBe('IP')
    })

    it('should aggregate metrics correctly for multiple flows', () => {
      const flows = [
        createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2', byteCount: 1000, packetCount: 10 }),
        createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2', byteCount: 2000, packetCount: 20 }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP')

      const edge = result.edges[0]
      expect(edge.metrics.flowCount).toBe(2)
      expect(edge.metrics.byteCount).toBe(3000)
      expect(edge.metrics.packetCount).toBe(30)
    })

    it('should handle multiple protocols in the same edge', () => {
      const flows = [
        createMockFlow({ protocol: 'TCP' }),
        createMockFlow({ protocol: 'UDP' }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP')

      expect(result.edges[0].metrics.protocols).toContain('TCP')
      expect(result.edges[0].metrics.protocols).toContain('UDP')
      expect(result.edges[0].metrics.protocols).toHaveLength(2)
    })

    it('should detect bidirectional traffic', () => {
      const flows = [
        createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2', direction: 'EGRESS' }),
        createMockFlow({ sourceIp: '10.0.0.2', destIp: '10.0.0.1', direction: 'INGRESS' }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP')

      const edge = result.edges.find(e => e.direction === 'BIDIRECTIONAL')
      expect(edge).toBeDefined()
    })

    it('should limit nodes to maxNodes', () => {
      const flows = [
        createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2', byteCount: 1000 }),
        createMockFlow({ sourceIp: '10.0.0.3', destIp: '10.0.0.4', byteCount: 2000 }),
        createMockFlow({ sourceIp: '10.0.0.5', destIp: '10.0.0.6', byteCount: 3000 }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP', 4)

      expect(result.nodes.length).toBeLessThanOrEqual(4)
    })

    it('should filter edges when limiting nodes', () => {
      const flows = [
        createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2', byteCount: 5000 }),
        createMockFlow({ sourceIp: '10.0.0.3', destIp: '10.0.0.4', byteCount: 1000 }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP', 2)

      // Only edges between top 2 nodes should remain
      const nodeIds = new Set(result.nodes.map(n => n.id))
      result.edges.forEach(edge => {
        expect(nodeIds.has(edge.source)).toBe(true)
        expect(nodeIds.has(edge.target)).toBe(true)
      })
    })
  })

  describe('aggregateFlowsToTopology - LABEL View', () => {
    it('should aggregate flows by service labels', () => {
      const flows = [
        createMockFlow({
          sourceLabels: { app: 'web' },
          destLabels: { app: 'api' },
        }),
      ]
      const result = aggregateFlowsToTopology(flows, 'LABEL')

      expect(result.nodes).toHaveLength(2)
      const labels = result.nodes.map(n => n.label)
      // getServiceLabel优先使用app标签,直接返回值(不带前缀)
      expect(labels).toContain('web')
      expect(labels).toContain('api')
    })

    it('should create SERVICE type nodes', () => {
      const flows = [
        createMockFlow({
          sourceLabels: { app: 'frontend' },
          destLabels: { app: 'backend' },
        }),
      ]
      const result = aggregateFlowsToTopology(flows, 'LABEL')

      result.nodes.forEach(node => {
        expect(node.type).toBe('SERVICE')
      })
    })

    it('should skip flows without labels', () => {
      const flows = [
        createMockFlow({ sourceLabels: {}, destLabels: { app: 'test' } }),
        createMockFlow({ sourceLabels: { app: 'valid' }, destLabels: { app: 'valid2' } }),
      ]
      const result = aggregateFlowsToTopology(flows, 'LABEL')

      // Only the second flow should be included
      expect(result.stats.totalFlows).toBe(2)
      expect(result.nodes.length).toBe(2) // valid and valid2
    })

    it('should aggregate multiple flows to same service', () => {
      const flows = [
        createMockFlow({ sourceLabels: { app: 'web' }, destLabels: { app: 'db' }, byteCount: 1000 }),
        createMockFlow({ sourceLabels: { app: 'web' }, destLabels: { app: 'db' }, byteCount: 2000 }),
      ]
      const result = aggregateFlowsToTopology(flows, 'LABEL')

      expect(result.nodes).toHaveLength(2)
      const edge = result.edges[0]
      expect(edge.metrics.flowCount).toBe(2)
      expect(edge.metrics.byteCount).toBe(3000)
    })
  })

  describe('calculateNodeSize', () => {
    it('should return minimum size for zero flow count', () => {
      const size = calculateNodeSize(0)
      expect(size).toBe(20)
    })

    it('should return value within range for normal counts', () => {
      const size = calculateNodeSize(100)
      expect(size).toBeGreaterThanOrEqual(20)
      expect(size).toBeLessThanOrEqual(80)
    })

    it('should use logarithmic scaling', () => {
      const size5 = calculateNodeSize(5)
      const size10 = calculateNodeSize(10)
      const size50 = calculateNodeSize(50)
      const size100 = calculateNodeSize(100)

      // Logarithmic scaling means equal ratios produce equal differences
      // 10/5 = 2, 100/50 = 2, so differences should be approximately equal
      const diff1 = size10 - size5
      const diff2 = size100 - size50

      expect(Math.abs(diff1 - diff2)).toBeLessThan(0.1)
    })

    it('should return maximum size for very large counts', () => {
      const size = calculateNodeSize(1000000)
      expect(size).toBe(80)
    })
  })

  describe('calculateEdgeWidth', () => {
    it('should return minimum width for zero bytes', () => {
      const width = calculateEdgeWidth(0)
      expect(width).toBe(1)
    })

    it('should return value within range for normal byte counts', () => {
      const width = calculateEdgeWidth(100000)
      expect(width).toBeGreaterThanOrEqual(1)
      expect(width).toBeLessThanOrEqual(10)
    })

    it('should use logarithmic scaling', () => {
      const width1k = calculateEdgeWidth(1000)
      const width1m = calculateEdgeWidth(1000000)
      const width1g = calculateEdgeWidth(1000000000)

      expect(width1m).toBeGreaterThan(width1k)
      expect(width1g).toBeGreaterThan(width1m)
    })

    it('should return maximum width for very large byte counts', () => {
      const width = calculateEdgeWidth(10000000000)
      expect(width).toBe(10)
    })
  })

  describe('mergeTopologyUpdate - IP View', () => {
    it('should add new nodes for new flow', () => {
      const existing: TopologyData = {
        nodes: [],
        edges: [],
        stats: { totalNodes: 0, totalEdges: 0, totalFlows: 0 },
      }
      const newFlow = createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.2' })

      const result = mergeTopologyUpdate(existing, newFlow, 'IP')

      expect(result.nodes).toHaveLength(2)
      expect(result.edges).toHaveLength(1)
    })

    it('should update existing node metrics', () => {
      const existing: TopologyData = {
        nodes: [
          {
            id: '10.0.0.1',
            label: '10.0.0.1',
            type: 'IP',
            metrics: { flowCount: 1, packetCount: 10, byteCount: 1000, activeFlows: 1 },
          },
          {
            id: '10.0.0.2',
            label: '10.0.0.2',
            type: 'IP',
            metrics: { flowCount: 1, packetCount: 10, byteCount: 1000, activeFlows: 1 },
          },
        ],
        edges: [],
        stats: { totalNodes: 2, totalEdges: 0, totalFlows: 1 },
      }
      const newFlow = createMockFlow({
        sourceIp: '10.0.0.1',
        destIp: '10.0.0.2',
        packetCount: 50,
        byteCount: 5000,
      })

      const result = mergeTopologyUpdate(existing, newFlow, 'IP')

      const sourceNode = result.nodes.find(n => n.id === '10.0.0.1')
      expect(sourceNode?.metrics.flowCount).toBe(2)
      expect(sourceNode?.metrics.packetCount).toBe(60)
      expect(sourceNode?.metrics.byteCount).toBe(6000)
    })

    it('should add protocols to existing edge', () => {
      const existing: TopologyData = {
        nodes: [
          {
            id: '10.0.0.1',
            label: '10.0.0.1',
            type: 'IP',
            metrics: { flowCount: 0, packetCount: 0, byteCount: 0, activeFlows: 0 },
          },
          {
            id: '10.0.0.2',
            label: '10.0.0.2',
            type: 'IP',
            metrics: { flowCount: 0, packetCount: 0, byteCount: 0, activeFlows: 0 },
          },
        ],
        edges: [
          {
            id: '10.0.0.1->10.0.0.2',
            source: '10.0.0.1',
            target: '10.0.0.2',
            metrics: { flowCount: 1, packetCount: 10, byteCount: 1000, protocols: ['TCP'] },
            direction: 'EGRESS',
          },
        ],
        stats: { totalNodes: 2, totalEdges: 1, totalFlows: 1 },
      }
      const newFlow = createMockFlow({
        sourceIp: '10.0.0.1',
        destIp: '10.0.0.2',
        protocol: 'UDP',
      })

      const result = mergeTopologyUpdate(existing, newFlow, 'IP')

      const edge = result.edges[0]
      expect(edge.metrics.protocols).toContain('TCP')
      expect(edge.metrics.protocols).toContain('UDP')
    })
  })

  describe('mergeTopologyUpdate - LABEL View', () => {
    it('should add new SERVICE nodes for new flow', () => {
      const existing: TopologyData = {
        nodes: [],
        edges: [],
        stats: { totalNodes: 0, totalEdges: 0, totalFlows: 0 },
      }
      const newFlow = createMockFlow({
        sourceLabels: { app: 'web' },
        destLabels: { app: 'api' },
      })

      const result = mergeTopologyUpdate(existing, newFlow, 'LABEL')

      expect(result.nodes).toHaveLength(2)
      expect(result.nodes.every(n => n.type === 'SERVICE')).toBe(true)
    })

    it('should skip flows without labels', () => {
      const existing: TopologyData = {
        nodes: [],
        edges: [],
        stats: { totalNodes: 0, totalEdges: 0, totalFlows: 0 },
      }
      const newFlow = createMockFlow({
        sourceLabels: {},
        destLabels: {},
      })

      const result = mergeTopologyUpdate(existing, newFlow, 'LABEL')

      expect(result.nodes).toHaveLength(0)
      expect(result.edges).toHaveLength(0)
    })

    it('should update existing SERVICE node metrics', () => {
      const existing: TopologyData = {
        nodes: [
          {
            id: 'web',
            label: 'web',
            type: 'SERVICE',
            metrics: { flowCount: 1, packetCount: 10, byteCount: 1000, activeFlows: 1 },
            labels: { app: 'web' },
          },
        ],
        edges: [],
        stats: { totalNodes: 1, totalEdges: 0, totalFlows: 1 },
      }
      const newFlow = createMockFlow({
        sourceLabels: { app: 'web' },
        destLabels: { app: 'api' },
        packetCount: 50,
        byteCount: 5000,
      })

      const result = mergeTopologyUpdate(existing, newFlow, 'LABEL')

      const webNode = result.nodes.find(n => n.id === 'web')
      expect(webNode?.metrics.flowCount).toBe(2)
      expect(webNode?.metrics.packetCount).toBe(60)
      expect(webNode?.metrics.byteCount).toBe(6000)
    })
  })

  describe('getNodeLabel', () => {
    it('should return node label', () => {
      const node: TopologyNode = {
        id: '10.0.0.1',
        label: 'My Node',
        type: 'IP',
        metrics: { flowCount: 0, packetCount: 0, byteCount: 0, activeFlows: 0 },
      }

      expect(getNodeLabel(node)).toBe('My Node')
    })

    it('should return label for SERVICE node', () => {
      const node: TopologyNode = {
        id: 'web',
        label: 'web',
        type: 'SERVICE',
        metrics: { flowCount: 0, packetCount: 0, byteCount: 0, activeFlows: 0 },
        labels: { app: 'web' },
      }

      expect(getNodeLabel(node)).toBe('web')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty flow array', () => {
      const result = aggregateFlowsToTopology([], 'IP')

      expect(result.nodes).toHaveLength(0)
      expect(result.edges).toHaveLength(0)
      expect(result.stats.totalFlows).toBe(0)
    })

    it('should handle flows with same source and dest (loopback)', () => {
      const flows = [createMockFlow({ sourceIp: '10.0.0.1', destIp: '10.0.0.1' })]
      const result = aggregateFlowsToTopology(flows, 'IP')

      expect(result.nodes).toHaveLength(1)
      expect(result.edges).toHaveLength(1)
    })

    it('should handle UNKNOWN direction correctly', () => {
      const flows = [createMockFlow({ direction: 'UNKNOWN' })]
      const result = aggregateFlowsToTopology(flows, 'IP')

      expect(result.edges[0].direction).toBe('EGRESS')
    })

    it('should count active flows correctly', () => {
      const flows = [
        createMockFlow({ state: 'ACTIVE' }),
        createMockFlow({ state: 'CLOSED' }),
      ]
      const result = aggregateFlowsToTopology(flows, 'IP')

      const node = result.nodes[0]
      expect(node.metrics.activeFlows).toBe(1)
    })
  })
})

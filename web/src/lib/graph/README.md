# Graph Database Library

A lightweight in-memory graph database for network topology visualization, inspired by NeuVector's implementation.

## Features

- **Directed Multigraph**: Support for multiple edge types between same nodes
- **O(1) Operations**: Fast node and edge queries using bidirectional indexing
- **Generic Attributes**: Flexible edge attribute storage with TypeScript generics
- **Graph Algorithms**: BFS, DFS, connected components, shortest path, centrality
- **Multiple Export Formats**: JSON, D3.js, Cytoscape.js
- **Event Callbacks**: Notifications for graph mutations

## Quick Start

```typescript
import { Graph } from '@/lib/graph';

// Create a graph
const graph = new Graph<{ weight: number }>();

// Add edges (nodes are created automatically)
graph.addLink('A', 'traffic', 'B', { weight: 10 });
graph.addLink('B', 'traffic', 'C', { weight: 20 });

// Query neighbors
const neighbors = graph.outs('A', 'traffic'); // ['B']

// Get edge attribute
const attr = graph.attr('A', 'traffic', 'B'); // { weight: 10 }

// Export to D3 format
const d3Data = graph.export({ format: 'd3' });
```

## Core API

### Graph Class

#### Adding and Removing

```typescript
// Add a directed edge
addLink(src: string, linkType: string, dst: string, attr: TAttr): void

// Delete an edge
deleteLink(src: string, linkType: string, dst: string): void

// Delete a node (cascades to all connected edges)
deleteNode(node: string): void
```

#### Querying

```typescript
// Get edge attribute
attr(src: string, linkType: string, dst: string): TAttr | undefined

// Get incoming neighbors
ins(node: string, linkType?: string): string[]

// Get outgoing neighbors
outs(node: string, linkType?: string): string[]

// Get all neighbors (both directions)
both(node: string, linkType?: string): string[]

// Get all nodes
all(): string[]

// Check if node exists
hasNode(node: string): boolean
```

#### Statistics

```typescript
// Get node count
getNodeCount(): number

// Get edge count
getEdgeCount(): number

// Get all link types
getLinkTypes(): string[]

// Get detailed statistics
getStats(): GraphStats
```

#### Export

```typescript
// Export graph data
export(options: GraphExportOptions): any

// Options:
// - format: 'json' | 'd3' | 'cytoscape'
// - includeAttributes?: boolean
// - linkTypes?: string[]
```

#### Callbacks

```typescript
// Register callbacks for graph events
registerNewLinkCallback(cb: (src, linkType, dst) => void): void
registerDelLinkCallback(cb: (src, linkType, dst) => void): void
registerDelNodeCallback(cb: (node) => void): void
registerUpdateLinkAttrCallback(cb: (src, linkType, dst) => void): void
```

## Graph Algorithms

### Connected Components

```typescript
import { findConnectedNodes, findConnectedComponents } from '@/lib/graph/algorithms';

// Find all nodes connected to 'A'
const connected = findConnectedNodes(graph, 'A', 'traffic');

// Find all connected components
const components = findConnectedComponents(graph, 'traffic');
```

### Shortest Path

```typescript
import { findShortestPath } from '@/lib/graph/algorithms';

const path = findShortestPath(graph, 'A', 'C', 'traffic');
// Returns: ['A', 'B', 'C']
```

### Centrality Analysis

```typescript
import { calculateDegreeCentrality, findHubs } from '@/lib/graph/algorithms';

// Calculate degree centrality for all nodes
const centrality = calculateDegreeCentrality(graph, 'traffic', 'both');

// Find top 10 traffic hubs
const hubs = findHubs(graph, 10, 'traffic', 'both');
// Returns: [['nodeA', 15], ['nodeB', 12], ...]
```

### Neighborhood Analysis

```typescript
import { findNodesWithinHops } from '@/lib/graph/algorithms';

// Find all nodes within 2 hops of 'A'
const neighborhood = findNodesWithinHops(graph, 'A', 2, 'traffic');
// Returns: Map<string, number> (node -> distance)
```

### Cycle Detection

```typescript
import { hasCycle, topologicalSort } from '@/lib/graph/algorithms';

// Check if graph has cycles
const cyclic = hasCycle(graph, 'traffic');

// Topological sort (only for DAGs)
const sorted = topologicalSort(graph, 'traffic');
```

## Topology Integration

### Using with Topology Aggregator

```typescript
import { createTopologyAggregator } from '@/utils/topologyAggregator';
import type { Flow } from '@/types/flow';

const aggregator = createTopologyAggregator();

// Aggregate flows into topology
const topology = aggregator.aggregate(flows, {
  viewMode: 'IP',
  maxNodes: 50,
});

// Access underlying graph
const graph = aggregator.getGraph();

// Find traffic hubs
const hubs = aggregator.findTrafficHubs(10);

// Find neighborhood
const neighborhood = aggregator.findNeighborhood('192.168.1.10', 2);
```

### Using with React Hooks

```typescript
import { useTopologyGraph } from '@/hooks/useTopologyGraph';

function MyComponent({ flows }: { flows: Flow[] }) {
  const {
    data,
    graph,
    findConnectedNodes,
    findTrafficHubs,
    getNodeDetail,
  } = useTopologyGraph({
    flows,
    filters: {
      viewMode: 'IP',
      maxNodes: 20,
    },
  });

  // Use topology data for visualization
  return <TopologyGraph data={data} />;
}
```

## Architecture

### Data Structure

The graph uses a three-layer indexing structure for O(1) operations:

```
Graph
├── nodes: Map<string, GraphNode>
│   └── GraphNode
│       ├── ins: Map<linkType, GraphLink>   // Incoming edges
│       └── outs: Map<linkType, GraphLink>  // Outgoing edges
│           └── GraphLink
│               └── ends: Map<targetNode, attribute>
```

### Example

For a graph with edges:
- A -[traffic]-> B (weight: 10)
- A -[policy]-> C (weight: 20)

Internal structure:

```typescript
{
  nodes: {
    'A': {
      outs: {
        'traffic': { ends: { 'B': { weight: 10 } } },
        'policy': { ends: { 'C': { weight: 20 } } }
      },
      ins: {}
    },
    'B': {
      ins: {
        'traffic': { ends: { 'A': { weight: 10 } } }
      },
      outs: {}
    },
    'C': {
      ins: {
        'policy': { ends: { 'A': { weight: 20 } } }
      },
      outs: {}
    }
  }
}
```

## Performance

- **Add Link**: O(1)
- **Delete Link**: O(1)
- **Delete Node**: O(d) where d is degree of node
- **Get Neighbors**: O(1) for specific link type, O(k) for all types (k = number of link types)
- **BFS Traversal**: O(V + E)
- **Shortest Path**: O(V + E)

## Comparison with NeuVector

This implementation closely follows NeuVector's graph database design:

| Feature | NeuVector (Go) | This Implementation (TypeScript) |
|---------|----------------|----------------------------------|
| Data Structure | map[string]*graphNode | Map<string, GraphNode> |
| Link Types | Multiple supported | Multiple supported |
| Edge Attributes | interface{} | Generic type TAttr |
| Bidirectional Index | ✅ ins/outs | ✅ ins/outs |
| Callbacks | ✅ | ✅ |
| Export Formats | Custom | JSON, D3, Cytoscape |
| Algorithms | Connected components | BFS, DFS, Centrality, etc. |

## Testing

Run tests:

```bash
npm test -- Graph.test.ts
```

Test coverage includes:
- Basic graph operations (add, delete, query)
- Multiple link types
- Neighbor queries
- Graph statistics
- Export formats
- Callbacks
- All graph algorithms

## Examples

See [examples/TopologyExample.tsx](../../examples/TopologyExample.tsx) for a complete integration example with:
- Topology visualization
- Session detail panel
- Graph statistics
- Interactive node selection
- Real-time flow aggregation

## References

- [NeuVector Controller Graph Implementation](https://github.com/neuvector/neuvector/tree/main/controller/graph)
- [Graph Theory](https://en.wikipedia.org/wiki/Graph_theory)
- [Force-Directed Graph Drawing](https://en.wikipedia.org/wiki/Force-directed_graph_drawing)

package topology

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
	assert.NotNil(t, m.graph)
	assert.Equal(t, 0, len(m.nodeAttrs))
	assert.Equal(t, 0, len(m.edgeAttrs))
}

func TestManager_UpdateNode(t *testing.T) {
	m := NewManager()

	// Create a new node
	attr := &NodeAttr{
		Name:      "web-server",
		Type:      NodeTypeWorkload,
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
		IPs:       []string{"10.0.1.10"},
	}
	m.UpdateNode("node1", attr)

	// Verify node was created
	stats := m.GetStats()
	assert.Equal(t, 1, stats.TotalNodes)

	// Update the same node
	updateAttr := &NodeAttr{
		Name:   "web-server-updated",
		Labels: map[string]string{"version": "v2"},
		IPs:    []string{"10.0.1.11"},
	}
	m.UpdateNode("node1", updateAttr)

	// Should still be one node
	stats = m.GetStats()
	assert.Equal(t, 1, stats.TotalNodes)
}

func TestManager_UpdateEdge(t *testing.T) {
	m := NewManager()

	// Create nodes first
	m.UpdateNode("nodeA", &NodeAttr{Name: "A", Type: NodeTypeWorkload})
	m.UpdateNode("nodeB", &NodeAttr{Name: "B", Type: NodeTypeWorkload})

	// Create edge
	edgeAttr := &EdgeAttr{
		Bytes:        1000,
		Sessions:     5,
		PacketCount:  100,
		Protocols:    []string{"TCP"},
		Ports:        []uint16{80},
		PolicyAction: PolicyActionAllow,
	}
	m.UpdateEdge("nodeA", "nodeB", edgeAttr)

	// Verify edge was created
	stats := m.GetStats()
	assert.Equal(t, 2, stats.TotalNodes)
	assert.Equal(t, 1, stats.TotalEdges)

	// Update edge with more data
	updateAttr := &EdgeAttr{
		Bytes:       500,
		Sessions:    2,
		PacketCount: 50,
		Protocols:   []string{"UDP"},
		Ports:       []uint16{443},
	}
	m.UpdateEdge("nodeA", "nodeB", updateAttr)

	// Should still be one edge with accumulated values
	stats = m.GetStats()
	assert.Equal(t, 1, stats.TotalEdges)

	// Check accumulated bytes
	detail := m.GetEdgeDetail("nodeA", "nodeB")
	require.NotNil(t, detail)
	assert.Equal(t, uint64(1500), detail.Bytes) // 1000 + 500
	assert.Equal(t, uint32(7), detail.Sessions) // 5 + 2
	assert.Contains(t, detail.Protocols, "TCP")
	assert.Contains(t, detail.Protocols, "UDP")
}

func TestManager_GetTopology(t *testing.T) {
	m := NewManager()

	// Create a small topology
	m.UpdateNode("web", &NodeAttr{Name: "web", Type: NodeTypeWorkload, Namespace: "prod"})
	m.UpdateNode("db", &NodeAttr{Name: "db", Type: NodeTypeWorkload, Namespace: "prod"})
	m.UpdateNode("external", &NodeAttr{Name: "external", Type: NodeTypeExternal})

	m.UpdateEdge("web", "db", &EdgeAttr{Bytes: 1000, Sessions: 10})
	m.UpdateEdge("external", "web", &EdgeAttr{Bytes: 5000, Sessions: 50})

	// Get full topology
	topo := m.GetTopology(nil)
	assert.Equal(t, 3, topo.Metadata.TotalNodes)
	assert.Equal(t, 2, topo.Metadata.TotalEdges)

	// Filter by namespace
	topo = m.GetTopology(&TopologyFilter{Namespace: "prod"})
	assert.Equal(t, 2, len(topo.Nodes))

	// Exclude external nodes
	topo = m.GetTopology(&TopologyFilter{IncludeExternal: false})
	nodeTypes := make([]NodeType, 0)
	for _, n := range topo.Nodes {
		nodeTypes = append(nodeTypes, n.Type)
	}
	assert.NotContains(t, nodeTypes, NodeTypeExternal)
}

func TestManager_GetNodeDetail(t *testing.T) {
	m := NewManager()

	m.UpdateNode("web", &NodeAttr{Name: "web", Type: NodeTypeWorkload})
	m.UpdateNode("db", &NodeAttr{Name: "db", Type: NodeTypeWorkload})
	m.UpdateNode("cache", &NodeAttr{Name: "cache", Type: NodeTypeWorkload})

	m.UpdateEdge("web", "db", &EdgeAttr{Bytes: 1000})
	m.UpdateEdge("web", "cache", &EdgeAttr{Bytes: 500})
	m.UpdateEdge("cache", "web", &EdgeAttr{Bytes: 200})

	// Get web node detail
	detail := m.GetNodeDetail("web")
	require.NotNil(t, detail)
	assert.Equal(t, "web", detail.Name)
	assert.Len(t, detail.OutboundEdges, 2)  // web -> db, web -> cache
	assert.Len(t, detail.InboundEdges, 1)   // cache -> web

	// Non-existent node
	detail = m.GetNodeDetail("nonexistent")
	assert.Nil(t, detail)
}

func TestManager_GetEdgeDetail(t *testing.T) {
	m := NewManager()

	m.UpdateNode("src", &NodeAttr{Name: "src", Type: NodeTypeWorkload, Namespace: "ns1"})
	m.UpdateNode("dst", &NodeAttr{Name: "dst", Type: NodeTypeWorkload, Namespace: "ns2"})
	m.UpdateEdge("src", "dst", &EdgeAttr{
		Bytes:        1000,
		Sessions:     10,
		Protocols:    []string{"TCP"},
		Ports:        []uint16{80, 443},
		PolicyAction: PolicyActionAllow,
	})

	detail := m.GetEdgeDetail("src", "dst")
	require.NotNil(t, detail)
	assert.Equal(t, "src", detail.Source)
	assert.Equal(t, "dst", detail.Target)
	assert.Equal(t, uint64(1000), detail.Bytes)
	assert.NotNil(t, detail.SourceNode)
	assert.NotNil(t, detail.TargetNode)
	assert.Equal(t, "ns1", detail.SourceNode.Namespace)
	assert.Equal(t, "ns2", detail.TargetNode.Namespace)

	// Non-existent edge
	detail = m.GetEdgeDetail("src", "nonexistent")
	assert.Nil(t, detail)
}

func TestManager_CleanupStale(t *testing.T) {
	m := NewManager()
	m.SetCleanupTTL(100 * time.Millisecond)

	// Create nodes and edges
	m.UpdateNode("node1", &NodeAttr{Name: "node1", Type: NodeTypeWorkload})
	m.UpdateNode("node2", &NodeAttr{Name: "node2", Type: NodeTypeWorkload})
	m.UpdateEdge("node1", "node2", &EdgeAttr{Bytes: 1000})

	assert.Equal(t, 2, m.GetStats().TotalNodes)
	assert.Equal(t, 1, m.GetStats().TotalEdges)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	removed := m.CleanupStale()
	assert.Greater(t, removed, 0)
}

func TestManager_Reset(t *testing.T) {
	m := NewManager()

	m.UpdateNode("node1", &NodeAttr{Name: "node1", Type: NodeTypeWorkload})
	m.UpdateNode("node2", &NodeAttr{Name: "node2", Type: NodeTypeWorkload})
	m.UpdateEdge("node1", "node2", &EdgeAttr{Bytes: 1000})

	assert.Equal(t, 2, m.GetStats().TotalNodes)
	assert.Equal(t, 1, m.GetStats().TotalEdges)

	m.Reset()

	assert.Equal(t, 0, m.GetStats().TotalNodes)
	assert.Equal(t, 0, m.GetStats().TotalEdges)
}

func TestManager_NodeStats(t *testing.T) {
	m := NewManager()

	m.UpdateNode("web", &NodeAttr{Name: "web", Type: NodeTypeWorkload})
	m.UpdateNode("db1", &NodeAttr{Name: "db1", Type: NodeTypeWorkload})
	m.UpdateNode("db2", &NodeAttr{Name: "db2", Type: NodeTypeWorkload})

	m.UpdateEdge("web", "db1", &EdgeAttr{Bytes: 1000, PacketCount: 100})
	m.UpdateEdge("web", "db2", &EdgeAttr{Bytes: 2000, PacketCount: 200})
	m.UpdateEdge("db1", "web", &EdgeAttr{Bytes: 500, PacketCount: 50})

	// Get topology with stats
	topo := m.GetTopology(nil)

	// Find web node
	var webNode *TopologyNode
	for _, n := range topo.Nodes {
		if n.ID == "web" {
			webNode = n
			break
		}
	}

	require.NotNil(t, webNode)
	require.NotNil(t, webNode.Stats)
	assert.Equal(t, 2, webNode.Stats.OutboundConnections)
	assert.Equal(t, 1, webNode.Stats.InboundConnections)
	assert.Equal(t, uint64(3000), webNode.Stats.TotalBytesOut)  // 1000 + 2000
	assert.Equal(t, uint64(500), webNode.Stats.TotalBytesIn)    // 500
}

func TestManager_FilterByLabels(t *testing.T) {
	m := NewManager()

	m.UpdateNode("web1", &NodeAttr{
		Name:   "web1",
		Type:   NodeTypeWorkload,
		Labels: map[string]string{"app": "web", "env": "prod"},
	})
	m.UpdateNode("web2", &NodeAttr{
		Name:   "web2",
		Type:   NodeTypeWorkload,
		Labels: map[string]string{"app": "web", "env": "staging"},
	})
	m.UpdateNode("db", &NodeAttr{
		Name:   "db",
		Type:   NodeTypeWorkload,
		Labels: map[string]string{"app": "db", "env": "prod"},
	})

	// Filter by env=prod
	topo := m.GetTopology(&TopologyFilter{
		Labels: map[string]string{"env": "prod"},
	})
	assert.Equal(t, 2, len(topo.Nodes))

	// Filter by app=web
	topo = m.GetTopology(&TopologyFilter{
		Labels: map[string]string{"app": "web"},
	})
	assert.Equal(t, 2, len(topo.Nodes))
}

func TestManager_PolicyActionPriority(t *testing.T) {
	m := NewManager()

	m.UpdateNode("src", &NodeAttr{Name: "src", Type: NodeTypeWorkload})
	m.UpdateNode("dst", &NodeAttr{Name: "dst", Type: NodeTypeWorkload})

	// First update with allow
	m.UpdateEdge("src", "dst", &EdgeAttr{
		Bytes:        1000,
		PolicyAction: PolicyActionAllow,
	})

	detail := m.GetEdgeDetail("src", "dst")
	assert.Equal(t, PolicyActionAllow, detail.PolicyAction)

	// Update with deny - should take priority
	m.UpdateEdge("src", "dst", &EdgeAttr{
		Bytes:        500,
		PolicyAction: PolicyActionDeny,
	})

	detail = m.GetEdgeDetail("src", "dst")
	assert.Equal(t, PolicyActionDeny, detail.PolicyAction)

	// Further updates with allow should not change it
	m.UpdateEdge("src", "dst", &EdgeAttr{
		Bytes:        200,
		PolicyAction: PolicyActionAllow,
	})

	detail = m.GetEdgeDetail("src", "dst")
	assert.Equal(t, PolicyActionDeny, detail.PolicyAction)
}

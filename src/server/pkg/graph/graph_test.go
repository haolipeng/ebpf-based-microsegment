package graph

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	assert.NotNil(t, g)
	assert.Equal(t, 0, g.GetNodeCount())
}

func TestAddLink(t *testing.T) {
	g := NewGraph()

	// Add a simple link
	g.AddLink("A", "flow", "B", map[string]interface{}{"bytes": 100})

	// Verify nodes were created
	assert.Equal(t, 2, g.GetNodeCount())
	assert.Equal(t, "A", g.Node("A"))
	assert.Equal(t, "B", g.Node("B"))

	// Verify edge attribute
	attr := g.Attr("A", "flow", "B")
	assert.NotNil(t, attr)
	attrMap := attr.(map[string]interface{})
	assert.Equal(t, 100, attrMap["bytes"])
}

func TestAddLink_UpdateAttribute(t *testing.T) {
	g := NewGraph()

	// Track attribute updates
	var updatedSrc, updatedLink, updatedDst string
	g.RegisterUpdateLinkAttrHook(func(src, link, dst string) {
		updatedSrc = src
		updatedLink = link
		updatedDst = dst
	})

	// Add initial link
	g.AddLink("A", "flow", "B", map[string]interface{}{"bytes": 100})

	// Update with same attribute - should not trigger callback
	g.AddLink("A", "flow", "B", map[string]interface{}{"bytes": 100})
	assert.Empty(t, updatedSrc)

	// Update with different attribute - should trigger callback
	g.AddLink("A", "flow", "B", map[string]interface{}{"bytes": 200})
	assert.Equal(t, "A", updatedSrc)
	assert.Equal(t, "flow", updatedLink)
	assert.Equal(t, "B", updatedDst)

	// Verify updated attribute
	attr := g.Attr("A", "flow", "B").(map[string]interface{})
	assert.Equal(t, 200, attr["bytes"])
}

func TestAddLink_NewLinkCallback(t *testing.T) {
	g := NewGraph()

	var newLinks []string
	g.RegisterNewLinkHook(func(src, link, dst string) {
		newLinks = append(newLinks, src+"-"+link+"-"+dst)
	})

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("B", "flow", "C", nil)

	assert.Len(t, newLinks, 2)
	assert.Contains(t, newLinks, "A-flow-B")
	assert.Contains(t, newLinks, "B-flow-C")
}

func TestDeleteLink(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", "flow-attr")
	g.AddLink("A", "policy", "B", "policy-attr")

	// Delete one link type
	g.DeleteLink("A", "flow", "B")

	// flow link should be gone, policy should remain
	assert.Nil(t, g.Attr("A", "flow", "B"))
	assert.Equal(t, "policy-attr", g.Attr("A", "policy", "B"))

	// Nodes should still exist
	assert.Equal(t, 2, g.GetNodeCount())
}

func TestDeleteNode(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", "A-B")
	g.AddLink("B", "flow", "C", "B-C")
	g.AddLink("A", "flow", "C", "A-C")

	// Delete B - should remove edges to/from B
	g.DeleteNode("B")

	assert.Equal(t, 2, g.GetNodeCount())
	assert.Empty(t, g.Node("B"))
	assert.Equal(t, "A-C", g.Attr("A", "flow", "C"))
}

func TestIns_Outs(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("A", "flow", "C", nil)
	g.AddLink("D", "flow", "A", nil)

	// Test Outs
	outs := g.Outs("A")
	assert.Equal(t, 2, outs.Cardinality())
	assert.True(t, outs.Contains("B"))
	assert.True(t, outs.Contains("C"))

	// Test Ins
	ins := g.Ins("A")
	assert.Equal(t, 1, ins.Cardinality())
	assert.True(t, ins.Contains("D"))

	// Test Both
	both := g.Both("A")
	assert.Equal(t, 3, both.Cardinality())
}

func TestInsByLink_OutsByLink(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("A", "policy", "C", nil)
	g.AddLink("D", "flow", "A", nil)

	// Should only return nodes connected by "flow" link
	outsFlow := g.OutsByLink("A", "flow")
	assert.Equal(t, 1, outsFlow.Cardinality())
	assert.True(t, outsFlow.Contains("B"))

	outsPolicy := g.OutsByLink("A", "policy")
	assert.Equal(t, 1, outsPolicy.Cardinality())
	assert.True(t, outsPolicy.Contains("C"))
}

func TestMultipleLinkTypes(t *testing.T) {
	g := NewGraph()

	// Same nodes can have multiple link types
	g.AddLink("A", "flow", "B", map[string]int{"bytes": 100})
	g.AddLink("A", "policy", "B", map[string]string{"action": "allow"})
	g.AddLink("A", "attr", "B", map[string]bool{"external": false})

	// Each link type should have its own attribute
	flowAttr := g.Attr("A", "flow", "B").(map[string]int)
	assert.Equal(t, 100, flowAttr["bytes"])

	policyAttr := g.Attr("A", "policy", "B").(map[string]string)
	assert.Equal(t, "allow", policyAttr["action"])

	attrAttr := g.Attr("A", "attr", "B").(map[string]bool)
	assert.False(t, attrAttr["external"])
}

func TestBetweenDirLinks(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", 100)
	g.AddLink("A", "policy", "B", "allow")

	links := g.BetweenDirLinks("A", "B")
	assert.Len(t, links, 2)
	assert.Equal(t, 100, links["flow"])
	assert.Equal(t, "allow", links["policy"])
}

func TestAll(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("C", "flow", "D", nil)

	all := g.All()
	assert.Equal(t, 4, all.Cardinality())
	assert.True(t, all.Contains("A"))
	assert.True(t, all.Contains("B"))
	assert.True(t, all.Contains("C"))
	assert.True(t, all.Contains("D"))
}

func TestNoIn_NoOut(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("B", "flow", "C", nil)

	// A has no incoming edges
	noIn := g.NoIn()
	assert.True(t, noIn.Contains("A"))
	assert.False(t, noIn.Contains("B"))

	// C has no outgoing edges
	noOut := g.NoOut()
	assert.True(t, noOut.Contains("C"))
	assert.False(t, noOut.Contains("B"))
}

func TestConnected(t *testing.T) {
	g := NewGraph()

	// Create two disconnected components
	g.AddLink("A", "flow", "B", nil)
	g.AddLink("B", "flow", "C", nil)
	g.AddLink("D", "flow", "E", nil)

	// All nodes reachable from A
	connected := g.Connected("A", func(node string) bool {
		return true
	})
	assert.Equal(t, 3, connected.Cardinality())
	assert.True(t, connected.Contains("A"))
	assert.True(t, connected.Contains("B"))
	assert.True(t, connected.Contains("C"))
	assert.False(t, connected.Contains("D"))
}

func TestReset(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("C", "flow", "D", nil)
	assert.Equal(t, 4, g.GetNodeCount())

	g.Reset()
	assert.Equal(t, 0, g.GetNodeCount())
}

func TestGetEdgeCount(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("A", "flow", "C", nil)
	g.AddLink("B", "policy", "C", nil)

	assert.Equal(t, 3, g.GetEdgeCount())
}

func TestToJSON(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", map[string]interface{}{"bytes": float64(100)})
	g.AddLink("B", "flow", "C", map[string]interface{}{"bytes": float64(200)})

	jsonBytes, err := g.ToJSON()
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal(jsonBytes, &snapshot)
	require.NoError(t, err)

	assert.Len(t, snapshot.Nodes, 3)
	assert.Len(t, snapshot.Edges, 2)
}

func TestForEachEdge(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", 100)
	g.AddLink("B", "flow", "C", 200)

	var edges []string
	g.ForEachEdge(func(src, link, dst string, attr interface{}) {
		edges = append(edges, src+"-"+link+"-"+dst)
	})

	assert.Len(t, edges, 2)
	assert.Contains(t, edges, "A-flow-B")
	assert.Contains(t, edges, "B-flow-C")
}

func TestForEachNode(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", nil)
	g.AddLink("C", "flow", "D", nil)

	var nodes []string
	g.ForEachNode(func(node string) {
		nodes = append(nodes, node)
	})

	assert.Len(t, nodes, 4)
	assert.Contains(t, nodes, "A")
	assert.Contains(t, nodes, "B")
	assert.Contains(t, nodes, "C")
	assert.Contains(t, nodes, "D")
}

func TestConcurrentAccess(t *testing.T) {
	g := NewGraph()

	var wg sync.WaitGroup
	const goroutines = 10
	const operations = 100

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				src := "node" + string(rune('A'+id))
				dst := "node" + string(rune('A'+((id+1)%goroutines)))
				g.AddLink(src, "flow", dst, j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				g.GetNodeCount()
				g.GetEdgeCount()
				g.All()
			}
		}()
	}

	wg.Wait()

	// Should have at least some nodes
	assert.Greater(t, g.GetNodeCount(), 0)
}

func TestPurgeOutLinks(t *testing.T) {
	g := NewGraph()

	g.AddLink("A", "flow", "B", 100)
	g.AddLink("A", "flow", "C", 200)
	g.AddLink("A", "flow", "D", 300)

	// Purge links where attribute > 150
	g.PurgeOutLinks("A", func(src, link, dst string, attr interface{}, param interface{}) bool {
		threshold := param.(int)
		return attr.(int) > threshold
	}, 150)

	// Only B should remain
	assert.NotNil(t, g.Attr("A", "flow", "B"))
	assert.Nil(t, g.Attr("A", "flow", "C"))
	assert.Nil(t, g.Attr("A", "flow", "D"))
}

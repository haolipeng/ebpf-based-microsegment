// input: node and edge operations (add/delete/update)
// output: graph query results, adjacency lookups, connectivity analysis
// pos: graph - in-memory directed multi-graph data structure
//
// Package graph provides an in-memory graph database implementation.
// Adapted from NeuVector's controller/graph/graph.go (Apache 2.0 License).

package graph

import (
	"encoding/json"
	"reflect"
	"sync"
)

// Callback types for graph events.
type (
	NewLinkCallback        func(src, link, dst string)
	DelNodeCallback        func(node string)
	DelLinkCallback        func(src, link, dst string)
	UpdateLinkAttrCallback func(src, link, dst string)
	ConnectedNodeCallback  func(node string) bool
	PurgeOutLinkCallback   func(src, link, dst string, attr interface{}, param interface{}) bool
)

// graphLink represents edges leading to other nodes, indexed by node name.
type graphLink struct {
	ends map[string]interface{} // destination node name -> edge attribute
}

// graphNode represents a node with incoming and outgoing links indexed by link type.
type graphNode struct {
	ins  map[string]*graphLink // link type -> incoming edges
	outs map[string]*graphLink // link type -> outgoing edges
}

// Graph is an in-memory directed multi-graph with typed edges.
// It supports:
// - Multiple edge types between the same pair of nodes (via link name)
// - Arbitrary attributes on edges
// - Callback hooks for graph mutations
type Graph struct {
	mu               sync.RWMutex
	nodes            map[string]*graphNode
	cbNewLink        NewLinkCallback
	cbDelNode        DelNodeCallback
	cbDelLink        DelLinkCallback
	cbUpdateLinkAttr UpdateLinkAttrCallback
}

// NewGraph creates and returns a new empty Graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*graphNode),
	}
}

// RegisterNewLinkHook sets the callback for new link creation.
func (g *Graph) RegisterNewLinkHook(cb NewLinkCallback) {
	g.cbNewLink = cb
}

// RegisterDelNodeHook sets the callback for node deletion.
func (g *Graph) RegisterDelNodeHook(cb DelNodeCallback) {
	g.cbDelNode = cb
}

// RegisterDelLinkHook sets the callback for link deletion.
func (g *Graph) RegisterDelLinkHook(cb DelLinkCallback) {
	g.cbDelLink = cb
}

// RegisterUpdateLinkAttrHook sets the callback for link attribute updates.
func (g *Graph) RegisterUpdateLinkAttrHook(cb UpdateLinkAttrCallback) {
	g.cbUpdateLinkAttr = cb
}

// Reset clears all nodes and edges from the graph.
func (g *Graph) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = make(map[string]*graphNode)
}

// AddLink adds or updates an edge from src to dst with the given link type and attribute.
// If the edge already exists and the attribute differs, it will be updated.
func (g *Graph) AddLink(src, link, dst string, attr interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.addLinkUnsafe(src, link, dst, attr)
}

// addLinkUnsafe is the internal implementation without locking.
func (g *Graph) addLinkUnsafe(src, link, dst string, attr interface{}) {
	var gn *graphNode
	var gl *graphLink
	var ok, newlink, updattr bool

	// Handle source node's outgoing edge
	if gn, ok = g.nodes[src]; !ok {
		gl = &graphLink{ends: make(map[string]interface{})}
		gl.ends[dst] = attr

		gn = &graphNode{
			ins:  make(map[string]*graphLink),
			outs: make(map[string]*graphLink),
		}
		gn.outs[link] = gl

		g.nodes[src] = gn
		newlink = true
	} else if gl, ok = gn.outs[link]; !ok {
		gl = &graphLink{ends: make(map[string]interface{})}
		gl.ends[dst] = attr

		gn.outs[link] = gl
		newlink = true
	} else if _, ok = gl.ends[dst]; !ok {
		gl.ends[dst] = attr
		newlink = true
	} else {
		if !reflect.DeepEqual(gl.ends[dst], attr) {
			gl.ends[dst] = attr
			updattr = true
		}
	}

	// Handle destination node's incoming edge
	if gn, ok = g.nodes[dst]; !ok {
		gl = &graphLink{ends: make(map[string]interface{})}
		gl.ends[src] = attr

		gn = &graphNode{
			ins:  make(map[string]*graphLink),
			outs: make(map[string]*graphLink),
		}
		gn.ins[link] = gl

		g.nodes[dst] = gn
		newlink = true
	} else if gl, ok = gn.ins[link]; !ok {
		gl = &graphLink{ends: make(map[string]interface{})}
		gl.ends[src] = attr

		gn.ins[link] = gl
		newlink = true
	} else if _, ok = gl.ends[src]; !ok {
		gl.ends[src] = attr
		newlink = true
	} else {
		if !reflect.DeepEqual(gl.ends[src], attr) {
			gl.ends[src] = attr
			updattr = true
		}
	}

	if newlink && g.cbNewLink != nil {
		g.cbNewLink(src, link, dst)
	}
	if updattr && g.cbUpdateLinkAttr != nil {
		g.cbUpdateLinkAttr(src, link, dst)
	}
}

// Attr returns the attribute of the edge from src to dst with the given link type.
// Returns nil if the edge doesn't exist.
func (g *Graph) Attr(src, link, dst string) interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if s, ok := g.nodes[src]; ok {
		if gl, ok := s.outs[link]; ok {
			if attr, ok := gl.ends[dst]; ok {
				return attr
			}
		}
	}
	return nil
}

// DeleteLink removes the edge from src to dst with the given link type.
func (g *Graph) DeleteLink(src, link, dst string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.deleteLinkUnsafe(src, link, dst)
}

// deleteLinkUnsafe is the internal implementation without locking.
func (g *Graph) deleteLinkUnsafe(src, link, dst string) {
	var s, d *graphNode
	var ok bool

	if s, ok = g.nodes[src]; !ok {
		return
	}
	if d, ok = g.nodes[dst]; !ok {
		return
	}

	if gl, ok := s.outs[link]; ok {
		if _, ok = gl.ends[dst]; ok {
			delete(gl.ends, dst)
			if len(gl.ends) == 0 {
				delete(s.outs, link)
				if g.cbDelLink != nil {
					g.cbDelLink(src, link, dst)
				}
			}
		}
	}

	if gl, ok := d.ins[link]; ok {
		if _, ok = gl.ends[src]; ok {
			delete(gl.ends, src)
			if len(gl.ends) == 0 {
				delete(d.ins, link)
			}
		}
	}
}

// DeleteNode removes a node and all its edges from the graph.
func (g *Graph) DeleteNode(node string) string {
	g.mu.Lock()
	defer g.mu.Unlock()

	var gn *graphNode
	var ok bool

	if gn, ok = g.nodes[node]; !ok {
		return ""
	}

	// Delete all incoming edges
	for link, gl := range gn.ins {
		for n := range gl.ends {
			g.deleteLinkUnsafe(n, link, node)
		}
	}

	// Delete all outgoing edges
	for link, gl := range gn.outs {
		for n := range gl.ends {
			g.deleteLinkUnsafe(node, link, n)
		}
	}

	delete(g.nodes, node)

	if g.cbDelNode != nil {
		g.cbDelNode(node)
	}

	return node
}

// Node returns the node name if it exists, empty string otherwise.
func (g *Graph) Node(v string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[v]; ok {
		return v
	}
	return ""
}

// All returns a set of all node names in the graph.
func (g *Graph) All() Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := NewSet()
	for v := range g.nodes {
		ret.Add(v)
	}
	return ret
}

// GetNodeCount returns the total number of nodes.
func (g *Graph) GetNodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// NoIn returns nodes with no incoming edges.
func (g *Graph) NoIn() Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := NewSet()
	for v, n := range g.nodes {
		if len(n.ins) == 0 {
			ret.Add(v)
		}
	}
	return ret
}

// NoInByLink returns nodes with no incoming edges of the specified link type.
func (g *Graph) NoInByLink(link string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := NewSet()
	for v, n := range g.nodes {
		if gl, ok := n.ins[link]; !ok || len(gl.ends) == 0 {
			ret.Add(v)
		}
	}
	return ret
}

// NoOut returns nodes with no outgoing edges.
func (g *Graph) NoOut() Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := NewSet()
	for v, n := range g.nodes {
		if len(n.outs) == 0 {
			ret.Add(v)
		}
	}
	return ret
}

// NoOutByLink returns nodes with no outgoing edges of the specified link type.
func (g *Graph) NoOutByLink(link string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := NewSet()
	for v, n := range g.nodes {
		if gl, ok := n.outs[link]; !ok || len(gl.ends) == 0 {
			ret.Add(v)
		}
	}
	return ret
}

// Ins returns all nodes with incoming edges to this node.
func (g *Graph) Ins(node string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[node]; !ok {
		return nil
	}

	ret := NewSet()
	n := g.nodes[node]
	for _, l := range n.ins {
		for v := range l.ends {
			ret.Add(v)
		}
	}
	return ret
}

// InsByLink returns all nodes with incoming edges of the specified link type.
func (g *Graph) InsByLink(node, link string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[node]; !ok {
		return nil
	}

	ret := NewSet()
	n := g.nodes[node]
	if gl, ok := n.ins[link]; ok {
		for v := range gl.ends {
			ret.Add(v)
		}
	}
	return ret
}

// Outs returns all nodes with outgoing edges from this node.
func (g *Graph) Outs(node string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[node]; !ok {
		return nil
	}

	ret := NewSet()
	n := g.nodes[node]
	for _, l := range n.outs {
		for v := range l.ends {
			ret.Add(v)
		}
	}
	return ret
}

// OutsByLink returns all nodes with outgoing edges of the specified link type.
func (g *Graph) OutsByLink(node, link string) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[node]; !ok {
		return nil
	}

	ret := NewSet()
	n := g.nodes[node]
	if gl, ok := n.outs[link]; ok {
		for v := range gl.ends {
			ret.Add(v)
		}
	}
	return ret
}

// Both returns all nodes connected to this node (incoming or outgoing).
func (g *Graph) Both(node string) Set {
	ins := g.Ins(node)
	outs := g.Outs(node)
	if ins == nil {
		return nil
	}
	return ins.Union(outs)
}

// BothByLink returns all nodes connected by the specified link type.
func (g *Graph) BothByLink(node, link string) Set {
	ins := g.InsByLink(node, link)
	outs := g.OutsByLink(node, link)
	if ins == nil {
		return nil
	}
	return ins.Union(outs)
}

// Connected returns all nodes reachable from the given node.
func (g *Graph) Connected(node string, cb ConnectedNodeCallback) Set {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[node]; !ok {
		return nil
	}

	ret := NewSet()
	ret.Add(node)
	q := []string{node}

	for len(q) > 0 {
		node, q = q[0], q[1:]

		both := g.bothUnsafe(node)
		for n := range both.Iter() {
			nodeStr := n.(string)
			if cb != nil && cb(nodeStr) {
				if !ret.Contains(n) {
					ret.Add(n)
					q = append(q, nodeStr)
				}
			}
		}
	}
	return ret
}

// bothUnsafe is internal implementation without locking.
func (g *Graph) bothUnsafe(node string) Set {
	ret := NewSet()
	if n, ok := g.nodes[node]; ok {
		for _, l := range n.ins {
			for v := range l.ends {
				ret.Add(v)
			}
		}
		for _, l := range n.outs {
			for v := range l.ends {
				ret.Add(v)
			}
		}
	}
	return ret
}

// BetweenDirLinks returns all link types and their attributes between src and dst.
func (g *Graph) BetweenDirLinks(src, dst string) map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ret := make(map[string]interface{})
	if n, ok := g.nodes[src]; ok {
		for ln, l := range n.outs {
			if attr, ok := l.ends[dst]; ok {
				ret[ln] = attr
			}
		}
	}
	return ret
}

// PurgeOutLinks removes outgoing edges from src that match the callback condition.
func (g *Graph) PurgeOutLinks(src string, cb PurgeOutLinkCallback, param interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if n, ok := g.nodes[src]; ok {
		for ln, l := range n.outs {
			for dst, attr := range l.ends {
				if cb(src, ln, dst, attr, param) {
					delete(l.ends, dst)
					if len(l.ends) == 0 {
						delete(n.outs, ln)
					}
				}
			}
		}
	}
}

// GetEdgeCount returns the total number of edges in the graph.
func (g *Graph) GetEdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := 0
	for _, n := range g.nodes {
		for _, l := range n.outs {
			count += len(l.ends)
		}
	}
	return count
}

// GraphSnapshot represents a serializable snapshot of the graph.
type GraphSnapshot struct {
	Nodes []string               `json:"nodes"`
	Edges []EdgeSnapshot         `json:"edges"`
}

// EdgeSnapshot represents a serializable edge.
type EdgeSnapshot struct {
	Src  string      `json:"src"`
	Link string      `json:"link"`
	Dst  string      `json:"dst"`
	Attr interface{} `json:"attr"`
}

// ToJSON serializes the graph to JSON.
func (g *Graph) ToJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	snapshot := GraphSnapshot{
		Nodes: make([]string, 0, len(g.nodes)),
		Edges: make([]EdgeSnapshot, 0),
	}

	for node := range g.nodes {
		snapshot.Nodes = append(snapshot.Nodes, node)
	}

	for src, n := range g.nodes {
		for link, l := range n.outs {
			for dst, attr := range l.ends {
				snapshot.Edges = append(snapshot.Edges, EdgeSnapshot{
					Src:  src,
					Link: link,
					Dst:  dst,
					Attr: attr,
				})
			}
		}
	}

	return json.Marshal(snapshot)
}

// ForEachEdge iterates over all edges in the graph.
// The callback receives src, link type, dst, and attribute.
func (g *Graph) ForEachEdge(cb func(src, link, dst string, attr interface{})) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for src, n := range g.nodes {
		for link, l := range n.outs {
			for dst, attr := range l.ends {
				cb(src, link, dst, attr)
			}
		}
	}
}

// ForEachNode iterates over all nodes in the graph.
func (g *Graph) ForEachNode(cb func(node string)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for node := range g.nodes {
		cb(node)
	}
}

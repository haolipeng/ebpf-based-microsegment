// input: node/edge update requests, query parameters
// output: topology graph state, node and edge queries, TTL cleanup
// pos: topology - manages in-memory network topology graph state

package topology

import (
	"sync"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/graph"
	"github.com/sirupsen/logrus"
)

// Manager manages the in-memory network topology graph.
type Manager struct {
	graph      *graph.Graph
	nodeAttrs  map[string]*NodeAttr // node ID -> node attributes
	edgeAttrs  map[string]*EdgeAttr // "src|dst" -> edge attributes
	mu         sync.RWMutex
	cleanupTTL time.Duration // TTL for stale entries
}

// NewManager creates a new topology manager.
func NewManager() *Manager {
	m := &Manager{
		graph:      graph.NewGraph(),
		nodeAttrs:  make(map[string]*NodeAttr),
		edgeAttrs:  make(map[string]*EdgeAttr),
		cleanupTTL: 5 * time.Minute,
	}

	// Register callbacks for graph events
	m.graph.RegisterDelNodeHook(func(node string) {
		logrus.Debugf("Topology node deleted: %s", node)
	})

	m.graph.RegisterNewLinkHook(func(src, link, dst string) {
		logrus.Debugf("Topology link created: %s -[%s]-> %s", src, link, dst)
	})

	return m
}

// SetCleanupTTL sets the TTL for stale entry cleanup.
func (m *Manager) SetCleanupTTL(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupTTL = ttl
}

// UpdateNode updates or creates a node in the topology.
func (m *Manager) UpdateNode(nodeID string, attr *NodeAttr) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if existing, ok := m.nodeAttrs[nodeID]; ok {
		// Update existing node
		existing.LastSeen = now
		if attr.Name != "" {
			existing.Name = attr.Name
		}
		if attr.DisplayName != "" {
			existing.DisplayName = attr.DisplayName
		}
		if attr.Type != "" {
			existing.Type = attr.Type
		}
		if attr.Namespace != "" {
			existing.Namespace = attr.Namespace
		}
		if len(attr.Labels) > 0 {
			if existing.Labels == nil {
				existing.Labels = make(map[string]string)
			}
			for k, v := range attr.Labels {
				existing.Labels[k] = v
			}
		}
		if len(attr.IPs) > 0 {
			existing.IPs = mergeIPs(existing.IPs, attr.IPs)
		}
		if attr.AgentID != "" {
			existing.AgentID = attr.AgentID
		}
	} else {
		// Create new node
		attr.ID = nodeID
		if attr.FirstSeen.IsZero() {
			attr.FirstSeen = now
		}
		attr.LastSeen = now
		m.nodeAttrs[nodeID] = attr
	}

	// Add node to graph with a self-link to track it
	m.graph.AddLink(nodeID, LinkTypeAttr, nodeID, m.nodeAttrs[nodeID])
}

// UpdateEdge updates or creates an edge in the topology.
func (m *Manager) UpdateEdge(srcID, dstID string, attr *EdgeAttr) {
	m.mu.Lock()
	defer m.mu.Unlock()

	edgeKey := makeEdgeKey(srcID, dstID)
	now := time.Now()

	if existing, ok := m.edgeAttrs[edgeKey]; ok {
		// Update existing edge
		existing.Bytes += attr.Bytes
		existing.Sessions += attr.Sessions
		existing.PacketCount += attr.PacketCount
		existing.LastSeen = now

		// Merge protocols
		existing.Protocols = mergeStrings(existing.Protocols, attr.Protocols)
		// Merge ports
		existing.Ports = mergePorts(existing.Ports, attr.Ports)
		// Merge applications
		existing.Applications = mergeStrings(existing.Applications, attr.Applications)

		// Update policy action (prefer deny over allow)
		if attr.PolicyAction == PolicyActionDeny {
			existing.PolicyAction = PolicyActionDeny
		} else if existing.PolicyAction == "" {
			existing.PolicyAction = attr.PolicyAction
		}

		// Update severity (keep highest)
		existing.Severity = maxSeverity(existing.Severity, attr.Severity)
	} else {
		// Create new edge
		if attr.FirstSeen.IsZero() {
			attr.FirstSeen = now
		}
		attr.LastSeen = now
		m.edgeAttrs[edgeKey] = attr
	}

	// Add edge to graph
	m.graph.AddLink(srcID, LinkTypeFlow, dstID, m.edgeAttrs[edgeKey])
}

// GetTopology returns the current topology with optional filtering.
func (m *Manager) GetTopology(filter *TopologyFilter) *Topology {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]*TopologyNode, 0)
	edges := make([]*TopologyEdge, 0)

	// Collect nodes
	nodeStats := m.calculateNodeStats()
	for nodeID, attr := range m.nodeAttrs {
		if !m.matchesNodeFilter(attr, filter) {
			continue
		}

		node := &TopologyNode{
			ID:          nodeID,
			Name:        attr.Name,
			DisplayName: attr.DisplayName,
			Type:        attr.Type,
			Namespace:   attr.Namespace,
			Labels:      attr.Labels,
			IPs:         attr.IPs,
			FirstSeen:   attr.FirstSeen,
			LastSeen:    attr.LastSeen,
			Stats:       nodeStats[nodeID],
		}
		nodes = append(nodes, node)
	}

	// Create a set of included node IDs for edge filtering
	includedNodes := make(map[string]bool)
	for _, node := range nodes {
		includedNodes[node.ID] = true
	}

	// Collect edges
	for edgeKey, attr := range m.edgeAttrs {
		srcID, dstID := parseEdgeKey(edgeKey)

		// Only include edges where both endpoints are in the filtered nodes
		if !includedNodes[srcID] || !includedNodes[dstID] {
			continue
		}

		if !m.matchesEdgeFilter(attr, filter) {
			continue
		}

		edge := &TopologyEdge{
			Source:       srcID,
			Target:       dstID,
			Bytes:        attr.Bytes,
			Sessions:     attr.Sessions,
			PacketCount:  attr.PacketCount,
			Protocols:    attr.Protocols,
			Ports:        attr.Ports,
			Applications: attr.Applications,
			PolicyAction: attr.PolicyAction,
			Severity:     attr.Severity,
			LastSeen:     attr.LastSeen,
		}
		edges = append(edges, edge)
	}

	return &Topology{
		Nodes: nodes,
		Edges: edges,
		Metadata: &TopologyMetadata{
			TotalNodes:  len(nodes),
			TotalEdges:  len(edges),
			GeneratedAt: time.Now(),
		},
	}
}

// GetNodeDetail returns detailed information about a specific node.
func (m *Manager) GetNodeDetail(nodeID string) *NodeDetail {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attr, ok := m.nodeAttrs[nodeID]
	if !ok {
		return nil
	}

	// Calculate stats for this node
	nodeStats := m.calculateNodeStats()

	detail := &NodeDetail{
		TopologyNode: &TopologyNode{
			ID:          nodeID,
			Name:        attr.Name,
			DisplayName: attr.DisplayName,
			Type:        attr.Type,
			Namespace:   attr.Namespace,
			Labels:      attr.Labels,
			IPs:         attr.IPs,
			FirstSeen:   attr.FirstSeen,
			LastSeen:    attr.LastSeen,
			Stats:       nodeStats[nodeID],
		},
		InboundEdges:  make([]*TopologyEdge, 0),
		OutboundEdges: make([]*TopologyEdge, 0),
		Metadata:      attr.Metadata,
	}

	// Find inbound and outbound edges
	for edgeKey, edgeAttr := range m.edgeAttrs {
		srcID, dstID := parseEdgeKey(edgeKey)

		edge := &TopologyEdge{
			Source:       srcID,
			Target:       dstID,
			Bytes:        edgeAttr.Bytes,
			Sessions:     edgeAttr.Sessions,
			PacketCount:  edgeAttr.PacketCount,
			Protocols:    edgeAttr.Protocols,
			Ports:        edgeAttr.Ports,
			Applications: edgeAttr.Applications,
			PolicyAction: edgeAttr.PolicyAction,
			Severity:     edgeAttr.Severity,
			LastSeen:     edgeAttr.LastSeen,
		}

		if dstID == nodeID {
			detail.InboundEdges = append(detail.InboundEdges, edge)
		}
		if srcID == nodeID {
			detail.OutboundEdges = append(detail.OutboundEdges, edge)
		}
	}

	return detail
}

// GetEdgeDetail returns detailed information about a specific edge.
func (m *Manager) GetEdgeDetail(srcID, dstID string) *EdgeDetail {
	m.mu.RLock()
	defer m.mu.RUnlock()

	edgeKey := makeEdgeKey(srcID, dstID)
	attr, ok := m.edgeAttrs[edgeKey]
	if !ok {
		return nil
	}

	detail := &EdgeDetail{
		TopologyEdge: &TopologyEdge{
			Source:       srcID,
			Target:       dstID,
			Bytes:        attr.Bytes,
			Sessions:     attr.Sessions,
			PacketCount:  attr.PacketCount,
			Protocols:    attr.Protocols,
			Ports:        attr.Ports,
			Applications: attr.Applications,
			PolicyAction: attr.PolicyAction,
			Severity:     attr.Severity,
			LastSeen:     attr.LastSeen,
		},
		Entries: attr.Entries,
	}

	// Add source and target node info
	if srcAttr, ok := m.nodeAttrs[srcID]; ok {
		detail.SourceNode = &TopologyNode{
			ID:          srcID,
			Name:        srcAttr.Name,
			DisplayName: srcAttr.DisplayName,
			Type:        srcAttr.Type,
			Namespace:   srcAttr.Namespace,
			Labels:      srcAttr.Labels,
		}
	}

	if dstAttr, ok := m.nodeAttrs[dstID]; ok {
		detail.TargetNode = &TopologyNode{
			ID:          dstID,
			Name:        dstAttr.Name,
			DisplayName: dstAttr.DisplayName,
			Type:        dstAttr.Type,
			Namespace:   dstAttr.Namespace,
			Labels:      dstAttr.Labels,
		}
	}

	return detail
}

// GetStats returns topology statistics.
func (m *Manager) GetStats() *TopologyMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &TopologyMetadata{
		TotalNodes:  len(m.nodeAttrs),
		TotalEdges:  len(m.edgeAttrs),
		GeneratedAt: time.Now(),
	}
}

// CleanupStale removes nodes and edges that haven't been seen recently.
func (m *Manager) CleanupStale() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.cleanupTTL)
	removed := 0

	// Remove stale edges
	for edgeKey, attr := range m.edgeAttrs {
		if attr.LastSeen.Before(cutoff) {
			srcID, dstID := parseEdgeKey(edgeKey)
			m.graph.DeleteLink(srcID, LinkTypeFlow, dstID)
			delete(m.edgeAttrs, edgeKey)
			removed++
		}
	}

	// Remove stale nodes (only if they have no edges)
	for nodeID, attr := range m.nodeAttrs {
		if attr.LastSeen.Before(cutoff) && !m.hasEdges(nodeID) {
			m.graph.DeleteNode(nodeID)
			delete(m.nodeAttrs, nodeID)
			removed++
		}
	}

	if removed > 0 {
		logrus.Infof("Cleaned up %d stale topology entries", removed)
	}

	return removed
}

// Reset clears all topology data.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.graph.Reset()
	m.nodeAttrs = make(map[string]*NodeAttr)
	m.edgeAttrs = make(map[string]*EdgeAttr)
}

// calculateNodeStats calculates statistics for all nodes.
func (m *Manager) calculateNodeStats() map[string]*NodeStats {
	stats := make(map[string]*NodeStats)

	// Initialize stats for all nodes
	for nodeID := range m.nodeAttrs {
		stats[nodeID] = &NodeStats{}
	}

	// Calculate from edges
	for edgeKey, attr := range m.edgeAttrs {
		srcID, dstID := parseEdgeKey(edgeKey)

		if s, ok := stats[srcID]; ok {
			s.OutboundConnections++
			s.TotalBytesOut += attr.Bytes
			s.TotalPacketsOut += attr.PacketCount
		}

		if s, ok := stats[dstID]; ok {
			s.InboundConnections++
			s.TotalBytesIn += attr.Bytes
			s.TotalPacketsIn += attr.PacketCount
		}
	}

	return stats
}

// matchesNodeFilter checks if a node matches the given filter.
func (m *Manager) matchesNodeFilter(attr *NodeAttr, filter *TopologyFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by namespace
	if filter.Namespace != "" && attr.Namespace != filter.Namespace {
		return false
	}

	// Filter by node types
	if len(filter.NodeTypes) > 0 {
		found := false
		for _, t := range filter.NodeTypes {
			if attr.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by labels
	if len(filter.Labels) > 0 {
		for k, v := range filter.Labels {
			if attr.Labels[k] != v {
				return false
			}
		}
	}

	// Exclude external nodes if not requested
	if !filter.IncludeExternal && attr.Type == NodeTypeExternal {
		return false
	}

	// Filter by time range
	if filter.TimeRange != nil {
		if attr.LastSeen.Before(filter.TimeRange.Start) || attr.FirstSeen.After(filter.TimeRange.End) {
			return false
		}
	}

	return true
}

// matchesEdgeFilter checks if an edge matches the given filter.
func (m *Manager) matchesEdgeFilter(attr *EdgeAttr, filter *TopologyFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by minimum flow count
	if filter.MinFlowCount > 0 && int(attr.Sessions) < filter.MinFlowCount {
		return false
	}

	// Filter by policy action
	if filter.PolicyAction != "" && attr.PolicyAction != filter.PolicyAction {
		return false
	}

	// Filter by time range
	if filter.TimeRange != nil {
		if attr.LastSeen.Before(filter.TimeRange.Start) {
			return false
		}
	}

	return true
}

// hasEdges checks if a node has any edges.
func (m *Manager) hasEdges(nodeID string) bool {
	for edgeKey := range m.edgeAttrs {
		srcID, dstID := parseEdgeKey(edgeKey)
		if srcID == nodeID || dstID == nodeID {
			return true
		}
	}
	return false
}

// Helper functions

func makeEdgeKey(srcID, dstID string) string {
	return srcID + "|" + dstID
}

func parseEdgeKey(key string) (srcID, dstID string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func mergeIPs(a, b []string) []string {
	seen := make(map[string]bool)
	for _, ip := range a {
		seen[ip] = true
	}
	for _, ip := range b {
		if !seen[ip] {
			a = append(a, ip)
			seen[ip] = true
		}
	}
	return a
}

func mergeStrings(a, b []string) []string {
	seen := make(map[string]bool)
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			a = append(a, s)
			seen[s] = true
		}
	}
	return a
}

func mergePorts(a, b []uint16) []uint16 {
	seen := make(map[uint16]bool)
	for _, p := range a {
		seen[p] = true
	}
	for _, p := range b {
		if !seen[p] {
			a = append(a, p)
			seen[p] = true
		}
	}
	return a
}

func maxSeverity(a, b string) string {
	order := map[string]int{
		"":               0,
		SeverityInfo:     1,
		SeverityLow:      2,
		SeverityMedium:   3,
		SeverityHigh:     4,
		SeverityCritical: 5,
	}

	if order[a] >= order[b] {
		return a
	}
	return b
}

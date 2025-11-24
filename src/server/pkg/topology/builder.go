package topology

import (
	"fmt"
	"net"
	"time"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/sirupsen/logrus"
)

// Builder converts flow events into topology updates.
type Builder struct {
	manager *Manager

	// Configuration
	externalCIDRs []*net.IPNet // CIDRs considered external
	internalCIDRs []*net.IPNet // CIDRs considered internal
}

// NewBuilder creates a new topology builder.
func NewBuilder(manager *Manager) *Builder {
	b := &Builder{
		manager:       manager,
		externalCIDRs: make([]*net.IPNet, 0),
		internalCIDRs: make([]*net.IPNet, 0),
	}

	// Default external CIDRs (public internet)
	defaultExternalCIDRs := []string{
		"0.0.0.0/0", // Default route (will be checked last)
	}

	// Default internal CIDRs (RFC 1918)
	defaultInternalCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // Link-local
	}

	for _, cidr := range defaultExternalCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			b.externalCIDRs = append(b.externalCIDRs, ipnet)
		}
	}

	for _, cidr := range defaultInternalCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			b.internalCIDRs = append(b.internalCIDRs, ipnet)
		}
	}

	return b
}

// ProcessFlowEvents processes a batch of flow events and updates the topology.
func (b *Builder) ProcessFlowEvents(events []*flowpb.FlowEvent) {
	for _, event := range events {
		b.processFlowEvent(event)
	}
}

// processFlowEvent processes a single flow event.
func (b *Builder) processFlowEvent(event *flowpb.FlowEvent) {
	// Convert IPs from uint32 to string
	srcIP := uint32ToIP(event.SrcIp)
	dstIP := uint32ToIP(event.DstIp)

	// Determine node IDs and types
	srcNodeID, srcNodeAttr := b.resolveNode(srcIP, event.SourceLabels, event.AgentId)
	dstNodeID, dstNodeAttr := b.resolveNode(dstIP, event.DestLabels, event.AgentId)

	// Update source node
	b.manager.UpdateNode(srcNodeID, srcNodeAttr)

	// Update destination node
	b.manager.UpdateNode(dstNodeID, dstNodeAttr)

	// Create edge attributes
	edgeAttr := &EdgeAttr{
		Bytes:       event.ByteCount,
		Sessions:    1,
		PacketCount: event.PacketCount,
		Protocols:   []string{protocolToString(event.Protocol)},
		Ports:       []uint16{uint16(event.DstPort)},
		LastSeen:    time.Now(),
	}

	// Set policy action
	edgeAttr.PolicyAction = policyActionToString(event.PolicyAction)

	// Update edge
	b.manager.UpdateEdge(srcNodeID, dstNodeID, edgeAttr)

	logrus.Debugf("Topology updated: %s -> %s (%d bytes)",
		srcNodeID, dstNodeID, event.ByteCount)
}

// resolveNode determines the node ID and attributes from an IP and labels.
func (b *Builder) resolveNode(ip string, labels map[string]string, agentID string) (string, *NodeAttr) {
	nodeType := b.determineNodeType(ip)

	var nodeID, name, displayName string

	switch nodeType {
	case NodeTypeExternal:
		// External nodes are grouped by IP for now
		nodeID = "external:" + ip
		name = ip
		displayName = "External: " + ip

	case NodeTypeWorkload:
		// Workload nodes use labels for identification
		nodeID = b.generateWorkloadID(ip, labels)
		name = b.getWorkloadName(labels)
		displayName = b.getWorkloadDisplayName(labels, ip)

	case NodeTypeHost:
		nodeID = "host:" + agentID
		name = agentID
		displayName = "Host: " + agentID

	default:
		nodeID = "unknown:" + ip
		name = ip
		displayName = "Unknown: " + ip
	}

	attr := &NodeAttr{
		ID:          nodeID,
		Name:        name,
		DisplayName: displayName,
		Type:        nodeType,
		Labels:      labels,
		IPs:         []string{ip},
		AgentID:     agentID,
		LastSeen:    time.Now(),
	}

	// Extract namespace from labels
	if ns, ok := labels["namespace"]; ok {
		attr.Namespace = ns
	} else if ns, ok := labels["kubernetes.io/namespace"]; ok {
		attr.Namespace = ns
	}

	return nodeID, attr
}

// determineNodeType determines the type of node based on IP address.
func (b *Builder) determineNodeType(ipStr string) NodeType {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return NodeTypeUnknown
	}

	// Check if it's an internal IP first
	for _, cidr := range b.internalCIDRs {
		if cidr.Contains(ip) {
			return NodeTypeWorkload
		}
	}

	// If not internal, it's external
	return NodeTypeExternal
}

// generateWorkloadID generates a unique ID for a workload based on labels.
func (b *Builder) generateWorkloadID(ip string, labels map[string]string) string {
	// Priority: pod name > deployment name > app label > IP
	if podName, ok := labels["pod"]; ok {
		if ns, ok := labels["namespace"]; ok {
			return fmt.Sprintf("pod:%s/%s", ns, podName)
		}
		return "pod:" + podName
	}

	if deployment, ok := labels["deployment"]; ok {
		if ns, ok := labels["namespace"]; ok {
			return fmt.Sprintf("deployment:%s/%s", ns, deployment)
		}
		return "deployment:" + deployment
	}

	if app, ok := labels["app"]; ok {
		if ns, ok := labels["namespace"]; ok {
			return fmt.Sprintf("app:%s/%s", ns, app)
		}
		return "app:" + app
	}

	// Fallback to IP-based ID
	return "ip:" + ip
}

// getWorkloadName gets a human-readable name for a workload.
func (b *Builder) getWorkloadName(labels map[string]string) string {
	if podName, ok := labels["pod"]; ok {
		return podName
	}
	if deployment, ok := labels["deployment"]; ok {
		return deployment
	}
	if app, ok := labels["app"]; ok {
		return app
	}
	if name, ok := labels["name"]; ok {
		return name
	}
	return "unknown"
}

// getWorkloadDisplayName gets a display name for a workload.
func (b *Builder) getWorkloadDisplayName(labels map[string]string, ip string) string {
	name := b.getWorkloadName(labels)
	if ns, ok := labels["namespace"]; ok {
		return fmt.Sprintf("%s/%s", ns, name)
	}
	if name != "unknown" {
		return name
	}
	return ip
}

// Helper functions

// uint32ToIP converts a uint32 IP to string format.
func uint32ToIP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24),
		byte(ip>>16),
		byte(ip>>8),
		byte(ip))
}

// protocolToString converts a protocol number to string.
func protocolToString(proto commonpb.Protocol) string {
	switch proto {
	case commonpb.Protocol_PROTOCOL_TCP:
		return "TCP"
	case commonpb.Protocol_PROTOCOL_UDP:
		return "UDP"
	case commonpb.Protocol_PROTOCOL_ICMP:
		return "ICMP"
	default:
		return fmt.Sprintf("PROTO_%d", proto)
	}
}

// policyActionToString converts a policy action to string.
func policyActionToString(action commonpb.PolicyAction) string {
	switch action {
	case commonpb.PolicyAction_ACTION_ALLOW:
		return PolicyActionAllow
	case commonpb.PolicyAction_ACTION_DENY:
		return PolicyActionDeny
	default:
		return ""
	}
}

// StartCleanupRoutine starts a background routine to clean up stale entries.
func (b *Builder) StartCleanupRoutine(interval time.Duration, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				b.manager.CleanupStale()
			case <-stopCh:
				logrus.Info("Topology cleanup routine stopped")
				return
			}
		}
	}()

	logrus.Infof("Topology cleanup routine started (interval: %v)", interval)
}

// GetManager returns the underlying topology manager.
func (b *Builder) GetManager() *Manager {
	return b.manager
}

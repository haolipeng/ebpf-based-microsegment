// Package topology provides network topology management for visualizing
// workload communication patterns.
package topology

import (
	"time"
)

// NodeType represents the type of a topology node.
type NodeType string

const (
	// NodeTypeWorkload represents a container/pod workload.
	NodeTypeWorkload NodeType = "workload"
	// NodeTypeHost represents a host/node.
	NodeTypeHost NodeType = "host"
	// NodeTypeExternal represents an external entity (internet, external service).
	NodeTypeExternal NodeType = "external"
	// NodeTypeService represents a K8s service.
	NodeTypeService NodeType = "service"
	// NodeTypeUnknown represents an unknown entity.
	NodeTypeUnknown NodeType = "unknown"
)

// LinkType represents the type of edge in the topology graph.
const (
	// LinkTypeFlow represents actual network flow between nodes.
	LinkTypeFlow = "flow"
	// LinkTypeAttr represents node attribute storage.
	LinkTypeAttr = "attr"
)

// NodeAttr represents the attributes of a topology node.
type NodeAttr struct {
	// ID is the unique identifier for the node.
	ID string `json:"id"`

	// Name is the human-readable name of the node.
	Name string `json:"name"`

	// DisplayName is the display name (may include namespace prefix).
	DisplayName string `json:"display_name"`

	// Type is the node type (workload, host, external, etc.).
	Type NodeType `json:"type"`

	// Namespace is the K8s namespace (if applicable).
	Namespace string `json:"namespace,omitempty"`

	// Labels are the workload labels.
	Labels map[string]string `json:"labels,omitempty"`

	// IPs are the IP addresses associated with this node.
	IPs []string `json:"ips,omitempty"`

	// AgentID is the agent that reported this node.
	AgentID string `json:"agent_id,omitempty"`

	// FirstSeen is when this node was first observed.
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen is when this node was last observed.
	LastSeen time.Time `json:"last_seen"`

	// Metadata contains additional node-specific information.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EdgeAttr represents the attributes of a topology edge (conversation).
type EdgeAttr struct {
	// Bytes is the total bytes transferred.
	Bytes uint64 `json:"bytes"`

	// Sessions is the number of sessions/connections.
	Sessions uint32 `json:"sessions"`

	// PacketCount is the total number of packets.
	PacketCount uint64 `json:"packet_count"`

	// Protocols is the list of protocols used (TCP, UDP, etc.).
	Protocols []string `json:"protocols,omitempty"`

	// Ports is the list of destination ports used.
	Ports []uint16 `json:"ports,omitempty"`

	// Applications is the list of applications detected (HTTP, MySQL, etc.).
	Applications []string `json:"applications,omitempty"`

	// PolicyAction is the dominant policy action (allow, deny).
	PolicyAction string `json:"policy_action,omitempty"`

	// Severity is the highest threat severity level.
	Severity string `json:"severity,omitempty"`

	// FirstSeen is when this edge was first observed.
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen is when this edge was last observed.
	LastSeen time.Time `json:"last_seen"`

	// Entries contains detailed flow entries.
	Entries []*EdgeEntry `json:"entries,omitempty"`
}

// EdgeEntry represents a detailed flow entry within an edge.
type EdgeEntry struct {
	// Protocol is the network protocol (TCP=6, UDP=17).
	Protocol uint8 `json:"protocol"`

	// Port is the destination port.
	Port uint16 `json:"port"`

	// Application is the detected application layer protocol.
	Application string `json:"application,omitempty"`

	// Bytes is the bytes for this specific entry.
	Bytes uint64 `json:"bytes"`

	// Sessions is the session count for this entry.
	Sessions uint32 `json:"sessions"`

	// PolicyAction is the policy action for this entry.
	PolicyAction string `json:"policy_action,omitempty"`

	// ClientIP is the client IP (for aggregated entries).
	ClientIP string `json:"client_ip,omitempty"`

	// ServerIP is the server IP.
	ServerIP string `json:"server_ip,omitempty"`

	// LastSeen is when this entry was last seen.
	LastSeen time.Time `json:"last_seen"`
}

// Topology represents the complete network topology for API responses.
type Topology struct {
	// Nodes are all nodes in the topology.
	Nodes []*TopologyNode `json:"nodes"`

	// Edges are all edges in the topology.
	Edges []*TopologyEdge `json:"edges"`

	// Metadata contains topology-level metadata.
	Metadata *TopologyMetadata `json:"metadata"`
}

// TopologyNode represents a node in the API response.
type TopologyNode struct {
	// ID is the node identifier.
	ID string `json:"id"`

	// Name is the human-readable name.
	Name string `json:"name"`

	// DisplayName is the display name.
	DisplayName string `json:"display_name"`

	// Type is the node type.
	Type NodeType `json:"type"`

	// Namespace is the K8s namespace.
	Namespace string `json:"namespace,omitempty"`

	// Labels are the node labels.
	Labels map[string]string `json:"labels,omitempty"`

	// IPs are the associated IP addresses.
	IPs []string `json:"ips,omitempty"`

	// Stats contains node-level statistics.
	Stats *NodeStats `json:"stats,omitempty"`

	// FirstSeen is the first observation time.
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen is the last observation time.
	LastSeen time.Time `json:"last_seen"`
}

// NodeStats contains statistics for a node.
type NodeStats struct {
	// InboundConnections is the number of incoming connections.
	InboundConnections int `json:"inbound_connections"`

	// OutboundConnections is the number of outgoing connections.
	OutboundConnections int `json:"outbound_connections"`

	// TotalBytesIn is the total bytes received.
	TotalBytesIn uint64 `json:"total_bytes_in"`

	// TotalBytesOut is the total bytes sent.
	TotalBytesOut uint64 `json:"total_bytes_out"`

	// TotalPacketsIn is the total packets received.
	TotalPacketsIn uint64 `json:"total_packets_in"`

	// TotalPacketsOut is the total packets sent.
	TotalPacketsOut uint64 `json:"total_packets_out"`
}

// TopologyEdge represents an edge in the API response.
type TopologyEdge struct {
	// Source is the source node ID.
	Source string `json:"source"`

	// Target is the target node ID.
	Target string `json:"target"`

	// Bytes is the total bytes transferred.
	Bytes uint64 `json:"bytes"`

	// Sessions is the number of sessions.
	Sessions uint32 `json:"sessions"`

	// PacketCount is the total packets.
	PacketCount uint64 `json:"packet_count"`

	// Protocols is the list of protocols.
	Protocols []string `json:"protocols,omitempty"`

	// Ports is the list of destination ports.
	Ports []uint16 `json:"ports,omitempty"`

	// Applications is the list of applications.
	Applications []string `json:"applications,omitempty"`

	// PolicyAction is the policy action.
	PolicyAction string `json:"policy_action,omitempty"`

	// Severity is the threat severity.
	Severity string `json:"severity,omitempty"`

	// LastSeen is the last observation time.
	LastSeen time.Time `json:"last_seen"`
}

// TopologyMetadata contains metadata about the topology.
type TopologyMetadata struct {
	// TotalNodes is the total number of nodes.
	TotalNodes int `json:"total_nodes"`

	// TotalEdges is the total number of edges.
	TotalEdges int `json:"total_edges"`

	// TimeRange is the time range of the topology data.
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// GeneratedAt is when the topology was generated.
	GeneratedAt time.Time `json:"generated_at"`
}

// TimeRange represents a time range.
type TimeRange struct {
	// Start is the start time.
	Start time.Time `json:"start"`

	// End is the end time.
	End time.Time `json:"end"`
}

// TopologyFilter represents filtering options for topology queries.
type TopologyFilter struct {
	// TimeRange filters by time range.
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// Namespace filters by K8s namespace.
	Namespace string `json:"namespace,omitempty"`

	// Labels filters by label key-value pairs.
	Labels map[string]string `json:"labels,omitempty"`

	// NodeTypes filters by node types.
	NodeTypes []NodeType `json:"node_types,omitempty"`

	// IncludeExternal includes external nodes.
	IncludeExternal bool `json:"include_external"`

	// MinFlowCount filters edges by minimum flow count.
	MinFlowCount int `json:"min_flow_count,omitempty"`

	// PolicyAction filters by policy action.
	PolicyAction string `json:"policy_action,omitempty"`

	// GroupBy specifies how to group nodes (ip, workload, label:<key>, namespace).
	GroupBy string `json:"group_by,omitempty"`
}

// NodeDetail represents detailed information about a node.
type NodeDetail struct {
	*TopologyNode

	// InboundEdges are edges where this node is the target.
	InboundEdges []*TopologyEdge `json:"inbound_edges,omitempty"`

	// OutboundEdges are edges where this node is the source.
	OutboundEdges []*TopologyEdge `json:"outbound_edges,omitempty"`

	// Metadata contains additional node information.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EdgeDetail represents detailed information about an edge.
type EdgeDetail struct {
	*TopologyEdge

	// SourceNode is the full source node information.
	SourceNode *TopologyNode `json:"source_node,omitempty"`

	// TargetNode is the full target node information.
	TargetNode *TopologyNode `json:"target_node,omitempty"`

	// Entries are the detailed flow entries.
	Entries []*EdgeEntry `json:"entries,omitempty"`
}

// PolicyActionAllow is the policy action for allowed traffic.
const PolicyActionAllow = "allow"

// PolicyActionDeny is the policy action for denied traffic.
const PolicyActionDeny = "deny"

// SeverityInfo is informational severity.
const SeverityInfo = "info"

// SeverityLow is low severity.
const SeverityLow = "low"

// SeverityMedium is medium severity.
const SeverityMedium = "medium"

// SeverityHigh is high severity.
const SeverityHigh = "high"

// SeverityCritical is critical severity.
const SeverityCritical = "critical"

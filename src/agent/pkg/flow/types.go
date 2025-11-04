// Package flow provides network flow data collection and management
package flow

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// FlowEventType represents the type of flow event from eBPF
type FlowEventType uint8

const (
	FlowEventNew     FlowEventType = 0 // New connection established
	FlowEventUpdate  FlowEventType = 1 // Connection active/updated
	FlowEventClosed  FlowEventType = 2 // Connection closed
	FlowEventTimeout FlowEventType = 3 // Connection timeout
)

func (t FlowEventType) String() string {
	switch t {
	case FlowEventNew:
		return "NEW"
	case FlowEventUpdate:
		return "UPDATE"
	case FlowEventClosed:
		return "CLOSED"
	case FlowEventTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// FlowDirection represents the direction of network traffic
type FlowDirection uint8

const (
	FlowDirectionIngress FlowDirection = 0 // Ingress traffic
	FlowDirectionEgress  FlowDirection = 1 // Egress traffic
	FlowDirectionUnknown FlowDirection = 2 // Unknown direction
)

func (d FlowDirection) String() string {
	switch d {
	case FlowDirectionIngress:
		return "INGRESS"
	case FlowDirectionEgress:
		return "EGRESS"
	default:
		return "UNKNOWN"
	}
}

// FlowState represents the state of a flow
type FlowState uint8

const (
	FlowStateActive  FlowState = 0 // Active flow
	FlowStateClosed  FlowState = 1 // Closed flow
	FlowStateTimeout FlowState = 2 // Timeout flow
)

func (s FlowState) String() string {
	switch s {
	case FlowStateActive:
		return "ACTIVE"
	case FlowStateClosed:
		return "CLOSED"
	case FlowStateTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// PolicyAction represents the policy decision for a flow
type PolicyAction uint8

const (
	PolicyActionAllow PolicyAction = 0 // Allow traffic
	PolicyActionDeny  PolicyAction = 1 // Deny traffic
	PolicyActionLog   PolicyAction = 2 // Log traffic
)

func (a PolicyAction) String() string {
	switch a {
	case PolicyActionAllow:
		return "ALLOW"
	case PolicyActionDeny:
		return "DENY"
	case PolicyActionLog:
		return "LOG"
	default:
		return "UNKNOWN"
	}
}

// Protocol represents network protocol
type Protocol uint8

const (
	ProtocolTCP  Protocol = 6  // TCP protocol
	ProtocolUDP  Protocol = 17 // UDP protocol
	ProtocolICMP Protocol = 1  // ICMP protocol
)

func (p Protocol) String() string {
	switch p {
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	case ProtocolICMP:
		return "ICMP"
	default:
		return fmt.Sprintf("PROTO_%d", p)
	}
}

// FlowEvent represents a flow event from eBPF Ring Buffer (48 bytes)
// This matches the C struct flow_event in common_types.h
type FlowEvent struct {
	// 5-tuple identification
	SrcIP   uint32 // Source IP (network byte order)
	DstIP   uint32 // Destination IP (network byte order)
	SrcPort uint16 // Source port (network byte order)
	DstPort uint16 // Destination port (network byte order)

	// Packet metadata
	Protocol  Protocol      // Protocol (TCP/UDP/ICMP)
	EventType FlowEventType // Event type
	Direction FlowDirection // Traffic direction
	Padding   uint8         // Padding for alignment

	// Traffic statistics
	PacketCount uint64 // Total packets in this flow
	ByteCount   uint64 // Total bytes in this flow
	TimestampNS uint64 // Event timestamp in nanoseconds

	// Policy context
	PolicyID     uint32       // Matched policy/rule ID
	PolicyAction PolicyAction // Policy action
	State        FlowState    // Flow state
	Reserved     uint16       // Reserved for future use
}

// ParseFlowEvent parses a raw byte slice from Ring Buffer into FlowEvent
// Assumes little-endian byte order (x86_64)
func ParseFlowEvent(data []byte) (*FlowEvent, error) {
	if len(data) < 48 {
		return nil, fmt.Errorf("invalid flow event size: expected 48 bytes, got %d", len(data))
	}

	event := &FlowEvent{
		SrcIP:        binary.LittleEndian.Uint32(data[0:4]),
		DstIP:        binary.LittleEndian.Uint32(data[4:8]),
		SrcPort:      binary.BigEndian.Uint16(data[8:10]),  // Network byte order
		DstPort:      binary.BigEndian.Uint16(data[10:12]), // Network byte order
		Protocol:     Protocol(data[12]),
		EventType:    FlowEventType(data[13]),
		Direction:    FlowDirection(data[14]),
		Padding:      data[15],
		PacketCount:  binary.LittleEndian.Uint64(data[16:24]),
		ByteCount:    binary.LittleEndian.Uint64(data[24:32]),
		TimestampNS:  binary.LittleEndian.Uint64(data[32:40]),
		PolicyID:     binary.LittleEndian.Uint32(data[40:44]),
		PolicyAction: PolicyAction(data[44]),
		State:        FlowState(data[45]),
		Reserved:     binary.LittleEndian.Uint16(data[46:48]),
	}

	return event, nil
}

// Flow represents a complete network flow with enrichment
type Flow struct {
	// Identification
	ID        string    `json:"id"`         // Unique flow ID (hash of 5-tuple)
	SourceIP  string    `json:"source_ip"`  // Source IP address
	SourcePort uint16   `json:"source_port"` // Source port
	DestIP    string    `json:"dest_ip"`    // Destination IP address
	DestPort  uint16    `json:"dest_port"`  // Destination port
	Protocol  string    `json:"protocol"`   // Protocol name (TCP/UDP/ICMP)

	// Traffic Statistics
	PacketCount uint64 `json:"packet_count"` // Total packets
	ByteCount   uint64 `json:"byte_count"`   // Total bytes
	Duration    int64  `json:"duration_ms"`  // Duration in milliseconds

	// Timestamps
	StartTime time.Time  `json:"start_time"`           // Flow start time
	EndTime   *time.Time `json:"end_time,omitempty"`   // Flow end time (nil if active)
	LastSeen  time.Time  `json:"last_seen"`            // Last packet timestamp

	// Workload Enrichment
	SourceLabels map[string]string `json:"source_labels,omitempty"` // Source workload labels
	DestLabels   map[string]string `json:"dest_labels,omitempty"`   // Destination workload labels

	// Policy Context
	PolicyID     uint32 `json:"policy_id,omitempty"`     // Matched policy ID
	PolicyAction string `json:"policy_action"`           // Policy action (ALLOW/DENY/LOG)

	// State
	State     string `json:"state"`     // Flow state (ACTIVE/CLOSED/TIMEOUT)
	Direction string `json:"direction"` // Traffic direction (INGRESS/EGRESS)
	EventType string `json:"event_type"` // Last event type
}

// FlowKey generates a unique key for the flow based on 5-tuple
func FlowKey(srcIP, dstIP uint32, srcPort, dstPort uint16, protocol uint8) string {
	return fmt.Sprintf("%d-%d-%d-%d-%d", srcIP, dstIP, srcPort, dstPort, protocol)
}

// ToFlow converts a FlowEvent to a Flow structure with enrichment
func (e *FlowEvent) ToFlow() *Flow {
	srcIP := make(net.IP, 4)
	binary.LittleEndian.PutUint32(srcIP, e.SrcIP)

	dstIP := make(net.IP, 4)
	binary.LittleEndian.PutUint32(dstIP, e.DstIP)

	flowID := FlowKey(e.SrcIP, e.DstIP, e.SrcPort, e.DstPort, uint8(e.Protocol))
	timestamp := time.Unix(0, int64(e.TimestampNS))

	flow := &Flow{
		ID:           flowID,
		SourceIP:     srcIP.String(),
		SourcePort:   e.SrcPort,
		DestIP:       dstIP.String(),
		DestPort:     e.DstPort,
		Protocol:     e.Protocol.String(),
		PacketCount:  e.PacketCount,
		ByteCount:    e.ByteCount,
		StartTime:    timestamp,
		LastSeen:     timestamp,
		PolicyID:     e.PolicyID,
		PolicyAction: e.PolicyAction.String(),
		State:        e.State.String(),
		Direction:    e.Direction.String(),
		EventType:    e.EventType.String(),
		SourceLabels: make(map[string]string),
		DestLabels:   make(map[string]string),
	}

	// Set end time if flow is closed
	if e.EventType == FlowEventClosed || e.State == FlowStateClosed {
		flow.EndTime = &timestamp
	}

	return flow
}

// FlowQuery represents query parameters for filtering flows
type FlowQuery struct {
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"` // Start of time range
	EndTime   *time.Time `json:"end_time,omitempty"`   // End of time range

	// Flow filters
	SourceIP   *string            `json:"source_ip,omitempty"`   // Filter by source IP
	DestIP     *string            `json:"dest_ip,omitempty"`     // Filter by destination IP
	Protocol   *string            `json:"protocol,omitempty"`    // Filter by protocol
	State      *string            `json:"state,omitempty"`       // Filter by state
	Direction  *string            `json:"direction,omitempty"`   // Filter by direction
	PolicyAction *string          `json:"policy_action,omitempty"` // Filter by policy action

	// Label filters
	SourceLabels map[string]string `json:"source_labels,omitempty"` // Filter by source labels
	DestLabels   map[string]string `json:"dest_labels,omitempty"`   // Filter by destination labels

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Max number of results (default: 100)
	Offset int `json:"offset,omitempty"` // Offset for pagination

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // Field to sort by (default: start_time)
	SortOrder string `json:"sort_order,omitempty"` // Sort order: asc/desc (default: desc)
}

// FlowSummary represents aggregated flow statistics
type FlowSummary struct {
	TotalFlows      int64   `json:"total_flows"`      // Total number of flows
	ActiveFlows     int64   `json:"active_flows"`     // Number of active flows
	ClosedFlows     int64   `json:"closed_flows"`     // Number of closed flows
	TotalPackets    uint64  `json:"total_packets"`    // Total packets across all flows
	TotalBytes      uint64  `json:"total_bytes"`      // Total bytes across all flows
	AllowedFlows    int64   `json:"allowed_flows"`    // Number of allowed flows
	DeniedFlows     int64   `json:"denied_flows"`     // Number of denied flows
	TopProtocols    []ProtocolStats `json:"top_protocols"`    // Top protocols by flow count
	TopSourceIPs    []IPStats       `json:"top_source_ips"`   // Top source IPs by flow count
	TopDestIPs      []IPStats       `json:"top_dest_ips"`     // Top destination IPs by flow count
}

// ProtocolStats represents protocol statistics
type ProtocolStats struct {
	Protocol   string `json:"protocol"`    // Protocol name
	FlowCount  int64  `json:"flow_count"`  // Number of flows
	PacketCount uint64 `json:"packet_count"` // Total packets
	ByteCount  uint64 `json:"byte_count"`  // Total bytes
}

// IPStats represents IP address statistics
type IPStats struct {
	IP         string `json:"ip"`          // IP address
	FlowCount  int64  `json:"flow_count"`  // Number of flows
	PacketCount uint64 `json:"packet_count"` // Total packets
	ByteCount  uint64 `json:"byte_count"`  // Total bytes
}

// Dependency represents a dependency between two workloads
type Dependency struct {
	SourceLabels map[string]string `json:"source_labels"` // Source workload labels
	DestLabels   map[string]string `json:"dest_labels"`   // Destination workload labels
	FlowCount    int64             `json:"flow_count"`    // Number of flows
	PacketCount  uint64            `json:"packet_count"`  // Total packets
	ByteCount    uint64            `json:"byte_count"`    // Total bytes
	Protocols    []string          `json:"protocols"`     // Protocols used
	LastSeen     time.Time         `json:"last_seen"`     // Last activity timestamp
}

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

// FlowEvent represents a flow event from eBPF Ring Buffer (IPv4/IPv6 support)
// This matches the C struct flow_event in common_types.h (84 bytes total)
type FlowEvent struct {
	// 5-tuple identification (36 bytes)
	// IPv4 addresses are stored as IPv4-mapped IPv6 (::ffff:a.b.c.d)
	SrcIP   [4]uint32 // Source IP address (128 bits)
	DstIP   [4]uint32 // Destination IP address (128 bits)
	SrcPort uint16    // Source port (network byte order)
	DstPort uint16    // Destination port (network byte order)

	// Packet metadata (8 bytes)
	Protocol  Protocol      // Protocol (TCP/UDP/ICMP)
	EventType FlowEventType // Event type
	Direction FlowDirection // Traffic direction
	IPVersion uint8         // IP version (4 or 6)
	VlanID    uint16        // VLAN ID (0 = no VLAN)
	TcpFlags  uint8         // TCP flags (SYN, FIN, RST, etc.)
	Flags     uint8         // Connection flags (CONN_FLAG_*)

	// Traffic statistics (24 bytes)
	PacketCount uint64 // Total packets in this flow
	ByteCount   uint64 // Total bytes in this flow
	TimestampNS uint64 // Event timestamp in nanoseconds

	// Enhanced TCP tracking (12 bytes)
	TcpSeq     uint32 // TCP sequence number
	TcpAck     uint32 // TCP acknowledgment number
	TcpWindow  uint16 // TCP window size
	TcpRetrans uint8  // Retransmission count
	TcpState   uint8  // TCP state

	// Policy context (8 bytes)
	PolicyID     uint32       // Matched policy/rule ID
	PolicyAction PolicyAction // Policy action
	State        FlowState    // Flow state
	Reserved     uint16       // Reserved for future use

	// Process context (92 bytes) - Issue #47, #48
	// These fields are populated by eBPF looking up process_info_map
	ProcessName     [16]byte // Process command name (from process cache)
	PID             uint32   // Process ID (0 if not available)
	ContainerID     [64]byte // Container ID (from process cache)
	ProcessExecTime uint64   // Process execution timestamp (from process cache)
}

// ParseFlowEvent parses a raw byte slice from Ring Buffer into FlowEvent
// Assumes little-endian byte order (x86_64)
// Structure size: 36 (5-tuple) + 8 (metadata) + 24 (stats) + 12 (TCP) + 8 (policy) + 92 (process) = 180 bytes
func ParseFlowEvent(data []byte) (*FlowEvent, error) {
	expectedSize := 180 // Total size with process context fields (Issue #47/#48)
	if len(data) < expectedSize {
		return nil, fmt.Errorf("invalid flow event size: expected at least %d bytes, got %d", expectedSize, len(data))
	}

	event := &FlowEvent{
		// 5-tuple identification (36 bytes)
		// Source IP: 4 x 32-bit words = 16 bytes (offset 0-15)
		SrcIP: [4]uint32{
			binary.LittleEndian.Uint32(data[0:4]),
			binary.LittleEndian.Uint32(data[4:8]),
			binary.LittleEndian.Uint32(data[8:12]),
			binary.LittleEndian.Uint32(data[12:16]),
		},
		// Destination IP: 4 x 32-bit words = 16 bytes (offset 16-31)
		DstIP: [4]uint32{
			binary.LittleEndian.Uint32(data[16:20]),
			binary.LittleEndian.Uint32(data[20:24]),
			binary.LittleEndian.Uint32(data[24:28]),
			binary.LittleEndian.Uint32(data[28:32]),
		},
		// Ports: 2 + 2 = 4 bytes (offset 32-35)
		SrcPort: binary.BigEndian.Uint16(data[32:34]), // Network byte order
		DstPort: binary.BigEndian.Uint16(data[34:36]), // Network byte order

		// Packet metadata (8 bytes, offset 36-43)
		Protocol:  Protocol(data[36]),
		EventType: FlowEventType(data[37]),
		Direction: FlowDirection(data[38]),
		IPVersion: data[39],
		VlanID:    binary.LittleEndian.Uint16(data[40:42]),
		TcpFlags:  data[42],
		Flags:     data[43],

		// Traffic statistics (24 bytes, offset 44-67)
		PacketCount: binary.LittleEndian.Uint64(data[44:52]),
		ByteCount:   binary.LittleEndian.Uint64(data[52:60]),
		TimestampNS: binary.LittleEndian.Uint64(data[60:68]),

		// Enhanced TCP tracking (12 bytes, offset 68-79)
		TcpSeq:     binary.LittleEndian.Uint32(data[68:72]),
		TcpAck:     binary.LittleEndian.Uint32(data[72:76]),
		TcpWindow:  binary.LittleEndian.Uint16(data[76:78]),
		TcpRetrans: data[78],
		TcpState:   data[79],

		// Policy context (8 bytes, offset 80-87)
		PolicyID:     binary.LittleEndian.Uint32(data[80:84]),
		PolicyAction: PolicyAction(data[84]),
		State:        FlowState(data[85]),
		Reserved:     binary.LittleEndian.Uint16(data[86:88]),
	}

	// Parse process context fields (92 bytes, offset 88-179) - Issue #47/#48
	copy(event.ProcessName[:], data[88:104])          // 16 bytes
	event.PID = binary.LittleEndian.Uint32(data[104:108]) // 4 bytes
	copy(event.ContainerID[:], data[108:172])         // 64 bytes
	event.ProcessExecTime = binary.LittleEndian.Uint64(data[172:180]) // 8 bytes

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

	// Process Context (Issue #47/#48)
	ProcessName     string `json:"process_name,omitempty"`      // Process command name
	ProcessPID      uint32 `json:"process_pid,omitempty"`       // Process ID
	ProcessPath     string `json:"process_path,omitempty"`      // Full executable path (from ProcessMonitor)
	ContainerID     string `json:"container_id,omitempty"`      // Container ID
	ProcessExecTime uint64 `json:"process_exec_time,omitempty"` // Process start timestamp

	// State
	State     string `json:"state"`     // Flow state (ACTIVE/CLOSED/TIMEOUT)
	Direction string `json:"direction"` // Traffic direction (INGRESS/EGRESS)
	EventType string `json:"event_type"` // Last event type
}

// FlowKey generates a unique key for the flow based on 5-tuple (IPv4/IPv6 support)
func FlowKey(srcIP, dstIP [4]uint32, srcPort, dstPort uint16, protocol uint8) string {
	// Use all 4 uint32 words for both IPv4 and IPv6
	return fmt.Sprintf("%08x%08x%08x%08x-%08x%08x%08x%08x-%d-%d-%d",
		srcIP[0], srcIP[1], srcIP[2], srcIP[3],
		dstIP[0], dstIP[1], dstIP[2], dstIP[3],
		srcPort, dstPort, protocol)
}

// ipv6ToNetIP converts [4]uint32 IPv6 address to net.IP
// Handles both native IPv6 and IPv4-mapped IPv6 addresses
func ipv6ToNetIP(ipv6 [4]uint32, ipVersion uint8) net.IP {
	// Check if this is IPv4-mapped IPv6 (::ffff:a.b.c.d)
	// IPv4-mapped: [0, 0, 0xffff0000, ipv4_addr] in network byte order
	if ipVersion == 4 || (ipv6[0] == 0 && ipv6[1] == 0 && ipv6[2] == 0x0000ffff) {
		// Extract IPv4 address from last 32 bits
		ipv4 := make(net.IP, 4)
		binary.LittleEndian.PutUint32(ipv4, ipv6[3])
		return ipv4
	}

	// Native IPv6 address
	ipv6Bytes := make(net.IP, 16)
	binary.LittleEndian.PutUint32(ipv6Bytes[0:4], ipv6[0])
	binary.LittleEndian.PutUint32(ipv6Bytes[4:8], ipv6[1])
	binary.LittleEndian.PutUint32(ipv6Bytes[8:12], ipv6[2])
	binary.LittleEndian.PutUint32(ipv6Bytes[12:16], ipv6[3])
	return ipv6Bytes
}

// ToFlow converts a FlowEvent to a Flow structure with enrichment (IPv4/IPv6 support)
func (e *FlowEvent) ToFlow() *Flow {
	// Convert IP addresses based on IP version
	srcIP := ipv6ToNetIP(e.SrcIP, e.IPVersion)
	dstIP := ipv6ToNetIP(e.DstIP, e.IPVersion)

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

	// Convert process context fields (Issue #47/#48)
	flow.ProcessName = nullTerminatedByteArrayToString(e.ProcessName[:])
	flow.ProcessPID = e.PID
	flow.ContainerID = nullTerminatedByteArrayToString(e.ContainerID[:])
	flow.ProcessExecTime = e.ProcessExecTime
	// Note: ProcessPath will be enriched by FlowCollector via ProcessMonitor lookup (Issue #49)

	// Set end time if flow is closed
	if e.EventType == FlowEventClosed || e.State == FlowStateClosed {
		flow.EndTime = &timestamp
	}

	return flow
}

// nullTerminatedByteArrayToString converts null-terminated byte array to Go string
func nullTerminatedByteArrayToString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
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

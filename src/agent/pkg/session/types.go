package session

import "time"

// SessionTimeoutConfig holds timeout configuration for session management
type SessionTimeoutConfig struct {
	// TCPIdleTimeout is the maximum idle time for TCP sessions before cleanup
	TCPIdleTimeout time.Duration

	// TCPCloseTimeout is the time to keep closed TCP sessions before removal
	TCPCloseTimeout time.Duration

	// UDPIdleTimeout is the maximum idle time for UDP sessions before cleanup
	UDPIdleTimeout time.Duration

	// MaxSessionDuration is the maximum duration for any session
	MaxSessionDuration time.Duration

	// ScanInterval is how often to scan the session map for timeouts
	ScanInterval time.Duration
}

// SessionTimeoutStats tracks timeout manager statistics
type SessionTimeoutStats struct {
	// TotalScans is the total number of scans performed
	TotalScans uint64 `json:"total_scans"`

	// TotalSessionsScanned is the total number of sessions scanned
	TotalSessionsScanned uint64 `json:"total_sessions_scanned"`

	// TotalTimedOut is the total number of sessions that timed out
	TotalTimedOut uint64 `json:"total_timed_out"`

	// TCPIdleTimeouts is the count of TCP idle timeouts
	TCPIdleTimeouts uint64 `json:"tcp_idle_timeouts"`

	// TCPCloseTimeouts is the count of TCP close timeouts
	TCPCloseTimeouts uint64 `json:"tcp_close_timeouts"`

	// UDPIdleTimeouts is the count of UDP idle timeouts
	UDPIdleTimeouts uint64 `json:"udp_idle_timeouts"`

	// MaxDurationTimeouts is the count of max duration timeouts
	MaxDurationTimeouts uint64 `json:"max_duration_timeouts"`

	// LastScanTime is the timestamp of the last scan
	LastScanTime time.Time `json:"last_scan_time"`

	// LastScanDuration is the duration of the last scan
	LastScanDuration time.Duration `json:"last_scan_duration_ms"`

	// ScanErrors is the count of scan errors
	ScanErrors uint64 `json:"scan_errors"`
}

// FlowKey represents the 5-tuple flow key (matches kernel struct flow_key)
// Supports both IPv4 and IPv6 (IPv4-mapped IPv6 format)
type FlowKey struct {
	SrcIP     [4]uint32 // Source IP address (128 bits)
	DstIP     [4]uint32 // Destination IP address (128 bits)
	SrcPort   uint16    // Source port
	DstPort   uint16    // Destination port
	Protocol  uint8     // Protocol (TCP/UDP/ICMP)
	IPVersion uint8     // IP version (4 or 6)
	VlanID    uint16    // VLAN ID (0 = no VLAN)
}

// SessionValue represents the session value (matches kernel struct session_value)
type SessionValue struct {
	CreatedTS       uint64 // Session creation timestamp (nanoseconds)
	LastSeenTS      uint64 // Last packet timestamp (nanoseconds)
	PacketsToServer uint64 // Packets from client to server
	PacketsToClient uint64 // Packets from server to client
	BytesToServer   uint64 // Bytes from client to server
	BytesToClient   uint64 // Bytes from server to client

	// Enhanced TCP tracking
	TCPSeqClient  uint32 // Last TCP sequence number from client
	TCPSeqServer  uint32 // Last TCP sequence number from server
	TCPAckClient  uint32 // Last TCP acknowledgment from client
	TCPAckServer  uint32 // Last TCP acknowledgment from server
	TCPWindowSize uint16 // TCP window size (last seen)
	TCPRetrans    uint8  // TCP retransmission count

	State        uint8 // Session state
	TCPState     uint8 // TCP state machine
	PolicyAction uint8 // Matched policy action
	Flags        uint8 // Connection flags (CONN_FLAG_*)
	Pad          uint8 // Padding for alignment
}

// TCP state constants (must match common_types.h)
const (
	TCPStateClosed    uint8 = 0
	TCPStateSynSent   uint8 = 1
	TCPStateSynRecv   uint8 = 2
	TCPStateEstablished uint8 = 3
	TCPStateFinWait1  uint8 = 4
	TCPStateFinWait2  uint8 = 5
	TCPStateCloseWait uint8 = 6
	TCPStateClosing   uint8 = 7
	TCPStateLastAck   uint8 = 8
	TCPStateTimeWait  uint8 = 9
)

// Protocol constants
const (
	ProtocolTCP uint8 = 6
	ProtocolUDP uint8 = 17
)

// isTCPClosing checks if a TCP state indicates the connection is closing or closed
func isTCPClosing(tcpState uint8) bool {
	return tcpState == TCPStateFinWait1 ||
		tcpState == TCPStateFinWait2 ||
		tcpState == TCPStateCloseWait ||
		tcpState == TCPStateClosing ||
		tcpState == TCPStateLastAck ||
		tcpState == TCPStateTimeWait ||
		tcpState == TCPStateClosed
}

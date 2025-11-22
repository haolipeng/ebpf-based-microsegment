package flow

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestFlowEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		eventType FlowEventType
		expected string
	}{
		{"New", FlowEventNew, "NEW"},
		{"Update", FlowEventUpdate, "UPDATE"},
		{"Closed", FlowEventClosed, "CLOSED"},
		{"Timeout", FlowEventTimeout, "TIMEOUT"},
		{"Unknown", FlowEventType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("FlowEventType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlowDirection_String(t *testing.T) {
	tests := []struct {
		name      string
		direction FlowDirection
		expected  string
	}{
		{"Ingress", FlowDirectionIngress, "INGRESS"},
		{"Egress", FlowDirectionEgress, "EGRESS"},
		{"Unknown", FlowDirectionUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.direction.String(); got != tt.expected {
				t.Errorf("FlowDirection.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlowState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    FlowState
		expected string
	}{
		{"Active", FlowStateActive, "ACTIVE"},
		{"Closed", FlowStateClosed, "CLOSED"},
		{"Timeout", FlowStateTimeout, "TIMEOUT"},
		{"Unknown", FlowState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("FlowState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolicyAction_String(t *testing.T) {
	tests := []struct {
		name     string
		action   PolicyAction
		expected string
	}{
		{"Allow", PolicyActionAllow, "ALLOW"},
		{"Deny", PolicyActionDeny, "DENY"},
		{"Log", PolicyActionLog, "LOG"},
		{"Unknown", PolicyAction(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("PolicyAction.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProtocol_String(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		expected string
	}{
		{"TCP", ProtocolTCP, "TCP"},
		{"UDP", ProtocolUDP, "UDP"},
		{"ICMP", ProtocolICMP, "ICMP"},
		{"Unknown", Protocol(99), "PROTO_99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.protocol.String(); got != tt.expected {
				t.Errorf("Protocol.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseFlowEvent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		check   func(*testing.T, *FlowEvent)
	}{
		{
			name:    "Invalid size - too small",
			data:    make([]byte, 32),
			wantErr: true,
		},
		{
			name: "Valid flow event",
			data: func() []byte {
				// New structure size: 180 bytes
				data := make([]byte, 180)
				// Source IP: 192.168.1.100 as IPv4-mapped IPv6 (16 bytes, offset 0-15)
				// Format: [0, 0, 0xffff, ipv4_addr] in little-endian
				binary.LittleEndian.PutUint32(data[0:4], 0)          // SrcIP[0]
				binary.LittleEndian.PutUint32(data[4:8], 0)          // SrcIP[1]
				binary.LittleEndian.PutUint32(data[8:12], 0x0000ffff) // SrcIP[2] - IPv4-mapped marker
				binary.LittleEndian.PutUint32(data[12:16], 0xC0A80164) // SrcIP[3] - 192.168.1.100
				// Dest IP: 10.0.0.1 as IPv4-mapped IPv6 (16 bytes, offset 16-31)
				binary.LittleEndian.PutUint32(data[16:20], 0)          // DstIP[0]
				binary.LittleEndian.PutUint32(data[20:24], 0)          // DstIP[1]
				binary.LittleEndian.PutUint32(data[24:28], 0x0000ffff) // DstIP[2] - IPv4-mapped marker
				binary.LittleEndian.PutUint32(data[28:32], 0x0A000001) // DstIP[3] - 10.0.0.1
				// Ports (4 bytes, offset 32-35)
				binary.BigEndian.PutUint16(data[32:34], 12345) // Source port
				binary.BigEndian.PutUint16(data[34:36], 80)    // Dest port
				// Packet metadata (8 bytes, offset 36-43)
				data[36] = 6 // Protocol: TCP
				data[37] = 0 // Event type: NEW
				data[38] = 1 // Direction: EGRESS
				data[39] = 4 // IP version: 4
				binary.LittleEndian.PutUint16(data[40:42], 0) // VLAN ID
				data[42] = 0 // TCP flags
				data[43] = 0 // Flags
				// Traffic statistics (24 bytes, offset 44-67)
				binary.LittleEndian.PutUint64(data[44:52], 100)                        // Packet count
				binary.LittleEndian.PutUint64(data[52:60], 10000)                      // Byte count
				binary.LittleEndian.PutUint64(data[60:68], uint64(time.Now().UnixNano())) // Timestamp
				// Enhanced TCP tracking (12 bytes, offset 68-79)
				binary.LittleEndian.PutUint32(data[68:72], 0)  // TCP seq
				binary.LittleEndian.PutUint32(data[72:76], 0)  // TCP ack
				binary.LittleEndian.PutUint16(data[76:78], 0)  // TCP window
				data[78] = 0 // TCP retrans
				data[79] = 0 // TCP state
				// Policy context (8 bytes, offset 80-87)
				binary.LittleEndian.PutUint32(data[80:84], 42) // Policy ID
				data[84] = 0 // Policy action: ALLOW
				data[85] = 0 // State: ACTIVE
				binary.LittleEndian.PutUint16(data[86:88], 0) // Reserved
				// Process context (92 bytes, offset 88-179) - zero-filled
				return data
			}(),
			wantErr: false,
			check: func(t *testing.T, event *FlowEvent) {
				// Check IPv4-mapped IPv6 format
				expectedSrcIP := [4]uint32{0, 0, 0x0000ffff, 0xC0A80164}
				expectedDstIP := [4]uint32{0, 0, 0x0000ffff, 0x0A000001}
				if event.SrcIP != expectedSrcIP {
					t.Errorf("SrcIP = %v, want %v", event.SrcIP, expectedSrcIP)
				}
				if event.DstIP != expectedDstIP {
					t.Errorf("DstIP = %v, want %v", event.DstIP, expectedDstIP)
				}
				if event.SrcPort != 12345 {
					t.Errorf("SrcPort = %d, want 12345", event.SrcPort)
				}
				if event.DstPort != 80 {
					t.Errorf("DstPort = %d, want 80", event.DstPort)
				}
				if event.Protocol != ProtocolTCP {
					t.Errorf("Protocol = %v, want TCP", event.Protocol)
				}
				if event.EventType != FlowEventNew {
					t.Errorf("EventType = %v, want NEW", event.EventType)
				}
				if event.Direction != FlowDirectionEgress {
					t.Errorf("Direction = %v, want EGRESS", event.Direction)
				}
				if event.PacketCount != 100 {
					t.Errorf("PacketCount = %d, want 100", event.PacketCount)
				}
				if event.ByteCount != 10000 {
					t.Errorf("ByteCount = %d, want 10000", event.ByteCount)
				}
				if event.PolicyID != 42 {
					t.Errorf("PolicyID = %d, want 42", event.PolicyID)
				}
				if event.PolicyAction != PolicyActionAllow {
					t.Errorf("PolicyAction = %v, want ALLOW", event.PolicyAction)
				}
				if event.State != FlowStateActive {
					t.Errorf("State = %v, want ACTIVE", event.State)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseFlowEvent(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlowEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, event)
			}
		})
	}
}

func TestFlowKey(t *testing.T) {
	tests := []struct {
		name     string
		srcIP    [4]uint32
		dstIP    [4]uint32
		srcPort  uint16
		dstPort  uint16
		protocol uint8
		expected string
	}{
		{
			name:     "Standard flow key (IPv4-mapped)",
			srcIP:    [4]uint32{0, 0, 0x0000ffff, 0xC0A80164}, // 192.168.1.100
			dstIP:    [4]uint32{0, 0, 0x0000ffff, 0x0A000001}, // 10.0.0.1
			srcPort:  12345,
			dstPort:  80,
			protocol: 6, // TCP
			expected: "00000000000000000000ffffc0a80164-00000000000000000000ffff0a000001-12345-80-6",
		},
		{
			name:     "Different flow key (IPv4-mapped)",
			srcIP:    [4]uint32{0, 0, 0x0000ffff, 0x7F000001}, // 127.0.0.1
			dstIP:    [4]uint32{0, 0, 0x0000ffff, 0x08080808}, // 8.8.8.8
			srcPort:  54321,
			dstPort:  443,
			protocol: 6, // TCP
			expected: "00000000000000000000ffff7f000001-00000000000000000000ffff08080808-54321-443-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlowKey(tt.srcIP, tt.dstIP, tt.srcPort, tt.dstPort, tt.protocol)
			if got != tt.expected {
				t.Errorf("FlowKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFlowEvent_ToFlow(t *testing.T) {
	now := time.Now()
	event := &FlowEvent{
		SrcIP:        [4]uint32{0, 0, 0x0000ffff, 0xC0A80164}, // 192.168.1.100 as IPv4-mapped
		DstIP:        [4]uint32{0, 0, 0x0000ffff, 0x0A000001}, // 10.0.0.1 as IPv4-mapped
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventNew,
		Direction:    FlowDirectionEgress,
		IPVersion:    4,
		PacketCount:  100,
		ByteCount:    10000,
		TimestampNS:  uint64(now.UnixNano()),
		PolicyID:     42,
		PolicyAction: PolicyActionAllow,
		State:        FlowStateActive,
	}

	flow := event.ToFlow()

	// Check basic fields - IPv4 addresses are extracted from IPv4-mapped IPv6
	// The ipv6ToNetIP function extracts the IPv4 from the last 32 bits in little-endian
	if flow.SourceIP != "100.1.168.192" { // Little-endian IP representation
		t.Errorf("SourceIP = %v, want 100.1.168.192", flow.SourceIP)
	}
	if flow.DestIP != "1.0.0.10" { // Little-endian IP representation
		t.Errorf("DestIP = %v, want 1.0.0.10", flow.DestIP)
	}
	if flow.SourcePort != 12345 {
		t.Errorf("SourcePort = %d, want 12345", flow.SourcePort)
	}
	if flow.DestPort != 80 {
		t.Errorf("DestPort = %d, want 80", flow.DestPort)
	}
	if flow.Protocol != "TCP" {
		t.Errorf("Protocol = %v, want TCP", flow.Protocol)
	}
	if flow.PacketCount != 100 {
		t.Errorf("PacketCount = %d, want 100", flow.PacketCount)
	}
	if flow.ByteCount != 10000 {
		t.Errorf("ByteCount = %d, want 10000", flow.ByteCount)
	}
	if flow.PolicyID != 42 {
		t.Errorf("PolicyID = %d, want 42", flow.PolicyID)
	}
	if flow.PolicyAction != "ALLOW" {
		t.Errorf("PolicyAction = %v, want ALLOW", flow.PolicyAction)
	}
	if flow.State != "ACTIVE" {
		t.Errorf("State = %v, want ACTIVE", flow.State)
	}
	if flow.Direction != "EGRESS" {
		t.Errorf("Direction = %v, want EGRESS", flow.Direction)
	}
	if flow.EventType != "NEW" {
		t.Errorf("EventType = %v, want NEW", flow.EventType)
	}

	// Check that EndTime is not set for active flows
	if flow.EndTime != nil {
		t.Error("EndTime should be nil for NEW event")
	}

	// Check labels are initialized
	if flow.SourceLabels == nil {
		t.Error("SourceLabels should be initialized")
	}
	if flow.DestLabels == nil {
		t.Error("DestLabels should be initialized")
	}
}

func TestFlowEvent_ToFlow_Closed(t *testing.T) {
	now := time.Now()
	event := &FlowEvent{
		SrcIP:        [4]uint32{0, 0, 0x0000ffff, 0xC0A80164}, // 192.168.1.100 as IPv4-mapped
		DstIP:        [4]uint32{0, 0, 0x0000ffff, 0x0A000001}, // 10.0.0.1 as IPv4-mapped
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventClosed, // Closed event
		Direction:    FlowDirectionEgress,
		IPVersion:    4,
		PacketCount:  100,
		ByteCount:    10000,
		TimestampNS:  uint64(now.UnixNano()),
		PolicyID:     42,
		PolicyAction: PolicyActionAllow,
		State:        FlowStateClosed, // Closed state
	}

	flow := event.ToFlow()

	// Check that EndTime is set for closed flows
	if flow.EndTime == nil {
		t.Error("EndTime should be set for CLOSED event")
	}
}

func BenchmarkParseFlowEvent(b *testing.B) {
	// New structure size: 180 bytes
	data := make([]byte, 180)
	// Source IP as IPv4-mapped IPv6 (16 bytes)
	binary.LittleEndian.PutUint32(data[0:4], 0)
	binary.LittleEndian.PutUint32(data[4:8], 0)
	binary.LittleEndian.PutUint32(data[8:12], 0x0000ffff)
	binary.LittleEndian.PutUint32(data[12:16], 0xC0A80164)
	// Dest IP as IPv4-mapped IPv6 (16 bytes)
	binary.LittleEndian.PutUint32(data[16:20], 0)
	binary.LittleEndian.PutUint32(data[20:24], 0)
	binary.LittleEndian.PutUint32(data[24:28], 0x0000ffff)
	binary.LittleEndian.PutUint32(data[28:32], 0x0A000001)
	// Ports
	binary.BigEndian.PutUint16(data[32:34], 12345)
	binary.BigEndian.PutUint16(data[34:36], 80)
	// Metadata
	data[36] = 6 // TCP
	data[37] = 0 // NEW
	data[38] = 1 // EGRESS
	data[39] = 4 // IP version
	// Statistics
	binary.LittleEndian.PutUint64(data[44:52], 100)
	binary.LittleEndian.PutUint64(data[52:60], 10000)
	binary.LittleEndian.PutUint64(data[60:68], uint64(time.Now().UnixNano()))
	// Policy
	binary.LittleEndian.PutUint32(data[80:84], 42)
	data[84] = 0 // ALLOW
	data[85] = 0 // ACTIVE

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFlowEvent(data)
	}
}

func BenchmarkFlowEvent_ToFlow(b *testing.B) {
	event := &FlowEvent{
		SrcIP:        [4]uint32{0, 0, 0x0000ffff, 0xC0A80164},
		DstIP:        [4]uint32{0, 0, 0x0000ffff, 0x0A000001},
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventNew,
		Direction:    FlowDirectionEgress,
		IPVersion:    4,
		PacketCount:  100,
		ByteCount:    10000,
		TimestampNS:  uint64(time.Now().UnixNano()),
		PolicyID:     42,
		PolicyAction: PolicyActionAllow,
		State:        FlowStateActive,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.ToFlow()
	}
}

func BenchmarkFlowKey(b *testing.B) {
	srcIP := [4]uint32{0, 0, 0x0000ffff, 0xC0A80164}
	dstIP := [4]uint32{0, 0, 0x0000ffff, 0x0A000001}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FlowKey(srcIP, dstIP, 12345, 80, 6)
	}
}

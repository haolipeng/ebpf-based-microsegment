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
				data := make([]byte, 48)
				// Source IP: 192.168.1.100 (0xC0A80164 in little-endian)
				binary.LittleEndian.PutUint32(data[0:4], 0xC0A80164)
				// Dest IP: 10.0.0.1 (0x0A000001 in little-endian)
				binary.LittleEndian.PutUint32(data[4:8], 0x0A000001)
				// Source port: 12345 (network byte order)
				binary.BigEndian.PutUint16(data[8:10], 12345)
				// Dest port: 80 (network byte order)
				binary.BigEndian.PutUint16(data[10:12], 80)
				// Protocol: TCP (6)
				data[12] = 6
				// Event type: NEW (0)
				data[13] = 0
				// Direction: EGRESS (1)
				data[14] = 1
				// Padding
				data[15] = 0
				// Packet count: 100
				binary.LittleEndian.PutUint64(data[16:24], 100)
				// Byte count: 10000
				binary.LittleEndian.PutUint64(data[24:32], 10000)
				// Timestamp: current time in nanoseconds
				binary.LittleEndian.PutUint64(data[32:40], uint64(time.Now().UnixNano()))
				// Policy ID: 42
				binary.LittleEndian.PutUint32(data[40:44], 42)
				// Policy action: ALLOW (0)
				data[44] = 0
				// State: ACTIVE (0)
				data[45] = 0
				// Reserved
				binary.LittleEndian.PutUint16(data[46:48], 0)
				return data
			}(),
			wantErr: false,
			check: func(t *testing.T, event *FlowEvent) {
				if event.SrcIP != 0xC0A80164 {
					t.Errorf("SrcIP = 0x%X, want 0xC0A80164", event.SrcIP)
				}
				if event.DstIP != 0x0A000001 {
					t.Errorf("DstIP = 0x%X, want 0x0A000001", event.DstIP)
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
		srcIP    uint32
		dstIP    uint32
		srcPort  uint16
		dstPort  uint16
		protocol uint8
		expected string
	}{
		{
			name:     "Standard flow key",
			srcIP:    0xC0A80164, // 192.168.1.100
			dstIP:    0x0A000001, // 10.0.0.1
			srcPort:  12345,
			dstPort:  80,
			protocol: 6, // TCP
			expected: "3232235876-167772161-12345-80-6",
		},
		{
			name:     "Different flow key",
			srcIP:    0x7F000001, // 127.0.0.1
			dstIP:    0x08080808, // 8.8.8.8
			srcPort:  54321,
			dstPort:  443,
			protocol: 6, // TCP
			expected: "2130706433-134744072-54321-443-6",
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
		SrcIP:        0xC0A80164, // 192.168.1.100
		DstIP:        0x0A000001, // 10.0.0.1
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventNew,
		Direction:    FlowDirectionEgress,
		PacketCount:  100,
		ByteCount:    10000,
		TimestampNS:  uint64(now.UnixNano()),
		PolicyID:     42,
		PolicyAction: PolicyActionAllow,
		State:        FlowStateActive,
	}

	flow := event.ToFlow()

	// Check basic fields
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
		SrcIP:        0xC0A80164,
		DstIP:        0x0A000001,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventClosed, // Closed event
		Direction:    FlowDirectionEgress,
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
	data := make([]byte, 48)
	binary.LittleEndian.PutUint32(data[0:4], 0xC0A80164)
	binary.LittleEndian.PutUint32(data[4:8], 0x0A000001)
	binary.BigEndian.PutUint16(data[8:10], 12345)
	binary.BigEndian.PutUint16(data[10:12], 80)
	data[12] = 6  // TCP
	data[13] = 0  // NEW
	data[14] = 1  // EGRESS
	binary.LittleEndian.PutUint64(data[16:24], 100)
	binary.LittleEndian.PutUint64(data[24:32], 10000)
	binary.LittleEndian.PutUint64(data[32:40], uint64(time.Now().UnixNano()))
	binary.LittleEndian.PutUint32(data[40:44], 42)
	data[44] = 0  // ALLOW
	data[45] = 0  // ACTIVE

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFlowEvent(data)
	}
}

func BenchmarkFlowEvent_ToFlow(b *testing.B) {
	event := &FlowEvent{
		SrcIP:        0xC0A80164,
		DstIP:        0x0A000001,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     ProtocolTCP,
		EventType:    FlowEventNew,
		Direction:    FlowDirectionEgress,
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FlowKey(0xC0A80164, 0x0A000001, 12345, 80, 6)
	}
}

package reporter

import (
	"net"
	"testing"
	"time"

	commonpb "github.com/ebpf-microsegment/src/proto/common"
	"github.com/ebpf-microsegment/src/agent/pkg/flow"
)

// Test_NewGRPCReporter tests constructor
func Test_NewGRPCReporter(t *testing.T) {
	tests := []struct {
		name          string
		serverAddr    string
		agentID       string
		batchSize     int
		expectedBatch int
	}{
		{
			name:          "with custom batch size",
			serverAddr:    "localhost:9090",
			agentID:       "agent-1",
			batchSize:     50,
			expectedBatch: 50,
		},
		{
			name:          "with zero batch size (defaults to 100)",
			serverAddr:    "localhost:9090",
			agentID:       "agent-2",
			batchSize:     0,
			expectedBatch: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewGRPCReporter(tt.serverAddr, tt.agentID, tt.batchSize)

			if reporter == nil {
				t.Fatal("Expected non-nil reporter")
			}
			if reporter.serverAddr != tt.serverAddr {
				t.Errorf("Expected serverAddr=%s, got %s", tt.serverAddr, reporter.serverAddr)
			}
			if reporter.agentID != tt.agentID {
				t.Errorf("Expected agentID=%s, got %s", tt.agentID, reporter.agentID)
			}
			if reporter.batchSize != tt.expectedBatch {
				t.Errorf("Expected batchSize=%d, got %d", tt.expectedBatch, reporter.batchSize)
			}
			if cap(reporter.batchQueue) != tt.expectedBatch*2 {
				t.Errorf("Expected queue capacity=%d, got %d", tt.expectedBatch*2, cap(reporter.batchQueue))
			}
		})
	}
}

// Test_ipToUint32 tests IP string to uint32 conversion
func Test_ipToUint32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "valid IP 192.168.1.1",
			ip:       "192.168.1.1",
			expected: 0xC0A80101, // 192<<24 | 168<<16 | 1<<8 | 1
		},
		{
			name:     "valid IP 10.0.0.1",
			ip:       "10.0.0.1",
			expected: 0x0A000001, // 10<<24 | 0<<16 | 0<<8 | 1
		},
		{
			name:     "localhost 127.0.0.1",
			ip:       "127.0.0.1",
			expected: 0x7F000001, // 127<<24 | 0<<16 | 0<<8 | 1
		},
		{
			name:     "invalid IP",
			ip:       "not-an-ip",
			expected: 0,
		},
		{
			name:     "empty string",
			ip:       "",
			expected: 0,
		},
		{
			name:     "IPv6 (unsupported)",
			ip:       "2001:db8::1",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToUint32(tt.ip)
			if result != tt.expected {
				t.Errorf("Expected 0x%08X, got 0x%08X", tt.expected, result)
			}
		})
	}
}

// Test_protocolStringToEnum tests protocol string conversion
func Test_protocolStringToEnum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected commonpb.Protocol
	}{
		{"TCP uppercase", "TCP", commonpb.Protocol_PROTOCOL_TCP},
		{"TCP lowercase", "tcp", commonpb.Protocol_PROTOCOL_TCP},
		{"UDP uppercase", "UDP", commonpb.Protocol_PROTOCOL_UDP},
		{"UDP lowercase", "udp", commonpb.Protocol_PROTOCOL_UDP},
		{"ICMP uppercase", "ICMP", commonpb.Protocol_PROTOCOL_ICMP},
		{"ICMP lowercase", "icmp", commonpb.Protocol_PROTOCOL_ICMP},
		{"ANY", "ANY", commonpb.Protocol_PROTOCOL_ANY},
		{"unknown protocol", "xyz", commonpb.Protocol_PROTOCOL_UNKNOWN},
		{"empty string", "", commonpb.Protocol_PROTOCOL_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocolStringToEnum(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test_eventTypeStringToEnum tests event type string conversion
func Test_eventTypeStringToEnum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected commonpb.FlowEventType
	}{
		{"NEW uppercase", "NEW", commonpb.FlowEventType_EVENT_NEW},
		{"NEW lowercase", "new", commonpb.FlowEventType_EVENT_NEW},
		{"UPDATE", "UPDATE", commonpb.FlowEventType_EVENT_UPDATE},
		{"CLOSED", "CLOSED", commonpb.FlowEventType_EVENT_CLOSED},
		{"TIMEOUT", "TIMEOUT", commonpb.FlowEventType_EVENT_TIMEOUT},
		{"unknown", "xyz", commonpb.FlowEventType_EVENT_UNKNOWN},
		{"empty", "", commonpb.FlowEventType_EVENT_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eventTypeStringToEnum(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test_stateStringToEnum tests state string conversion
func Test_stateStringToEnum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected commonpb.FlowState
	}{
		{"ACTIVE uppercase", "ACTIVE", commonpb.FlowState_STATE_ACTIVE},
		{"ACTIVE lowercase", "active", commonpb.FlowState_STATE_ACTIVE},
		{"CLOSED", "CLOSED", commonpb.FlowState_STATE_CLOSED},
		{"TIMEOUT", "TIMEOUT", commonpb.FlowState_STATE_TIMEOUT},
		{"unknown", "xyz", commonpb.FlowState_STATE_UNKNOWN},
		{"empty", "", commonpb.FlowState_STATE_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stateStringToEnum(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test_directionStringToEnum tests direction string conversion
func Test_directionStringToEnum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected commonpb.FlowDirection
	}{
		{"INGRESS uppercase", "INGRESS", commonpb.FlowDirection_DIRECTION_INGRESS},
		{"INGRESS lowercase", "ingress", commonpb.FlowDirection_DIRECTION_INGRESS},
		{"EGRESS uppercase", "EGRESS", commonpb.FlowDirection_DIRECTION_EGRESS},
		{"EGRESS lowercase", "egress", commonpb.FlowDirection_DIRECTION_EGRESS},
		{"unknown", "xyz", commonpb.FlowDirection_DIRECTION_UNKNOWN},
		{"empty", "", commonpb.FlowDirection_DIRECTION_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := directionStringToEnum(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test_policyActionStringToEnum tests policy action string conversion
func Test_policyActionStringToEnum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected commonpb.PolicyAction
	}{
		{"ALLOW uppercase", "ALLOW", commonpb.PolicyAction_ACTION_ALLOW},
		{"ALLOW lowercase", "allow", commonpb.PolicyAction_ACTION_ALLOW},
		{"DENY uppercase", "DENY", commonpb.PolicyAction_ACTION_DENY},
		{"DENY lowercase", "deny", commonpb.PolicyAction_ACTION_DENY},
		{"LOG", "LOG", commonpb.PolicyAction_ACTION_LOG},
		{"unknown", "xyz", commonpb.PolicyAction_ACTION_UNKNOWN},
		{"empty", "", commonpb.PolicyAction_ACTION_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policyActionStringToEnum(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test_flowToProto tests flow to protobuf conversion
func Test_flowToProto(t *testing.T) {
	reporter := NewGRPCReporter("localhost:9090", "test-agent", 100)

	now := time.Now()
	testFlow := &flow.Flow{
		SourceIP:     "192.168.1.100",
		DestIP:       "10.0.0.50",
		SourcePort:   8080,
		DestPort:     443,
		Protocol:     "TCP",
		EventType:    "NEW",
		Direction:    "EGRESS",
		PacketCount:  100,
		ByteCount:    5000,
		StartTime:    now,
		PolicyID:     123,
		PolicyAction: "ALLOW",
		State:        "ACTIVE",
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"db": "postgres"},
	}

	event := reporter.flowToProto(testFlow)

	if event == nil {
		t.Fatal("Expected non-nil event")
	}

	// Verify IP conversion
	expectedSrcIP := ipToUint32("192.168.1.100")
	if event.SrcIp != expectedSrcIP {
		t.Errorf("Expected SrcIp=0x%08X, got 0x%08X", expectedSrcIP, event.SrcIp)
	}

	expectedDstIP := ipToUint32("10.0.0.50")
	if event.DstIp != expectedDstIP {
		t.Errorf("Expected DstIp=0x%08X, got 0x%08X", expectedDstIP, event.DstIp)
	}

	// Verify ports
	if event.SrcPort != 8080 {
		t.Errorf("Expected SrcPort=8080, got %d", event.SrcPort)
	}
	if event.DstPort != 443 {
		t.Errorf("Expected DstPort=443, got %d", event.DstPort)
	}

	// Verify protocol
	if event.Protocol != commonpb.Protocol_PROTOCOL_TCP {
		t.Errorf("Expected Protocol=PROTOCOL_TCP, got %v", event.Protocol)
	}

	// Verify event type
	if event.EventType != commonpb.FlowEventType_EVENT_NEW {
		t.Errorf("Expected EventType=EVENT_NEW, got %v", event.EventType)
	}

	// Verify direction
	if event.Direction != commonpb.FlowDirection_DIRECTION_EGRESS {
		t.Errorf("Expected Direction=DIRECTION_EGRESS, got %v", event.Direction)
	}

	// Verify counts
	if event.PacketCount != 100 {
		t.Errorf("Expected PacketCount=100, got %d", event.PacketCount)
	}
	if event.ByteCount != 5000 {
		t.Errorf("Expected ByteCount=5000, got %d", event.ByteCount)
	}

	// Verify timestamp
	expectedTimestamp := uint64(now.UnixNano())
	if event.TimestampNs != expectedTimestamp {
		t.Errorf("Expected TimestampNs=%d, got %d", expectedTimestamp, event.TimestampNs)
	}

	// Verify policy
	if event.PolicyId != 123 {
		t.Errorf("Expected PolicyId=123, got %d", event.PolicyId)
	}
	if event.PolicyAction != commonpb.PolicyAction_ACTION_ALLOW {
		t.Errorf("Expected PolicyAction=ACTION_ALLOW, got %v", event.PolicyAction)
	}

	// Verify state
	if event.State != commonpb.FlowState_STATE_ACTIVE {
		t.Errorf("Expected State=STATE_ACTIVE, got %v", event.State)
	}

	// Verify agent ID
	if event.AgentId != "test-agent" {
		t.Errorf("Expected AgentId=test-agent, got %s", event.AgentId)
	}

	// Verify labels
	if event.SourceLabels["app"] != "web" {
		t.Errorf("Expected SourceLabels[app]=web, got %s", event.SourceLabels["app"])
	}
	if event.DestLabels["db"] != "postgres" {
		t.Errorf("Expected DestLabels[db]=postgres, got %s", event.DestLabels["db"])
	}
}

// Test_flowToProto_InvalidIP tests handling of invalid IPs
func Test_flowToProto_InvalidIP(t *testing.T) {
	reporter := NewGRPCReporter("localhost:9090", "test-agent", 100)

	testFlow := &flow.Flow{
		SourceIP:  "invalid-ip",
		DestIP:    "also-invalid",
		Protocol:  "TCP",
		StartTime: time.Now(),
	}

	event := reporter.flowToProto(testFlow)

	if event.SrcIp != 0 {
		t.Errorf("Expected SrcIp=0 for invalid IP, got 0x%08X", event.SrcIp)
	}
	if event.DstIp != 0 {
		t.Errorf("Expected DstIp=0 for invalid IP, got 0x%08X", event.DstIp)
	}
}

// Test_flowToProto_UnknownEnums tests handling of unknown enum values
func Test_flowToProto_UnknownEnums(t *testing.T) {
	reporter := NewGRPCReporter("localhost:9090", "test-agent", 100)

	testFlow := &flow.Flow{
		SourceIP:     "192.168.1.1",
		DestIP:       "192.168.1.2",
		Protocol:     "UNKNOWN_PROTOCOL",
		EventType:    "UNKNOWN_EVENT",
		Direction:    "UNKNOWN_DIR",
		PolicyAction: "UNKNOWN_ACTION",
		State:        "UNKNOWN_STATE",
		StartTime:    time.Now(),
	}

	event := reporter.flowToProto(testFlow)

	if event.Protocol != commonpb.Protocol_PROTOCOL_UNKNOWN {
		t.Errorf("Expected PROTOCOL_UNKNOWN, got %v", event.Protocol)
	}
	if event.EventType != commonpb.FlowEventType_EVENT_UNKNOWN {
		t.Errorf("Expected EVENT_UNKNOWN, got %v", event.EventType)
	}
	if event.Direction != commonpb.FlowDirection_DIRECTION_UNKNOWN {
		t.Errorf("Expected DIRECTION_UNKNOWN, got %v", event.Direction)
	}
	if event.PolicyAction != commonpb.PolicyAction_ACTION_UNKNOWN {
		t.Errorf("Expected ACTION_UNKNOWN, got %v", event.PolicyAction)
	}
	if event.State != commonpb.FlowState_STATE_UNKNOWN {
		t.Errorf("Expected STATE_UNKNOWN, got %v", event.State)
	}
}

// Test_ipToUint32_RealWorldIPs tests conversion with real-world IPs
func Test_ipToUint32_RealWorldIPs(t *testing.T) {
	// Test that conversion is reversible
	testIPs := []string{
		"8.8.8.8",       // Google DNS
		"1.1.1.1",       // Cloudflare DNS
		"172.16.0.1",    // Private network
		"192.168.255.255", // Broadcast
		"0.0.0.0",       // Any
		"255.255.255.255", // Broadcast
	}

	for _, ipStr := range testIPs {
		t.Run(ipStr, func(t *testing.T) {
			// Convert to uint32
			asInt := ipToUint32(ipStr)

			// Convert back to IP
			ip := net.IPv4(
				byte(asInt>>24),
				byte(asInt>>16),
				byte(asInt>>8),
				byte(asInt),
			)

			// Should match original
			if ip.String() != ipStr {
				t.Errorf("IP conversion not reversible: %s -> 0x%08X -> %s", ipStr, asInt, ip.String())
			}
		})
	}
}

package flow

import (
	"testing"

	commonpb "github.com/ebpf-microsegment/src/proto/common"
	"google.golang.org/protobuf/proto"
)

// TestFlowEventSerialization tests basic FlowEvent serialization
func TestFlowEventSerialization(t *testing.T) {
	original := &FlowEvent{
		SrcIp:        0x0A000101, // 10.0.1.1
		DstIp:        0x0A000202, // 10.0.2.2
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		EventType:    commonpb.FlowEventType_EVENT_NEW,
		Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(1609459200000000000),
		PolicyId:     1001,
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		State:        commonpb.FlowState_STATE_ACTIVE,
		AgentId:      "agent-001",
	}

	// Marshal to bytes
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("FlowEvent serialized size (without labels): %d bytes", len(data))

	// Verify size is reasonable (should be around 48 bytes for base fields)
	if len(data) > 100 {
		t.Errorf("Serialized size %d bytes exceeds expected size (~48-60 bytes)", len(data))
	}

	// Unmarshal back
	decoded := &FlowEvent{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify critical fields
	if decoded.SrcIp != original.SrcIp {
		t.Errorf("SrcIp = %x, expected %x", decoded.SrcIp, original.SrcIp)
	}
	if decoded.DstIp != original.DstIp {
		t.Errorf("DstIp = %x, expected %x", decoded.DstIp, original.DstIp)
	}
	if decoded.SrcPort != original.SrcPort {
		t.Errorf("SrcPort = %d, expected %d", decoded.SrcPort, original.SrcPort)
	}
	if decoded.DstPort != original.DstPort {
		t.Errorf("DstPort = %d, expected %d", decoded.DstPort, original.DstPort)
	}
	if decoded.Protocol != original.Protocol {
		t.Errorf("Protocol = %v, expected %v", decoded.Protocol, original.Protocol)
	}
	if decoded.AgentId != original.AgentId {
		t.Errorf("AgentId = %q, expected %q", decoded.AgentId, original.AgentId)
	}
}

// TestFlowEventWithLabels tests FlowEvent with labels
func TestFlowEventWithLabels(t *testing.T) {
	original := &FlowEvent{
		SrcIp:        0x0A000101,
		DstIp:        0x0A000202,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		EventType:    commonpb.FlowEventType_EVENT_NEW,
		Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(1609459200000000000),
		PolicyId:     1001,
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		State:        commonpb.FlowState_STATE_ACTIVE,
		AgentId:      "agent-001",
		SourceLabels: map[string]string{
			"role": "web",
			"env":  "prod",
			"app":  "frontend",
		},
		DestLabels: map[string]string{
			"role": "api",
			"env":  "prod",
			"app":  "backend",
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("FlowEvent serialized size (with 6 labels): %d bytes", len(data))

	// Unmarshal and verify labels
	decoded := &FlowEvent{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.SourceLabels) != 3 {
		t.Errorf("SourceLabels count = %d, expected 3", len(decoded.SourceLabels))
	}
	if decoded.SourceLabels["role"] != "web" {
		t.Errorf("SourceLabels[role] = %q, expected 'web'", decoded.SourceLabels["role"])
	}
	if len(decoded.DestLabels) != 3 {
		t.Errorf("DestLabels count = %d, expected 3", len(decoded.DestLabels))
	}
}

// TestFlowQuerySerialization tests FlowQuery message
func TestFlowQuerySerialization(t *testing.T) {
	query := &FlowQuery{
		TimeRange: &commonpb.TimeRange{
			StartTime: 1609459200000000000,
			EndTime:   1609545600000000000,
		},
		SrcIp:        "10.0.1.0/24",
		DstIp:        "10.0.2.0/24",
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		AgentId:      "agent-001",
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		Limit:        100,
		Offset:       0,
	}

	data, err := proto.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("FlowQuery serialized size: %d bytes", len(data))

	decoded := &FlowQuery{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SrcIp != query.SrcIp {
		t.Errorf("SrcIp = %q, expected %q", decoded.SrcIp, query.SrcIp)
	}
	if decoded.Limit != query.Limit {
		t.Errorf("Limit = %d, expected %d", decoded.Limit, query.Limit)
	}
}

// BenchmarkFlowEventMarshal benchmarks FlowEvent serialization
// Target: > 100K ops/s as per tasks.md specification
func BenchmarkFlowEventMarshal(b *testing.B) {
	event := &FlowEvent{
		SrcIp:        0x0A000101,
		DstIp:        0x0A000202,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		EventType:    commonpb.FlowEventType_EVENT_NEW,
		Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(1609459200000000000),
		PolicyId:     1001,
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		State:        commonpb.FlowState_STATE_ACTIVE,
		AgentId:      "agent-001",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFlowEventUnmarshal benchmarks FlowEvent deserialization
func BenchmarkFlowEventUnmarshal(b *testing.B) {
	event := &FlowEvent{
		SrcIp:        0x0A000101,
		DstIp:        0x0A000202,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		EventType:    commonpb.FlowEventType_EVENT_NEW,
		Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(1609459200000000000),
		PolicyId:     1001,
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		State:        commonpb.FlowState_STATE_ACTIVE,
		AgentId:      "agent-001",
	}

	data, err := proto.Marshal(event)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := &FlowEvent{}
		if err := proto.Unmarshal(data, result); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFlowEventMarshalWithLabels benchmarks FlowEvent with labels
func BenchmarkFlowEventMarshalWithLabels(b *testing.B) {
	event := &FlowEvent{
		SrcIp:        0x0A000101,
		DstIp:        0x0A000202,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		EventType:    commonpb.FlowEventType_EVENT_NEW,
		Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(1609459200000000000),
		PolicyId:     1001,
		PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
		State:        commonpb.FlowState_STATE_ACTIVE,
		AgentId:      "agent-001",
		SourceLabels: map[string]string{
			"role": "web",
			"env":  "prod",
			"app":  "frontend",
		},
		DestLabels: map[string]string{
			"role": "api",
			"env":  "prod",
			"app":  "backend",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBatchFlowEventMarshal benchmarks batch serialization
// Target: 1000 events in batch as per tasks.md specification
func BenchmarkBatchFlowEventMarshal(b *testing.B) {
	// Create batch of 1000 events
	events := make([]*FlowEvent, 1000)
	for i := 0; i < 1000; i++ {
		events[i] = &FlowEvent{
			SrcIp:        0x0A000101,
			DstIp:        0x0A000202,
			SrcPort:      uint32(10000 + i),
			DstPort:      80,
			Protocol:     commonpb.Protocol_PROTOCOL_TCP,
			EventType:    commonpb.FlowEventType_EVENT_NEW,
			Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
			PacketCount:  uint64(100 + i),
			ByteCount:    uint64(15000 + i*100),
			TimestampNs:  uint64(1609459200000000000 + int64(i)*1000000),
			PolicyId:     1001,
			PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
			State:        commonpb.FlowState_STATE_ACTIVE,
			AgentId:      "agent-001",
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			_, err := proto.Marshal(event)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

package common

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestProtocolEnumValues verifies Protocol enum values match IANA standards
func TestProtocolEnumValues(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		expected int32
	}{
		{"Unknown", Protocol_PROTOCOL_UNKNOWN, 0},
		{"ICMP", Protocol_PROTOCOL_ICMP, 1},
		{"TCP", Protocol_PROTOCOL_TCP, 6},
		{"UDP", Protocol_PROTOCOL_UDP, 17},
		{"ANY", Protocol_PROTOCOL_ANY, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.protocol) != tt.expected {
				t.Errorf("Protocol %s = %d, expected %d", tt.name, tt.protocol, tt.expected)
			}
		})
	}
}

// TestPolicyActionEnumValues verifies PolicyAction enum values
func TestPolicyActionEnumValues(t *testing.T) {
	tests := []struct {
		name     string
		action   PolicyAction
		expected int32
	}{
		{"Unknown", PolicyAction_ACTION_UNKNOWN, 0},
		{"Allow", PolicyAction_ACTION_ALLOW, 1},
		{"Deny", PolicyAction_ACTION_DENY, 2},
		{"Log", PolicyAction_ACTION_LOG, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.action) != tt.expected {
				t.Errorf("PolicyAction %s = %d, expected %d", tt.name, tt.action, tt.expected)
			}
		})
	}
}

// TestFlowEventTypeEnumValues verifies FlowEventType enum values
func TestFlowEventTypeEnumValues(t *testing.T) {
	tests := []struct {
		name      string
		eventType FlowEventType
		expected  int32
	}{
		{"Unknown", FlowEventType_EVENT_UNKNOWN, 0},
		{"New", FlowEventType_EVENT_NEW, 1},
		{"Update", FlowEventType_EVENT_UPDATE, 2},
		{"Closed", FlowEventType_EVENT_CLOSED, 3},
		{"Timeout", FlowEventType_EVENT_TIMEOUT, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.eventType) != tt.expected {
				t.Errorf("FlowEventType %s = %d, expected %d", tt.name, tt.eventType, tt.expected)
			}
		})
	}
}

// TestReportResponseSerialization tests ReportResponse message serialization
func TestReportResponseSerialization(t *testing.T) {
	original := &ReportResponse{
		Success:        true,
		Message:        "Test message",
		AcceptedCount:  100,
		RejectedCount:  5,
	}

	// Marshal to bytes
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	decoded := &ReportResponse{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Success != original.Success {
		t.Errorf("Success = %v, expected %v", decoded.Success, original.Success)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message = %q, expected %q", decoded.Message, original.Message)
	}
	if decoded.AcceptedCount != original.AcceptedCount {
		t.Errorf("AcceptedCount = %d, expected %d", decoded.AcceptedCount, original.AcceptedCount)
	}
	if decoded.RejectedCount != original.RejectedCount {
		t.Errorf("RejectedCount = %d, expected %d", decoded.RejectedCount, original.RejectedCount)
	}

	t.Logf("ReportResponse serialized size: %d bytes", len(data))
}

// TestTimeRangeSerialization tests TimeRange message serialization
func TestTimeRangeSerialization(t *testing.T) {
	original := &TimeRange{
		StartTime: 1609459200000000000, // 2021-01-01 00:00:00 UTC in nanoseconds
		EndTime:   1609545600000000000, // 2021-01-02 00:00:00 UTC in nanoseconds
	}

	// Marshal to bytes
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	decoded := &TimeRange{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.StartTime != original.StartTime {
		t.Errorf("StartTime = %d, expected %d", decoded.StartTime, original.StartTime)
	}
	if decoded.EndTime != original.EndTime {
		t.Errorf("EndTime = %d, expected %d", decoded.EndTime, original.EndTime)
	}

	t.Logf("TimeRange serialized size: %d bytes", len(data))
}

// BenchmarkReportResponseMarshal benchmarks ReportResponse serialization
func BenchmarkReportResponseMarshal(b *testing.B) {
	msg := &ReportResponse{
		Success:        true,
		Message:        "Processing completed successfully",
		AcceptedCount:  1000,
		RejectedCount:  10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReportResponseUnmarshal benchmarks ReportResponse deserialization
func BenchmarkReportResponseUnmarshal(b *testing.B) {
	msg := &ReportResponse{
		Success:        true,
		Message:        "Processing completed successfully",
		AcceptedCount:  1000,
		RejectedCount:  10,
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &ReportResponse{}
		if err := proto.Unmarshal(data, result); err != nil {
			b.Fatal(err)
		}
	}
}

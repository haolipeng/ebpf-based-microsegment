package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidLabelKey tests label key validation
func TestValidLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"valid_simple", "app", true},
		{"valid_underscore", "app_name", true},
		{"valid_mixed", "App123", true},
		{"invalid_sql_injection", "foo;DROP TABLE", false},
		{"invalid_special_chars", "app-name", false},
		{"invalid_space", "app name", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validLabelKey(tt.key)
			assert.Equal(t, tt.expected, result, "key: %s", tt.key)
		})
	}
}

// TestFlowRowConversion tests FlowRow struct initialization
func TestFlowRowConversion(t *testing.T) {
	row := &FlowRow{
		SrcIP:        "192.168.1.1",
		DstIP:        "192.168.1.2",
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     6,
		Direction:    1,
		PacketCount:  100,
		ByteCount:    10240,
		AgentID:      "agent-test",
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "db"},
	}

	assert.Equal(t, "192.168.1.1", row.SrcIP)
	assert.Equal(t, int32(6), row.Protocol)
	assert.Equal(t, "web", row.SourceLabels["app"])
}

// TestPolicyRowConversion tests PolicyRow struct initialization
func TestPolicyRowConversion(t *testing.T) {
	row := &PolicyRow{
		RuleID:       1,
		SrcIP:        "10.0.0.0/8",
		DstIP:        "192.168.0.0/16",
		Protocol:     6,
		Action:       1,
		Priority:     100,
		SourceLabels: map[string]string{"env": "prod"},
		DestLabels:   map[string]string{"app": "api"},
	}

	assert.Equal(t, uint32(1), row.RuleID)
	assert.Equal(t, int32(6), row.Protocol)
	assert.Equal(t, "prod", row.SourceLabels["env"])
}

// TestAlertRowConversion tests AlertRow struct initialization
func TestAlertRowConversion(t *testing.T) {
	pid := uint32(1234)
	exePath := "/usr/bin/test"

	row := &AlertRow{
		AlertID:   "alert-123",
		Level:     2,
		Type:      1,
		PID:       &pid,
		ExePath:   &exePath,
		Reason:    "Suspicious connection attempt",
		Metadata:  map[string]string{"key": "value"},
		Timestamp: 1234567890,
	}

	assert.Equal(t, "alert-123", row.AlertID)
	assert.Equal(t, int32(2), row.Level)
	assert.Equal(t, uint32(1234), *row.PID)
	assert.Equal(t, "value", row.Metadata["key"])
}

// TestAgentRowConversion tests AgentRow struct initialization
func TestAgentRowConversion(t *testing.T) {
	row := &AgentRow{
		AgentID:       "agent-001",
		Hostname:      "test-host",
		Version:       "1.0.0",
		Interface:     "eth0",
		IPAddresses:   []string{"192.168.1.100", "10.0.0.100"},
		OS:            "linux",
		KernelVersion: "5.15.0",
		Status:        "active",
	}

	assert.Equal(t, "agent-001", row.AgentID)
	assert.Equal(t, "test-host", row.Hostname)
	assert.Len(t, row.IPAddresses, 2)
	assert.Equal(t, "192.168.1.100", row.IPAddresses[0])
}

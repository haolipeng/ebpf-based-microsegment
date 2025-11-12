// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"testing"
)

// TestPolicyDirectionConstants 验证 Direction 常量定义
func TestPolicyDirectionConstants(t *testing.T) {
	if DirectionAny != "any" {
		t.Errorf("Expected DirectionAny='any', got '%s'", DirectionAny)
	}
	if DirectionIngress != "ingress" {
		t.Errorf("Expected DirectionIngress='ingress', got '%s'", DirectionIngress)
	}
	if DirectionEgress != "egress" {
		t.Errorf("Expected DirectionEgress='egress', got '%s'", DirectionEgress)
	}
}

// TestGetDirectionValue 验证 Direction 字符串到数值的转换
func TestGetDirectionValue(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		expected  uint8
	}{
		{
			name:      "any direction",
			direction: "any",
			expected:  0,
		},
		{
			name:      "ingress direction",
			direction: "ingress",
			expected:  1,
		},
		{
			name:      "egress direction",
			direction: "egress",
			expected:  2,
		},
		{
			name:      "uppercase INGRESS",
			direction: "INGRESS",
			expected:  1,
		},
		{
			name:      "mixed case Egress",
			direction: "Egress",
			expected:  2,
		},
		{
			name:      "empty string defaults to any",
			direction: "",
			expected:  0,
		},
		{
			name:      "invalid value defaults to any",
			direction: "invalid",
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Direction: tt.direction}
			result := p.GetDirectionValue()
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestNormalizeDirection 验证 Direction 规范化功能
func TestNormalizeDirection(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
	}{
		{
			name:     "uppercase ANY",
			input:    "ANY",
			expected: "any",
		},
		{
			name:     "uppercase INGRESS",
			input:    "INGRESS",
			expected: "ingress",
		},
		{
			name:     "mixed case Egress",
			input:    "Egress",
			expected: "egress",
		},
		{
			name:     "empty string becomes any",
			input:    "",
			expected: "any",
		},
		{
			name:     "invalid value becomes any",
			input:    "invalid",
			expected: "any",
		},
		{
			name:     "whitespace becomes any",
			input:    "  ",
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Direction: tt.input}
			p.NormalizeDirection()
			if p.Direction != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, p.Direction)
			}
		})
	}
}

// TestValidateDirection 验证 Direction 验证逻辑
func TestValidateDirection(t *testing.T) {
	tests := []struct {
		name      string
		policy    Policy
		expectErr bool
	}{
		{
			name: "valid any direction",
			policy: Policy{
				RuleID:    1,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "any",
			},
			expectErr: false,
		},
		{
			name: "valid ingress direction",
			policy: Policy{
				RuleID:    2,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "ingress",
			},
			expectErr: false,
		},
		{
			name: "valid egress direction",
			policy: Policy{
				RuleID:    3,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "egress",
			},
			expectErr: false,
		},
		{
			name: "invalid direction",
			policy: Policy{
				RuleID:    4,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "invalid",
			},
			expectErr: true,
		},
		{
			name: "invalid protocol",
			policy: Policy{
				RuleID:    5,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "invalid_proto",
				Action:    "allow",
				Direction: "any",
			},
			expectErr: true,
		},
		{
			name: "invalid action",
			policy: Policy{
				RuleID:    6,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "invalid_action",
				Direction: "any",
			},
			expectErr: true,
		},
		{
			name: "invalid source IP",
			policy: Policy{
				RuleID:    7,
				SrcIP:     "invalid_ip",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "any",
			},
			expectErr: true,
		},
		{
			name: "empty direction (should be valid, defaults to any)",
			policy: Policy{
				RuleID:    8,
				SrcIP:     "192.168.1.0/24",
				DstIP:     "10.0.0.5/32",
				DstPort:   22,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestDirectionToString 验证数值到字符串的转换
func TestDirectionToString(t *testing.T) {
	tests := []struct {
		name     string
		value    uint8
		expected string
	}{
		{
			name:     "value 0 = any",
			value:    0,
			expected: "any",
		},
		{
			name:     "value 1 = ingress",
			value:    1,
			expected: "ingress",
		},
		{
			name:     "value 2 = egress",
			value:    2,
			expected: "egress",
		},
		{
			name:     "invalid value defaults to any",
			value:    99,
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := directionToString(tt.value)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestPolicyDirectionRoundTrip 验证 Direction 字段的往返转换
func TestPolicyDirectionRoundTrip(t *testing.T) {
	directions := []string{"any", "ingress", "egress"}

	for _, dir := range directions {
		t.Run(dir, func(t *testing.T) {
			p := &Policy{Direction: dir}

			// String -> uint8
			value := p.GetDirectionValue()

			// uint8 -> String
			result := directionToString(value)

			if result != dir {
				t.Errorf("Round trip failed: '%s' -> %d -> '%s'", dir, value, result)
			}
		})
	}
}

// TestPolicyStructureWithDirection 验证 Policy 结构包含 Direction 字段
func TestPolicyStructureWithDirection(t *testing.T) {
	p := Policy{
		RuleID:    1,
		SrcIP:     "192.168.1.100/32",
		DstIP:     "10.0.0.5/32",
		SrcPort:   12345,
		DstPort:   22,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}

	// Verify all fields are accessible
	if p.RuleID != 1 {
		t.Errorf("RuleID mismatch")
	}
	if p.Direction != "ingress" {
		t.Errorf("Direction mismatch: expected 'ingress', got '%s'", p.Direction)
	}
	if p.GetDirectionValue() != 1 {
		t.Errorf("Direction value mismatch: expected 1, got %d", p.GetDirectionValue())
	}
}

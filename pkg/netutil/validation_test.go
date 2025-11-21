package netutil

import (
	"testing"
)

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid IPv4 CIDR",
			cidr:        "192.168.1.0/24",
			expectError: false,
		},
		{
			name:        "valid IPv6 CIDR",
			cidr:        "2001:db8::/32",
			expectError: false,
		},
		{
			name:        "IPv4 with host bits",
			cidr:        "192.168.1.1/24",
			expectError: true,
			errorMsg:    "has host bits set",
		},
		{
			name:        "IPv6 with host bits",
			cidr:        "2001:db8::1/32",
			expectError: true,
			errorMsg:    "has host bits set",
		},
		{
			name:        "invalid mask",
			cidr:        "192.168.1.0/33",
			expectError: true,
		},
		{
			name:        "empty CIDR",
			cidr:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ValidateCIDR(tt.cidr)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateCIDR(%s) expected error, got nil", tt.cidr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateCIDR(%s) unexpected error: %v", tt.cidr, err)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		expectError bool
	}{
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv6", "::1", false},
		{"invalid IP", "invalid", true},
		{"empty IP", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateIP(tt.ip)
			if tt.expectError && err == nil {
				t.Errorf("ValidateIP(%s) expected error, got nil", tt.ip)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateIP(%s) unexpected error: %v", tt.ip, err)
			}
		})
	}
}

func TestValidateProtocol(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		expectError bool
	}{
		{"tcp", "tcp", false},
		{"TCP", "TCP", false}, // Case insensitive
		{"udp", "udp", false},
		{"icmp", "icmp", false},
		{"empty (any)", "", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProtocol(tt.protocol)
			if tt.expectError && err == nil {
				t.Errorf("ValidateProtocol(%s) expected error, got nil", tt.protocol)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateProtocol(%s) unexpected error: %v", tt.protocol, err)
			}
		})
	}
}

func TestProtocolStringToNumber(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected uint8
	}{
		{"tcp", "tcp", 6},
		{"TCP", "TCP", 6}, // Case insensitive
		{"udp", "udp", 17},
		{"icmp", "icmp", 1},
		{"icmpv6", "icmpv6", 58},
		{"unknown", "unknown", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProtocolStringToNumber(tt.protocol)
			if result != tt.expected {
				t.Errorf("ProtocolStringToNumber(%s) = %d, want %d", tt.protocol, result, tt.expected)
			}
		})
	}
}

func TestProtocolNumberToString(t *testing.T) {
	tests := []struct {
		name     string
		number   uint8
		expected string
	}{
		{"tcp", 6, "tcp"},
		{"udp", 17, "udp"},
		{"icmp", 1, "icmp"},
		{"icmpv6", 58, "icmpv6"},
		{"unknown", 99, "proto-99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProtocolNumberToString(tt.number)
			if result != tt.expected {
				t.Errorf("ProtocolNumberToString(%d) = %s, want %s", tt.number, result, tt.expected)
			}
		})
	}
}

func TestValidateDirection(t *testing.T) {
	tests := []struct {
		name        string
		direction   string
		expectError bool
	}{
		{"ingress", "ingress", false},
		{"egress", "egress", false},
		{"any", "any", false},
		{"INGRESS", "INGRESS", false}, // Case insensitive
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirection(tt.direction)
			if tt.expectError && err == nil {
				t.Errorf("ValidateDirection(%s) expected error, got nil", tt.direction)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateDirection(%s) unexpected error: %v", tt.direction, err)
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name        string
		action      string
		expectError bool
	}{
		{"allow", "allow", false},
		{"deny", "deny", false},
		{"log", "log", false},
		{"ALLOW", "ALLOW", false}, // Case insensitive
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAction(tt.action)
			if tt.expectError && err == nil {
				t.Errorf("ValidateAction(%s) expected error, got nil", tt.action)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateAction(%s) unexpected error: %v", tt.action, err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10.0.0.1", "10.0.0.1", true},
		{"172.16.0.1", "172.16.0.1", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"8.8.8.8", "8.8.8.8", false}, // Public IP
		{"fc00::1", "fc00::1", true},  // IPv6 ULA
		{"2001:db8::1", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := ValidateIP(tt.ip)
			if err != nil {
				t.Fatalf("Failed to parse IP: %s, error: %v", tt.ip, err)
			}
			result := IsPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

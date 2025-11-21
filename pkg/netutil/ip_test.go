package netutil

import (
	"net"
	"testing"
)

func TestIPToUint32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "192.168.1.1",
			ip:       "192.168.1.1",
			expected: 0xc0a80101, // big-endian
		},
		{
			name:     "10.0.0.1",
			ip:       "10.0.0.1",
			expected: 0x0a000001,
		},
		{
			name:     "0.0.0.0",
			ip:       "0.0.0.0",
			expected: 0x00000000,
		},
		{
			name:     "255.255.255.255",
			ip:       "255.255.255.255",
			expected: 0xffffffff,
		},
		{
			name:     "invalid IPv6",
			ip:       "::1",
			expected: 0, // IPv6 should return 0
		},
		{
			name:     "invalid IP",
			ip:       "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := IPToUint32(ip)
			if result != tt.expected {
				t.Errorf("IPToUint32(%s) = 0x%x, want 0x%x", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIPToUint32LE(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "192.168.1.1",
			ip:       "192.168.1.1",
			expected: 0x0101a8c0, // little-endian
		},
		{
			name:     "10.0.0.1",
			ip:       "10.0.0.1",
			expected: 0x0100000a,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := IPToUint32LE(ip)
			if result != tt.expected {
				t.Errorf("IPToUint32LE(%s) = 0x%x, want 0x%x", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestUint32ToIP(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected string
	}{
		{
			name:     "192.168.1.1",
			value:    0xc0a80101,
			expected: "192.168.1.1",
		},
		{
			name:     "10.0.0.1",
			value:    0x0a000001,
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint32ToIP(tt.value)
			if result.String() != tt.expected {
				t.Errorf("Uint32ToIP(0x%x) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func TestUint32LEToIP(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected string
	}{
		{
			name:     "192.168.1.1",
			value:    0x0101a8c0, // little-endian
			expected: "192.168.1.1",
		},
		{
			name:     "10.0.0.1",
			value:    0x0100000a,
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint32LEToIP(tt.value)
			if result.String() != tt.expected {
				t.Errorf("Uint32LEToIP(0x%x) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		expectIP    string
		expectMask  string
		expectError bool
	}{
		{
			name:       "valid IPv4 CIDR",
			cidr:       "192.168.1.0/24",
			expectIP:   "192.168.1.0",
			expectMask: "ffffff00",
		},
		{
			name:       "IPv4 without mask",
			cidr:       "192.168.1.1",
			expectIP:   "192.168.1.1",
			expectMask: "ffffffff", // /32
		},
		{
			name:       "IPv6 without mask",
			cidr:       "2001:db8::1",
			expectIP:   "2001:db8::1",
			expectMask: "ffffffffffffffffffffffffffffffff", // /128
		},
		{
			name:        "empty CIDR",
			cidr:        "",
			expectError: true,
		},
		{
			name:        "invalid CIDR",
			cidr:        "invalid/24",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, ipnet, err := ParseCIDR(tt.cidr)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseCIDR(%s) expected error, got nil", tt.cidr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCIDR(%s) unexpected error: %v", tt.cidr, err)
				return
			}

			if ip.String() != tt.expectIP {
				t.Errorf("ParseCIDR(%s) IP = %s, want %s", tt.cidr, ip, tt.expectIP)
			}

			maskHex := ""
			for _, b := range ipnet.Mask {
				maskHex += string("0123456789abcdef"[b>>4])
				maskHex += string("0123456789abcdef"[b&0xf])
			}
			if maskHex != tt.expectMask {
				t.Errorf("ParseCIDR(%s) Mask = %s, want %s", tt.cidr, maskHex, tt.expectMask)
			}
		})
	}
}

func TestIPv6ToUint32Array(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected [4]uint32
	}{
		{
			name: "::1 (loopback)",
			ip:   "::1",
			expected: [4]uint32{0, 0, 0, 1},
		},
		{
			name: "2001:db8::1",
			ip:   "2001:db8::1",
			expected: [4]uint32{0x20010db8, 0, 0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := IPv6ToUint32Array(ip)
			if result != tt.expected {
				t.Errorf("IPv6ToUint32Array(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4", "192.168.1.1", true},
		{"IPv6", "::1", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", true}, // Should be treated as IPv4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := IsIPv4(ip)
			if result != tt.expected {
				t.Errorf("IsIPv4(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4", "192.168.1.1", false},
		{"IPv6", "::1", true},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", false}, // Should be treated as IPv4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := IsIPv6(ip)
			if result != tt.expected {
				t.Errorf("IsIPv6(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

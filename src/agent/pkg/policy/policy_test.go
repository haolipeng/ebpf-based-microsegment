// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"fmt"
	"net"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEBPFMap is a mock implementation of ebpf.Map for testing
type MockEBPFMap struct {
	mock.Mock
	data map[string]interface{} // Simple in-memory storage for testing
}

func NewMockEBPFMap() *MockEBPFMap {
	return &MockEBPFMap{
		data: make(map[string]interface{}),
	}
}

func (m *MockEBPFMap) Put(key, value interface{}) error {
	args := m.Called(key, value)
	if args.Error(0) == nil {
		// Store in map for later retrieval
		keyStr := fmt.Sprintf("%v", key)
		m.data[keyStr] = value
	}
	return args.Error(0)
}

func (m *MockEBPFMap) Delete(key interface{}) error {
	args := m.Called(key)
	if args.Error(0) == nil {
		keyStr := fmt.Sprintf("%v", key)
		delete(m.data, keyStr)
	}
	return args.Error(0)
}

func (m *MockEBPFMap) Iterate() *MockMapIterator {
	return &MockMapIterator{
		data:     m.data,
		keys:     make([]string, 0, len(m.data)),
		current:  -1,
		mockCall: m.Called(),
	}
}

// MockMapIterator mocks the map iterator
type MockMapIterator struct {
	data     map[string]interface{}
	keys     []string
	current  int
	mockCall mock.Arguments
}

func (m *MockMapIterator) Next(key, value interface{}) bool {
	if m.current == -1 {
		// Initialize keys on first call
		for k := range m.data {
			m.keys = append(m.keys, k)
		}
	}
	m.current++
	if m.current >= len(m.keys) {
		return false
	}
	// Note: In real implementation, we would copy key/value here
	// For testing, we just return true/false
	return true
}

func (m *MockMapIterator) Err() error {
	if m.mockCall.Get(0) == nil {
		return nil
	}
	return m.mockCall.Error(0)
}

// MockDataPlane is a mock implementation of DataPlaneInterface
type MockDataPlane struct {
	mock.Mock
	policyMap *MockEBPFMap
}

func NewMockDataPlane() *MockDataPlane {
	return &MockDataPlane{
		policyMap: NewMockEBPFMap(),
	}
}

func (m *MockDataPlane) GetPolicyMap() *ebpf.Map {
	// We can't return MockEBPFMap as *ebpf.Map, so we need to work around this
	// For testing purposes, we'll test helper functions directly
	return nil
}

// TestParseCIDR tests CIDR parsing functionality
func TestParseCIDR(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectIP    string
		expectMask  string
		expectError bool
	}{
		{
			name:        "valid IP without CIDR",
			input:       "192.168.1.100",
			expectIP:    "192.168.1.100",
			expectMask:  "255.255.255.255",
			expectError: false,
		},
		{
			name:        "valid IP with /32 CIDR",
			input:       "192.168.1.100/32",
			expectIP:    "192.168.1.100",
			expectMask:  "255.255.255.255",
			expectError: false,
		},
		{
			name:        "valid IP with /24 CIDR",
			input:       "192.168.1.0/24",
			expectIP:    "192.168.1.0",
			expectMask:  "255.255.255.0",
			expectError: false,
		},
		{
			name:        "valid IP with /16 CIDR",
			input:       "192.168.0.0/16",
			expectIP:    "192.168.0.0",
			expectMask:  "255.255.0.0",
			expectError: false,
		},
		{
			name:        "valid IP with /8 CIDR",
			input:       "10.0.0.0/8",
			expectIP:    "10.0.0.0",
			expectMask:  "255.0.0.0",
			expectError: false,
		},
		{
			name:        "wildcard IP 0.0.0.0",
			input:       "0.0.0.0",
			expectIP:    "0.0.0.0",
			expectMask:  "255.255.255.255",
			expectError: false,
		},
		{
			name:        "invalid IP",
			input:       "999.999.999.999",
			expectError: true,
		},
		{
			name:        "invalid CIDR format",
			input:       "192.168.1.1/",
			expectError: true,
		},
		{
			name:        "invalid CIDR range",
			input:       "192.168.1.1/33",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, mask, err := parseCIDR(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectIP, ip.String())
			if tc.expectMask != "" {
				maskIP := net.IP(*mask)
				assert.Equal(t, tc.expectMask, maskIP.String())
			}
		})
	}
}

// TestParseProtocol tests protocol parsing
func TestParseProtocol(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    uint8
		expectError bool
	}{
		{name: "tcp lowercase", input: "tcp", expected: 6},
		{name: "tcp uppercase", input: "TCP", expected: 6},
		{name: "udp lowercase", input: "udp", expected: 17},
		{name: "udp uppercase", input: "UDP", expected: 17},
		{name: "icmp lowercase", input: "icmp", expected: 1},
		{name: "icmp uppercase", input: "ICMP", expected: 1},
		{name: "any lowercase", input: "any", expected: 0},
		{name: "any uppercase", input: "ANY", expected: 0},
		{name: "empty string", input: "", expected: 0},
		{name: "invalid protocol", input: "http", expectError: true},
		{name: "invalid protocol2", input: "xyz", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseProtocol(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown protocol")
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseAction tests action parsing
func TestParseAction(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    uint8
		expectError bool
	}{
		{name: "allow lowercase", input: "allow", expected: 0},
		{name: "allow uppercase", input: "ALLOW", expected: 0},
		{name: "deny lowercase", input: "deny", expected: 1},
		{name: "deny uppercase", input: "DENY", expected: 1},
		{name: "log lowercase", input: "log", expected: 2},
		{name: "log uppercase", input: "LOG", expected: 2},
		{name: "invalid action", input: "block", expectError: true},
		{name: "invalid action2", input: "drop", expectError: true},
		{name: "empty string", input: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseAction(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown action")
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIPToUint32 tests IP to uint32 conversion
func TestIPToUint32(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected uint32
	}{
		{
			name:     "0.0.0.0",
			input:    "0.0.0.0",
			expected: 0x00000000,
		},
		{
			name:     "127.0.0.1",
			input:    "127.0.0.1",
			expected: 0x0100007f, // Little endian
		},
		{
			name:     "192.168.1.1",
			input:    "192.168.1.1",
			expected: 0x0101a8c0, // Little endian
		},
		{
			name:     "255.255.255.255",
			input:    "255.255.255.255",
			expected: 0xffffffff,
		},
		{
			name:     "10.0.0.1",
			input:    "10.0.0.1",
			expected: 0x0100000a, // Little endian
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.input)
			result := ipToUint32(ip)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestUint32ToIP tests uint32 to IP conversion
func TestUint32ToIP(t *testing.T) {
	testCases := []struct {
		name     string
		input    uint32
		expected string
	}{
		{
			name:     "0.0.0.0",
			input:    0x00000000,
			expected: "0.0.0.0",
		},
		{
			name:     "127.0.0.1",
			input:    0x0100007f,
			expected: "127.0.0.1",
		},
		{
			name:     "192.168.1.1",
			input:    0x0101a8c0,
			expected: "192.168.1.1",
		},
		{
			name:     "255.255.255.255",
			input:    0xffffffff,
			expected: "255.255.255.255",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := uint32ToIP(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIPConversionRoundTrip tests round-trip conversion
func TestIPConversionRoundTrip(t *testing.T) {
	testIPs := []string{
		"0.0.0.0",
		"127.0.0.1",
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"8.8.8.8",
		"255.255.255.255",
	}

	for _, ipStr := range testIPs {
		t.Run(ipStr, func(t *testing.T) {
			// IP -> uint32 -> IP
			ip := net.ParseIP(ipStr)
			uint32Val := ipToUint32(ip)
			resultIP := uint32ToIP(uint32Val)
			assert.Equal(t, ipStr, resultIP)
		})
	}
}

// TestHtons tests host to network short conversion
func TestHtons(t *testing.T) {
	testCases := []struct {
		input    uint16
		expected uint16
	}{
		{input: 80, expected: 0x5000},     // HTTP port
		{input: 443, expected: 0xbb01},    // HTTPS port
		{input: 22, expected: 0x1600},     // SSH port
		{input: 8080, expected: 0x901f},   // Alt HTTP port
		{input: 0, expected: 0},           // Zero
		{input: 65535, expected: 0xffff},  // Max uint16
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("port_%d", tc.input), func(t *testing.T) {
			result := htons(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestNtohs tests network to host short conversion
func TestNtohs(t *testing.T) {
	testCases := []struct {
		input    uint16
		expected uint16
	}{
		{input: 0x5000, expected: 80},
		{input: 0xbb01, expected: 443},
		{input: 0x1600, expected: 22},
		{input: 0, expected: 0},
		{input: 0xffff, expected: 65535},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("network_%x", tc.input), func(t *testing.T) {
			result := ntohs(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestPortConversionRoundTrip tests round-trip port conversion
func TestPortConversionRoundTrip(t *testing.T) {
	testPorts := []uint16{0, 22, 80, 443, 8080, 3306, 5432, 27017, 65535}

	for _, port := range testPorts {
		t.Run(fmt.Sprintf("port_%d", port), func(t *testing.T) {
			// host -> network -> host
			networkPort := htons(port)
			hostPort := ntohs(networkPort)
			assert.Equal(t, port, hostPort)
		})
	}
}

// TestProtoToString tests protocol uint8 to string conversion
func TestProtoToString(t *testing.T) {
	testCases := []struct {
		input    uint8
		expected string
	}{
		{input: 0, expected: "any"},
		{input: 1, expected: "icmp"},
		{input: 6, expected: "tcp"},
		{input: 17, expected: "udp"},
		{input: 255, expected: "255"}, // Unknown protocol
		{input: 50, expected: "50"},   // Unknown protocol
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("proto_%d", tc.input), func(t *testing.T) {
			result := protoToString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestActionToString tests action uint8 to string conversion
func TestActionToString(t *testing.T) {
	testCases := []struct {
		input    uint8
		expected string
	}{
		{input: 0, expected: "allow"},
		{input: 1, expected: "deny"},
		{input: 2, expected: "log"},
		{input: 255, expected: "255"}, // Unknown action
		{input: 10, expected: "10"},   // Unknown action
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("action_%d", tc.input), func(t *testing.T) {
			result := actionToString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestBoolToUint8 tests boolean to uint8 conversion
func TestBoolToUint8(t *testing.T) {
	testCases := []struct {
		input    bool
		expected uint8
	}{
		{input: true, expected: 1},
		{input: false, expected: 0},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("bool_%t", tc.input), func(t *testing.T) {
			result := boolToUint8(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestPolicyValidation tests policy validation scenarios
func TestPolicyValidation(t *testing.T) {
	testCases := []struct {
		name        string
		policy      *Policy
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid policy with all fields",
			policy: &Policy{
				RuleID:   1,
				SrcIP:    "192.168.1.100",
				DstIP:    "10.0.0.1",
				SrcPort:  12345,
				DstPort:  80,
				Protocol: "tcp",
				Action:   "allow",
				Priority: 100,
			},
			expectError: false,
		},
		{
			name: "valid policy with CIDR",
			policy: &Policy{
				RuleID:   2,
				SrcIP:    "192.168.1.0/24",
				DstIP:    "10.0.0.0/16",
				DstPort:  443,
				Protocol: "tcp",
				Action:   "deny",
				Priority: 200,
			},
			expectError: false,
		},
		{
			name: "valid policy with wildcard",
			policy: &Policy{
				RuleID:   3,
				SrcIP:    "0.0.0.0",
				DstIP:    "0.0.0.0",
				Protocol: "any",
				Action:   "log",
			},
			expectError: false,
		},
		{
			name: "invalid source IP",
			policy: &Policy{
				RuleID:   4,
				SrcIP:    "invalid",
				DstIP:    "10.0.0.1",
				Protocol: "tcp",
				Action:   "allow",
			},
			expectError: true,
			errorMsg:    "invalid source IP",
		},
		{
			name: "invalid destination IP",
			policy: &Policy{
				RuleID:   5,
				SrcIP:    "192.168.1.1",
				DstIP:    "999.999.999.999",
				Protocol: "tcp",
				Action:   "allow",
			},
			expectError: true,
			errorMsg:    "invalid destination IP",
		},
		{
			name: "invalid protocol",
			policy: &Policy{
				RuleID:   6,
				SrcIP:    "192.168.1.1",
				DstIP:    "10.0.0.1",
				Protocol: "http",
				Action:   "allow",
			},
			expectError: true,
			errorMsg:    "invalid protocol",
		},
		{
			name: "invalid action",
			policy: &Policy{
				RuleID:   7,
				SrcIP:    "192.168.1.1",
				DstIP:    "10.0.0.1",
				Protocol: "tcp",
				Action:   "block",
			},
			expectError: true,
			errorMsg:    "invalid action",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test parsing individually
			if tc.expectError {
				// Try parsing each field to see which one fails
				if tc.errorMsg == "invalid source IP" {
					_, _, err := parseCIDR(tc.policy.SrcIP)
					assert.Error(t, err)
				} else if tc.errorMsg == "invalid destination IP" {
					_, _, err := parseCIDR(tc.policy.DstIP)
					assert.Error(t, err)
				} else if tc.errorMsg == "invalid protocol" {
					_, err := parseProtocol(tc.policy.Protocol)
					assert.Error(t, err)
				} else if tc.errorMsg == "invalid action" {
					_, err := parseAction(tc.policy.Action)
					assert.Error(t, err)
				}
			} else {
				// Validate all fields parse successfully
				_, _, err := parseCIDR(tc.policy.SrcIP)
				assert.NoError(t, err)

				_, _, err = parseCIDR(tc.policy.DstIP)
				assert.NoError(t, err)

				_, err = parseProtocol(tc.policy.Protocol)
				assert.NoError(t, err)

				_, err = parseAction(tc.policy.Action)
				assert.NoError(t, err)
			}
		})
	}
}

// TestPolicyKeyConstruction tests that policy keys are constructed correctly
func TestPolicyKeyConstruction(t *testing.T) {
	testCases := []struct {
		name   string
		policy *Policy
	}{
		{
			name: "HTTP policy",
			policy: &Policy{
				RuleID:   1,
				SrcIP:    "192.168.1.100",
				DstIP:    "10.0.0.1",
				SrcPort:  0,
				DstPort:  80,
				Protocol: "tcp",
				Action:   "allow",
			},
		},
		{
			name: "HTTPS policy",
			policy: &Policy{
				RuleID:   2,
				SrcIP:    "0.0.0.0",
				DstIP:    "10.0.0.1",
				SrcPort:  0,
				DstPort:  443,
				Protocol: "tcp",
				Action:   "allow",
			},
		},
		{
			name: "DNS policy",
			policy: &Policy{
				RuleID:   3,
				SrcIP:    "192.168.1.0/24",
				DstIP:    "8.8.8.8",
				DstPort:  53,
				Protocol: "udp",
				Action:   "allow",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse all components
			srcIP, _, err := parseCIDR(tc.policy.SrcIP)
			assert.NoError(t, err)

			dstIP, _, err := parseCIDR(tc.policy.DstIP)
			assert.NoError(t, err)

			proto, err := parseProtocol(tc.policy.Protocol)
			assert.NoError(t, err)

			action, err := parseAction(tc.policy.Action)
			assert.NoError(t, err)

			// Construct key components
			srcIPUint := ipToUint32(srcIP)
			dstIPUint := ipToUint32(dstIP)
			srcPortNet := htons(tc.policy.SrcPort)
			dstPortNet := htons(tc.policy.DstPort)

			// Verify conversions are reversible
			assert.Equal(t, srcIP.String(), uint32ToIP(srcIPUint))
			assert.Equal(t, dstIP.String(), uint32ToIP(dstIPUint))
			assert.Equal(t, tc.policy.SrcPort, ntohs(srcPortNet))
			assert.Equal(t, tc.policy.DstPort, ntohs(dstPortNet))
			assert.Equal(t, tc.policy.Protocol, protoToString(proto))
			assert.Equal(t, tc.policy.Action, actionToString(action))
		})
	}
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package conntrack

import (
	"net"
	"testing"

	ct "github.com/florianl/go-conntrack"
	"golang.org/x/sys/unix"
)

// TestGetIPVersion tests IP version detection
func TestGetIPVersion(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected uint8
	}{
		{
			name:     "IPv4 address",
			ip:       net.ParseIP("192.168.1.1"),
			expected: 4,
		},
		{
			name:     "IPv6 address",
			ip:       net.ParseIP("2001:db8::1"),
			expected: 6,
		},
		{
			name:     "IPv4-mapped IPv6",
			ip:       net.ParseIP("::ffff:192.168.1.1"),
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := &ct.IPTuple{Src: &tt.ip}
			version := getIPVersion(tuple)
			if version != tt.expected {
				t.Errorf("Expected IP version %d, got %d", tt.expected, version)
			}
		})
	}
}

// TestGetNATTypeFromStatus tests NAT type detection
func TestGetNATTypeFromStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   uint32
		expected uint8
	}{
		{
			name:     "No NAT",
			status:   ConntrackStatusAssured,
			expected: NATTypeNone,
		},
		{
			name:     "SNAT only",
			status:   ConntrackStatusSrcNAT | ConntrackStatusAssured,
			expected: NATTypeSNAT,
		},
		{
			name:     "DNAT only",
			status:   ConntrackStatusDstNAT | ConntrackStatusAssured,
			expected: NATTypeDNAT,
		},
		{
			name:     "Both SNAT and DNAT",
			status:   ConntrackStatusSrcNAT | ConntrackStatusDstNAT | ConntrackStatusAssured,
			expected: NATTypeBoth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			natType := getNATTypeFromStatus(&tt.status)
			if natType != tt.expected {
				t.Errorf("Expected NAT type %d, got %d", tt.expected, natType)
			}
		})
	}
}

// TestCopyNetIPToArray tests IP address conversion
func TestCopyNetIPToArray(t *testing.T) {
	tests := []struct {
		name      string
		ip        net.IP
		ipVersion uint8
		wantErr   bool
	}{
		{
			name:      "IPv4 address",
			ip:        net.ParseIP("192.168.1.1"),
			ipVersion: 4,
			wantErr:   false,
		},
		{
			name:      "IPv6 address",
			ip:        net.ParseIP("2001:db8::1"),
			ipVersion: 6,
			wantErr:   false,
		},
		{
			name:      "Invalid IPv4",
			ip:        net.ParseIP("invalid"),
			ipVersion: 4,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arr [4]uint32
			err := copyNetIPToArray(&arr, tt.ip, tt.ipVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyNetIPToArray() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.ipVersion == 4 {
					// IPv4: only last element should be non-zero
					if arr[0] != 0 || arr[1] != 0 || arr[2] != 0 {
						t.Errorf("IPv4 should only use last element, got: %v", arr)
					}
					if arr[3] == 0 {
						t.Error("IPv4 last element should be non-zero")
					}
				} else {
					// IPv6: at least one element should be non-zero
					allZero := true
					for _, v := range arr {
						if v != 0 {
							allZero = false
							break
						}
					}
					if allZero {
						t.Error("IPv6 should have at least one non-zero element")
					}
				}
			}
		})
	}
}

// TestShouldSync tests entry filtering logic
func TestShouldSync(t *testing.T) {
	tcpProto := uint8(unix.IPPROTO_TCP)
	udpProto := uint8(unix.IPPROTO_UDP)
	icmpProto := uint8(unix.IPPROTO_ICMP)

	tests := []struct {
		name     string
		con      ct.Con
		config   *SyncConfig
		expected bool
	}{
		{
			name: "TCP with TCP-only filter",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Proto: &ct.ProtoTuple{Number: &tcpProto},
				},
			},
			config:   &SyncConfig{SyncTCPOnly: true},
			expected: true,
		},
		{
			name: "ICMP with TCP-only filter",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Proto: &ct.ProtoTuple{Number: &icmpProto},
				},
			},
			config:   &SyncConfig{SyncTCPOnly: true},
			expected: false,
		},
		{
			name: "UDP with TCP-only filter",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Proto: &ct.ProtoTuple{Number: &udpProto},
				},
			},
			config:   &SyncConfig{SyncTCPOnly: true},
			expected: true,
		},
		{
			name: "Assured connection with OnlyEstablished",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Proto: &ct.ProtoTuple{Number: &tcpProto},
				},
				Status: ptrUint32(ConntrackStatusAssured),
			},
			config:   &SyncConfig{SyncTCPOnly: false, OnlyEstablished: true},
			expected: true,
		},
		{
			name: "Non-assured connection with OnlyEstablished",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Proto: &ct.ProtoTuple{Number: &tcpProto},
				},
				Status: ptrUint32(0),
			},
			config:   &SyncConfig{SyncTCPOnly: false, OnlyEstablished: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSync(tt.con, tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSync() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestConvertToMapEntry tests complete conversion
func TestConvertToMapEntry(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("10.0.0.1")
	srcPort := uint16(12345)
	dstPort := uint16(80)
	tcpProto := uint8(unix.IPPROTO_TCP)
	status := uint32(ConntrackStatusAssured | ConntrackStatusSrcNAT)

	con := ct.Con{
		Origin: &ct.IPTuple{
			Src: &srcIP,
			Dst: &dstIP,
			Proto: &ct.ProtoTuple{
				Number:  &tcpProto,
				SrcPort: &srcPort,
				DstPort: &dstPort,
			},
		},
		Reply: &ct.IPTuple{
			Src: &dstIP,
			Dst: &srcIP,
			Proto: &ct.ProtoTuple{
				Number:  &tcpProto,
				SrcPort: &dstPort,
				DstPort: &srcPort,
			},
		},
		Status: &status,
	}

	key, entry, err := ConvertToMapEntry(con)
	if err != nil {
		t.Fatalf("ConvertToMapEntry() error = %v", err)
	}

	// Verify key
	if key == nil {
		t.Fatal("key is nil")
	}
	if key.Protocol != tcpProto {
		t.Errorf("key.Protocol = %d, want %d", key.Protocol, tcpProto)
	}
	if key.IPVersion != 4 {
		t.Errorf("key.IPVersion = %d, want 4", key.IPVersion)
	}

	// Verify entry
	if entry == nil {
		t.Fatal("entry is nil")
	}
	if entry.NATType != NATTypeSNAT {
		t.Errorf("entry.NATType = %d, want %d (SNAT)", entry.NATType, NATTypeSNAT)
	}
	if entry.Status != status {
		t.Errorf("entry.Status = 0x%08x, want 0x%08x", entry.Status, status)
	}
}

// TestConvertToMapEntry_MissingFields tests error handling
func TestConvertToMapEntry_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		con     ct.Con
		wantErr bool
	}{
		{
			name: "Missing origin tuple",
			con: ct.Con{
				Reply: &ct.IPTuple{},
			},
			wantErr: true,
		},
		{
			name: "Missing reply tuple",
			con: ct.Con{
				Origin: &ct.IPTuple{},
			},
			wantErr: true,
		},
		{
			name: "Valid entry",
			con: ct.Con{
				Origin: &ct.IPTuple{
					Src: ptrIP("192.168.1.1"),
					Dst: ptrIP("10.0.0.1"),
				},
				Reply: &ct.IPTuple{
					Src: ptrIP("10.0.0.1"),
					Dst: ptrIP("192.168.1.1"),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ConvertToMapEntry(tt.con)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToMapEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFormatConntrackEntry tests entry formatting
func TestFormatConntrackEntry(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("10.0.0.1")
	srcPort := uint16(12345)
	dstPort := uint16(80)
	tcpProto := uint8(unix.IPPROTO_TCP)
	status := uint32(ConntrackStatusSrcNAT)

	con := ct.Con{
		Origin: &ct.IPTuple{
			Src: &srcIP,
			Dst: &dstIP,
			Proto: &ct.ProtoTuple{
				Number:  &tcpProto,
				SrcPort: &srcPort,
				DstPort: &dstPort,
			},
		},
		Reply: &ct.IPTuple{
			Src: &dstIP,
			Dst: &srcIP,
			Proto: &ct.ProtoTuple{
				Number:  &tcpProto,
				SrcPort: &dstPort,
				DstPort: &srcPort,
			},
		},
		Status: &status,
	}

	result := FormatConntrackEntry(con)
	if result == "" {
		t.Error("FormatConntrackEntry() returned empty string")
	}

	// Check that result contains expected components
	if !contains(result, "TCP") {
		t.Errorf("Expected 'TCP' in result, got: %s", result)
	}
	if !contains(result, "192.168.1.1") {
		t.Errorf("Expected source IP in result, got: %s", result)
	}
	if !contains(result, "SNAT") {
		t.Errorf("Expected 'SNAT' in result, got: %s", result)
	}
}

// Helper functions

func ptrUint32(v uint32) *uint32 {
	return &v
}

func ptrIP(s string) *net.IP {
	ip := net.ParseIP(s)
	return &ip
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

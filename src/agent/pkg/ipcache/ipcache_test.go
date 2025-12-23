// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package ipcache

import (
	"net/netip"
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/identity"
)

func TestIPCache(t *testing.T) {
	cache := NewIPCache()

	// Test Upsert and Lookup
	prefix := netip.MustParsePrefix("10.0.0.1/32")
	id := identity.NumericIdentity(256)
	metadata := identity.IPIdentityMetadata{Source: "test"}

	err := cache.Upsert(prefix, id, metadata)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Lookup by exact address
	addr := netip.MustParseAddr("10.0.0.1")
	gotID, ok := cache.Lookup(addr)
	if !ok {
		t.Fatal("Lookup should find the IP")
	}
	if gotID != id {
		t.Errorf("Lookup returned %d, want %d", gotID, id)
	}

	// Lookup by string
	gotID, ok = cache.LookupByIP("10.0.0.1")
	if !ok {
		t.Fatal("LookupByIP should find the IP")
	}
	if gotID != id {
		t.Errorf("LookupByIP returned %d, want %d", gotID, id)
	}

	// Lookup non-existent IP
	_, ok = cache.LookupByIP("10.0.0.2")
	if ok {
		t.Error("LookupByIP should not find non-existent IP")
	}
}

func TestIPCacheCIDR(t *testing.T) {
	cache := NewIPCache()

	// Add a CIDR prefix
	prefix := netip.MustParsePrefix("10.0.0.0/24")
	id := identity.NumericIdentity(257)
	metadata := identity.IPIdentityMetadata{Source: "cidr-test"}

	err := cache.Upsert(prefix, id, metadata)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Lookup IPs within the CIDR should match
	testCases := []struct {
		ip      string
		wantID  identity.NumericIdentity
		wantOK  bool
	}{
		{"10.0.0.0", id, true},
		{"10.0.0.1", id, true},
		{"10.0.0.255", id, true},
		{"10.0.1.0", identity.IdentityUnknown, false},
		{"192.168.0.1", identity.IdentityUnknown, false},
	}

	for _, tc := range testCases {
		gotID, ok := cache.LookupByIP(tc.ip)
		if ok != tc.wantOK {
			t.Errorf("Lookup(%s) ok = %v, want %v", tc.ip, ok, tc.wantOK)
		}
		if ok && gotID != tc.wantID {
			t.Errorf("Lookup(%s) = %d, want %d", tc.ip, gotID, tc.wantID)
		}
	}
}

func TestIPCacheLPM(t *testing.T) {
	cache := NewIPCache()

	// Add multiple overlapping prefixes
	// More specific prefix should match first
	cache.Upsert(netip.MustParsePrefix("10.0.0.0/8"), identity.NumericIdentity(100), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("10.0.0.0/16"), identity.NumericIdentity(200), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("10.0.0.0/24"), identity.NumericIdentity(300), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("10.0.0.1/32"), identity.NumericIdentity(400), identity.IPIdentityMetadata{})

	// Test LPM matching - most specific should win
	testCases := []struct {
		ip     string
		wantID identity.NumericIdentity
	}{
		{"10.0.0.1", identity.NumericIdentity(400)},   // /32 match
		{"10.0.0.2", identity.NumericIdentity(300)},   // /24 match
		{"10.0.1.1", identity.NumericIdentity(200)},   // /16 match
		{"10.1.0.1", identity.NumericIdentity(100)},   // /8 match
	}

	for _, tc := range testCases {
		gotID, ok := cache.LookupByIP(tc.ip)
		if !ok {
			t.Errorf("Lookup(%s) should find an identity", tc.ip)
			continue
		}
		if gotID != tc.wantID {
			t.Errorf("Lookup(%s) = %d, want %d", tc.ip, gotID, tc.wantID)
		}
	}
}

func TestIPCacheDelete(t *testing.T) {
	cache := NewIPCache()

	prefix := netip.MustParsePrefix("10.0.0.1/32")
	cache.Upsert(prefix, identity.NumericIdentity(256), identity.IPIdentityMetadata{})

	// Verify it exists
	_, ok := cache.LookupByIP("10.0.0.1")
	if !ok {
		t.Fatal("Entry should exist before delete")
	}

	// Delete it
	err := cache.Delete(prefix)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, ok = cache.LookupByIP("10.0.0.1")
	if ok {
		t.Error("Entry should not exist after delete")
	}
}

func TestIPCacheGetAll(t *testing.T) {
	cache := NewIPCache()

	// Add multiple entries
	cache.Upsert(netip.MustParsePrefix("10.0.0.1/32"), identity.NumericIdentity(256), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("10.0.0.2/32"), identity.NumericIdentity(257), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("192.168.0.0/24"), identity.NumericIdentity(258), identity.IPIdentityMetadata{})

	entries := cache.GetAll()
	if len(entries) != 3 {
		t.Errorf("GetAll() returned %d entries, want 3", len(entries))
	}
}

func TestIPCacheClear(t *testing.T) {
	cache := NewIPCache()

	// Add entries
	cache.Upsert(netip.MustParsePrefix("10.0.0.1/32"), identity.NumericIdentity(256), identity.IPIdentityMetadata{})
	cache.Upsert(netip.MustParsePrefix("10.0.0.2/32"), identity.NumericIdentity(257), identity.IPIdentityMetadata{})

	// Clear
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if cache.Size() != 0 {
		t.Errorf("Size after clear = %d, want 0", cache.Size())
	}
}

func TestIPCacheIPv6(t *testing.T) {
	cache := NewIPCache()

	// Test IPv6 address
	prefix := netip.MustParsePrefix("2001:db8::1/128")
	id := identity.NumericIdentity(259)

	err := cache.Upsert(prefix, id, identity.IPIdentityMetadata{Source: "ipv6-test"})
	if err != nil {
		t.Fatalf("Upsert IPv6 failed: %v", err)
	}

	gotID, ok := cache.Lookup(netip.MustParseAddr("2001:db8::1"))
	if !ok {
		t.Fatal("Lookup should find IPv6 address")
	}
	if gotID != id {
		t.Errorf("Lookup returned %d, want %d", gotID, id)
	}
}

func TestParseIPOrCIDR(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"10.0.0.1", false},
		{"10.0.0.0/24", false},
		{"192.168.1.1", false},
		{"192.168.1.0/16", false},
		{"2001:db8::1", false},
		{"2001:db8::/32", false},
		{"invalid", true},
		{"256.0.0.1", true},
		{"10.0.0.0/33", true},
	}

	for _, tt := range tests {
		_, err := ParseIPOrCIDR(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseIPOrCIDR(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestPrefixToBPFKey(t *testing.T) {
	// Test IPv4 conversion
	prefix := netip.MustParsePrefix("10.0.0.1/32")
	key := prefixToBPFKey(prefix)

	// Check prefix length (32 + 96 = 128 for IPv4-mapped IPv6)
	if key.PrefixLen != 128 {
		t.Errorf("IPv4 PrefixLen = %d, want 128", key.PrefixLen)
	}

	// Check IPv4-mapped format
	// First 10 bytes should be 0
	for i := 0; i < 10; i++ {
		if key.IP[i] != 0 {
			t.Errorf("IPv4 IP[%d] = %d, want 0", i, key.IP[i])
		}
	}
	// Bytes 10-11 should be 0xff
	if key.IP[10] != 0xff || key.IP[11] != 0xff {
		t.Error("IPv4-mapped marker should be 0xffff")
	}
	// Bytes 12-15 should be the IPv4 address
	if key.IP[12] != 10 || key.IP[13] != 0 || key.IP[14] != 0 || key.IP[15] != 1 {
		t.Error("IPv4 address bytes incorrect")
	}

	// Test IPv6 conversion
	prefix6 := netip.MustParsePrefix("2001:db8::1/64")
	key6 := prefixToBPFKey(prefix6)

	if key6.PrefixLen != 64 {
		t.Errorf("IPv6 PrefixLen = %d, want 64", key6.PrefixLen)
	}

	// First two bytes should be 0x20, 0x01 (2001)
	if key6.IP[0] != 0x20 || key6.IP[1] != 0x01 {
		t.Error("IPv6 address prefix incorrect")
	}
}

func TestBPFKeyToPrefix(t *testing.T) {
	// Test round-trip for IPv4
	originalPrefix := netip.MustParsePrefix("10.0.0.1/32")
	key := prefixToBPFKey(originalPrefix)
	recovered, err := BPFKeyToPrefix(&key)
	if err != nil {
		t.Fatalf("BPFKeyToPrefix failed: %v", err)
	}
	if recovered.String() != originalPrefix.String() {
		t.Errorf("Round-trip failed: got %s, want %s", recovered, originalPrefix)
	}

	// Test round-trip for IPv6
	originalPrefix6 := netip.MustParsePrefix("2001:db8::1/128")
	key6 := prefixToBPFKey(originalPrefix6)
	recovered6, err := BPFKeyToPrefix(&key6)
	if err != nil {
		t.Fatalf("BPFKeyToPrefix IPv6 failed: %v", err)
	}
	if recovered6.String() != originalPrefix6.String() {
		t.Errorf("IPv6 Round-trip failed: got %s, want %s", recovered6, originalPrefix6)
	}
}

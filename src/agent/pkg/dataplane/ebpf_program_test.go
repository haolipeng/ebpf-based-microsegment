// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"encoding/binary"
	"net"
	"os"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test eBPF program loading and basic functionality
// Note: These tests require CAP_BPF/CAP_NET_ADMIN capabilities
func TestEBPFProgramLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF program test in short mode")
	}

	t.Run("load TC program", func(t *testing.T) {
		// Load the TC eBPF program
		spec, err := loadBpf()
		if err != nil {
			t.Skipf("Cannot load eBPF program (may require kernel >= 5.10): %v", err)
		}

		require.NotNil(t, spec)
		require.NotNil(t, spec.Programs)

		// Check that the main TC program exists
		tcProg, exists := spec.Programs["tc_microsegment_filter"]
		require.True(t, exists, "tc_microsegment_filter program not found")
		require.NotNil(t, tcProg)

		t.Logf("TC program loaded successfully: %d instructions", len(tcProg.Instructions))
	})

	t.Run("load XDP program", func(t *testing.T) {
		// Load the XDP eBPF program
		spec, err := loadXdpbpf()
		if err != nil {
			t.Skipf("Cannot load XDP eBPF program: %v", err)
		}

		require.NotNil(t, spec)
		require.NotNil(t, spec.Programs)

		// Check that the main XDP program exists
		xdpProg, exists := spec.Programs["xdp_microsegment_prog"]
		require.True(t, exists, "xdp_microsegment_prog program not found")
		require.NotNil(t, xdpProg)

		t.Logf("XDP program loaded successfully: %d instructions", len(xdpProg.Instructions))
	})

	t.Run("verify maps exist", func(t *testing.T) {
		spec, err := loadBpf()
		if err != nil {
			t.Skipf("Cannot load eBPF program: %v", err)
		}

		require.NotNil(t, spec.Maps)

		// Check required maps
		requiredMaps := []string{
			"session_map",
			"policy_map",
			"wildcard_policy_map",
			"stats_map",
			"flow_events",
		}

		for _, mapName := range requiredMaps {
			m, exists := spec.Maps[mapName]
			require.True(t, exists, "required map %s not found", mapName)
			require.NotNil(t, m)
			t.Logf("Map %s: type=%v, max_entries=%d", mapName, m.Type, m.MaxEntries)
		}
	})
}

// Test policy map operations
func TestPolicyMapOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF map test in short mode")
	}

	// Create a temporary collection for testing
	// Use /sys/fs/bpf for map pinning (required for BPF maps)
	tmpPinPath := "/sys/fs/bpf/test_policy_map"
	err := os.MkdirAll(tmpPinPath, 0755)
	require.NoError(t, err, "failed to create pin path directory")
	defer os.RemoveAll(tmpPinPath)

	objs := &bpfObjects{}
	opts := &ebpf.CollectionOptions{
		Maps: ebpf.MapOptions{
			PinPath: tmpPinPath,
		},
	}
	err = loadBpfObjects(objs, opts)
	if err != nil {
		t.Skipf("Cannot create eBPF collection: %v", err)
	}
	defer objs.Close()

	t.Run("add and lookup exact policy", func(t *testing.T) {
		policyMap := objs.PolicyMap
		require.NotNil(t, policyMap)

		// Create policy key (matches eBPF struct policy_key - 44 bytes)
		key := struct {
			SrcIP     [4]uint32
			DstIP     [4]uint32
			SrcPort   uint16
			DstPort   uint16
			Protocol  uint8
			Direction uint8
			IPVersion uint8
			Pad       uint8
			VlanID    uint16
			Pad2      uint16
		}{
			SrcIP:     ipToArray(net.ParseIP("192.168.1.100")),
			DstIP:     ipToArray(net.ParseIP("192.168.1.1")),
			SrcPort:   htons(12345),
			DstPort:   htons(80),
			Protocol:  6, // TCP
			Direction: 0, // INGRESS
			IPVersion: 4, // IPv4
			Pad:       0,
			VlanID:    0,
			Pad2:      0,
		}

		// Create policy value (matches eBPF struct policy_value - 16 bytes)
		value := struct {
			Action     uint8
			LogEnabled uint8
			Priority   uint16
			RuleID     uint32
			HitCount   uint64
		}{
			Action:     1,   // ALLOW
			LogEnabled: 0,
			Priority:   100,
			RuleID:     100,
			HitCount:   0,
		}

		// Add policy
		err := policyMap.Put(&key, &value)
		require.NoError(t, err, "failed to add policy")

		// Lookup policy
		var result struct {
			Action     uint8
			LogEnabled uint8
			Priority   uint16
			RuleID     uint32
			HitCount   uint64
		}
		err = policyMap.Lookup(&key, &result)
		require.NoError(t, err, "failed to lookup policy")
		assert.Equal(t, uint8(1), result.Action)
		assert.Equal(t, uint32(100), result.RuleID)
	})

	t.Run("add wildcard policy", func(t *testing.T) {
		wildcardMap := objs.WildcardPolicyMap
		require.NotNil(t, wildcardMap)

		// Wildcard policy: allow all traffic to port 443
		policy := createWildcardPolicy(
			"0.0.0.0/0",    // Any source
			"0.0.0.0/0",    // Any destination
			0,              // Any source port
			443,            // HTTPS port
			6,              // TCP
			1,              // ALLOW
			0,              // INGRESS
			200,            // Rule ID
		)

		// Add at index 0
		idx := uint32(0)
		err := wildcardMap.Put(&idx, &policy)
		require.NoError(t, err, "failed to add wildcard policy")

		// Verify it was added
		var result wildcardPolicyStruct
		err = wildcardMap.Lookup(&idx, &result)
		require.NoError(t, err, "failed to lookup wildcard policy")
		assert.Equal(t, uint16(443), ntohs(result.DstPort))
		assert.Equal(t, uint8(6), result.Protocol)
		assert.Equal(t, uint8(1), result.Action)
	})
}

// Test session map operations
func TestSessionMapOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF map test in short mode")
	}

	// Use /sys/fs/bpf for map pinning (required for BPF maps)
	tmpPinPath := "/sys/fs/bpf/test_session_map"
	err := os.MkdirAll(tmpPinPath, 0755)
	require.NoError(t, err, "failed to create pin path directory")
	defer os.RemoveAll(tmpPinPath)

	objs := &bpfObjects{}
	opts := &ebpf.CollectionOptions{
		Maps: ebpf.MapOptions{
			PinPath: tmpPinPath,
		},
	}
	err = loadBpfObjects(objs, opts)
	if err != nil {
		t.Skipf("Cannot create eBPF collection: %v", err)
	}
	defer objs.Close()

	t.Run("add and lookup session", func(t *testing.T) {
		sessionMap := objs.SessionMap
		require.NotNil(t, sessionMap)

		// Create flow key (matches eBPF struct flow_key - 40 bytes)
		key := struct {
			SrcIP     [4]uint32
			DstIP     [4]uint32
			SrcPort   uint16
			DstPort   uint16
			Protocol  uint8
			IPVersion uint8
			VlanID    uint16
		}{
			SrcIP:     ipToArray(net.ParseIP("192.168.1.100")),
			DstIP:     ipToArray(net.ParseIP("192.168.1.1")),
			SrcPort:   htons(12345),
			DstPort:   htons(80),
			Protocol:  6, // TCP
			IPVersion: 4, // IPv4
			VlanID:    0,
		}

		// Create session value (matches eBPF struct session_value - 72 bytes)
		value := struct {
			CreatedTS       uint64
			LastSeenTS      uint64
			PacketsToServer uint64
			PacketsToClient uint64
			BytesToServer   uint64
			BytesToClient   uint64
			TCPSeqClient    uint32
			TCPSeqServer    uint32
			TCPAckClient    uint32
			TCPAckServer    uint32
			TCPWindowSize   uint16
			TCPRetransCount uint8
			State           uint8
			TCPState        uint8
			PolicyAction    uint8
			Flags           uint8
			Pad             uint8
		}{
			CreatedTS:       1000000,
			LastSeenTS:      1000000,
			PacketsToServer: 1,
			BytesToServer:   60,
			TCPSeqClient:    12345,
			TCPSeqServer:    0,
			TCPAckClient:    0,
			TCPAckServer:    0,
			TCPWindowSize:   65535,
			TCPRetransCount: 0,
			State:           1, // SESSION_STATE_NEW
			TCPState:        1, // TCP_STATE_SYN_SENT
			PolicyAction:    1, // ALLOW
			Flags:           0,
			Pad:             0,
		}

		// Add session
		err := sessionMap.Put(&key, &value)
		require.NoError(t, err, "failed to add session")

		// Lookup session
		var result struct {
			CreatedTS       uint64
			LastSeenTS      uint64
			PacketsToServer uint64
			PacketsToClient uint64
			BytesToServer   uint64
			BytesToClient   uint64
			TCPSeqClient    uint32
			TCPSeqServer    uint32
			TCPAckClient    uint32
			TCPAckServer    uint32
			TCPWindowSize   uint16
			TCPRetransCount uint8
			State           uint8
			TCPState        uint8
			PolicyAction    uint8
			Flags           uint8
			Pad             uint8
		}
		err = sessionMap.Lookup(&key, &result)
		require.NoError(t, err, "failed to lookup session")
		assert.Equal(t, uint64(1), result.PacketsToServer)
		assert.Equal(t, uint8(1), result.State)
		assert.Equal(t, uint8(1), result.PolicyAction)
	})
}

// Helper functions

// wildcardPolicyStruct matches the eBPF wildcard_policy structure
type wildcardPolicyStruct struct {
	SrcIP      [4]uint32
	SrcIPMask  [4]uint32
	DstIP      [4]uint32
	DstIPMask  [4]uint32
	SrcPort    uint16
	DstPort    uint16
	Protocol   uint8
	Action     uint8
	LogEnabled uint8
	Direction  uint8
	IPVersion  uint8
	Pad        [3]uint8
	Priority   uint16
	VlanID     uint16
	RuleID     uint32
	ProcessName [16]byte
}

// createWildcardPolicy creates a wildcard policy structure
func createWildcardPolicy(srcCIDR, dstCIDR string, srcPort, dstPort uint16, protocol, action, direction uint8, ruleID uint32) wildcardPolicyStruct {
	srcIP, srcMask := parseCIDRForTest(srcCIDR)
	dstIP, dstMask := parseCIDRForTest(dstCIDR)

	return wildcardPolicyStruct{
		SrcIP:      ipToArray(srcIP),
		SrcIPMask:  maskToArray(srcMask),
		DstIP:      ipToArray(dstIP),
		DstIPMask:  maskToArray(dstMask),
		SrcPort:    htons(srcPort),
		DstPort:    htons(dstPort),
		Protocol:   protocol,
		Action:     action,
		LogEnabled: 0,
		Direction:  direction,
		IPVersion:  4,
		Priority:   100,
		VlanID:     0,
		RuleID:     ruleID,
	}
}

// createFlowKey creates a flow key structure (matches eBPF struct flow_key - 40 bytes)
func createFlowKey(srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8) interface{} {
	return struct {
		SrcIP     [4]uint32
		DstIP     [4]uint32
		SrcPort   uint16
		DstPort   uint16
		Protocol  uint8
		IPVersion uint8
		VlanID    uint16
	}{
		SrcIP:     ipToArray(net.ParseIP(srcIP)),
		DstIP:     ipToArray(net.ParseIP(dstIP)),
		SrcPort:   htons(srcPort),
		DstPort:   htons(dstPort),
		Protocol:  protocol,
		IPVersion: 4, // IPv4
		VlanID:    0,
	}
}

// parseCIDRForTest parses CIDR notation
func parseCIDRForTest(cidr string) (net.IP, net.IPMask) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return net.IPv4zero, net.CIDRMask(0, 32)
	}
	return ip, ipnet.Mask
}

// ipToArray converts IP to [4]uint32 array (little-endian)
func ipToArray(ip net.IP) [4]uint32 {
	var arr [4]uint32
	ip4 := ip.To4()
	if ip4 != nil {
		// IPv4-mapped IPv6
		arr[3] = binary.LittleEndian.Uint32(ip4)
	} else {
		// IPv6
		for i := 0; i < 4; i++ {
			arr[i] = binary.LittleEndian.Uint32(ip[i*4 : (i+1)*4])
		}
	}
	return arr
}

// maskToArray converts netmask to [4]uint32 array
func maskToArray(mask net.IPMask) [4]uint32 {
	var arr [4]uint32
	if len(mask) == 4 {
		// IPv4 mask
		arr[3] = binary.BigEndian.Uint32(mask)
	} else {
		// IPv6 mask
		for i := 0; i < 4; i++ {
			arr[i] = binary.BigEndian.Uint32(mask[i*4 : (i+1)*4])
		}
	}
	return arr
}

// ipToUint32LE converts IPv4 to uint32 (little-endian)
func ipToUint32LE(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(ip4)
}

// htons converts host byte order to network byte order (big-endian)
func htons(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}

// ntohs converts network byte order to host byte order
func ntohs(v uint16) uint16 {
	return htons(v) // Same operation
}

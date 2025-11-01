// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package testutil

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/cilium/ebpf"
)

// FlowKey represents a 5-tuple flow key for eBPF maps.
// This must match the kernel-side struct exactly.
type FlowKey struct {
	SrcIP    uint32
	DstIP    uint32
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
	Pad      [3]uint8
}

// SessionValue represents session tracking data in eBPF map.
type SessionValue struct {
	CreatedAt           uint64
	LastSeen            uint64
	PacketsToServer     uint64
	PacketsToClient     uint64
	BytesToServer       uint64
	BytesToClient       uint64
	State               uint8
	TCPState            uint8
	Action              uint8
	Pad                 [5]uint8
}

// PolicyValue represents policy data in eBPF map.
type PolicyValue struct {
	Action     uint8
	LogEnabled uint8
	Priority   uint16
	RuleID     uint32
	HitCount   uint64
}

// NewFlowKey creates a FlowKey from network parameters.
// IP addresses can be in CIDR format (e.g., "192.168.1.0/24") or plain IP.
// This function matches the exact behavior of PolicyManager's parseCIDR.
func NewFlowKey(srcIP, dstIP string, srcPort, dstPort uint16, protocol string) (*FlowKey, error) {
	// Parse source IP - add /32 if no CIDR notation (matches PolicyManager)
	if !strings.Contains(srcIP, "/") {
		srcIP = srcIP + "/32"
	}
	srcIPParsed, _, err := net.ParseCIDR(srcIP)
	if err != nil {
		return nil, fmt.Errorf("invalid source IP: %s", srcIP)
	}
	srcIP4 := srcIPParsed.To4()

	// Parse destination IP - add /32 if no CIDR notation (matches PolicyManager)
	if !strings.Contains(dstIP, "/") {
		dstIP = dstIP + "/32"
	}
	dstIPParsed, _, err := net.ParseCIDR(dstIP)
	if err != nil {
		return nil, fmt.Errorf("invalid destination IP: %s", dstIP)
	}
	dstIP4 := dstIPParsed.To4()

	if srcIP4 == nil || dstIP4 == nil {
		return nil, fmt.Errorf("only IPv4 is supported")
	}

	// Convert protocol string to number
	proto, err := protocolToNumber(protocol)
	if err != nil {
		return nil, err
	}

	key := &FlowKey{
		SrcIP:    ipToUint32(srcIP4),
		DstIP:    ipToUint32(dstIP4),
		SrcPort:  htons(srcPort),
		DstPort:  htons(dstPort),
		Protocol: proto,
	}

	return key, nil
}

// LookupSession looks up a session in the session map.
func LookupSession(sessionMap *ebpf.Map, key *FlowKey) (*SessionValue, error) {
	var value SessionValue

	if err := sessionMap.Lookup(key, &value); err != nil {
		return nil, fmt.Errorf("session lookup failed: %w", err)
	}

	return &value, nil
}

// LookupPolicy looks up a policy in the policy map.
func LookupPolicy(policyMap *ebpf.Map, key *FlowKey) (*PolicyValue, error) {
	var value PolicyValue

	if err := policyMap.Lookup(key, &value); err != nil {
		return nil, fmt.Errorf("policy lookup failed: %w", err)
	}

	return &value, nil
}

// CountSessions counts the total number of sessions in the map.
func CountSessions(sessionMap *ebpf.Map) (int, error) {
	count := 0
	var key FlowKey
	var value SessionValue

	iter := sessionMap.Iterate()
	for iter.Next(&key, &value) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate sessions: %w", err)
	}

	return count, nil
}

// CountPolicies counts the total number of policies in the map.
func CountPolicies(policyMap *ebpf.Map) (int, error) {
	count := 0
	var key FlowKey
	var value PolicyValue

	iter := policyMap.Iterate()
	for iter.Next(&key, &value) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate policies: %w", err)
	}

	return count, nil
}

// GetStatistic reads a single statistic from the stats map.
func GetStatistic(statsMap *ebpf.Map, statType uint32) (uint64, error) {
	var values []uint64

	if err := statsMap.Lookup(&statType, &values); err != nil {
		return 0, fmt.Errorf("failed to lookup stat %d: %w", statType, err)
	}

	// Sum up per-CPU values
	var total uint64
	for _, v := range values {
		total += v
	}

	return total, nil
}

// GetAllStatistics reads all statistics from the stats map.
func GetAllStatistics(statsMap *ebpf.Map) (map[string]uint64, error) {
	stats := make(map[string]uint64)

	// Statistic types (must match kernel definitions)
	statTypes := map[string]uint32{
		"total_packets":   0,
		"allowed_packets": 1,
		"denied_packets":  2,
		"new_sessions":    3,
		"closed_sessions": 4,
		"active_sessions": 5,
		"policy_hits":     6,
		"policy_misses":   7,
	}

	for name, typ := range statTypes {
		value, err := GetStatistic(statsMap, typ)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s: %w", name, err)
		}
		stats[name] = value
	}

	return stats, nil
}

// VerifyPolicyExists checks if a policy exists in the map.
func VerifyPolicyExists(policyMap *ebpf.Map, srcIP, dstIP string, srcPort, dstPort uint16, protocol string) (bool, error) {
	key, err := NewFlowKey(srcIP, dstIP, srcPort, dstPort, protocol)
	if err != nil {
		return false, err
	}

	var value PolicyValue
	err = policyMap.Lookup(key, &value)
	if err != nil {
		if err == ebpf.ErrKeyNotExist {
			return false, nil
		}
		return false, fmt.Errorf("lookup failed: %w", err)
	}

	return true, nil
}

// VerifySessionExists checks if a session exists in the map.
func VerifySessionExists(sessionMap *ebpf.Map, srcIP, dstIP string, srcPort, dstPort uint16, protocol string) (bool, error) {
	key, err := NewFlowKey(srcIP, dstIP, srcPort, dstPort, protocol)
	if err != nil {
		return false, err
	}

	var value SessionValue
	err = sessionMap.Lookup(key, &value)
	if err != nil {
		if err == ebpf.ErrKeyNotExist {
			return false, nil
		}
		return false, fmt.Errorf("lookup failed: %w", err)
	}

	return true, nil
}

// Helper functions

func ipToUint32(ip net.IP) uint32 {
	// Must match the byte order used by PolicyManager (LittleEndian)
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(ip)
}

func htons(port uint16) uint16 {
	return (port>>8)&0xff | (port<<8)&0xff00
}

func protocolToNumber(proto string) (uint8, error) {
	switch proto {
	case "tcp":
		return 6, nil
	case "udp":
		return 17, nil
	case "icmp":
		return 1, nil
	case "any":
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported protocol: %s", proto)
	}
}

// ActionToString converts action number to string.
func ActionToString(action uint8) string {
	switch action {
	case 0:
		return "allow"
	case 1:
		return "deny"
	case 2:
		return "log"
	default:
		return "unknown"
	}
}

// StateToString converts session state to string.
func StateToString(state uint8) string {
	states := []string{
		"new",
		"established",
		"closing",
		"closed",
	}

	if int(state) < len(states) {
		return states[state]
	}
	return "unknown"
}

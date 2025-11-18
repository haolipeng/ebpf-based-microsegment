// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package conntrack

import (
	"fmt"
	"net"
	"time"

	ct "github.com/florianl/go-conntrack"
	"golang.org/x/sys/unix"
)

// ConvertToMapEntry converts a conntrack entry to eBPF map key/value
func ConvertToMapEntry(con ct.Con) (*ConntrackKey, *ConntrackEntry, error) {
	// Validate required fields
	if con.Origin == nil || con.Reply == nil {
		return nil, nil, fmt.Errorf("missing origin or reply tuple")
	}

	// Extract original tuple (pre-NAT)
	originalTuple, err := extractFlowKey(con.Origin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract original tuple: %w", err)
	}

	// Extract reply tuple
	replyTuple, err := extractFlowKey(con.Reply)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract reply tuple: %w", err)
	}

	// Determine NAT type based on status flags
	natType := getNATTypeFromStatus(con.Status)

	// For cache key, use the post-NAT addresses
	// If SNAT: use reply destination (that's where packets come from after NAT)
	// If DNAT: use reply source (that's where packets go to after NAT)
	// If both: use reply tuple
	cacheKey := &ConntrackKey{
		Protocol:  originalTuple.Protocol,
		IPVersion: getIPVersion(con.Origin),
	}

	// For NAT connections, the cache key should use the translated addresses
	// (the addresses as they appear on the wire after NAT)
	if natType != NATTypeNone {
		// Use reply tuple for cache key (post-NAT addresses)
		copyIPToArray(&cacheKey.SrcIP, &replyTuple.DstIP, cacheKey.IPVersion)
		copyIPToArray(&cacheKey.DstIP, &replyTuple.SrcIP, cacheKey.IPVersion)
		cacheKey.SrcPort = replyTuple.DstPort
		cacheKey.DstPort = replyTuple.SrcPort
	} else {
		// No NAT: use original tuple
		copyIPToArray(&cacheKey.SrcIP, &originalTuple.SrcIP, cacheKey.IPVersion)
		copyIPToArray(&cacheKey.DstIP, &originalTuple.DstIP, cacheKey.IPVersion)
		cacheKey.SrcPort = originalTuple.SrcPort
		cacheKey.DstPort = originalTuple.DstPort
	}

	// Build cache entry
	entry := &ConntrackEntry{
		OriginalTuple: *originalTuple,
		ReplyTuple:    *replyTuple,
		Timestamp:     uint64(time.Now().UnixNano()),
		Status:        getStatus(con.Status),
		NATType:       natType,
	}

	return cacheKey, entry, nil
}

// extractFlowKey extracts a FlowKey from conntrack tuple
func extractFlowKey(tuple *ct.IPTuple) (*FlowKey, error) {
	if tuple == nil {
		return nil, fmt.Errorf("nil tuple")
	}

	key := &FlowKey{
		IPVersion: getIPVersion(tuple),
		VlanID:    0, // Conntrack doesn't track VLAN
	}

	// Extract IP addresses
	if tuple.Src != nil {
		if err := copyNetIPToArray(&key.SrcIP, *tuple.Src, key.IPVersion); err != nil {
			return nil, fmt.Errorf("failed to copy source IP: %w", err)
		}
	}

	if tuple.Dst != nil {
		if err := copyNetIPToArray(&key.DstIP, *tuple.Dst, key.IPVersion); err != nil {
			return nil, fmt.Errorf("failed to copy destination IP: %w", err)
		}
	}

	// Extract ports and protocol from ProtoTuple
	if tuple.Proto != nil {
		if tuple.Proto.Number != nil {
			key.Protocol = *tuple.Proto.Number
		}
		if tuple.Proto.SrcPort != nil {
			key.SrcPort = *tuple.Proto.SrcPort
		}
		if tuple.Proto.DstPort != nil {
			key.DstPort = *tuple.Proto.DstPort
		}
	}

	return key, nil
}

// getIPVersion determines IP version from tuple
func getIPVersion(tuple *ct.IPTuple) uint8 {
	if tuple == nil || tuple.Src == nil {
		return 4 // Default to IPv4
	}

	ip := *tuple.Src
	if ip.To4() != nil {
		return 4
	}
	return 6
}

// getProtocol extracts protocol number from conntrack entry
func getProtocol(protoInfo *ct.ProtoInfo) uint8 {
	// ProtoInfo is for protocol-specific state, not the protocol number
	// Protocol number is in IPTuple.Proto
	return 0 // Not used in current implementation
}

// getStatus extracts status flags
func getStatus(status *uint32) uint32 {
	if status == nil {
		return 0
	}
	return *status
}

// getNATTypeFromStatus determines NAT type from conntrack status
func getNATTypeFromStatus(status *uint32) uint8 {
	if status == nil {
		return NATTypeNone
	}

	s := *status
	hasSNAT := (s & ConntrackStatusSrcNAT) != 0
	hasDNAT := (s & ConntrackStatusDstNAT) != 0

	if hasSNAT && hasDNAT {
		return NATTypeBoth
	} else if hasSNAT {
		return NATTypeSNAT
	} else if hasDNAT {
		return NATTypeDNAT
	}

	return NATTypeNone
}

// copyNetIPToArray copies net.IP to [4]uint32 array
// Supports both IPv4 and IPv6
func copyNetIPToArray(dst *[4]uint32, src net.IP, ipVersion uint8) error {
	if ipVersion == 4 {
		// IPv4: convert to IPv4-mapped IPv6 format (::ffff:a.b.c.d)
		// Store in last element only
		ip4 := src.To4()
		if ip4 == nil {
			return fmt.Errorf("invalid IPv4 address")
		}
		dst[0] = 0
		dst[1] = 0
		dst[2] = 0
		// Network byte order: big-endian
		dst[3] = uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
	} else {
		// IPv6: use all 4 elements
		ip6 := src.To16()
		if ip6 == nil {
			return fmt.Errorf("invalid IPv6 address")
		}
		// Network byte order: big-endian
		for i := 0; i < 4; i++ {
			offset := i * 4
			dst[i] = uint32(ip6[offset])<<24 | uint32(ip6[offset+1])<<16 |
				uint32(ip6[offset+2])<<8 | uint32(ip6[offset+3])
		}
	}
	return nil
}

// copyIPToArray copies IP from uint32 array to uint32 array
func copyIPToArray(dst *[4]uint32, src *[4]uint32, ipVersion uint8) {
	if ipVersion == 4 {
		// IPv4: only copy last element
		dst[0] = 0
		dst[1] = 0
		dst[2] = 0
		dst[3] = src[3]
	} else {
		// IPv6: copy all elements
		copy(dst[:], src[:])
	}
}

// ShouldSync determines if a conntrack entry should be synced
func ShouldSync(con ct.Con, config *SyncConfig) bool {
	// Filter by protocol if configured
	if config.SyncTCPOnly {
		// Get protocol from origin tuple
		if con.Origin == nil || con.Origin.Proto == nil || con.Origin.Proto.Number == nil {
			return false
		}
		protoNum := *con.Origin.Proto.Number
		if protoNum != unix.IPPROTO_TCP && protoNum != unix.IPPROTO_UDP {
			return false
		}
	}

	// Filter by connection state if configured
	if config.OnlyEstablished {
		status := getStatus(con.Status)
		if (status & ConntrackStatusAssured) == 0 {
			return false
		}
	}

	return true
}

// FormatConntrackEntry formats a conntrack entry for logging
func FormatConntrackEntry(con ct.Con) string {
	if con.Origin == nil || con.Reply == nil {
		return "invalid conntrack entry"
	}

	proto := "unknown"
	if con.Origin.Proto != nil && con.Origin.Proto.Number != nil {
		switch *con.Origin.Proto.Number {
		case unix.IPPROTO_TCP:
			proto = "TCP"
		case unix.IPPROTO_UDP:
			proto = "UDP"
		case unix.IPPROTO_ICMP:
			proto = "ICMP"
		case unix.IPPROTO_ICMPV6:
			proto = "ICMPv6"
		}
	}

	status := getStatus(con.Status)
	natType := getNATTypeFromStatus(con.Status)
	natStr := "NONE"
	switch natType {
	case NATTypeSNAT:
		natStr = "SNAT"
	case NATTypeDNAT:
		natStr = "DNAT"
	case NATTypeBoth:
		natStr = "BOTH"
	}

	return fmt.Sprintf("%s %s:%d -> %s:%d [NAT=%s, Status=0x%08x]",
		proto,
		formatIP(con.Origin.Src), formatPort(con.Origin.Proto, true),
		formatIP(con.Origin.Dst), formatPort(con.Origin.Proto, false),
		natStr, status,
	)
}

// formatIP formats IP address for display
func formatIP(ip *net.IP) string {
	if ip == nil {
		return "?"
	}
	return ip.String()
}

// formatPort formats port for display
func formatPort(protoTuple *ct.ProtoTuple, isSrc bool) uint16 {
	if protoTuple == nil {
		return 0
	}
	if isSrc && protoTuple.SrcPort != nil {
		return *protoTuple.SrcPort
	}
	if !isSrc && protoTuple.DstPort != nil {
		return *protoTuple.DstPort
	}
	return 0
}

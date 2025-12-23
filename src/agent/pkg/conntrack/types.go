// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: N/A (type definition)
// output: conntrack data structures
// pos: conntrack type definitions - if file updated, must sync with this header comment and pkg/conntrack/CLAUDE.md
package conntrack

import (
	"time"
)

// SyncStats tracks conntrack synchronization statistics
type SyncStats struct {
	TotalEntries    uint64    // Total conntrack entries
	AddedEntries    uint64    // Entries added via NEW events
	UpdatedEntries  uint64    // Entries updated via UPDATE events
	DeletedEntries  uint64    // Entries deleted via DESTROY events
	Errors          uint64    // Number of errors encountered
	LastSyncTime    time.Time // Last successful full sync time
	CacheHits       uint64    // Successful cache lookups
	CacheMisses     uint64    // Failed cache lookups
	ConvertErrors   uint64    // Conversion errors
}

// ConntrackKey represents the key for conntrack cache map
// This matches the BPF structure in nat_support.h
type ConntrackKey struct {
	SrcIP     [4]uint32 // Source IP (post-NAT, 128-bit for IPv4/IPv6)
	DstIP     [4]uint32 // Destination IP (post-NAT, 128-bit)
	SrcPort   uint16    // Source port (post-NAT)
	DstPort   uint16    // Destination port (post-NAT)
	Protocol  uint8     // L4 protocol (TCP/UDP/ICMP)
	IPVersion uint8     // 4 = IPv4, 6 = IPv6
	Pad       uint16    // Padding for alignment
}

// FlowKey represents a flow 5-tuple
// This matches the BPF structure in common_types.h
type FlowKey struct {
	SrcIP     [4]uint32 // Source IP (128-bit for IPv4/IPv6)
	DstIP     [4]uint32 // Destination IP (128-bit)
	SrcPort   uint16    // Source port
	DstPort   uint16    // Destination port
	Protocol  uint8     // L4 protocol
	IPVersion uint8     // 4 = IPv4, 6 = IPv6
	VlanID    uint16    // VLAN ID (0 = no VLAN)
}

// ConntrackEntry represents the value for conntrack cache map
// This matches the BPF structure in nat_support.h
type ConntrackEntry struct {
	OriginalTuple FlowKey   // Original 5-tuple (pre-NAT)
	ReplyTuple    FlowKey   // Reply direction 5-tuple
	Timestamp     uint64    // Last update timestamp (nanoseconds)
	Status        uint32    // Conntrack status flags
	NATType       uint8     // NAT type (NONE/SNAT/DNAT/BOTH)
	Pad           [3]uint8  // Padding for alignment
}

// NAT type constants (must match BPF definitions)
const (
	NATTypeNone = 0
	NATTypeSNAT = 1
	NATTypeDNAT = 2
	NATTypeBoth = 3
)

// Conntrack status flags (from Linux nf_conntrack)
const (
	ConntrackStatusSeenReply = 0x00000002
	ConntrackStatusAssured   = 0x00000004
	ConntrackStatusConfirmed = 0x00000008
	ConntrackStatusSrcNAT    = 0x00000010 // SNAT applied
	ConntrackStatusDstNAT    = 0x00000020 // DNAT applied
	ConntrackStatusNATMask   = 0x00000030 // SNAT | DNAT
)

// SyncConfig configures the conntrack syncer behavior
type SyncConfig struct {
	SyncInterval      time.Duration // Full sync interval (default: 30s)
	EventBufferSize   int           // Event channel buffer size (default: 1000)
	EnableFullSync    bool          // Enable periodic full sync (default: true)
	EnableEventSync   bool          // Enable event-based sync (default: true)
	OnlyEstablished   bool          // Only sync established connections (default: false)
	SyncTCPOnly       bool          // Only sync TCP/UDP connections (default: true)
	MaxRetries        int           // Max retries for failed operations (default: 3)
	RetryDelay        time.Duration // Delay between retries (default: 100ms)
}

// DefaultSyncConfig returns the default sync configuration
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		SyncInterval:    30 * time.Second,
		EventBufferSize: 1000,
		EnableFullSync:  true,
		EnableEventSync: true,
		OnlyEstablished: false,
		SyncTCPOnly:     true,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}
}

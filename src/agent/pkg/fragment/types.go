// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: N/A (type definition)
// output: fragment data structures
// pos: fragment type definitions - if file updated, must sync with this header comment and pkg/fragment/CLAUDE.md
package fragment

import "time"

// FragmentCleanerConfig holds configuration for fragment cleanup
type FragmentCleanerConfig struct {
	// TimeoutDuration is how long to keep fragment state before cleanup
	TimeoutDuration time.Duration

	// ScanInterval is how often to scan the fragment map for expired entries
	ScanInterval time.Duration

	// EnableLogging controls whether to log cleanup events
	EnableLogging bool
}

// FragmentCleanerStats tracks fragment cleaner statistics
type FragmentCleanerStats struct {
	// TotalScans is the total number of scans performed
	TotalScans uint64 `json:"total_scans"`

	// TotalFragmentsScanned is the total number of fragment entries scanned
	TotalFragmentsScanned uint64 `json:"total_fragments_scanned"`

	// TotalTimedOut is the total number of fragments that timed out
	TotalTimedOut uint64 `json:"total_timed_out"`

	// IPv4TimeoutCount is the count of IPv4 fragment timeouts
	IPv4TimeoutCount uint64 `json:"ipv4_timeout_count"`

	// IPv6TimeoutCount is the count of IPv6 fragment timeouts
	IPv6TimeoutCount uint64 `json:"ipv6_timeout_count"`

	// LastScanTime is the timestamp of the last scan
	LastScanTime time.Time `json:"last_scan_time"`

	// LastScanDuration is the duration of the last scan
	LastScanDuration time.Duration `json:"last_scan_duration"`

	// ScanErrors is the count of scan errors
	ScanErrors uint64 `json:"scan_errors"`
}

// FragKey represents the fragment key (must match struct frag_key in fragment_tracking.h)
type FragKey struct {
	SrcIP     [4]uint32 // Source IP address (128 bits, supports IPv4-mapped IPv6)
	DstIP     [4]uint32 // Destination IP address (128 bits)
	FragID    uint32    // IPv4: identification, IPv6: fragment ID
	Protocol  uint8     // IP protocol (TCP/UDP/etc)
	IPVersion uint8     // IP version (4 or 6)
	Pad       [6]uint8  // Padding for alignment (total size: 32+4+1+1+6 = 44 bytes)
}

// FragValue represents the fragment value (must match struct frag_value in fragment_tracking.h)
type FragValue struct {
	// CompleteKey is the complete 5-tuple flow key from the first fragment
	CompleteKey struct {
		SrcIP     [4]uint32
		DstIP     [4]uint32
		SrcPort   uint16
		DstPort   uint16
		Protocol  uint8
		IPVersion uint8
		VlanID    uint16
	}

	Timestamp    uint64 // Fragment timestamp (nanoseconds)
	PolicyAction uint8  // Policy action (ALLOW/DENY)
	Pad          [7]uint8 // Padding for alignment
}

// DefaultFragmentCleanerConfig returns the default configuration
func DefaultFragmentCleanerConfig() FragmentCleanerConfig {
	return FragmentCleanerConfig{
		TimeoutDuration: 30 * time.Second, // 30 seconds default (matches BPF default)
		ScanInterval:    10 * time.Second, // Scan every 10 seconds
		EnableLogging:   true,
	}
}

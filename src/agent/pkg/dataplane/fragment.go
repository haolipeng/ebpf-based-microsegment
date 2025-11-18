// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"fmt"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// FragmentMode defines how fragmented packets are handled
type FragmentMode uint8

const (
	// FragmentModeStrict drops all fragmented packets (safest, may break some apps)
	FragmentModeStrict FragmentMode = 0
	// FragmentModeNormal allows first fragment if policy matches, drops subsequent fragments (recommended)
	FragmentModeNormal FragmentMode = 1
	// FragmentModePermissive allows first fragment and subsequent fragments if first was allowed (least safe)
	FragmentModePermissive FragmentMode = 2
)

// String returns the string representation of FragmentMode
func (m FragmentMode) String() string {
	switch m {
	case FragmentModeStrict:
		return "STRICT"
	case FragmentModeNormal:
		return "NORMAL"
	case FragmentModePermissive:
		return "PERMISSIVE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", m)
	}
}

// FragmentConfig represents fragment handling configuration
type FragmentConfig struct {
	Mode      FragmentMode `json:"mode"`       // Fragment handling mode
	LogEvents bool         `json:"log_events"` // Log fragment events
	TimeoutNs uint64       `json:"timeout_ns"` // Fragment timeout in nanoseconds
}

// fragConfigBPF is the BPF map structure for fragment configuration
// Must match struct frag_config in fragment_tracking.h
type fragConfigBPF struct {
	Mode      uint8    // Fragment mode (FRAG_MODE_*)
	LogEvents uint8    // Boolean: 0 = false, 1 = true
	Reserved  uint16   // Padding
	TimeoutNs uint64   // Fragment timeout in nanoseconds
	Reserved2 [16]byte // Reserved for future use
}

// FragmentStats represents fragment processing statistics
type FragmentStats struct {
	FirstFragments      uint64  `json:"first_fragments"`       // First fragments processed
	SubsequentFragments uint64  `json:"subsequent_fragments"`  // Subsequent fragments processed
	FragmentsAllowed    uint64  `json:"fragments_allowed"`     // Fragments allowed through
	FragmentsDenied     uint64  `json:"fragments_denied"`      // Fragments denied
	FragmentsTimeout    uint64  `json:"fragments_timeout"`     // Fragments timed out
	CacheHits           uint64  `json:"cache_hits"`            // Fragment cache hits
	CacheMisses         uint64  `json:"cache_misses"`          // Fragment cache misses
	IPv4Fragments       uint64  `json:"ipv4_fragments"`        // IPv4 fragments detected
	IPv6Fragments       uint64  `json:"ipv6_fragments"`        // IPv6 fragments detected
	CacheHitRate        float64 `json:"cache_hit_rate"`        // Cache hit rate (calculated)
}

// Fragment statistics map keys (must match enum frag_stats_key in fragment_tracking.h)
const (
	fragStatFirstFragments      = 0
	fragStatSubsequentFragments = 1
	fragStatFragmentsAllowed    = 2
	fragStatFragmentsDenied     = 3
	fragStatFragmentsTimeout    = 4
	fragStatCacheHits           = 5
	fragStatCacheMisses         = 6
	fragStatIPv4Fragments       = 7
	fragStatIPv6Fragments       = 8
	fragStatMax                 = 9
)

// Default fragment timeout (30 seconds)
const DefaultFragmentTimeoutNs = 30 * 1000000000

// SetFragmentConfig configures fragment handling behavior
func (dp *DataPlane) SetFragmentConfig(config *FragmentConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Get fragment config map
	fragConfigMap, err := dp.getFragmentConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get fragment config map: %w", err)
	}

	// Convert to BPF format
	bpfConfig := fragConfigBPF{
		Mode:      uint8(config.Mode),
		LogEvents: boolToUint8(config.LogEvents),
		TimeoutNs: config.TimeoutNs,
	}

	// Update map (key is always 0 for single config entry)
	key := uint32(0)
	if err := fragConfigMap.Update(&key, &bpfConfig, ebpf.UpdateAny); err != nil {
		return fmt.Errorf("failed to update fragment config map: %w", err)
	}

	log.WithFields(log.Fields{
		"mode":       config.Mode.String(),
		"log_events": config.LogEvents,
		"timeout_ns": config.TimeoutNs,
	}).Info("Fragment configuration updated")

	return nil
}

// GetFragmentConfig retrieves current fragment configuration
func (dp *DataPlane) GetFragmentConfig() (*FragmentConfig, error) {
	// Get fragment config map
	fragConfigMap, err := dp.getFragmentConfigMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get fragment config map: %w", err)
	}

	// Read config (key is always 0)
	key := uint32(0)
	var bpfConfig fragConfigBPF
	if err := fragConfigMap.Lookup(&key, &bpfConfig); err != nil {
		return nil, fmt.Errorf("failed to lookup fragment config: %w", err)
	}

	// Convert to Go format
	config := &FragmentConfig{
		Mode:      FragmentMode(bpfConfig.Mode),
		LogEvents: uint8ToBool(bpfConfig.LogEvents),
		TimeoutNs: bpfConfig.TimeoutNs,
	}

	return config, nil
}

// GetFragmentStats retrieves fragment processing statistics
func (dp *DataPlane) GetFragmentStats() (*FragmentStats, error) {
	// Get fragment stats map
	fragStatsMap, err := dp.getFragmentStatsMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get fragment stats map: %w", err)
	}

	stats := &FragmentStats{}

	// Read all statistics
	stats.FirstFragments = dp.readFragmentStat(fragStatsMap, fragStatFirstFragments)
	stats.SubsequentFragments = dp.readFragmentStat(fragStatsMap, fragStatSubsequentFragments)
	stats.FragmentsAllowed = dp.readFragmentStat(fragStatsMap, fragStatFragmentsAllowed)
	stats.FragmentsDenied = dp.readFragmentStat(fragStatsMap, fragStatFragmentsDenied)
	stats.FragmentsTimeout = dp.readFragmentStat(fragStatsMap, fragStatFragmentsTimeout)
	stats.CacheHits = dp.readFragmentStat(fragStatsMap, fragStatCacheHits)
	stats.CacheMisses = dp.readFragmentStat(fragStatsMap, fragStatCacheMisses)
	stats.IPv4Fragments = dp.readFragmentStat(fragStatsMap, fragStatIPv4Fragments)
	stats.IPv6Fragments = dp.readFragmentStat(fragStatsMap, fragStatIPv6Fragments)

	// Calculate cache hit rate
	totalLookups := stats.CacheHits + stats.CacheMisses
	if totalLookups > 0 {
		stats.CacheHitRate = float64(stats.CacheHits) / float64(totalLookups)
	}

	return stats, nil
}

// ResetFragmentStats resets all fragment statistics to zero
func (dp *DataPlane) ResetFragmentStats() error {
	// Get fragment stats map
	fragStatsMap, err := dp.getFragmentStatsMap()
	if err != nil {
		return fmt.Errorf("failed to get fragment stats map: %w", err)
	}

	// Reset all counters
	zeroValue := uint64(0)
	for i := uint32(0); i < fragStatMax; i++ {
		if err := fragStatsMap.Update(&i, &zeroValue, ebpf.UpdateAny); err != nil {
			log.Warnf("Failed to reset fragment stat %d: %v", i, err)
		}
	}

	log.Info("Fragment statistics reset")
	return nil
}

// Helper functions

// getFragmentConfigMap retrieves the fragment config map from loaded programs
func (dp *DataPlane) getFragmentConfigMap() (*ebpf.Map, error) {
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get maps: %w", err)
	}

	if maps.FragConfigMap == nil {
		return nil, fmt.Errorf("fragment config map not available")
	}

	return maps.FragConfigMap, nil
}

// getFragmentStatsMap retrieves the fragment statistics map from loaded programs
func (dp *DataPlane) getFragmentStatsMap() (*ebpf.Map, error) {
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get maps: %w", err)
	}

	if maps.FragStatsMap == nil {
		return nil, fmt.Errorf("fragment stats map not available")
	}

	return maps.FragStatsMap, nil
}

// readFragmentStat reads a single fragment statistic from the per-CPU array map
func (dp *DataPlane) readFragmentStat(statsMap *ebpf.Map, key uint32) uint64 {
	var perCPUValues []uint64

	if err := statsMap.Lookup(&key, &perCPUValues); err != nil {
		log.Debugf("Failed to read fragment stat %d: %v", key, err)
		return 0
	}

	// Sum across all CPUs
	var total uint64
	for _, val := range perCPUValues {
		total += val
	}

	return total
}

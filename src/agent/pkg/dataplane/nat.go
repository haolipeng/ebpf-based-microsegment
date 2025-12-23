// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: NAT config (match mode, cache enable, BPF helper enable)
// output: NAT detection capabilities, conntrack cache updates
// pos: NAT detection and conntrack integration - if file updated, must sync with this header comment and pkg/dataplane/CLAUDE.md
package dataplane

import (
	"fmt"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// NATMatchMode defines how NAT addresses are matched in policies
type NATMatchMode uint8

const (
	// NATMatchModeOriginal matches policies using pre-NAT addresses (recommended)
	NATMatchModeOriginal NATMatchMode = 0
	// NATMatchModeTranslated matches policies using post-NAT addresses
	NATMatchModeTranslated NATMatchMode = 1
	// NATMatchModeBoth tries both original and translated addresses
	NATMatchModeBoth NATMatchMode = 2
)

// NATConfig represents NAT detection configuration
type NATConfig struct {
	MatchMode       NATMatchMode `json:"match_mode"`        // Policy matching mode
	EnableCache     bool         `json:"enable_cache"`      // Enable conntrack cache lookup
	EnableBPFHelper bool         `json:"enable_bpf_helper"` // Enable BPF conntrack helper
	LogEvents       bool         `json:"log_events"`        // Log NAT detection events
}

// natConfigBPF is the BPF map structure for NAT configuration
// Must match struct nat_config in nat_support.h
type natConfigBPF struct {
	MatchMode       uint8    // NAT_MATCH_MODE_* constant
	EnableCache     uint8    // Boolean: 0 = false, 1 = true
	EnableBPFHelper uint8    // Boolean: 0 = false, 1 = true
	LogEvents       uint8    // Boolean: 0 = false, 1 = true
	Reserved        [16]byte // Reserved for future use
}

// NATStats represents NAT detection statistics
type NATStats struct {
	TotalLookups       uint64 `json:"total_lookups"`        // Total NAT lookups attempted
	CacheHits          uint64 `json:"cache_hits"`           // Cache hits (fast path)
	CacheMisses        uint64 `json:"cache_misses"`         // Cache misses (slow path)
	BPFHelperSuccess   uint64 `json:"bpf_helper_success"`   // BPF helper successful lookups
	BPFHelperFailed    uint64 `json:"bpf_helper_failed"`    // BPF helper failures
	SNATDetected       uint64 `json:"snat_detected"`        // SNAT detections
	DNATDetected       uint64 `json:"dnat_detected"`        // DNAT detections
	BothDetected       uint64 `json:"both_detected"`        // Both SNAT and DNAT
	NoNATDetected      uint64 `json:"no_nat_detected"`      // No NAT detected
	RestoreSuccess     uint64 `json:"restore_success"`      // Successfully restored original address
	RestoreFailed      uint64 `json:"restore_failed"`       // Failed to restore original address
	CacheHitRate       float64 `json:"cache_hit_rate"`       // Cache hit rate (calculated)
}

// NAT statistics map keys (must match enum nat_stats_key in nat_support.h)
const (
	natStatsTotalLookups      = 0
	natStatsCacheHits         = 1
	natStatsCacheMisses       = 2
	natStatsBPFHelperSuccess  = 3
	natStatsBPFHelperFailed   = 4
	natStatsSNATDetected      = 5
	natStatsDNATDetected      = 6
	natStatsBothDetected      = 7
	natStatsNoNATDetected     = 8
	natStatsRestoreSuccess    = 9
	natStatsRestoreFailed     = 10
	natStatsMax               = 11
)

// natStatsValue is the BPF map structure for NAT statistics
type natStatsValue struct {
	Count uint64
}

// SetNATConfig configures NAT detection behavior
func (dp *DataPlane) SetNATConfig(config *NATConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Get NAT config map
	natConfigMap, err := dp.getNATConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get NAT config map: %w", err)
	}

	// Convert to BPF format
	bpfConfig := natConfigBPF{
		MatchMode:       uint8(config.MatchMode),
		EnableCache:     boolToUint8(config.EnableCache),
		EnableBPFHelper: boolToUint8(config.EnableBPFHelper),
		LogEvents:       boolToUint8(config.LogEvents),
	}

	// Update map (key is always 0 for single config entry)
	key := uint32(0)
	if err := natConfigMap.Update(&key, &bpfConfig, ebpf.UpdateAny); err != nil {
		return fmt.Errorf("failed to update NAT config map: %w", err)
	}

	log.WithFields(log.Fields{
		"match_mode":        config.MatchMode,
		"enable_cache":      config.EnableCache,
		"enable_bpf_helper": config.EnableBPFHelper,
		"log_events":        config.LogEvents,
	}).Info("NAT configuration updated")

	return nil
}

// GetNATConfig retrieves current NAT configuration
func (dp *DataPlane) GetNATConfig() (*NATConfig, error) {
	// Get NAT config map
	natConfigMap, err := dp.getNATConfigMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get NAT config map: %w", err)
	}

	// Read config (key is always 0)
	key := uint32(0)
	var bpfConfig natConfigBPF
	if err := natConfigMap.Lookup(&key, &bpfConfig); err != nil {
		return nil, fmt.Errorf("failed to lookup NAT config: %w", err)
	}

	// Convert to Go format
	config := &NATConfig{
		MatchMode:       NATMatchMode(bpfConfig.MatchMode),
		EnableCache:     uint8ToBool(bpfConfig.EnableCache),
		EnableBPFHelper: uint8ToBool(bpfConfig.EnableBPFHelper),
		LogEvents:       uint8ToBool(bpfConfig.LogEvents),
	}

	return config, nil
}

// GetNATStats retrieves NAT detection statistics
func (dp *DataPlane) GetNATStats() (*NATStats, error) {
	// Get NAT stats map
	natStatsMap, err := dp.getNATStatsMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get NAT stats map: %w", err)
	}

	stats := &NATStats{}

	// Read all statistics
	stats.TotalLookups = dp.readNATStat(natStatsMap, natStatsTotalLookups)
	stats.CacheHits = dp.readNATStat(natStatsMap, natStatsCacheHits)
	stats.CacheMisses = dp.readNATStat(natStatsMap, natStatsCacheMisses)
	stats.BPFHelperSuccess = dp.readNATStat(natStatsMap, natStatsBPFHelperSuccess)
	stats.BPFHelperFailed = dp.readNATStat(natStatsMap, natStatsBPFHelperFailed)
	stats.SNATDetected = dp.readNATStat(natStatsMap, natStatsSNATDetected)
	stats.DNATDetected = dp.readNATStat(natStatsMap, natStatsDNATDetected)
	stats.BothDetected = dp.readNATStat(natStatsMap, natStatsBothDetected)
	stats.NoNATDetected = dp.readNATStat(natStatsMap, natStatsNoNATDetected)
	stats.RestoreSuccess = dp.readNATStat(natStatsMap, natStatsRestoreSuccess)
	stats.RestoreFailed = dp.readNATStat(natStatsMap, natStatsRestoreFailed)

	// Calculate cache hit rate
	totalLookups := stats.CacheHits + stats.CacheMisses
	if totalLookups > 0 {
		stats.CacheHitRate = float64(stats.CacheHits) / float64(totalLookups)
	}

	return stats, nil
}

// ResetNATStats resets all NAT statistics to zero
func (dp *DataPlane) ResetNATStats() error {
	// Get NAT stats map
	natStatsMap, err := dp.getNATStatsMap()
	if err != nil {
		return fmt.Errorf("failed to get NAT stats map: %w", err)
	}

	// Reset all counters
	zeroValue := natStatsValue{Count: 0}
	for i := uint32(0); i < natStatsMax; i++ {
		if err := natStatsMap.Update(&i, &zeroValue, ebpf.UpdateAny); err != nil {
			log.Warnf("Failed to reset NAT stat %d: %v", i, err)
		}
	}

	log.Info("NAT statistics reset")
	return nil
}

// Helper functions

// getNATConfigMap retrieves the NAT config map from loaded programs
func (dp *DataPlane) getNATConfigMap() (*ebpf.Map, error) {
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get maps: %w", err)
	}

	if maps.NATConfigMap == nil {
		return nil, fmt.Errorf("NAT config map not available")
	}

	return maps.NATConfigMap, nil
}

// getNATStatsMap retrieves the NAT statistics map from loaded programs
func (dp *DataPlane) getNATStatsMap() (*ebpf.Map, error) {
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get maps: %w", err)
	}

	if maps.NATStatsMap == nil {
		return nil, fmt.Errorf("NAT stats map not available")
	}

	return maps.NATStatsMap, nil
}

// readNATStat reads a single NAT statistic from the per-CPU array map
func (dp *DataPlane) readNATStat(statsMap *ebpf.Map, key uint32) uint64 {
	var values []natStatsValue
	if err := statsMap.Lookup(&key, &values); err != nil {
		log.Debugf("Failed to read NAT stat %d: %v", key, err)
		return 0
	}

	// Sum across all CPUs
	var total uint64
	for _, v := range values {
		total += v.Count
	}

	return total
}

// boolToUint8 converts bool to uint8 for BPF maps
func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// uint8ToBool converts uint8 to bool from BPF maps
func uint8ToBool(u uint8) bool {
	return u != 0
}

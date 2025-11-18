// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package fragment

import (
	"testing"
	"time"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestFragmentMaps creates test fragment maps for testing
func createTestFragmentMaps(t *testing.T) (*ebpf.Map, *ebpf.Map) {
	// Create fragment state map
	// KeySize: struct frag_key = 4*4 (src_ip) + 4*4 (dst_ip) + 4 (frag_id) + 1 (protocol) + 1 (ip_version) + 2 (pad) = 42 -> padded to 44
	// ValueSize: struct frag_value = 40 (flow_key) + 8 (timestamp) + 1 (policy_action) + 7 (reserved) = 56
	fragStateMap, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LRUHash,
		KeySize:    uint32(44), // sizeof(struct frag_key) with padding
		ValueSize:  uint32(56), // sizeof(struct frag_value)
		MaxEntries: 100,
	})
	require.NoError(t, err)

	// Create fragment stats map
	fragStatsMap, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.PerCPUArray,
		KeySize:    4,
		ValueSize:  8,
		MaxEntries: 9, // FRAG_STAT_MAX
	})
	require.NoError(t, err)

	return fragStateMap, fragStatsMap
}

// TestNewFragmentCleaner tests cleaner creation
func TestNewFragmentCleaner(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	config := DefaultFragmentCleanerConfig()
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	assert.NotNil(t, cleaner)
	assert.Equal(t, config.TimeoutDuration, cleaner.config.TimeoutDuration)
	assert.Equal(t, config.ScanInterval, cleaner.config.ScanInterval)
}

// TestFragmentCleanerStartStop tests starting and stopping the cleaner
func TestFragmentCleanerStartStop(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	config := DefaultFragmentCleanerConfig()
	config.ScanInterval = 100 * time.Millisecond
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	// Start the cleaner
	err := cleaner.Start()
	require.NoError(t, err)

	// Wait for initial delay (5s) + some scans
	time.Sleep(6 * time.Second)

	// Stop the cleaner
	err = cleaner.Stop()
	require.NoError(t, err)

	// Verify statistics were updated
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalScans, uint64(0))
}

// TestFragmentTimeout tests fragment timeout cleanup
func TestFragmentTimeout(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	// Create cleaner with short timeout
	config := DefaultFragmentCleanerConfig()
	config.TimeoutDuration = 100 * time.Millisecond
	config.ScanInterval = 50 * time.Millisecond
	config.EnableLogging = false
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	// Insert a fragment entry with old timestamp
	nowNS := uint64(time.Now().UnixNano())
	oldTimestamp := nowNS - uint64(200*time.Millisecond) // 200ms ago (should timeout)

	key := FragKey{
		SrcIP:     [4]uint32{0, 0, 0, 0x0100007f}, // 127.0.0.1 (little-endian)
		DstIP:     [4]uint32{0, 0, 0, 0x0200007f}, // 127.0.0.2
		FragID:    12345,
		Protocol:  6, // TCP
		IPVersion: 4,
	}

	value := FragValue{
		Timestamp:    oldTimestamp,
		PolicyAction: 1, // ALLOW
	}

	err := fragStateMap.Update(&key, &value, ebpf.UpdateAny)
	require.NoError(t, err)

	// Verify fragment was inserted
	var readValue FragValue
	err = fragStateMap.Lookup(&key, &readValue)
	require.NoError(t, err)
	assert.Equal(t, oldTimestamp, readValue.Timestamp)

	// Start cleaner
	err = cleaner.Start()
	require.NoError(t, err)

	// Wait for cleanup to occur (need to wait for initial 5s delay + scan time)
	time.Sleep(6 * time.Second)

	// Stop cleaner
	err = cleaner.Stop()
	require.NoError(t, err)

	// Verify fragment was removed
	err = fragStateMap.Lookup(&key, &readValue)
	assert.Error(t, err) // Should return ErrKeyNotExist

	// Verify statistics
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalScans, uint64(0))
	assert.Greater(t, stats.TotalFragmentsScanned, uint64(0))
	assert.Equal(t, uint64(1), stats.TotalTimedOut)
	assert.Equal(t, uint64(1), stats.IPv4TimeoutCount)
	assert.Equal(t, uint64(0), stats.IPv6TimeoutCount)
}

// TestFragmentNotTimeout tests that non-expired fragments are not removed
func TestFragmentNotTimeout(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	// Create cleaner with long timeout (longer than test duration)
	config := DefaultFragmentCleanerConfig()
	config.TimeoutDuration = 30 * time.Second // Much longer than test duration
	config.ScanInterval = 50 * time.Millisecond
	config.EnableLogging = false
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	// Start cleaner first
	err := cleaner.Start()
	require.NoError(t, err)

	// Wait for scan loop to start
	time.Sleep(6 * time.Second)

	// Insert a fragment entry with current timestamp (just now)
	nowNS := uint64(time.Now().UnixNano())

	key := FragKey{
		SrcIP:     [4]uint32{0, 0, 0, 0x0100007f},
		DstIP:     [4]uint32{0, 0, 0, 0x0200007f},
		FragID:    54321,
		Protocol:  17, // UDP
		IPVersion: 4,
	}

	value := FragValue{
		Timestamp:    nowNS,
		PolicyAction: 1,
	}

	err = fragStateMap.Update(&key, &value, ebpf.UpdateAny)
	require.NoError(t, err)

	// Wait a bit more for scans to run
	time.Sleep(1 * time.Second)

	// Stop cleaner
	err = cleaner.Stop()
	require.NoError(t, err)

	// Verify fragment is still present (not timed out)
	var readValue FragValue
	err = fragStateMap.Lookup(&key, &readValue)
	require.NoError(t, err)
	assert.Equal(t, nowNS, readValue.Timestamp)

	// Verify statistics show no timeouts
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalScans, uint64(0))
	assert.Greater(t, stats.TotalFragmentsScanned, uint64(0))
	assert.Equal(t, uint64(0), stats.TotalTimedOut)
}

// TestIPv6FragmentTimeout tests IPv6 fragment timeout
func TestIPv6FragmentTimeout(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	config := DefaultFragmentCleanerConfig()
	config.TimeoutDuration = 100 * time.Millisecond
	config.ScanInterval = 50 * time.Millisecond
	config.EnableLogging = false
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	// Insert an IPv6 fragment entry with old timestamp
	nowNS := uint64(time.Now().UnixNano())
	oldTimestamp := nowNS - uint64(200*time.Millisecond)

	key := FragKey{
		SrcIP:     [4]uint32{0x20010db8, 0, 0, 1}, // 2001:db8::1
		DstIP:     [4]uint32{0x20010db8, 0, 0, 2}, // 2001:db8::2
		FragID:    98765,
		Protocol:  6,
		IPVersion: 6, // IPv6
	}

	value := FragValue{
		Timestamp:    oldTimestamp,
		PolicyAction: 1,
	}

	err := fragStateMap.Update(&key, &value, ebpf.UpdateAny)
	require.NoError(t, err)

	// Start cleaner
	err = cleaner.Start()
	require.NoError(t, err)

	// Wait for cleanup (initial delay + scan time)
	time.Sleep(6 * time.Second)

	// Stop cleaner
	err = cleaner.Stop()
	require.NoError(t, err)

	// Verify fragment was removed
	var readValue FragValue
	err = fragStateMap.Lookup(&key, &readValue)
	assert.Error(t, err)

	// Verify IPv6 timeout count
	stats := cleaner.GetStats()
	assert.Equal(t, uint64(1), stats.TotalTimedOut)
	assert.Equal(t, uint64(0), stats.IPv4TimeoutCount)
	assert.Equal(t, uint64(1), stats.IPv6TimeoutCount)
}

// TestResetStats tests statistics reset
func TestResetStats(t *testing.T) {
	fragStateMap, fragStatsMap := createTestFragmentMaps(t)
	defer fragStateMap.Close()
	defer fragStatsMap.Close()

	config := DefaultFragmentCleanerConfig()
	config.ScanInterval = 50 * time.Millisecond
	config.EnableLogging = false
	cleaner := NewFragmentCleaner(fragStateMap, fragStatsMap, config)

	// Start and run for a bit
	err := cleaner.Start()
	require.NoError(t, err)

	time.Sleep(6 * time.Second)

	err = cleaner.Stop()
	require.NoError(t, err)

	// Verify stats exist
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalScans, uint64(0))

	// Reset stats
	cleaner.ResetStats()

	// Verify stats are reset
	stats = cleaner.GetStats()
	assert.Equal(t, uint64(0), stats.TotalScans)
	assert.Equal(t, uint64(0), stats.TotalFragmentsScanned)
	assert.Equal(t, uint64(0), stats.TotalTimedOut)
}

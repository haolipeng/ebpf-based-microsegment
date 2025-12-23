// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: fragment timeout config
// output: expired fragment cleanup operations
// pos: fragment state cleanup manager - if file updated, must sync with this header comment and pkg/fragment/CLAUDE.md
package fragment

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// FragmentCleaner manages fragment state cleanup and timeout
type FragmentCleaner struct {
	fragStateMap *ebpf.Map
	fragStatsMap *ebpf.Map
	config       FragmentCleanerConfig

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats      FragmentCleanerStats
	statsMutex sync.RWMutex
}

// NewFragmentCleaner creates a new fragment cleaner
func NewFragmentCleaner(fragStateMap, fragStatsMap *ebpf.Map, config FragmentCleanerConfig) *FragmentCleaner {
	ctx, cancel := context.WithCancel(context.Background())

	return &FragmentCleaner{
		fragStateMap: fragStateMap,
		fragStatsMap: fragStatsMap,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		stats:        FragmentCleanerStats{},
	}
}

// Start begins the fragment cleanup loop
func (fc *FragmentCleaner) Start() error {
	log.Info("[Fragment Cleaner] Starting fragment cleaner...")

	fc.wg.Add(1)
	go fc.scanLoop()

	log.Infof("[Fragment Cleaner] Fragment cleaner started (scan interval: %v, timeout: %v)",
		fc.config.ScanInterval, fc.config.TimeoutDuration)
	return nil
}

// Stop gracefully stops the fragment cleaner
func (fc *FragmentCleaner) Stop() error {
	log.Info("[Fragment Cleaner] Stopping fragment cleaner...")
	fc.cancel()
	fc.wg.Wait()
	log.Info("[Fragment Cleaner] Fragment cleaner stopped")
	return nil
}

// scanLoop periodically scans the fragment map for expired entries
func (fc *FragmentCleaner) scanLoop() {
	defer fc.wg.Done()

	// Initial delay before first scan
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(fc.config.ScanInterval)
	defer ticker.Stop()

	log.Infof("[Fragment Cleaner] Scan loop started (timeout: %v, scan interval: %v)",
		fc.config.TimeoutDuration, fc.config.ScanInterval)

	for {
		select {
		case <-fc.ctx.Done():
			log.Info("[Fragment Cleaner] Scan loop stopped")
			return

		case <-ticker.C:
			if err := fc.runScan(); err != nil {
				log.Errorf("[Fragment Cleaner] Scan failed: %v", err)
				fc.statsMutex.Lock()
				fc.stats.ScanErrors++
				fc.statsMutex.Unlock()
			}
		}
	}
}

// timedOutFragment stores information about a timed-out fragment entry
type timedOutFragment struct {
	key   FragKey
	value FragValue
}

// runScan executes a single fragment cleanup scan
func (fc *FragmentCleaner) runScan() error {
	startTime := time.Now()
	nowNS := uint64(startTime.UnixNano())
	timeoutNS := uint64(fc.config.TimeoutDuration.Nanoseconds())

	log.Debug("[Fragment Cleaner] Starting fragment cleanup scan...")

	// Collect timed-out fragment entries
	var timedOutFragments []timedOutFragment
	var ipv4Count, ipv6Count uint64
	fragmentsScanned := uint64(0)

	var key FragKey
	var value FragValue

	// Iterate through all fragment entries
	iter := fc.fragStateMap.Iterate()
	for iter.Next(&key, &value) {
		fragmentsScanned++

		// Calculate elapsed time since fragment was cached
		elapsed := nowNS - value.Timestamp

		// Check if fragment has timed out
		if elapsed > timeoutNS {
			timedOutFragments = append(timedOutFragments, timedOutFragment{
				key:   key,
				value: value,
			})

			// Count by IP version
			if key.IPVersion == 4 {
				ipv4Count++
			} else if key.IPVersion == 6 {
				ipv6Count++
			}
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("iteration error: %w", err)
	}

	// Delete timed-out fragments and log events
	deletedCount := 0
	for _, frag := range timedOutFragments {
		if err := fc.fragStateMap.Delete(&frag.key); err != nil {
			log.Debugf("[Fragment Cleaner] Failed to delete fragment: %v", err)
		} else {
			deletedCount++

			// Log detailed timeout event if logging is enabled
			if fc.config.EnableLogging {
				srcIP := ipv6ToNetIP(frag.key.SrcIP, frag.key.IPVersion)
				dstIP := ipv6ToNetIP(frag.key.DstIP, frag.key.IPVersion)
				protocol := protocolToString(frag.key.Protocol)

				elapsedSec := float64(nowNS-frag.value.Timestamp) / 1e9

				log.Infof("[FRAGMENT TIMEOUT] %s -> %s proto=%s frag_id=%d ip_version=%d elapsed=%.1fs",
					srcIP, dstIP, protocol, frag.key.FragID, frag.key.IPVersion, elapsedSec)
			}
		}
	}

	// Update BPF fragment timeout statistics if any fragments were deleted
	if deletedCount > 0 && fc.fragStatsMap != nil {
		fc.updateBPFTimeoutStat(uint64(deletedCount))
	}

	duration := time.Since(startTime)

	// Update internal statistics
	fc.statsMutex.Lock()
	fc.stats.TotalScans++
	fc.stats.TotalFragmentsScanned += fragmentsScanned
	fc.stats.TotalTimedOut += uint64(deletedCount)
	fc.stats.IPv4TimeoutCount += ipv4Count
	fc.stats.IPv6TimeoutCount += ipv6Count
	fc.stats.LastScanTime = time.Now()
	fc.stats.LastScanDuration = duration
	fc.statsMutex.Unlock()

	// Log scan result
	if deletedCount > 0 {
		log.Infof("[Fragment Cleaner] Scan completed: scanned %d fragments, deleted %d (IPv4: %d, IPv6: %d, duration: %v)",
			fragmentsScanned, deletedCount, ipv4Count, ipv6Count, duration)
	} else {
		log.Debugf("[Fragment Cleaner] Scan completed: scanned %d fragments, no timeouts (duration: %v)",
			fragmentsScanned, duration)
	}

	return nil
}

// updateBPFTimeoutStat updates the FRAG_STAT_FRAGMENTS_TIMEOUT counter in BPF
func (fc *FragmentCleaner) updateBPFTimeoutStat(count uint64) {
	// FRAG_STAT_FRAGMENTS_TIMEOUT = 4 (from fragment_tracking.h)
	fragStatKey := uint32(4)

	// Read current per-CPU values
	var perCPUValues []uint64
	if err := fc.fragStatsMap.Lookup(&fragStatKey, &perCPUValues); err != nil {
		log.Debugf("[Fragment Cleaner] Failed to read timeout stat: %v", err)
		return
	}

	// Increment the first CPU's counter (simple approach)
	// Note: In a multi-CPU system, we could distribute the count,
	// but for timeout stats, incrementing on CPU 0 is sufficient
	if len(perCPUValues) > 0 {
		perCPUValues[0] += count

		if err := fc.fragStatsMap.Update(&fragStatKey, &perCPUValues, ebpf.UpdateAny); err != nil {
			log.Debugf("[Fragment Cleaner] Failed to update timeout stat: %v", err)
		}
	}
}

// GetStats returns current fragment cleaner statistics
func (fc *FragmentCleaner) GetStats() FragmentCleanerStats {
	fc.statsMutex.RLock()
	defer fc.statsMutex.RUnlock()
	return fc.stats
}

// ResetStats resets the fragment cleaner statistics
func (fc *FragmentCleaner) ResetStats() {
	fc.statsMutex.Lock()
	defer fc.statsMutex.Unlock()
	fc.stats = FragmentCleanerStats{}
}

// ipv6ToNetIP converts [4]uint32 IPv6 address to net.IP
// Handles both native IPv6 and IPv4-mapped IPv6 addresses
func ipv6ToNetIP(ipv6 [4]uint32, ipVersion uint8) net.IP {
	// Check if this is IPv4 or IPv4-mapped IPv6
	if ipVersion == 4 || (ipv6[0] == 0 && ipv6[1] == 0 && ipv6[2] == 0x0000ffff) {
		// Extract IPv4 address from last 32 bits (little-endian)
		ip := ipv6[3]
		return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
	}

	// Native IPv6 address (convert 4 x uint32 to 16 bytes)
	ipv6Bytes := make(net.IP, 16)
	for i := 0; i < 4; i++ {
		// Little-endian conversion
		ipv6Bytes[i*4] = byte(ipv6[i])
		ipv6Bytes[i*4+1] = byte(ipv6[i] >> 8)
		ipv6Bytes[i*4+2] = byte(ipv6[i] >> 16)
		ipv6Bytes[i*4+3] = byte(ipv6[i] >> 24)
	}
	return ipv6Bytes
}

// protocolToString converts protocol number to string
func protocolToString(protocol uint8) string {
	switch protocol {
	case 6: // TCP
		return "TCP"
	case 17: // UDP
		return "UDP"
	case 1: // ICMP
		return "ICMP"
	case 58: // ICMPv6
		return "ICMPv6"
	default:
		return fmt.Sprintf("PROTO_%d", protocol)
	}
}

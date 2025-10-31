// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"

	log "github.com/sirupsen/logrus"
)

var (
	ifaceName     = flag.String("iface", "lo", "Network interface to attach eBPF program")
	duration      = flag.Int("duration", 30, "Test duration in seconds")
	statsInterval = flag.Int("interval", 5, "Statistics reporting interval in seconds")
)

func main() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	log.Info("=== eBPF Microsegmentation Performance Test ===")
	log.Infof("Interface: %s", *ifaceName)
	log.Infof("Duration: %d seconds", *duration)
	log.Infof("Stats Interval: %d seconds", *statsInterval)
	log.Info("================================================")

	// Initialize DataPlane
	dp, err := dataplane.New(*ifaceName)
	if err != nil {
		log.Fatalf("Failed to create data plane: %v", err)
	}
	defer dp.Close()

	log.Info("✓ eBPF programs loaded and attached successfully")

	// Initialize Policy Manager
	pm := policy.NewManager(dp)

	// Add test policies
	testPolicies := []policy.Policy{
		{
			SrcIP:    "0.0.0.0",
			DstIP:    "127.0.0.1",
			DstPort:  80,
			Protocol: "tcp",
			Action:   "allow",
			RuleID:   1001,
			Priority: 100,
		},
		{
			SrcIP:    "0.0.0.0",
			DstIP:    "127.0.0.1",
			DstPort:  443,
			Protocol: "tcp",
			Action:   "allow",
			RuleID:   1002,
			Priority: 100,
		},
		{
			SrcIP:    "0.0.0.0",
			DstIP:    "192.168.1.100",
			DstPort:  22,
			Protocol: "tcp",
			Action:   "deny",
			RuleID:   2001,
			Priority: 200,
		},
	}

	for i := range testPolicies {
		err = pm.AddPolicy(&testPolicies[i])
		if err != nil {
			log.Errorf("Failed to add policy: %v", err)
		}
	}
	log.Infof("✓ Added %d test policies", len(testPolicies))

	// Start flow event monitor
	go dp.MonitorFlowEvents()

	// Get baseline statistics
	log.Info("\n=== Baseline Statistics ===")
	baselineStats := dp.GetStatistics()
	printStats(baselineStats)

	// Start periodic stats reporting
	ticker := time.NewTicker(time.Duration(*statsInterval) * time.Second)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		lastStats := baselineStats
		for {
			select {
			case <-ticker.C:
				currentStats := dp.GetStatistics()
				log.Info("\n=== Current Statistics ===")
				printStats(currentStats)

				// Calculate rates
				deltaStats := calculateDelta(currentStats, lastStats)
				log.Info("\n=== Delta Statistics (last interval) ===")
				printStats(deltaStats)

				pps := float64(deltaStats.TotalPackets) / float64(*statsInterval)
				log.Infof("Packet Rate: %.2f pps", pps)

				if deltaStats.TotalPackets > 0 {
					allowRate := float64(deltaStats.AllowedPackets) / float64(deltaStats.TotalPackets) * 100
					denyRate := float64(deltaStats.DeniedPackets) / float64(deltaStats.TotalPackets) * 100
					hitRate := float64(deltaStats.PolicyHits) / float64(deltaStats.TotalPackets) * 100

					log.Infof("Allow Rate: %.2f%%", allowRate)
					log.Infof("Deny Rate: %.2f%%", denyRate)
					log.Infof("Cache Hit Rate: %.2f%%", hitRate)
				}

				lastStats = currentStats
			case <-done:
				return
			}
		}
	}()

	// Wait for test duration or interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(time.Duration(*duration) * time.Second):
		log.Info("\n=== Test duration completed ===")
	case <-sigChan:
		log.Info("\n=== Test interrupted by user ===")
	}

	done <- true

	// Final statistics
	log.Info("\n=== Final Statistics ===")
	finalStats := dp.GetStatistics()
	printStats(finalStats)

	// Calculate overall metrics
	totalStats := calculateDelta(finalStats, baselineStats)
	log.Info("\n=== Total Test Statistics ===")
	printStats(totalStats)

	if totalStats.TotalPackets > 0 {
		avgPps := float64(totalStats.TotalPackets) / float64(*duration)
		log.Infof("Average Packet Rate: %.2f pps", avgPps)

		allowRate := float64(totalStats.AllowedPackets) / float64(totalStats.TotalPackets) * 100
		denyRate := float64(totalStats.DeniedPackets) / float64(totalStats.TotalPackets) * 100

		log.Infof("Overall Allow Rate: %.2f%%", allowRate)
		log.Infof("Overall Deny Rate: %.2f%%", denyRate)

		// Estimate latency (very rough)
		// For <10μs target, we need to process >100k pps per core
		if avgPps > 100000 {
			log.Info("✓ Performance Target: LIKELY MET (<10μs per packet)")
		} else {
			log.Info("⚠ Performance Target: NEEDS VALIDATION (actual latency measurement required)")
		}
	} else {
		log.Warn("No packets processed during test")
		log.Info("Tip: Generate traffic to test, e.g., 'ping 127.0.0.1' or 'curl http://127.0.0.1'")
	}

	log.Info("\n=== Test Complete ===")
}

func printStats(stats dataplane.Statistics) {
	log.Infof("  Total Packets:     %d", stats.TotalPackets)
	log.Infof("  Allowed Packets:   %d", stats.AllowedPackets)
	log.Infof("  Denied Packets:    %d", stats.DeniedPackets)
	log.Infof("  New Sessions:      %d", stats.NewSessions)
	log.Infof("  Closed Sessions:   %d", stats.ClosedSessions)
	log.Infof("  Active Sessions:   %d", stats.ActiveSessions)
	log.Infof("  Policy Hits:       %d", stats.PolicyHits)
	log.Infof("  Policy Misses:     %d", stats.PolicyMisses)
}

func calculateDelta(current, previous dataplane.Statistics) dataplane.Statistics {
	return dataplane.Statistics{
		TotalPackets:   current.TotalPackets - previous.TotalPackets,
		AllowedPackets: current.AllowedPackets - previous.AllowedPackets,
		DeniedPackets:  current.DeniedPackets - previous.DeniedPackets,
		NewSessions:    current.NewSessions - previous.NewSessions,
		ClosedSessions: current.ClosedSessions - previous.ClosedSessions,
		ActiveSessions: current.ActiveSessions, // Absolute value, not delta
		PolicyHits:     current.PolicyHits - previous.PolicyHits,
		PolicyMisses:   current.PolicyMisses - previous.PolicyMisses,
	}
}

// Generate test traffic (optional helper)
func generateTestTraffic(targetIP string, port int, count int) {
	log.Infof("Generating %d test connections to %s:%d", count, targetIP, port)

	for i := 0; i < count; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, port), 1*time.Second)
		if err == nil {
			conn.Close()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

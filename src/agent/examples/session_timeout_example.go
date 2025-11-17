// Example: Session timeout manager usage
//
// This example demonstrates how to enable and configure session timeout management
// in the microsegmentation data plane.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/session"
)

func main() {
	// Set log level
	log.SetLevel(log.InfoLevel)

	// Specify network interface (change to match your system)
	iface := "eth0" // Example: eth0, ens33, enp0s3

	log.Infof("Starting session timeout example on interface %s", iface)

	// 1. Create data plane instance
	dp, err := dataplane.New(iface)
	if err != nil {
		log.Fatalf("Failed to create data plane: %v", err)
	}
	defer dp.Close()

	log.Info("✓ Data plane initialized")

	// 2. Configure session timeout parameters
	timeoutConfig := session.SessionTimeoutConfig{
		// TCP session idle timeout (no packets)
		TCPIdleTimeout: 5 * time.Minute,

		// TCP session close timeout (after FIN/RST)
		TCPCloseTimeout: 30 * time.Second,

		// UDP session idle timeout
		UDPIdleTimeout: 1 * time.Minute,

		// Maximum session duration (prevents indefinite sessions)
		MaxSessionDuration: 1 * time.Hour,

		// How often to scan the session map for timeouts
		ScanInterval: 30 * time.Second,
	}

	// 3. Enable session timeout management
	if err := dp.EnableSessionTimeout(timeoutConfig); err != nil {
		log.Fatalf("Failed to enable session timeout: %v", err)
	}

	log.Info("✓ Session timeout manager enabled")

	// 4. Start flow event monitoring (runs in goroutine)
	go dp.MonitorFlowEvents()

	// 5. Periodically display statistics
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Get data plane statistics
			stats := dp.GetStatistics()
			fmt.Printf("\n=== Data Plane Statistics ===\n")
			fmt.Printf("Total Packets:   %d\n", stats.TotalPackets)
			fmt.Printf("Allowed:         %d\n", stats.AllowedPackets)
			fmt.Printf("Denied:          %d\n", stats.DeniedPackets)
			fmt.Printf("New Sessions:    %d\n", stats.NewSessions)
			fmt.Printf("Closed Sessions: %d\n", stats.ClosedSessions)
			fmt.Printf("Active Sessions: %d\n", stats.ActiveSessions)

			// Get timeout manager statistics
			timeoutStats := dp.GetTimeoutStats()
			fmt.Printf("\n=== Session Timeout Statistics ===\n")
			fmt.Printf("Total Scans:     %d\n", timeoutStats.TotalScans)
			fmt.Printf("Sessions Scanned: %d\n", timeoutStats.TotalSessionsScanned)
			fmt.Printf("Total Timed Out: %d\n", timeoutStats.TotalTimedOut)
			fmt.Printf("  TCP Idle:      %d\n", timeoutStats.TCPIdleTimeouts)
			fmt.Printf("  TCP Close:     %d\n", timeoutStats.TCPCloseTimeouts)
			fmt.Printf("  UDP Idle:      %d\n", timeoutStats.UDPIdleTimeouts)
			fmt.Printf("  Max Duration:  %d\n", timeoutStats.MaxDurationTimeouts)
			if !timeoutStats.LastScanTime.IsZero() {
				fmt.Printf("Last Scan:       %s (took %v)\n",
					timeoutStats.LastScanTime.Format("15:04:05"),
					timeoutStats.LastScanDuration)
			}
			fmt.Printf("Scan Errors:     %d\n", timeoutStats.ScanErrors)
		}
	}()

	// 6. Wait for interrupt signal
	log.Info("Session timeout manager running. Press Ctrl+C to stop.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info("\nShutting down...")
}

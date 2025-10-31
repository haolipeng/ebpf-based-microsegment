// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/api"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	iface         string
	logLevel      string
	statsInterval int
	enableAPI     bool
	apiHost       string
	apiPort       int
)

var rootCmd = &cobra.Command{
	Use:   "microsegment-agent",
	Short: "eBPF-based microsegmentation agent",
	Long:  `A high-performance microsegmentation agent using eBPF for packet filtering and policy enforcement`,
	Run:   runAgent,
}

func init() {
	rootCmd.Flags().StringVarP(&iface, "interface", "i", "lo", "Network interface to attach eBPF program")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().IntVarP(&statsInterval, "stats-interval", "s", 5, "Statistics print interval in seconds")
	rootCmd.Flags().BoolVarP(&enableAPI, "enable-api", "a", true, "Enable REST API server")
	rootCmd.Flags().StringVar(&apiHost, "api-host", "127.0.0.1", "API server host")
	rootCmd.Flags().IntVar(&apiPort, "api-port", 8080, "API server port")
}

func runAgent(cmd *cobra.Command, args []string) {
	// Setup logging
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	log.Infof("Starting microsegmentation agent on interface %s", iface)

	// Create data plane
	dp, err := dataplane.New(iface)
	if err != nil {
		log.Fatalf("Failed to create data plane: %v", err)
	}
	defer dp.Close()

	log.Info("✓ Data plane initialized")

	// Create policy manager
	pm := policy.NewManager(dp)

	// Add default allow-all policy for testing
	err = pm.AddPolicy(&policy.Policy{
		RuleID:   1,
		SrcIP:    "0.0.0.0/0",
		DstIP:    "0.0.0.0/0",
		DstPort:  0,
		Protocol: "any",
		Action:   "allow",
	})
	if err != nil {
		log.Warnf("Failed to add default policy: %v", err)
	}

	log.Info("✓ Policy manager initialized")

	// Start API server if enabled
	var apiServer *api.Server
	if enableAPI {
		apiConfig := &api.Config{
			Host:       apiHost,
			Port:       apiPort,
			EnableCORS: true,
			LogLevel:   logLevel,
		}

		apiServer, err = api.NewAPIServer(apiConfig, dp, pm)
		if err != nil {
			log.Fatalf("Failed to create API server: %v", err)
		}

		if err := apiServer.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}

		log.Infof("✓ API server started on http://%s:%d", apiHost, apiPort)
	}

	// Start flow event monitoring
	go dp.MonitorFlowEvents()

	// Print statistics periodically
	ticker := time.NewTicker(time.Duration(statsInterval) * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			stats := dp.GetStatistics()
			log.Info("=== Statistics ===")
			log.Infof("  Total Packets:    %d", stats.TotalPackets)
			log.Infof("  Allowed Packets:  %d", stats.AllowedPackets)
			log.Infof("  Denied Packets:   %d", stats.DeniedPackets)
			log.Infof("  New Sessions:     %d", stats.NewSessions)
			log.Infof("  Policy Hits:      %d", stats.PolicyHits)
			log.Infof("  Policy Misses:    %d", stats.PolicyMisses)
		}
	}()

	// Wait for interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	log.Info("✓ Agent running. Press Ctrl+C to exit")

	<-sig
	log.Info("Shutting down...")

	// Stop API server if running
	if apiServer != nil {
		if err := apiServer.Stop(); err != nil {
			log.Errorf("Error stopping API server: %v", err)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

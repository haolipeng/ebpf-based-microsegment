// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/client"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/config"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/reporter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "microsegment-agent",
	Short: "eBPF-based microsegmentation agent",
	Long: `A high-performance microsegmentation agent using eBPF for packet filtering and policy enforcement.

Supports two operation modes:
  - agent-server: Connects to control plane server for centralized policy management
  - standalone: Runs independently with local API for debugging and monitoring`,
	Run: runAgent,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (YAML)")
}

func runAgent(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	log.Info("Starting microsegmentation agent")
	log.Infof("Mode: %s", cfg.Mode)
	log.Infof("Interface: %s", cfg.Interface)

	// Create data plane
	dp, err := dataplane.New(cfg.Interface)
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

	// Initialize mode-specific components
	var rep reporter.Reporter
	var agentClient *client.AgentClient
	var flowStorage flow.Storage

	if cfg.IsAgentServerMode() {
		// Agent-Server mode: initialize reporter and server connection
		log.Info("Connecting to control plane server...")
		rep, agentClient = initAgentServerMode(cfg, pm)

		// Start reporter
		if err := rep.Start(); err != nil {
			log.Fatalf("Failed to start reporter: %v", err)
		}
		defer rep.Stop()
	} else {
		// Standalone mode: initialize local SQLite storage
		log.Info("Running in standalone mode (no server connection)")

		flowStorage = initStorage(cfg)
		if flowStorage != nil {
			defer flowStorage.Close()
			log.Info("✓ Standalone mode storage initialized")
		}
	}

	// Initialize flow collector (always enabled as it's core functionality)
	log.Info("Initializing flow collection...")
	flowCollector := initFlowCollector(cfg, dp, flowStorage, rep)
	if flowCollector != nil {
		defer flowCollector.Stop()
		log.Info("✓ Flow collection initialized")
	}

	// Start API server if enabled
	var apiServer *api.Server
	if cfg.API.Enabled {
		apiConfig := &api.Config{
			Host:          cfg.API.Host,
			Port:          cfg.API.Port,
			EnableCORS:    cfg.API.EnableCORS,
			LogLevel:      cfg.LogLevel,
			Interface:     cfg.Interface,
			StatsInterval: cfg.StatsInterval,
		}

		apiServer, err = api.NewAPIServer(apiConfig, dp, pm)
		if err != nil {
			log.Fatalf("Failed to create API server: %v", err)
		}

		// Register flow components with API server
		// flowStorage may be nil in agent-server mode
		if flowCollector != nil {
			apiServer.SetFlowComponents(flowCollector, flowStorage)
			log.Debug("Flow components registered with API server")
		}

		if err := apiServer.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}

		log.Infof("✓ API server started on http://%s:%d", cfg.API.Host, cfg.API.Port)
	}

	// Start flow event monitoring
	go dp.MonitorFlowEvents()

	// Print statistics periodically
	ticker := time.NewTicker(time.Duration(cfg.StatsInterval) * time.Second)
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

			// Update agent metrics if in agent-server mode
			if agentClient != nil {
				flowCount := stats.NewSessions
				policyCount := uint32(pm.GetPolicyCount())
				agentClient.UpdateMetrics(flowCount, policyCount)
			}
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

	// Stop agent client if running
	if agentClient != nil {
		if err := agentClient.Close(); err != nil {
			log.Errorf("Error closing agent client: %v", err)
		}
	}

	log.Info("Shutdown complete")
}

func loadConfiguration() (*config.Config, error) {
	if configPath != "" {
		log.Infof("Loading configuration from %s", configPath)
		return config.LoadConfig(configPath)
	}

	log.Info("No config file specified, using defaults")
	return config.DefaultConfig(), nil
}

func setupLogging(logLevel string) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func initAgentServerMode(cfg *config.Config, pm *policy.PolicyManager) (reporter.Reporter, *client.AgentClient) {
	log.Info("Initializing agent-server mode...")

	agentCfg := cfg.AgentServer

	// Create GRPCReporter with retry configuration
	rep := reporter.NewGRPCReporterWithRetry(
		agentCfg.ServerAddr,
		agentCfg.AgentID,
		agentCfg.BatchSize,
		agentCfg.MaxRetries,
		agentCfg.RetryBaseDelay,
		agentCfg.RetryMaxDelay,
	)

	log.Infof("Flow reporter configured with retry: max_retries=%d, base_delay=%v, max_delay=%v",
		agentCfg.MaxRetries, agentCfg.RetryBaseDelay, agentCfg.RetryMaxDelay)

	// Create AgentClient
	hostname, _ := os.Hostname()
	agentClient := client.NewAgentClient(
		agentCfg.ServerAddr,
		agentCfg.AgentID,
		hostname,
		"1.0.0", // version
	)

	// Connect and register with server
	if err := agentClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	log.Infof("✓ Connected to server at %s", agentCfg.ServerAddr)

	// Start heartbeat goroutine
	go agentClient.StartHeartbeat()

	// Sync policies on startup
	currentVersion := uint64(0) // TODO: Get from policy manager
	if policies, version, err := agentClient.SyncPolicies(currentVersion); err == nil {
		log.Infof("✓ Synced %d policies (version %d)", len(policies), version)

		// Apply policies to policy manager
		if err := pm.SyncPoliciesFromServer(policies, version); err != nil {
			log.Errorf("Failed to apply synced policies: %v", err)
		} else {
			log.Infof("✓ Applied %d policies to data plane", len(policies))
		}
	} else {
		log.Warnf("Failed to sync policies: %v", err)
	}

	return rep, agentClient
}

// initStorage initializes SQLite storage for standalone mode.
// This function should only be called in standalone mode.
func initStorage(cfg *config.Config) flow.Storage {
	log.Info("Initializing local SQLite storage...")

	// Create storage directory if it doesn't exist
	storageDir := "./data"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Errorf("Failed to create storage directory: %v", err)
		return nil
	}

	// Initialize SQLite storage
	storage, err := flow.NewSQLiteStorage(cfg.Flow.StoragePath)
	if err != nil {
		log.Errorf("Failed to create flow storage: %v", err)
		return nil
	}

	log.Infof("✓ SQLite storage initialized at %s", cfg.Flow.StoragePath)
	return storage
}

// initFlowCollector creates and configures the flow collector.
// Behavior varies by operation mode:
// - agent-server mode: configures collector to send flows to server via reporter
// - standalone mode: configures collector to persist flows to local storage
func initFlowCollector(cfg *config.Config, dp *dataplane.DataPlane, storage flow.Storage, rep reporter.Reporter) *flow.Collector {
	// Get Ring Buffer from dataplane
	ringBuf := dp.GetFlowRingBuffer()
	if ringBuf == nil {
		log.Error("Failed to get flow ring buffer from dataplane")
		return nil
	}

	// Create collector configuration
	collectorConfig := flow.CollectorConfig{
		FlowTimeout:       cfg.Flow.FlowTimeout,
		BatchSize:         100,
		EnableEnrichment:  true,
		EnablePersistence: storage != nil, // Only persist if storage is available
		CleanupInterval:   cfg.Flow.CleanupInterval,
	}

	// Create collector
	// - workloadMgr is nil for now, can be added later for Kubernetes integration
	// - storage may be nil in agent-server mode
	collector := flow.NewCollector(ringBuf, storage, nil, collectorConfig)

	// Agent-server mode: configure reporter to send flows to server
	if cfg.IsAgentServerMode() && rep != nil {
		collector.SetReporter(rep)
		log.Info("✓ Collector configured to send flows to server via gRPC")
	}

	// Create and start WebSocket Hub for real-time streaming
	wsHub := flow.NewHub()
	go wsHub.Run()
	collector.SetWebSocketHub(wsHub)
	log.Info("✓ WebSocket hub started for real-time flow streaming")

	// Start collector
	if err := collector.Start(); err != nil {
		log.Errorf("Failed to start flow collector: %v", err)
		return nil
	}

	// Create and start lifecycle manager (only if storage exists)
	if storage != nil {
		lifecycleConfig := flow.LifecycleConfig{
			CleanupInterval:           24 * time.Hour, // Daily cleanup
			RetentionDuration:         time.Duration(cfg.Flow.RetentionDays) * 24 * time.Hour,
			StoragePath:               cfg.Flow.StoragePath,
			DiskSpaceThresholdPercent: 80, // Warn when disk usage exceeds 80%
			EnableDiskMonitoring:      true,
		}

		lifecycleManager := flow.NewLifecycleManager(storage, lifecycleConfig)
		if err := lifecycleManager.Start(); err != nil {
			log.Warnf("Failed to start lifecycle manager: %v", err)
		} else {
			log.Info("✓ Lifecycle manager started for data cleanup and monitoring")
		}

		// Store lifecycle manager in collector for API access
		collector.SetLifecycleManager(lifecycleManager)
	}

	return collector
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

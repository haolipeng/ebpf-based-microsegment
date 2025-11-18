// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/client"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/config"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/conntrack"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/fragment"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/reporter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configPath string
	// version can be set via ldflags during build: -ldflags "-X main.version=v1.0.0"
	version = "0.1.0"
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

	// Initialize NAT support (conntrack synchronization)
	var ctSyncer *conntrack.ConntrackSyncer
	ctSyncer, err = initNATSupport(dp)
	if err != nil {
		log.Warnf("NAT support initialization failed: %v (continuing without NAT support)", err)
	} else if ctSyncer != nil {
		defer ctSyncer.Stop()
		log.Info("✓ NAT support initialized (conntrack sync started)")
	}

	// Initialize fragment cleanup (fragment timeout cleaner)
	var fragCleaner *fragment.FragmentCleaner
	fragCleaner, err = initFragmentSupport(dp)
	if err != nil {
		log.Warnf("Fragment support initialization failed: %v (continuing without fragment cleanup)", err)
	} else if fragCleaner != nil {
		defer fragCleaner.Stop()
		log.Info("✓ Fragment support initialized (fragment cleaner started)")
	}

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
		rep, agentClient = initGrpcControlPlane(cfg, pm)

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
	var flowCollector *flow.Collector
	if cfg.IsAgentServerMode() {
		flowCollector = initFlowCollectorForServerMode(cfg, dp, rep)
	} else {
		flowCollector = initFlowCollectorForStandaloneMode(cfg, dp, flowStorage)
	}

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
			Version:       version,
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

func initGrpcControlPlane(cfg *config.Config, pm *policy.PolicyManager) (reporter.Reporter, *client.AgentClient) {
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
	// Extract directory path from storage file path
	storageDir := filepath.Dir(cfg.Flow.StoragePath)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Errorf("Failed to create storage directory %s: %v", storageDir, err)
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

// initFlowCollectorForServerMode initializes flow collector for agent-server mode.
// In this mode, flows are sent to the control plane server via gRPC reporter.
// No local storage or lifecycle management is configured.
func initFlowCollectorForServerMode(cfg *config.Config, dp *dataplane.DataPlane, rep reporter.Reporter) *flow.Collector {
	// Get Ring Buffer from dataplane
	ringBuf := dp.GetFlowRingBuffer()
	if ringBuf == nil {
		log.Error("Failed to get flow ring buffer from dataplane")
		return nil
	}

	// Create collector configuration (no persistence in server mode)
	collectorConfig := flow.CollectorConfig{
		FlowTimeout:       cfg.Flow.FlowTimeout,
		BatchSize:         100,
		EnableEnrichment:  true,
		EnablePersistence: false, // Server mode: flows are sent to server, not persisted locally
		CleanupInterval:   cfg.Flow.CleanupInterval,
	}

	// Create collector without storage (workloadMgr can be added later for K8s)
	collector := flow.NewCollector(ringBuf, nil, nil, collectorConfig)

	// Configure reporter to send flows to server
	if rep != nil {
		collector.SetReporter(rep)
		log.Info("✓ Collector configured to send flows to server via gRPC")
	}

	// Start collector
	if err := collector.Start(); err != nil {
		log.Errorf("Failed to start flow collector: %v", err)
		return nil
	}

	return collector
}

// initFlowCollectorForStandaloneMode initializes flow collector for standalone mode.
// In this mode, flows are persisted to local SQLite storage with lifecycle management.
func initFlowCollectorForStandaloneMode(cfg *config.Config, dp *dataplane.DataPlane, storage flow.Storage) *flow.Collector {
	// Get Ring Buffer from dataplane
	ringBuf := dp.GetFlowRingBuffer()
	if ringBuf == nil {
		log.Error("Failed to get flow ring buffer from dataplane")
		return nil
	}

	// Create collector configuration (with persistence in standalone mode)
	collectorConfig := flow.CollectorConfig{
		FlowTimeout:       cfg.Flow.FlowTimeout,
		BatchSize:         100,
		EnableEnrichment:  true,
		EnablePersistence: storage != nil, // Persist flows to local storage
		CleanupInterval:   cfg.Flow.CleanupInterval,
	}

	// Create collector with storage (workloadMgr can be added later for K8s)
	collector := flow.NewCollector(ringBuf, storage, nil, collectorConfig)

	// Start collector
	if err := collector.Start(); err != nil {
		log.Errorf("Failed to start flow collector: %v", err)
		return nil
	}

	// Create and start lifecycle manager for storage cleanup
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

// initNATSupport initializes NAT detection and conntrack synchronization
func initNATSupport(dp *dataplane.DataPlane) (*conntrack.ConntrackSyncer, error) {
	log.Info("Initializing NAT support...")

	// Get conntrack cache map from dataplane
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get dataplane maps: %w", err)
	}

	if maps.ConntrackCacheMap == nil {
		return nil, fmt.Errorf("conntrack cache map not available")
	}

	// Create conntrack syncer with default config
	syncer, err := conntrack.NewConntrackSyncer(maps.ConntrackCacheMap, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create conntrack syncer: %w", err)
	}

	// Start conntrack synchronization
	if err := syncer.Start(); err != nil {
		syncer.Stop()
		return nil, fmt.Errorf("failed to start conntrack syncer: %w", err)
	}

	// Configure NAT detection (enable cache lookup by default)
	natConfig := &dataplane.NATConfig{
		MatchMode:       dataplane.NATMatchModeOriginal, // Match using pre-NAT addresses
		EnableCache:     true,                            // Enable conntrack cache lookup
		EnableBPFHelper: false,                           // BPF helper not yet fully implemented
		LogEvents:       false,                           // Disable event logging for performance
	}

	if err := dp.SetNATConfig(natConfig); err != nil {
		log.Warnf("Failed to set NAT config: %v", err)
	} else {
		log.Info("NAT detection configured: match_mode=original, cache=enabled")
	}

	// Log initial sync statistics
	go func() {
		time.Sleep(2 * time.Second) // Wait for initial sync to complete
		stats := syncer.GetStats()
		log.Infof("Conntrack sync statistics: total=%d, last_sync=%v",
			stats.TotalEntries, stats.LastSyncTime)

		// Log NAT statistics
		natStats, err := dp.GetNATStats()
		if err == nil && natStats != nil {
			log.Infof("NAT statistics: lookups=%d, cache_hits=%d, hit_rate=%.2f%%",
				natStats.TotalLookups, natStats.CacheHits, natStats.CacheHitRate*100)
		}
	}()

	return syncer, nil
}

// initFragmentSupport initializes fragment handling and timeout cleanup
func initFragmentSupport(dp *dataplane.DataPlane) (*fragment.FragmentCleaner, error) {
	log.Info("Initializing fragment support...")

	// Get fragment maps from dataplane
	maps, err := dp.GetMaps()
	if err != nil {
		return nil, fmt.Errorf("failed to get dataplane maps: %w", err)
	}

	if maps.FragStateMap == nil {
		return nil, fmt.Errorf("fragment state map not available")
	}

	if maps.FragStatsMap == nil {
		return nil, fmt.Errorf("fragment stats map not available")
	}

	// Set default fragment configuration
	fragConfig := &dataplane.FragmentConfig{
		Mode:      dataplane.FragmentModeNormal, // NORMAL: allow first fragment, drop subsequent
		LogEvents: true,                         // Enable fragment event logging
		TimeoutNs: 30 * 1000000000,              // 30 seconds timeout
	}

	if err := dp.SetFragmentConfig(fragConfig); err != nil {
		return nil, fmt.Errorf("failed to set fragment config: %w", err)
	}

	log.Infof("Fragment configuration set: mode=%s, timeout=30s", fragConfig.Mode.String())

	// Create fragment cleaner with default config
	cleanerConfig := fragment.DefaultFragmentCleanerConfig()
	cleaner := fragment.NewFragmentCleaner(maps.FragStateMap, maps.FragStatsMap, cleanerConfig)

	// Start fragment cleaner
	if err := cleaner.Start(); err != nil {
		return nil, fmt.Errorf("failed to start fragment cleaner: %w", err)
	}

	// Log initial statistics after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		stats, err := dp.GetFragmentStats()
		if err == nil && stats != nil {
			log.Infof("Fragment statistics: first=%d, subsequent=%d, allowed=%d, denied=%d",
				stats.FirstFragments, stats.SubsequentFragments,
				stats.FragmentsAllowed, stats.FragmentsDenied)
		}
	}()

	return cleaner, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

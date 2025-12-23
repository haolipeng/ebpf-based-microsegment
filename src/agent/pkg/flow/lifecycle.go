// input: flow lifecycle events (create, update, close, timeout)
// output: flow state transitions, graceful shutdown handling
// pos: flow lifecycle manager - if file updated, must sync with this header comment and pkg/flow/CLAUDE.md
package flow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// LifecycleManager manages flow data lifecycle including cleanup and archival
type LifecycleManager struct {
	storage Storage
	config  LifecycleConfig

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats      LifecycleStats
	statsMutex sync.RWMutex
}

// LifecycleConfig holds configuration for lifecycle management
type LifecycleConfig struct {
	// CleanupInterval is how often to run cleanup tasks
	CleanupInterval time.Duration

	// RetentionDuration is how long to keep flow data
	RetentionDuration time.Duration

	// StoragePath is the path to the database file
	StoragePath string

	// DiskSpaceThresholdPercent is the disk usage threshold to trigger warnings
	DiskSpaceThresholdPercent int

	// EnableDiskMonitoring enables disk space monitoring
	EnableDiskMonitoring bool
}

// LifecycleStats tracks lifecycle management statistics
type LifecycleStats struct {
	// Total cleanup runs
	TotalCleanupRuns int64 `json:"total_cleanup_runs"`

	// Total flows deleted
	TotalFlowsDeleted int64 `json:"total_flows_deleted"`

	// Last cleanup time
	LastCleanupTime time.Time `json:"last_cleanup_time"`

	// Flows deleted in last cleanup
	LastCleanupDeleted int64 `json:"last_cleanup_deleted"`

	// Last cleanup duration
	LastCleanupDuration time.Duration `json:"last_cleanup_duration_ms"`

	// Cleanup errors
	CleanupErrors int64 `json:"cleanup_errors"`

	// Disk space info
	DiskTotalBytes uint64  `json:"disk_total_bytes"`
	DiskUsedBytes  uint64  `json:"disk_used_bytes"`
	DiskFreeBytes  uint64  `json:"disk_free_bytes"`
	DiskUsagePercent float64 `json:"disk_usage_percent"`
	LastDiskCheck  time.Time `json:"last_disk_check"`
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(storage Storage, config LifecycleConfig) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &LifecycleManager{
		storage: storage,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		stats:   LifecycleStats{},
	}
}

// Start begins the lifecycle management loops
func (lm *LifecycleManager) Start() error {
	log.Info("[Lifecycle Manager] Starting flow data lifecycle management...")

	// Start cleanup loop
	lm.wg.Add(1)
	go lm.cleanupLoop()

	// Start disk monitoring if enabled
	if lm.config.EnableDiskMonitoring {
		lm.wg.Add(1)
		go lm.diskMonitoringLoop()
	}

	log.Info("[Lifecycle Manager] Lifecycle management started")
	return nil
}

// Stop gracefully stops the lifecycle manager
func (lm *LifecycleManager) Stop() error {
	log.Info("[Lifecycle Manager] Stopping lifecycle management...")
	lm.cancel()
	lm.wg.Wait()
	log.Info("[Lifecycle Manager] Lifecycle management stopped")
	return nil
}

// cleanupLoop periodically runs cleanup tasks
func (lm *LifecycleManager) cleanupLoop() {
	defer lm.wg.Done()

	// Run initial cleanup after a short delay
	time.Sleep(30 * time.Second)

	ticker := time.NewTicker(lm.config.CleanupInterval)
	defer ticker.Stop()

	log.Infof("[Lifecycle Manager] Cleanup loop started (interval: %v, retention: %v)",
		lm.config.CleanupInterval, lm.config.RetentionDuration)

	for {
		select {
		case <-lm.ctx.Done():
			log.Info("[Lifecycle Manager] Cleanup loop stopped")
			return

		case <-ticker.C:
			if err := lm.runCleanup(); err != nil {
				log.Errorf("[Lifecycle Manager] Cleanup failed: %v", err)
				lm.statsMutex.Lock()
				lm.stats.CleanupErrors++
				lm.statsMutex.Unlock()
			}
		}
	}
}

// runCleanup executes the cleanup task
func (lm *LifecycleManager) runCleanup() error {
	startTime := time.Now()

	log.Debug("[Lifecycle Manager] Starting cleanup task...")

	// Delete old flows
	deleted, err := lm.storage.DeleteOldFlows(lm.config.RetentionDuration)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)

	// Update statistics
	lm.statsMutex.Lock()
	lm.stats.TotalCleanupRuns++
	lm.stats.TotalFlowsDeleted += deleted
	lm.stats.LastCleanupTime = time.Now()
	lm.stats.LastCleanupDeleted = deleted
	lm.stats.LastCleanupDuration = duration
	lm.statsMutex.Unlock()

	if deleted > 0 {
		log.Infof("[Lifecycle Manager] Cleanup completed: deleted %d flows (retention: %v, duration: %v)",
			deleted, lm.config.RetentionDuration, duration)
	} else {
		log.Debug("[Lifecycle Manager] Cleanup completed: no flows to delete")
	}

	return nil
}

// diskMonitoringLoop monitors disk space usage
func (lm *LifecycleManager) diskMonitoringLoop() {
	defer lm.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	log.Info("[Lifecycle Manager] Disk monitoring started")

	// Run initial check
	lm.checkDiskSpace()

	for {
		select {
		case <-lm.ctx.Done():
			log.Info("[Lifecycle Manager] Disk monitoring stopped")
			return

		case <-ticker.C:
			lm.checkDiskSpace()
		}
	}
}

// checkDiskSpace checks available disk space
func (lm *LifecycleManager) checkDiskSpace() {
	// Get absolute path to storage
	absPath, err := filepath.Abs(lm.config.StoragePath)
	if err != nil {
		log.Warnf("[Lifecycle Manager] Failed to get absolute path: %v", err)
		return
	}

	// Get file info
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		// Database might not exist yet
		log.Debugf("[Lifecycle Manager] Database file not found: %v", err)
		return
	}

	// Get filesystem stats
	var stat = getDiskUsage(filepath.Dir(absPath))

	// Calculate usage percentage
	usagePercent := float64(stat.Used) / float64(stat.Total) * 100

	// Update statistics
	lm.statsMutex.Lock()
	lm.stats.DiskTotalBytes = stat.Total
	lm.stats.DiskUsedBytes = stat.Used
	lm.stats.DiskFreeBytes = stat.Free
	lm.stats.DiskUsagePercent = usagePercent
	lm.stats.LastDiskCheck = time.Now()
	lm.statsMutex.Unlock()

	// Check threshold
	if usagePercent >= float64(lm.config.DiskSpaceThresholdPercent) {
		log.Warnf("[Lifecycle Manager] Disk usage is high: %.1f%% (threshold: %d%%, database size: %s)",
			usagePercent, lm.config.DiskSpaceThresholdPercent, formatBytes(uint64(fileInfo.Size())))
	} else {
		log.Debugf("[Lifecycle Manager] Disk usage: %.1f%% (database size: %s)",
			usagePercent, formatBytes(uint64(fileInfo.Size())))
	}
}

// GetStats returns current lifecycle statistics
func (lm *LifecycleManager) GetStats() LifecycleStats {
	lm.statsMutex.RLock()
	defer lm.statsMutex.RUnlock()
	return lm.stats
}

// DiskUsage holds disk usage statistics
type DiskUsage struct {
	Total uint64
	Free  uint64
	Used  uint64
}

// getDiskUsage returns disk usage for a given path
func getDiskUsage(path string) DiskUsage {
	// This is a simplified implementation
	// For production, use syscall.Statfs on Linux or similar platform-specific APIs

	// Try to get filesystem info using os.Stat
	// Note: This doesn't give us total/free space, just file size
	// For a complete implementation, we'd need to use platform-specific syscalls

	// Placeholder: return dummy values for now
	// In production, implement proper filesystem stats using syscall.Statfs
	return DiskUsage{
		Total: 100 * 1024 * 1024 * 1024, // 100 GB
		Free:  50 * 1024 * 1024 * 1024,  // 50 GB
		Used:  50 * 1024 * 1024 * 1024,  // 50 GB
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp+1])
}

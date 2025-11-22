package process

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/cilium/ebpf/ringbuf"
)

const (
	// Default cache capacity: 20000 entries
	DefaultCacheCapacity = 20000

	// Default cache TTL: 5 minutes
	DefaultCacheTTL = 5 * time.Minute

	// Default cleanup interval: 30 seconds
	DefaultCleanupInterval = 30 * time.Second
)

// ProcessMonitor monitors process events from eBPF and maintains a process cache
type ProcessMonitor struct {
	ringBuf *ringbuf.Reader
	cache   *ProcessCache

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	eventsProcessed uint64
	eventsDropped   uint64
	pathErrors      uint64
	metricsMutex    sync.RWMutex

	// Configuration
	config MonitorConfig
}

// MonitorConfig holds configuration for the process monitor
type MonitorConfig struct {
	// CacheCapacity is the maximum number of entries in the cache
	CacheCapacity int

	// CacheTTL is the time-to-live for cache entries
	CacheTTL time.Duration

	// CleanupInterval is the interval for cleaning up expired entries
	CleanupInterval time.Duration

	// EnablePathResolution enables querying /proc/<pid>/exe for process path
	EnablePathResolution bool
}

// DefaultMonitorConfig returns default monitor configuration
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		CacheCapacity:        DefaultCacheCapacity,
		CacheTTL:             DefaultCacheTTL,
		CleanupInterval:      DefaultCleanupInterval,
		EnablePathResolution: true,
	}
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor(ringBuf *ringbuf.Reader, config MonitorConfig) *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	cache := NewProcessCache(config.CacheCapacity, config.CacheTTL)

	return &ProcessMonitor{
		ringBuf: ringBuf,
		cache:   cache,
		ctx:     ctx,
		cancel:  cancel,
		config:  config,
	}
}

// Start begins monitoring process events
func (m *ProcessMonitor) Start() error {
	log.Println("[Process Monitor] Starting process monitor...")

	// Start event collection loop
	m.wg.Add(1)
	go m.collectLoop()

	// Start cleanup loop for expired entries
	m.wg.Add(1)
	go m.cleanupLoop()

	log.Printf("[Process Monitor] Process monitor started (capacity=%d, TTL=%v)",
		m.config.CacheCapacity, m.config.CacheTTL)
	return nil
}

// Stop gracefully stops the process monitor
func (m *ProcessMonitor) Stop() error {
	log.Println("[Process Monitor] Stopping process monitor...")
	m.cancel()
	m.wg.Wait()

	if m.ringBuf != nil {
		if err := m.ringBuf.Close(); err != nil {
			log.Printf("[Process Monitor] Error closing ring buffer: %v", err)
		}
	}

	log.Println("[Process Monitor] Process monitor stopped successfully")
	return nil
}

// GetProcessInfo retrieves process information by PID
// This is the primary API used by FlowCollector
// Returns interface{} to avoid circular dependency with flow package
// The returned value is actually *ProcessInfo
func (m *ProcessMonitor) GetProcessInfo(pid uint32) (interface{}, bool) {
	info := m.cache.Get(pid)
	if info != nil {
		return info, true
	}

	// If not in cache, try to resolve from /proc (for processes that started before agent)
	if m.config.EnablePathResolution {
		info = m.resolveProcessInfo(pid)
		if info != nil && info.IsValid() {
			m.cache.Set(info)
			return info, true
		}
	}

	return nil, false
}

// GetCacheStats returns cache statistics
func (m *ProcessMonitor) GetCacheStats() CacheStats {
	return m.cache.GetStats()
}

// GetMetrics returns monitor metrics
func (m *ProcessMonitor) GetMetrics() MonitorMetrics {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()

	return MonitorMetrics{
		EventsProcessed: m.eventsProcessed,
		EventsDropped:   m.eventsDropped,
		PathErrors:      m.pathErrors,
		CacheStats:      m.cache.GetStats(),
	}
}

// collectLoop continuously reads process events from Ring Buffer
func (m *ProcessMonitor) collectLoop() {
	defer m.wg.Done()

	log.Println("[Process Monitor] Starting event collection loop...")

	for {
		select {
		case <-m.ctx.Done():
			log.Println("[Process Monitor] Collection loop stopped")
			return
		default:
			// Read event from ring buffer (blocking with timeout)
			record, err := m.ringBuf.Read()
			if err != nil {
				if err == ringbuf.ErrClosed {
					log.Println("[Process Monitor] Ring buffer closed")
					return
				}
				// Log error and continue
				m.incrementDropped()
				continue
			}

			// Parse process event
			event, err := m.parseProcessEvent(record.RawSample)
			if err != nil {
				log.Printf("[Process Monitor] Error parsing process event: %v", err)
				m.incrementDropped()
				continue
			}

			// Process event
			if err := m.processEvent(event); err != nil {
				log.Printf("[Process Monitor] Error processing process event: %v", err)
				m.incrementDropped()
				continue
			}

			m.incrementProcessed()
		}
	}
}

// cleanupLoop periodically removes expired cache entries
func (m *ProcessMonitor) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	log.Printf("[Process Monitor] Starting cleanup loop (interval=%v)", m.config.CleanupInterval)

	for {
		select {
		case <-m.ctx.Done():
			log.Println("[Process Monitor] Cleanup loop stopped")
			return
		case <-ticker.C:
			removed := m.cache.CleanExpired()
			if removed > 0 {
				log.Printf("[Process Monitor] Cleaned up %d expired cache entries", removed)
			}
		}
	}
}

// parseProcessEvent parses raw bytes into ProcessEvent structure
func (m *ProcessMonitor) parseProcessEvent(rawData []byte) (*ProcessEvent, error) {
	// Verify size matches ProcessEvent structure
	expectedSize := int(unsafe.Sizeof(ProcessEvent{}))
	if len(rawData) < expectedSize {
		return nil, fmt.Errorf("invalid event size: got %d, expected %d", len(rawData), expectedSize)
	}

	event := &ProcessEvent{}
	reader := bytes.NewReader(rawData)

	// Parse fields in order matching kernel structure
	if err := binary.Read(reader, binary.LittleEndian, &event.PID); err != nil {
		return nil, fmt.Errorf("failed to read PID: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &event.Comm); err != nil {
		return nil, fmt.Errorf("failed to read Comm: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &event.ExecTime); err != nil {
		return nil, fmt.Errorf("failed to read ExecTime: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &event.ContainerID); err != nil {
		return nil, fmt.Errorf("failed to read ContainerID: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &event.Flags); err != nil {
		return nil, fmt.Errorf("failed to read Flags: %w", err)
	}

	return event, nil
}

// processEvent processes a single process event
func (m *ProcessMonitor) processEvent(event *ProcessEvent) error {
	// Convert to ProcessInfo
	info := event.ToProcessInfo()

	// Resolve process path if enabled
	if m.config.EnablePathResolution {
		path, err := m.resolvePath(info.PID)
		if err != nil {
			m.incrementPathErrors()
			// Don't fail the entire event processing, just log the error
			log.Printf("[Process Monitor] Failed to resolve path for PID %d: %v", info.PID, err)
		} else {
			info.Path = path
			info.PathResolved = true
		}
	}

	// Add to cache
	m.cache.Set(info)

	log.Printf("[Process Monitor] Cached process: PID=%d, Comm=%s, Path=%s, Container=%s",
		info.PID, info.Comm, info.Path, info.ContainerID)

	return nil
}

// resolvePath queries /proc/<pid>/exe to get the full executable path
func (m *ProcessMonitor) resolvePath(pid uint32) (string, error) {
	procPath := fmt.Sprintf("/proc/%d/exe", pid)
	path, err := os.Readlink(procPath)
	if err != nil {
		return "", fmt.Errorf("readlink failed: %w", err)
	}
	return path, nil
}

// resolveProcessInfo attempts to resolve process info from /proc for existing processes
func (m *ProcessMonitor) resolveProcessInfo(pid uint32) *ProcessInfo {
	// Read comm from /proc/<pid>/comm
	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	commData, err := os.ReadFile(commPath)
	if err != nil {
		return nil
	}

	comm := string(bytes.TrimSpace(commData))

	// Read exe path
	path := ""
	pathResolved := false
	if m.config.EnablePathResolution {
		if exePath, err := m.resolvePath(pid); err == nil {
			path = exePath
			pathResolved = true
		}
	}

	// Note: We cannot determine exact exec_time from /proc, so we use current time
	// This means PID reuse detection won't work for processes started before agent
	return &ProcessInfo{
		PID:          pid,
		Comm:         comm,
		Path:         path,
		ContainerID:  "", // Cannot determine from /proc alone
		ExecTime:     0,  // Unknown for existing processes
		CachedTime:   time.Now(),
		PathResolved: pathResolved,
	}
}

// incrementProcessed increments the processed events counter
func (m *ProcessMonitor) incrementProcessed() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.eventsProcessed++
}

// incrementDropped increments the dropped events counter
func (m *ProcessMonitor) incrementDropped() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.eventsDropped++
}

// incrementPathErrors increments the path resolution errors counter
func (m *ProcessMonitor) incrementPathErrors() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.pathErrors++
}

// MonitorMetrics represents process monitor metrics
type MonitorMetrics struct {
	EventsProcessed uint64     `json:"events_processed"`
	EventsDropped   uint64     `json:"events_dropped"`
	PathErrors      uint64     `json:"path_errors"`
	CacheStats      CacheStats `json:"cache_stats"`
}

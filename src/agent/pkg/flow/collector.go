package flow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cilium/ebpf/ringbuf"
)

// WorkloadManager interface for workload label lookup
type WorkloadManager interface {
	// GetLabelsByIP returns labels for a workload identified by IP address
	GetLabelsByIP(ip string) (map[string]string, bool)
}

// Storage interface for persisting flow data
type Storage interface {
	// SaveFlow persists a flow to storage
	SaveFlow(flow *Flow) error

	// UpdateFlow updates an existing flow
	UpdateFlow(flow *Flow) error

	// GetFlow retrieves a flow by ID
	GetFlow(id string) (*Flow, error)

	// QueryFlows queries flows based on filters
	QueryFlows(query *FlowQuery) ([]*Flow, error)

	// GetFlowSummary returns aggregated flow statistics
	GetFlowSummary(startTime, endTime time.Time) (*FlowSummary, error)

	// DeleteOldFlows deletes flows older than the specified duration
	DeleteOldFlows(olderThan time.Duration) (int64, error)

	// Close closes the storage backend
	Close() error
}

// Collector collects flow events from eBPF Ring Buffer
type Collector struct {
	ringBuf     *ringbuf.Reader
	storage     Storage
	workloadMgr WorkloadManager

	// Flow tracking
	activeFlows map[string]*Flow // Keyed by flow ID
	flowsMutex  sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	eventsProcessed uint64
	eventsDropped   uint64
	metricsMutex    sync.RWMutex

	// Configuration
	config CollectorConfig
}

// CollectorConfig holds configuration for the flow collector
type CollectorConfig struct {
	// FlowTimeout is the duration after which inactive flows are considered closed
	FlowTimeout time.Duration

	// BatchSize is the number of flows to batch before persisting
	BatchSize int

	// EnableEnrichment enables workload label enrichment
	EnableEnrichment bool

	// EnablePersistence enables flow persistence to storage
	EnablePersistence bool

	// CleanupInterval is the interval for cleaning up inactive flows
	CleanupInterval time.Duration
}

// DefaultCollectorConfig returns default collector configuration
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		FlowTimeout:       5 * time.Minute,
		BatchSize:         100,
		EnableEnrichment:  true,
		EnablePersistence: true,
		CleanupInterval:   1 * time.Minute,
	}
}

// NewCollector creates a new flow collector
func NewCollector(ringBuf *ringbuf.Reader, storage Storage, workloadMgr WorkloadManager, config CollectorConfig) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		ringBuf:     ringBuf,
		storage:     storage,
		workloadMgr: workloadMgr,
		activeFlows: make(map[string]*Flow),
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
	}
}

// Start begins collecting flow events from Ring Buffer
func (c *Collector) Start() error {
	log.Println("[Flow Collector] Starting flow collector...")

	// Start event collection loop
	c.wg.Add(1)
	go c.collectLoop()

	// Start cleanup loop for inactive flows
	c.wg.Add(1)
	go c.cleanupLoop()

	log.Println("[Flow Collector] Flow collector started successfully")
	return nil
}

// Stop gracefully stops the flow collector
func (c *Collector) Stop() error {
	log.Println("[Flow Collector] Stopping flow collector...")
	c.cancel()
	c.wg.Wait()

	// Flush remaining active flows
	if err := c.flushActiveFlows(); err != nil {
		log.Printf("[Flow Collector] Error flushing active flows: %v", err)
	}

	if c.ringBuf != nil {
		if err := c.ringBuf.Close(); err != nil {
			log.Printf("[Flow Collector] Error closing ring buffer: %v", err)
		}
	}

	log.Println("[Flow Collector] Flow collector stopped successfully")
	return nil
}

// collectLoop continuously reads flow events from Ring Buffer
func (c *Collector) collectLoop() {
	defer c.wg.Done()

	log.Println("[Flow Collector] Starting event collection loop...")

	for {
		select {
		case <-c.ctx.Done():
			log.Println("[Flow Collector] Collection loop stopped")
			return
		default:
			// Read event from ring buffer (blocking with timeout)
			record, err := c.ringBuf.Read()
			if err != nil {
				if err == ringbuf.ErrClosed {
					log.Println("[Flow Collector] Ring buffer closed")
					return
				}
				// Log error and continue
				c.incrementDropped()
				continue
			}

			// Parse flow event
			event, err := ParseFlowEvent(record.RawSample)
			if err != nil {
				log.Printf("[Flow Collector] Error parsing flow event: %v", err)
				c.incrementDropped()
				continue
			}

			// Process flow event
			if err := c.processFlowEvent(event); err != nil {
				log.Printf("[Flow Collector] Error processing flow event: %v", err)
				c.incrementDropped()
				continue
			}

			c.incrementProcessed()
		}
	}
}

// processFlowEvent processes a single flow event
func (c *Collector) processFlowEvent(event *FlowEvent) error {
	// Convert event to flow
	flow := event.ToFlow()

	// Enrich with workload labels if enabled
	if c.config.EnableEnrichment && c.workloadMgr != nil {
		c.enrichWithLabels(flow)
	}

	// Handle different event types
	switch event.EventType {
	case FlowEventNew:
		return c.handleNewFlow(flow)
	case FlowEventUpdate:
		return c.handleUpdateFlow(flow)
	case FlowEventClosed, FlowEventTimeout:
		return c.handleClosedFlow(flow)
	default:
		return fmt.Errorf("unknown event type: %v", event.EventType)
	}
}

// handleNewFlow handles a new flow event
func (c *Collector) handleNewFlow(flow *Flow) error {
	c.flowsMutex.Lock()
	defer c.flowsMutex.Unlock()

	// Check if flow already exists (duplicate event)
	if _, exists := c.activeFlows[flow.ID]; exists {
		// Update existing flow instead
		return c.updateExistingFlow(flow)
	}

	// Add to active flows
	c.activeFlows[flow.ID] = flow

	// Persist to storage if enabled
	if c.config.EnablePersistence && c.storage != nil {
		if err := c.storage.SaveFlow(flow); err != nil {
			return fmt.Errorf("failed to save flow: %w", err)
		}
	}

	return nil
}

// handleUpdateFlow handles a flow update event
func (c *Collector) handleUpdateFlow(flow *Flow) error {
	c.flowsMutex.Lock()
	defer c.flowsMutex.Unlock()

	return c.updateExistingFlow(flow)
}

// handleClosedFlow handles a flow closed/timeout event
func (c *Collector) handleClosedFlow(flow *Flow) error {
	c.flowsMutex.Lock()
	defer c.flowsMutex.Unlock()

	// Update existing flow with final statistics
	if err := c.updateExistingFlow(flow); err != nil {
		return err
	}

	// Remove from active flows
	delete(c.activeFlows, flow.ID)

	return nil
}

// updateExistingFlow updates an existing flow (must be called with flowsMutex held)
func (c *Collector) updateExistingFlow(flow *Flow) error {
	existing, exists := c.activeFlows[flow.ID]
	if !exists {
		// Flow not found, treat as new flow
		c.activeFlows[flow.ID] = flow
		if c.config.EnablePersistence && c.storage != nil {
			return c.storage.SaveFlow(flow)
		}
		return nil
	}

	// Update statistics
	existing.PacketCount = flow.PacketCount
	existing.ByteCount = flow.ByteCount
	existing.LastSeen = flow.LastSeen
	existing.State = flow.State
	existing.EventType = flow.EventType

	// Update end time if flow is closed
	if flow.EndTime != nil {
		existing.EndTime = flow.EndTime
		existing.Duration = flow.EndTime.Sub(existing.StartTime).Milliseconds()
	}

	// Persist update to storage if enabled
	if c.config.EnablePersistence && c.storage != nil {
		if err := c.storage.UpdateFlow(existing); err != nil {
			return fmt.Errorf("failed to update flow: %w", err)
		}
	}

	return nil
}

// enrichWithLabels enriches flow with workload labels
func (c *Collector) enrichWithLabels(flow *Flow) {
	// Enrich source labels
	if labels, found := c.workloadMgr.GetLabelsByIP(flow.SourceIP); found {
		flow.SourceLabels = labels
	}

	// Enrich destination labels
	if labels, found := c.workloadMgr.GetLabelsByIP(flow.DestIP); found {
		flow.DestLabels = labels
	}
}

// cleanupLoop periodically cleans up inactive flows
func (c *Collector) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	log.Printf("[Flow Collector] Starting cleanup loop (interval: %v, timeout: %v)",
		c.config.CleanupInterval, c.config.FlowTimeout)

	for {
		select {
		case <-c.ctx.Done():
			log.Println("[Flow Collector] Cleanup loop stopped")
			return
		case <-ticker.C:
			c.cleanupInactiveFlows()
		}
	}
}

// cleanupInactiveFlows removes flows that have been inactive for longer than FlowTimeout
func (c *Collector) cleanupInactiveFlows() {
	c.flowsMutex.Lock()
	defer c.flowsMutex.Unlock()

	now := time.Now()
	timeoutThreshold := now.Add(-c.config.FlowTimeout)

	var closedCount int
	for id, flow := range c.activeFlows {
		if flow.LastSeen.Before(timeoutThreshold) {
			// Mark as timeout and persist
			flow.State = "TIMEOUT"
			endTime := flow.LastSeen.Add(c.config.FlowTimeout)
			flow.EndTime = &endTime
			flow.Duration = flow.EndTime.Sub(flow.StartTime).Milliseconds()

			if c.config.EnablePersistence && c.storage != nil {
				if err := c.storage.UpdateFlow(flow); err != nil {
					log.Printf("[Flow Collector] Error updating timeout flow %s: %v", id, err)
				}
			}

			delete(c.activeFlows, id)
			closedCount++
		}
	}

	if closedCount > 0 {
		log.Printf("[Flow Collector] Cleaned up %d inactive flows", closedCount)
	}
}

// flushActiveFlows persists all active flows to storage
func (c *Collector) flushActiveFlows() error {
	c.flowsMutex.Lock()
	defer c.flowsMutex.Unlock()

	log.Printf("[Flow Collector] Flushing %d active flows...", len(c.activeFlows))

	for _, flow := range c.activeFlows {
		if c.config.EnablePersistence && c.storage != nil {
			if err := c.storage.UpdateFlow(flow); err != nil {
				log.Printf("[Flow Collector] Error flushing flow %s: %v", flow.ID, err)
			}
		}
	}

	return nil
}

// GetActiveFlows returns a snapshot of currently active flows
func (c *Collector) GetActiveFlows() []*Flow {
	c.flowsMutex.RLock()
	defer c.flowsMutex.RUnlock()

	flows := make([]*Flow, 0, len(c.activeFlows))
	for _, flow := range c.activeFlows {
		// Return a copy to avoid race conditions
		flowCopy := *flow
		flows = append(flows, &flowCopy)
	}

	return flows
}

// GetMetrics returns collector metrics
func (c *Collector) GetMetrics() (processed, dropped uint64, activeFlows int) {
	c.metricsMutex.RLock()
	defer c.metricsMutex.RUnlock()

	c.flowsMutex.RLock()
	activeFlows = len(c.activeFlows)
	c.flowsMutex.RUnlock()

	return c.eventsProcessed, c.eventsDropped, activeFlows
}

// incrementProcessed increments the processed events counter
func (c *Collector) incrementProcessed() {
	c.metricsMutex.Lock()
	c.eventsProcessed++
	c.metricsMutex.Unlock()
}

// incrementDropped increments the dropped events counter
func (c *Collector) incrementDropped() {
	c.metricsMutex.Lock()
	c.eventsDropped++
	c.metricsMutex.Unlock()
}

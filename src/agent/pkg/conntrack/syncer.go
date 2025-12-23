// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: netlink conntrack events
// output: eBPF conntrack_cache_map updates
// pos: conntrack state synchronizer - if file updated, must sync with this header comment and pkg/conntrack/CLAUDE.md
package conntrack

import (
	"fmt"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	ct "github.com/florianl/go-conntrack"
	log "github.com/sirupsen/logrus"
)

// ConntrackSyncer synchronizes kernel conntrack entries to eBPF map
type ConntrackSyncer struct {
	nfct     *ct.Nfct
	cacheMap *ebpf.Map
	config   *SyncConfig
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Statistics
	mu    sync.RWMutex
	stats SyncStats
}

// NewConntrackSyncer creates a new conntrack syncer
func NewConntrackSyncer(cacheMap *ebpf.Map, config *SyncConfig) (*ConntrackSyncer, error) {
	if cacheMap == nil {
		return nil, fmt.Errorf("cache map is nil")
	}

	if config == nil {
		config = DefaultSyncConfig()
	}

	// Open netlink conntrack connection
	nfct, err := ct.Open(&ct.Config{
		// Listen to conntrack events in current network namespace
		NetNS: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open conntrack: %w", err)
	}

	syncer := &ConntrackSyncer{
		nfct:     nfct,
		cacheMap: cacheMap,
		config:   config,
		stopCh:   make(chan struct{}),
	}

	return syncer, nil
}

// Start starts the conntrack synchronization
func (s *ConntrackSyncer) Start() error {
	log.Info("Starting conntrack syncer")

	// Perform initial full sync
	if s.config.EnableFullSync {
		if err := s.fullSync(); err != nil {
			log.Warnf("Initial conntrack sync failed: %v", err)
		}
	}

	// Start event-based sync
	// Note: Event-based sync is currently disabled due to API changes in go-conntrack
	// Will be re-enabled in future version with proper context support
	if s.config.EnableEventSync {
		log.Warn("Event-based conntrack sync is currently disabled, using periodic sync only")
		// s.wg.Add(1)
		// go s.eventLoop()
	}

	// Start periodic full sync
	if s.config.EnableFullSync && s.config.SyncInterval > 0 {
		s.wg.Add(1)
		go s.periodicSync()
	}

	log.Info("Conntrack syncer started")
	return nil
}

// Stop stops the conntrack synchronization
func (s *ConntrackSyncer) Stop() error {
	log.Info("Stopping conntrack syncer")

	close(s.stopCh)
	s.wg.Wait()

	if s.nfct != nil {
		if err := s.nfct.Close(); err != nil {
			log.Warnf("Failed to close conntrack: %v", err)
		}
	}

	log.Info("Conntrack syncer stopped")
	return nil
}

// fullSync performs a full synchronization of conntrack table
func (s *ConntrackSyncer) fullSync() error {
	startTime := time.Now()
	log.Debug("Starting full conntrack sync")

	// Dump IPv4 conntrack entries
	sessions4, err := s.nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		log.Warnf("Failed to dump IPv4 conntrack: %v", err)
		s.incrementError()
	}

	// Dump IPv6 conntrack entries
	sessions6, err := s.nfct.Dump(ct.Conntrack, ct.IPv6)
	if err != nil {
		log.Warnf("Failed to dump IPv6 conntrack: %v", err)
		s.incrementError()
	}

	// Combine all sessions
	sessions := append(sessions4, sessions6...)

	synced := 0
	skipped := 0
	errors := 0

	for _, session := range sessions {
		// Check if should sync this entry
		if !ShouldSync(session, s.config) {
			skipped++
			continue
		}

		// Sync entry with retries
		if err := s.syncEntryWithRetry(session); err != nil {
			errors++
			log.Debugf("Failed to sync conntrack entry: %v", err)
			continue
		}
		synced++
	}

	// Update statistics
	s.mu.Lock()
	s.stats.TotalEntries = uint64(len(sessions))
	s.stats.LastSyncTime = time.Now()
	s.mu.Unlock()

	duration := time.Since(startTime)
	log.Infof("Conntrack full sync completed: %d synced, %d skipped, %d errors in %v",
		synced, skipped, errors, duration)

	return nil
}

// eventLoop listens for conntrack events and syncs them
// Note: Currently disabled due to API changes in go-conntrack library
// Will be re-implemented in future version using context-based Register API
func (s *ConntrackSyncer) eventLoop() {
	defer s.wg.Done()

	log.Warn("Event-based conntrack sync is not yet implemented")
	log.Info("Using periodic full sync as fallback")

	// TODO: Implement using new Register API:
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// err := s.nfct.Register(ctx, ct.Conntrack, ct.NetlinkCtNew, func(c ct.Con) int {
	//     s.handleEvent(c)
	//     return 0
	// })

	<-s.stopCh
	log.Debug("Conntrack event loop stopped")
}

// periodicSync performs periodic full synchronization
func (s *ConntrackSyncer) periodicSync() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	log.Debugf("Starting periodic conntrack sync (interval: %v)", s.config.SyncInterval)

	for {
		select {
		case <-s.stopCh:
			log.Debug("Periodic conntrack sync stopped")
			return

		case <-ticker.C:
			if err := s.fullSync(); err != nil {
				log.Warnf("Periodic conntrack sync failed: %v", err)
			}
		}
	}
}

// handleEvent handles a single conntrack event
// Note: Currently simplified - event type handling will be added when
// event-based sync is re-enabled
func (s *ConntrackSyncer) handleEvent(con ct.Con) error {
	// Check if should sync this entry
	if !ShouldSync(con, s.config) {
		return nil
	}

	// For now, just sync the entry (we don't know if it's NEW/UPDATE/DESTROY)
	s.mu.Lock()
	s.stats.UpdatedEntries++
	s.mu.Unlock()

	return s.syncEntryWithRetry(con)
}

// syncEntryWithRetry syncs a conntrack entry to eBPF map with retries
func (s *ConntrackSyncer) syncEntryWithRetry(con ct.Con) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.config.RetryDelay)
			log.Debugf("Retrying conntrack sync (attempt %d/%d)", attempt, s.config.MaxRetries)
		}

		if err := s.syncEntry(con); err != nil {
			lastErr = err
			continue
		}

		// Success
		return nil
	}

	// All retries failed
	s.incrementError()
	return lastErr
}

// deleteEntryWithRetry deletes a conntrack entry from eBPF map with retries
func (s *ConntrackSyncer) deleteEntryWithRetry(con ct.Con) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.config.RetryDelay)
			log.Debugf("Retrying conntrack delete (attempt %d/%d)", attempt, s.config.MaxRetries)
		}

		if err := s.deleteEntry(con); err != nil {
			lastErr = err
			continue
		}

		// Success
		return nil
	}

	// All retries failed
	s.incrementError()
	return lastErr
}

// syncEntry synchronizes a single conntrack entry to eBPF map
func (s *ConntrackSyncer) syncEntry(con ct.Con) error {
	// Convert conntrack entry to eBPF map format
	key, value, err := ConvertToMapEntry(con)
	if err != nil {
		s.mu.Lock()
		s.stats.ConvertErrors++
		s.mu.Unlock()
		return fmt.Errorf("failed to convert conntrack entry: %w", err)
	}

	// Update eBPF map
	if err := s.cacheMap.Update(key, value, ebpf.UpdateAny); err != nil {
		return fmt.Errorf("failed to update conntrack cache: %w", err)
	}

	log.Tracef("Synced conntrack entry: %s", FormatConntrackEntry(con))
	return nil
}

// deleteEntry deletes a conntrack entry from eBPF map
func (s *ConntrackSyncer) deleteEntry(con ct.Con) error {
	// Convert conntrack entry to get the key
	key, _, err := ConvertToMapEntry(con)
	if err != nil {
		s.mu.Lock()
		s.stats.ConvertErrors++
		s.mu.Unlock()
		return fmt.Errorf("failed to convert conntrack entry: %w", err)
	}

	// Delete from eBPF map
	if err := s.cacheMap.Delete(key); err != nil {
		// Ignore "key not exist" errors
		if err == ebpf.ErrKeyNotExist {
			return nil
		}
		return fmt.Errorf("failed to delete conntrack entry: %w", err)
	}

	log.Tracef("Deleted conntrack entry: %s", FormatConntrackEntry(con))
	return nil
}

// GetStats returns current synchronization statistics
func (s *ConntrackSyncer) GetStats() SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// ResetStats resets synchronization statistics
func (s *ConntrackSyncer) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = SyncStats{
		LastSyncTime: s.stats.LastSyncTime, // Keep last sync time
	}
}

// incrementError increments error counter (thread-safe)
func (s *ConntrackSyncer) incrementError() {
	s.mu.Lock()
	s.stats.Errors++
	s.mu.Unlock()
}

// IsRunning returns true if the syncer is running
func (s *ConntrackSyncer) IsRunning() bool {
	select {
	case <-s.stopCh:
		return false
	default:
		return true
	}
}

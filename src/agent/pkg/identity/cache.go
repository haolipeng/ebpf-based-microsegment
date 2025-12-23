// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: labels, IP addresses
// output: numeric identity mapping
// pos: identity cache (label to ID mapping) - if file updated, must sync with this header comment and pkg/identity/CLAUDE.md
package identity

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// CacheConfig holds configuration for the identity cache
type CacheConfig struct {
	// MaxSize is the maximum number of identities to cache
	MaxSize int

	// TTL is the time-to-live for cached entries (0 = no expiration)
	TTL time.Duration

	// CleanupInterval is the interval for running cache cleanup
	CleanupInterval time.Duration
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize:         10000,
		TTL:             0, // No expiration by default
		CleanupInterval: 5 * time.Minute,
	}
}

// cacheEntry represents a cached identity with metadata
type cacheEntry struct {
	identity   *Identity
	lastAccess time.Time
}

// Cache provides a thread-safe local cache for identities
// It maintains two indexes for fast lookup:
//   - byID: NumericIdentity -> Identity
//   - byHash: LabelHash -> Identity
type Cache struct {
	mu      sync.RWMutex
	byID    map[NumericIdentity]*cacheEntry
	byHash  map[string]*cacheEntry
	config  CacheConfig
	stopCh  chan struct{}
	started bool

	// Listeners for cache events
	listeners []func(event IdentityAllocatorEvent)
}

// NewCache creates a new identity cache with the given configuration
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		byID:      make(map[NumericIdentity]*cacheEntry),
		byHash:    make(map[string]*cacheEntry),
		config:    config,
		stopCh:    make(chan struct{}),
		listeners: make([]func(event IdentityAllocatorEvent), 0),
	}
}

// NewDefaultCache creates a new identity cache with default configuration
func NewDefaultCache() *Cache {
	return NewCache(DefaultCacheConfig())
}

// Start begins the background cleanup goroutine
func (c *Cache) Start() {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.mu.Unlock()

	if c.config.TTL > 0 {
		go c.cleanupLoop()
	}
}

// Stop stops the background cleanup goroutine
func (c *Cache) Stop() {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return
	}
	c.started = false
	c.mu.Unlock()

	close(c.stopCh)
}

// cleanupLoop periodically removes expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes expired entries from the cache
func (c *Cache) cleanup() {
	if c.config.TTL == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := make([]NumericIdentity, 0)

	for id, entry := range c.byID {
		if now.Sub(entry.lastAccess) > c.config.TTL {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		entry := c.byID[id]
		if entry != nil {
			delete(c.byID, id)
			if entry.identity.LabelHash != "" {
				delete(c.byHash, entry.identity.LabelHash)
			}
			log.WithField("identity", id).Debug("Expired identity from cache")
		}
	}

	if len(expired) > 0 {
		log.WithField("count", len(expired)).Debug("Cleaned up expired identities")
	}
}

// Upsert adds or updates an identity in the cache
func (c *Cache) Upsert(identity *Identity) {
	if identity == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict for size limits
	if len(c.byID) >= c.config.MaxSize {
		c.evictOldest()
	}

	entry := &cacheEntry{
		identity:   identity.Clone(),
		lastAccess: time.Now(),
	}

	// Check for existing entry
	oldEntry := c.byID[identity.ID]
	eventType := IdentityEventAdd
	if oldEntry != nil {
		eventType = IdentityEventUpdate
		// Remove old hash entry if hash changed
		if oldEntry.identity.LabelHash != identity.LabelHash {
			delete(c.byHash, oldEntry.identity.LabelHash)
		}
	}

	// Insert new entry
	c.byID[identity.ID] = entry
	if identity.LabelHash != "" {
		c.byHash[identity.LabelHash] = entry
	}

	// Notify listeners
	c.notifyListeners(IdentityAllocatorEvent{
		Type:     eventType,
		Identity: identity,
	})

	log.WithFields(log.Fields{
		"id":     identity.ID,
		"labels": identity.Labels,
	}).Debug("Upserted identity to cache")
}

// Delete removes an identity from the cache
func (c *Cache) Delete(id NumericIdentity) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.byID[id]
	if !exists {
		return false
	}

	delete(c.byID, id)
	if entry.identity.LabelHash != "" {
		delete(c.byHash, entry.identity.LabelHash)
	}

	// Notify listeners
	c.notifyListeners(IdentityAllocatorEvent{
		Type:     IdentityEventDelete,
		Identity: entry.identity,
	})

	log.WithField("identity", id).Debug("Deleted identity from cache")
	return true
}

// GetByID retrieves an identity by its numeric ID
func (c *Cache) GetByID(id NumericIdentity) (*Identity, bool) {
	c.mu.RLock()
	entry, exists := c.byID[id]
	if !exists {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	// Update last access time
	c.mu.Lock()
	entry.lastAccess = time.Now()
	c.mu.Unlock()

	return entry.identity.Clone(), true
}

// GetByLabels retrieves an identity by its label set
func (c *Cache) GetByLabels(labels map[string]string) (*Identity, bool) {
	hash := ComputeLabelHash(labels)
	return c.GetByLabelHash(hash)
}

// GetByLabelHash retrieves an identity by its label hash
func (c *Cache) GetByLabelHash(hash string) (*Identity, bool) {
	if hash == "" {
		return nil, false
	}

	c.mu.RLock()
	entry, exists := c.byHash[hash]
	if !exists {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	// Update last access time
	c.mu.Lock()
	entry.lastAccess = time.Now()
	c.mu.Unlock()

	return entry.identity.Clone(), true
}

// GetAll returns all identities in the cache
func (c *Cache) GetAll() []*Identity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	identities := make([]*Identity, 0, len(c.byID))
	for _, entry := range c.byID {
		identities = append(identities, entry.identity.Clone())
	}
	return identities
}

// Size returns the number of identities in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.byID)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byID = make(map[NumericIdentity]*cacheEntry)
	c.byHash = make(map[string]*cacheEntry)

	log.Debug("Cleared identity cache")
}

// evictOldest removes the oldest entry from the cache
// Must be called with lock held
func (c *Cache) evictOldest() {
	var oldestID NumericIdentity
	var oldestTime time.Time
	first := true

	for id, entry := range c.byID {
		// Skip reserved identities
		if id.IsReserved() {
			continue
		}
		if first || entry.lastAccess.Before(oldestTime) {
			oldestID = id
			oldestTime = entry.lastAccess
			first = false
		}
	}

	if !first {
		entry := c.byID[oldestID]
		delete(c.byID, oldestID)
		if entry != nil && entry.identity.LabelHash != "" {
			delete(c.byHash, entry.identity.LabelHash)
		}
		log.WithField("identity", oldestID).Debug("Evicted oldest identity from cache")
	}
}

// AddListener adds a listener for cache events
func (c *Cache) AddListener(listener func(event IdentityAllocatorEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// notifyListeners notifies all listeners of an event
// Must be called with lock held
func (c *Cache) notifyListeners(event IdentityAllocatorEvent) {
	for _, listener := range c.listeners {
		go listener(event)
	}
}

// UpsertReservedIdentities adds all reserved identities to the cache
func (c *Cache) UpsertReservedIdentities() {
	reservedIdentities := []struct {
		id     NumericIdentity
		labels map[string]string
	}{
		{IdentityHost, map[string]string{"reserved": "host"}},
		{IdentityWorld, map[string]string{"reserved": "world"}},
		{IdentityUnmanaged, map[string]string{"reserved": "unmanaged"}},
		{IdentityHealth, map[string]string{"reserved": "health"}},
		{IdentityInit, map[string]string{"reserved": "init"}},
		{IdentityRemoteNode, map[string]string{"reserved": "remote-node"}},
		{IdentityKubeAPIServer, map[string]string{"reserved": "kube-apiserver"}},
	}

	for _, ri := range reservedIdentities {
		c.Upsert(NewIdentity(ri.id, ri.labels))
	}

	log.Debug("Upserted reserved identities to cache")
}

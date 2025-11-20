package process

import (
	"container/list"
	"sync"
	"time"
)

// ProcessCache is a thread-safe LRU cache with TTL for process information
type ProcessCache struct {
	mu         sync.RWMutex
	capacity   int
	ttl        time.Duration
	items      map[uint32]*list.Element // PID -> list element
	evictList  *list.List               // LRU list (front = most recent, back = least recent)
}

// cacheEntry represents an entry in the LRU cache
type cacheEntry struct {
	pid  uint32
	info *ProcessInfo
}

// NewProcessCache creates a new LRU cache with specified capacity and TTL
func NewProcessCache(capacity int, ttl time.Duration) *ProcessCache {
	return &ProcessCache{
		capacity:  capacity,
		ttl:       ttl,
		items:     make(map[uint32]*list.Element, capacity),
		evictList: list.New(),
	}
}

// Get retrieves process info from cache by PID
// Returns nil if not found or expired
func (c *ProcessCache) Get(pid uint32) *ProcessInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[pid]
	if !exists {
		return nil
	}

	entry := elem.Value.(*cacheEntry)

	// Check TTL expiration
	if entry.info.IsExpired(c.ttl) {
		// Remove expired entry
		c.removeElement(elem)
		return nil
	}

	// Move to front (most recently used)
	c.evictList.MoveToFront(elem)

	return entry.info
}

// Set adds or updates process info in cache
func (c *ProcessCache) Set(info *ProcessInfo) {
	if info == nil || !info.IsValid() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	pid := info.PID

	// Check if already exists
	if elem, exists := c.items[pid]; exists {
		// Update existing entry
		entry := elem.Value.(*cacheEntry)
		entry.info = info
		c.evictList.MoveToFront(elem)
		return
	}

	// Add new entry
	entry := &cacheEntry{
		pid:  pid,
		info: info,
	}
	elem := c.evictList.PushFront(entry)
	c.items[pid] = elem

	// Evict oldest entry if capacity exceeded
	if c.evictList.Len() > c.capacity {
		c.evictOldest()
	}
}

// Delete removes process info from cache by PID
func (c *ProcessCache) Delete(pid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[pid]; exists {
		c.removeElement(elem)
	}
}

// Size returns the current number of entries in cache
func (c *ProcessCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Len()
}

// Clear removes all entries from cache
func (c *ProcessCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[uint32]*list.Element, c.capacity)
	c.evictList.Init()
}

// CleanExpired removes all expired entries from cache
// Returns the number of entries removed
func (c *ProcessCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for elem := c.evictList.Back(); elem != nil; {
		entry := elem.Value.(*cacheEntry)
		if entry.info.IsExpired(c.ttl) {
			prev := elem.Prev()
			c.removeElement(elem)
			removed++
			elem = prev
		} else {
			// Since entries are ordered by access time,
			// if this entry is not expired, we can stop
			break
		}
	}

	return removed
}

// GetStats returns cache statistics
func (c *ProcessCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		Size:     c.evictList.Len(),
		Capacity: c.capacity,
		TTL:      c.ttl,
	}

	// Count expired entries
	for elem := c.evictList.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*cacheEntry)
		if entry.info.IsExpired(c.ttl) {
			stats.ExpiredCount++
		}
	}

	return stats
}

// evictOldest removes the least recently used entry
func (c *ProcessCache) evictOldest() {
	elem := c.evictList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from both map and list
func (c *ProcessCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(c.items, entry.pid)
	c.evictList.Remove(elem)
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size         int           `json:"size"`
	Capacity     int           `json:"capacity"`
	ExpiredCount int           `json:"expired_count"`
	TTL          time.Duration `json:"ttl"`
}

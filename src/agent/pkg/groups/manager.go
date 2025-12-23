// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: group definitions, label selectors, workload manager
// output: resolved group members (workload IDs and IPs), member cache
// pos: group manager with member resolution and caching - if file updated, must sync with this header comment and pkg/groups/CLAUDE.md
package groups

import (
	"fmt"
	"sync"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	log "github.com/sirupsen/logrus"
)

// GroupManager manages group membership resolution and caching
type GroupManager struct {
	storage     Storage
	workloadMgr *workload.Manager

	// Cache for group member resolution
	cacheMu     sync.RWMutex
	memberCache map[string]*cachedMembers // groupName -> cached members
	cacheEnabled bool
}

// cachedMembers represents cached group membership information
type cachedMembers struct {
	workloadIDs []string  // List of workload IDs that are members
	timestamp   time.Time // When this cache entry was created
}

// NewGroupManager creates a new group manager
// If workloadMgr is nil, member resolution will not be available
func NewGroupManager(storage Storage, workloadMgr *workload.Manager) *GroupManager {
	return &GroupManager{
		storage:      storage,
		workloadMgr:  workloadMgr,
		memberCache:  make(map[string]*cachedMembers),
		cacheEnabled: true, // Cache enabled by default
	}
}

// ResolveGroupMembers resolves which workloads are members of a group
// Returns a list of workloads that match the group's label selectors
func (m *GroupManager) ResolveGroupMembers(groupName string) ([]*workload.Workload, error) {
	// Validate that we have a workload manager
	if m.workloadMgr == nil {
		return nil, fmt.Errorf("workload manager not initialized")
	}

	// Get the group
	group, err := m.storage.GetGroup(groupName)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Get all workloads
	allWorkloads, err := m.workloadMgr.ListWorkloads()
	if err != nil {
		return nil, fmt.Errorf("failed to list workloads: %w", err)
	}

	// Filter workloads that match the group's selectors
	var members []*workload.Workload
	for _, wl := range allWorkloads {
		if MatchesGroup(wl, group) {
			members = append(members, wl)
		}
	}

	// Update cache if enabled
	if m.cacheEnabled {
		m.updateCache(groupName, members)
	}

	log.Debugf("Resolved group members: group=%s, count=%d", groupName, len(members))
	return members, nil
}

// ResolveGroupMemberIDs resolves which workload IDs are members of a group
// This is a lighter-weight version that only returns IDs
func (m *GroupManager) ResolveGroupMemberIDs(groupName string) ([]string, error) {
	// Check cache first if enabled
	if m.cacheEnabled {
		if cached := m.getCachedMemberIDs(groupName); cached != nil {
			log.Debugf("Cache hit for group: %s", groupName)
			return cached, nil
		}
	}

	// Resolve full workloads
	members, err := m.ResolveGroupMembers(groupName)
	if err != nil {
		return nil, err
	}

	// Extract IDs
	ids := make([]string, len(members))
	for i, wl := range members {
		ids[i] = wl.ID
	}

	return ids, nil
}

// IsWorkloadInGroup checks if a specific workload is a member of a group
func (m *GroupManager) IsWorkloadInGroup(workloadID, groupName string) (bool, error) {
	// Validate that we have a workload manager
	if m.workloadMgr == nil {
		return false, fmt.Errorf("workload manager not initialized")
	}

	// Get the workload
	wl, err := m.workloadMgr.GetWorkload(workloadID)
	if err != nil {
		return false, fmt.Errorf("failed to get workload: %w", err)
	}

	// Get the group
	group, err := m.storage.GetGroup(groupName)
	if err != nil {
		return false, fmt.Errorf("failed to get group: %w", err)
	}

	// Check if workload matches group
	matches := MatchesGroup(wl, group)
	return matches, nil
}

// ResolveAllGroupMemberships resolves memberships for all groups
// Returns a map of groupName -> list of workload IDs
func (m *GroupManager) ResolveAllGroupMemberships() (map[string][]string, error) {
	// Get all groups
	groups, err := m.storage.ListGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Resolve members for each group
	result := make(map[string][]string)
	for _, group := range groups {
		memberIDs, err := m.ResolveGroupMemberIDs(group.Name)
		if err != nil {
			log.Warnf("Failed to resolve members for group %s: %v", group.Name, err)
			continue
		}
		result[group.Name] = memberIDs
	}

	return result, nil
}

// InvalidateCache clears the entire membership cache
func (m *GroupManager) InvalidateCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.memberCache = make(map[string]*cachedMembers)
	log.Debug("Group membership cache invalidated")
}

// InvalidateGroupCache clears the cache for a specific group
func (m *GroupManager) InvalidateGroupCache(groupName string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	delete(m.memberCache, groupName)
	log.Debugf("Cache invalidated for group: %s", groupName)
}

// SetCacheEnabled enables or disables caching
func (m *GroupManager) SetCacheEnabled(enabled bool) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.cacheEnabled = enabled
	if !enabled {
		// Clear cache when disabling
		m.memberCache = make(map[string]*cachedMembers)
	}
	log.Debugf("Group membership cache enabled: %v", enabled)
}

// GetCacheStats returns cache statistics
func (m *GroupManager) GetCacheStats() map[string]interface{} {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	return map[string]interface{}{
		"enabled":     m.cacheEnabled,
		"entry_count": len(m.memberCache),
	}
}

// updateCache updates the cache with resolved members
func (m *GroupManager) updateCache(groupName string, members []*workload.Workload) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// Extract workload IDs
	ids := make([]string, len(members))
	for i, wl := range members {
		ids[i] = wl.ID
	}

	// Store in cache
	m.memberCache[groupName] = &cachedMembers{
		workloadIDs: ids,
		timestamp:   time.Now(),
	}
}

// getCachedMemberIDs retrieves cached member IDs if available
func (m *GroupManager) getCachedMemberIDs(groupName string) []string {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	cached, exists := m.memberCache[groupName]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]string, len(cached.workloadIDs))
	copy(result, cached.workloadIDs)
	return result
}

// CreateGroup creates a new group with validation
func (m *GroupManager) CreateGroup(group *Group) error {
	// Validate group
	if err := group.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create group
	if err := m.storage.CreateGroup(group); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	log.Infof("Group created: name=%s, selectors=%d", group.Name, len(group.Selectors))
	return nil
}

// GetGroup retrieves a group by name
func (m *GroupManager) GetGroup(name string) (*Group, error) {
	group, err := m.storage.GetGroup(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return group, nil
}

// UpdateGroup updates an existing group and invalidates its cache
func (m *GroupManager) UpdateGroup(group *Group) error {
	// Validate group
	if err := group.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Update group
	if err := m.storage.UpdateGroup(group); err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	// Invalidate cache for this group
	m.InvalidateGroupCache(group.Name)

	log.Infof("Group updated: name=%s", group.Name)
	return nil
}

// DeleteGroup deletes a group and invalidates its cache
func (m *GroupManager) DeleteGroup(name string) error {
	if err := m.storage.DeleteGroup(name); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	// Invalidate cache for this group
	m.InvalidateGroupCache(name)

	log.Infof("Group deleted: name=%s", name)
	return nil
}

// ListGroups returns all groups
func (m *GroupManager) ListGroups() ([]*Group, error) {
	groups, err := m.storage.ListGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	return groups, nil
}

// Close closes the manager and underlying storage
func (m *GroupManager) Close() error {
	if m.storage != nil {
		return m.storage.Close()
	}
	return nil
}

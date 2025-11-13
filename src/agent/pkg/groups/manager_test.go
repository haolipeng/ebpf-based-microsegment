// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
)

// Helper function to create a test database
func createTestManagerDB(t *testing.T) (*SQLiteGroupStorage, *workload.SQLiteWorkloadStorage, func()) {
	t.Helper()

	// Create temporary database file
	dbPath := fmt.Sprintf("/tmp/test_manager_%d.db", time.Now().UnixNano())

	// Create group storage
	groupStorage, err := NewSQLiteGroupStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create group storage: %v", err)
	}

	// Create workload storage
	workloadStorage, err := workload.NewSQLiteWorkloadStorage(dbPath)
	if err != nil {
		groupStorage.Close()
		t.Fatalf("Failed to create workload storage: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		groupStorage.Close()
		workloadStorage.Close()
		os.Remove(dbPath)
	}

	return groupStorage, workloadStorage, cleanup
}

// Helper function to create test workloads
func createTestWorkload(t *testing.T, mgr *workload.Manager, id, name string, labels map[string]string) *workload.Workload {
	t.Helper()

	wl := workload.NewWorkload(id, name, "test-host")
	wl.Labels = labels

	if err := mgr.CreateWorkload(wl); err != nil {
		t.Fatalf("Failed to create test workload: %v", err)
	}

	return wl
}

// TestNewGroupManager tests manager creation
func TestNewGroupManager(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}

	if mgr.storage == nil {
		t.Error("Expected non-nil storage")
	}

	if mgr.workloadMgr == nil {
		t.Error("Expected non-nil workload manager")
	}

	// Check cache is initialized and enabled
	stats := mgr.GetCacheStats()
	if !stats["enabled"].(bool) {
		t.Error("Expected cache to be enabled by default")
	}
}

// TestResolveGroupMembers_NoWorkloads tests resolution with zero workloads
func TestResolveGroupMembers_NoWorkloads(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create a group
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))

	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Resolve members (should be empty)
	members, err := mgr.ResolveGroupMembers("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}

	if len(members) != 0 {
		t.Errorf("Expected 0 members, got %d", len(members))
	}
}

// TestResolveGroupMembers_OneMatch tests resolution with one matching workload
func TestResolveGroupMembers_OneMatch(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "nginx", map[string]string{"role": "web"})
	createTestWorkload(t, workloadMgr, "wl-2", "mysql", map[string]string{"role": "db"})

	// Create a group that matches only wl-1
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))

	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Resolve members
	members, err := mgr.ResolveGroupMembers("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}

	if len(members) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(members))
	}

	if members[0].ID != "wl-1" {
		t.Errorf("Expected wl-1, got %s", members[0].ID)
	}
}

// TestResolveGroupMembers_MultipleMatches tests resolution with multiple matches
func TestResolveGroupMembers_MultipleMatches(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "nginx", map[string]string{"role": "web", "env": "prod"})
	createTestWorkload(t, workloadMgr, "wl-2", "apache", map[string]string{"role": "web", "env": "prod"})
	createTestWorkload(t, workloadMgr, "wl-3", "haproxy", map[string]string{"role": "lb", "env": "prod"})
	createTestWorkload(t, workloadMgr, "wl-4", "nginx-dev", map[string]string{"role": "web", "env": "dev"})

	// Create a group that matches wl-1 and wl-2 (role=web AND env=prod)
	group := NewGroup("prod-web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	group.AddSelector(NewEqualSelector("env", "prod"))

	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Resolve members
	members, err := mgr.ResolveGroupMembers("prod-web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("Expected 2 members, got %d", len(members))
	}

	// Check that both wl-1 and wl-2 are present
	memberIDs := make(map[string]bool)
	for _, m := range members {
		memberIDs[m.ID] = true
	}

	if !memberIDs["wl-1"] || !memberIDs["wl-2"] {
		t.Errorf("Expected wl-1 and wl-2, got %v", memberIDs)
	}
}

// TestResolveGroupMembers_ComplexSelectors tests complex selector combinations
func TestResolveGroupMembers_ComplexSelectors(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads with various labels
	createTestWorkload(t, workloadMgr, "wl-1", "web-prod", map[string]string{
		"role": "web",
		"env":  "prod",
		"tier": "frontend",
	})
	createTestWorkload(t, workloadMgr, "wl-2", "api-prod", map[string]string{
		"role": "api",
		"env":  "prod",
		"tier": "backend",
	})
	createTestWorkload(t, workloadMgr, "wl-3", "web-staging", map[string]string{
		"role": "web",
		"env":  "staging",
		"tier": "frontend",
	})
	createTestWorkload(t, workloadMgr, "wl-4", "cache", map[string]string{
		"role": "cache",
		"env":  "prod",
	})

	tests := []struct {
		name           string
		groupName      string
		selectors      []LabelSelector
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name:      "in operator - multiple roles",
			groupName: "backends",
			selectors: []LabelSelector{
				NewInSelector("role", []string{"api", "cache"}),
				NewEqualSelector("env", "prod"),
			},
			expectedCount: 2,
			expectedIDs:   []string{"wl-2", "wl-4"},
		},
		{
			name:      "exists operator",
			groupName: "tiered-services",
			selectors: []LabelSelector{
				{Key: "tier", Operator: OpExists},
				NewEqualSelector("env", "prod"),
			},
			expectedCount: 2,
			expectedIDs:   []string{"wl-1", "wl-2"},
		},
		{
			name:      "not-equal operator",
			groupName: "non-web-prod",
			selectors: []LabelSelector{
				NewNotEqualSelector("role", "web"),
				NewEqualSelector("env", "prod"),
			},
			expectedCount: 2,
			expectedIDs:   []string{"wl-2", "wl-4"},
		},
		{
			name:      "not-in operator",
			groupName: "non-api-cache",
			selectors: []LabelSelector{
				NewNotInSelector("role", []string{"api", "cache"}),
			},
			expectedCount: 2,
			expectedIDs:   []string{"wl-1", "wl-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create group
			group := NewGroup(tt.groupName)
			for _, sel := range tt.selectors {
				group.AddSelector(sel)
			}

			if err := mgr.CreateGroup(group); err != nil {
				t.Fatalf("Failed to create group: %v", err)
			}

			// Resolve members
			members, err := mgr.ResolveGroupMembers(tt.groupName)
			if err != nil {
				t.Fatalf("Failed to resolve members: %v", err)
			}

			if len(members) != tt.expectedCount {
				t.Errorf("Expected %d members, got %d", tt.expectedCount, len(members))
			}

			// Check expected IDs
			memberIDs := make(map[string]bool)
			for _, m := range members {
				memberIDs[m.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !memberIDs[expectedID] {
					t.Errorf("Expected member %s not found", expectedID)
				}
			}
		})
	}
}

// TestResolveGroupMemberIDs tests the lighter-weight ID resolution
func TestResolveGroupMemberIDs(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "web1", map[string]string{"role": "web"})
	createTestWorkload(t, workloadMgr, "wl-2", "web2", map[string]string{"role": "web"})

	// Create group
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Resolve member IDs
	ids, err := mgr.ResolveGroupMemberIDs("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve member IDs: %v", err)
	}

	if len(ids) != 2 {
		t.Fatalf("Expected 2 IDs, got %d", len(ids))
	}

	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	if !idMap["wl-1"] || !idMap["wl-2"] {
		t.Errorf("Expected wl-1 and wl-2, got %v", ids)
	}
}

// TestIsWorkloadInGroup tests checking if a workload is in a group
func TestIsWorkloadInGroup(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "web", map[string]string{"role": "web"})
	createTestWorkload(t, workloadMgr, "wl-2", "db", map[string]string{"role": "db"})

	// Create group
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Test wl-1 (should be in group)
	inGroup, err := mgr.IsWorkloadInGroup("wl-1", "web-servers")
	if err != nil {
		t.Fatalf("Failed to check membership: %v", err)
	}
	if !inGroup {
		t.Error("Expected wl-1 to be in group")
	}

	// Test wl-2 (should NOT be in group)
	inGroup, err = mgr.IsWorkloadInGroup("wl-2", "web-servers")
	if err != nil {
		t.Fatalf("Failed to check membership: %v", err)
	}
	if inGroup {
		t.Error("Expected wl-2 to NOT be in group")
	}
}

// TestResolveAllGroupMemberships tests resolving all groups at once
func TestResolveAllGroupMemberships(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "web", map[string]string{"role": "web"})
	createTestWorkload(t, workloadMgr, "wl-2", "db", map[string]string{"role": "db"})
	createTestWorkload(t, workloadMgr, "wl-3", "cache", map[string]string{"role": "cache"})

	// Create groups
	webGroup := NewGroup("web-servers")
	webGroup.AddSelector(NewEqualSelector("role", "web"))
	if err := mgr.CreateGroup(webGroup); err != nil {
		t.Fatalf("Failed to create web group: %v", err)
	}

	dbGroup := NewGroup("databases")
	dbGroup.AddSelector(NewEqualSelector("role", "db"))
	if err := mgr.CreateGroup(dbGroup); err != nil {
		t.Fatalf("Failed to create db group: %v", err)
	}

	// Resolve all memberships
	memberships, err := mgr.ResolveAllGroupMemberships()
	if err != nil {
		t.Fatalf("Failed to resolve all memberships: %v", err)
	}

	if len(memberships) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(memberships))
	}

	// Check web-servers
	if len(memberships["web-servers"]) != 1 {
		t.Errorf("Expected 1 member in web-servers, got %d", len(memberships["web-servers"]))
	}

	// Check databases
	if len(memberships["databases"]) != 1 {
		t.Errorf("Expected 1 member in databases, got %d", len(memberships["databases"]))
	}
}

// TestCaching tests the caching mechanism
func TestCaching(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "web", map[string]string{"role": "web"})

	// Create group
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// First resolution (cache miss)
	ids1, err := mgr.ResolveGroupMemberIDs("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}

	stats := mgr.GetCacheStats()
	if stats["entry_count"].(int) != 1 {
		t.Errorf("Expected 1 cache entry, got %d", stats["entry_count"].(int))
	}

	// Second resolution (cache hit)
	ids2, err := mgr.ResolveGroupMemberIDs("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}

	if len(ids1) != len(ids2) {
		t.Errorf("Cache returned different results: %v vs %v", ids1, ids2)
	}

	// Invalidate cache
	mgr.InvalidateGroupCache("web-servers")

	stats = mgr.GetCacheStats()
	if stats["entry_count"].(int) != 0 {
		t.Errorf("Expected 0 cache entries after invalidation, got %d", stats["entry_count"].(int))
	}
}

// TestCacheInvalidationOnUpdate tests cache invalidation when group is updated
func TestCacheInvalidationOnUpdate(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create workloads
	createTestWorkload(t, workloadMgr, "wl-1", "web", map[string]string{"role": "web", "env": "prod"})
	createTestWorkload(t, workloadMgr, "wl-2", "web2", map[string]string{"role": "web", "env": "dev"})

	// Create initial group (matches both)
	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	if err := mgr.CreateGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// First resolution
	ids1, err := mgr.ResolveGroupMemberIDs("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}
	if len(ids1) != 2 {
		t.Fatalf("Expected 2 members initially, got %d", len(ids1))
	}

	// Update group to add env=prod selector (should match only wl-1)
	group.AddSelector(NewEqualSelector("env", "prod"))
	if err := mgr.UpdateGroup(group); err != nil {
		t.Fatalf("Failed to update group: %v", err)
	}

	// Verify cache was invalidated
	stats := mgr.GetCacheStats()
	if stats["entry_count"].(int) != 0 {
		t.Errorf("Expected cache to be cleared after update, got %d entries", stats["entry_count"].(int))
	}

	// Second resolution (should reflect updated selectors)
	ids2, err := mgr.ResolveGroupMemberIDs("web-servers")
	if err != nil {
		t.Fatalf("Failed to resolve members: %v", err)
	}
	if len(ids2) != 1 {
		t.Fatalf("Expected 1 member after update, got %d", len(ids2))
	}
}

// TestPerformance tests performance with 100 workloads and 10 groups
func TestPerformance(t *testing.T) {
	groupStorage, workloadStorage, cleanup := createTestManagerDB(t)
	defer cleanup()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create 100 workloads with various labels
	roles := []string{"web", "api", "db", "cache", "mq"}
	envs := []string{"prod", "staging", "dev"}

	for i := 0; i < 100; i++ {
		labels := map[string]string{
			"role": roles[i%len(roles)],
			"env":  envs[i%len(envs)],
			"id":   fmt.Sprintf("service-%d", i),
		}
		createTestWorkload(t, workloadMgr, fmt.Sprintf("wl-%d", i), fmt.Sprintf("service-%d", i), labels)
	}

	// Create 10 groups with different selector combinations
	for i := 0; i < 10; i++ {
		group := NewGroup(fmt.Sprintf("group-%d", i))
		group.AddSelector(NewEqualSelector("role", roles[i%len(roles)]))
		if i%2 == 0 {
			group.AddSelector(NewEqualSelector("env", envs[i%len(envs)]))
		}
		if err := mgr.CreateGroup(group); err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}
	}

	// Measure resolution performance
	start := time.Now()
	memberships, err := mgr.ResolveAllGroupMemberships()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to resolve memberships: %v", err)
	}

	if len(memberships) != 10 {
		t.Errorf("Expected 10 groups, got %d", len(memberships))
	}

	// Performance target: < 100ms for 100 workloads + 10 groups
	if elapsed > 100*time.Millisecond {
		t.Errorf("Performance target not met: took %v (target: <100ms)", elapsed)
	}

	t.Logf("Performance: Resolved 10 groups with 100 workloads in %v", elapsed)
}

// BenchmarkResolveGroupMembers benchmarks member resolution
func BenchmarkResolveGroupMembers(b *testing.B) {
	// Create test database
	dbPath := fmt.Sprintf("/tmp/bench_manager_%d.db", time.Now().UnixNano())
	defer os.Remove(dbPath)

	groupStorage, _ := NewSQLiteGroupStorage(dbPath)
	defer groupStorage.Close()

	workloadStorage, _ := workload.NewSQLiteWorkloadStorage(dbPath)
	defer workloadStorage.Close()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create test data
	for i := 0; i < 50; i++ {
		wl := workload.NewWorkload(fmt.Sprintf("wl-%d", i), fmt.Sprintf("service-%d", i), "test-host")
		wl.Labels = map[string]string{
			"role": "web",
			"env":  "prod",
		}
		workloadMgr.CreateWorkload(wl)
	}

	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	mgr.CreateGroup(group)

	// Disable cache for fair benchmarking
	mgr.SetCacheEnabled(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.ResolveGroupMembers("web-servers")
	}
}

// BenchmarkResolveGroupMembersWithCache benchmarks with cache enabled
func BenchmarkResolveGroupMembersWithCache(b *testing.B) {
	// Create test database
	dbPath := fmt.Sprintf("/tmp/bench_manager_cache_%d.db", time.Now().UnixNano())
	defer os.Remove(dbPath)

	groupStorage, _ := NewSQLiteGroupStorage(dbPath)
	defer groupStorage.Close()

	workloadStorage, _ := workload.NewSQLiteWorkloadStorage(dbPath)
	defer workloadStorage.Close()

	workloadMgr := workload.NewManager(workloadStorage)
	mgr := NewGroupManager(groupStorage, workloadMgr)

	// Create test data
	for i := 0; i < 50; i++ {
		wl := workload.NewWorkload(fmt.Sprintf("wl-%d", i), fmt.Sprintf("service-%d", i), "test-host")
		wl.Labels = map[string]string{
			"role": "web",
			"env":  "prod",
		}
		workloadMgr.CreateWorkload(wl)
	}

	group := NewGroup("web-servers")
	group.AddSelector(NewEqualSelector("role", "web"))
	mgr.CreateGroup(group)

	// Cache enabled by default

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.ResolveGroupMemberIDs("web-servers")
	}
}

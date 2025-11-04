// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"os"
	"testing"
)

// setupTestStorage creates a temporary SQLite database for testing
func setupTestStorage(t *testing.T) (*SQLiteGroupStorage, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "group_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()

	storage, err := NewSQLiteGroupStorage(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.Remove(dbPath)
	}

	return storage, cleanup
}

func TestCreateGroup(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	group := NewGroup("web-tier")
	group.Description = "Web servers"
	group.AddSelector(NewEqualSelector("role", "web"))

	err := storage.CreateGroup(group)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Verify group was created
	count, err := storage.GetGroupCount()
	if err != nil {
		t.Fatalf("Failed to get group count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 group, got %d", count)
	}
}

func TestGetGroup(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create test group
	original := NewGroup("db-tier")
	original.Description = "Database servers"
	original.AddSelector(NewEqualSelector("role", "db"))
	original.AddSelector(NewEqualSelector("env", "prod"))

	err := storage.CreateGroup(original)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Retrieve group
	retrieved, err := storage.GetGroup("db-tier")
	if err != nil {
		t.Fatalf("Failed to get group: %v", err)
	}

	// Verify all fields
	if retrieved.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, retrieved.Name)
	}

	if retrieved.Description != original.Description {
		t.Errorf("Description mismatch: expected %s, got %s", original.Description, retrieved.Description)
	}

	if len(retrieved.Selectors) != 2 {
		t.Errorf("Expected 2 selectors, got %d", len(retrieved.Selectors))
	}

	// Verify selectors
	if retrieved.Selectors[0].Key != "role" || retrieved.Selectors[0].Values[0] != "db" {
		t.Errorf("First selector mismatch")
	}

	if retrieved.Selectors[1].Key != "env" || retrieved.Selectors[1].Values[0] != "prod" {
		t.Errorf("Second selector mismatch")
	}
}

func TestGetGroupNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetGroup("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent group, got nil")
	}
}

func TestUpdateGroup(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create initial group
	group := NewGroup("api-tier")
	group.Description = "API servers"
	group.AddSelector(NewEqualSelector("role", "api"))

	err := storage.CreateGroup(group)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Update group
	group.Description = "Updated API servers"
	group.AddSelector(NewEqualSelector("env", "prod"))

	err = storage.UpdateGroup(group)
	if err != nil {
		t.Fatalf("Failed to update group: %v", err)
	}

	// Verify update
	updated, err := storage.GetGroup("api-tier")
	if err != nil {
		t.Fatalf("Failed to get updated group: %v", err)
	}

	if updated.Description != "Updated API servers" {
		t.Errorf("Description not updated: got %s", updated.Description)
	}

	if len(updated.Selectors) != 2 {
		t.Errorf("Expected 2 selectors after update, got %d", len(updated.Selectors))
	}

	// Verify updated_at changed
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestDeleteGroup(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create group
	group := NewGroup("test-group")
	group.AddSelector(NewExistsSelector("version"))

	err := storage.CreateGroup(group)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Verify it exists
	count, _ := storage.GetGroupCount()
	if count != 1 {
		t.Errorf("Expected 1 group before delete, got %d", count)
	}

	// Delete group
	err = storage.DeleteGroup("test-group")
	if err != nil {
		t.Fatalf("Failed to delete group: %v", err)
	}

	// Verify it's gone
	count, _ = storage.GetGroupCount()
	if count != 0 {
		t.Errorf("Expected 0 groups after delete, got %d", count)
	}

	// Try to get deleted group
	_, err = storage.GetGroup("test-group")
	if err == nil {
		t.Error("Expected error when getting deleted group")
	}
}

func TestDeleteGroupNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	err := storage.DeleteGroup("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent group")
	}
}

func TestListGroups(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create multiple groups
	groups := []*Group{
		NewGroup("group-1"),
		NewGroup("group-2"),
		NewGroup("group-3"),
	}

	for _, g := range groups {
		g.AddSelector(NewExistsSelector("test"))
		if err := storage.CreateGroup(g); err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}
	}

	// List all groups
	list, err := storage.ListGroups()
	if err != nil {
		t.Fatalf("Failed to list groups: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(list))
	}
}

func TestGroupExists(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create group
	group := NewGroup("exists-test")
	group.AddSelector(NewExistsSelector("test"))
	storage.CreateGroup(group)

	// Test existing group
	exists, err := storage.GroupExists("exists-test")
	if err != nil {
		t.Fatalf("Failed to check group existence: %v", err)
	}

	if !exists {
		t.Error("Group should exist")
	}

	// Test nonexistent group
	exists, err = storage.GroupExists("nonexistent")
	if err != nil {
		t.Fatalf("Failed to check group existence: %v", err)
	}

	if exists {
		t.Error("Nonexistent group should not exist")
	}
}

func TestClearAll(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create multiple groups
	for i := 0; i < 5; i++ {
		g := NewGroup(string(rune('a' + i)))
		g.AddSelector(NewExistsSelector("test"))
		if err := storage.CreateGroup(g); err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}
	}

	// Verify they exist
	count, _ := storage.GetGroupCount()
	if count != 5 {
		t.Errorf("Expected 5 groups, got %d", count)
	}

	// Clear all
	err := storage.ClearAll()
	if err != nil {
		t.Fatalf("Failed to clear all groups: %v", err)
	}

	// Verify all are gone
	count, _ = storage.GetGroupCount()
	if count != 0 {
		t.Errorf("Expected 0 groups after clear, got %d", count)
	}
}

func TestGroupValidation(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	tests := []struct {
		name    string
		group   *Group
		wantErr bool
	}{
		{
			name: "valid group",
			group: func() *Group {
				g := NewGroup("valid")
				g.AddSelector(NewEqualSelector("role", "web"))
				return g
			}(),
			wantErr: false,
		},
		{
			name: "empty name",
			group: func() *Group {
				g := NewGroup("")
				g.AddSelector(NewExistsSelector("test"))
				return g
			}(),
			wantErr: true,
		},
		{
			name: "no selectors",
			group: func() *Group {
				return NewGroup("no-selectors")
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.CreateGroup(tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSelectorOperators(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	tests := []struct {
		name     string
		selector LabelSelector
	}{
		{"equal", NewEqualSelector("key", "value")},
		{"not equal", NewNotEqualSelector("key", "value")},
		{"in", NewInSelector("key", []string{"v1", "v2"})},
		{"not in", NewNotInSelector("key", []string{"v1", "v2"})},
		{"exists", NewExistsSelector("key")},
		{"not exists", NewNotExistsSelector("key")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := NewGroup("test-" + tt.name)
			group.AddSelector(tt.selector)

			err := storage.CreateGroup(group)
			if err != nil {
				t.Errorf("Failed to create group with %s selector: %v", tt.name, err)
			}

			// Retrieve and verify
			retrieved, err := storage.GetGroup(group.Name)
			if err != nil {
				t.Errorf("Failed to retrieve group: %v", err)
			}

			if len(retrieved.Selectors) != 1 {
				t.Errorf("Expected 1 selector, got %d", len(retrieved.Selectors))
			}

			if retrieved.Selectors[0].Operator != tt.selector.Operator {
				t.Errorf("Operator mismatch: expected %s, got %s",
					tt.selector.Operator, retrieved.Selectors[0].Operator)
			}
		})
	}
}

func TestComplexSelectors(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create group with multiple complex selectors
	group := NewGroup("complex-group")
	group.Description = "Group with complex selectors"

	selectors := []LabelSelector{
		NewEqualSelector("role", "web"),
		NewInSelector("env", []string{"prod", "staging"}),
		NewNotInSelector("region", []string{"us-west-1", "us-west-2"}),
		NewExistsSelector("version"),
		NewNotExistsSelector("deprecated"),
	}

	for _, sel := range selectors {
		group.AddSelector(sel)
	}

	err := storage.CreateGroup(group)
	if err != nil {
		t.Fatalf("Failed to create complex group: %v", err)
	}

	// Retrieve and verify
	retrieved, err := storage.GetGroup("complex-group")
	if err != nil {
		t.Fatalf("Failed to retrieve complex group: %v", err)
	}

	if len(retrieved.Selectors) != 5 {
		t.Errorf("Expected 5 selectors, got %d", len(retrieved.Selectors))
	}

	// Verify each selector type
	operators := make(map[SelectorOperator]bool)
	for _, sel := range retrieved.Selectors {
		operators[sel.Operator] = true
	}

	expectedOps := []SelectorOperator{OpEqual, OpIn, OpNotIn, OpExists, OpNotExists}
	for _, op := range expectedOps {
		if !operators[op] {
			t.Errorf("Missing operator: %s", op)
		}
	}
}

func TestListGroupSummaries(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create groups with different selector counts
	group1 := NewGroup("group-1")
	group1.AddSelector(NewExistsSelector("test"))
	storage.CreateGroup(group1)

	group2 := NewGroup("group-2")
	group2.AddSelector(NewEqualSelector("role", "web"))
	group2.AddSelector(NewEqualSelector("env", "prod"))
	storage.CreateGroup(group2)

	// Get summaries
	summaries, err := storage.ListGroupSummaries()
	if err != nil {
		t.Fatalf("Failed to list summaries: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Verify selector counts
	for _, summary := range summaries {
		if summary.Name == "group-1" && summary.SelectorCount != 1 {
			t.Errorf("group-1 should have 1 selector, got %d", summary.SelectorCount)
		}
		if summary.Name == "group-2" && summary.SelectorCount != 2 {
			t.Errorf("group-2 should have 2 selectors, got %d", summary.SelectorCount)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	done := make(chan bool)
	numGoroutines := 10

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			g := NewGroup(string(rune('a' + id)))
			g.AddSelector(NewEqualSelector("id", string(rune('0'+id))))

			if err := storage.CreateGroup(g); err != nil {
				t.Errorf("Concurrent create failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all groups were created
	count, err := storage.GetGroupCount()
	if err != nil {
		t.Fatalf("Failed to get group count: %v", err)
	}

	if count != numGoroutines {
		t.Errorf("Expected %d groups, got %d", numGoroutines, count)
	}
}

func TestJSONSerialization(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create group with complex selectors
	group := NewGroup("json-test")
	group.Description = "Test JSON serialization"
	group.AddSelector(NewEqualSelector("role", "web"))
	group.AddSelector(NewInSelector("env", []string{"prod", "staging", "qa"}))
	group.AddSelector(NewExistsSelector("version"))

	// Create and retrieve
	err := storage.CreateGroup(group)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	retrieved, err := storage.GetGroup("json-test")
	if err != nil {
		t.Fatalf("Failed to get group: %v", err)
	}

	// Verify all selectors were correctly serialized/deserialized
	if len(retrieved.Selectors) != 3 {
		t.Errorf("Expected 3 selectors, got %d", len(retrieved.Selectors))
	}

	// Check in selector values
	for _, sel := range retrieved.Selectors {
		if sel.Operator == OpIn {
			if len(sel.Values) != 3 {
				t.Errorf("Expected 3 values in 'in' selector, got %d", len(sel.Values))
			}
		}
	}
}

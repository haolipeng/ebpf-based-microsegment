// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package workload

import (
	"net"
	"os"
	"testing"
)

// setupTestManager creates a manager with temporary SQLite storage
func setupTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "workload_manager_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()

	storage, err := NewSQLiteWorkloadStorage(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to create storage: %v", err)
	}

	manager := NewManager(storage)

	cleanup := func() {
		manager.Close()
		os.Remove(dbPath)
	}

	return manager, cleanup
}

func TestManagerCreateWorkload(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	w := NewWorkload("wl-manager-001", "nginx-web", "host-1")
	w.AddIP(net.ParseIP("10.0.1.10"))
	w.AddLabel("role", "web")

	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Verify workload was created
	retrieved, err := manager.GetWorkload("wl-manager-001")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	if retrieved.Name != "nginx-web" {
		t.Errorf("Expected name=nginx-web, got %s", retrieved.Name)
	}
}

func TestManagerCreateDuplicateWorkload(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	w := NewWorkload("wl-dup", "test", "host-1")

	// Create first time
	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Try to create duplicate
	err = manager.CreateWorkload(w)
	if err == nil {
		t.Error("Expected error when creating duplicate workload")
	}
}

func TestManagerValidation(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	tests := []struct {
		name        string
		workload    *Workload
		expectError bool
	}{
		{
			name:        "Empty ID",
			workload:    &Workload{Name: "test", HostID: "host-1"},
			expectError: true,
		},
		{
			name:        "Empty Name",
			workload:    &Workload{ID: "wl-001", HostID: "host-1"},
			expectError: true,
		},
		{
			name:        "Empty HostID",
			workload:    &Workload{ID: "wl-001", Name: "test"},
			expectError: true,
		},
		{
			name:        "Invalid State",
			workload:    &Workload{ID: "wl-001", Name: "test", HostID: "host-1", State: "invalid"},
			expectError: true,
		},
		{
			name:        "Valid Workload",
			workload:    NewWorkload("wl-valid", "valid-test", "host-1"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateWorkload(tt.workload)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestManagerUpdateWorkload(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create initial workload
	w := NewWorkload("wl-update", "test", "host-1")
	w.AddLabel("version", "1.0")

	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Update workload
	w.AddLabel("version", "2.0")
	w.AddLabel("env", "prod")

	err = manager.UpdateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to update workload: %v", err)
	}

	// Verify update
	updated, err := manager.GetWorkload("wl-update")
	if err != nil {
		t.Fatalf("Failed to get updated workload: %v", err)
	}

	if updated.Labels["version"] != "2.0" {
		t.Errorf("Expected version=2.0, got %s", updated.Labels["version"])
	}

	if updated.Labels["env"] != "prod" {
		t.Errorf("Expected env=prod, got %s", updated.Labels["env"])
	}
}

func TestManagerDeleteWorkload(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workload
	w := NewWorkload("wl-delete", "test", "host-1")
	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Delete workload
	err = manager.DeleteWorkload("wl-delete")
	if err != nil {
		t.Fatalf("Failed to delete workload: %v", err)
	}

	// Verify deletion
	_, err = manager.GetWorkload("wl-delete")
	if err == nil {
		t.Error("Expected error when getting deleted workload")
	}
}

func TestManagerListWorkloads(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create multiple workloads
	workloads := []string{"wl-list-1", "wl-list-2", "wl-list-3"}
	for _, id := range workloads {
		w := NewWorkload(id, "test", "host-1")
		if err := manager.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// List all workloads
	list, err := manager.ListWorkloads()
	if err != nil {
		t.Fatalf("Failed to list workloads: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 workloads, got %d", len(list))
	}
}

func TestManagerListWorkloadsByLabel(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workloads with different labels
	web1 := NewWorkload("wl-web-1", "web-1", "host-1")
	web1.AddLabel("role", "web")

	web2 := NewWorkload("wl-web-2", "web-2", "host-1")
	web2.AddLabel("role", "web")

	db1 := NewWorkload("wl-db-1", "db-1", "host-2")
	db1.AddLabel("role", "db")

	for _, w := range []*Workload{web1, web2, db1} {
		if err := manager.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// List web workloads
	webWorkloads, err := manager.ListWorkloadsByLabel("role", "web")
	if err != nil {
		t.Fatalf("Failed to list workloads by label: %v", err)
	}

	if len(webWorkloads) != 2 {
		t.Errorf("Expected 2 web workloads, got %d", len(webWorkloads))
	}
}

func TestManagerListRunningWorkloads(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workloads with different states
	running := NewWorkload("wl-running", "running-1", "host-1")
	running.State = WorkloadRunning

	stopped := NewWorkload("wl-stopped", "stopped-1", "host-2")
	stopped.State = WorkloadStopped

	for _, w := range []*Workload{running, stopped} {
		if err := manager.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// List only running workloads
	runningWorkloads, err := manager.ListRunningWorkloads()
	if err != nil {
		t.Fatalf("Failed to list running workloads: %v", err)
	}

	if len(runningWorkloads) != 1 {
		t.Errorf("Expected 1 running workload, got %d", len(runningWorkloads))
	}

	if runningWorkloads[0].ID != "wl-running" {
		t.Errorf("Expected wl-running, got %s", runningWorkloads[0].ID)
	}
}

func TestManagerUpdateWorkloadLabels(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workload
	w := NewWorkload("wl-labels", "test", "host-1")
	w.AddLabel("old", "value")

	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Update labels
	newLabels := map[string]string{
		"role": "web",
		"env":  "prod",
	}

	err = manager.UpdateWorkloadLabels("wl-labels", newLabels)
	if err != nil {
		t.Fatalf("Failed to update labels: %v", err)
	}

	// Verify
	updated, err := manager.GetWorkload("wl-labels")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	if len(updated.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(updated.Labels))
	}

	if updated.Labels["role"] != "web" {
		t.Errorf("Expected role=web, got %s", updated.Labels["role"])
	}

	// Old label should be replaced
	if _, exists := updated.Labels["old"]; exists {
		t.Error("Old label should be replaced")
	}
}

func TestManagerAddWorkloadLabel(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workload
	w := NewWorkload("wl-add-label", "test", "host-1")
	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Add label
	err = manager.AddWorkloadLabel("wl-add-label", "role", "web")
	if err != nil {
		t.Fatalf("Failed to add label: %v", err)
	}

	// Verify
	updated, err := manager.GetWorkload("wl-add-label")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	if updated.Labels["role"] != "web" {
		t.Errorf("Expected role=web, got %s", updated.Labels["role"])
	}
}

func TestManagerRemoveWorkloadLabel(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workload with labels
	w := NewWorkload("wl-remove-label", "test", "host-1")
	w.AddLabel("role", "web")
	w.AddLabel("env", "prod")

	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Remove label
	err = manager.RemoveWorkloadLabel("wl-remove-label", "env")
	if err != nil {
		t.Fatalf("Failed to remove label: %v", err)
	}

	// Verify
	updated, err := manager.GetWorkload("wl-remove-label")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	if len(updated.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(updated.Labels))
	}

	if _, exists := updated.Labels["env"]; exists {
		t.Error("Env label should be removed")
	}

	if updated.Labels["role"] != "web" {
		t.Error("Role label should still exist")
	}
}

func TestManagerUpdateWorkloadState(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create workload
	w := NewWorkload("wl-state", "test", "host-1")
	err := manager.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Update state
	err = manager.UpdateWorkloadState("wl-state", WorkloadPaused)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Verify
	updated, err := manager.GetWorkload("wl-state")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	if updated.State != WorkloadPaused {
		t.Errorf("Expected state=paused, got %s", updated.State)
	}
}

func TestManagerGetWorkloadCount(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Initially no workloads
	count, err := manager.GetWorkloadCount()
	if err != nil {
		t.Fatalf("Failed to get workload count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 workloads, got %d", count)
	}

	// Create workloads
	for i := 0; i < 5; i++ {
		w := NewWorkload(string(rune('a'+i)), "test", "host-1")
		if err := manager.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// Check count
	count, err = manager.GetWorkloadCount()
	if err != nil {
		t.Fatalf("Failed to get workload count: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 workloads, got %d", count)
	}
}

func TestManagerConcurrentOperations(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	done := make(chan bool)
	numGoroutines := 10

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			w := NewWorkload(
				string(rune('a'+id)),
				"concurrent-test",
				"host-1",
			)
			if err := manager.CreateWorkload(w); err != nil {
				t.Errorf("Concurrent create failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all creates
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify count
	count, err := manager.GetWorkloadCount()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != numGoroutines {
		t.Errorf("Expected %d workloads, got %d", numGoroutines, count)
	}
}

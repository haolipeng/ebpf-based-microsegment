// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package workload

import (
	"net"
	"os"
	"testing"
	"time"
)

// setupTestStorage creates a temporary SQLite database for testing
func setupTestStorage(t *testing.T) (*SQLiteWorkloadStorage, func()) {
	t.Helper()

	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "workload_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()

	// Create storage
	storage, err := NewSQLiteWorkloadStorage(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		storage.Close()
		os.Remove(dbPath)
	}

	return storage, cleanup
}

func TestCreateWorkload(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	workload := NewWorkload("wl-001", "nginx-web", "host-1")
	workload.AddIP(net.ParseIP("10.0.1.10"))
	workload.AddMAC("00:11:22:33:44:55")
	workload.AddPort(80)
	workload.AddLabel("role", "web")
	workload.AddLabel("env", "prod")
	workload.Image = "nginx:1.21"
	workload.Namespace = "production"

	err := storage.CreateWorkload(workload)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Verify workload was created
	count, err := storage.GetWorkloadCount()
	if err != nil {
		t.Fatalf("Failed to get workload count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 workload, got %d", count)
	}
}

func TestGetWorkload(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create test workload
	original := NewWorkload("wl-002", "mysql-db", "host-2")
	original.AddIP(net.ParseIP("10.0.2.20"))
	original.AddIP(net.ParseIP("10.0.2.21"))
	original.AddMAC("00:11:22:33:44:66")
	original.AddPort(3306)
	original.AddLabel("role", "db")
	original.AddLabel("app", "mysql")
	original.Image = "mysql:8.0"
	original.Namespace = "production"

	err := storage.CreateWorkload(original)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Retrieve workload
	retrieved, err := storage.GetWorkload("wl-002")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	// Verify all fields
	if retrieved.ID != original.ID {
		t.Errorf("ID mismatch: expected %s, got %s", original.ID, retrieved.ID)
	}

	if retrieved.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, retrieved.Name)
	}

	if len(retrieved.IPs) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(retrieved.IPs))
	}

	if len(retrieved.MACs) != 1 {
		t.Errorf("Expected 1 MAC, got %d", len(retrieved.MACs))
	}

	if len(retrieved.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(retrieved.Ports))
	}

	if len(retrieved.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(retrieved.Labels))
	}

	if retrieved.Labels["role"] != "db" {
		t.Errorf("Expected role=db, got %s", retrieved.Labels["role"])
	}

	if retrieved.Image != "mysql:8.0" {
		t.Errorf("Expected image=mysql:8.0, got %s", retrieved.Image)
	}

	if retrieved.State != WorkloadRunning {
		t.Errorf("Expected state=running, got %s", retrieved.State)
	}
}

func TestGetWorkloadNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetWorkload("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workload, got nil")
	}
}

func TestUpdateWorkload(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create initial workload
	workload := NewWorkload("wl-003", "redis-cache", "host-3")
	workload.AddIP(net.ParseIP("10.0.3.30"))
	workload.AddLabel("role", "cache")

	err := storage.CreateWorkload(workload)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Wait a bit to ensure updated_at will be different
	time.Sleep(10 * time.Millisecond)

	// Update workload
	workload.AddLabel("env", "staging")
	workload.AddIP(net.ParseIP("10.0.3.31"))
	workload.State = WorkloadPaused

	err = storage.UpdateWorkload(workload)
	if err != nil {
		t.Fatalf("Failed to update workload: %v", err)
	}

	// Retrieve and verify
	updated, err := storage.GetWorkload("wl-003")
	if err != nil {
		t.Fatalf("Failed to get updated workload: %v", err)
	}

	if len(updated.Labels) != 2 {
		t.Errorf("Expected 2 labels after update, got %d", len(updated.Labels))
	}

	if updated.Labels["env"] != "staging" {
		t.Errorf("Expected env=staging, got %s", updated.Labels["env"])
	}

	if len(updated.IPs) != 2 {
		t.Errorf("Expected 2 IPs after update, got %d", len(updated.IPs))
	}

	if updated.State != WorkloadPaused {
		t.Errorf("Expected state=paused, got %s", updated.State)
	}

	// Verify updated_at changed
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestDeleteWorkload(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create workload
	workload := NewWorkload("wl-004", "test-workload", "host-4")
	err := storage.CreateWorkload(workload)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	// Verify it exists
	count, _ := storage.GetWorkloadCount()
	if count != 1 {
		t.Errorf("Expected 1 workload before delete, got %d", count)
	}

	// Delete workload
	err = storage.DeleteWorkload("wl-004")
	if err != nil {
		t.Fatalf("Failed to delete workload: %v", err)
	}

	// Verify it's gone
	count, _ = storage.GetWorkloadCount()
	if count != 0 {
		t.Errorf("Expected 0 workloads after delete, got %d", count)
	}

	// Try to get deleted workload
	_, err = storage.GetWorkload("wl-004")
	if err == nil {
		t.Error("Expected error when getting deleted workload")
	}
}

func TestDeleteWorkloadNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	err := storage.DeleteWorkload("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent workload")
	}
}

func TestListWorkloads(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create multiple workloads
	workloads := []*Workload{
		NewWorkload("wl-101", "web-1", "host-1"),
		NewWorkload("wl-102", "web-2", "host-1"),
		NewWorkload("wl-103", "db-1", "host-2"),
	}

	for _, w := range workloads {
		if err := storage.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// List all workloads
	list, err := storage.ListWorkloads()
	if err != nil {
		t.Fatalf("Failed to list workloads: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 workloads, got %d", len(list))
	}
}

func TestListWorkloadsByLabel(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create workloads with different labels
	web1 := NewWorkload("wl-201", "web-1", "host-1")
	web1.AddLabel("role", "web")
	web1.AddLabel("env", "prod")

	web2 := NewWorkload("wl-202", "web-2", "host-1")
	web2.AddLabel("role", "web")
	web2.AddLabel("env", "staging")

	db1 := NewWorkload("wl-203", "db-1", "host-2")
	db1.AddLabel("role", "db")
	db1.AddLabel("env", "prod")

	for _, w := range []*Workload{web1, web2, db1} {
		if err := storage.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// Filter by role=web
	webWorkloads, err := storage.ListWorkloadsByLabel("role", "web")
	if err != nil {
		t.Fatalf("Failed to list workloads by label: %v", err)
	}

	if len(webWorkloads) != 2 {
		t.Errorf("Expected 2 web workloads, got %d", len(webWorkloads))
	}

	// Filter by env=prod
	prodWorkloads, err := storage.ListWorkloadsByLabel("env", "prod")
	if err != nil {
		t.Fatalf("Failed to list workloads by label: %v", err)
	}

	if len(prodWorkloads) != 2 {
		t.Errorf("Expected 2 prod workloads, got %d", len(prodWorkloads))
	}

	// Filter by nonexistent label
	noneWorkloads, err := storage.ListWorkloadsByLabel("nonexistent", "value")
	if err != nil {
		t.Fatalf("Failed to list workloads by label: %v", err)
	}

	if len(noneWorkloads) != 0 {
		t.Errorf("Expected 0 workloads with nonexistent label, got %d", len(noneWorkloads))
	}
}

func TestListWorkloadsByState(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create workloads with different states
	running1 := NewWorkload("wl-301", "running-1", "host-1")
	running1.State = WorkloadRunning

	running2 := NewWorkload("wl-302", "running-2", "host-1")
	running2.State = WorkloadRunning

	stopped := NewWorkload("wl-303", "stopped-1", "host-2")
	stopped.State = WorkloadStopped

	paused := NewWorkload("wl-304", "paused-1", "host-3")
	paused.State = WorkloadPaused

	for _, w := range []*Workload{running1, running2, stopped, paused} {
		if err := storage.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// List running workloads
	runningWorkloads, err := storage.ListWorkloadsByState(WorkloadRunning)
	if err != nil {
		t.Fatalf("Failed to list running workloads: %v", err)
	}

	if len(runningWorkloads) != 2 {
		t.Errorf("Expected 2 running workloads, got %d", len(runningWorkloads))
	}

	// List stopped workloads
	stoppedWorkloads, err := storage.ListWorkloadsByState(WorkloadStopped)
	if err != nil {
		t.Fatalf("Failed to list stopped workloads: %v", err)
	}

	if len(stoppedWorkloads) != 1 {
		t.Errorf("Expected 1 stopped workload, got %d", len(stoppedWorkloads))
	}

	// List paused workloads
	pausedWorkloads, err := storage.ListWorkloadsByState(WorkloadPaused)
	if err != nil {
		t.Fatalf("Failed to list paused workloads: %v", err)
	}

	if len(pausedWorkloads) != 1 {
		t.Errorf("Expected 1 paused workload, got %d", len(pausedWorkloads))
	}
}

func TestConcurrentAccess(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test concurrent writes
	done := make(chan bool)
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			w := NewWorkload(
				string(rune('a'+id)),
				"concurrent-test",
				"host-1",
			)
			w.AddLabel("id", string(rune('0'+id)))

			if err := storage.CreateWorkload(w); err != nil {
				t.Errorf("Concurrent create failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all workloads were created
	count, err := storage.GetWorkloadCount()
	if err != nil {
		t.Fatalf("Failed to get workload count: %v", err)
	}

	if count != numGoroutines {
		t.Errorf("Expected %d workloads, got %d", numGoroutines, count)
	}
}

func TestClearAll(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create multiple workloads
	for i := 0; i < 5; i++ {
		w := NewWorkload(
			string(rune('a'+i)),
			"test",
			"host-1",
		)
		if err := storage.CreateWorkload(w); err != nil {
			t.Fatalf("Failed to create workload: %v", err)
		}
	}

	// Verify they exist
	count, _ := storage.GetWorkloadCount()
	if count != 5 {
		t.Errorf("Expected 5 workloads, got %d", count)
	}

	// Clear all
	err := storage.ClearAll()
	if err != nil {
		t.Fatalf("Failed to clear all workloads: %v", err)
	}

	// Verify all are gone
	count, _ = storage.GetWorkloadCount()
	if count != 0 {
		t.Errorf("Expected 0 workloads after clear, got %d", count)
	}
}

func TestJSONSerialization(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create workload with complex data
	w := NewWorkload("wl-json", "json-test", "host-1")

	// Multiple IPs
	w.AddIP(net.ParseIP("10.0.1.1"))
	w.AddIP(net.ParseIP("192.168.1.100"))
	w.AddIP(net.ParseIP("::1"))

	// Multiple MACs
	w.AddMAC("00:11:22:33:44:55")
	w.AddMAC("AA:BB:CC:DD:EE:FF")

	// Multiple Ports
	w.AddPort(80)
	w.AddPort(443)
	w.AddPort(8080)

	// Multiple Labels
	w.AddLabel("role", "web")
	w.AddLabel("app", "nginx")
	w.AddLabel("env", "prod")
	w.AddLabel("version", "1.21.0")

	// Create and retrieve
	err := storage.CreateWorkload(w)
	if err != nil {
		t.Fatalf("Failed to create workload: %v", err)
	}

	retrieved, err := storage.GetWorkload("wl-json")
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	// Verify all arrays and maps
	if len(retrieved.IPs) != 3 {
		t.Errorf("Expected 3 IPs, got %d", len(retrieved.IPs))
	}

	if len(retrieved.MACs) != 2 {
		t.Errorf("Expected 2 MACs, got %d", len(retrieved.MACs))
	}

	if len(retrieved.Ports) != 3 {
		t.Errorf("Expected 3 ports, got %d", len(retrieved.Ports))
	}

	if len(retrieved.Labels) != 4 {
		t.Errorf("Expected 4 labels, got %d", len(retrieved.Labels))
	}

	// Verify specific values
	if retrieved.Labels["version"] != "1.21.0" {
		t.Errorf("Expected version=1.21.0, got %s", retrieved.Labels["version"])
	}

	if retrieved.Ports[2] != 8080 {
		t.Errorf("Expected port 8080 at index 2, got %d", retrieved.Ports[2])
	}
}

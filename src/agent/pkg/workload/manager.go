// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package workload

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Manager provides high-level workload management operations
type Manager struct {
	storage Storage
	mu      sync.RWMutex // Protects concurrent operations
}

// NewManager creates a new workload manager
func NewManager(storage Storage) *Manager {
	return &Manager{
		storage: storage,
	}
}

// CreateWorkload creates a new workload with validation
func (m *Manager) CreateWorkload(w *Workload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate workload
	if err := m.validateWorkload(w); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if workload already exists
	existing, err := m.storage.GetWorkload(w.ID)
	if err == nil && existing != nil {
		return fmt.Errorf("workload already exists: id=%s", w.ID)
	}

	// Create workload
	if err := m.storage.CreateWorkload(w); err != nil {
		return fmt.Errorf("failed to create workload: %w", err)
	}

	log.Infof("Workload created: id=%s, name=%s, labels=%v", w.ID, w.Name, w.Labels)
	return nil
}

// GetWorkload retrieves a workload by ID
func (m *Manager) GetWorkload(id string) (*Workload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w, err := m.storage.GetWorkload(id)
	if err != nil {
		return nil, fmt.Errorf("workload not found: %w", err)
	}

	return w, nil
}

// UpdateWorkload updates an existing workload
func (m *Manager) UpdateWorkload(w *Workload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate workload
	if err := m.validateWorkload(w); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Update workload
	if err := m.storage.UpdateWorkload(w); err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}

	log.Infof("Workload updated: id=%s, name=%s", w.ID, w.Name)
	return nil
}

// DeleteWorkload removes a workload
func (m *Manager) DeleteWorkload(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.storage.DeleteWorkload(id); err != nil {
		return fmt.Errorf("failed to delete workload: %w", err)
	}

	log.Infof("Workload deleted: id=%s", id)
	return nil
}

// ListWorkloads returns all workloads
func (m *Manager) ListWorkloads() ([]*Workload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workloads, err := m.storage.ListWorkloads()
	if err != nil {
		return nil, fmt.Errorf("failed to list workloads: %w", err)
	}

	return workloads, nil
}

// ListWorkloadsByLabel returns workloads matching a specific label
func (m *Manager) ListWorkloadsByLabel(key, value string) ([]*Workload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workloads, err := m.storage.ListWorkloadsByLabel(key, value)
	if err != nil {
		return nil, fmt.Errorf("failed to list workloads by label: %w", err)
	}

	return workloads, nil
}

// ListRunningWorkloads returns all running workloads
func (m *Manager) ListRunningWorkloads() ([]*Workload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workloads, err := m.storage.ListWorkloadsByState(WorkloadRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to list running workloads: %w", err)
	}

	return workloads, nil
}

// UpdateWorkloadLabels updates only the labels of a workload
func (m *Manager) UpdateWorkloadLabels(id string, labels map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing workload
	w, err := m.storage.GetWorkload(id)
	if err != nil {
		return fmt.Errorf("workload not found: %w", err)
	}

	// Update labels
	w.Labels = labels

	// Save
	if err := m.storage.UpdateWorkload(w); err != nil {
		return fmt.Errorf("failed to update workload labels: %w", err)
	}

	log.Infof("Workload labels updated: id=%s, labels=%v", id, labels)
	return nil
}

// AddWorkloadLabel adds or updates a single label
func (m *Manager) AddWorkloadLabel(id, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing workload
	w, err := m.storage.GetWorkload(id)
	if err != nil {
		return fmt.Errorf("workload not found: %w", err)
	}

	// Add label
	w.AddLabel(key, value)

	// Save
	if err := m.storage.UpdateWorkload(w); err != nil {
		return fmt.Errorf("failed to add workload label: %w", err)
	}

	log.Debugf("Label added to workload: id=%s, %s=%s", id, key, value)
	return nil
}

// RemoveWorkloadLabel removes a label from a workload
func (m *Manager) RemoveWorkloadLabel(id, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing workload
	w, err := m.storage.GetWorkload(id)
	if err != nil {
		return fmt.Errorf("workload not found: %w", err)
	}

	// Remove label
	w.RemoveLabel(key)

	// Save
	if err := m.storage.UpdateWorkload(w); err != nil {
		return fmt.Errorf("failed to remove workload label: %w", err)
	}

	log.Debugf("Label removed from workload: id=%s, key=%s", id, key)
	return nil
}

// UpdateWorkloadState updates the state of a workload
func (m *Manager) UpdateWorkloadState(id string, state WorkloadState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing workload
	w, err := m.storage.GetWorkload(id)
	if err != nil {
		return fmt.Errorf("workload not found: %w", err)
	}

	// Update state
	w.State = state

	// Save
	if err := m.storage.UpdateWorkload(w); err != nil {
		return fmt.Errorf("failed to update workload state: %w", err)
	}

	log.Infof("Workload state updated: id=%s, state=%s", id, state)
	return nil
}

// GetWorkloadCount returns the total number of workloads
func (m *Manager) GetWorkloadCount() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count, err := m.storage.GetWorkloadCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get workload count: %w", err)
	}

	return count, nil
}

// validateWorkload performs validation on workload fields
func (m *Manager) validateWorkload(w *Workload) error {
	if w.ID == "" {
		return fmt.Errorf("workload ID cannot be empty")
	}

	if w.Name == "" {
		return fmt.Errorf("workload name cannot be empty")
	}

	if w.HostID == "" {
		return fmt.Errorf("workload host_id cannot be empty")
	}

	// Validate state
	switch w.State {
	case WorkloadRunning, WorkloadStopped, WorkloadPaused:
		// Valid states
	case "":
		// Empty state defaults to running in NewWorkload
		w.State = WorkloadRunning
	default:
		return fmt.Errorf("invalid workload state: %s", w.State)
	}

	// Initialize labels if nil
	if w.Labels == nil {
		w.Labels = make(map[string]string)
	}

	return nil
}

// Close closes the manager and underlying storage
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.storage != nil {
		return m.storage.Close()
	}
	return nil
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: workload records (CRUD operations), query filters
// output: persisted workload data (SQLite), IP-to-labels mapping
// pos: workload persistence layer (SQLite storage) - if file updated, must sync with this header comment and pkg/workload/CLAUDE.md
package workload

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

// Storage defines the interface for workload persistence
type Storage interface {
	// CreateWorkload creates a new workload in storage
	CreateWorkload(w *Workload) error

	// GetWorkload retrieves a workload by ID
	GetWorkload(id string) (*Workload, error)

	// UpdateWorkload updates an existing workload
	UpdateWorkload(w *Workload) error

	// DeleteWorkload removes a workload from storage
	DeleteWorkload(id string) error

	// ListWorkloads returns all workloads
	ListWorkloads() ([]*Workload, error)

	// ListWorkloadsByLabel returns workloads matching a specific label
	ListWorkloadsByLabel(key, value string) ([]*Workload, error)

	// ListWorkloadsByState returns workloads in a specific state
	ListWorkloadsByState(state WorkloadState) ([]*Workload, error)

	// GetWorkloadCount returns the total number of workloads
	GetWorkloadCount() (int, error)

	// Close closes the storage connection
	Close() error
}

// SQLiteWorkloadStorage implements Storage using SQLite database
type SQLiteWorkloadStorage struct {
	db *sql.DB
}

// NewSQLiteWorkloadStorage creates a new SQLite workload storage instance
func NewSQLiteWorkloadStorage(dbPath string) (*SQLiteWorkloadStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	storage := &SQLiteWorkloadStorage{db: db}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Infof("Workload storage initialized: %s", dbPath)
	return storage, nil
}

// initSchema creates the workloads table if it doesn't exist
func (s *SQLiteWorkloadStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS workloads (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		host_id TEXT NOT NULL,
		ips TEXT NOT NULL,
		macs TEXT NOT NULL,
		ports TEXT,
		labels TEXT NOT NULL DEFAULT '{}',
		image TEXT,
		namespace TEXT,
		service_name TEXT,
		pod_name TEXT,
		state TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_workload_host ON workloads(host_id);
	CREATE INDEX IF NOT EXISTS idx_workload_state ON workloads(state);
	CREATE INDEX IF NOT EXISTS idx_workload_namespace ON workloads(namespace);
	CREATE INDEX IF NOT EXISTS idx_workload_name ON workloads(name);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CreateWorkload creates a new workload in the database
func (s *SQLiteWorkloadStorage) CreateWorkload(w *Workload) error {
	// Serialize IPs to JSON
	ipsJSON, err := json.Marshal(ipSliceToStrings(w.IPs))
	if err != nil {
		return fmt.Errorf("failed to marshal IPs: %w", err)
	}

	// Serialize MACs to JSON
	macsJSON, err := json.Marshal(w.MACs)
	if err != nil {
		return fmt.Errorf("failed to marshal MACs: %w", err)
	}

	// Serialize Ports to JSON
	portsJSON, err := json.Marshal(w.Ports)
	if err != nil {
		return fmt.Errorf("failed to marshal Ports: %w", err)
	}

	// Serialize Labels to JSON
	labelsJSON, err := json.Marshal(w.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal Labels: %w", err)
	}

	query := `
	INSERT INTO workloads (
		id, name, host_id, ips, macs, ports, labels,
		image, namespace, service_name, pod_name, state,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		w.ID,
		w.Name,
		w.HostID,
		string(ipsJSON),
		string(macsJSON),
		string(portsJSON),
		string(labelsJSON),
		w.Image,
		w.Namespace,
		w.ServiceName,
		w.PodName,
		string(w.State),
		w.CreatedAt,
		w.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create workload: %w", err)
	}

	log.Debugf("Workload created in storage: id=%s, name=%s", w.ID, w.Name)
	return nil
}

// GetWorkload retrieves a workload by ID
func (s *SQLiteWorkloadStorage) GetWorkload(id string) (*Workload, error) {
	query := `
	SELECT id, name, host_id, ips, macs, ports, labels,
	       image, namespace, service_name, pod_name, state,
	       created_at, updated_at
	FROM workloads
	WHERE id = ?
	`

	row := s.db.QueryRow(query, id)

	var w Workload
	var ipsJSON, macsJSON, portsJSON, labelsJSON string
	var state string

	err := row.Scan(
		&w.ID,
		&w.Name,
		&w.HostID,
		&ipsJSON,
		&macsJSON,
		&portsJSON,
		&labelsJSON,
		&w.Image,
		&w.Namespace,
		&w.ServiceName,
		&w.PodName,
		&state,
		&w.CreatedAt,
		&w.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workload not found: id=%s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan workload: %w", err)
	}

	w.State = WorkloadState(state)

	// Deserialize IPs
	var ipStrings []string
	if err := json.Unmarshal([]byte(ipsJSON), &ipStrings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IPs: %w", err)
	}
	w.IPs = stringsToIPSlice(ipStrings)

	// Deserialize MACs
	if err := json.Unmarshal([]byte(macsJSON), &w.MACs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MACs: %w", err)
	}

	// Deserialize Ports
	if err := json.Unmarshal([]byte(portsJSON), &w.Ports); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Ports: %w", err)
	}

	// Deserialize Labels
	if err := json.Unmarshal([]byte(labelsJSON), &w.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Labels: %w", err)
	}

	return &w, nil
}

// UpdateWorkload updates an existing workload in the database
func (s *SQLiteWorkloadStorage) UpdateWorkload(w *Workload) error {
	// Set updated_at to current time
	w.UpdatedAt = time.Now()

	// Serialize fields to JSON
	ipsJSON, err := json.Marshal(ipSliceToStrings(w.IPs))
	if err != nil {
		return fmt.Errorf("failed to marshal IPs: %w", err)
	}

	macsJSON, err := json.Marshal(w.MACs)
	if err != nil {
		return fmt.Errorf("failed to marshal MACs: %w", err)
	}

	portsJSON, err := json.Marshal(w.Ports)
	if err != nil {
		return fmt.Errorf("failed to marshal Ports: %w", err)
	}

	labelsJSON, err := json.Marshal(w.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal Labels: %w", err)
	}

	query := `
	UPDATE workloads SET
		name = ?,
		host_id = ?,
		ips = ?,
		macs = ?,
		ports = ?,
		labels = ?,
		image = ?,
		namespace = ?,
		service_name = ?,
		pod_name = ?,
		state = ?,
		updated_at = ?
	WHERE id = ?
	`

	result, err := s.db.Exec(query,
		w.Name,
		w.HostID,
		string(ipsJSON),
		string(macsJSON),
		string(portsJSON),
		string(labelsJSON),
		w.Image,
		w.Namespace,
		w.ServiceName,
		w.PodName,
		string(w.State),
		w.UpdatedAt,
		w.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("workload not found: id=%s", w.ID)
	}

	log.Debugf("Workload updated in storage: id=%s", w.ID)
	return nil
}

// DeleteWorkload removes a workload from the database
func (s *SQLiteWorkloadStorage) DeleteWorkload(id string) error {
	query := `DELETE FROM workloads WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workload: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("workload not found: id=%s", id)
	}

	log.Debugf("Workload deleted from storage: id=%s", id)
	return nil
}

// ListWorkloads returns all workloads
func (s *SQLiteWorkloadStorage) ListWorkloads() ([]*Workload, error) {
	query := `
	SELECT id, name, host_id, ips, macs, ports, labels,
	       image, namespace, service_name, pod_name, state,
	       created_at, updated_at
	FROM workloads
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workloads: %w", err)
	}
	defer rows.Close()

	return s.scanWorkloads(rows)
}

// ListWorkloadsByLabel returns workloads matching a specific label
func (s *SQLiteWorkloadStorage) ListWorkloadsByLabel(key, value string) ([]*Workload, error) {
	// Note: This implementation does a full scan and filters in Go
	// For better performance with large datasets, consider using JSON1 extension
	allWorkloads, err := s.ListWorkloads()
	if err != nil {
		return nil, err
	}

	var filtered []*Workload
	for _, w := range allWorkloads {
		if labelValue, exists := w.Labels[key]; exists {
			if labelValue == value {
				filtered = append(filtered, w)
			}
		}
	}

	return filtered, nil
}

// ListWorkloadsByState returns workloads in a specific state
func (s *SQLiteWorkloadStorage) ListWorkloadsByState(state WorkloadState) ([]*Workload, error) {
	query := `
	SELECT id, name, host_id, ips, macs, ports, labels,
	       image, namespace, service_name, pod_name, state,
	       created_at, updated_at
	FROM workloads
	WHERE state = ?
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, string(state))
	if err != nil {
		return nil, fmt.Errorf("failed to query workloads by state: %w", err)
	}
	defer rows.Close()

	return s.scanWorkloads(rows)
}

// GetWorkloadCount returns the total number of workloads
func (s *SQLiteWorkloadStorage) GetWorkloadCount() (int, error) {
	query := `SELECT COUNT(*) FROM workloads`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get workload count: %w", err)
	}

	return count, nil
}

// Close closes the database connection
func (s *SQLiteWorkloadStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ClearAll removes all workloads from storage (useful for testing)
func (s *SQLiteWorkloadStorage) ClearAll() error {
	query := `DELETE FROM workloads`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear workloads: %w", err)
	}

	log.Info("All workloads cleared from storage")
	return nil
}

// scanWorkloads is a helper function to scan multiple workloads from query results
func (s *SQLiteWorkloadStorage) scanWorkloads(rows *sql.Rows) ([]*Workload, error) {
	var workloads []*Workload

	for rows.Next() {
		var w Workload
		var ipsJSON, macsJSON, portsJSON, labelsJSON string
		var state string

		err := rows.Scan(
			&w.ID,
			&w.Name,
			&w.HostID,
			&ipsJSON,
			&macsJSON,
			&portsJSON,
			&labelsJSON,
			&w.Image,
			&w.Namespace,
			&w.ServiceName,
			&w.PodName,
			&state,
			&w.CreatedAt,
			&w.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan workload: %w", err)
		}

		w.State = WorkloadState(state)

		// Deserialize IPs
		var ipStrings []string
		if err := json.Unmarshal([]byte(ipsJSON), &ipStrings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal IPs: %w", err)
		}
		w.IPs = stringsToIPSlice(ipStrings)

		// Deserialize MACs
		if err := json.Unmarshal([]byte(macsJSON), &w.MACs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MACs: %w", err)
		}

		// Deserialize Ports
		if err := json.Unmarshal([]byte(portsJSON), &w.Ports); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Ports: %w", err)
		}

		// Deserialize Labels
		if err := json.Unmarshal([]byte(labelsJSON), &w.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Labels: %w", err)
		}

		workloads = append(workloads, &w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workloads: %w", err)
	}

	log.Debugf("Scanned %d workloads from storage", len(workloads))
	return workloads, nil
}

// Helper functions for IP conversion

// ipSliceToStrings converts []net.IP to []string
func ipSliceToStrings(ips []net.IP) []string {
	strings := make([]string, len(ips))
	for i, ip := range ips {
		strings[i] = ip.String()
	}
	return strings
}

// stringsToIPSlice converts []string to []net.IP
func stringsToIPSlice(strings []string) []net.IP {
	ips := make([]net.IP, 0, len(strings))
	for _, s := range strings {
		if ip := net.ParseIP(s); ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: group records (CRUD operations)
// output: persisted group data (SQLite), query results
// pos: group persistence layer (SQLite storage) - if file updated, must sync with this header comment and pkg/groups/CLAUDE.md
package groups

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

// Storage defines the interface for group persistence
type Storage interface {
	// CreateGroup creates a new group in storage
	CreateGroup(g *Group) error

	// GetGroup retrieves a group by name
	GetGroup(name string) (*Group, error)

	// UpdateGroup updates an existing group
	UpdateGroup(g *Group) error

	// DeleteGroup removes a group from storage
	DeleteGroup(name string) error

	// ListGroups returns all groups
	ListGroups() ([]*Group, error)

	// GroupExists checks if a group with the given name exists
	GroupExists(name string) (bool, error)

	// GetGroupCount returns the total number of groups
	GetGroupCount() (int, error)

	// Close closes the storage connection
	Close() error
}

// SQLiteGroupStorage implements Storage using SQLite database
type SQLiteGroupStorage struct {
	db *sql.DB
}

// NewSQLiteGroupStorage creates a new SQLite group storage instance
func NewSQLiteGroupStorage(dbPath string) (*SQLiteGroupStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	storage := &SQLiteGroupStorage{db: db}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Infof("Group storage initialized: %s", dbPath)
	return storage, nil
}

// initSchema creates the groups table if it doesn't exist
func (s *SQLiteGroupStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS groups (
		name TEXT PRIMARY KEY,
		description TEXT,
		selectors TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
	CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CreateGroup creates a new group in the database
func (s *SQLiteGroupStorage) CreateGroup(g *Group) error {
	// Validate group
	if err := g.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Serialize selectors to JSON
	selectorsJSON, err := json.Marshal(g.Selectors)
	if err != nil {
		return fmt.Errorf("failed to marshal selectors: %w", err)
	}

	query := `
	INSERT INTO groups (name, description, selectors, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		g.Name,
		g.Description,
		string(selectorsJSON),
		g.CreatedAt,
		g.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	log.Debugf("Group created in storage: name=%s, selectors=%d", g.Name, len(g.Selectors))
	return nil
}

// GetGroup retrieves a group by name
func (s *SQLiteGroupStorage) GetGroup(name string) (*Group, error) {
	query := `
	SELECT name, description, selectors, created_at, updated_at
	FROM groups
	WHERE name = ?
	`

	row := s.db.QueryRow(query, name)

	var g Group
	var selectorsJSON string

	err := row.Scan(
		&g.Name,
		&g.Description,
		&selectorsJSON,
		&g.CreatedAt,
		&g.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found: name=%s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan group: %w", err)
	}

	// Deserialize selectors
	if err := json.Unmarshal([]byte(selectorsJSON), &g.Selectors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
	}

	return &g, nil
}

// UpdateGroup updates an existing group in the database
func (s *SQLiteGroupStorage) UpdateGroup(g *Group) error {
	// Validate group
	if err := g.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set updated_at to current time
	g.UpdatedAt = time.Now()

	// Serialize selectors to JSON
	selectorsJSON, err := json.Marshal(g.Selectors)
	if err != nil {
		return fmt.Errorf("failed to marshal selectors: %w", err)
	}

	query := `
	UPDATE groups SET
		description = ?,
		selectors = ?,
		updated_at = ?
	WHERE name = ?
	`

	result, err := s.db.Exec(query,
		g.Description,
		string(selectorsJSON),
		g.UpdatedAt,
		g.Name,
	)

	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("group not found: name=%s", g.Name)
	}

	log.Debugf("Group updated in storage: name=%s", g.Name)
	return nil
}

// DeleteGroup removes a group from the database
func (s *SQLiteGroupStorage) DeleteGroup(name string) error {
	query := `DELETE FROM groups WHERE name = ?`

	result, err := s.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("group not found: name=%s", name)
	}

	log.Debugf("Group deleted from storage: name=%s", name)
	return nil
}

// ListGroups returns all groups
func (s *SQLiteGroupStorage) ListGroups() ([]*Group, error) {
	query := `
	SELECT name, description, selectors, created_at, updated_at
	FROM groups
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	return s.scanGroups(rows)
}

// GroupExists checks if a group with the given name exists
func (s *SQLiteGroupStorage) GroupExists(name string) (bool, error) {
	query := `SELECT COUNT(*) FROM groups WHERE name = ?`

	var count int
	err := s.db.QueryRow(query, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check group existence: %w", err)
	}

	return count > 0, nil
}

// GetGroupCount returns the total number of groups
func (s *SQLiteGroupStorage) GetGroupCount() (int, error) {
	query := `SELECT COUNT(*) FROM groups`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get group count: %w", err)
	}

	return count, nil
}

// Close closes the database connection
func (s *SQLiteGroupStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ClearAll removes all groups from storage (useful for testing)
func (s *SQLiteGroupStorage) ClearAll() error {
	query := `DELETE FROM groups`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear groups: %w", err)
	}

	log.Info("All groups cleared from storage")
	return nil
}

// scanGroups is a helper function to scan multiple groups from query results
func (s *SQLiteGroupStorage) scanGroups(rows *sql.Rows) ([]*Group, error) {
	var groups []*Group

	for rows.Next() {
		var g Group
		var selectorsJSON string

		err := rows.Scan(
			&g.Name,
			&g.Description,
			&selectorsJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		// Deserialize selectors
		if err := json.Unmarshal([]byte(selectorsJSON), &g.Selectors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		groups = append(groups, &g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	log.Debugf("Scanned %d groups from storage", len(groups))
	return groups, nil
}

// ListGroupSummaries returns lightweight summaries of all groups
func (s *SQLiteGroupStorage) ListGroupSummaries() ([]*GroupSummary, error) {
	groups, err := s.ListGroups()
	if err != nil {
		return nil, err
	}

	summaries := make([]*GroupSummary, len(groups))
	for i, g := range groups {
		summaries[i] = g.ToSummary()
	}

	return summaries, nil
}

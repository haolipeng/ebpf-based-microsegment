// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

// Storage defines the interface for policy persistence
type Storage interface {
	// SavePolicy saves a policy to persistent storage
	SavePolicy(p *Policy) error

	// DeletePolicy removes a policy from persistent storage
	DeletePolicy(ruleID uint32) error

	// LoadPolicies loads all policies from persistent storage
	LoadPolicies() ([]Policy, error)

	// Close closes the storage connection
	Close() error
}

// SQLiteStorage implements Storage using SQLite database
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Infof("Policy storage initialized: %s", dbPath)
	return storage, nil
}

// initSchema creates the policies table if it doesn't exist
func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS policies (
		rule_id INTEGER PRIMARY KEY,
		src_ip TEXT NOT NULL,
		dst_ip TEXT NOT NULL,
		src_port INTEGER NOT NULL,
		dst_port INTEGER NOT NULL,
		protocol TEXT NOT NULL,
		action TEXT NOT NULL,
		priority INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_src_ip ON policies(src_ip);
	CREATE INDEX IF NOT EXISTS idx_dst_ip ON policies(dst_ip);
	CREATE INDEX IF NOT EXISTS idx_protocol ON policies(protocol);
	CREATE INDEX IF NOT EXISTS idx_action ON policies(action);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SavePolicy saves a policy to the database
func (s *SQLiteStorage) SavePolicy(p *Policy) error {
	query := `
	INSERT INTO policies (rule_id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(rule_id) DO UPDATE SET
		src_ip = excluded.src_ip,
		dst_ip = excluded.dst_ip,
		src_port = excluded.src_port,
		dst_port = excluded.dst_port,
		protocol = excluded.protocol,
		action = excluded.action,
		priority = excluded.priority,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query,
		p.RuleID,
		p.SrcIP,
		p.DstIP,
		p.SrcPort,
		p.DstPort,
		p.Protocol,
		p.Action,
		p.Priority,
	)

	if err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	log.Debugf("Policy saved to storage: rule_id=%d", p.RuleID)
	return nil
}

// DeletePolicy removes a policy from the database
func (s *SQLiteStorage) DeletePolicy(ruleID uint32) error {
	query := `DELETE FROM policies WHERE rule_id = ?`

	result, err := s.db.Exec(query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("policy not found: rule_id=%d", ruleID)
	}

	log.Debugf("Policy deleted from storage: rule_id=%d", ruleID)
	return nil
}

// LoadPolicies loads all policies from the database
func (s *SQLiteStorage) LoadPolicies() ([]Policy, error) {
	query := `
	SELECT rule_id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority
	FROM policies
	ORDER BY priority DESC, rule_id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		err := rows.Scan(
			&p.RuleID,
			&p.SrcIP,
			&p.DstIP,
			&p.SrcPort,
			&p.DstPort,
			&p.Protocol,
			&p.Action,
			&p.Priority,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	log.Infof("Loaded %d policies from storage", len(policies))
	return policies, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetPolicyCount returns the total number of policies in storage
func (s *SQLiteStorage) GetPolicyCount() (int, error) {
	query := `SELECT COUNT(*) FROM policies`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get policy count: %w", err)
	}

	return count, nil
}

// ClearAll removes all policies from storage (useful for testing)
func (s *SQLiteStorage) ClearAll() error {
	query := `DELETE FROM policies`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear policies: %w", err)
	}

	log.Info("All policies cleared from storage")
	return nil
}

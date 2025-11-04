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

	// PolicyRule operations
	CreatePolicyRule(r *PolicyRule) error
	GetPolicyRule(id uint32) (*PolicyRule, error)
	UpdatePolicyRule(r *PolicyRule) error
	DeletePolicyRule(id uint32) error
	ListPolicyRules() ([]*PolicyRule, error)
	ListEnabledPolicyRules() ([]*PolicyRule, error)

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

	CREATE TABLE IF NOT EXISTS policy_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		from_group TEXT NOT NULL,
		to_group TEXT NOT NULL,
		ports TEXT NOT NULL,
		action TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 100,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (from_group) REFERENCES groups(name) ON DELETE CASCADE,
		FOREIGN KEY (to_group) REFERENCES groups(name) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_policy_rules_id ON policy_rules(id);
	CREATE INDEX IF NOT EXISTS idx_policy_rules_enabled ON policy_rules(enabled);
	CREATE INDEX IF NOT EXISTS idx_policy_rules_from_group ON policy_rules(from_group);
	CREATE INDEX IF NOT EXISTS idx_policy_rules_to_group ON policy_rules(to_group);
	CREATE INDEX IF NOT EXISTS idx_policy_rules_priority ON policy_rules(priority);
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

// CreatePolicyRule creates a new policy rule in the database
func (s *SQLiteStorage) CreatePolicyRule(r *PolicyRule) error {
	// Validate rule
	if err := r.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Serialize ports to JSON
	portsJSON, err := r.PortsToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize ports: %w", err)
	}

	query := `
	INSERT INTO policy_rules (name, description, from_group, to_group, ports, action, priority, enabled)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		r.Name,
		r.Description,
		r.FromGroup,
		r.ToGroup,
		portsJSON,
		r.Action,
		r.Priority,
		boolToInt(r.Enabled),
	)

	if err != nil {
		return fmt.Errorf("failed to create policy rule: %w", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	r.ID = uint32(id)
	log.Infof("Policy rule created: id=%d, name=%s", r.ID, r.Name)
	return nil
}

// GetPolicyRule retrieves a policy rule by ID
func (s *SQLiteStorage) GetPolicyRule(id uint32) (*PolicyRule, error) {
	query := `
	SELECT id, name, description, from_group, to_group, ports, action, priority, enabled,
	       created_at, updated_at
	FROM policy_rules
	WHERE id = ?
	`

	var r PolicyRule
	var portsJSON string
	var enabled int
	var createdAt, updatedAt sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&r.ID,
		&r.Name,
		&r.Description,
		&r.FromGroup,
		&r.ToGroup,
		&portsJSON,
		&r.Action,
		&r.Priority,
		&enabled,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy rule not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy rule: %w", err)
	}

	// Deserialize ports
	if err := r.PortsFromJSON(portsJSON); err != nil {
		return nil, fmt.Errorf("failed to deserialize ports: %w", err)
	}

	r.Enabled = intToBool(enabled)
	if createdAt.Valid {
		r.CreatedAt = createdAt.String
	}
	if updatedAt.Valid {
		r.UpdatedAt = updatedAt.String
	}

	return &r, nil
}

// UpdatePolicyRule updates an existing policy rule
func (s *SQLiteStorage) UpdatePolicyRule(r *PolicyRule) error {
	// Validate rule
	if err := r.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Serialize ports to JSON
	portsJSON, err := r.PortsToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize ports: %w", err)
	}

	query := `
	UPDATE policy_rules
	SET name = ?, description = ?, from_group = ?, to_group = ?,
	    ports = ?, action = ?, priority = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`

	result, err := s.db.Exec(query,
		r.Name,
		r.Description,
		r.FromGroup,
		r.ToGroup,
		portsJSON,
		r.Action,
		r.Priority,
		boolToInt(r.Enabled),
		r.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("policy rule not found: id=%d", r.ID)
	}

	log.Infof("Policy rule updated: id=%d, name=%s", r.ID, r.Name)
	return nil
}

// DeletePolicyRule deletes a policy rule by ID
func (s *SQLiteStorage) DeletePolicyRule(id uint32) error {
	query := `DELETE FROM policy_rules WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("policy rule not found: id=%d", id)
	}

	log.Infof("Policy rule deleted: id=%d", id)
	return nil
}

// ListPolicyRules returns all policy rules
func (s *SQLiteStorage) ListPolicyRules() ([]*PolicyRule, error) {
	query := `
	SELECT id, name, description, from_group, to_group, ports, action, priority, enabled,
	       created_at, updated_at
	FROM policy_rules
	ORDER BY priority DESC, id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy rules: %w", err)
	}
	defer rows.Close()

	var rules []*PolicyRule
	for rows.Next() {
		var r PolicyRule
		var portsJSON string
		var enabled int
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.Description,
			&r.FromGroup,
			&r.ToGroup,
			&portsJSON,
			&r.Action,
			&r.Priority,
			&enabled,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy rule: %w", err)
		}

		// Deserialize ports
		if err := r.PortsFromJSON(portsJSON); err != nil {
			log.Warnf("Failed to deserialize ports for rule %d: %v", r.ID, err)
			continue
		}

		r.Enabled = intToBool(enabled)
		if createdAt.Valid {
			r.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			r.UpdatedAt = updatedAt.String
		}

		rules = append(rules, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy rules: %w", err)
	}

	log.Debugf("Loaded %d policy rules from storage", len(rules))
	return rules, nil
}

// ListEnabledPolicyRules returns only enabled policy rules
func (s *SQLiteStorage) ListEnabledPolicyRules() ([]*PolicyRule, error) {
	query := `
	SELECT id, name, description, from_group, to_group, ports, action, priority, enabled,
	       created_at, updated_at
	FROM policy_rules
	WHERE enabled = 1
	ORDER BY priority DESC, id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled policy rules: %w", err)
	}
	defer rows.Close()

	var rules []*PolicyRule
	for rows.Next() {
		var r PolicyRule
		var portsJSON string
		var enabled int
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.Description,
			&r.FromGroup,
			&r.ToGroup,
			&portsJSON,
			&r.Action,
			&r.Priority,
			&enabled,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy rule: %w", err)
		}

		// Deserialize ports
		if err := r.PortsFromJSON(portsJSON); err != nil {
			log.Warnf("Failed to deserialize ports for rule %d: %v", r.ID, err)
			continue
		}

		r.Enabled = intToBool(enabled)
		if createdAt.Valid {
			r.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			r.UpdatedAt = updatedAt.String
		}

		rules = append(rules, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating enabled policy rules: %w", err)
	}

	log.Debugf("Loaded %d enabled policy rules from storage", len(rules))
	return rules, nil
}

// Helper functions for boolean conversion
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"database/sql"
	"fmt"
	"time"

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

	// CompiledPolicy operations
	SaveCompiledPolicy(cp *CompiledPolicy) error
	GetCompiledPolicy(ruleID uint32) (*CompiledPolicy, error)
	ListCompiledPolicies() ([]*CompiledPolicy, error)
	ListCompiledPoliciesForRule(sourceRuleID uint32) ([]*CompiledPolicy, error)
	GetPolicySource(compiledRuleID uint32) (*PolicyRule, error)
	DeleteCompiledPoliciesForRule(sourceRuleID uint32) error

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

	CREATE TABLE IF NOT EXISTS policy_compilation (
		compiled_rule_id INTEGER PRIMARY KEY,
		source_rule_id INTEGER NOT NULL,
		from_group TEXT NOT NULL,
		to_group TEXT NOT NULL,
		from_workload_id TEXT NOT NULL,
		to_workload_id TEXT NOT NULL,
		compilation_time DATETIME NOT NULL,
		compiler_version TEXT NOT NULL,
		FOREIGN KEY (compiled_rule_id) REFERENCES policies(rule_id) ON DELETE CASCADE,
		FOREIGN KEY (source_rule_id) REFERENCES policy_rules(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_policy_compilation_source ON policy_compilation(source_rule_id);
	CREATE INDEX IF NOT EXISTS idx_policy_compilation_from_group ON policy_compilation(from_group);
	CREATE INDEX IF NOT EXISTS idx_policy_compilation_to_group ON policy_compilation(to_group);
	CREATE INDEX IF NOT EXISTS idx_policy_compilation_from_workload ON policy_compilation(from_workload_id);
	CREATE INDEX IF NOT EXISTS idx_policy_compilation_to_workload ON policy_compilation(to_workload_id);
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

// SaveCompiledPolicy saves a compiled policy to the database
// It saves both the base policy in the policies table and the compilation metadata in policy_compilation
func (s *SQLiteStorage) SaveCompiledPolicy(cp *CompiledPolicy) error {
	// Validate the compiled policy
	if err := cp.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save the base policy
	policyQuery := `
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

	_, err = tx.Exec(policyQuery,
		cp.RuleID,
		cp.SrcIP,
		cp.DstIP,
		cp.SrcPort,
		cp.DstPort,
		cp.Protocol,
		cp.Action,
		cp.Priority,
	)
	if err != nil {
		return fmt.Errorf("failed to save base policy: %w", err)
	}

	// Save the compilation metadata
	compilationQuery := `
	INSERT INTO policy_compilation (compiled_rule_id, source_rule_id, from_group, to_group,
	                                 from_workload_id, to_workload_id, compilation_time, compiler_version)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(compiled_rule_id) DO UPDATE SET
		source_rule_id = excluded.source_rule_id,
		from_group = excluded.from_group,
		to_group = excluded.to_group,
		from_workload_id = excluded.from_workload_id,
		to_workload_id = excluded.to_workload_id,
		compilation_time = excluded.compilation_time,
		compiler_version = excluded.compiler_version
	`

	_, err = tx.Exec(compilationQuery,
		cp.RuleID,
		cp.SourceRuleID,
		cp.FromGroup,
		cp.ToGroup,
		cp.FromWorkloadID,
		cp.ToWorkloadID,
		cp.CompilationTime,
		cp.CompilerVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to save compilation metadata: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debugf("Compiled policy saved: rule_id=%d, source=%d", cp.RuleID, cp.SourceRuleID)
	return nil
}

// GetCompiledPolicy retrieves a compiled policy by its compiled rule ID
func (s *SQLiteStorage) GetCompiledPolicy(ruleID uint32) (*CompiledPolicy, error) {
	query := `
	SELECT p.rule_id, p.src_ip, p.dst_ip, p.src_port, p.dst_port, p.protocol, p.action, p.priority,
	       c.source_rule_id, c.from_group, c.to_group, c.from_workload_id, c.to_workload_id,
	       c.compilation_time, c.compiler_version
	FROM policies p
	INNER JOIN policy_compilation c ON p.rule_id = c.compiled_rule_id
	WHERE p.rule_id = ?
	`

	var cp CompiledPolicy
	var compilationTime string

	err := s.db.QueryRow(query, ruleID).Scan(
		&cp.RuleID,
		&cp.SrcIP,
		&cp.DstIP,
		&cp.SrcPort,
		&cp.DstPort,
		&cp.Protocol,
		&cp.Action,
		&cp.Priority,
		&cp.SourceRuleID,
		&cp.FromGroup,
		&cp.ToGroup,
		&cp.FromWorkloadID,
		&cp.ToWorkloadID,
		&compilationTime,
		&cp.CompilerVersion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("compiled policy not found: rule_id=%d", ruleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get compiled policy: %w", err)
	}

	// Parse compilation time
	cp.CompilationTime, err = parseTime(compilationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compilation time: %w", err)
	}

	return &cp, nil
}

// ListCompiledPolicies returns all compiled policies
func (s *SQLiteStorage) ListCompiledPolicies() ([]*CompiledPolicy, error) {
	query := `
	SELECT p.rule_id, p.src_ip, p.dst_ip, p.src_port, p.dst_port, p.protocol, p.action, p.priority,
	       c.source_rule_id, c.from_group, c.to_group, c.from_workload_id, c.to_workload_id,
	       c.compilation_time, c.compiler_version
	FROM policies p
	INNER JOIN policy_compilation c ON p.rule_id = c.compiled_rule_id
	ORDER BY c.source_rule_id ASC, p.priority DESC, p.rule_id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query compiled policies: %w", err)
	}
	defer rows.Close()

	var policies []*CompiledPolicy
	for rows.Next() {
		var cp CompiledPolicy
		var compilationTime string

		err := rows.Scan(
			&cp.RuleID,
			&cp.SrcIP,
			&cp.DstIP,
			&cp.SrcPort,
			&cp.DstPort,
			&cp.Protocol,
			&cp.Action,
			&cp.Priority,
			&cp.SourceRuleID,
			&cp.FromGroup,
			&cp.ToGroup,
			&cp.FromWorkloadID,
			&cp.ToWorkloadID,
			&compilationTime,
			&cp.CompilerVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan compiled policy: %w", err)
		}

		// Parse compilation time
		cp.CompilationTime, err = parseTime(compilationTime)
		if err != nil {
			log.Warnf("Failed to parse compilation time for rule %d: %v", cp.RuleID, err)
			continue
		}

		policies = append(policies, &cp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating compiled policies: %w", err)
	}

	log.Debugf("Loaded %d compiled policies from storage", len(policies))
	return policies, nil
}

// ListCompiledPoliciesForRule returns all compiled policies for a specific source rule
func (s *SQLiteStorage) ListCompiledPoliciesForRule(sourceRuleID uint32) ([]*CompiledPolicy, error) {
	query := `
	SELECT p.rule_id, p.src_ip, p.dst_ip, p.src_port, p.dst_port, p.protocol, p.action, p.priority,
	       c.source_rule_id, c.from_group, c.to_group, c.from_workload_id, c.to_workload_id,
	       c.compilation_time, c.compiler_version
	FROM policies p
	INNER JOIN policy_compilation c ON p.rule_id = c.compiled_rule_id
	WHERE c.source_rule_id = ?
	ORDER BY p.priority DESC, p.rule_id ASC
	`

	rows, err := s.db.Query(query, sourceRuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query compiled policies for rule %d: %w", sourceRuleID, err)
	}
	defer rows.Close()

	var policies []*CompiledPolicy
	for rows.Next() {
		var cp CompiledPolicy
		var compilationTime string

		err := rows.Scan(
			&cp.RuleID,
			&cp.SrcIP,
			&cp.DstIP,
			&cp.SrcPort,
			&cp.DstPort,
			&cp.Protocol,
			&cp.Action,
			&cp.Priority,
			&cp.SourceRuleID,
			&cp.FromGroup,
			&cp.ToGroup,
			&cp.FromWorkloadID,
			&cp.ToWorkloadID,
			&compilationTime,
			&cp.CompilerVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan compiled policy: %w", err)
		}

		// Parse compilation time
		cp.CompilationTime, err = parseTime(compilationTime)
		if err != nil {
			log.Warnf("Failed to parse compilation time for rule %d: %v", cp.RuleID, err)
			continue
		}

		policies = append(policies, &cp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating compiled policies: %w", err)
	}

	log.Debugf("Loaded %d compiled policies for source rule %d", len(policies), sourceRuleID)
	return policies, nil
}

// GetPolicySource retrieves the source PolicyRule for a compiled policy
func (s *SQLiteStorage) GetPolicySource(compiledRuleID uint32) (*PolicyRule, error) {
	query := `
	SELECT pr.id, pr.name, pr.description, pr.from_group, pr.to_group, pr.ports, pr.action,
	       pr.priority, pr.enabled, pr.created_at, pr.updated_at
	FROM policy_rules pr
	INNER JOIN policy_compilation c ON pr.id = c.source_rule_id
	WHERE c.compiled_rule_id = ?
	`

	var r PolicyRule
	var portsJSON string
	var enabled int
	var createdAt, updatedAt sql.NullString

	err := s.db.QueryRow(query, compiledRuleID).Scan(
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
		return nil, fmt.Errorf("source rule not found for compiled policy: rule_id=%d", compiledRuleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source rule: %w", err)
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

// DeleteCompiledPoliciesForRule deletes all compiled policies for a source rule
func (s *SQLiteStorage) DeleteCompiledPoliciesForRule(sourceRuleID uint32) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all compiled rule IDs for this source rule
	query := `SELECT compiled_rule_id FROM policy_compilation WHERE source_rule_id = ?`
	rows, err := tx.Query(query, sourceRuleID)
	if err != nil {
		return fmt.Errorf("failed to query compiled rule IDs: %w", err)
	}
	defer rows.Close()

	var compiledRuleIDs []uint32
	for rows.Next() {
		var ruleID uint32
		if err := rows.Scan(&ruleID); err != nil {
			return fmt.Errorf("failed to scan rule ID: %w", err)
		}
		compiledRuleIDs = append(compiledRuleIDs, ruleID)
	}
	rows.Close()

	// Delete from policy_compilation (will cascade to policies via FK constraint)
	deleteCompilationQuery := `DELETE FROM policy_compilation WHERE source_rule_id = ?`
	result, err := tx.Exec(deleteCompilationQuery, sourceRuleID)
	if err != nil {
		return fmt.Errorf("failed to delete compilation metadata: %w", err)
	}

	// Also delete from policies table to ensure cleanup
	if len(compiledRuleIDs) > 0 {
		deletePoliciesQuery := `DELETE FROM policies WHERE rule_id = ?`
		for _, ruleID := range compiledRuleIDs {
			_, err := tx.Exec(deletePoliciesQuery, ruleID)
			if err != nil {
				log.Warnf("Failed to delete policy %d: %v", ruleID, err)
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	rows_affected, _ := result.RowsAffected()
	log.Infof("Deleted %d compiled policies for source rule %d", rows_affected, sourceRuleID)
	return nil
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

// parseTime parses a SQLite datetime string to time.Time
func parseTime(s string) (time.Time, error) {
	// Try multiple datetime formats
	formats := []string{
		"2006-01-02 15:04:05",           // SQLite default format
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,                 // "2006-01-02T15:04:05.999999999Z07:00"
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("failed to parse time %q: %w", s, lastErr)
}

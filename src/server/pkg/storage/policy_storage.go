package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
)

// PolicyStorage handles policy data persistence
type PolicyStorage struct {
	db *sql.DB
}

// NewPolicyStorage creates a new PolicyStorage
func NewPolicyStorage(db *sql.DB) *PolicyStorage {
	return &PolicyStorage{db: db}
}

// GetAllPolicies retrieves all policies with current version
func (s *PolicyStorage) GetAllPolicies(ctx context.Context) ([]*policypb.Policy, uint64, error) {
	// Get current version
	var version uint64
	err := s.db.QueryRowContext(ctx, "SELECT version FROM policy_version WHERE id = 1").Scan(&version)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get policy version: %w", err)
	}

	// Query all policies
	rows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority,
		       source_labels, dest_labels, description,
		       process_name, process_path, match_mode,
		       FLOOR(EXTRACT(EPOCH FROM created_at)*1000000000)::bigint as created_at,
		       FLOOR(EXTRACT(EPOCH FROM updated_at)*1000000000)::bigint as updated_at
		FROM policies
		ORDER BY priority DESC, rule_id
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	policies := []*policypb.Policy{}
	for rows.Next() {
		var policy policypb.Policy
		var sourceLabelsJSON, destLabelsJSON []byte
		var processName, processPath sql.NullString
		var matchMode sql.NullInt32

		err := rows.Scan(
			&policy.RuleId,
			&policy.SrcIp,
			&policy.DstIp,
			&policy.SrcPort,
			&policy.DstPort,
			&policy.Protocol,
			&policy.Action,
			&policy.Priority,
			&sourceLabelsJSON,
			&destLabelsJSON,
			&policy.Description,
			&processName,
			&processPath,
			&matchMode,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan policy: %w", err)
		}

		// Unmarshal labels
		policy.SourceLabels = make(map[string]string)
		policy.DestLabels = make(map[string]string)
		json.Unmarshal(sourceLabelsJSON, &policy.SourceLabels)
		json.Unmarshal(destLabelsJSON, &policy.DestLabels)

		// Set process fields if present
		if processName.Valid {
			policy.ProcessName = processName.String
		}
		if processPath.Valid {
			policy.ProcessPath = processPath.String
		}
		if matchMode.Valid {
			policy.MatchMode = policypb.ProcessMatchMode(matchMode.Int32)
		}

		policies = append(policies, &policy)
	}

	return policies, version, nil
}

// CreatePolicy creates a new policy
func (s *PolicyStorage) CreatePolicy(ctx context.Context, policy *policypb.Policy) error {
	sourceLabelsJSON, _ := json.Marshal(policy.SourceLabels)
	destLabelsJSON, _ := json.Marshal(policy.DestLabels)

	// Convert process fields to nullable types
	var processName, processPath sql.NullString
	var matchMode sql.NullInt32
	if policy.ProcessName != "" {
		processName = sql.NullString{String: policy.ProcessName, Valid: true}
	}
	if policy.ProcessPath != "" {
		processPath = sql.NullString{String: policy.ProcessPath, Valid: true}
	}
	if policy.MatchMode != 0 {
		matchMode = sql.NullInt32{Int32: int32(policy.MatchMode), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO policies (rule_id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority,
		                     source_labels, dest_labels, description, process_name, process_path, match_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, policy.RuleId, policy.SrcIp, policy.DstIp, policy.SrcPort, policy.DstPort,
		policy.Protocol, policy.Action, policy.Priority,
		sourceLabelsJSON, destLabelsJSON, policy.Description,
		processName, processPath, matchMode)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	// Increment version
	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// UpdatePolicy updates an existing policy
func (s *PolicyStorage) UpdatePolicy(ctx context.Context, policy *policypb.Policy) error {
	sourceLabelsJSON, _ := json.Marshal(policy.SourceLabels)
	destLabelsJSON, _ := json.Marshal(policy.DestLabels)

	// Convert process fields to nullable types
	var processName, processPath sql.NullString
	var matchMode sql.NullInt32
	if policy.ProcessName != "" {
		processName = sql.NullString{String: policy.ProcessName, Valid: true}
	}
	if policy.ProcessPath != "" {
		processPath = sql.NullString{String: policy.ProcessPath, Valid: true}
	}
	if policy.MatchMode != 0 {
		matchMode = sql.NullInt32{Int32: int32(policy.MatchMode), Valid: true}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE policies
		SET src_ip = $2, dst_ip = $3, src_port = $4, dst_port = $5,
		    protocol = $6, action = $7, priority = $8,
		    source_labels = $9, dest_labels = $10, description = $11,
		    process_name = $12, process_path = $13, match_mode = $14,
		    updated_at = CURRENT_TIMESTAMP
		WHERE rule_id = $1
	`, policy.RuleId, policy.SrcIp, policy.DstIp, policy.SrcPort, policy.DstPort,
		policy.Protocol, policy.Action, policy.Priority,
		sourceLabelsJSON, destLabelsJSON, policy.Description,
		processName, processPath, matchMode)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy not found")
	}

	// Increment version
	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// DeletePolicy deletes a policy by rule ID
func (s *PolicyStorage) DeletePolicy(ctx context.Context, ruleID uint32) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM policies WHERE rule_id = $1", ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy not found: %d", ruleID)
	}

	// Increment version
	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// incrementPolicyVersion increments the global policy version
func (s *PolicyStorage) incrementPolicyVersion(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE policy_version SET version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1")
	return err
}

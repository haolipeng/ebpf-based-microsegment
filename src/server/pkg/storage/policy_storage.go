package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

// GetPolicyUpdates retrieves incremental policy updates since the given version
func (s *PolicyStorage) GetPolicyUpdates(ctx context.Context, sinceVersion uint64) ([]*policypb.PolicyUpdate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT version, update_type, rule_id, policy_data, timestamp
		FROM policy_updates
		WHERE version > $1
		ORDER BY version ASC, timestamp ASC
	`, sinceVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy updates: %w", err)
	}
	defer rows.Close()

	var updates []*policypb.PolicyUpdate
	for rows.Next() {
		var (
			version    uint64
			updateType string
			ruleID     uint32
			policyJSON []byte
			timestamp  int64
		)

		if err := rows.Scan(&version, &updateType, &ruleID, &policyJSON, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan policy update: %w", err)
		}

		update := &policypb.PolicyUpdate{
			PolicyVersion: version,
			Timestamp:     timestamp,
		}

		// Map update type string to enum
		switch updateType {
		case "UPDATE_ADD":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_ADD
		case "UPDATE_MODIFY":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_MODIFY
		case "UPDATE_DELETE":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_DELETE
		default:
			return nil, fmt.Errorf("unknown update type: %s", updateType)
		}

		// Unmarshal policy data if present (not for DELETE)
		if policyJSON != nil {
			var policy policypb.Policy
			if err := json.Unmarshal(policyJSON, &policy); err != nil {
				return nil, fmt.Errorf("failed to unmarshal policy data: %w", err)
			}
			update.Policy = &policy
		} else if update.UpdateType != policypb.PolicyUpdateType_UPDATE_DELETE {
			return nil, fmt.Errorf("policy data is null for non-DELETE update")
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// RecordPolicyUpdate records a policy change in the updates log
func (s *PolicyStorage) RecordPolicyUpdate(ctx context.Context, updateType policypb.PolicyUpdateType, policy *policypb.Policy, newVersion uint64) error {
	var updateTypeStr string
	switch updateType {
	case policypb.PolicyUpdateType_UPDATE_ADD:
		updateTypeStr = "UPDATE_ADD"
	case policypb.PolicyUpdateType_UPDATE_MODIFY:
		updateTypeStr = "UPDATE_MODIFY"
	case policypb.PolicyUpdateType_UPDATE_DELETE:
		updateTypeStr = "UPDATE_DELETE"
	default:
		return fmt.Errorf("unknown update type: %v", updateType)
	}

	var policyJSON []byte
	var ruleID uint32

	if policy != nil {
		var err error
		policyJSON, err = json.Marshal(policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy: %w", err)
		}
		ruleID = policy.RuleId
	} else if updateType == policypb.PolicyUpdateType_UPDATE_DELETE {
		// For DELETE, policy can be nil but we need to get ruleID from somewhere
		// This should be passed separately
		return fmt.Errorf("ruleID required for DELETE operation")
	}

	timestamp := time.Now().UnixNano()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO policy_updates (version, update_type, rule_id, policy_data, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`, newVersion, updateTypeStr, ruleID, policyJSON, timestamp)

	if err != nil {
		return fmt.Errorf("failed to record policy update: %w", err)
	}

	return nil
}

// RecordPolicyDelete records a policy deletion in the updates log
func (s *PolicyStorage) RecordPolicyDelete(ctx context.Context, ruleID uint32, newVersion uint64) error {
	timestamp := time.Now().UnixNano()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO policy_updates (version, update_type, rule_id, policy_data, timestamp)
		VALUES ($1, $2, $3, NULL, $4)
	`, newVersion, "UPDATE_DELETE", ruleID, timestamp)

	if err != nil {
		return fmt.Errorf("failed to record policy delete: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current policy version
func (s *PolicyStorage) GetCurrentVersion(ctx context.Context) (uint64, error) {
	var version uint64
	err := s.db.QueryRowContext(ctx, "SELECT version FROM policy_version WHERE id = 1").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get policy version: %w", err)
	}
	return version, nil
}

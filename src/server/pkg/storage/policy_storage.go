package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
)

// PolicyStorage handles policy data persistence using Bun
type PolicyStorage struct {
	db *bun.DB
}

// NewPolicyStorage creates a new PolicyStorage
func NewPolicyStorage(db *sql.DB) *PolicyStorage {
	bunDB := bun.NewDB(db, pgdialect.New())
	return &PolicyStorage{db: bunDB}
}

// PolicyRow represents a row in the policies table
type PolicyRow struct {
	bun.BaseModel `bun:"table:policies,alias:p"`

	ID           int64             `bun:"id,pk,autoincrement"`
	RuleID       uint32            `bun:"rule_id,unique,notnull"`
	SrcIP        string            `bun:"src_ip"`
	DstIP        string            `bun:"dst_ip"`
	SrcPort      uint32            `bun:"src_port"`
	DstPort      uint32            `bun:"dst_port"`
	Protocol     int32             `bun:"protocol"`
	Action       int32             `bun:"action"`
	Priority     uint32            `bun:"priority"`
	SourceLabels map[string]string `bun:"source_labels,type:jsonb"`
	DestLabels   map[string]string `bun:"dest_labels,type:jsonb"`
	Description  string            `bun:"description"`
	ProcessName  *string           `bun:"process_name"`
	ProcessPath  *string           `bun:"process_path"`
	MatchMode    *int32            `bun:"match_mode"`
	CreatedAt    time.Time         `bun:"created_at,default:current_timestamp"`
	UpdatedAt    time.Time         `bun:"updated_at,default:current_timestamp"`
}

// PolicyVersionRow represents a row in the policy_version table
type PolicyVersionRow struct {
	bun.BaseModel `bun:"table:policy_version,alias:pv"`

	ID        int64     `bun:"id,pk"`
	Version   uint64    `bun:"version"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
}

// PolicyUpdateRow represents a row in the policy_updates table
type PolicyUpdateRow struct {
	bun.BaseModel `bun:"table:policy_updates,alias:pu"`

	ID         int64           `bun:"id,pk,autoincrement"`
	Version    uint64          `bun:"version"`
	UpdateType string          `bun:"update_type"`
	RuleID     uint32          `bun:"rule_id"`
	PolicyData json.RawMessage `bun:"policy_data,type:jsonb"`
	Timestamp  int64           `bun:"timestamp"`
}

// PolicyStatRow represents a row in the policy_stats table
type PolicyStatRow struct {
	bun.BaseModel `bun:"table:policy_stats,alias:ps"`

	ID            int64      `bun:"id,pk,autoincrement"`
	AgentID       string     `bun:"agent_id,notnull"`
	RuleID        uint32     `bun:"rule_id,notnull"`
	PacketCount   uint64     `bun:"packet_count"`
	ByteCount     uint64     `bun:"byte_count"`
	FlowCount     uint64     `bun:"flow_count"`
	HitCount      uint64     `bun:"hit_count"`
	LastMatchTime *time.Time `bun:"last_match_time"`
	ReportTime    time.Time  `bun:"report_time,default:current_timestamp"`
}

// GetAllPolicies retrieves all policies with current version
func (s *PolicyStorage) GetAllPolicies(ctx context.Context) ([]*policypb.Policy, uint64, error) {
	var versionRow PolicyVersionRow
	err := s.db.NewSelect().
		Model(&versionRow).
		Where("id = 1").
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get policy version: %w", err)
	}

	var rows []PolicyRow
	err = s.db.NewSelect().
		Model(&rows).
		Order("priority DESC", "rule_id ASC").
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query policies: %w", err)
	}

	policies := make([]*policypb.Policy, 0, len(rows))
	for _, row := range rows {
		policy := &policypb.Policy{
			RuleId:       row.RuleID,
			SrcIp:        row.SrcIP,
			DstIp:        row.DstIP,
			SrcPort:      row.SrcPort,
			DstPort:      row.DstPort,
			Protocol:     commonpb.Protocol(row.Protocol),
			Action:       commonpb.PolicyAction(row.Action),
			Priority:     row.Priority,
			SourceLabels: row.SourceLabels,
			DestLabels:   row.DestLabels,
			Description:  row.Description,
			CreatedAt:    row.CreatedAt.UnixNano(),
			UpdatedAt:    row.UpdatedAt.UnixNano(),
		}

		if row.ProcessName != nil {
			policy.ProcessName = *row.ProcessName
		}
		if row.ProcessPath != nil {
			policy.ProcessPath = *row.ProcessPath
		}
		if row.MatchMode != nil {
			policy.MatchMode = policypb.ProcessMatchMode(*row.MatchMode)
		}

		policies = append(policies, policy)
	}

	return policies, versionRow.Version, nil
}

// CreatePolicy creates a new policy
func (s *PolicyStorage) CreatePolicy(ctx context.Context, policy *policypb.Policy) error {
	row := &PolicyRow{
		RuleID:       policy.RuleId,
		SrcIP:        policy.SrcIp,
		DstIP:        policy.DstIp,
		SrcPort:      policy.SrcPort,
		DstPort:      policy.DstPort,
		Protocol:     int32(policy.Protocol),
		Action:       int32(policy.Action),
		Priority:     policy.Priority,
		SourceLabels: policy.SourceLabels,
		DestLabels:   policy.DestLabels,
		Description:  policy.Description,
	}

	if policy.ProcessName != "" {
		row.ProcessName = &policy.ProcessName
	}
	if policy.ProcessPath != "" {
		row.ProcessPath = &policy.ProcessPath
	}
	if policy.MatchMode != 0 {
		mode := int32(policy.MatchMode)
		row.MatchMode = &mode
	}

	_, err := s.db.NewInsert().
		Model(row).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// UpdatePolicy updates an existing policy
func (s *PolicyStorage) UpdatePolicy(ctx context.Context, policy *policypb.Policy) error {
	row := &PolicyRow{
		RuleID:       policy.RuleId,
		SrcIP:        policy.SrcIp,
		DstIP:        policy.DstIp,
		SrcPort:      policy.SrcPort,
		DstPort:      policy.DstPort,
		Protocol:     int32(policy.Protocol),
		Action:       int32(policy.Action),
		Priority:     policy.Priority,
		SourceLabels: policy.SourceLabels,
		DestLabels:   policy.DestLabels,
		Description:  policy.Description,
	}

	if policy.ProcessName != "" {
		row.ProcessName = &policy.ProcessName
	}
	if policy.ProcessPath != "" {
		row.ProcessPath = &policy.ProcessPath
	}
	if policy.MatchMode != 0 {
		mode := int32(policy.MatchMode)
		row.MatchMode = &mode
	}

	result, err := s.db.NewUpdate().
		Model(row).
		Set("src_ip = ?", row.SrcIP).
		Set("dst_ip = ?", row.DstIP).
		Set("src_port = ?", row.SrcPort).
		Set("dst_port = ?", row.DstPort).
		Set("protocol = ?", row.Protocol).
		Set("action = ?", row.Action).
		Set("priority = ?", row.Priority).
		Set("source_labels = ?", row.SourceLabels).
		Set("dest_labels = ?", row.DestLabels).
		Set("description = ?", row.Description).
		Set("process_name = ?", row.ProcessName).
		Set("process_path = ?", row.ProcessPath).
		Set("match_mode = ?", row.MatchMode).
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("rule_id = ?", policy.RuleId).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("policy not found")
	}

	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// DeletePolicy deletes a policy by rule ID
func (s *PolicyStorage) DeletePolicy(ctx context.Context, ruleID uint32) error {
	result, err := s.db.NewDelete().
		Model((*PolicyRow)(nil)).
		Where("rule_id = ?", ruleID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %d", ruleID)
	}

	if err := s.incrementPolicyVersion(ctx); err != nil {
		return err
	}

	return nil
}

// incrementPolicyVersion increments the global policy version
func (s *PolicyStorage) incrementPolicyVersion(ctx context.Context) error {
	_, err := s.db.NewUpdate().
		Model((*PolicyVersionRow)(nil)).
		Set("version = version + 1").
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("id = 1").
		Exec(ctx)
	return err
}

// GetPolicyUpdates retrieves incremental policy updates since the given version
func (s *PolicyStorage) GetPolicyUpdates(ctx context.Context, sinceVersion uint64) ([]*policypb.PolicyUpdate, error) {
	var rows []PolicyUpdateRow
	err := s.db.NewSelect().
		Model(&rows).
		Where("version > ?", sinceVersion).
		Order("version ASC", "timestamp ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy updates: %w", err)
	}

	updates := make([]*policypb.PolicyUpdate, 0, len(rows))
	for _, row := range rows {
		update := &policypb.PolicyUpdate{
			PolicyVersion: row.Version,
			Timestamp:     row.Timestamp,
		}

		switch row.UpdateType {
		case "UPDATE_ADD":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_ADD
		case "UPDATE_MODIFY":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_MODIFY
		case "UPDATE_DELETE":
			update.UpdateType = policypb.PolicyUpdateType_UPDATE_DELETE
		default:
			return nil, fmt.Errorf("unknown update type: %s", row.UpdateType)
		}

		if len(row.PolicyData) > 0 {
			var policy policypb.Policy
			if err := json.Unmarshal(row.PolicyData, &policy); err != nil {
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

	var policyJSON json.RawMessage
	var ruleID uint32

	if policy != nil {
		var err error
		policyJSON, err = json.Marshal(policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy: %w", err)
		}
		ruleID = policy.RuleId
	} else if updateType == policypb.PolicyUpdateType_UPDATE_DELETE {
		return fmt.Errorf("ruleID required for DELETE operation")
	}

	row := &PolicyUpdateRow{
		Version:    newVersion,
		UpdateType: updateTypeStr,
		RuleID:     ruleID,
		PolicyData: policyJSON,
		Timestamp:  time.Now().UnixNano(),
	}

	_, err := s.db.NewInsert().
		Model(row).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to record policy update: %w", err)
	}

	return nil
}

// RecordPolicyDelete records a policy deletion in the updates log
func (s *PolicyStorage) RecordPolicyDelete(ctx context.Context, ruleID uint32, newVersion uint64) error {
	row := &PolicyUpdateRow{
		Version:    newVersion,
		UpdateType: "UPDATE_DELETE",
		RuleID:     ruleID,
		PolicyData: nil,
		Timestamp:  time.Now().UnixNano(),
	}

	_, err := s.db.NewInsert().
		Model(row).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to record policy delete: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current policy version
func (s *PolicyStorage) GetCurrentVersion(ctx context.Context) (uint64, error) {
	var versionRow PolicyVersionRow
	err := s.db.NewSelect().
		Model(&versionRow).
		Where("id = 1").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get policy version: %w", err)
	}
	return versionRow.Version, nil
}

// SavePolicyStats saves policy enforcement statistics from an agent
func (s *PolicyStorage) SavePolicyStats(ctx context.Context, agentID string, stats []*policypb.PolicyStats) error {
	if len(stats) == 0 {
		return nil
	}

	rows := make([]*PolicyStatRow, 0, len(stats))
	for _, stat := range stats {
		var lastMatchTime *time.Time
		if stat.LastMatchTime > 0 {
			t := time.Unix(0, stat.LastMatchTime)
			lastMatchTime = &t
		}

		rows = append(rows, &PolicyStatRow{
			AgentID:       agentID,
			RuleID:        stat.RuleId,
			PacketCount:   stat.PacketCount,
			ByteCount:     stat.ByteCount,
			FlowCount:     stat.FlowCount,
			HitCount:      stat.HitCount,
			LastMatchTime: lastMatchTime,
		})
	}

	_, err := s.db.NewInsert().
		Model(&rows).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save policy stats: %w", err)
	}

	return nil
}

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"

	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	"github.com/sirupsen/logrus"
)

// FlowStorage handles flow data persistence
type FlowStorage struct {
	db *sql.DB
}

// NewFlowStorage creates a new FlowStorage
func NewFlowStorage(db *sql.DB) *FlowStorage {
	return &FlowStorage{db: db}
}

// BatchSaveFlowEvents saves a batch of flow events
func (s *FlowStorage) BatchSaveFlowEvents(ctx context.Context, events []*flowpb.FlowEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO flows (
			timestamp_ns, src_ip, dst_ip, src_port, dst_port, protocol,
			direction, packet_count, byte_count, policy_id, policy_action,
			state, agent_id, source_labels, dest_labels
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		// Convert IPs from uint32 to string
		srcIP := intToIP(event.SrcIp)
		dstIP := intToIP(event.DstIp)

		// Convert labels to JSONB
		sourceLabelsJSON, _ := json.Marshal(event.SourceLabels)
		destLabelsJSON, _ := json.Marshal(event.DestLabels)

		_, err = stmt.ExecContext(ctx,
			event.TimestampNs,
			srcIP,
			dstIP,
			event.SrcPort,
			event.DstPort,
			event.Protocol,
			event.Direction,
			event.PacketCount,
			event.ByteCount,
			event.PolicyId,
			event.PolicyAction,
			event.State,
			event.AgentId,
			sourceLabelsJSON,
			destLabelsJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert flow event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.Debugf("Saved %d flow events to database", len(events))
	return nil
}

// QueryFlows queries flows with filtering
func (s *FlowStorage) QueryFlows(ctx context.Context, query *flowpb.FlowQuery) ([]*flowpb.Flow, int64, error) {
	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if query.TimeRange != nil {
		where += fmt.Sprintf(" AND timestamp_ns >= $%d AND timestamp_ns < $%d", argIdx, argIdx+1)
		args = append(args, query.TimeRange.StartTime, query.TimeRange.EndTime)
		argIdx += 2
	}

	if query.AgentId != "" {
		where += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, query.AgentId)
		argIdx++
	}

	if query.Protocol != 0 {
		where += fmt.Sprintf(" AND protocol = $%d", argIdx)
		args = append(args, query.Protocol)
		argIdx++
	}

	// Count total
	var total int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM flows "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count flows: %w", err)
	}

	// Add limit and offset
	limit := query.Limit
	if limit == 0 {
		limit = 100
	}
	querySQL := fmt.Sprintf(`
		SELECT id, timestamp_ns, src_ip, dst_ip, src_port, dst_port, protocol,
		       direction, packet_count, byte_count, policy_id, policy_action,
		       state, agent_id, source_labels, dest_labels
		FROM flows %s
		ORDER BY timestamp_ns DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, query.Offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query flows: %w", err)
	}
	defer rows.Close()

	flows := []*flowpb.Flow{}
	for rows.Next() {
		var flow flowpb.Flow
		var srcIP, dstIP string
		var sourceLabelsJSON, destLabelsJSON []byte

		err := rows.Scan(
			&flow.Id,
			&flow.StartTime, // Using StartTime for timestamp_ns
			&srcIP,
			&dstIP,
			&flow.SrcPort,
			&flow.DstPort,
			&flow.Protocol,
			&flow.Direction,
			&flow.PacketCount,
			&flow.ByteCount,
			&flow.PolicyId,
			&flow.PolicyAction,
			&flow.State,
			&flow.AgentId,
			&sourceLabelsJSON,
			&destLabelsJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan flow: %w", err)
		}

		flow.SrcIp = srcIP
		flow.DstIp = dstIP

		// Unmarshal labels
		flow.SourceLabels = make(map[string]string)
		flow.DestLabels = make(map[string]string)
		json.Unmarshal(sourceLabelsJSON, &flow.SourceLabels)
		json.Unmarshal(destLabelsJSON, &flow.DestLabels)

		flows = append(flows, &flow)
	}

	return flows, total, nil
}

// intToIP converts uint32 IP to string
func intToIP(ip uint32) string {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)).String()
}

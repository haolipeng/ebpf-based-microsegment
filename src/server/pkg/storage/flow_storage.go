package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

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

// FlowSummary represents aggregated flow statistics
type FlowSummary struct {
	TotalFlows      int64   `json:"total_flows"`
	TotalPackets    int64   `json:"total_packets"`
	TotalBytes      int64   `json:"total_bytes"`
	UniqueSourceIPs int64   `json:"unique_source_ips"`
	UniqueDestIPs   int64   `json:"unique_dest_ips"`
	AvgDuration     float64 `json:"avg_duration_ms"`
	TopProtocols    []ProtocolStat `json:"top_protocols"`
}

// ProtocolStat represents protocol statistics
type ProtocolStat struct {
	Protocol string `json:"protocol"`
	Count    int64  `json:"count"`
	Bytes    int64  `json:"bytes"`
}

// FlowDependency represents a dependency between two workloads
type FlowDependency struct {
	SourceLabel string `json:"source_label"`
	DestLabel   string `json:"dest_label"`
	FlowCount   int64  `json:"flow_count"`
	TotalBytes  int64  `json:"total_bytes"`
	Protocols   []string `json:"protocols"`
}

// GetFlowSummary returns aggregated flow statistics for a time range
func (s *FlowStorage) GetFlowSummary(ctx context.Context, startTime, endTime time.Time) (*FlowSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_flows,
			COALESCE(SUM(packet_count), 0) as total_packets,
			COALESCE(SUM(byte_count), 0) as total_bytes,
			COUNT(DISTINCT src_ip) as unique_source_ips,
			COUNT(DISTINCT dst_ip) as unique_dest_ips,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(end_time, last_seen) - start_time)) * 1000), 0) as avg_duration_ms
		FROM flows
		WHERE timestamp_ns >= $1 AND timestamp_ns <= $2
	`

	summary := &FlowSummary{}
	err := s.db.QueryRowContext(
		ctx,
		query,
		startTime.UnixNano(),
		endTime.UnixNano(),
	).Scan(
		&summary.TotalFlows,
		&summary.TotalPackets,
		&summary.TotalBytes,
		&summary.UniqueSourceIPs,
		&summary.UniqueDestIPs,
		&summary.AvgDuration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get flow summary: %w", err)
	}

	// Get top protocols
	protocolQuery := `
		SELECT
			protocol,
			COUNT(*) as count,
			SUM(byte_count) as bytes
		FROM flows
		WHERE timestamp_ns >= $1 AND timestamp_ns <= $2
		GROUP BY protocol
		ORDER BY count DESC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, protocolQuery, startTime.UnixNano(), endTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol stats: %w", err)
	}
	defer rows.Close()

	summary.TopProtocols = []ProtocolStat{}
	for rows.Next() {
		var stat ProtocolStat
		if err := rows.Scan(&stat.Protocol, &stat.Count, &stat.Bytes); err != nil {
			continue
		}
		summary.TopProtocols = append(summary.TopProtocols, stat)
	}

	return summary, nil
}

// GetFlowDependencies returns application dependencies based on label-based flow aggregation
func (s *FlowStorage) GetFlowDependencies(ctx context.Context, startTime, endTime time.Time, groupBy string) ([]*FlowDependency, error) {
	// Query dependencies by aggregating flows based on labels
	// group_by can be "app", "env", "tier", etc.

	query := `
		SELECT
			COALESCE(source_labels->>'` + groupBy + `', 'unknown') as source_label,
			COALESCE(dest_labels->>'` + groupBy + `', 'unknown') as dest_label,
			COUNT(*) as flow_count,
			SUM(byte_count) as total_bytes,
			array_agg(DISTINCT protocol) as protocols
		FROM flows
		WHERE timestamp_ns >= $1 AND timestamp_ns <= $2
		  AND source_labels ? $3
		  AND dest_labels ? $3
		GROUP BY source_label, dest_label
		ORDER BY flow_count DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query, startTime.UnixNano(), endTime.UnixNano(), groupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow dependencies: %w", err)
	}
	defer rows.Close()

	dependencies := []*FlowDependency{}
	for rows.Next() {
		var dep FlowDependency
		var protocolsJSON []byte

		err := rows.Scan(
			&dep.SourceLabel,
			&dep.DestLabel,
			&dep.FlowCount,
			&dep.TotalBytes,
			&protocolsJSON,
		)
		if err != nil {
			logrus.Warnf("Failed to scan dependency: %v", err)
			continue
		}

		// Parse protocols array from PostgreSQL
		json.Unmarshal(protocolsJSON, &dep.Protocols)

		dependencies = append(dependencies, &dep)
	}

	return dependencies, nil
}

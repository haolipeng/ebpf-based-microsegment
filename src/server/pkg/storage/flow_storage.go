package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/netutil"
)

// FlowStorage handles flow data persistence using Bun
type FlowStorage struct {
	db *bun.DB
}

// NewFlowStorage creates a new FlowStorage
func NewFlowStorage(db *sql.DB) *FlowStorage {
	bunDB := bun.NewDB(db, pgdialect.New())
	return &FlowStorage{db: bunDB}
}

// FlowSummary represents aggregated flow statistics
type FlowSummary struct {
	TotalFlows      int64          `json:"total_flows"`
	ActiveFlows     int64          `json:"active_flows"`
	ClosedFlows     int64          `json:"closed_flows"`
	TotalPackets    int64          `json:"total_packets"`
	TotalBytes      int64          `json:"total_bytes"`
	AllowedFlows    int64          `json:"allowed_flows"`
	DeniedFlows     int64          `json:"denied_flows"`
	UniqueSourceIPs int64          `json:"unique_source_ips"`
	UniqueDestIPs   int64          `json:"unique_dest_ips"`
	AvgDuration     float64        `json:"avg_duration_ms"`
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
	SourceLabel string   `json:"source_label"`
	DestLabel   string   `json:"dest_label"`
	FlowCount   int64    `json:"flow_count"`
	TotalBytes  int64    `json:"total_bytes"`
	Protocols   []string `json:"protocols"`
}

// FlowRow represents a row in the flows table
type FlowRow struct {
	bun.BaseModel `bun:"table:flows,alias:f"`

	ID           int64             `bun:"id,pk,autoincrement"`
	AgentID      string            `bun:"agent_id"`
	SrcIP        string            `bun:"src_ip"`
	DstIP        string            `bun:"dst_ip"`
	SrcPort      int32             `bun:"src_port"`
	DstPort      int32             `bun:"dst_port"`
	Protocol     int32             `bun:"protocol"`
	Direction    int32             `bun:"direction"`
	PacketCount  int64             `bun:"packet_count"`
	ByteCount    int64             `bun:"byte_count"`
	PolicyID     *int32            `bun:"policy_id"`
	PolicyAction *int32            `bun:"policy_action"`
	State        *int32            `bun:"state"`
	TimestampNS  int64             `bun:"timestamp_ns"`
	StartTime    *time.Time        `bun:"start_time"`
	EndTime      *time.Time        `bun:"end_time"`
	LastSeen     *time.Time        `bun:"last_seen"`
	SourceLabels map[string]string `bun:"source_labels,type:jsonb"`
	DestLabels   map[string]string `bun:"dest_labels,type:jsonb"`
	CreatedAt    time.Time         `bun:"created_at,default:current_timestamp"`

	// Process info fields
	SrcPID         *uint32 `bun:"src_pid"`
	SrcPPID        *uint32 `bun:"src_ppid"`
	SrcUID         *uint32 `bun:"src_uid"`
	SrcGID         *uint32 `bun:"src_gid"`
	SrcComm        *string `bun:"src_comm"`
	SrcExePath     *string `bun:"src_exe_path"`
	SrcCmdline     *string `bun:"src_cmdline"`
	SrcContainerID *string `bun:"src_container_id"`
	DstPID         *uint32 `bun:"dst_pid"`
	DstComm        *string `bun:"dst_comm"`
}

// BatchSaveFlowEvents saves a batch of flow events
func (s *FlowStorage) BatchSaveFlowEvents(ctx context.Context, events []*flowpb.FlowEvent) error {
	if len(events) == 0 {
		return nil
	}

	rows := make([]*FlowRow, 0, len(events))
	for _, event := range events {
		rows = append(rows, s.flowEventToRow(event))
	}

	_, err := s.db.NewInsert().
		Model(&rows).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert flow events: %w", err)
	}

	logrus.Debugf("Saved %d flow events to database", len(events))
	return nil
}

// QueryFlows queries flows with filtering
func (s *FlowStorage) QueryFlows(ctx context.Context, query *flowpb.FlowQuery) ([]*flowpb.Flow, int64, error) {
	baseQuery := s.db.NewSelect().Model((*FlowRow)(nil))

	if query.TimeRange != nil {
		baseQuery = baseQuery.Where("timestamp_ns >= ? AND timestamp_ns < ?", query.TimeRange.StartTime, query.TimeRange.EndTime)
	}
	if query.AgentId != "" {
		baseQuery = baseQuery.Where("agent_id = ?", query.AgentId)
	}
	if query.Protocol != 0 {
		baseQuery = baseQuery.Where("protocol = ?", query.Protocol)
	}

	// JSONB label filtering using @> (contains) operator
	if len(query.SourceLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.SourceLabels)
		baseQuery = baseQuery.Where("source_labels @> ?", string(labelsJSON))
	}
	if len(query.DestLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.DestLabels)
		baseQuery = baseQuery.Where("dest_labels @> ?", string(labelsJSON))
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count flows: %w", err)
	}

	limit := query.Limit
	if limit == 0 {
		limit = 100
	}

	var rows []FlowRow
	err = s.db.NewSelect().
		Model(&rows).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if query.TimeRange != nil {
				q = q.Where("timestamp_ns >= ? AND timestamp_ns < ?", query.TimeRange.StartTime, query.TimeRange.EndTime)
			}
			if query.AgentId != "" {
				q = q.Where("agent_id = ?", query.AgentId)
			}
			if query.Protocol != 0 {
				q = q.Where("protocol = ?", query.Protocol)
			}
			if len(query.SourceLabels) > 0 {
				labelsJSON, _ := json.Marshal(query.SourceLabels)
				q = q.Where("source_labels @> ?", string(labelsJSON))
			}
			if len(query.DestLabels) > 0 {
				labelsJSON, _ := json.Marshal(query.DestLabels)
				q = q.Where("dest_labels @> ?", string(labelsJSON))
			}
			return q
		}).
		Order("timestamp_ns DESC").
		Limit(int(limit)).
		Offset(int(query.Offset)).
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query flows: %w", err)
	}

	flows := make([]*flowpb.Flow, 0, len(rows))
	for i := range rows {
		flows = append(flows, s.flowRowToProto(&rows[i]))
	}

	return flows, int64(total), nil
}

// GetFlowSummary returns aggregated flow statistics for a time range
func (s *FlowStorage) GetFlowSummary(ctx context.Context, startTime, endTime time.Time) (*FlowSummary, error) {
	var row struct {
		TotalFlows      int64   `bun:"total_flows"`
		ActiveFlows     int64   `bun:"active_flows"`
		ClosedFlows     int64   `bun:"closed_flows"`
		TotalPackets    int64   `bun:"total_packets"`
		TotalBytes      int64   `bun:"total_bytes"`
		AllowedFlows    int64   `bun:"allowed_flows"`
		DeniedFlows     int64   `bun:"denied_flows"`
		UniqueSourceIPs int64   `bun:"unique_source_ips"`
		UniqueDestIPs   int64   `bun:"unique_dest_ips"`
		AvgDuration     float64 `bun:"avg_duration_ms"`
	}

	err := s.db.NewSelect().
		Model((*FlowRow)(nil)).
		ColumnExpr(`
			COUNT(*) as total_flows,
			COALESCE(SUM(CASE WHEN state = 1 THEN 1 ELSE 0 END), 0) as active_flows,
			COALESCE(SUM(CASE WHEN state = 2 THEN 1 ELSE 0 END), 0) as closed_flows,
			COALESCE(SUM(packet_count), 0) as total_packets,
			COALESCE(SUM(byte_count), 0) as total_bytes,
			COALESCE(SUM(CASE WHEN policy_action = 1 THEN 1 ELSE 0 END), 0) as allowed_flows,
			COALESCE(SUM(CASE WHEN policy_action = 2 THEN 1 ELSE 0 END), 0) as denied_flows,
			COUNT(DISTINCT src_ip) as unique_source_ips,
			COUNT(DISTINCT dst_ip) as unique_dest_ips,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(end_time, last_seen) - start_time)) * 1000), 0) as avg_duration_ms
		`).
		Where("timestamp_ns >= ? AND timestamp_ns <= ?", startTime.UnixNano(), endTime.UnixNano()).
		Scan(ctx, &row)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow summary: %w", err)
	}

	summary := &FlowSummary{
		TotalFlows:      row.TotalFlows,
		ActiveFlows:     row.ActiveFlows,
		ClosedFlows:     row.ClosedFlows,
		TotalPackets:    row.TotalPackets,
		TotalBytes:      row.TotalBytes,
		AllowedFlows:    row.AllowedFlows,
		DeniedFlows:     row.DeniedFlows,
		UniqueSourceIPs: row.UniqueSourceIPs,
		UniqueDestIPs:   row.UniqueDestIPs,
		AvgDuration:     row.AvgDuration,
		TopProtocols:    []ProtocolStat{},
	}

	var protoRows []struct {
		Protocol string `bun:"protocol"`
		Count    int64  `bun:"count"`
		Bytes    int64  `bun:"bytes"`
	}
	err = s.db.NewSelect().
		Model((*FlowRow)(nil)).
		ColumnExpr("protocol::text as protocol, COUNT(*) as count, COALESCE(SUM(byte_count), 0) as bytes").
		Where("timestamp_ns >= ? AND timestamp_ns <= ?", startTime.UnixNano(), endTime.UnixNano()).
		Group("protocol").
		Order("count DESC").
		Limit(5).
		Scan(ctx, &protoRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol stats: %w", err)
	}

	for _, row := range protoRows {
		summary.TopProtocols = append(summary.TopProtocols, ProtocolStat{
			Protocol: row.Protocol,
			Count:    row.Count,
			Bytes:    row.Bytes,
		})
	}

	return summary, nil
}

// GetFlowDependencies returns application dependencies based on label-based flow aggregation
func (s *FlowStorage) GetFlowDependencies(ctx context.Context, startTime, endTime time.Time, groupBy string) ([]*FlowDependency, error) {
	if groupBy == "" {
		groupBy = "app"
	}

	if !validLabelKey(groupBy) {
		return nil, fmt.Errorf("invalid groupBy value: %s", groupBy)
	}

	sourceExpr := fmt.Sprintf("COALESCE(source_labels->>'%s', 'unknown')", groupBy)
	destExpr := fmt.Sprintf("COALESCE(dest_labels->>'%s', 'unknown')", groupBy)

	type depRow struct {
		SourceLabel string          `bun:"source_label"`
		DestLabel   string          `bun:"dest_label"`
		FlowCount   int64           `bun:"flow_count"`
		TotalBytes  int64           `bun:"total_bytes"`
		Protocols   json.RawMessage `bun:"protocols"`
	}

	var rows []depRow
	selectClause := fmt.Sprintf(`%s AS source_label, %s AS dest_label, COUNT(*) AS flow_count,
		COALESCE(SUM(byte_count), 0) AS total_bytes,
		array_to_json(array_agg(DISTINCT protocol::text)) AS protocols`, sourceExpr, destExpr)

	err := s.db.NewSelect().
		TableExpr("flows").
		ColumnExpr(selectClause).
		Where("timestamp_ns >= ? AND timestamp_ns <= ?", startTime.UnixNano(), endTime.UnixNano()).
		Where("source_labels ? ?", groupBy).
		Where("dest_labels ? ?", groupBy).
		Group("source_label, dest_label").
		Order("flow_count DESC").
		Limit(100).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow dependencies: %w", err)
	}

	dependencies := make([]*FlowDependency, 0, len(rows))
	for _, row := range rows {
		var protocols []string
		if len(row.Protocols) > 0 {
			if err := json.Unmarshal(row.Protocols, &protocols); err != nil {
				logrus.Warnf("failed to decode protocol array: %v", err)
			}
		}

		dependencies = append(dependencies, &FlowDependency{
			SourceLabel: row.SourceLabel,
			DestLabel:   row.DestLabel,
			FlowCount:   row.FlowCount,
			TotalBytes:  row.TotalBytes,
			Protocols:   protocols,
		})
	}

	return dependencies, nil
}

// flowEventToRow converts a FlowEvent to FlowRow
func (s *FlowStorage) flowEventToRow(event *flowpb.FlowEvent) *FlowRow {
	row := &FlowRow{
		TimestampNS:  int64(event.TimestampNs),
		SrcIP:        netutil.Uint32ToString(event.SrcIp),
		DstIP:        netutil.Uint32ToString(event.DstIp),
		SrcPort:      int32(event.SrcPort),
		DstPort:      int32(event.DstPort),
		Protocol:     int32(event.Protocol),
		Direction:    int32(event.Direction),
		PacketCount:  int64(event.PacketCount),
		ByteCount:    int64(event.ByteCount),
		AgentID:      event.AgentId,
		SourceLabels: event.SourceLabels,
		DestLabels:   event.DestLabels,
	}

	if event.PolicyId != 0 {
		val := int32(event.PolicyId)
		row.PolicyID = &val
	}

	if event.PolicyAction != 0 {
		val := int32(event.PolicyAction)
		row.PolicyAction = &val
	}

	if event.State != 0 {
		val := int32(event.State)
		row.State = &val
	}

	return row
}

// flowRowToProto converts FlowRow to Flow proto
func (s *FlowStorage) flowRowToProto(row *FlowRow) *flowpb.Flow {
	flow := &flowpb.Flow{
		Id:           uint64(row.ID),
		SrcIp:        row.SrcIP,
		DstIp:        row.DstIP,
		SrcPort:      uint32(row.SrcPort),
		DstPort:      uint32(row.DstPort),
		Protocol:     commonpb.Protocol(row.Protocol),
		Direction:    commonpb.FlowDirection(row.Direction),
		PacketCount:  uint64(row.PacketCount),
		ByteCount:    uint64(row.ByteCount),
		AgentId:      row.AgentID,
		SourceLabels: row.SourceLabels,
		DestLabels:   row.DestLabels,
	}

	if row.PolicyID != nil {
		flow.PolicyId = uint32(*row.PolicyID)
	}
	if row.PolicyAction != nil {
		flow.PolicyAction = commonpb.PolicyAction(*row.PolicyAction)
	}
	if row.State != nil {
		flow.State = commonpb.FlowState(*row.State)
	}

	if row.StartTime != nil {
		flow.StartTime = row.StartTime.UnixNano()
	} else {
		flow.StartTime = row.TimestampNS
	}

	if row.EndTime != nil {
		flow.EndTime = row.EndTime.UnixNano()
	}

	return flow
}

var labelKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validLabelKey(key string) bool {
	return labelKeyPattern.MatchString(key)
}

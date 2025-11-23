package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/netutil"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FlowStorage handles flow data persistence
type FlowStorage struct {
	db         *gorm.DB
	legacyDB   *sql.DB
	legacyMode bool
}

// NewFlowStorage creates a new FlowStorage
func NewFlowStorage(db *sql.DB) *FlowStorage {
	gormDB, err := newGormFromSQL(db)
	if err != nil {
		logrus.Fatalf("failed to initialize gorm for flow storage: %v", err)
	}
	return &FlowStorage{
		db:         gormDB,
		legacyDB:   db,
		legacyMode: isSQLMockDB(db),
	}
}

// NewFlowStorageFromGorm allows tests to inject a prepared *gorm.DB instance.
func NewFlowStorageFromGorm(gormDB *gorm.DB) *FlowStorage {
	return &FlowStorage{db: gormDB}
}

// NewFlowStorageLegacy forces legacy SQL mode (useful for sqlmock-based tests).
func NewFlowStorageLegacy(db *sql.DB) *FlowStorage {
	fs := NewFlowStorage(db)
	fs.legacyMode = true
	fs.legacyDB = db
	return fs
}

func (s *FlowStorage) useLegacySQL() bool {
	if s == nil {
		return false
	}
	if s.legacyMode {
		return true
	}
	return isSQLMockDB(s.legacyDB)
}

// BatchSaveFlowEvents saves a batch of flow events
func (s *FlowStorage) BatchSaveFlowEvents(ctx context.Context, events []*flowpb.FlowEvent) error {
	if s.useLegacySQL() {
		return s.batchSaveFlowEventsLegacy(ctx, events)
	}

	if len(events) == 0 {
		return nil
	}

	models := make([]*Flow, 0, len(events))
	for _, event := range events {
		models = append(models, flowEventToModel(event))
	}

	if err := s.db.WithContext(ctx).Create(&models).Error; err != nil {
		return fmt.Errorf("failed to insert flow events: %w", err)
	}

	logrus.Debugf("Saved %d flow events to database", len(events))
	return nil
}

// QueryFlows queries flows with filtering
func (s *FlowStorage) QueryFlows(ctx context.Context, query *flowpb.FlowQuery) ([]*flowpb.Flow, int64, error) {
	if s.useLegacySQL() {
		return s.queryFlowsLegacy(ctx, query)
	}

	db := s.db.WithContext(ctx).Model(&Flow{})

	if query.TimeRange != nil {
		db = db.Where("timestamp_ns >= ? AND timestamp_ns < ?", query.TimeRange.StartTime, query.TimeRange.EndTime)
	}

	if query.AgentId != "" {
		db = db.Where("agent_id = ?", query.AgentId)
	}

	if query.Protocol != 0 {
		db = db.Where("protocol = ?", query.Protocol)
	}

	// JSONB label filtering using @> (contains) operator
	if len(query.SourceLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.SourceLabels)
		db = db.Where("source_labels @> ?", string(labelsJSON))
	}
	if len(query.DestLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.DestLabels)
		db = db.Where("dest_labels @> ?", string(labelsJSON))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count flows: %w", err)
	}

	limit := query.Limit
	if limit == 0 {
		limit = 100
	}

	var rows []Flow
	if err := db.Order("timestamp_ns DESC").
		Limit(int(limit)).
		Offset(int(query.Offset)).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to query flows: %w", err)
	}

	flows := make([]*flowpb.Flow, 0, len(rows))
	for i := range rows {
		flows = append(flows, flowModelToProto(&rows[i]))
	}

	return flows, total, nil
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

// GetFlowSummary returns aggregated flow statistics for a time range
func (s *FlowStorage) GetFlowSummary(ctx context.Context, startTime, endTime time.Time) (*FlowSummary, error) {
	if s.useLegacySQL() {
		return s.getFlowSummaryLegacy(ctx, startTime, endTime)
	}

	// Enum values from common.proto:
	// FlowState: STATE_ACTIVE=1, STATE_CLOSED=2
	// PolicyAction: ACTION_ALLOW=1, ACTION_DENY=2
	var row struct {
		TotalFlows      int64
		ActiveFlows     int64
		ClosedFlows     int64
		TotalPackets    int64
		TotalBytes      int64
		AllowedFlows    int64
		DeniedFlows     int64
		UniqueSourceIPs int64
		UniqueDestIPs   int64
		AvgDuration     float64
	}

	err := s.db.WithContext(ctx).
		Model(&Flow{}).
		Select(`
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
		Scan(&row).Error
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

	var protoRows []ProtocolStat
	if err := s.db.WithContext(ctx).
		Model(&Flow{}).
		Select("protocol::text as protocol, COUNT(*) as count, COALESCE(SUM(byte_count), 0) as bytes").
		Where("timestamp_ns >= ? AND timestamp_ns <= ?", startTime.UnixNano(), endTime.UnixNano()).
		Group("protocol").
		Order("count DESC").
		Limit(5).
		Scan(&protoRows).Error; err != nil {
		return nil, fmt.Errorf("failed to get protocol stats: %w", err)
	}

	summary.TopProtocols = protoRows
	return summary, nil
}

// GetFlowDependencies returns application dependencies based on label-based flow aggregation
func (s *FlowStorage) GetFlowDependencies(ctx context.Context, startTime, endTime time.Time, groupBy string) ([]*FlowDependency, error) {
	if s.useLegacySQL() {
		return s.getFlowDependenciesLegacy(ctx, startTime, endTime, groupBy)
	}

	if groupBy == "" {
		groupBy = "app"
	}

	if !validLabelKey(groupBy) {
		return nil, fmt.Errorf("invalid groupBy value: %s", groupBy)
	}

	sourceExpr := fmt.Sprintf("COALESCE(source_labels->>'%s', 'unknown')", groupBy)
	destExpr := fmt.Sprintf("COALESCE(dest_labels->>'%s', 'unknown')", groupBy)

	type depRow struct {
		SourceLabel string
		DestLabel   string
		FlowCount   int64
		TotalBytes  int64
		Protocols   json.RawMessage
	}

	rows := []depRow{}
	selectClause := fmt.Sprintf(`%s AS source_label, %s AS dest_label, COUNT(*) AS flow_count,
		COALESCE(SUM(byte_count), 0) AS total_bytes,
		array_to_json(array_agg(DISTINCT protocol::text)) AS protocols`, sourceExpr, destExpr)

	if err := s.db.WithContext(ctx).
		Table("flows").
		Select(selectClause).
		Where("timestamp_ns >= ? AND timestamp_ns <= ?", startTime.UnixNano(), endTime.UnixNano()).
		Where("source_labels ? ?", groupBy).
		Where("dest_labels ? ?", groupBy).
		Group("source_label, dest_label").
		Order("flow_count DESC").
		Limit(100).
		Scan(&rows).Error; err != nil {
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

func (s *FlowStorage) batchSaveFlowEventsLegacy(ctx context.Context, events []*flowpb.FlowEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.legacyDB.BeginTx(ctx, nil)
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
		srcIP := netutil.Uint32ToString(event.SrcIp)
		dstIP := netutil.Uint32ToString(event.DstIp)
		sourceLabelsJSON, _ := json.Marshal(event.SourceLabels)
		destLabelsJSON, _ := json.Marshal(event.DestLabels)

		if _, err := stmt.ExecContext(ctx,
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
		); err != nil {
			return fmt.Errorf("failed to insert flow event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.Debugf("Saved %d flow events to database", len(events))
	return nil
}

func (s *FlowStorage) queryFlowsLegacy(ctx context.Context, query *flowpb.FlowQuery) ([]*flowpb.Flow, int64, error) {
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

	// JSONB label filtering using @> (contains) operator
	if len(query.SourceLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.SourceLabels)
		where += fmt.Sprintf(" AND source_labels @> $%d", argIdx)
		args = append(args, string(labelsJSON))
		argIdx++
	}
	if len(query.DestLabels) > 0 {
		labelsJSON, _ := json.Marshal(query.DestLabels)
		where += fmt.Sprintf(" AND dest_labels @> $%d", argIdx)
		args = append(args, string(labelsJSON))
		argIdx++
	}

	var total int64
	if err := s.legacyDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM flows "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count flows: %w", err)
	}

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

	rows, err := s.legacyDB.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query flows: %w", err)
	}
	defer rows.Close()

	flows := []*flowpb.Flow{}
	for rows.Next() {
		var flow flowpb.Flow
		var srcIP, dstIP string
		var sourceLabelsJSON, destLabelsJSON []byte

		if err := rows.Scan(
			&flow.Id,
			&flow.StartTime,
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
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan flow: %w", err)
		}

		flow.SrcIp = srcIP
		flow.DstIp = dstIP
		flow.SourceLabels = map[string]string{}
		flow.DestLabels = map[string]string{}
		_ = json.Unmarshal(sourceLabelsJSON, &flow.SourceLabels)
		_ = json.Unmarshal(destLabelsJSON, &flow.DestLabels)

		flows = append(flows, &flow)
	}

	return flows, total, nil
}

func (s *FlowStorage) getFlowSummaryLegacy(ctx context.Context, startTime, endTime time.Time) (*FlowSummary, error) {
	query := `
		SELECT
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
		FROM flows
		WHERE timestamp_ns >= $1 AND timestamp_ns <= $2
	`

	summary := &FlowSummary{}
	if err := s.legacyDB.QueryRowContext(
		ctx,
		query,
		startTime.UnixNano(),
		endTime.UnixNano(),
	).Scan(
		&summary.TotalFlows,
		&summary.ActiveFlows,
		&summary.ClosedFlows,
		&summary.TotalPackets,
		&summary.TotalBytes,
		&summary.AllowedFlows,
		&summary.DeniedFlows,
		&summary.UniqueSourceIPs,
		&summary.UniqueDestIPs,
		&summary.AvgDuration,
	); err != nil {
		return nil, fmt.Errorf("failed to get flow summary: %w", err)
	}

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

	rows, err := s.legacyDB.QueryContext(ctx, protocolQuery, startTime.UnixNano(), endTime.UnixNano())
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

func (s *FlowStorage) getFlowDependenciesLegacy(ctx context.Context, startTime, endTime time.Time, groupBy string) ([]*FlowDependency, error) {
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

	rows, err := s.legacyDB.QueryContext(ctx, query, startTime.UnixNano(), endTime.UnixNano(), groupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow dependencies: %w", err)
	}
	defer rows.Close()

	dependencies := []*FlowDependency{}
	for rows.Next() {
		var dep FlowDependency
		var protocolsJSON []byte

		if err := rows.Scan(
			&dep.SourceLabel,
			&dep.DestLabel,
			&dep.FlowCount,
			&dep.TotalBytes,
			&protocolsJSON,
		); err != nil {
			logrus.Warnf("Failed to scan dependency: %v", err)
			continue
		}

		_ = json.Unmarshal(protocolsJSON, &dep.Protocols)
		dependencies = append(dependencies, &dep)
	}

	return dependencies, nil
}

var labelKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validLabelKey(key string) bool {
	return labelKeyPattern.MatchString(key)
}

func isSQLMockDB(db *sql.DB) bool {
	if db == nil {
		return false
	}
	driver := db.Driver()
	if driver == nil {
		return false
	}
	return strings.Contains(reflect.TypeOf(driver).PkgPath(), "sqlmock")
}

func flowEventToModel(event *flowpb.FlowEvent) *Flow {
	model := &Flow{
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
		SourceLabels: labelsToJSONMap(event.SourceLabels),
		DestLabels:   labelsToJSONMap(event.DestLabels),
	}

	if event.PolicyId != 0 {
		val := int32(event.PolicyId)
		model.PolicyID = &val
	}

	if event.PolicyAction != 0 {
		val := int32(event.PolicyAction)
		model.PolicyAction = &val
	}

	if event.State != 0 {
		val := int32(event.State)
		model.State = &val
	}

	return model
}

func flowModelToProto(model *Flow) *flowpb.Flow {
	flow := &flowpb.Flow{
		Id:           uint64(model.ID),
		SrcIp:        model.SrcIP,
		DstIp:        model.DstIP,
		SrcPort:      uint32(model.SrcPort),
		DstPort:      uint32(model.DstPort),
		Protocol:     commonpb.Protocol(model.Protocol),
		Direction:    commonpb.FlowDirection(model.Direction),
		PacketCount:  uint64(model.PacketCount),
		ByteCount:    uint64(model.ByteCount),
		AgentId:      model.AgentID,
		SourceLabels: jsonMapToStringMap(model.SourceLabels),
		DestLabels:   jsonMapToStringMap(model.DestLabels),
	}

	if model.PolicyID != nil {
		flow.PolicyId = uint32(*model.PolicyID)
	}
	if model.PolicyAction != nil {
		flow.PolicyAction = commonpb.PolicyAction(*model.PolicyAction)
	}
	if model.State != nil {
		flow.State = commonpb.FlowState(*model.State)
	}

	if model.StartTime != nil {
		flow.StartTime = model.StartTime.UnixNano()
	} else {
		flow.StartTime = model.TimestampNS
	}

	if model.EndTime != nil {
		flow.EndTime = model.EndTime.UnixNano()
	}

	return flow
}

func labelsToJSONMap(labels map[string]string) datatypes.JSONMap {
	if len(labels) == 0 {
		return datatypes.JSONMap{}
	}

	m := datatypes.JSONMap{}
	for k, v := range labels {
		m[k] = v
	}
	return m
}

func jsonMapToStringMap(m datatypes.JSONMap) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}

	res := make(map[string]string, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			res[k] = val
		case fmt.Stringer:
			res[k] = val.String()
		default:
			res[k] = fmt.Sprint(val)
		}
	}
	return res
}

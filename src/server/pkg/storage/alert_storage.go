// input: security alert events
// output: alert list queries, alert creation results
// pos: storage - PostgreSQL storage layer for security alerts

package storage

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	alertpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/alert"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
)

// AlertStorage handles security alert data persistence using Bun
type AlertStorage struct {
	db *bun.DB
}

// NewAlertStorage creates a new AlertStorage
func NewAlertStorage(db *sql.DB) *AlertStorage {
	bunDB := bun.NewDB(db, pgdialect.New())
	return &AlertStorage{db: bunDB}
}

// AlertQueryOptions contains query parameters for alerts
type AlertQueryOptions struct {
	Level       *int32
	Type        *int32
	ProcessPath *string
	StartTime   *int64
	EndTime     *int64
	Page        int
	PageSize    int
}

// AlertStats contains aggregated alert statistics
type AlertStats struct {
	ByLevel      map[string]int64
	ByType       map[string]int64
	TopProcesses []ProcessAlertCount
	Timeline     []TimelineBucket
}

// ProcessAlertCount represents alert count for a specific process
type ProcessAlertCount struct {
	ExePath string
	Count   int64
}

// TimelineBucket represents alert count in a time bucket
type TimelineBucket struct {
	Timestamp int64
	Count     int64
}

// AlertRow represents a row in the security_alerts table
type AlertRow struct {
	bun.BaseModel `bun:"table:security_alerts,alias:sa"`

	ID          int64             `bun:"id,pk,autoincrement"`
	AlertID     string            `bun:"alert_id,unique,notnull"`
	Level       int32             `bun:"level,notnull"`
	Type        int32             `bun:"type,notnull"`
	PID         *uint32           `bun:"pid"`
	PPID        *uint32           `bun:"ppid"`
	UID         *uint32           `bun:"uid"`
	GID         *uint32           `bun:"gid"`
	Comm        *string           `bun:"comm"`
	ExePath     *string           `bun:"exe_path"`
	ContainerID *string           `bun:"container_id"`
	FlowID      *int64            `bun:"flow_id"`
	SrcIP       *string           `bun:"src_ip"`
	DstIP       *string           `bun:"dst_ip"`
	DstPort     *uint32           `bun:"dst_port"`
	Protocol    *string           `bun:"protocol"`
	Reason      string            `bun:"reason,type:text,notnull"`
	Metadata    map[string]string `bun:"metadata,type:jsonb"`
	Timestamp   int64             `bun:"timestamp,notnull"`
	CreatedAt   time.Time         `bun:"created_at,default:current_timestamp"`
	AgentID     *string           `bun:"agent_id"`
}

// QueryAlerts retrieves security alerts with pagination and filtering
func (s *AlertStorage) QueryAlerts(ctx context.Context, opts *AlertQueryOptions) ([]*alertpb.SecurityAlert, int64, error) {
	query := s.db.NewSelect().
		Model((*AlertRow)(nil))

	if opts.Level != nil {
		query = query.Where("level = ?", *opts.Level)
	}
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	}
	if opts.ProcessPath != nil {
		query = query.Where("exe_path LIKE ?", *opts.ProcessPath+"%")
	}
	if opts.StartTime != nil {
		query = query.Where("timestamp >= ?", *opts.StartTime)
	}
	if opts.EndTime != nil {
		query = query.Where("timestamp <= ?", *opts.EndTime)
	}

	totalCount, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get alert count: %w", err)
	}

	offset := (opts.Page - 1) * opts.PageSize
	var rows []AlertRow
	err = s.db.NewSelect().
		Model(&rows).
		Where("1=1").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if opts.Level != nil {
				q = q.Where("level = ?", *opts.Level)
			}
			if opts.Type != nil {
				q = q.Where("type = ?", *opts.Type)
			}
			if opts.ProcessPath != nil {
				q = q.Where("exe_path LIKE ?", *opts.ProcessPath+"%")
			}
			if opts.StartTime != nil {
				q = q.Where("timestamp >= ?", *opts.StartTime)
			}
			if opts.EndTime != nil {
				q = q.Where("timestamp <= ?", *opts.EndTime)
			}
			return q
		}).
		Order("timestamp DESC").
		Limit(opts.PageSize).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}

	alerts := make([]*alertpb.SecurityAlert, 0, len(rows))
	for _, row := range rows {
		alerts = append(alerts, s.alertRowToProto(&row))
	}

	return alerts, int64(totalCount), nil
}

// GetAlertByID retrieves a single alert by ID
func (s *AlertStorage) GetAlertByID(ctx context.Context, alertID string) (*alertpb.SecurityAlert, error) {
	var row AlertRow
	err := s.db.NewSelect().
		Model(&row).
		Where("alert_id = ?", alertID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return s.alertRowToProto(&row), nil
}

// CreateAlert creates a new security alert
func (s *AlertStorage) CreateAlert(ctx context.Context, alert *alertpb.SecurityAlert) error {
	row := &AlertRow{
		AlertID:   alert.AlertId,
		Level:     int32(alert.Level),
		Type:      int32(alert.Type),
		Reason:    alert.Reason,
		Metadata:  alert.Metadata,
		Timestamp: alert.Timestamp,
	}

	if alert.AgentId != "" {
		row.AgentID = &alert.AgentId
	}

	if alert.ProcessInfo != nil {
		row.PID = &alert.ProcessInfo.Pid
		row.PPID = &alert.ProcessInfo.Ppid
		row.UID = &alert.ProcessInfo.Uid
		row.GID = &alert.ProcessInfo.Gid
		if alert.ProcessInfo.Comm != "" {
			row.Comm = &alert.ProcessInfo.Comm
		}
		if alert.ProcessInfo.ExePath != "" {
			row.ExePath = &alert.ProcessInfo.ExePath
		}
		if alert.ProcessInfo.ContainerId != "" {
			row.ContainerID = &alert.ProcessInfo.ContainerId
		}
	}

	if alert.FlowEvent != nil {
		if alert.FlowEvent.SrcIp != 0 {
			ipStr := uint32ToIPString(alert.FlowEvent.SrcIp)
			row.SrcIP = &ipStr
		}
		if alert.FlowEvent.DstIp != 0 {
			ipStr := uint32ToIPString(alert.FlowEvent.DstIp)
			row.DstIP = &ipStr
		}
		if alert.FlowEvent.DstPort != 0 {
			row.DstPort = &alert.FlowEvent.DstPort
		}
		protocolStr := alert.FlowEvent.Protocol.String()
		if protocolStr != "" && protocolStr != "PROTOCOL_UNKNOWN" {
			row.Protocol = &protocolStr
		}
	}

	_, err := s.db.NewInsert().
		Model(row).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// GetAlertStats retrieves aggregated alert statistics
func (s *AlertStorage) GetAlertStats(ctx context.Context, startTime, endTime int64, timeWindow string) (*AlertStats, error) {
	stats := &AlertStats{
		ByLevel:      make(map[string]int64),
		ByType:       make(map[string]int64),
		TopProcesses: []ProcessAlertCount{},
		Timeline:     []TimelineBucket{},
	}

	var levelRows []struct {
		Level int32 `bun:"level"`
		Count int64 `bun:"count"`
	}
	err := s.db.NewSelect().
		Model((*AlertRow)(nil)).
		ColumnExpr("level, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Group("level").
		Order("level").
		Scan(ctx, &levelRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats by level: %w", err)
	}

	for _, row := range levelRows {
		levelName := alertpb.AlertLevel(row.Level).String()
		stats.ByLevel[levelName] = row.Count
	}

	var typeRows []struct {
		Type  int32 `bun:"type"`
		Count int64 `bun:"count"`
	}
	err = s.db.NewSelect().
		Model((*AlertRow)(nil)).
		ColumnExpr("type, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Group("type").
		Order("type").
		Scan(ctx, &typeRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats by type: %w", err)
	}

	for _, row := range typeRows {
		typeName := alertpb.AlertType(row.Type).String()
		stats.ByType[typeName] = row.Count
	}

	var processRows []struct {
		ExePath string `bun:"exe_path"`
		Count   int64  `bun:"count"`
	}
	err = s.db.NewSelect().
		Model((*AlertRow)(nil)).
		ColumnExpr("exe_path, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Where("exe_path IS NOT NULL AND exe_path != ''").
		Group("exe_path").
		Order("count DESC").
		Limit(10).
		Scan(ctx, &processRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get top processes: %w", err)
	}

	for _, row := range processRows {
		stats.TopProcesses = append(stats.TopProcesses, ProcessAlertCount{
			ExePath: row.ExePath,
			Count:   row.Count,
		})
	}

	var truncFunc string
	switch timeWindow {
	case "7d", "30d":
		truncFunc = "day"
	default:
		truncFunc = "hour"
	}

	var timelineRows []struct {
		Bucket int64 `bun:"bucket"`
		Count  int64 `bun:"count"`
	}
	err = s.db.NewSelect().
		Model((*AlertRow)(nil)).
		ColumnExpr("EXTRACT(EPOCH FROM date_trunc(?, to_timestamp(timestamp / 1000000000))) * 1000000000 AS bucket", truncFunc).
		ColumnExpr("COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Group("bucket").
		Order("bucket").
		Scan(ctx, &timelineRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline data: %w", err)
	}

	for _, row := range timelineRows {
		stats.Timeline = append(stats.Timeline, TimelineBucket{
			Timestamp: row.Bucket,
			Count:     row.Count,
		})
	}

	return stats, nil
}

// alertRowToProto converts database row to protobuf message
func (s *AlertStorage) alertRowToProto(row *AlertRow) *alertpb.SecurityAlert {
	pbAlert := &alertpb.SecurityAlert{
		AlertId:   row.AlertID,
		Level:     alertpb.AlertLevel(row.Level),
		Type:      alertpb.AlertType(row.Type),
		Reason:    row.Reason,
		Timestamp: row.Timestamp,
		Metadata:  row.Metadata,
	}

	if row.AgentID != nil {
		pbAlert.AgentId = *row.AgentID
	}

	if row.PID != nil || row.Comm != nil || row.ExePath != nil {
		pbAlert.ProcessInfo = &flowpb.ProcessInfo{}
		if row.PID != nil {
			pbAlert.ProcessInfo.Pid = *row.PID
		}
		if row.PPID != nil {
			pbAlert.ProcessInfo.Ppid = *row.PPID
		}
		if row.UID != nil {
			pbAlert.ProcessInfo.Uid = *row.UID
		}
		if row.GID != nil {
			pbAlert.ProcessInfo.Gid = *row.GID
		}
		if row.Comm != nil {
			pbAlert.ProcessInfo.Comm = *row.Comm
		}
		if row.ExePath != nil {
			pbAlert.ProcessInfo.ExePath = *row.ExePath
		}
		if row.ContainerID != nil {
			pbAlert.ProcessInfo.ContainerId = *row.ContainerID
		}
	}

	if row.SrcIP != nil || row.DstIP != nil {
		pbAlert.FlowEvent = &flowpb.FlowEvent{}
		if row.SrcIP != nil {
			pbAlert.FlowEvent.SrcIp = ipStringToUint32(*row.SrcIP)
		}
		if row.DstIP != nil {
			pbAlert.FlowEvent.DstIp = ipStringToUint32(*row.DstIP)
		}
		if row.DstPort != nil {
			pbAlert.FlowEvent.DstPort = *row.DstPort
		}
	}

	return pbAlert
}

// uint32ToIPString converts a uint32 IP address to string format
func uint32ToIPString(ipUint32 uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipUint32)
	return ip.String()
}

// ipStringToUint32 converts an IP address string to uint32
func ipStringToUint32(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

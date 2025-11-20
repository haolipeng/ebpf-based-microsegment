package storage

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	alertpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/alert"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
)

// AlertStorage handles security alert data persistence
type AlertStorage struct {
	db *sql.DB
}

// NewAlertStorage creates a new AlertStorage
func NewAlertStorage(db *sql.DB) *AlertStorage {
	return &AlertStorage{db: db}
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

// QueryAlerts retrieves security alerts with pagination and filtering
func (s *AlertStorage) QueryAlerts(ctx context.Context, opts *AlertQueryOptions) ([]*alertpb.SecurityAlert, int64, error) {
	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if opts.Level != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("level = $%d", argIdx))
		args = append(args, *opts.Level)
		argIdx++
	}

	if opts.Type != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *opts.Type)
		argIdx++
	}

	if opts.ProcessPath != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("exe_path LIKE $%d", argIdx))
		args = append(args, *opts.ProcessPath+"%")
		argIdx++
	}

	if opts.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *opts.StartTime)
		argIdx++
	}

	if opts.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *opts.EndTime)
		argIdx++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Get total count
	var totalCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM security_alerts WHERE %s", whereClause)
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get alert count: %w", err)
	}

	// Query alerts with pagination
	offset := (opts.Page - 1) * opts.PageSize
	args = append(args, opts.PageSize, offset)

	query := fmt.Sprintf(`
		SELECT id, alert_id, level, type, pid, ppid, uid, gid, comm, exe_path, container_id,
		       flow_id, src_ip, dst_ip, dst_port, protocol, reason, metadata,
		       timestamp, created_at, agent_id
		FROM security_alerts
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	alerts := []*alertpb.SecurityAlert{}
	for rows.Next() {
		var alert SecurityAlert
		var metadataJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.AlertID,
			&alert.Level,
			&alert.Type,
			&alert.PID,
			&alert.PPID,
			&alert.UID,
			&alert.GID,
			&alert.Comm,
			&alert.ExePath,
			&alert.ContainerID,
			&alert.FlowID,
			&alert.SrcIP,
			&alert.DstIP,
			&alert.DstPort,
			&alert.Protocol,
			&alert.Reason,
			&metadataJSON,
			&alert.Timestamp,
			&alert.CreatedAt,
			&alert.AgentID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan alert: %w", err)
		}

		// Convert to protobuf
		pbAlert := s.alertModelToProto(&alert, metadataJSON)
		alerts = append(alerts, pbAlert)
	}

	return alerts, totalCount, nil
}

// GetAlertByID retrieves a single alert by ID
func (s *AlertStorage) GetAlertByID(ctx context.Context, alertID string) (*alertpb.SecurityAlert, error) {
	var alert SecurityAlert
	var metadataJSON []byte

	query := `
		SELECT id, alert_id, level, type, pid, ppid, uid, gid, comm, exe_path, container_id,
		       flow_id, src_ip, dst_ip, dst_port, protocol, reason, metadata,
		       timestamp, created_at, agent_id
		FROM security_alerts
		WHERE alert_id = $1
	`

	err := s.db.QueryRowContext(ctx, query, alertID).Scan(
		&alert.ID,
		&alert.AlertID,
		&alert.Level,
		&alert.Type,
		&alert.PID,
		&alert.PPID,
		&alert.UID,
		&alert.GID,
		&alert.Comm,
		&alert.ExePath,
		&alert.ContainerID,
		&alert.FlowID,
		&alert.SrcIP,
		&alert.DstIP,
		&alert.DstPort,
		&alert.Protocol,
		&alert.Reason,
		&metadataJSON,
		&alert.Timestamp,
		&alert.CreatedAt,
		&alert.AgentID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return s.alertModelToProto(&alert, metadataJSON), nil
}

// CreateAlert creates a new security alert
func (s *AlertStorage) CreateAlert(ctx context.Context, alert *alertpb.SecurityAlert) error {
	metadataJSON, _ := json.Marshal(alert.Metadata)

	// Extract process info
	var pid, ppid, uid, gid *uint32
	var comm, exePath, containerID *string
	if alert.ProcessInfo != nil {
		pid = &alert.ProcessInfo.Pid
		ppid = &alert.ProcessInfo.Ppid
		uid = &alert.ProcessInfo.Uid
		gid = &alert.ProcessInfo.Gid
		if alert.ProcessInfo.Comm != "" {
			comm = &alert.ProcessInfo.Comm
		}
		if alert.ProcessInfo.ExePath != "" {
			exePath = &alert.ProcessInfo.ExePath
		}
		if alert.ProcessInfo.ContainerId != "" {
			containerID = &alert.ProcessInfo.ContainerId
		}
	}

	// Extract flow info
	var flowID *int64
	var srcIP, dstIP *string
	var dstPort *uint32
	var protocol *string
	if alert.FlowEvent != nil {
		if alert.FlowEvent.SrcIp != 0 {
			ipStr := uint32ToIPString(alert.FlowEvent.SrcIp)
			srcIP = &ipStr
		}
		if alert.FlowEvent.DstIp != 0 {
			ipStr := uint32ToIPString(alert.FlowEvent.DstIp)
			dstIP = &ipStr
		}
		if alert.FlowEvent.DstPort != 0 {
			dstPort = &alert.FlowEvent.DstPort
		}
		// Convert protocol enum to string
		protocolStr := alert.FlowEvent.Protocol.String()
		if protocolStr != "" && protocolStr != "PROTOCOL_UNKNOWN" {
			protocol = &protocolStr
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO security_alerts (
			alert_id, level, type, pid, ppid, uid, gid, comm, exe_path, container_id,
			flow_id, src_ip, dst_ip, dst_port, protocol, reason, metadata,
			timestamp, agent_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, alert.AlertId, alert.Level, alert.Type, pid, ppid, uid, gid, comm, exePath, containerID,
		flowID, srcIP, dstIP, dstPort, protocol, alert.Reason, metadataJSON,
		alert.Timestamp, alert.AgentId)

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

	// Get counts by level
	levelRows, err := s.db.QueryContext(ctx, `
		SELECT level, COUNT(*) as count
		FROM security_alerts
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY level
		ORDER BY level
	`, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats by level: %w", err)
	}
	defer levelRows.Close()

	for levelRows.Next() {
		var level int32
		var count int64
		levelRows.Scan(&level, &count)
		levelName := alertpb.AlertLevel(level).String()
		stats.ByLevel[levelName] = count
	}

	// Get counts by type
	typeRows, err := s.db.QueryContext(ctx, `
		SELECT type, COUNT(*) as count
		FROM security_alerts
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY type
		ORDER BY type
	`, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats by type: %w", err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var alertType int32
		var count int64
		typeRows.Scan(&alertType, &count)
		typeName := alertpb.AlertType(alertType).String()
		stats.ByType[typeName] = count
	}

	// Get top processes by alert count
	processRows, err := s.db.QueryContext(ctx, `
		SELECT exe_path, COUNT(*) as count
		FROM security_alerts
		WHERE timestamp >= $1 AND timestamp <= $2
		  AND exe_path IS NOT NULL AND exe_path != ''
		GROUP BY exe_path
		ORDER BY count DESC
		LIMIT 10
	`, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get top processes: %w", err)
	}
	defer processRows.Close()

	for processRows.Next() {
		var exePath string
		var count int64
		processRows.Scan(&exePath, &count)
		stats.TopProcesses = append(stats.TopProcesses, ProcessAlertCount{
			ExePath: exePath,
			Count:   count,
		})
	}

	// Get timeline data (hourly or daily buckets based on time window)
	var bucketInterval string
	switch timeWindow {
	case "24h":
		bucketInterval = "1 hour"
	case "7d":
		bucketInterval = "1 day"
	case "30d":
		bucketInterval = "1 day"
	default:
		bucketInterval = "1 hour"
	}

	timelineQuery := fmt.Sprintf(`
		SELECT
			EXTRACT(EPOCH FROM date_trunc('%s', to_timestamp(timestamp / 1000000000))) * 1000000000 AS bucket,
			COUNT(*) as count
		FROM security_alerts
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY bucket
		ORDER BY bucket
	`, strings.TrimSuffix(bucketInterval, " hour"))

	if strings.Contains(bucketInterval, "day") {
		timelineQuery = fmt.Sprintf(`
			SELECT
				EXTRACT(EPOCH FROM date_trunc('day', to_timestamp(timestamp / 1000000000))) * 1000000000 AS bucket,
				COUNT(*) as count
			FROM security_alerts
			WHERE timestamp >= $1 AND timestamp <= $2
			GROUP BY bucket
			ORDER BY bucket
		`)
	}

	timelineRows, err := s.db.QueryContext(ctx, timelineQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline data: %w", err)
	}
	defer timelineRows.Close()

	for timelineRows.Next() {
		var bucket, count int64
		timelineRows.Scan(&bucket, &count)
		stats.Timeline = append(stats.Timeline, TimelineBucket{
			Timestamp: bucket,
			Count:     count,
		})
	}

	return stats, nil
}

// alertModelToProto converts database model to protobuf message
func (s *AlertStorage) alertModelToProto(alert *SecurityAlert, metadataJSON []byte) *alertpb.SecurityAlert {
	pbAlert := &alertpb.SecurityAlert{
		AlertId:   alert.AlertID,
		Level:     alertpb.AlertLevel(alert.Level),
		Type:      alertpb.AlertType(alert.Type),
		Reason:    alert.Reason,
		Timestamp: alert.Timestamp,
		Metadata:  make(map[string]string),
	}

	if alert.AgentID != nil {
		pbAlert.AgentId = *alert.AgentID
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &pbAlert.Metadata)
	}

	// Build ProcessInfo if any process fields are present
	if alert.PID != nil || alert.Comm != nil || alert.ExePath != nil {
		pbAlert.ProcessInfo = &flowpb.ProcessInfo{}
		if alert.PID != nil {
			pbAlert.ProcessInfo.Pid = uint32(*alert.PID)
		}
		if alert.PPID != nil {
			pbAlert.ProcessInfo.Ppid = uint32(*alert.PPID)
		}
		if alert.UID != nil {
			pbAlert.ProcessInfo.Uid = uint32(*alert.UID)
		}
		if alert.GID != nil {
			pbAlert.ProcessInfo.Gid = uint32(*alert.GID)
		}
		if alert.Comm != nil {
			pbAlert.ProcessInfo.Comm = *alert.Comm
		}
		if alert.ExePath != nil {
			pbAlert.ProcessInfo.ExePath = *alert.ExePath
		}
		if alert.ContainerID != nil {
			pbAlert.ProcessInfo.ContainerId = *alert.ContainerID
		}
	}

	// Build FlowEvent if any flow fields are present
	if alert.SrcIP != nil || alert.DstIP != nil {
		pbAlert.FlowEvent = &flowpb.FlowEvent{}
		if alert.SrcIP != nil {
			pbAlert.FlowEvent.SrcIp = ipStringToUint32(*alert.SrcIP)
		}
		if alert.DstIP != nil {
			pbAlert.FlowEvent.DstIp = ipStringToUint32(*alert.DstIP)
		}
		if alert.DstPort != nil {
			pbAlert.FlowEvent.DstPort = *alert.DstPort
		}
		// Note: Protocol stored as string in DB, would need conversion back to enum
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

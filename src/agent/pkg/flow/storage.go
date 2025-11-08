package flow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage backend
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure SQLite for better performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",   // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous=NORMAL", // Faster writes (still safe)
		"PRAGMA cache_size=-64000",  // 64MB cache
		"PRAGMA temp_store=MEMORY",  // Use memory for temp tables
		"PRAGMA foreign_keys=ON",    // Enable foreign keys
		"PRAGMA busy_timeout=5000",  // 5s timeout for locks
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("[Flow Storage] SQLite storage initialized at %s", dbPath)
	return storage, nil
}

// initSchema creates the necessary database tables and indexes
func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS flows (
		id TEXT PRIMARY KEY,
		source_ip TEXT NOT NULL,
		source_port INTEGER NOT NULL,
		dest_ip TEXT NOT NULL,
		dest_port INTEGER NOT NULL,
		protocol TEXT NOT NULL,

		packet_count INTEGER NOT NULL,
		byte_count INTEGER NOT NULL,
		duration_ms INTEGER,

		start_time DATETIME NOT NULL,
		end_time DATETIME,
		last_seen DATETIME NOT NULL,

		source_labels TEXT,  -- JSON
		dest_labels TEXT,    -- JSON

		policy_id INTEGER,
		policy_action TEXT NOT NULL,

		state TEXT NOT NULL,
		direction TEXT NOT NULL,
		event_type TEXT NOT NULL,

		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_flows_start_time ON flows(start_time DESC);
	CREATE INDEX IF NOT EXISTS idx_flows_last_seen ON flows(last_seen DESC);
	CREATE INDEX IF NOT EXISTS idx_flows_source_ip ON flows(source_ip);
	CREATE INDEX IF NOT EXISTS idx_flows_dest_ip ON flows(dest_ip);
	CREATE INDEX IF NOT EXISTS idx_flows_protocol ON flows(protocol);
	CREATE INDEX IF NOT EXISTS idx_flows_state ON flows(state);
	CREATE INDEX IF NOT EXISTS idx_flows_policy_action ON flows(policy_action);
	CREATE INDEX IF NOT EXISTS idx_flows_direction ON flows(direction);

	-- Composite indexes for complex queries
	CREATE INDEX IF NOT EXISTS idx_flows_time_state ON flows(start_time, state);
	CREATE INDEX IF NOT EXISTS idx_flows_src_dst ON flows(source_ip, dest_ip);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SaveFlow persists a new flow to storage
func (s *SQLiteStorage) SaveFlow(flow *Flow) error {
	query := `
		INSERT INTO flows (
			id, source_ip, source_port, dest_ip, dest_port, protocol,
			packet_count, byte_count, duration_ms,
			start_time, end_time, last_seen,
			source_labels, dest_labels,
			policy_id, policy_action,
			state, direction, event_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	srcLabelsJSON, err := json.Marshal(flow.SourceLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal source labels: %w", err)
	}

	dstLabelsJSON, err := json.Marshal(flow.DestLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal dest labels: %w", err)
	}

	var endTime interface{}
	if flow.EndTime != nil {
		endTime = flow.EndTime
	}

	var policyID interface{}
	if flow.PolicyID > 0 {
		policyID = flow.PolicyID
	}

	_, err = s.db.Exec(query,
		flow.ID, flow.SourceIP, flow.SourcePort, flow.DestIP, flow.DestPort, flow.Protocol,
		flow.PacketCount, flow.ByteCount, flow.Duration,
		flow.StartTime, endTime, flow.LastSeen,
		string(srcLabelsJSON), string(dstLabelsJSON),
		policyID, flow.PolicyAction,
		flow.State, flow.Direction, flow.EventType,
	)

	if err != nil {
		return fmt.Errorf("failed to insert flow: %w", err)
	}

	return nil
}

// UpdateFlow updates an existing flow
func (s *SQLiteStorage) UpdateFlow(flow *Flow) error {
	query := `
		UPDATE flows SET
			packet_count = ?,
			byte_count = ?,
			duration_ms = ?,
			end_time = ?,
			last_seen = ?,
			source_labels = ?,
			dest_labels = ?,
			policy_id = ?,
			policy_action = ?,
			state = ?,
			direction = ?,
			event_type = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	srcLabelsJSON, err := json.Marshal(flow.SourceLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal source labels: %w", err)
	}

	dstLabelsJSON, err := json.Marshal(flow.DestLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal dest labels: %w", err)
	}

	var endTime interface{}
	if flow.EndTime != nil {
		endTime = flow.EndTime
	}

	var policyID interface{}
	if flow.PolicyID > 0 {
		policyID = flow.PolicyID
	}

	result, err := s.db.Exec(query,
		flow.PacketCount, flow.ByteCount, flow.Duration,
		endTime, flow.LastSeen,
		string(srcLabelsJSON), string(dstLabelsJSON),
		policyID, flow.PolicyAction,
		flow.State, flow.Direction, flow.EventType,
		flow.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update flow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Flow doesn't exist, insert it instead
		return s.SaveFlow(flow)
	}

	return nil
}

// GetFlow retrieves a flow by ID
func (s *SQLiteStorage) GetFlow(id string) (*Flow, error) {
	query := `
		SELECT id, source_ip, source_port, dest_ip, dest_port, protocol,
			   packet_count, byte_count, duration_ms,
			   start_time, end_time, last_seen,
			   source_labels, dest_labels,
			   policy_id, policy_action,
			   state, direction, event_type
		FROM flows
		WHERE id = ?
	`

	flow := &Flow{}
	var srcLabelsJSON, dstLabelsJSON string
	var endTime sql.NullTime
	var policyID sql.NullInt64

	err := s.db.QueryRow(query, id).Scan(
		&flow.ID, &flow.SourceIP, &flow.SourcePort, &flow.DestIP, &flow.DestPort, &flow.Protocol,
		&flow.PacketCount, &flow.ByteCount, &flow.Duration,
		&flow.StartTime, &endTime, &flow.LastSeen,
		&srcLabelsJSON, &dstLabelsJSON,
		&policyID, &flow.PolicyAction,
		&flow.State, &flow.Direction, &flow.EventType,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("flow not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query flow: %w", err)
	}

	// Parse JSON labels
	if err := json.Unmarshal([]byte(srcLabelsJSON), &flow.SourceLabels); err != nil {
		flow.SourceLabels = make(map[string]string)
	}
	if err := json.Unmarshal([]byte(dstLabelsJSON), &flow.DestLabels); err != nil {
		flow.DestLabels = make(map[string]string)
	}

	if endTime.Valid {
		flow.EndTime = &endTime.Time
	}
	if policyID.Valid {
		flow.PolicyID = uint32(policyID.Int64)
	}

	return flow, nil
}

// QueryFlows queries flows based on filters with pagination
func (s *SQLiteStorage) QueryFlows(query *FlowQuery) ([]*Flow, error) {
	// Build SQL query dynamically based on filters
	sqlQuery := `
		SELECT id, source_ip, source_port, dest_ip, dest_port, protocol,
			   packet_count, byte_count, duration_ms,
			   start_time, end_time, last_seen,
			   source_labels, dest_labels,
			   policy_id, policy_action,
			   state, direction, event_type
		FROM flows
		WHERE 1=1
	`

	var args []interface{}
	var conditions []string

	// Time range filters
	if query.StartTime != nil {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, query.StartTime)
	}
	if query.EndTime != nil {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, query.EndTime)
	}

	// Flow filters
	if query.SourceIP != nil {
		conditions = append(conditions, "source_ip = ?")
		args = append(args, *query.SourceIP)
	}
	if query.DestIP != nil {
		conditions = append(conditions, "dest_ip = ?")
		args = append(args, *query.DestIP)
	}
	if query.Protocol != nil {
		conditions = append(conditions, "protocol = ?")
		args = append(args, *query.Protocol)
	}
	if query.State != nil {
		conditions = append(conditions, "state = ?")
		args = append(args, *query.State)
	}
	if query.Direction != nil {
		conditions = append(conditions, "direction = ?")
		args = append(args, *query.Direction)
	}
	if query.PolicyAction != nil {
		conditions = append(conditions, "policy_action = ?")
		args = append(args, *query.PolicyAction)
	}

	// Label filters (simple JSON search - not optimal but functional)
	for key, value := range query.SourceLabels {
		conditions = append(conditions, "source_labels LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":\"%s\"%%", key, value))
	}
	for key, value := range query.DestLabels {
		conditions = append(conditions, "dest_labels LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":\"%s\"%%", key, value))
	}

	// Append conditions
	if len(conditions) > 0 {
		sqlQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Sorting
	sortBy := "start_time"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	sortOrder := "DESC"
	if query.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	sqlQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	limit := 100
	if query.Limit > 0 {
		limit = query.Limit
	}
	sqlQuery += " LIMIT ?"
	args = append(args, limit)

	if query.Offset > 0 {
		sqlQuery += " OFFSET ?"
		args = append(args, query.Offset)
	}

	// Execute query
	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query flows: %w", err)
	}
	defer rows.Close()

	// Parse results
	var flows []*Flow
	for rows.Next() {
		flow := &Flow{}
		var srcLabelsJSON, dstLabelsJSON string
		var endTime sql.NullTime
		var policyID sql.NullInt64

		err := rows.Scan(
			&flow.ID, &flow.SourceIP, &flow.SourcePort, &flow.DestIP, &flow.DestPort, &flow.Protocol,
			&flow.PacketCount, &flow.ByteCount, &flow.Duration,
			&flow.StartTime, &endTime, &flow.LastSeen,
			&srcLabelsJSON, &dstLabelsJSON,
			&policyID, &flow.PolicyAction,
			&flow.State, &flow.Direction, &flow.EventType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flow: %w", err)
		}

		// Parse JSON labels
		if err := json.Unmarshal([]byte(srcLabelsJSON), &flow.SourceLabels); err != nil {
			flow.SourceLabels = make(map[string]string)
		}
		if err := json.Unmarshal([]byte(dstLabelsJSON), &flow.DestLabels); err != nil {
			flow.DestLabels = make(map[string]string)
		}

		if endTime.Valid {
			flow.EndTime = &endTime.Time
		}
		if policyID.Valid {
			flow.PolicyID = uint32(policyID.Int64)
		}

		flows = append(flows, flow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating flows: %w", err)
	}

	return flows, nil
}

// GetFlowSummary returns aggregated flow statistics
func (s *SQLiteStorage) GetFlowSummary(startTime, endTime time.Time) (*FlowSummary, error) {
	summary := &FlowSummary{}

	// Overall statistics
	query := `
		SELECT
			COUNT(*) as total_flows,
			SUM(CASE WHEN state = 'ACTIVE' THEN 1 ELSE 0 END) as active_flows,
			SUM(CASE WHEN state = 'CLOSED' THEN 1 ELSE 0 END) as closed_flows,
			SUM(packet_count) as total_packets,
			SUM(byte_count) as total_bytes,
			SUM(CASE WHEN policy_action = 'ALLOW' THEN 1 ELSE 0 END) as allowed_flows,
			SUM(CASE WHEN policy_action = 'DENY' THEN 1 ELSE 0 END) as denied_flows
		FROM flows
		WHERE start_time >= ? AND start_time <= ?
	`

	err := s.db.QueryRow(query, startTime, endTime).Scan(
		&summary.TotalFlows,
		&summary.ActiveFlows,
		&summary.ClosedFlows,
		&summary.TotalPackets,
		&summary.TotalBytes,
		&summary.AllowedFlows,
		&summary.DeniedFlows,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Top protocols
	protocolQuery := `
		SELECT protocol, COUNT(*) as flow_count,
			   SUM(packet_count) as packet_count, SUM(byte_count) as byte_count
		FROM flows
		WHERE start_time >= ? AND start_time <= ?
		GROUP BY protocol
		ORDER BY flow_count DESC
		LIMIT 10
	`

	rows, err := s.db.Query(protocolQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query protocols: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stats ProtocolStats
		if err := rows.Scan(&stats.Protocol, &stats.FlowCount, &stats.PacketCount, &stats.ByteCount); err != nil {
			return nil, fmt.Errorf("failed to scan protocol: %w", err)
		}
		summary.TopProtocols = append(summary.TopProtocols, stats)
	}

	// Top source IPs
	srcIPQuery := `
		SELECT source_ip, COUNT(*) as flow_count,
			   SUM(packet_count) as packet_count, SUM(byte_count) as byte_count
		FROM flows
		WHERE start_time >= ? AND start_time <= ?
		GROUP BY source_ip
		ORDER BY flow_count DESC
		LIMIT 10
	`

	rows, err = s.db.Query(srcIPQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query source IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stats IPStats
		if err := rows.Scan(&stats.IP, &stats.FlowCount, &stats.PacketCount, &stats.ByteCount); err != nil {
			return nil, fmt.Errorf("failed to scan source IP: %w", err)
		}
		summary.TopSourceIPs = append(summary.TopSourceIPs, stats)
	}

	// Top destination IPs
	dstIPQuery := `
		SELECT dest_ip, COUNT(*) as flow_count,
			   SUM(packet_count) as packet_count, SUM(byte_count) as byte_count
		FROM flows
		WHERE start_time >= ? AND start_time <= ?
		GROUP BY dest_ip
		ORDER BY flow_count DESC
		LIMIT 10
	`

	rows, err = s.db.Query(dstIPQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query dest IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stats IPStats
		if err := rows.Scan(&stats.IP, &stats.FlowCount, &stats.PacketCount, &stats.ByteCount); err != nil {
			return nil, fmt.Errorf("failed to scan dest IP: %w", err)
		}
		summary.TopDestIPs = append(summary.TopDestIPs, stats)
	}

	return summary, nil
}

// DeleteOldFlows deletes flows older than the specified duration
func (s *SQLiteStorage) DeleteOldFlows(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	query := `DELETE FROM flows WHERE start_time < ?`
	result, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old flows: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Printf("[Flow Storage] Deleted %d flows older than %v", rowsAffected, olderThan)
	}

	return rowsAffected, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	log.Println("[Flow Storage] Closing SQLite storage...")
	return s.db.Close()
}

// Vacuum optimizes the database (should be run periodically)
func (s *SQLiteStorage) Vacuum() error {
	log.Println("[Flow Storage] Running VACUUM to optimize database...")
	_, err := s.db.Exec("VACUUM")
	return err
}

// GetDatabaseSize returns the database file size in bytes
func (s *SQLiteStorage) GetDatabaseSize() (int64, error) {
	var pageCount, pageSize int64
	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0, err
	}
	if err := s.db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, err
	}
	return pageCount * pageSize, nil
}

// input: agent registration data, heartbeat updates
// output: agent list queries, status updates, heartbeat timestamps
// pos: storage - PostgreSQL storage layer for Agent lifecycle data

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
)

// AgentStorage handles agent data persistence using Bun
type AgentStorage struct {
	db *bun.DB
}

// NewAgentStorage creates a new AgentStorage
func NewAgentStorage(db *sql.DB) *AgentStorage {
	bunDB := bun.NewDB(db, pgdialect.New())
	return &AgentStorage{db: bunDB}
}

// Agent represents an agent with metrics
type Agent struct {
	AgentID          string
	Hostname         string
	Version          string
	Interface        string
	IPAddresses      []string
	OS               string
	KernelVersion    string
	StartTime        int64
	LastHeartbeat    int64
	Status           string
	CPUUsage         float32
	MemoryUsage      uint64
	PacketsProcessed uint64
	ActiveSessions   uint32
	FlowsReported    uint64
	ActivePolicies   uint32
}

// AgentRow represents a row in the agents table
type AgentRow struct {
	bun.BaseModel `bun:"table:agents,alias:a"`

	ID            int64      `bun:"id,pk,autoincrement"`
	AgentID       string     `bun:"agent_id,unique,notnull"`
	Hostname      string     `bun:"hostname"`
	Version       string     `bun:"version"`
	Interface     string     `bun:"interface"`
	IPAddresses   []string   `bun:"ip_addresses,array"`
	OS            string     `bun:"os"`
	KernelVersion string     `bun:"kernel_version"`
	StartTime     time.Time  `bun:"start_time"`
	LastHeartbeat time.Time  `bun:"last_heartbeat"`
	Status        string     `bun:"status"`
	OfflineAt     *time.Time `bun:"offline_at"`
	OfflineReason *string    `bun:"offline_reason"`
	CreatedAt     time.Time  `bun:"created_at,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"updated_at,default:current_timestamp"`
}

// AgentMetricsRow represents a row in the agent_metrics table
type AgentMetricsRow struct {
	bun.BaseModel `bun:"table:agent_metrics,alias:m"`

	ID               int64             `bun:"id,pk,autoincrement"`
	AgentID          string            `bun:"agent_id,unique,notnull"`
	CPUUsage         float32           `bun:"cpu_usage"`
	MemoryUsage      uint64            `bun:"memory_usage"`
	PacketsProcessed uint64            `bun:"packets_processed"`
	ActiveSessions   uint32            `bun:"active_sessions"`
	FlowsReported    uint64            `bun:"flows_reported"`
	ActivePolicies   uint32            `bun:"active_policies"`
	PolicyCount      uint32            `bun:"policy_count"`
	PolicyVersion    uint64            `bun:"policy_version"`
	WorkloadCount    uint32            `bun:"workload_count"`
	AgentStatus      string            `bun:"agent_status"`
	Uptime           uint64            `bun:"uptime"`
	Errors           []string          `bun:"errors,array"`
	Metadata         map[string]string `bun:"metadata,type:jsonb"`
	LastStatusReport *time.Time        `bun:"last_status_report"`
	UpdatedAt        time.Time         `bun:"updated_at,default:current_timestamp"`
}

// AgentWithMetrics combines agent and metrics data
type AgentWithMetrics struct {
	bun.BaseModel `bun:"table:agents,alias:a"`

	AgentID       string    `bun:"agent_id"`
	Hostname      string    `bun:"hostname"`
	Version       string    `bun:"version"`
	Interface     string    `bun:"interface"`
	IPAddresses   []string  `bun:"ip_addresses,array"`
	OS            string    `bun:"os"`
	KernelVersion string    `bun:"kernel_version"`
	StartTime     time.Time `bun:"start_time"`
	LastHeartbeat time.Time `bun:"last_heartbeat"`
	Status        string    `bun:"status"`

	CPUUsage         float32 `bun:"cpu_usage"`
	MemoryUsage      uint64  `bun:"memory_usage"`
	PacketsProcessed uint64  `bun:"packets_processed"`
	ActiveSessions   uint32  `bun:"active_sessions"`
	FlowsReported    uint64  `bun:"flows_reported"`
	ActivePolicies   uint32  `bun:"active_policies"`
}

// RegisterAgent registers or updates an agent
func (s *AgentStorage) RegisterAgent(ctx context.Context, req *agentpb.RegisterRequest) error {
	agent := &AgentRow{
		AgentID:       req.AgentId,
		Hostname:      req.Hostname,
		Version:       req.Version,
		Interface:     req.Interface,
		IPAddresses:   req.IpAddresses,
		OS:            req.Os,
		KernelVersion: req.KernelVersion,
		StartTime:     time.Unix(0, req.StartTime),
		LastHeartbeat: time.Now(),
		Status:        "active",
	}

	_, err := s.db.NewInsert().
		Model(agent).
		On("CONFLICT (agent_id) DO UPDATE").
		Set("hostname = EXCLUDED.hostname").
		Set("version = EXCLUDED.version").
		Set("interface = EXCLUDED.interface").
		Set("ip_addresses = EXCLUDED.ip_addresses").
		Set("os = EXCLUDED.os").
		Set("kernel_version = EXCLUDED.kernel_version").
		Set("start_time = EXCLUDED.start_time").
		Set("last_heartbeat = CURRENT_TIMESTAMP").
		Set("status = 'active'").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	return nil
}

// UpdateHeartbeat updates agent last heartbeat timestamp
func (s *AgentStorage) UpdateHeartbeat(ctx context.Context, agentID string, metrics *agentpb.AgentMetrics) error {
	_, err := s.db.NewUpdate().
		Model((*AgentRow)(nil)).
		Set("last_heartbeat = CURRENT_TIMESTAMP").
		Set("status = 'active'").
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("agent_id = ?", agentID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	if metrics != nil {
		metricsRow := &AgentMetricsRow{
			AgentID:          agentID,
			CPUUsage:         metrics.CpuUsage,
			MemoryUsage:      metrics.MemoryUsage,
			PacketsProcessed: metrics.PacketsProcessed,
			ActiveSessions:   metrics.ActiveSessions,
			FlowsReported:    metrics.FlowsReported,
			ActivePolicies:   metrics.ActivePolicies,
		}

		_, err = s.db.NewInsert().
			Model(metricsRow).
			On("CONFLICT (agent_id) DO UPDATE").
			Set("cpu_usage = EXCLUDED.cpu_usage").
			Set("memory_usage = EXCLUDED.memory_usage").
			Set("packets_processed = EXCLUDED.packets_processed").
			Set("active_sessions = EXCLUDED.active_sessions").
			Set("flows_reported = EXCLUDED.flows_reported").
			Set("active_policies = EXCLUDED.active_policies").
			Set("updated_at = CURRENT_TIMESTAMP").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update agent metrics: %w", err)
		}
	}

	return nil
}

// GetAllAgents retrieves all registered agents with metrics
func (s *AgentStorage) GetAllAgents(ctx context.Context) ([]*Agent, error) {
	var rows []AgentWithMetrics

	err := s.db.NewSelect().
		ColumnExpr("a.agent_id, a.hostname, a.version, a.interface, a.ip_addresses").
		ColumnExpr("a.os, a.kernel_version, a.start_time, a.last_heartbeat, a.status").
		ColumnExpr("COALESCE(m.cpu_usage, 0) as cpu_usage").
		ColumnExpr("COALESCE(m.memory_usage, 0) as memory_usage").
		ColumnExpr("COALESCE(m.packets_processed, 0) as packets_processed").
		ColumnExpr("COALESCE(m.active_sessions, 0) as active_sessions").
		ColumnExpr("COALESCE(m.flows_reported, 0) as flows_reported").
		ColumnExpr("COALESCE(m.active_policies, 0) as active_policies").
		Table("agents").
		TableExpr("LEFT JOIN agent_metrics AS m ON a.agent_id = m.agent_id").
		Order("a.last_heartbeat DESC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}

	agents := make([]*Agent, 0, len(rows))
	for _, row := range rows {
		agents = append(agents, &Agent{
			AgentID:          row.AgentID,
			Hostname:         row.Hostname,
			Version:          row.Version,
			Interface:        row.Interface,
			IPAddresses:      row.IPAddresses,
			OS:               row.OS,
			KernelVersion:    row.KernelVersion,
			StartTime:        row.StartTime.UnixNano(),
			LastHeartbeat:    row.LastHeartbeat.UnixNano(),
			Status:           row.Status,
			CPUUsage:         row.CPUUsage,
			MemoryUsage:      row.MemoryUsage,
			PacketsProcessed: row.PacketsProcessed,
			ActiveSessions:   row.ActiveSessions,
			FlowsReported:    row.FlowsReported,
			ActivePolicies:   row.ActivePolicies,
		})
	}

	return agents, nil
}

// GetAgentByID retrieves a single agent by ID
func (s *AgentStorage) GetAgentByID(ctx context.Context, agentID string) (*Agent, error) {
	var row AgentWithMetrics

	err := s.db.NewSelect().
		ColumnExpr("a.agent_id, a.hostname, a.version, a.interface, a.ip_addresses").
		ColumnExpr("a.os, a.kernel_version, a.start_time, a.last_heartbeat, a.status").
		ColumnExpr("COALESCE(m.cpu_usage, 0) as cpu_usage").
		ColumnExpr("COALESCE(m.memory_usage, 0) as memory_usage").
		ColumnExpr("COALESCE(m.packets_processed, 0) as packets_processed").
		ColumnExpr("COALESCE(m.active_sessions, 0) as active_sessions").
		ColumnExpr("COALESCE(m.flows_reported, 0) as flows_reported").
		ColumnExpr("COALESCE(m.active_policies, 0) as active_policies").
		Table("agents").
		TableExpr("LEFT JOIN agent_metrics AS m ON a.agent_id = m.agent_id").
		Where("a.agent_id = ?", agentID).
		Scan(ctx, &row)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	return &Agent{
		AgentID:          row.AgentID,
		Hostname:         row.Hostname,
		Version:          row.Version,
		Interface:        row.Interface,
		IPAddresses:      row.IPAddresses,
		OS:               row.OS,
		KernelVersion:    row.KernelVersion,
		StartTime:        row.StartTime.UnixNano(),
		LastHeartbeat:    row.LastHeartbeat.UnixNano(),
		Status:           row.Status,
		CPUUsage:         row.CPUUsage,
		MemoryUsage:      row.MemoryUsage,
		PacketsProcessed: row.PacketsProcessed,
		ActiveSessions:   row.ActiveSessions,
		FlowsReported:    row.FlowsReported,
		ActivePolicies:   row.ActivePolicies,
	}, nil
}

// UpdateStatusReport persists a detailed status report from an agent
func (s *AgentStorage) UpdateStatusReport(ctx context.Context, report *agentpb.StatusReport) error {
	agentStatus := agentStatusToString(report.Status)

	metricsRow := &AgentMetricsRow{
		AgentID:       report.AgentId,
		PolicyCount:   report.PolicyCount,
		PolicyVersion: report.PolicyVersion,
		WorkloadCount: report.WorkloadCount,
		AgentStatus:   agentStatus,
		Uptime:        report.Uptime,
		Errors:        report.Errors,
		Metadata:      report.Metadata,
	}

	if report.Metrics != nil {
		metricsRow.CPUUsage = report.Metrics.CpuUsage
		metricsRow.MemoryUsage = report.Metrics.MemoryUsage
		metricsRow.PacketsProcessed = report.Metrics.PacketsProcessed
		metricsRow.ActiveSessions = report.Metrics.ActiveSessions
		metricsRow.FlowsReported = report.Metrics.FlowsReported
		metricsRow.ActivePolicies = report.Metrics.ActivePolicies
	}

	_, err := s.db.NewInsert().
		Model(metricsRow).
		On("CONFLICT (agent_id) DO UPDATE").
		Set("cpu_usage = EXCLUDED.cpu_usage").
		Set("memory_usage = EXCLUDED.memory_usage").
		Set("packets_processed = EXCLUDED.packets_processed").
		Set("active_sessions = EXCLUDED.active_sessions").
		Set("flows_reported = EXCLUDED.flows_reported").
		Set("active_policies = EXCLUDED.active_policies").
		Set("policy_count = EXCLUDED.policy_count").
		Set("policy_version = EXCLUDED.policy_version").
		Set("workload_count = EXCLUDED.workload_count").
		Set("agent_status = EXCLUDED.agent_status").
		Set("uptime = EXCLUDED.uptime").
		Set("errors = EXCLUDED.errors").
		Set("metadata = EXCLUDED.metadata").
		Set("last_status_report = CURRENT_TIMESTAMP").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update status report: %w", err)
	}

	newStatus := "active"
	if agentStatus == "error" || agentStatus == "degraded" {
		newStatus = "unhealthy"
	}

	_, err = s.db.NewUpdate().
		Model((*AgentRow)(nil)).
		Set("status = ?", newStatus).
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("agent_id = ?", report.AgentId).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// MarkAgentOffline marks an agent as inactive/offline with reason
func (s *AgentStorage) MarkAgentOffline(ctx context.Context, agentID, reason string) error {
	result, err := s.db.NewUpdate().
		Model((*AgentRow)(nil)).
		Set("status = 'inactive'").
		Set("offline_at = CURRENT_TIMESTAMP").
		Set("offline_reason = ?", reason).
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("agent_id = ?", agentID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark agent offline: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	return nil
}

// agentStatusToString converts AgentStatus enum to database string
func agentStatusToString(status agentpb.AgentStatus) string {
	switch status {
	case agentpb.AgentStatus_STATUS_STARTING:
		return "starting"
	case agentpb.AgentStatus_STATUS_RUNNING:
		return "running"
	case agentpb.AgentStatus_STATUS_DEGRADED:
		return "degraded"
	case agentpb.AgentStatus_STATUS_STOPPING:
		return "stopping"
	case agentpb.AgentStatus_STATUS_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

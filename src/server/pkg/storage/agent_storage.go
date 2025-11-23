package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
)

// AgentStorage handles agent data persistence
type AgentStorage struct {
	db *sql.DB
}

// NewAgentStorage creates a new AgentStorage
func NewAgentStorage(db *sql.DB) *AgentStorage {
	return &AgentStorage{db: db}
}

// RegisterAgent registers or updates an agent
func (s *AgentStorage) RegisterAgent(ctx context.Context, req *agentpb.RegisterRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (agent_id, hostname, version, interface, ip_addresses, os, kernel_version, start_time, last_heartbeat, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, to_timestamp($8/1000000000.0), CURRENT_TIMESTAMP, 'active')
		ON CONFLICT (agent_id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			version = EXCLUDED.version,
			interface = EXCLUDED.interface,
			ip_addresses = EXCLUDED.ip_addresses,
			os = EXCLUDED.os,
			kernel_version = EXCLUDED.kernel_version,
			start_time = EXCLUDED.start_time,
			last_heartbeat = CURRENT_TIMESTAMP,
			status = 'active',
			updated_at = CURRENT_TIMESTAMP
	`, req.AgentId, req.Hostname, req.Version, req.Interface,
		pq.Array(req.IpAddresses), req.Os, req.KernelVersion, req.StartTime)

	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	return nil
}

// UpdateHeartbeat updates agent last heartbeat timestamp
func (s *AgentStorage) UpdateHeartbeat(ctx context.Context, agentID string, metrics *agentpb.AgentMetrics) error {
	// Update agent heartbeat
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents
		SET last_heartbeat = CURRENT_TIMESTAMP, status = 'active', updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = $1
	`, agentID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	// Update metrics
	if metrics != nil {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO agent_metrics (agent_id, cpu_usage, memory_usage, packets_processed, active_sessions, flows_reported, active_policies)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (agent_id) DO UPDATE SET
				cpu_usage = EXCLUDED.cpu_usage,
				memory_usage = EXCLUDED.memory_usage,
				packets_processed = EXCLUDED.packets_processed,
				active_sessions = EXCLUDED.active_sessions,
				flows_reported = EXCLUDED.flows_reported,
				active_policies = EXCLUDED.active_policies,
				updated_at = CURRENT_TIMESTAMP
		`, agentID, metrics.CpuUsage, metrics.MemoryUsage, metrics.PacketsProcessed,
			metrics.ActiveSessions, metrics.FlowsReported, metrics.ActivePolicies)
		if err != nil {
			return fmt.Errorf("failed to update agent metrics: %w", err)
		}
	}

	return nil
}

// GetAllAgents retrieves all registered agents
func (s *AgentStorage) GetAllAgents(ctx context.Context) ([]*Agent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.agent_id, a.hostname, a.version, a.interface, a.ip_addresses, a.os, a.kernel_version,
		       FLOOR(EXTRACT(EPOCH FROM a.start_time)*1000000000)::bigint as start_time,
		       FLOOR(EXTRACT(EPOCH FROM a.last_heartbeat)*1000000000)::bigint as last_heartbeat,
		       a.status,
		       COALESCE(m.cpu_usage, 0), COALESCE(m.memory_usage, 0),
		       COALESCE(m.packets_processed, 0), COALESCE(m.active_sessions, 0),
		       COALESCE(m.flows_reported, 0), COALESCE(m.active_policies, 0)
		FROM agents a
		LEFT JOIN agent_metrics m ON a.agent_id = m.agent_id
		ORDER BY a.last_heartbeat DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	agents := []*Agent{}
	for rows.Next() {
		var agent Agent
		var ipAddresses pq.StringArray

		err := rows.Scan(
			&agent.AgentID,
			&agent.Hostname,
			&agent.Version,
			&agent.Interface,
			&ipAddresses,
			&agent.OS,
			&agent.KernelVersion,
			&agent.StartTime,
			&agent.LastHeartbeat,
			&agent.Status,
			&agent.CPUUsage,
			&agent.MemoryUsage,
			&agent.PacketsProcessed,
			&agent.ActiveSessions,
			&agent.FlowsReported,
			&agent.ActivePolicies,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}

		agent.IPAddresses = []string(ipAddresses)
		agents = append(agents, &agent)
	}

	return agents, nil
}

// GetAgentByID retrieves a single agent by ID
func (s *AgentStorage) GetAgentByID(ctx context.Context, agentID string) (*Agent, error) {
	var agent Agent
	var ipAddresses pq.StringArray

	err := s.db.QueryRowContext(ctx, `
		SELECT a.agent_id, a.hostname, a.version, a.interface, a.ip_addresses, a.os, a.kernel_version,
		       FLOOR(EXTRACT(EPOCH FROM a.start_time)*1000000000)::bigint as start_time,
		       FLOOR(EXTRACT(EPOCH FROM a.last_heartbeat)*1000000000)::bigint as last_heartbeat,
		       a.status,
		       COALESCE(m.cpu_usage, 0), COALESCE(m.memory_usage, 0),
		       COALESCE(m.packets_processed, 0), COALESCE(m.active_sessions, 0),
		       COALESCE(m.flows_reported, 0), COALESCE(m.active_policies, 0)
		FROM agents a
		LEFT JOIN agent_metrics m ON a.agent_id = m.agent_id
		WHERE a.agent_id = $1
	`, agentID).Scan(
		&agent.AgentID,
		&agent.Hostname,
		&agent.Version,
		&agent.Interface,
		&ipAddresses,
		&agent.OS,
		&agent.KernelVersion,
		&agent.StartTime,
		&agent.LastHeartbeat,
		&agent.Status,
		&agent.CPUUsage,
		&agent.MemoryUsage,
		&agent.PacketsProcessed,
		&agent.ActiveSessions,
		&agent.FlowsReported,
		&agent.ActivePolicies,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	agent.IPAddresses = []string(ipAddresses)
	return &agent, nil
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

// UpdateStatusReport persists a detailed status report from an agent
func (s *AgentStorage) UpdateStatusReport(ctx context.Context, report *agentpb.StatusReport) error {
	// Convert metadata map to JSON
	var metadataJSON []byte
	var err error
	if report.Metadata != nil && len(report.Metadata) > 0 {
		metadataJSON, err = json.Marshal(report.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Convert agent status enum to string
	agentStatus := agentStatusToString(report.Status)

	// Update agent_metrics with status report data
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO agent_metrics (
			agent_id, cpu_usage, memory_usage, packets_processed, active_sessions,
			flows_reported, active_policies, policy_count, policy_version,
			workload_count, agent_status, uptime, errors, metadata, last_status_report
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, CURRENT_TIMESTAMP)
		ON CONFLICT (agent_id) DO UPDATE SET
			cpu_usage = EXCLUDED.cpu_usage,
			memory_usage = EXCLUDED.memory_usage,
			packets_processed = EXCLUDED.packets_processed,
			active_sessions = EXCLUDED.active_sessions,
			flows_reported = EXCLUDED.flows_reported,
			active_policies = EXCLUDED.active_policies,
			policy_count = EXCLUDED.policy_count,
			policy_version = EXCLUDED.policy_version,
			workload_count = EXCLUDED.workload_count,
			agent_status = EXCLUDED.agent_status,
			uptime = EXCLUDED.uptime,
			errors = EXCLUDED.errors,
			metadata = EXCLUDED.metadata,
			last_status_report = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`,
		report.AgentId,
		getMetricValue(report.Metrics, func(m *agentpb.AgentMetrics) float32 { return m.CpuUsage }),
		getMetricValueUint64(report.Metrics, func(m *agentpb.AgentMetrics) uint64 { return m.MemoryUsage }),
		getMetricValueUint64(report.Metrics, func(m *agentpb.AgentMetrics) uint64 { return m.PacketsProcessed }),
		getMetricValueUint32(report.Metrics, func(m *agentpb.AgentMetrics) uint32 { return m.ActiveSessions }),
		getMetricValueUint64(report.Metrics, func(m *agentpb.AgentMetrics) uint64 { return m.FlowsReported }),
		getMetricValueUint32(report.Metrics, func(m *agentpb.AgentMetrics) uint32 { return m.ActivePolicies }),
		report.PolicyCount,
		report.PolicyVersion,
		report.WorkloadCount,
		agentStatus,
		report.Uptime,
		pq.Array(report.Errors),
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update status report: %w", err)
	}

	// Also update agents table status
	_, err = s.db.ExecContext(ctx, `
		UPDATE agents
		SET status = CASE
			WHEN $2 = 'error' THEN 'unhealthy'
			WHEN $2 = 'degraded' THEN 'unhealthy'
			ELSE 'active'
		END,
		updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = $1
	`, report.AgentId, agentStatus)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
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

// Helper functions to safely extract metric values
func getMetricValue(m *agentpb.AgentMetrics, f func(*agentpb.AgentMetrics) float32) float32 {
	if m == nil {
		return 0
	}
	return f(m)
}

func getMetricValueUint64(m *agentpb.AgentMetrics, f func(*agentpb.AgentMetrics) uint64) uint64 {
	if m == nil {
		return 0
	}
	return f(m)
}

func getMetricValueUint32(m *agentpb.AgentMetrics, f func(*agentpb.AgentMetrics) uint32) uint32 {
	if m == nil {
		return 0
	}
	return f(m)
}

// MarkAgentOffline marks an agent as inactive/offline with reason
func (s *AgentStorage) MarkAgentOffline(ctx context.Context, agentID, reason string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE agents
		SET status = 'inactive',
		    offline_at = CURRENT_TIMESTAMP,
		    offline_reason = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = $1
	`, agentID, reason)
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

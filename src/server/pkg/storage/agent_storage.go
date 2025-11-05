package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	agentpb "github.com/ebpf-microsegment/src/proto/agent"
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
		       EXTRACT(EPOCH FROM a.start_time)*1000000000 as start_time,
		       EXTRACT(EPOCH FROM a.last_heartbeat)*1000000000 as last_heartbeat,
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

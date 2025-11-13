package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentStorage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)
	assert.NotNil(t, storage)
	assert.Equal(t, db, storage.db)
}

func TestRegisterAgent_NewAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	req := &agentpb.RegisterRequest{
		AgentId:       "agent-test-1",
		Hostname:      "test-host",
		Version:       "1.0.0",
		Interface:     "eth0",
		IpAddresses:   []string{"10.0.0.1", "192.168.1.100"},
		Os:            "Linux",
		KernelVersion: "5.10.0",
		StartTime:     int64(time.Now().UnixNano()),
	}

	// 期望插入或更新agent
	mock.ExpectExec("INSERT INTO agents").
		WithArgs(req.AgentId, req.Hostname, req.Version, req.Interface,
			pq.Array(req.IpAddresses), req.Os, req.KernelVersion, req.StartTime).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storage.RegisterAgent(context.Background(), req)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRegisterAgent_UpdateExisting(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	req := &agentpb.RegisterRequest{
		AgentId:       "agent-test-1",
		Hostname:      "test-host-updated",
		Version:       "1.1.0",
		Interface:     "eth0",
		IpAddresses:   []string{"10.0.0.1"},
		Os:            "Linux",
		KernelVersion: "5.11.0",
		StartTime:     int64(time.Now().UnixNano()),
	}

	// 期望执行 UPSERT（INSERT ... ON CONFLICT DO UPDATE）
	mock.ExpectExec("INSERT INTO agents").
		WithArgs(req.AgentId, req.Hostname, req.Version, req.Interface,
			pq.Array(req.IpAddresses), req.Os, req.KernelVersion, req.StartTime).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.RegisterAgent(context.Background(), req)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRegisterAgent_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	req := &agentpb.RegisterRequest{
		AgentId:  "agent-test-1",
		Hostname: "test-host",
		Version:  "1.0.0",
	}

	mock.ExpectExec("INSERT INTO agents").
		WillReturnError(sql.ErrConnDone)

	err = storage.RegisterAgent(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register agent")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdateHeartbeat_WithoutMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"

	// 期望更新heartbeat
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat.*WHERE agent_id = .*").
		WithArgs(agentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.UpdateHeartbeat(context.Background(), agentID, nil)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdateHeartbeat_WithMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"
	metrics := &agentpb.AgentMetrics{
		CpuUsage:         25.5,
		MemoryUsage:      1024 * 1024 * 512,
		PacketsProcessed: 10000,
		ActiveSessions:   50,
		FlowsReported:    500,
		ActivePolicies:   10,
	}

	// 期望更新heartbeat
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat.*WHERE agent_id = .*").
		WithArgs(agentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 期望插入或更新metrics
	mock.ExpectExec("INSERT INTO agent_metrics").
		WithArgs(agentID, metrics.CpuUsage, metrics.MemoryUsage, metrics.PacketsProcessed,
			metrics.ActiveSessions, metrics.FlowsReported, metrics.ActivePolicies).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storage.UpdateHeartbeat(context.Background(), agentID, metrics)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdateHeartbeat_HeartbeatError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"

	mock.ExpectExec("UPDATE agents.*SET last_heartbeat.*WHERE agent_id = .*").
		WillReturnError(sql.ErrConnDone)

	err = storage.UpdateHeartbeat(context.Background(), agentID, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update heartbeat")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdateHeartbeat_MetricsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"
	metrics := &agentpb.AgentMetrics{
		CpuUsage:    25.5,
		MemoryUsage: 1024 * 1024 * 512,
	}

	// heartbeat更新成功
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat.*WHERE agent_id = .*").
		WithArgs(agentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// metrics更新失败
	mock.ExpectExec("INSERT INTO agent_metrics").
		WillReturnError(sql.ErrConnDone)

	err = storage.UpdateHeartbeat(context.Background(), agentID, metrics)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update agent metrics")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllAgents_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	// 创建mock返回数据
	rows := sqlmock.NewRows([]string{
		"agent_id", "hostname", "version", "interface", "ip_addresses", "os", "kernel_version",
		"start_time", "last_heartbeat", "status",
		"cpu_usage", "memory_usage", "packets_processed", "active_sessions",
		"flows_reported", "active_policies",
	}).
		AddRow("agent-1", "host-1", "1.0.0", "eth0", pq.Array([]string{"10.0.0.1"}),
			"Linux", "5.10.0", int64(1234567890), int64(1234567900), "active",
			float32(25.5), uint64(1024*1024*512), uint64(10000), uint32(50),
			uint64(500), uint32(10)).
		AddRow("agent-2", "host-2", "1.0.1", "eth1", pq.Array([]string{"10.0.0.2", "192.168.1.100"}),
			"Linux", "5.11.0", int64(1234567891), int64(1234567901), "active",
			float32(30.0), uint64(1024*1024*1024), uint64(20000), uint32(100),
			uint64(1000), uint32(15))

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*LEFT JOIN agent_metrics m.*ORDER BY a.last_heartbeat DESC").
		WillReturnRows(rows)

	agents, err := storage.GetAllAgents(context.Background())
	assert.NoError(t, err)
	assert.Len(t, agents, 2)
	assert.Equal(t, "agent-1", agents[0].AgentID)
	assert.Equal(t, "host-1", agents[0].Hostname)
	assert.Equal(t, []string{"10.0.0.1"}, agents[0].IPAddresses)
	assert.Equal(t, float32(25.5), agents[0].CPUUsage)
	assert.Equal(t, "agent-2", agents[1].AgentID)
	assert.Len(t, agents[1].IPAddresses, 2)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllAgents_EmptyResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	rows := sqlmock.NewRows([]string{
		"agent_id", "hostname", "version", "interface", "ip_addresses", "os", "kernel_version",
		"start_time", "last_heartbeat", "status",
		"cpu_usage", "memory_usage", "packets_processed", "active_sessions",
		"flows_reported", "active_policies",
	})

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*").
		WillReturnRows(rows)

	agents, err := storage.GetAllAgents(context.Background())
	assert.NoError(t, err)
	assert.Len(t, agents, 0)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllAgents_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*").
		WillReturnError(sql.ErrConnDone)

	agents, err := storage.GetAllAgents(context.Background())
	assert.Error(t, err)
	assert.Nil(t, agents)
	assert.Contains(t, err.Error(), "failed to query agents")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAgentByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"

	rows := sqlmock.NewRows([]string{
		"agent_id", "hostname", "version", "interface", "ip_addresses", "os", "kernel_version",
		"start_time", "last_heartbeat", "status",
		"cpu_usage", "memory_usage", "packets_processed", "active_sessions",
		"flows_reported", "active_policies",
	}).AddRow(agentID, "test-host", "1.0.0", "eth0", pq.Array([]string{"10.0.0.1"}),
		"Linux", "5.10.0", int64(1234567890), int64(1234567900), "active",
		float32(25.5), uint64(1024*1024*512), uint64(10000), uint32(50),
		uint64(500), uint32(10))

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*LEFT JOIN agent_metrics m.*WHERE a.agent_id = .*").
		WithArgs(agentID).
		WillReturnRows(rows)

	agent, err := storage.GetAgentByID(context.Background(), agentID)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, agentID, agent.AgentID)
	assert.Equal(t, "test-host", agent.Hostname)
	assert.Equal(t, "1.0.0", agent.Version)
	assert.Equal(t, []string{"10.0.0.1"}, agent.IPAddresses)
	assert.Equal(t, float32(25.5), agent.CPUUsage)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAgentByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "non-existent-agent"

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*LEFT JOIN agent_metrics m.*WHERE a.agent_id = .*").
		WithArgs(agentID).
		WillReturnError(sql.ErrNoRows)

	agent, err := storage.GetAgentByID(context.Background(), agentID)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "agent not found")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAgentByID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewAgentStorage(db)

	agentID := "agent-test-1"

	mock.ExpectQuery("SELECT a.agent_id.*FROM agents a.*LEFT JOIN agent_metrics m.*WHERE a.agent_id = .*").
		WithArgs(agentID).
		WillReturnError(sql.ErrConnDone)

	agent, err := storage.GetAgentByID(context.Background(), agentID)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "failed to query agent")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

package grpc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	agentpb "github.com/ebpf-microsegment/src/proto/agent"
	"github.com/ebpf-microsegment/src/server/pkg/storage"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAgentServiceServer tests the constructor
func TestNewAgentServiceServer(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	assert.NotNil(t, server)
	assert.NotNil(t, server.agentStorage)
}

// TestRegisterAgent_Success tests successful agent registration
func TestRegisterAgent_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.RegisterRequest{
		AgentId:       "agent-test-1",
		Hostname:      "test-host",
		Version:       "1.0.0",
		Interface:     "eth0",
		IpAddresses:   []string{"10.0.0.1", "192.168.1.100"},
		Os:            "Linux",
		KernelVersion: "5.10.0",
		StartTime:     time.Now().UnixNano(),
	}

	// Expect agent registration in database
	mock.ExpectExec("INSERT INTO agents").
		WithArgs(req.AgentId, req.Hostname, req.Version, req.Interface,
			pq.Array(req.IpAddresses), req.Os, req.KernelVersion, req.StartTime).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := server.RegisterAgent(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Agent registered successfully", resp.Message)
	assert.Equal(t, "1.0.0-mvp", resp.ServerVersion)
	assert.Greater(t, resp.ServerTime, int64(0))
	assert.NotNil(t, resp.Config)
	assert.Equal(t, uint32(30), resp.Config.HeartbeatInterval)
	assert.Equal(t, uint32(60), resp.Config.StatsInterval)
	assert.Equal(t, uint32(100), resp.Config.FlowBatchSize)
	assert.Equal(t, uint32(5), resp.Config.FlowBatchTimeout)
	assert.False(t, resp.Config.DebugMode)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestRegisterAgent_StorageError tests handling storage errors
func TestRegisterAgent_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.RegisterRequest{
		AgentId:  "agent-test-1",
		Hostname: "test-host",
		Version:  "1.0.0",
	}

	// Expect storage error
	mock.ExpectExec("INSERT INTO agents").
		WillReturnError(sql.ErrConnDone)

	resp, err := server.RegisterAgent(context.Background(), req)
	require.NoError(t, err) // gRPC method succeeds, error is in response
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "Registration failed")
}

// TestHeartbeat_Success tests successful heartbeat processing
func TestHeartbeat_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.HeartbeatRequest{
		AgentId: "agent-test-1",
		Metrics: &agentpb.AgentMetrics{
			CpuUsage:         25.5,
			MemoryUsage:      1024 * 1024 * 512,
			PacketsProcessed: 10000,
			ActiveSessions:   50,
			FlowsReported:    500,
			ActivePolicies:   10,
		},
	}

	// Expect heartbeat update
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat").
		WithArgs(req.AgentId).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect metrics update
	mock.ExpectExec("INSERT INTO agent_metrics").
		WithArgs(req.AgentId, req.Metrics.CpuUsage, req.Metrics.MemoryUsage,
			req.Metrics.PacketsProcessed, req.Metrics.ActiveSessions,
			req.Metrics.FlowsReported, req.Metrics.ActivePolicies).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp, err := server.Heartbeat(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Healthy)
	assert.Equal(t, "Heartbeat received", resp.Message)
	assert.Greater(t, resp.ServerTime, int64(0))
	assert.NotNil(t, resp.Commands)
	assert.Len(t, resp.Commands, 0) // MVP returns no commands

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestHeartbeat_WithoutMetrics tests heartbeat without metrics
func TestHeartbeat_WithoutMetrics(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.HeartbeatRequest{
		AgentId: "agent-test-1",
		Metrics: nil, // No metrics
	}

	// Expect only heartbeat update (no metrics update)
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat").
		WithArgs(req.AgentId).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := server.Heartbeat(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Healthy)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestHeartbeat_StorageError tests handling storage errors in heartbeat
func TestHeartbeat_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.HeartbeatRequest{
		AgentId: "agent-test-1",
	}

	// Expect storage error
	mock.ExpectExec("UPDATE agents.*SET last_heartbeat").
		WillReturnError(sql.ErrConnDone)

	resp, err := server.Heartbeat(context.Background(), req)
	require.NoError(t, err) // gRPC method succeeds, error is in response
	assert.NotNil(t, resp)
	assert.False(t, resp.Healthy)
	assert.Contains(t, resp.Message, "Heartbeat processing failed")
}

// TestReportStatus_Success tests successful status reporting
func TestReportStatus_Success(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	report := &agentpb.StatusReport{
		AgentId:   "agent-test-1",
		Timestamp: time.Now().UnixNano(),
		Status:    agentpb.AgentStatus_STATUS_RUNNING,
		Uptime:    3600, // 1 hour
		Metrics: &agentpb.AgentMetrics{
			CpuUsage:         15.0,
			MemoryUsage:      1024 * 1024 * 256,
			PacketsProcessed: 5000,
			ActiveSessions:   20,
			FlowsReported:    200,
			ActivePolicies:   5,
		},
		PolicyCount:   5,
		PolicyVersion: 1,
		WorkloadCount: 10,
		Errors:        []string{"Temporary connection loss", "High memory usage"},
	}

	// Note: MVP implementation just acknowledges receipt
	resp, err := server.ReportStatus(context.Background(), report)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Status report received", resp.Message)
	assert.NotNil(t, resp.Commands)
	assert.Len(t, resp.Commands, 0) // MVP returns no commands
}

// TestReportStatus_EmptyReport tests handling empty status report
func TestReportStatus_EmptyReport(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	report := &agentpb.StatusReport{
		AgentId:   "agent-test-1",
		Timestamp: time.Now().UnixNano(),
		Status:    agentpb.AgentStatus_STATUS_UNKNOWN,
		Uptime:    0,
	}

	resp, err := server.ReportStatus(context.Background(), report)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

// TestUnregisterAgent_Success tests successful agent unregistration
func TestUnregisterAgent_Success(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.UnregisterRequest{
		AgentId: "agent-test-1",
		Reason:  "Graceful shutdown",
	}

	// Note: MVP implementation just acknowledges, no database update
	resp, err := server.UnregisterAgent(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Agent unregistered", resp.Message)
}

// TestUnregisterAgent_EmptyReason tests unregistration without reason
func TestUnregisterAgent_EmptyReason(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	agentStorage := storage.NewAgentStorage(db)
	server := NewAgentServiceServer(agentStorage)

	req := &agentpb.UnregisterRequest{
		AgentId: "agent-test-1",
		Reason:  "",
	}

	resp, err := server.UnregisterAgent(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

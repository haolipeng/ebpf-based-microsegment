package grpc

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// intToIP converts a uint32 IP address to string format (for testing)
func intToIP(ipUint32 uint32) string {
	ip := net.IPv4(
		byte(ipUint32>>24),
		byte(ipUint32>>16),
		byte(ipUint32>>8),
		byte(ipUint32),
	)
	return ip.String()
}

// Suppress unused import warning
var _ = fmt.Sprint

// mockReportFlowEventsStream is a mock implementation of the gRPC stream
type mockReportFlowEventsStream struct {
	events   []*flowpb.FlowEvent
	index    int
	response *commonpb.ReportResponse
	ctx      context.Context
}

func (m *mockReportFlowEventsStream) Recv() (*flowpb.FlowEvent, error) {
	if m.index >= len(m.events) {
		return nil, io.EOF
	}
	event := m.events[m.index]
	m.index++
	return event, nil
}

func (m *mockReportFlowEventsStream) SendAndClose(resp *commonpb.ReportResponse) error {
	m.response = resp
	return nil
}

func (m *mockReportFlowEventsStream) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockReportFlowEventsStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockReportFlowEventsStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockReportFlowEventsStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockReportFlowEventsStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockReportFlowEventsStream) SetTrailer(md metadata.MD) {
}

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

// TestNewFlowServiceServer tests the constructor
func TestNewFlowServiceServer(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	assert.NotNil(t, server)
	assert.NotNil(t, server.flowStorage)
}

// TestReportFlowEvents_SingleEvent tests receiving a single flow event
func TestReportFlowEvents_SingleEvent(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	event := &flowpb.FlowEvent{
		TimestampNs:  uint64(time.Now().UnixNano()),
		SrcIp:        0xC0A80101, // 192.168.1.1
		DstIp:        0xC0A80102, // 192.168.1.2
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     6, // TCP
		Direction:    1,
		PacketCount:  10,
		ByteCount:    1024,
		PolicyId:     1,
		PolicyAction: 1,
		State:        1,
		AgentId:      "test-agent-1",
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "db"},
	}

	mockStream := &mockReportFlowEventsStream{
		events: []*flowpb.FlowEvent{event},
	}

	// Expect database operations for saving the event (bun uses INSERT ... RETURNING)
	returnRows := sqlmock.NewRows([]string{
		"id", "start_time", "end_time", "last_seen", "created_at",
		"src_pid", "src_ppid", "src_uid", "src_gid", "src_comm",
		"src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	}).AddRow(1, nil, nil, nil, time.Now(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("INSERT INTO \"flows\"").
		WillReturnRows(returnRows)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err)
	assert.NotNil(t, mockStream.response)
	assert.True(t, mockStream.response.Success)
	assert.Equal(t, uint32(1), mockStream.response.AcceptedCount)
	assert.Equal(t, uint32(0), mockStream.response.RejectedCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestReportFlowEvents_MultipleEvents tests receiving multiple flow events
func TestReportFlowEvents_MultipleEvents(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	events := []*flowpb.FlowEvent{
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80101,
			DstIp:       0xC0A80102,
			SrcPort:     12345,
			DstPort:     80,
			Protocol:    6,
			AgentId:     "agent-1",
		},
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80103,
			DstIp:       0xC0A80104,
			SrcPort:     54321,
			DstPort:     443,
			Protocol:    6,
			AgentId:     "agent-1",
		},
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80105,
			DstIp:       0xC0A80106,
			SrcPort:     9999,
			DstPort:     8080,
			Protocol:    6,
			AgentId:     "agent-1",
		},
	}

	mockStream := &mockReportFlowEventsStream{
		events: events,
	}

	// Expect database operations for batch saving (bun uses INSERT ... RETURNING)
	returnRows := sqlmock.NewRows([]string{
		"id", "policy_id", "policy_action", "state", "start_time", "end_time",
		"last_seen", "created_at", "src_pid", "src_ppid", "src_uid", "src_gid",
		"src_comm", "src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	})
	for i := range events {
		returnRows.AddRow(int64(i+1), nil, nil, nil, nil, nil, nil, time.Now(),
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}

	mock.ExpectQuery("INSERT INTO \"flows\"").
		WillReturnRows(returnRows)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err)
	assert.NotNil(t, mockStream.response)
	assert.True(t, mockStream.response.Success)
	assert.Equal(t, uint32(3), mockStream.response.AcceptedCount)
	assert.Equal(t, uint32(0), mockStream.response.RejectedCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestReportFlowEvents_RejectsInvalidEvent tests event validation
func TestReportFlowEvents_RejectsInvalidEvent(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	events := []*flowpb.FlowEvent{
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80101,
			DstIp:       0xC0A80102,
			AgentId:     "agent-1", // Valid
		},
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80103,
			DstIp:       0xC0A80104,
			AgentId:     "", // Invalid: missing agent_id
		},
		{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       0xC0A80105,
			DstIp:       0xC0A80106,
			AgentId:     "agent-2", // Valid
		},
	}

	mockStream := &mockReportFlowEventsStream{
		events: events,
	}

	// Only 2 events should be saved (1 rejected due to missing agent_id)
	// bun uses INSERT ... RETURNING for batch inserts
	returnRows := sqlmock.NewRows([]string{
		"id", "policy_id", "policy_action", "state", "start_time", "end_time",
		"last_seen", "created_at", "src_pid", "src_ppid", "src_uid", "src_gid",
		"src_comm", "src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	}).
		AddRow(int64(1), nil, nil, nil, nil, nil, nil, time.Now(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
		AddRow(int64(2), nil, nil, nil, nil, nil, nil, time.Now(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("INSERT INTO \"flows\"").
		WillReturnRows(returnRows)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err)
	assert.NotNil(t, mockStream.response)
	assert.True(t, mockStream.response.Success)
	assert.Equal(t, uint32(2), mockStream.response.AcceptedCount)
	assert.Equal(t, uint32(1), mockStream.response.RejectedCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestReportFlowEvents_StorageError tests handling storage errors
func TestReportFlowEvents_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	event := &flowpb.FlowEvent{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0xC0A80101,
		DstIp:       0xC0A80102,
		AgentId:     "agent-1",
	}

	mockStream := &mockReportFlowEventsStream{
		events: []*flowpb.FlowEvent{event},
	}

	// Storage returns an error (bun uses INSERT ... RETURNING which is a Query)
	mock.ExpectQuery("INSERT INTO \"flows\"").
		WillReturnError(sql.ErrConnDone)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err) // The gRPC method itself succeeds, error is in response
	assert.NotNil(t, mockStream.response)
	assert.False(t, mockStream.response.Success)
	assert.Equal(t, uint32(1), mockStream.response.RejectedCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestReportFlowEvents_EmptyStream tests handling empty stream
func TestReportFlowEvents_EmptyStream(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	mockStream := &mockReportFlowEventsStream{
		events: []*flowpb.FlowEvent{}, // Empty
	}

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err)
	assert.NotNil(t, mockStream.response)
	assert.True(t, mockStream.response.Success)
	assert.Equal(t, uint32(0), mockStream.response.AcceptedCount)
	assert.Equal(t, uint32(0), mockStream.response.RejectedCount)

	// No database operations expected for empty stream
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestQueryFlows_Success tests successful flow query
func TestQueryFlows_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	query := &flowpb.FlowQuery{
		TimeRange: &commonpb.TimeRange{
			StartTime: time.Now().Add(-1 * time.Hour).UnixNano(),
			EndTime:   time.Now().UnixNano(),
		},
		Limit:  10,
		Offset: 0,
	}

	// Mock COUNT query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(
		uint64(1), uint64(time.Now().UnixNano()), "192.168.1.1", "192.168.1.2",
		uint32(12345), uint32(80), uint32(6), uint32(1),
		uint64(10), uint64(1024), uint32(1), uint32(1), uint32(1),
		"agent-1", []byte("{}"), []byte("{}"),
	)
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	resp, err := server.QueryFlows(context.Background(), query)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(1), resp.TotalCount)
	assert.Len(t, resp.Flows, 1)
	assert.Equal(t, uint64(1), resp.Flows[0].Id)
	assert.False(t, resp.HasMore) // offset(0) + limit(10) >= total(1)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestQueryFlows_WithPagination tests pagination logic
func TestQueryFlows_WithPagination(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	query := &flowpb.FlowQuery{
		Limit:  10,
		Offset: 0,
	}

	// Mock COUNT query returning 100 total
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock SELECT query returning empty (we only care about pagination logic)
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count",
		"policy_id", "policy_action", "state", "agent_id", "source_labels", "dest_labels",
	})
	mock.ExpectQuery("SELECT (.+) FROM flows").
		WillReturnRows(rows)

	resp, err := server.QueryFlows(context.Background(), query)
	require.NoError(t, err)
	assert.True(t, resp.HasMore) // offset(0) + limit(10) < total(100)
	assert.Equal(t, uint64(100), resp.TotalCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestQueryFlows_StorageError tests handling storage errors
func TestQueryFlows_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	query := &flowpb.FlowQuery{
		Limit: 10,
	}

	// Mock query error
	mock.ExpectQuery("SELECT COUNT").
		WillReturnError(sql.ErrConnDone)

	resp, err := server.QueryFlows(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to query flows")
}

// TestGetFlowSummary_ReturnsBasicStats tests flow summary aggregation
func TestGetFlowSummary_ReturnsBasicStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	req := &flowpb.FlowSummaryRequest{
		TimeRange: &commonpb.TimeRange{
			StartTime: time.Now().Add(-1 * time.Hour).UnixNano(),
			EndTime:   time.Now().UnixNano(),
		},
	}

	// Mock the main aggregation query
	summaryRows := sqlmock.NewRows([]string{
		"total_flows", "active_flows", "closed_flows", "total_packets", "total_bytes",
		"allowed_flows", "denied_flows", "unique_source_ips", "unique_dest_ips", "avg_duration_ms",
	}).AddRow(100, 30, 70, 5000, 1024000, 90, 10, 15, 20, 150.5)

	mock.ExpectQuery("SELECT").WillReturnRows(summaryRows)

	// Mock the protocol stats query
	protoRows := sqlmock.NewRows([]string{"protocol", "count", "bytes"}).
		AddRow("TCP", 80, 800000).
		AddRow("UDP", 20, 224000)

	mock.ExpectQuery("SELECT").WillReturnRows(protoRows)

	resp, err := server.GetFlowSummary(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(100), resp.TotalFlows)
	assert.Equal(t, uint64(5000), resp.TotalPackets)
	assert.Equal(t, uint64(1024000), resp.TotalBytes)
}

// TestReportFlowEvents_BatchProcessing tests that events are processed in batches
// to prevent memory exhaustion when receiving large volumes of events.
func TestReportFlowEvents_BatchProcessing(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorage(db)
	server := NewFlowServiceServer(flowStorage)

	// Create events exceeding the batch size (maxFlowBatchSize = 1000)
	// We'll create 2500 events to trigger 3 batches (1000 + 1000 + 500)
	numEvents := 2500
	events := make([]*flowpb.FlowEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &flowpb.FlowEvent{
			TimestampNs: uint64(time.Now().UnixNano()),
			SrcIp:       uint32(0xC0A80001 + i), // 192.168.0.1 + i
			DstIp:       0xC0A80102,             // 192.168.1.2
			SrcPort:     uint32(10000 + i),
			DstPort:     80,
			Protocol:    6,
			AgentId:     "agent-batch-test",
		}
	}

	mockStream := &mockReportFlowEventsStream{
		events: events,
	}

	// Expect 3 batch inserts (1000 + 1000 + 500)
	// First batch (1000 events)
	returnRows1 := sqlmock.NewRows([]string{
		"id", "policy_id", "policy_action", "state", "start_time", "end_time",
		"last_seen", "created_at", "src_pid", "src_ppid", "src_uid", "src_gid",
		"src_comm", "src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	})
	for i := 0; i < 1000; i++ {
		returnRows1.AddRow(int64(i+1), nil, nil, nil, nil, nil, nil, time.Now(),
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}
	mock.ExpectQuery("INSERT INTO \"flows\"").WillReturnRows(returnRows1)

	// Second batch (1000 events)
	returnRows2 := sqlmock.NewRows([]string{
		"id", "policy_id", "policy_action", "state", "start_time", "end_time",
		"last_seen", "created_at", "src_pid", "src_ppid", "src_uid", "src_gid",
		"src_comm", "src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	})
	for i := 0; i < 1000; i++ {
		returnRows2.AddRow(int64(1001+i), nil, nil, nil, nil, nil, nil, time.Now(),
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}
	mock.ExpectQuery("INSERT INTO \"flows\"").WillReturnRows(returnRows2)

	// Third batch (500 remaining events)
	returnRows3 := sqlmock.NewRows([]string{
		"id", "policy_id", "policy_action", "state", "start_time", "end_time",
		"last_seen", "created_at", "src_pid", "src_ppid", "src_uid", "src_gid",
		"src_comm", "src_exe_path", "src_cmdline", "src_container_id", "dst_pid", "dst_comm",
	})
	for i := 0; i < 500; i++ {
		returnRows3.AddRow(int64(2001+i), nil, nil, nil, nil, nil, nil, time.Now(),
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}
	mock.ExpectQuery("INSERT INTO \"flows\"").WillReturnRows(returnRows3)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err)
	assert.NotNil(t, mockStream.response)
	assert.True(t, mockStream.response.Success)
	assert.Equal(t, uint32(numEvents), mockStream.response.AcceptedCount)
	assert.Equal(t, uint32(0), mockStream.response.RejectedCount)

	// Verify all 3 batches were executed
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestIntToIP tests IP conversion utility
func TestIntToIP(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected string
	}{
		{
			name:     "192.168.1.1",
			input:    0xC0A80101,
			expected: "192.168.1.1",
		},
		{
			name:     "10.0.0.1",
			input:    0x0A000001,
			expected: "10.0.0.1",
		},
		{
			name:     "255.255.255.255",
			input:    0xFFFFFFFF,
			expected: "255.255.255.255",
		},
		{
			name:     "0.0.0.0",
			input:    0x00000000,
			expected: "0.0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intToIP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}


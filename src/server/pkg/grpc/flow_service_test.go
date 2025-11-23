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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.flowStorage)
	assert.Nil(t, server.wsHub)
}

// TestReportFlowEvents_SingleEvent tests receiving a single flow event
func TestReportFlowEvents_SingleEvent(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	// Expect database operations for saving the event
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")
	mock.ExpectExec("INSERT INTO flows").
		WithArgs(event.TimestampNs, "192.168.1.1", "192.168.1.2",
			event.SrcPort, event.DstPort, event.Protocol,
			event.Direction, event.PacketCount, event.ByteCount,
			event.PolicyId, event.PolicyAction, event.State,
			event.AgentId, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	// Expect database operations for batch saving
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")
	for range events {
		mock.ExpectExec("INSERT INTO flows").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")
	mock.ExpectExec("INSERT INTO flows").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO flows").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

	event := &flowpb.FlowEvent{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0xC0A80101,
		DstIp:       0xC0A80102,
		AgentId:     "agent-1",
	}

	mockStream := &mockReportFlowEventsStream{
		events: []*flowpb.FlowEvent{event},
	}

	// Storage returns an error
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows").
		WillReturnError(sql.ErrConnDone)

	err := server.ReportFlowEvents(mockStream)
	assert.NoError(t, err) // The gRPC method itself succeeds, error is in response
	assert.NotNil(t, mockStream.response)
	assert.False(t, mockStream.response.Success)
	assert.Equal(t, uint32(1), mockStream.response.RejectedCount)

	// We don't check expectations here because the transaction failed early
}

// TestReportFlowEvents_EmptyStream tests handling empty stream
func TestReportFlowEvents_EmptyStream(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

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

// TestEventToFlow tests event conversion
func TestEventToFlow(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	flowStorage := storage.NewFlowStorageLegacy(db)
	server := NewFlowServiceServer(flowStorage, nil)

	event := &flowpb.FlowEvent{
		TimestampNs:  uint64(1234567890123456789),
		SrcIp:        0xC0A80101, // 192.168.1.1
		DstIp:        0xC0A80102, // 192.168.1.2
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     6,
		Direction:    1,
		PacketCount:  100,
		ByteCount:    10240,
		PolicyId:     5,
		PolicyAction: 1,
		State:        2,
		AgentId:      "test-agent",
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "db"},
	}

	flow := server.eventToFlow(event)

	assert.Equal(t, "test-agent", flow.AgentId)
	assert.Equal(t, "192.168.1.1", flow.SrcIp)
	assert.Equal(t, "192.168.1.2", flow.DstIp)
	assert.Equal(t, uint32(12345), flow.SrcPort)
	assert.Equal(t, uint32(80), flow.DstPort)
	assert.Equal(t, commonpb.Protocol(6), flow.Protocol)
	assert.Equal(t, commonpb.FlowDirection(1), flow.Direction)
	assert.Equal(t, uint64(100), flow.PacketCount)
	assert.Equal(t, uint64(10240), flow.ByteCount)
	assert.Equal(t, int64(1234567890123456789), flow.StartTime)
	assert.Equal(t, uint32(5), flow.PolicyId)
	assert.Equal(t, commonpb.PolicyAction(1), flow.PolicyAction)
	assert.Equal(t, commonpb.FlowState(2), flow.State)
	assert.Equal(t, event.SourceLabels, flow.SourceLabels)
	assert.Equal(t, event.DestLabels, flow.DestLabels)
}

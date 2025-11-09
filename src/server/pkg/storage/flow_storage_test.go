package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	commonpb "github.com/ebpf-microsegment/src/proto/common"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlowStorage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)
	assert.NotNil(t, storage)
	assert.Equal(t, db, storage.db)
}

func TestBatchSaveFlowEvents_EmptyEvents(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)
	err = storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{})
	assert.NoError(t, err)
}

func TestBatchSaveFlowEvents_SingleEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

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

	// 期望开始事务
	mock.ExpectBegin()

	// 期望准备语句
	mock.ExpectPrepare("INSERT INTO flows")

	// 期望执行插入
	mock.ExpectExec("INSERT INTO flows").
		WithArgs(
			event.TimestampNs,
			"192.168.1.1",
			"192.168.1.2",
			event.SrcPort,
			event.DstPort,
			event.Protocol,
			event.Direction,
			event.PacketCount,
			event.ByteCount,
			event.PolicyId,
			event.PolicyAction,
			event.State,
			event.AgentId,
			sqlmock.AnyArg(), // source_labels JSON
			sqlmock.AnyArg(), // dest_labels JSON
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 期望提交事务
	mock.ExpectCommit()

	err = storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{event})
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestBatchSaveFlowEvents_MultipleEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

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
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")

	for range events {
		mock.ExpectExec("INSERT INTO flows").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	mock.ExpectCommit()

	err = storage.BatchSaveFlowEvents(context.Background(), events)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestBatchSaveFlowEvents_TransactionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	event := &flowpb.FlowEvent{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0xC0A80101,
		DstIp:       0xC0A80102,
		AgentId:     "test-agent",
	}

	// 期望开始事务失败
	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	err = storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{event})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestBatchSaveFlowEvents_InsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	event := &flowpb.FlowEvent{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0xC0A80101,
		DstIp:       0xC0A80102,
		AgentId:     "test-agent",
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")
	mock.ExpectExec("INSERT INTO flows").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err = storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{event})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert flow event")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestBatchSaveFlowEvents_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	event := &flowpb.FlowEvent{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0xC0A80101,
		DstIp:       0xC0A80102,
		AgentId:     "test-agent",
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO flows")
	mock.ExpectExec("INSERT INTO flows").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(sql.ErrTxDone)

	err = storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{event})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_Basic(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	query := &flowpb.FlowQuery{
		Limit:  10,
		Offset: 0,
	}

	// 期望 COUNT 查询
	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 期望主查询
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).
		AddRow(1, int64(1234567890), "192.168.1.1", "192.168.1.2", 12345, 80,
			6, 1, 10, 1024, 1, 1, 1, "agent-1", []byte("{}"), []byte("{}")).
		AddRow(2, int64(1234567891), "192.168.1.3", "192.168.1.4", 54321, 443,
			6, 1, 20, 2048, 2, 1, 1, "agent-2", []byte("{}"), []byte("{}"))

	mock.ExpectQuery("SELECT id, timestamp_ns.*FROM flows WHERE 1=1.*ORDER BY timestamp_ns DESC").
		WithArgs(10, 0).
		WillReturnRows(rows)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, flows, 2)
	assert.Equal(t, uint64(1), flows[0].Id)
	assert.Equal(t, uint64(2), flows[1].Id)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_WithTimeRange(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	startTime := int64(1000000)
	endTime := int64(2000000)

	query := &flowpb.FlowQuery{
		TimeRange: &commonpb.TimeRange{
			StartTime: startTime,
			EndTime:   endTime,
		},
		Limit: 10,
	}

	// 期望带时间范围的 COUNT 查询
	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1 AND timestamp_ns >= .* AND timestamp_ns < .*").
		WithArgs(startTime, endTime).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 期望主查询
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(1, int64(1500000), "192.168.1.1", "192.168.1.2", 12345, 80,
		6, 1, 10, 1024, 1, 1, 1, "agent-1", []byte("{}"), []byte("{}"))

	mock.ExpectQuery("SELECT id, timestamp_ns.*FROM flows WHERE 1=1 AND timestamp_ns >= .* AND timestamp_ns < .*").
		WithArgs(startTime, endTime, 10, int64(0)).
		WillReturnRows(rows)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, flows, 1)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_WithAgentID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	query := &flowpb.FlowQuery{
		AgentId: "test-agent-1",
		Limit:   10,
	}

	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1 AND agent_id = .*").
		WithArgs("test-agent-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(1, int64(1234567890), "192.168.1.1", "192.168.1.2", 12345, 80,
		6, 1, 10, 1024, 1, 1, 1, "test-agent-1", []byte("{}"), []byte("{}"))

	mock.ExpectQuery("SELECT id, timestamp_ns.*FROM flows WHERE 1=1 AND agent_id = .*").
		WithArgs("test-agent-1", 10, int64(0)).
		WillReturnRows(rows)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, flows, 1)
	assert.Equal(t, "test-agent-1", flows[0].AgentId)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_WithProtocol(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	query := &flowpb.FlowQuery{
		Protocol: 6, // TCP
		Limit:    10,
	}

	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1 AND protocol = .*").
		WithArgs(int32(6)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
	}).AddRow(1, int64(1234567890), "192.168.1.1", "192.168.1.2", 12345, 80,
		int32(6), 1, 10, 1024, 1, 1, 1, "agent-1", []byte("{}"), []byte("{}"))

	mock.ExpectQuery("SELECT id, timestamp_ns.*FROM flows WHERE 1=1 AND protocol = .*").
		WithArgs(int32(6), 10, int64(0)).
		WillReturnRows(rows)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, flows, 1)
	assert.Equal(t, commonpb.Protocol(6), flows[0].Protocol)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_CountError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	query := &flowpb.FlowQuery{Limit: 10}

	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1").
		WillReturnError(sql.ErrNoRows)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, flows)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "failed to count flows")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestQueryFlows_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	query := &flowpb.FlowQuery{Limit: 10}

	mock.ExpectQuery("SELECT COUNT.*FROM flows WHERE 1=1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery("SELECT id, timestamp_ns.*FROM flows WHERE 1=1.*").
		WillReturnError(sql.ErrConnDone)

	flows, total, err := storage.QueryFlows(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, flows)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "failed to query flows")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

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
			name:     "127.0.0.1",
			input:    0x7F000001,
			expected: "127.0.0.1",
		},
		{
			name:     "0.0.0.0",
			input:    0x00000000,
			expected: "0.0.0.0",
		},
		{
			name:     "255.255.255.255",
			input:    0xFFFFFFFF,
			expected: "255.255.255.255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intToIP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFlowSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	// 期望主查询
	summaryRows := sqlmock.NewRows([]string{
		"total_flows", "total_packets", "total_bytes", "unique_source_ips", "unique_dest_ips", "avg_duration_ms",
	}).AddRow(100, 1000, 100000, 10, 15, 50.5)

	mock.ExpectQuery("SELECT.*FROM flows WHERE timestamp_ns >= .* AND timestamp_ns <= .*").
		WithArgs(startTime.UnixNano(), endTime.UnixNano()).
		WillReturnRows(summaryRows)

	// 期望协议统计查询
	protocolRows := sqlmock.NewRows([]string{"protocol", "count", "bytes"}).
		AddRow("6", 80, 80000).
		AddRow("17", 20, 20000)

	mock.ExpectQuery("SELECT.*protocol.*FROM flows WHERE timestamp_ns >= .* AND timestamp_ns <= .*GROUP BY protocol.*").
		WithArgs(startTime.UnixNano(), endTime.UnixNano()).
		WillReturnRows(protocolRows)

	summary, err := storage.GetFlowSummary(context.Background(), startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, int64(100), summary.TotalFlows)
	assert.Equal(t, int64(1000), summary.TotalPackets)
	assert.Equal(t, int64(100000), summary.TotalBytes)
	assert.Equal(t, int64(10), summary.UniqueSourceIPs)
	assert.Equal(t, int64(15), summary.UniqueDestIPs)
	assert.Equal(t, 50.5, summary.AvgDuration)
	assert.Len(t, summary.TopProtocols, 2)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetFlowSummary_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	mock.ExpectQuery("SELECT.*FROM flows WHERE timestamp_ns >= .* AND timestamp_ns <= .*").
		WillReturnError(sql.ErrConnDone)

	summary, err := storage.GetFlowSummary(context.Background(), startTime, endTime)
	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "failed to get flow summary")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetFlowDependencies(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()
	groupBy := "app"

	protocols := []string{"tcp", "udp"}
	protocolsJSON, _ := json.Marshal(protocols)

	rows := sqlmock.NewRows([]string{
		"source_label", "dest_label", "flow_count", "total_bytes", "protocols",
	}).AddRow("web", "db", 100, 50000, protocolsJSON).
		AddRow("api", "cache", 50, 25000, protocolsJSON)

	mock.ExpectQuery("SELECT.*FROM flows WHERE timestamp_ns >= .* AND timestamp_ns <= .*").
		WithArgs(startTime.UnixNano(), endTime.UnixNano(), groupBy).
		WillReturnRows(rows)

	deps, err := storage.GetFlowDependencies(context.Background(), startTime, endTime, groupBy)
	assert.NoError(t, err)
	assert.Len(t, deps, 2)
	assert.Equal(t, "web", deps[0].SourceLabel)
	assert.Equal(t, "db", deps[0].DestLabel)
	assert.Equal(t, int64(100), deps[0].FlowCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetFlowDependencies_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	mock.ExpectQuery("SELECT.*FROM flows WHERE timestamp_ns >= .* AND timestamp_ns <= .*").
		WillReturnError(sql.ErrConnDone)

	deps, err := storage.GetFlowDependencies(context.Background(), startTime, endTime, "app")
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to get flow dependencies")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

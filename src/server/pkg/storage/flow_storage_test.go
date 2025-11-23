package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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

func TestNewFlowStorage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewFlowStorage(db)
	assert.NotNil(t, storage)
}

func TestBatchSaveFlowEvents_Success(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "flows".*RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	event := &flowpb.FlowEvent{
		TimestampNs:  uint64(time.Now().UnixNano()),
		SrcIp:        0x7F000001,
		DstIp:        0xC0A80101,
		SrcPort:      12345,
		DstPort:      80,
		Protocol:     6,
		Direction:    1,
		PacketCount:  10,
		ByteCount:    1024,
		AgentId:      "agent-1",
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "db"},
	}

	err := storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{event})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchSaveFlowEvents_InsertError(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "flows".*RETURNING "id"`).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := storage.BatchSaveFlowEvents(context.Background(), []*flowpb.FlowEvent{{
		TimestampNs: uint64(time.Now().UnixNano()),
		SrcIp:       0x7F000001,
		DstIp:       0xC0A80101,
		AgentId:     "agent-1",
	}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert flow events")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryFlows_Success(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "flows"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
		"start_time", "end_time", "last_seen",
	}).AddRow(
		1, int64(1111), "192.168.1.1", "192.168.1.2", 12345, 80,
		6, 1, 10, 1000, 1, 1, 1, "agent-1", []byte(`{"app":"web"}`), []byte(`{"app":"db"}`), time.Now(), time.Now(), time.Now(),
	).AddRow(
		2, int64(2222), "10.0.0.1", "10.0.0.2", 443, 8080,
		17, 2, 20, 2000, nil, nil, nil, "agent-2", []byte(`{}`), []byte(`{}`), nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT .* FROM "flows" ORDER BY timestamp_ns DESC LIMIT \$1`).
		WithArgs(10).
		WillReturnRows(rows)

	result, total, err := storage.QueryFlows(context.Background(), &flowpb.FlowQuery{Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
	assert.Equal(t, "agent-1", result[0].AgentId)
	assert.Equal(t, commonpb.Protocol(6), result[0].Protocol)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryFlows_CountError(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "flows"`).
		WillReturnError(sql.ErrConnDone)

	flows, total, err := storage.QueryFlows(context.Background(), &flowpb.FlowQuery{})
	assert.Error(t, err)
	assert.Nil(t, flows)
	assert.Equal(t, int64(0), total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFlowSummary(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	summaryRows := sqlmock.NewRows([]string{
		"total_flows", "active_flows", "closed_flows", "total_packets", "total_bytes",
		"allowed_flows", "denied_flows", "unique_source_ips", "unique_dest_ips", "avg_duration_ms",
	}).AddRow(100, 60, 40, 1000, 2048, 80, 20, 10, 12, 50.5)

	mock.ExpectQuery(`SELECT.*total_flows.*FROM "flows"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(summaryRows)

	protocolRows := sqlmock.NewRows([]string{"protocol", "count", "bytes"}).
		AddRow("6", 80, 1500).
		AddRow("17", 20, 548)

	mock.ExpectQuery(`SELECT.*protocol::text.*FROM "flows"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 5).
		WillReturnRows(protocolRows)

	summary, err := storage.GetFlowSummary(context.Background(), time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	assert.Equal(t, int64(100), summary.TotalFlows)
	assert.Len(t, summary.TopProtocols, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFlowDependencies(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	protocols := []string{"tcp", "udp"}
	protoJSON, _ := json.Marshal(protocols)

	rows := sqlmock.NewRows([]string{"source_label", "dest_label", "flow_count", "total_bytes", "protocols"}).
		AddRow("web", "db", 50, 1024, protoJSON)

	mock.ExpectQuery(`SELECT.*source_labels`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "app", "app", 100).
		WillReturnRows(rows)

	deps, err := storage.GetFlowDependencies(context.Background(), time.Now().Add(-time.Hour), time.Now(), "app")
	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, protocols, deps[0].Protocols)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFlowDependencies_InvalidGroupBy(t *testing.T) {
	storage, _, cleanup := newMockFlowStorage(t)
	defer cleanup()

	_, err := storage.GetFlowDependencies(context.Background(), time.Now(), time.Now(), "foo;DROP TABLE")
	assert.Error(t, err)
}

func TestQueryFlows_WithLabelFilters(t *testing.T) {
	storage, mock, cleanup := newMockFlowStorage(t)
	defer cleanup()

	sourceLabels := map[string]string{"app": "web", "env": "prod"}
	destLabels := map[string]string{"app": "db"}
	sourceLabelsJSON, _ := json.Marshal(sourceLabels)
	destLabelsJSON, _ := json.Marshal(destLabels)

	// Expect count query with JSONB filter
	mock.ExpectQuery(`SELECT count\(\*\) FROM "flows" WHERE source_labels @> \$1 AND dest_labels @> \$2`).
		WithArgs(string(sourceLabelsJSON), string(destLabelsJSON)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Expect select query with JSONB filter
	rows := sqlmock.NewRows([]string{
		"id", "timestamp_ns", "src_ip", "dst_ip", "src_port", "dst_port",
		"protocol", "direction", "packet_count", "byte_count", "policy_id",
		"policy_action", "state", "agent_id", "source_labels", "dest_labels",
		"start_time", "end_time", "last_seen",
	}).AddRow(
		1, int64(1111), "192.168.1.1", "192.168.1.2", 12345, 80,
		6, 1, 10, 1000, 1, 1, 1, "agent-1",
		[]byte(`{"app":"web","env":"prod"}`), []byte(`{"app":"db"}`),
		time.Now(), time.Now(), time.Now(),
	)

	mock.ExpectQuery(`SELECT .* FROM "flows" WHERE source_labels @> \$1 AND dest_labels @> \$2 ORDER BY timestamp_ns DESC LIMIT \$3`).
		WithArgs(string(sourceLabelsJSON), string(destLabelsJSON), 10).
		WillReturnRows(rows)

	result, total, err := storage.QueryFlows(context.Background(), &flowpb.FlowQuery{
		Limit:        10,
		SourceLabels: sourceLabels,
		DestLabels:   destLabels,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
	assert.Equal(t, "web", result[0].SourceLabels["app"])
	assert.Equal(t, "prod", result[0].SourceLabels["env"])
	assert.Equal(t, "db", result[0].DestLabels["app"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntToIP(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected string
	}{
		{"loopback", 0x7F000001, "127.0.0.1"},
		{"private", 0xC0A80101, "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, intToIP(tt.input))
		})
	}
}

func newMockFlowStorage(t *testing.T) (*FlowStorage, sqlmock.Sqlmock, func()) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Discard,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)

	storage := NewFlowStorageFromGorm(gormDB)

	cleanup := func() {
		sqlDB.Close()
	}

	return storage, mock, cleanup
}

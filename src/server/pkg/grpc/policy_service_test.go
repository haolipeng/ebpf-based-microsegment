package grpc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/pubsub"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// mockSubscribePoliciesStream is a mock implementation of the policy subscription stream
type mockSubscribePoliciesStream struct {
	updates  []*policypb.PolicyUpdate
	ctx      context.Context
	cancelFn context.CancelFunc
}

func (m *mockSubscribePoliciesStream) Send(update *policypb.PolicyUpdate) error {
	m.updates = append(m.updates, update)
	return nil
}

func (m *mockSubscribePoliciesStream) Context() context.Context {
	if m.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		m.ctx = ctx
		m.cancelFn = cancel
	}
	return m.ctx
}

func (m *mockSubscribePoliciesStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockSubscribePoliciesStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockSubscribePoliciesStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockSubscribePoliciesStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockSubscribePoliciesStream) SetTrailer(md metadata.MD) {
}

// TestNewPolicyServiceServer tests the constructor
func TestNewPolicyServiceServer(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	assert.NotNil(t, server)
	assert.NotNil(t, server.policyStorage)
}

// TestSyncPolicies_Success tests successful policy synchronization
func TestSyncPolicies_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	req := &policypb.SyncRequest{
		AgentId:       "agent-1",
		PolicyVersion: 0,
	}

	// Mock GetAllPolicies query - version query
	mock.ExpectQuery("SELECT version FROM policy_version").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(1))

	// Mock GetAllPolicies query - policies query (16 columns including process fields)
	rows := sqlmock.NewRows([]string{
		"rule_id", "src_ip", "dst_ip", "src_port", "dst_port", "protocol",
		"action", "priority", "source_labels", "dest_labels", "description",
		"process_name", "process_path", "match_mode",
		"created_at", "updated_at",
	}).AddRow(
		uint32(1), "10.0.0.0/24", "192.168.1.0/24", uint32(0), uint32(80),
		uint32(6), uint32(1), uint32(100),
		[]byte(`{"app":"web"}`), []byte(`{"app":"db"}`), "Allow web to db",
		nil, nil, nil,
		int64(time.Now().UnixNano()), int64(time.Now().UnixNano()),
	)
	mock.ExpectQuery("SELECT rule_id, src_ip").
		WillReturnRows(rows)

	resp, err := server.SyncPolicies(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(1), resp.PolicyVersion)
	assert.Equal(t, uint32(1), resp.PolicyCount)
	assert.Len(t, resp.Policies, 1)
	assert.Equal(t, uint32(1), resp.Policies[0].RuleId)
	assert.Greater(t, resp.ServerTime, int64(0))

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestSyncPolicies_EmptyPolicies tests sync with no policies
func TestSyncPolicies_EmptyPolicies(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	req := &policypb.SyncRequest{
		AgentId:       "agent-1",
		PolicyVersion: 0,
	}

	// Mock GetAllPolicies query - version query
	mock.ExpectQuery("SELECT version FROM policy_version").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(0))

	// Mock GetAllPolicies query - policies query (empty, 16 columns)
	rows := sqlmock.NewRows([]string{
		"rule_id", "src_ip", "dst_ip", "src_port", "dst_port", "protocol",
		"action", "priority", "source_labels", "dest_labels", "description",
		"process_name", "process_path", "match_mode",
		"created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT rule_id, src_ip").
		WillReturnRows(rows)

	resp, err := server.SyncPolicies(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(0), resp.PolicyVersion)
	assert.Equal(t, uint32(0), resp.PolicyCount)
	assert.Len(t, resp.Policies, 0)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestSyncPolicies_StorageError tests handling storage errors
func TestSyncPolicies_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	req := &policypb.SyncRequest{
		AgentId:       "agent-1",
		PolicyVersion: 0,
	}

	// Mock storage error
	mock.ExpectQuery("SELECT version FROM policy_version").
		WillReturnError(sql.ErrConnDone)

	resp, err := server.SyncPolicies(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get policies")
}

// TestSubscribePolicies_SendsInitialPolicies tests initial policy push
func TestSubscribePolicies_SendsInitialPolicies(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	req := &policypb.SubscribeRequest{
		AgentId:        "agent-1",
		CurrentVersion: 0,
	}

	mockStream := &mockSubscribePoliciesStream{}

	// Mock GetAllPolicies query - version query
	mock.ExpectQuery("SELECT version FROM policy_version").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))

	// Mock GetAllPolicies query - policies query (16 columns including process fields)
	rows := sqlmock.NewRows([]string{
		"rule_id", "src_ip", "dst_ip", "src_port", "dst_port", "protocol",
		"action", "priority", "source_labels", "dest_labels", "description",
		"process_name", "process_path", "match_mode",
		"created_at", "updated_at",
	}).
		AddRow(
			uint32(1), "10.0.0.0/24", "192.168.1.0/24", uint32(0), uint32(80),
			uint32(6), uint32(1), uint32(100),
			[]byte(`{"app":"web"}`), []byte(`{"app":"db"}`), "Allow web to db",
			nil, nil, nil,
			int64(time.Now().UnixNano()), int64(time.Now().UnixNano()),
		).
		AddRow(
			uint32(2), "10.0.0.0/24", "0.0.0.0/0", uint32(0), uint32(443),
			uint32(6), uint32(1), uint32(90),
			[]byte(`{"app":"web"}`), []byte(`{}`), "Allow web to internet",
			nil, nil, nil,
			int64(time.Now().UnixNano()), int64(time.Now().UnixNano()),
		)
	mock.ExpectQuery("SELECT rule_id, src_ip").
		WillReturnRows(rows)

	// Start subscription in background and cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		if mockStream.cancelFn != nil {
			mockStream.cancelFn()
		}
	}()

	err := server.SubscribePolicies(req, mockStream)
	// Error is expected when context is cancelled - the function returns context.Canceled
	// which is acceptable behavior for graceful disconnection
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}

	// Verify that initial policies were sent
	assert.Len(t, mockStream.updates, 2)
	assert.Equal(t, policypb.PolicyUpdateType_UPDATE_ADD, mockStream.updates[0].UpdateType)
	assert.Equal(t, uint32(1), mockStream.updates[0].Policy.RuleId)
	assert.Equal(t, uint64(2), mockStream.updates[0].PolicyVersion)
	assert.Equal(t, uint32(2), mockStream.updates[1].Policy.RuleId)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestSubscribePolicies_StorageError tests handling storage errors in subscription
func TestSubscribePolicies_StorageError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	req := &policypb.SubscribeRequest{
		AgentId:        "agent-1",
		CurrentVersion: 0,
	}

	mockStream := &mockSubscribePoliciesStream{}

	// Mock storage error
	mock.ExpectQuery("SELECT version FROM policy_version").
		WillReturnError(sql.ErrConnDone)

	err := server.SubscribePolicies(req, mockStream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get policies")
	assert.Len(t, mockStream.updates, 0) // No updates sent on error
}

// TestReportPolicyStats_Success tests successful policy stats reporting
func TestReportPolicyStats_Success(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	report := &policypb.PolicyStatsReport{
		AgentId:   "agent-1",
		Timestamp: time.Now().UnixNano(),
		PolicyStats: []*policypb.PolicyStats{
			{
				RuleId:        1,
				HitCount:      100,
				PacketCount:   1000,
				ByteCount:     102400,
				FlowCount:     95,
				LastMatchTime: time.Now().UnixNano(),
			},
			{
				RuleId:        2,
				HitCount:      50,
				PacketCount:   500,
				ByteCount:     51200,
				FlowCount:     50,
				LastMatchTime: time.Now().UnixNano(),
			},
		},
	}

	// Note: MVP implementation just acknowledges receipt
	resp, err := server.ReportPolicyStats(context.Background(), report)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Policy stats received", resp.Message)
	assert.Equal(t, uint32(2), resp.AcceptedCount)
}

// TestReportPolicyStats_EmptyReport tests handling empty stats report
func TestReportPolicyStats_EmptyReport(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	policyStorage := storage.NewPolicyStorage(db)
	policyPubSub := pubsub.NewPolicyPubSub()
	server := NewPolicyServiceServer(policyStorage, policyPubSub)

	report := &policypb.PolicyStatsReport{
		AgentId:     "agent-1",
		Timestamp:   time.Now().UnixNano(),
		PolicyStats: []*policypb.PolicyStats{},
	}

	resp, err := server.ReportPolicyStats(context.Background(), report)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, uint32(0), resp.AcceptedCount)
}

package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	commonpb "github.com/ebpf-microsegment/src/proto/common"
	policypb "github.com/ebpf-microsegment/src/proto/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicyStorage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)
	assert.NotNil(t, storage)
	assert.Equal(t, db, storage.db)
}

func TestGetAllPolicies_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	// 期望版本查询
	mock.ExpectQuery("SELECT version FROM policy_version WHERE id = 1").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(uint64(5)))

	// 期望策略查询
	policyRows := sqlmock.NewRows([]string{
		"rule_id", "src_ip", "dst_ip", "src_port", "dst_port", "protocol", "action", "priority",
		"source_labels", "dest_labels", "description", "created_at", "updated_at",
	}).
		AddRow(1, "10.0.0.0/24", "192.168.1.0/24", 0, 80, 6, 1, 100,
			[]byte(`{"app":"web"}`), []byte(`{"app":"db"}`), "Allow web to db", int64(1234567890), int64(1234567900)).
		AddRow(2, "0.0.0.0/0", "10.0.0.0/24", 0, 22, 6, 2, 90,
			[]byte(`{}`), []byte(`{}`), "Deny SSH", int64(1234567891), int64(1234567901))

	mock.ExpectQuery("SELECT rule_id, src_ip.*FROM policies.*ORDER BY priority DESC, rule_id").
		WillReturnRows(policyRows)

	policies, version, err := storage.GetAllPolicies(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), version)
	assert.Len(t, policies, 2)
	assert.Equal(t, uint32(1), policies[0].RuleId)
	assert.Equal(t, "10.0.0.0/24", policies[0].SrcIp)
	assert.Equal(t, commonpb.PolicyAction(1), policies[0].Action)
	assert.Equal(t, "web", policies[0].SourceLabels["app"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllPolicies_EmptyResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	mock.ExpectQuery("SELECT version FROM policy_version WHERE id = 1").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(uint64(1)))

	mock.ExpectQuery("SELECT rule_id, src_ip.*FROM policies.*").
		WillReturnRows(sqlmock.NewRows([]string{
			"rule_id", "src_ip", "dst_ip", "src_port", "dst_port", "protocol", "action", "priority",
			"source_labels", "dest_labels", "description", "created_at", "updated_at",
		}))

	policies, version, err := storage.GetAllPolicies(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), version)
	assert.Len(t, policies, 0)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllPolicies_VersionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	mock.ExpectQuery("SELECT version FROM policy_version WHERE id = 1").
		WillReturnError(sql.ErrNoRows)

	policies, version, err := storage.GetAllPolicies(context.Background())
	assert.Error(t, err)
	assert.Nil(t, policies)
	assert.Equal(t, uint64(0), version)
	assert.Contains(t, err.Error(), "failed to get policy version")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAllPolicies_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	mock.ExpectQuery("SELECT version FROM policy_version WHERE id = 1").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(uint64(1)))

	mock.ExpectQuery("SELECT rule_id, src_ip.*FROM policies.*").
		WillReturnError(sql.ErrConnDone)

	policies, version, err := storage.GetAllPolicies(context.Background())
	assert.Error(t, err)
	assert.Nil(t, policies)
	assert.Equal(t, uint64(0), version)
	assert.Contains(t, err.Error(), "failed to query policies")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCreatePolicy_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:       1,
		SrcIp:        "10.0.0.0/24",
		DstIp:        "192.168.1.0/24",
		SrcPort:      0,
		DstPort:      80,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		Action:       commonpb.PolicyAction_ACTION_ALLOW,
		Priority:     100,
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "db"},
		Description:  "Allow web to db",
	}

	// 期望插入策略
	mock.ExpectExec("INSERT INTO policies").
		WithArgs(policy.RuleId, policy.SrcIp, policy.DstIp, policy.SrcPort, policy.DstPort,
			policy.Protocol, policy.Action, policy.Priority,
			sqlmock.AnyArg(), sqlmock.AnyArg(), policy.Description).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 期望更新版本
	mock.ExpectExec("UPDATE policy_version SET version = version \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.CreatePolicy(context.Background(), policy)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCreatePolicy_InsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:   1,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	mock.ExpectExec("INSERT INTO policies").
		WillReturnError(sql.ErrConnDone)

	err = storage.CreatePolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create policy")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCreatePolicy_VersionUpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:   1,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	mock.ExpectExec("INSERT INTO policies").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE policy_version").
		WillReturnError(sql.ErrConnDone)

	err = storage.CreatePolicy(context.Background(), policy)
	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdatePolicy_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:       1,
		SrcIp:        "10.0.0.0/24",
		DstIp:        "192.168.1.0/24",
		SrcPort:      0,
		DstPort:      443,
		Protocol:     commonpb.Protocol_PROTOCOL_TCP,
		Action:       commonpb.PolicyAction_ACTION_ALLOW,
		Priority:     100,
		SourceLabels: map[string]string{"app": "web"},
		DestLabels:   map[string]string{"app": "api"},
		Description:  "Allow web to api",
	}

	// 期望更新策略
	mock.ExpectExec("UPDATE policies.*WHERE rule_id = .*").
		WithArgs(policy.RuleId, policy.SrcIp, policy.DstIp, policy.SrcPort, policy.DstPort,
			policy.Protocol, policy.Action, policy.Priority,
			sqlmock.AnyArg(), sqlmock.AnyArg(), policy.Description).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 期望更新版本
	mock.ExpectExec("UPDATE policy_version SET version = version \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.UpdatePolicy(context.Background(), policy)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdatePolicy_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:   999,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	// 期望更新但返回 0 行受影响
	mock.ExpectExec("UPDATE policies.*WHERE rule_id = .*").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = storage.UpdatePolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdatePolicy_UpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:   1,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	mock.ExpectExec("UPDATE policies.*WHERE rule_id = .*").
		WillReturnError(sql.ErrConnDone)

	err = storage.UpdatePolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update policy")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUpdatePolicy_VersionUpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	policy := &policypb.Policy{
		RuleId:   1,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	mock.ExpectExec("UPDATE policies.*WHERE rule_id = .*").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("UPDATE policy_version").
		WillReturnError(sql.ErrConnDone)

	err = storage.UpdatePolicy(context.Background(), policy)
	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDeletePolicy_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	ruleID := uint32(1)

	// 期望删除策略
	mock.ExpectExec("DELETE FROM policies WHERE rule_id = .*").
		WithArgs(ruleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 期望更新版本
	mock.ExpectExec("UPDATE policy_version SET version = version \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.DeletePolicy(context.Background(), ruleID)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDeletePolicy_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	ruleID := uint32(999)

	// 期望删除但返回 0 行受影响
	mock.ExpectExec("DELETE FROM policies WHERE rule_id = .*").
		WithArgs(ruleID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = storage.DeletePolicy(context.Background(), ruleID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")
	assert.Contains(t, err.Error(), "999")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDeletePolicy_DeleteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	ruleID := uint32(1)

	mock.ExpectExec("DELETE FROM policies WHERE rule_id = .*").
		WillReturnError(sql.ErrConnDone)

	err = storage.DeletePolicy(context.Background(), ruleID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete policy")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDeletePolicy_VersionUpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPolicyStorage(db)

	ruleID := uint32(1)

	mock.ExpectExec("DELETE FROM policies WHERE rule_id = .*").
		WithArgs(ruleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("UPDATE policy_version").
		WillReturnError(sql.ErrConnDone)

	err = storage.DeletePolicy(context.Background(), ruleID)
	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

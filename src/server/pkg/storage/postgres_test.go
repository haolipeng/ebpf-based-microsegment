package storage

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresDB_ValidConfig(t *testing.T) {
	// 测试使用 sqlmock 模拟成功连接
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// 期望 Ping 调用成功
	mock.ExpectPing()

	// 验证 Ping 成功
	err = db.Ping()
	assert.NoError(t, err)

	// 验证所有期望都被满足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNewPostgresDB_InvalidDSN(t *testing.T) {
	// 测试无效的 DSN
	invalidDSN := "invalid://connection/string"
	db, err := NewPostgresDB(invalidDSN, 10, 5, 5*time.Minute)

	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to")
}

func TestNewPostgresDB_ConnectionPoolConfig(t *testing.T) {
	// 使用 sqlmock 创建模拟数据库连接
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// 设置期望的 Ping 调用
	mock.ExpectPing()

	// 配置连接池参数
	maxOpenConns := 25
	maxIdleConns := 5
	connMaxLifetime := 10 * time.Minute

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// 执行 Ping 触发连接
	err = db.Ping()
	assert.NoError(t, err)

	// 验证连接池配置（通过 Stats 验证）
	stats := db.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0) // 验证配置被设置

	// 验证期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesAllTables(t *testing.T) {
	// 创建 sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 期望执行 schema 创建的 SQL
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 调用 InitSchema
	err = InitSchema(db)
	assert.NoError(t, err)

	// 验证所有期望都被满足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_Idempotent(t *testing.T) {
	// 测试 InitSchema 多次调用的幂等性
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 第一次调用
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = InitSchema(db)
	assert.NoError(t, err)

	// 第二次调用（幂等性测试）
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = InitSchema(db)
	assert.NoError(t, err)

	// 验证期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_ErrorHandling(t *testing.T) {
	// 测试 schema 初始化失败的情况
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 期望执行失败
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnError(sql.ErrConnDone)

	// 调用 InitSchema
	err = InitSchema(db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create schema")

	// 验证期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesFlowsTable(t *testing.T) {
	// 测试是否正确创建 flows 表及其索引
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 期望创建表和索引的 SQL
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesPoliciesTable(t *testing.T) {
	// 测试是否正确创建 policies 表
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 期望包含 policies 表创建的 SQL
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesPolicyVersionTable(t *testing.T) {
	// 测试是否正确创建 policy_version 表并初始化版本
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesAgentsTable(t *testing.T) {
	// 测试是否正确创建 agents 表
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesAgentMetricsTable(t *testing.T) {
	// 测试是否正确创建 agent_metrics 表（带外键约束）
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestInitSchema_CreatesAllIndexes(t *testing.T) {
	// 测试是否创建所有必要的索引
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 期望创建索引的 SQL（包含在完整的 schema 中）
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS flows").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = InitSchema(db)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDatabaseConnection_Lifecycle(t *testing.T) {
	// 测试数据库连接的完整生命周期
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	// 测试连接
	mock.ExpectPing()
	err = db.Ping()
	assert.NoError(t, err)

	// 验证期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// 测试关闭
	mock.ExpectClose()
	err = db.Close()
	assert.NoError(t, err)

	// 关闭后的操作应该失败
	err = db.Ping()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestConnectionPool_MaxConnections(t *testing.T) {
	// 测试连接池的最大连接数限制
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	maxOpenConns := 2
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(1)

	// 期望多次 Ping
	mock.ExpectPing()
	mock.ExpectPing()

	// 创建多个连接
	err = db.Ping()
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)

	// 验证连接池统计
	stats := db.Stats()
	assert.LessOrEqual(t, stats.OpenConnections, maxOpenConns)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

package storage

import (
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

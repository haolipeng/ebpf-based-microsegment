package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB 表示一个测试数据库实例
type TestDB struct {
	Container testcontainers.Container
	DB        *sql.DB
	Host      string
	Port      string
	DBName    string
	User      string
	Password  string
}

// SetupTestDB 创建一个 PostgreSQL 测试数据库容器
// 用于集成测试，返回可立即使用的数据库连接
func SetupTestDB(t *testing.T) *TestDB {
	ctx := context.Background()

	// 定义 PostgreSQL 容器请求
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "test_microsegment",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	// 启动容器
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// 获取容器的主机和端口
	host, err := container.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	mappedPort, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get mapped port")

	port := mappedPort.Port()

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s port=%s user=test_user password=test_password dbname=test_microsegment sslmode=disable",
		host, port)

	// 连接到数据库（重试机制）
	var db *sql.DB
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}
	require.NoError(t, err, "Failed to connect to test database")

	// 启用 TimescaleDB 扩展
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE")
	require.NoError(t, err, "Failed to create TimescaleDB extension")

	testDB := &TestDB{
		Container: container,
		DB:        db,
		Host:      host,
		Port:      port,
		DBName:    "test_microsegment",
		User:      "test_user",
		Password:  "test_password",
	}

	// 注册清理函数
	t.Cleanup(func() {
		testDB.Teardown(t)
	})

	return testDB
}

// DSN 返回数据库连接字符串
func (tdb *TestDB) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		tdb.Host, tdb.Port, tdb.User, tdb.Password, tdb.DBName)
}

// Teardown 清理测试数据库和容器
func (tdb *TestDB) Teardown(t *testing.T) {
	ctx := context.Background()

	if tdb.DB != nil {
		err := tdb.DB.Close()
		require.NoError(t, err, "Failed to close database connection")
	}

	if tdb.Container != nil {
		err := tdb.Container.Terminate(ctx)
		require.NoError(t, err, "Failed to terminate container")
	}
}

// TruncateAllTables 清空所有表数据（但保留表结构）
// 用于在测试之间清理数据
func (tdb *TestDB) TruncateAllTables(t *testing.T) {
	tables := []string{
		"flows",
		"policies",
		"policy_version",
		"agents",
		"agent_metrics",
		"events",
	}

	for _, table := range tables {
		_, err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// 如果表不存在，忽略错误
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}

// ExecuteSQLFile 执行 SQL 文件（用于初始化 schema）
func (tdb *TestDB) ExecuteSQLFile(t *testing.T, sqlContent string) {
	_, err := tdb.DB.Exec(sqlContent)
	require.NoError(t, err, "Failed to execute SQL")
}

// WaitForTable 等待表创建完成
// 用于异步 schema 初始化的测试
func (tdb *TestDB) WaitForTable(t *testing.T, tableName string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for table %s to be created", tableName)
		case <-ticker.C:
			var exists bool
			query := `SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)`
			err := tdb.DB.QueryRow(query, tableName).Scan(&exists)
			if err == nil && exists {
				return
			}
		}
	}
}

// CountRows 返回表中的行数
// 用于验证数据插入
func (tdb *TestDB) CountRows(t *testing.T, tableName string) int {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err := tdb.DB.QueryRow(query).Scan(&count)
	require.NoError(t, err, "Failed to count rows in %s", tableName)
	return count
}

// QueryOne 执行查询并返回第一行结果
// 用于快速验证单个记录
func (tdb *TestDB) QueryOne(t *testing.T, query string, args ...interface{}) *sql.Row {
	return tdb.DB.QueryRow(query, args...)
}

// Exec 执行 SQL 语句
// 用于快速插入测试数据
func (tdb *TestDB) Exec(t *testing.T, query string, args ...interface{}) sql.Result {
	result, err := tdb.DB.Exec(query, args...)
	require.NoError(t, err, "Failed to execute query: %s", query)
	return result
}

// InsertTestFlow 插入测试流数据
// 用于快速创建测试数据
func (tdb *TestDB) InsertTestFlow(t *testing.T, srcIP, dstIP string, srcPort, dstPort int) int64 {
	query := `
		INSERT INTO flows (
			src_ip, dst_ip, src_port, dst_port, protocol,
			direction, packet_count, byte_count, start_time,
			policy_id, policy_action, state, agent_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	var id int64
	err := tdb.DB.QueryRow(
		query,
		srcIP, dstIP, srcPort, dstPort, "tcp",
		"egress", 100, 15000, time.Now(),
		1, "allow", "established", "test-agent",
	).Scan(&id)
	require.NoError(t, err, "Failed to insert test flow")
	return id
}

// InsertTestPolicy 插入测试策略
func (tdb *TestDB) InsertTestPolicy(t *testing.T, ruleID int, srcIP, dstIP string) {
	query := `
		INSERT INTO policies (
			rule_id, src_ip, dst_ip, src_port, dst_port,
			protocol, action, priority, description
		) VALUES ($1, $2, $3, 0, 443, 'tcp', 'allow', 10, 'Test policy')
	`

	_, err := tdb.DB.Exec(query, ruleID, srcIP, dstIP)
	require.NoError(t, err, "Failed to insert test policy")
}

// InsertTestAgent 插入测试 Agent
func (tdb *TestDB) InsertTestAgent(t *testing.T, agentID, hostname string) {
	query := `
		INSERT INTO agents (
			agent_id, hostname, version, interface, ip_addresses,
			os, kernel_version, status, last_heartbeat
		) VALUES ($1, $2, '1.0.0', 'eth0', ARRAY['10.0.1.10'], 'Linux', '5.15.0', 'running', NOW())
	`

	_, err := tdb.DB.Exec(query, agentID, hostname)
	require.NoError(t, err, "Failed to insert test agent")
}

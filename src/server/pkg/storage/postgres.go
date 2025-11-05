package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// NewPostgresDB creates a new PostgreSQL connection
func NewPostgresDB(dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Infof("Connected to PostgreSQL database")
	return db, nil
}

// InitSchema creates the database schema if it doesn't exist
func InitSchema(db *sql.DB) error {
	schema := `
	-- Flows table (simplified, no TimescaleDB for MVP)
	CREATE TABLE IF NOT EXISTS flows (
		id BIGSERIAL PRIMARY KEY,
		timestamp_ns BIGINT NOT NULL,
		src_ip INET NOT NULL,
		dst_ip INET NOT NULL,
		src_port INTEGER NOT NULL,
		dst_port INTEGER NOT NULL,
		protocol INTEGER NOT NULL,
		direction INTEGER NOT NULL,
		packet_count BIGINT NOT NULL,
		byte_count BIGINT NOT NULL,
		policy_id INTEGER,
		policy_action INTEGER,
		state INTEGER,
		agent_id TEXT NOT NULL,
		source_labels JSONB,
		dest_labels JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_flows_timestamp ON flows(timestamp_ns);
	CREATE INDEX IF NOT EXISTS idx_flows_agent_id ON flows(agent_id);
	CREATE INDEX IF NOT EXISTS idx_flows_src_ip ON flows(src_ip);
	CREATE INDEX IF NOT EXISTS idx_flows_dst_ip ON flows(dst_ip);
	CREATE INDEX IF NOT EXISTS idx_flows_created_at ON flows(created_at);

	-- Policies table
	CREATE TABLE IF NOT EXISTS policies (
		rule_id INTEGER PRIMARY KEY,
		src_ip TEXT NOT NULL,
		dst_ip TEXT NOT NULL,
		src_port INTEGER NOT NULL,
		dst_port INTEGER NOT NULL,
		protocol INTEGER NOT NULL,
		action INTEGER NOT NULL,
		priority INTEGER NOT NULL,
		source_labels JSONB,
		dest_labels JSONB,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Policy version tracking
	CREATE TABLE IF NOT EXISTS policy_version (
		id INTEGER PRIMARY KEY DEFAULT 1,
		version BIGINT NOT NULL DEFAULT 1,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CHECK (id = 1)
	);
	INSERT INTO policy_version (id, version) VALUES (1, 1) ON CONFLICT DO NOTHING;

	-- Agents table
	CREATE TABLE IF NOT EXISTS agents (
		agent_id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		version TEXT NOT NULL,
		interface TEXT,
		ip_addresses TEXT[],
		os TEXT,
		kernel_version TEXT,
		start_time TIMESTAMP,
		last_heartbeat TIMESTAMP,
		status TEXT DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Agent metrics (latest snapshot)
	CREATE TABLE IF NOT EXISTS agent_metrics (
		agent_id TEXT PRIMARY KEY REFERENCES agents(agent_id) ON DELETE CASCADE,
		cpu_usage REAL,
		memory_usage BIGINT,
		packets_processed BIGINT,
		active_sessions INTEGER,
		flows_reported BIGINT,
		active_policies INTEGER,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	logrus.Info("Database schema initialized")
	return nil
}

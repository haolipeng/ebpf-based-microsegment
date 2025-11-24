-- Migration: 001_initial_schema
-- Description: Create initial database schema for microsegment-server
-- Author: Claude Code
-- Date: 2025-11-06

-- =================================================================
-- Flows Table (Time-series flow events from agents)
-- =================================================================
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

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_flows_timestamp ON flows(timestamp_ns);
CREATE INDEX IF NOT EXISTS idx_flows_agent_id ON flows(agent_id);
CREATE INDEX IF NOT EXISTS idx_flows_src_ip ON flows(src_ip);
CREATE INDEX IF NOT EXISTS idx_flows_dst_ip ON flows(dst_ip);
CREATE INDEX IF NOT EXISTS idx_flows_created_at ON flows(created_at);

-- JSONB indexes for label-based queries
CREATE INDEX IF NOT EXISTS idx_flows_source_labels ON flows USING GIN(source_labels);
CREATE INDEX IF NOT EXISTS idx_flows_dest_labels ON flows USING GIN(dest_labels);

-- Composite index for time-based queries
CREATE INDEX IF NOT EXISTS idx_flows_timestamp_agent ON flows(timestamp_ns, agent_id);

COMMENT ON TABLE flows IS 'Stores flow events reported by agents';
COMMENT ON COLUMN flows.timestamp_ns IS 'Flow timestamp in nanoseconds (Unix epoch)';
COMMENT ON COLUMN flows.direction IS '0=INGRESS, 1=EGRESS';
COMMENT ON COLUMN flows.state IS '0=NEW, 1=ESTABLISHED, 2=CLOSING, 3=CLOSED';
COMMENT ON COLUMN flows.policy_action IS '0=ALLOW, 1=DENY, 2=LOG';

-- =================================================================
-- Policies Table (Network policies to be enforced by agents)
-- =================================================================
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

-- Index for priority-based policy matching
CREATE INDEX IF NOT EXISTS idx_policies_priority ON policies(priority DESC);

-- JSONB indexes for label-based policy matching
CREATE INDEX IF NOT EXISTS idx_policies_source_labels ON policies USING GIN(source_labels);
CREATE INDEX IF NOT EXISTS idx_policies_dest_labels ON policies USING GIN(dest_labels);

COMMENT ON TABLE policies IS 'Network policies for microsegmentation';
COMMENT ON COLUMN policies.protocol IS '0=TCP, 1=UDP, 2=ICMP, 3=ALL';
COMMENT ON COLUMN policies.action IS '0=ALLOW, 1=DENY, 2=LOG';
COMMENT ON COLUMN policies.priority IS 'Higher priority rules are evaluated first';

-- =================================================================
-- Policy Version Tracking (for synchronization)
-- =================================================================
CREATE TABLE IF NOT EXISTS policy_version (
    id INTEGER PRIMARY KEY DEFAULT 1,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1)
);

-- Initialize with version 1
INSERT INTO policy_version (id, version)
VALUES (1, 1)
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE policy_version IS 'Tracks global policy version for agent synchronization';

-- =================================================================
-- Agents Table (Registered agents)
-- =================================================================
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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT status_check CHECK (status IN ('active', 'inactive', 'unhealthy'))
);

-- Index for heartbeat-based queries
CREATE INDEX IF NOT EXISTS idx_agents_last_heartbeat ON agents(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);

COMMENT ON TABLE agents IS 'Registered microsegment agents';
COMMENT ON COLUMN agents.last_heartbeat IS 'Last heartbeat timestamp for health monitoring';
COMMENT ON COLUMN agents.status IS 'Agent status: active, inactive, unhealthy';

-- =================================================================
-- Agent Metrics Table (Latest metrics snapshot)
-- =================================================================
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

COMMENT ON TABLE agent_metrics IS 'Latest metrics snapshot from each agent';
COMMENT ON COLUMN agent_metrics.cpu_usage IS 'CPU usage percentage (0-100)';
COMMENT ON COLUMN agent_metrics.memory_usage IS 'Memory usage in bytes';

-- =================================================================
-- Events Table (Audit log for important system events)
-- =================================================================
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    source TEXT NOT NULL,
    agent_id TEXT,
    message TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for time-based event queries
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_agent_id ON events(agent_id);

COMMENT ON TABLE events IS 'System audit log for agent registration, policy changes, etc.';

-- =================================================================
-- Triggers for automatic timestamp updates
-- =================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for policies table
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
CREATE TRIGGER update_policies_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for agents table
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for policy_version table
DROP TRIGGER IF EXISTS update_policy_version_updated_at ON policy_version;
CREATE TRIGGER update_policy_version_updated_at
    BEFORE UPDATE ON policy_version
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =================================================================
-- Optional: TimescaleDB Hypertable (if TimescaleDB is installed)
-- =================================================================

-- Uncomment the following lines if you have TimescaleDB extension installed:
-- CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
-- SELECT create_hypertable('flows', 'created_at', if_not_exists => TRUE);
--
-- -- Set compression policy (compress data older than 7 days)
-- ALTER TABLE flows SET (
--   timescaledb.compress,
--   timescaledb.compress_segmentby = 'agent_id'
-- );
-- SELECT add_compression_policy('flows', INTERVAL '7 days');
--
-- -- Set retention policy (delete data older than 90 days)
-- SELECT add_retention_policy('flows', INTERVAL '90 days');

-- =================================================================
-- Initial Data (Optional)
-- =================================================================

-- Insert a default ALLOW ALL policy (rule_id = 0, lowest priority)
INSERT INTO policies (rule_id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority, description)
VALUES (0, '0.0.0.0/0', '0.0.0.0/0', 0, 0, 3, 0, 0, 'Default ALLOW ALL policy')
ON CONFLICT (rule_id) DO NOTHING;

-- =================================================================
-- Database statistics and vacuum settings
-- =================================================================

-- Enable auto-vacuum for high-write tables
ALTER TABLE flows SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE events SET (autovacuum_vacuum_scale_factor = 0.1);

-- =================================================================
-- Grant permissions (if using specific user)
-- =================================================================

-- Uncomment and adjust if you need to grant permissions:
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO microsegment_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO microsegment_user;

-- =================================================================
-- Migration complete
-- =================================================================

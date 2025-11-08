-- Initialize database schema for testing

-- Create flows table
CREATE TABLE IF NOT EXISTS flows (
    id SERIAL PRIMARY KEY,
    agent_id TEXT NOT NULL,
    src_ip INET NOT NULL,
    dst_ip INET NOT NULL,
    src_port INTEGER,
    dst_port INTEGER,
    protocol TEXT,
    direction TEXT,
    packet_count BIGINT DEFAULT 0,
    byte_count BIGINT DEFAULT 0,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    last_seen TIMESTAMPTZ,
    state TEXT,
    policy_id INTEGER,
    policy_action TEXT,
    source_labels JSONB,
    dest_labels JSONB,
    timestamp_ns BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_flows_agent_id ON flows(agent_id);
CREATE INDEX IF NOT EXISTS idx_flows_src_ip ON flows(src_ip);
CREATE INDEX IF NOT EXISTS idx_flows_dst_ip ON flows(dst_ip);
CREATE INDEX IF NOT EXISTS idx_flows_timestamp ON flows(timestamp_ns);
CREATE INDEX IF NOT EXISTS idx_flows_created_at ON flows(created_at);
CREATE INDEX IF NOT EXISTS idx_flows_source_labels ON flows USING GIN (source_labels);
CREATE INDEX IF NOT EXISTS idx_flows_dest_labels ON flows USING GIN (dest_labels);

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO microsegment_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO microsegment_user;

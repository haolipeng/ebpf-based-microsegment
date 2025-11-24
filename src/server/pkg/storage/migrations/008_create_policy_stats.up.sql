-- =================================================================
-- Create policy_stats table for storing policy enforcement statistics
-- =================================================================

CREATE TABLE IF NOT EXISTS policy_stats (
    id BIGSERIAL PRIMARY KEY,
    agent_id TEXT NOT NULL,
    rule_id INTEGER NOT NULL,
    packet_count BIGINT DEFAULT 0,
    byte_count BIGINT DEFAULT 0,
    flow_count BIGINT DEFAULT 0,
    hit_count BIGINT DEFAULT 0,
    last_match_time TIMESTAMP,
    report_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_policy_stats_agent_id ON policy_stats(agent_id);
CREATE INDEX IF NOT EXISTS idx_policy_stats_rule_id ON policy_stats(rule_id);
CREATE INDEX IF NOT EXISTS idx_policy_stats_report_time ON policy_stats(report_time);
CREATE INDEX IF NOT EXISTS idx_policy_stats_agent_rule ON policy_stats(agent_id, rule_id);

-- Comments
COMMENT ON TABLE policy_stats IS 'Policy enforcement statistics reported by agents';
COMMENT ON COLUMN policy_stats.agent_id IS 'Agent that reported these statistics';
COMMENT ON COLUMN policy_stats.rule_id IS 'Policy rule ID';
COMMENT ON COLUMN policy_stats.packet_count IS 'Number of packets matched by this rule';
COMMENT ON COLUMN policy_stats.byte_count IS 'Number of bytes matched by this rule';
COMMENT ON COLUMN policy_stats.flow_count IS 'Number of flows matched by this rule';
COMMENT ON COLUMN policy_stats.hit_count IS 'Number of times rule was evaluated';
COMMENT ON COLUMN policy_stats.last_match_time IS 'Timestamp of last match';
COMMENT ON COLUMN policy_stats.report_time IS 'Timestamp when this report was received';

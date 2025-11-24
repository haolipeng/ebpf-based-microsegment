-- =================================================================
-- Extend agent_metrics table for StatusReport persistence
-- =================================================================

-- Add new columns to store StatusReport data
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS policy_count INTEGER DEFAULT 0;
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS policy_version BIGINT DEFAULT 0;
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS workload_count INTEGER DEFAULT 0;
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS agent_status TEXT DEFAULT 'unknown';
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS uptime BIGINT DEFAULT 0;
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS errors TEXT[];
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE agent_metrics ADD COLUMN IF NOT EXISTS last_status_report TIMESTAMP;

-- Add constraint for agent_status values
ALTER TABLE agent_metrics DROP CONSTRAINT IF EXISTS agent_metrics_status_check;
ALTER TABLE agent_metrics ADD CONSTRAINT agent_metrics_status_check
    CHECK (agent_status IN ('unknown', 'starting', 'running', 'degraded', 'stopping', 'error'));

-- Add comments
COMMENT ON COLUMN agent_metrics.policy_count IS 'Number of policy rules currently enforced';
COMMENT ON COLUMN agent_metrics.policy_version IS 'Current policy version on this agent';
COMMENT ON COLUMN agent_metrics.workload_count IS 'Number of workloads registered on this host';
COMMENT ON COLUMN agent_metrics.agent_status IS 'Agent status: unknown, starting, running, degraded, stopping, error';
COMMENT ON COLUMN agent_metrics.uptime IS 'Agent uptime in seconds';
COMMENT ON COLUMN agent_metrics.errors IS 'Error messages or warnings from agent';
COMMENT ON COLUMN agent_metrics.metadata IS 'Additional status information as JSON';
COMMENT ON COLUMN agent_metrics.last_status_report IS 'Timestamp of last status report';

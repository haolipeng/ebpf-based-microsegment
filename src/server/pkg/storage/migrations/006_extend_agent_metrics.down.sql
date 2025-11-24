-- =================================================================
-- Rollback: Remove StatusReport columns from agent_metrics
-- =================================================================

ALTER TABLE agent_metrics DROP CONSTRAINT IF EXISTS agent_metrics_status_check;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS policy_count;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS policy_version;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS workload_count;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS agent_status;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS uptime;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS errors;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS metadata;
ALTER TABLE agent_metrics DROP COLUMN IF EXISTS last_status_report;

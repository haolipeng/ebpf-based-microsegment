-- =================================================================
-- Rollback: Drop policy_stats table
-- =================================================================

DROP INDEX IF EXISTS idx_policy_stats_agent_rule;
DROP INDEX IF EXISTS idx_policy_stats_report_time;
DROP INDEX IF EXISTS idx_policy_stats_rule_id;
DROP INDEX IF EXISTS idx_policy_stats_agent_id;
DROP TABLE IF EXISTS policy_stats;

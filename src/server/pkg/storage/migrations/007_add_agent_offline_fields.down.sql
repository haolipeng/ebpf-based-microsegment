-- =================================================================
-- Rollback: Remove offline tracking fields from agents table
-- =================================================================

DROP INDEX IF EXISTS idx_agents_offline_at;
ALTER TABLE agents DROP COLUMN IF EXISTS offline_at;
ALTER TABLE agents DROP COLUMN IF EXISTS offline_reason;

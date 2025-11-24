-- =================================================================
-- Add offline tracking fields to agents table
-- =================================================================

-- Add offline_at timestamp to record when agent went offline
ALTER TABLE agents ADD COLUMN IF NOT EXISTS offline_at TIMESTAMP;

-- Add offline_reason to record why agent went offline
ALTER TABLE agents ADD COLUMN IF NOT EXISTS offline_reason TEXT;

-- Add comments
COMMENT ON COLUMN agents.offline_at IS 'Timestamp when agent was marked offline/inactive';
COMMENT ON COLUMN agents.offline_reason IS 'Reason for agent going offline (e.g., graceful shutdown, timeout, error)';

-- Add index for querying offline agents
CREATE INDEX IF NOT EXISTS idx_agents_offline_at ON agents(offline_at) WHERE offline_at IS NOT NULL;

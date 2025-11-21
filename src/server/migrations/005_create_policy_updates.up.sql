-- Migration: 005_create_policy_updates
-- Description: Create policy_updates table for storing incremental policy changes (Issue #60)
-- Author: Claude Code
-- Date: 2025-11-21

-- Create policy updates table for tracking incremental policy changes
CREATE TABLE IF NOT EXISTS policy_updates (
    id BIGSERIAL PRIMARY KEY,

    -- Update metadata
    version BIGINT NOT NULL,
    update_type VARCHAR(20) NOT NULL,  -- 'UPDATE_ADD', 'UPDATE_MODIFY', 'UPDATE_DELETE'

    -- Policy reference
    rule_id INTEGER NOT NULL,

    -- Policy data (full policy for ADD/MODIFY, NULL for DELETE)
    policy_data JSONB,

    -- Timestamps
    timestamp BIGINT NOT NULL,  -- Unix nanoseconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add comments for documentation
COMMENT ON TABLE policy_updates IS 'Log of incremental policy changes for efficient agent synchronization (Task 4.2)';
COMMENT ON COLUMN policy_updates.version IS 'Global policy version number after this update';
COMMENT ON COLUMN policy_updates.update_type IS 'Type of change: UPDATE_ADD, UPDATE_MODIFY, UPDATE_DELETE';
COMMENT ON COLUMN policy_updates.rule_id IS 'Policy rule ID being changed';
COMMENT ON COLUMN policy_updates.policy_data IS 'Full policy object for ADD/MODIFY, NULL for DELETE';
COMMENT ON COLUMN policy_updates.timestamp IS 'Update timestamp in Unix nanoseconds';

-- Create indexes for efficient querying
CREATE INDEX idx_policy_updates_version ON policy_updates(version ASC);
CREATE INDEX idx_policy_updates_timestamp ON policy_updates(timestamp DESC);
CREATE INDEX idx_policy_updates_rule_id ON policy_updates(rule_id);
CREATE INDEX idx_policy_updates_type ON policy_updates(update_type);

-- Create composite index for common query pattern (get updates since version X)
CREATE INDEX idx_policy_updates_version_time ON policy_updates(version ASC, timestamp ASC);

-- Add check constraint for update_type
ALTER TABLE policy_updates ADD CONSTRAINT chk_update_type
    CHECK (update_type IN ('UPDATE_ADD', 'UPDATE_MODIFY', 'UPDATE_DELETE'));

-- Add partial index for non-DELETE updates (where policy_data should exist)
CREATE INDEX idx_policy_updates_with_data ON policy_updates(version)
    WHERE update_type IN ('UPDATE_ADD', 'UPDATE_MODIFY') AND policy_data IS NOT NULL;

-- Create GIN index for JSONB policy_data queries
CREATE INDEX idx_policy_updates_policy_data ON policy_updates USING GIN(policy_data);

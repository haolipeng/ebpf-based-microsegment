-- Migration: 002_add_process_policy_fields
-- Description: Add process-level policy matching fields to policies table (Issue #51)
-- Author: Claude Code
-- Date: 2025-11-20

-- Add process matching fields to policies table
ALTER TABLE policies ADD COLUMN IF NOT EXISTS process_name VARCHAR(255);
ALTER TABLE policies ADD COLUMN IF NOT EXISTS process_path VARCHAR(1024);
ALTER TABLE policies ADD COLUMN IF NOT EXISTS match_mode INTEGER DEFAULT 0;

-- Add comments for documentation
COMMENT ON COLUMN policies.process_name IS 'Process command name for process-level matching (e.g., nginx, curl). Empty = match all processes.';
COMMENT ON COLUMN policies.process_path IS 'Process executable path prefix for path-based matching (e.g., /usr/bin/, /tmp/). Empty = match all paths.';
COMMENT ON COLUMN policies.match_mode IS 'Process matching mode: 0=EXACT, 1=PREFIX, 2=WILDCARD';

-- Create indexes for efficient process policy queries
CREATE INDEX IF NOT EXISTS idx_policies_process_name ON policies(process_name) WHERE process_name IS NOT NULL AND process_name != '';
CREATE INDEX IF NOT EXISTS idx_policies_process_path ON policies(process_path) WHERE process_path IS NOT NULL AND process_path != '';

-- Create composite index for process + network matching
CREATE INDEX IF NOT EXISTS idx_policies_process_network ON policies(process_name, dst_port, protocol)
    WHERE process_name IS NOT NULL AND process_name != '';

-- Add check constraint to ensure valid match_mode values
ALTER TABLE policies ADD CONSTRAINT chk_match_mode CHECK (match_mode >= 0 AND match_mode <= 2);

-- Migration: 002_add_process_policy_fields (ROLLBACK)
-- Description: Remove process-level policy fields from policies table
-- Author: Claude Code
-- Date: 2025-11-20

-- Drop indexes
DROP INDEX IF EXISTS idx_policies_process_network;
DROP INDEX IF EXISTS idx_policies_process_path;
DROP INDEX IF EXISTS idx_policies_process_name;

-- Drop check constraint
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_match_mode;

-- Drop columns
ALTER TABLE policies DROP COLUMN IF EXISTS match_mode;
ALTER TABLE policies DROP COLUMN IF EXISTS process_path;
ALTER TABLE policies DROP COLUMN IF EXISTS process_name;

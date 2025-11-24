-- Migration: 003_add_process_info_to_flows (ROLLBACK)
-- Description: Remove process information fields from flows table
-- Author: Claude Code
-- Date: 2025-11-20

-- Drop indexes
DROP INDEX IF EXISTS idx_flows_suspicious_path;
DROP INDEX IF EXISTS idx_flows_process_time;
DROP INDEX IF EXISTS idx_flows_suspicious;
DROP INDEX IF EXISTS idx_flows_container_id;
DROP INDEX IF EXISTS idx_flows_comm;
DROP INDEX IF EXISTS idx_flows_exe_path;
DROP INDEX IF EXISTS idx_flows_pid;

-- Drop columns
ALTER TABLE flows DROP COLUMN IF EXISTS container_id;
ALTER TABLE flows DROP COLUMN IF EXISTS is_suspicious;
ALTER TABLE flows DROP COLUMN IF EXISTS start_time;
ALTER TABLE flows DROP COLUMN IF EXISTS cmdline;
ALTER TABLE flows DROP COLUMN IF EXISTS exe_path;
ALTER TABLE flows DROP COLUMN IF EXISTS comm;
ALTER TABLE flows DROP COLUMN IF EXISTS gid;
ALTER TABLE flows DROP COLUMN IF EXISTS uid;
ALTER TABLE flows DROP COLUMN IF EXISTS ppid;
ALTER TABLE flows DROP COLUMN IF EXISTS pid;

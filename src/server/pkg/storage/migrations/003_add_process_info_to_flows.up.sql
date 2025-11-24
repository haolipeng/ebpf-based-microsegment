-- Migration: 003_add_process_info_to_flows
-- Description: Add process information fields to flows table (Issue #51)
-- Author: Claude Code
-- Date: 2025-11-20

-- Add process information fields to flows table
ALTER TABLE flows ADD COLUMN IF NOT EXISTS pid INTEGER;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS ppid INTEGER;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS uid INTEGER;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS gid INTEGER;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS comm VARCHAR(16);
ALTER TABLE flows ADD COLUMN IF NOT EXISTS exe_path VARCHAR(1024);
ALTER TABLE flows ADD COLUMN IF NOT EXISTS cmdline TEXT;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS start_time BIGINT;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS is_suspicious BOOLEAN DEFAULT FALSE;
ALTER TABLE flows ADD COLUMN IF NOT EXISTS container_id VARCHAR(64);

-- Add comments for documentation
COMMENT ON COLUMN flows.pid IS 'Process ID that generated the flow';
COMMENT ON COLUMN flows.ppid IS 'Parent process ID';
COMMENT ON COLUMN flows.uid IS 'User ID of the process';
COMMENT ON COLUMN flows.gid IS 'Group ID of the process';
COMMENT ON COLUMN flows.comm IS 'Process command name (16 bytes, kernel truncated)';
COMMENT ON COLUMN flows.exe_path IS 'Full executable path resolved from /proc/<pid>/exe';
COMMENT ON COLUMN flows.cmdline IS 'Command line arguments (optional)';
COMMENT ON COLUMN flows.start_time IS 'Process start timestamp (Unix nanoseconds) for PID reuse detection';
COMMENT ON COLUMN flows.is_suspicious IS 'Marked as suspicious by security validator (Issue #50)';
COMMENT ON COLUMN flows.container_id IS 'Container ID extracted from cgroup path';

-- Create indexes for efficient process-based flow queries
CREATE INDEX IF NOT EXISTS idx_flows_pid ON flows(pid) WHERE pid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_flows_exe_path ON flows(exe_path) WHERE exe_path IS NOT NULL AND exe_path != '';
CREATE INDEX IF NOT EXISTS idx_flows_comm ON flows(comm) WHERE comm IS NOT NULL AND comm != '';
CREATE INDEX IF NOT EXISTS idx_flows_container_id ON flows(container_id) WHERE container_id IS NOT NULL AND container_id != '';
CREATE INDEX IF NOT EXISTS idx_flows_suspicious ON flows(is_suspicious) WHERE is_suspicious = TRUE;

-- Create composite index for process + time range queries
CREATE INDEX IF NOT EXISTS idx_flows_process_time ON flows(pid, start_time) WHERE pid IS NOT NULL;

-- Create composite index for suspicious flows by path
CREATE INDEX IF NOT EXISTS idx_flows_suspicious_path ON flows(exe_path, is_suspicious)
    WHERE is_suspicious = TRUE AND exe_path IS NOT NULL;

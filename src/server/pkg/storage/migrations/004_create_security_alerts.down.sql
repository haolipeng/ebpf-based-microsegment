-- Migration: 004_create_security_alerts (ROLLBACK)
-- Description: Drop security_alerts table
-- Author: Claude Code
-- Date: 2025-11-20

-- Drop the security_alerts table
DROP TABLE IF EXISTS security_alerts CASCADE;

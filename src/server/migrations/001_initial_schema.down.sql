-- Migration: 001_initial_schema (ROLLBACK)
-- Description: Drop all tables and functions created by initial schema
-- Author: Claude Code
-- Date: 2025-11-06
-- WARNING: This will delete all data!

-- =================================================================
-- Drop triggers
-- =================================================================
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
DROP TRIGGER IF EXISTS update_policy_version_updated_at ON policy_version;

-- =================================================================
-- Drop function
-- =================================================================
DROP FUNCTION IF EXISTS update_updated_at_column();

-- =================================================================
-- Drop tables (in reverse dependency order)
-- =================================================================

-- Drop dependent tables first
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS agent_metrics CASCADE;
DROP TABLE IF EXISTS flows CASCADE;
DROP TABLE IF EXISTS policy_version CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS agents CASCADE;

-- =================================================================
-- Drop TimescaleDB extension (if needed)
-- =================================================================
-- Uncomment if you want to completely remove TimescaleDB:
-- DROP EXTENSION IF EXISTS timescaledb CASCADE;

-- =================================================================
-- Rollback complete
-- =================================================================

#!/bin/bash
# Database Upgrade Script: Pre-1.0.0 to v1.0.0
# Description: Upgrades existing database to support label-based policies
# Date: 2025-01-05

set -e

DB_PATH="${DB_PATH:-/var/lib/ebpf-agent/agent.db}"

echo "==================================================================="
echo "  eBPF Microsegmentation - Database Upgrade to v1.0.0"
echo "==================================================================="
echo "Database path: $DB_PATH"
echo ""

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo "ERROR: Database not found at $DB_PATH"
    echo "For fresh installation, use: ./v1.0.0_initial_schema.sh"
    exit 1
fi

# Check current schema version
CURRENT_VERSION=$(sqlite3 "$DB_PATH" "SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1;" 2>/dev/null || echo "pre-1.0.0")

echo "Current schema version: $CURRENT_VERSION"

if [ "$CURRENT_VERSION" = "1.0.0" ]; then
    echo "Database is already at version 1.0.0. No upgrade needed."
    exit 0
fi

echo ""
echo "WARNING: This will modify your database schema!"
read -p "Continue with upgrade? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Upgrade cancelled."
    exit 0
fi

# Create backup
echo ""
echo "Creating backup..."
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$DB_PATH" "$BACKUP_PATH"
echo "Backup created at: $BACKUP_PATH"

echo ""
echo "Upgrading database schema to v1.0.0..."

sqlite3 "$DB_PATH" <<'EOF'
-- Enable foreign key support
PRAGMA foreign_keys=ON;

-- Set performance optimizations
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=-64000;
PRAGMA temp_store=MEMORY;

-- Add workloads table
CREATE TABLE IF NOT EXISTS workloads (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host_id TEXT NOT NULL,
    ips TEXT NOT NULL,
    macs TEXT NOT NULL,
    labels TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workload_host ON workloads(host_id);
CREATE INDEX IF NOT EXISTS idx_workload_created ON workloads(created_at);

-- Add groups table
CREATE TABLE IF NOT EXISTS groups (
    name TEXT PRIMARY KEY,
    description TEXT,
    selectors TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);

-- Add policy_rules table
CREATE TABLE IF NOT EXISTS policy_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    from_group TEXT NOT NULL,
    to_group TEXT NOT NULL,
    ports TEXT NOT NULL,
    action TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_group) REFERENCES groups(name) ON DELETE CASCADE,
    FOREIGN KEY (to_group) REFERENCES groups(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_policy_rule_name ON policy_rules(name);
CREATE INDEX IF NOT EXISTS idx_policy_rule_from_group ON policy_rules(from_group);
CREATE INDEX IF NOT EXISTS idx_policy_rule_to_group ON policy_rules(to_group);
CREATE INDEX IF NOT EXISTS idx_policy_rule_enabled ON policy_rules(enabled);

-- Add policy_compilation metadata table
CREATE TABLE IF NOT EXISTS policy_compilation (
    compiled_rule_id INTEGER PRIMARY KEY,
    source_rule_id INTEGER NOT NULL,
    from_group TEXT NOT NULL,
    to_group TEXT NOT NULL,
    from_workload_id TEXT NOT NULL,
    to_workload_id TEXT NOT NULL,
    compilation_time DATETIME NOT NULL,
    compiler_version TEXT NOT NULL,
    FOREIGN KEY (compiled_rule_id) REFERENCES policies(rule_id) ON DELETE CASCADE,
    FOREIGN KEY (source_rule_id) REFERENCES policy_rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_policy_compilation_source ON policy_compilation(source_rule_id);
CREATE INDEX IF NOT EXISTS idx_policy_compilation_from_group ON policy_compilation(from_group);
CREATE INDEX IF NOT EXISTS idx_policy_compilation_to_group ON policy_compilation(to_group);
CREATE INDEX IF NOT EXISTS idx_policy_compilation_from_workload ON policy_compilation(from_workload_id);
CREATE INDEX IF NOT EXISTS idx_policy_compilation_to_workload ON policy_compilation(to_workload_id);

-- Create schema_version table if it doesn't exist
CREATE TABLE IF NOT EXISTS schema_version (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- Record the upgrade
INSERT OR IGNORE INTO schema_version (version, description)
VALUES ('1.0.0', 'Upgraded to label-based policy system from ' || 'pre-1.0.0');

-- Analyze tables for query optimizer
ANALYZE;

EOF

echo ""
echo "==================================================================="
echo "  Upgrade Complete!"
echo "==================================================================="
echo ""

# Display schema version
echo "Applied schema versions:"
sqlite3 "$DB_PATH" "SELECT version, applied_at, description FROM schema_version ORDER BY applied_at;"

echo ""
echo "New tables added:"
echo "  - workloads (for workload registration)"
echo "  - groups (for label-based grouping)"
echo "  - policy_rules (for high-level policy rules)"
echo "  - policy_compilation (for compilation metadata)"
echo ""

echo "Existing tables preserved:"
echo "  - policies (compiled IP-based rules)"
echo "  - flows (network flow data)"
echo ""

echo "Database file: $DB_PATH"
echo "Backup file: $BACKUP_PATH"
echo "Database size: $(du -h "$DB_PATH" | cut -f1)"
echo ""
echo "Upgrade successful!"
echo ""
echo "Next steps:"
echo "  1. Register workloads via API: POST /api/v1/workloads"
echo "  2. Create groups via API: POST /api/v1/groups"
echo "  3. Define policy rules via API: POST /api/v1/policy-rules"
echo "  4. View compiled policies: GET /api/v1/policy-rules/{id}/compiled"

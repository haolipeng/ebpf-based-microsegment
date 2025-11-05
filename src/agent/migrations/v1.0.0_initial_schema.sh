#!/bin/bash
# Database Migration Script: v1.0.0 Initial Schema
# Description: Creates the initial database schema for label-based policy system
# Date: 2025-01-05

set -e

DB_PATH="${DB_PATH:-/var/lib/ebpf-agent/agent.db}"

echo "==================================================================="
echo "  eBPF Microsegmentation - Database Migration v1.0.0"
echo "==================================================================="
echo "Database path: $DB_PATH"
echo ""

# Create directory if it doesn't exist
DB_DIR=$(dirname "$DB_PATH")
if [ ! -d "$DB_DIR" ]; then
    echo "Creating directory: $DB_DIR"
    mkdir -p "$DB_DIR"
fi

# Check if database already exists
if [ -f "$DB_PATH" ]; then
    echo "WARNING: Database already exists at $DB_PATH"
    echo "Creating backup before migration..."
    BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$DB_PATH" "$BACKUP_PATH"
    echo "Backup created at: $BACKUP_PATH"
    echo ""
fi

echo "Creating database schema..."

sqlite3 "$DB_PATH" <<'EOF'
-- Enable foreign key support
PRAGMA foreign_keys=ON;

-- Set performance optimizations
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=-64000;
PRAGMA temp_store=MEMORY;

-- =========================================================================
-- Table: workloads
-- Description: Stores workload registration and label information
-- =========================================================================
CREATE TABLE IF NOT EXISTS workloads (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host_id TEXT NOT NULL,
    ips TEXT NOT NULL,                -- JSON array: ["10.0.1.10", "10.0.1.11"]
    macs TEXT NOT NULL,               -- JSON array: ["aa:bb:cc:dd:ee:ff"]
    labels TEXT NOT NULL,             -- JSON object: {"app":"web","tier":"frontend"}
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workload_host ON workloads(host_id);
CREATE INDEX IF NOT EXISTS idx_workload_created ON workloads(created_at);

-- =========================================================================
-- Table: groups
-- Description: Stores group definitions with label selectors
-- =========================================================================
CREATE TABLE IF NOT EXISTS groups (
    name TEXT PRIMARY KEY,
    description TEXT,
    selectors TEXT NOT NULL,          -- JSON array of selector objects
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);

-- =========================================================================
-- Table: policy_rules
-- Description: High-level policy rules using group names
-- =========================================================================
CREATE TABLE IF NOT EXISTS policy_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    from_group TEXT NOT NULL,
    to_group TEXT NOT NULL,
    ports TEXT NOT NULL,              -- JSON array: [{"start":80,"end":80,"protocol":"tcp"}]
    action TEXT NOT NULL,             -- allow, deny, log
    priority INTEGER NOT NULL DEFAULT 100,
    enabled INTEGER NOT NULL DEFAULT 1,  -- 1=enabled, 0=disabled
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_group) REFERENCES groups(name) ON DELETE CASCADE,
    FOREIGN KEY (to_group) REFERENCES groups(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_policy_rule_name ON policy_rules(name);
CREATE INDEX IF NOT EXISTS idx_policy_rule_from_group ON policy_rules(from_group);
CREATE INDEX IF NOT EXISTS idx_policy_rule_to_group ON policy_rules(to_group);
CREATE INDEX IF NOT EXISTS idx_policy_rule_enabled ON policy_rules(enabled);

-- =========================================================================
-- Table: policies
-- Description: Compiled IP-based firewall rules (loaded into eBPF)
-- =========================================================================
CREATE TABLE IF NOT EXISTS policies (
    rule_id INTEGER PRIMARY KEY,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL,
    src_port INTEGER NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,           -- tcp, udp, icmp, any
    action TEXT NOT NULL,             -- allow, deny, log
    priority INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_src_ip ON policies(src_ip);
CREATE INDEX IF NOT EXISTS idx_dst_ip ON policies(dst_ip);
CREATE INDEX IF NOT EXISTS idx_protocol ON policies(protocol);
CREATE INDEX IF NOT EXISTS idx_priority ON policies(priority);

-- =========================================================================
-- Table: policy_compilation
-- Description: Metadata linking compiled policies to source rules
-- =========================================================================
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

-- =========================================================================
-- Table: flows (existing)
-- Description: Network flow capture data
-- =========================================================================
CREATE TABLE IF NOT EXISTS flows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL,
    src_port INTEGER NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    bytes_sent INTEGER NOT NULL,
    bytes_received INTEGER NOT NULL,
    packets_sent INTEGER NOT NULL,
    packets_received INTEGER NOT NULL,
    action TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_flow_timestamp ON flows(timestamp);
CREATE INDEX IF NOT EXISTS idx_flow_src_ip ON flows(src_ip);
CREATE INDEX IF NOT EXISTS idx_flow_dst_ip ON flows(dst_ip);

-- =========================================================================
-- Table: schema_version
-- Description: Track applied database migrations
-- =========================================================================
CREATE TABLE IF NOT EXISTS schema_version (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT OR IGNORE INTO schema_version (version, description)
VALUES ('1.0.0', 'Initial schema for label-based policy system');

-- Analyze tables for query optimizer
ANALYZE;

EOF

echo ""
echo "==================================================================="
echo "  Migration Complete!"
echo "==================================================================="
echo ""

# Display schema version
echo "Applied schema versions:"
sqlite3 "$DB_PATH" "SELECT version, applied_at, description FROM schema_version;"

echo ""
echo "Database statistics:"
sqlite3 "$DB_PATH" <<'EOF'
SELECT
    (SELECT COUNT(*) FROM workloads) as workloads,
    (SELECT COUNT(*) FROM groups) as groups,
    (SELECT COUNT(*) FROM policy_rules) as policy_rules,
    (SELECT COUNT(*) FROM policies) as compiled_policies,
    (SELECT COUNT(*) FROM flows) as flows;
EOF

echo ""
echo "Database file: $DB_PATH"
echo "Database size: $(du -h "$DB_PATH" | cut -f1)"
echo ""
echo "Migration successful!"

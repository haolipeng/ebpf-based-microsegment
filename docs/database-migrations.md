# Database Migration Guide

## Overview

This document describes the database schema for the eBPF-based microsegmentation system and provides migration scripts for setting up or upgrading the database.

## Database Engine

- **Engine**: SQLite
- **Location**: Configurable (default: `/var/lib/ebpf-agent/agent.db`)
- **Format**: SQLite 3.x compatible

## Schema Version

- **Current Version**: 1.0.0
- **Date**: 2025-01-05
- **Description**: Initial schema for label-based policy system

## Tables

### 1. workloads

Stores workload registration information.

```sql
CREATE TABLE IF NOT EXISTS workloads (
    id TEXT PRIMARY KEY,              -- Unique workload identifier (e.g., container ID)
    name TEXT NOT NULL,               -- Human-readable name
    host_id TEXT NOT NULL,            -- Host identifier where workload runs
    ips TEXT NOT NULL,                -- JSON array of IP addresses
    macs TEXT NOT NULL,               -- JSON array of MAC addresses
    labels TEXT NOT NULL,             -- JSON object of key-value labels
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workload_host ON workloads(host_id);
CREATE INDEX IF NOT EXISTS idx_workload_created ON workloads(created_at);
```

**Example Data**:
```json
{
  "id": "container-abc123",
  "name": "web-frontend-1",
  "host_id": "node-01",
  "ips": "[\"10.0.1.10\", \"192.168.1.5\"]",
  "labels": "{\"app\":\"web\",\"tier\":\"frontend\",\"env\":\"prod\"}"
}
```

### 2. groups

Stores group definitions with label selectors.

```sql
CREATE TABLE IF NOT EXISTS groups (
    name TEXT PRIMARY KEY,            -- Unique group name
    description TEXT,                 -- Optional description
    selectors TEXT NOT NULL,          -- JSON array of label selectors
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);
```

**Example Data**:
```json
{
  "name": "frontend",
  "description": "Frontend web servers",
  "selectors": "[{\"key\":\"tier\",\"operator\":\"=\",\"value\":\"frontend\"}]"
}
```

### 3. policy_rules

Stores high-level policy rules using group names.

```sql
CREATE TABLE IF NOT EXISTS policy_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,        -- Unique rule name
    description TEXT,                 -- Optional description
    from_group TEXT NOT NULL,         -- Source group name
    to_group TEXT NOT NULL,           -- Destination group name
    ports TEXT NOT NULL,              -- JSON array of port ranges
    action TEXT NOT NULL,             -- allow, deny, or log
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
```

**Example Data**:
```json
{
  "id": 1,
  "name": "frontend-to-backend",
  "description": "Allow frontend to access backend API",
  "from_group": "frontend",
  "to_group": "backend",
  "ports": "[{\"start\":80,\"end\":80,\"protocol\":\"tcp\"},{\"start\":443,\"end\":443,\"protocol\":\"tcp\"}]",
  "action": "allow",
  "priority": 100,
  "enabled": 1
}
```

### 4. policies

Stores compiled, IP-based firewall rules generated from policy_rules.

```sql
CREATE TABLE IF NOT EXISTS policies (
    rule_id INTEGER PRIMARY KEY,      -- Unique compiled rule ID (starts at 100000)
    src_ip TEXT NOT NULL,             -- Source IP address
    dst_ip TEXT NOT NULL,             -- Destination IP address
    src_port INTEGER NOT NULL,        -- Source port (0 for any)
    dst_port INTEGER NOT NULL,        -- Destination port
    protocol TEXT NOT NULL,           -- tcp, udp, icmp, any
    action TEXT NOT NULL,             -- allow, deny, log
    priority INTEGER NOT NULL,        -- Inherited from policy_rule
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_src_ip ON policies(src_ip);
CREATE INDEX IF NOT EXISTS idx_dst_ip ON policies(dst_ip);
CREATE INDEX IF NOT EXISTS idx_protocol ON policies(protocol);
CREATE INDEX IF NOT EXISTS idx_priority ON policies(priority);
```

**Example Data**:
```json
{
  "rule_id": 100001,
  "src_ip": "10.0.1.10",
  "dst_ip": "10.0.2.20",
  "src_port": 0,
  "dst_port": 80,
  "protocol": "tcp",
  "action": "allow",
  "priority": 100
}
```

### 5. policy_compilation

Stores metadata linking compiled policies back to their source rules and workloads.

```sql
CREATE TABLE IF NOT EXISTS policy_compilation (
    compiled_rule_id INTEGER PRIMARY KEY,
    source_rule_id INTEGER NOT NULL,     -- References policy_rules.id
    from_group TEXT NOT NULL,            -- Group name from source rule
    to_group TEXT NOT NULL,              -- Group name from source rule
    from_workload_id TEXT NOT NULL,      -- Specific workload ID that matched from_group
    to_workload_id TEXT NOT NULL,        -- Specific workload ID that matched to_group
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
```

**Example Data**:
```json
{
  "compiled_rule_id": 100001,
  "source_rule_id": 1,
  "from_group": "frontend",
  "to_group": "backend",
  "from_workload_id": "web-1",
  "to_workload_id": "api-1",
  "compilation_time": "2025-01-05T10:30:00Z",
  "compiler_version": "v1.0.0"
}
```

### 6. flows (existing)

Stores captured network flow data (not modified by this change).

```sql
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
```

## Relationships

```
workloads
    ├─> labels (used by groups for matching)
    └─> ips (used by compiler for policy expansion)

groups
    ├─> selectors (match workloads by labels)
    └─> [referenced by policy_rules]

policy_rules
    ├─> from_group FK → groups(name)
    ├─> to_group FK → groups(name)
    └─> [compiled to] → policies (via compiler)

policies
    ├─> [metadata in] → policy_compilation
    └─> [loaded into] → eBPF maps

policy_compilation
    ├─> compiled_rule_id FK → policies(rule_id)
    ├─> source_rule_id FK → policy_rules(id)
    ├─> from_workload_id → workloads(id)
    └─> to_workload_id → workloads(id)
```

## Migration Scripts

### Fresh Installation (v1.0.0)

Run this script to create a fresh database with all tables:

```bash
#!/bin/bash
# File: migrations/v1.0.0_initial_schema.sh

DB_PATH="${DB_PATH:-/var/lib/ebpf-agent/agent.db}"

echo "Creating database at: $DB_PATH"

sqlite3 "$DB_PATH" <<EOF
-- Workloads table
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

-- Groups table
CREATE TABLE IF NOT EXISTS groups (
    name TEXT PRIMARY KEY,
    description TEXT,
    selectors TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);

-- Policy rules table
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

-- Compiled policies table
CREATE TABLE IF NOT EXISTS policies (
    rule_id INTEGER PRIMARY KEY,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL,
    src_port INTEGER NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    action TEXT NOT NULL,
    priority INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_src_ip ON policies(src_ip);
CREATE INDEX IF NOT EXISTS idx_dst_ip ON policies(dst_ip);
CREATE INDEX IF NOT EXISTS idx_protocol ON policies(protocol);
CREATE INDEX IF NOT EXISTS idx_priority ON policies(priority);

-- Policy compilation metadata table
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

-- Flows table (existing)
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

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT OR IGNORE INTO schema_version (version, description)
VALUES ('1.0.0', 'Initial schema for label-based policy system');

EOF

echo "Database created successfully at: $DB_PATH"
sqlite3 "$DB_PATH" "SELECT version, applied_at, description FROM schema_version;"
```

### Upgrading from Pre-1.0.0 (No Label Support)

If upgrading from an older version that only had `policies` and `flows` tables:

```bash
#!/bin/bash
# File: migrations/upgrade_to_v1.0.0.sh

DB_PATH="${DB_PATH:-/var/lib/ebpf-agent/agent.db}"

echo "Upgrading database to v1.0.0 at: $DB_PATH"

# Backup existing database
cp "$DB_PATH" "${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"

sqlite3 "$DB_PATH" <<EOF
-- Add new tables (workloads, groups, policy_rules, policy_compilation)
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

CREATE TABLE IF NOT EXISTS groups (
    name TEXT PRIMARY KEY,
    description TEXT,
    selectors TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_group_created ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_group_updated ON groups(updated_at);

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

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT OR IGNORE INTO schema_version (version, description)
VALUES ('1.0.0', 'Upgraded to label-based policy system');

EOF

echo "Upgrade complete. Database backed up to: ${DB_PATH}.backup.*"
sqlite3 "$DB_PATH" "SELECT version, applied_at, description FROM schema_version;"
```

## Backup and Restore

### Backup

```bash
# Simple file copy
cp /var/lib/ebpf-agent/agent.db /backup/agent.db.$(date +%Y%m%d)

# Or SQL dump
sqlite3 /var/lib/ebpf-agent/agent.db .dump > /backup/agent_dump.sql
```

### Restore

```bash
# From file copy
cp /backup/agent.db.20250105 /var/lib/ebpf-agent/agent.db

# From SQL dump
sqlite3 /var/lib/ebpf-agent/agent.db < /backup/agent_dump.sql
```

## Data Integrity Checks

```sql
-- Check for orphaned policy_rules (referencing non-existent groups)
SELECT pr.id, pr.name, pr.from_group, pr.to_group
FROM policy_rules pr
WHERE NOT EXISTS (SELECT 1 FROM groups g WHERE g.name = pr.from_group)
   OR NOT EXISTS (SELECT 1 FROM groups g WHERE g.name = pr.to_group);

-- Check for orphaned policy_compilation records
SELECT pc.compiled_rule_id
FROM policy_compilation pc
WHERE NOT EXISTS (SELECT 1 FROM policies p WHERE p.rule_id = pc.compiled_rule_id)
   OR NOT EXISTS (SELECT 1 FROM policy_rules pr WHERE pr.id = pc.source_rule_id);

-- Count policies per rule
SELECT pr.name, COUNT(pc.compiled_rule_id) as compiled_count
FROM policy_rules pr
LEFT JOIN policy_compilation pc ON pr.id = pc.source_rule_id
GROUP BY pr.id, pr.name
ORDER BY compiled_count DESC;

-- Check for duplicate IP-based policies
SELECT src_ip, dst_ip, dst_port, protocol, COUNT(*) as count
FROM policies
GROUP BY src_ip, dst_ip, dst_port, protocol
HAVING count > 1;
```

## Performance Tuning

### Query Optimization

```sql
-- Analyze tables for query planner
ANALYZE;

-- Enable WAL mode for better concurrency
PRAGMA journal_mode=WAL;

-- Increase cache size (in KB)
PRAGMA cache_size=-64000;  -- 64MB cache

-- Optimize for writes
PRAGMA synchronous=NORMAL;
PRAGMA temp_store=MEMORY;
```

### Maintenance

```sql
-- Vacuum database to reclaim space
VACUUM;

-- Rebuild indexes
REINDEX;

-- Update statistics
ANALYZE;
```

## Monitoring Queries

```sql
-- Database statistics
SELECT
    (SELECT COUNT(*) FROM workloads) as workload_count,
    (SELECT COUNT(*) FROM groups) as group_count,
    (SELECT COUNT(*) FROM policy_rules) as policy_rule_count,
    (SELECT COUNT(*) FROM policies) as compiled_policy_count,
    (SELECT COUNT(*) FROM policy_compilation) as compilation_metadata_count;

-- Recent policy changes
SELECT name, action, enabled, updated_at
FROM policy_rules
ORDER BY updated_at DESC
LIMIT 10;

-- Top groups by workload membership
SELECT g.name, COUNT(w.id) as workload_count
FROM groups g
LEFT JOIN workloads w ON json_extract(w.labels, '$.' || json_extract(g.selectors, '$[0].key')) = json_extract(g.selectors, '$[0].value')
GROUP BY g.name
ORDER BY workload_count DESC;

-- Policy compilation explosion factor
SELECT
    pr.name,
    pr.from_group,
    pr.to_group,
    COUNT(pc.compiled_rule_id) as expanded_policies
FROM policy_rules pr
LEFT JOIN policy_compilation pc ON pr.id = pc.source_rule_id
GROUP BY pr.id
ORDER BY expanded_policies DESC;
```

## Troubleshooting

### Issue: Foreign Key Constraint Violation

```sql
-- Enable foreign keys (required for SQLite)
PRAGMA foreign_keys=ON;

-- Check constraints
PRAGMA foreign_key_check;
```

### Issue: Database Locked

- Check for long-running transactions
- Enable WAL mode: `PRAGMA journal_mode=WAL;`
- Increase busy timeout: `PRAGMA busy_timeout=5000;`

### Issue: Poor Query Performance

- Run `ANALYZE` to update statistics
- Check query plan: `EXPLAIN QUERY PLAN SELECT ...`
- Add appropriate indexes
- Consider denormalization for read-heavy workloads

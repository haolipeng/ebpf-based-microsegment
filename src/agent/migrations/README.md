# Database Migrations

This directory contains database migration scripts for the eBPF-based microsegmentation system.

## Available Migrations

### v1.0.0 - Initial Schema (Fresh Installation)

**File**: `v1.0.0_initial_schema.sh`

Creates a fresh database with the complete schema for label-based policy system.

**Usage**:
```bash
# Use default path (/var/lib/ebpf-agent/agent.db)
./v1.0.0_initial_schema.sh

# Use custom path
DB_PATH=/path/to/database.db ./v1.0.0_initial_schema.sh
```

**What it creates**:
- `workloads` table - Workload registration and labels
- `groups` table - Label-based groups
- `policy_rules` table - High-level policy rules
- `policies` table - Compiled IP-based rules
- `policy_compilation` table - Compilation metadata
- `flows` table - Network flow data
- `schema_version` table - Migration tracking
- All necessary indexes for performance

### Upgrade to v1.0.0 (Existing Database)

**File**: `upgrade_to_v1.0.0.sh`

Upgrades an existing database (pre-1.0.0) to support label-based policies.

**Usage**:
```bash
# Use default path
./upgrade_to_v1.0.0.sh

# Use custom path
DB_PATH=/path/to/database.db ./upgrade_to_v1.0.0.sh
```

**What it does**:
- Creates automatic backup before upgrade
- Adds new tables (workloads, groups, policy_rules, policy_compilation)
- Preserves existing data in policies and flows tables
- Updates schema_version tracking
- Interactive confirmation required

## Migration Best Practices

### Before Migration

1. **Backup your database**:
   ```bash
   cp /var/lib/ebpf-agent/agent.db /backup/agent.db.$(date +%Y%m%d)
   ```

2. **Check current schema version**:
   ```bash
   sqlite3 /var/lib/ebpf-agent/agent.db \
     "SELECT version, applied_at FROM schema_version ORDER BY applied_at DESC LIMIT 1;"
   ```

3. **Stop the agent service**:
   ```bash
   systemctl stop ebpf-agent
   ```

### Running Migrations

1. **Make scripts executable** (if not already):
   ```bash
   chmod +x *.sh
   ```

2. **Run the appropriate migration script**:
   ```bash
   # For fresh install
   ./v1.0.0_initial_schema.sh

   # For upgrade
   ./upgrade_to_v1.0.0.sh
   ```

3. **Verify migration success**:
   ```bash
   sqlite3 /var/lib/ebpf-agent/agent.db \
     "SELECT version, applied_at, description FROM schema_version;"
   ```

### After Migration

1. **Start the agent service**:
   ```bash
   systemctl start ebpf-agent
   ```

2. **Verify agent is running**:
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

3. **Check logs for errors**:
   ```bash
   journalctl -u ebpf-agent -f
   ```

## Rollback Procedure

If a migration fails or causes issues:

1. **Stop the agent**:
   ```bash
   systemctl stop ebpf-agent
   ```

2. **Restore from backup**:
   ```bash
   cp /backup/agent.db.20250105 /var/lib/ebpf-agent/agent.db
   ```

   Or use the automatic backup created by the migration script:
   ```bash
   # Find the backup
   ls -lt /var/lib/ebpf-agent/agent.db.backup.*

   # Restore the most recent backup
   cp /var/lib/ebpf-agent/agent.db.backup.YYYYMMDD_HHMMSS /var/lib/ebpf-agent/agent.db
   ```

3. **Restart the agent**:
   ```bash
   systemctl start ebpf-agent
   ```

## Testing Migrations

### Test Fresh Installation

```bash
# Create test database
DB_PATH=/tmp/test_fresh.db ./v1.0.0_initial_schema.sh

# Verify tables
sqlite3 /tmp/test_fresh.db ".tables"

# Check schema
sqlite3 /tmp/test_fresh.db ".schema"

# Clean up
rm /tmp/test_fresh.db*
```

### Test Upgrade

```bash
# Create a pre-1.0.0 database (with only policies and flows)
sqlite3 /tmp/test_old.db <<EOF
CREATE TABLE policies (
    rule_id INTEGER PRIMARY KEY,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL,
    src_port INTEGER NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    action TEXT NOT NULL,
    priority INTEGER NOT NULL
);

CREATE TABLE flows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL
);

INSERT INTO policies VALUES (1, '10.0.1.10', '10.0.2.20', 0, 80, 'tcp', 'allow', 100);
EOF

# Run upgrade
DB_PATH=/tmp/test_old.db ./upgrade_to_v1.0.0.sh

# Verify new tables exist
sqlite3 /tmp/test_old.db ".tables"

# Verify old data preserved
sqlite3 /tmp/test_old.db "SELECT * FROM policies;"

# Clean up
rm /tmp/test_old.db*
```

## Database Maintenance

### Regular Maintenance Commands

```bash
# Vacuum (reclaim space)
sqlite3 /var/lib/ebpf-agent/agent.db "VACUUM;"

# Rebuild indexes
sqlite3 /var/lib/ebpf-agent/agent.db "REINDEX;"

# Update statistics
sqlite3 /var/lib/ebpf-agent/agent.db "ANALYZE;"

# Check integrity
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA integrity_check;"

# Check foreign keys
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA foreign_key_check;"
```

### Performance Optimization

```bash
sqlite3 /var/lib/ebpf-agent/agent.db <<EOF
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=-64000;
PRAGMA temp_store=MEMORY;
ANALYZE;
EOF
```

## Troubleshooting

### Issue: "database is locked"

```bash
# Check for processes using the database
lsof /var/lib/ebpf-agent/agent.db

# Kill the process if necessary
systemctl stop ebpf-agent

# Enable WAL mode to reduce locking
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA journal_mode=WAL;"
```

### Issue: Foreign key constraint violations

```bash
# Enable foreign keys
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA foreign_keys=ON;"

# Check for violations
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA foreign_key_check;"
```

### Issue: Corrupted database

```bash
# Check integrity
sqlite3 /var/lib/ebpf-agent/agent.db "PRAGMA integrity_check;"

# If corrupted, restore from backup
cp /backup/agent.db.YYYYMMDD /var/lib/ebpf-agent/agent.db
```

## Schema Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2025-01-05 | Initial schema with label-based policy support |

## Additional Resources

- **Full documentation**: `../../docs/database-migrations.md`
- **API documentation**: `../../docs/api-label-based-policies.md`
- **Architecture**: `../../docs/architecture-comparison.md`
- **OpenSpec change**: `../../openspec/changes/add-label-based-policy/`

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review the full documentation in `docs/`
3. Check agent logs: `journalctl -u ebpf-agent -n 100`
4. Open an issue in the project repository

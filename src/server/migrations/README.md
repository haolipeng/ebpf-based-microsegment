# Database Migrations

This directory contains SQL migration scripts for the microsegment-server database schema.

## Migration Files

| Migration | Description | Status |
|-----------|-------------|--------|
| 001_initial_schema | Creates tables for flows, policies, agents, metrics, and events | ✅ Ready |

## Quick Start

### Prerequisites

- PostgreSQL 14+
- Database and user created:
  ```bash
  sudo -u postgres psql <<EOF
  CREATE DATABASE microsegment;
  CREATE USER microsegment_user WITH PASSWORD 'secret';
  GRANT ALL PRIVILEGES ON DATABASE microsegment TO microsegment_user;
  \q
  EOF
  ```

### Method 1: Using golang-migrate Tool (Recommended)

**Install golang-migrate**:
```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Run migrations**:
```bash
# From project root
cd src/server

# Set database URL
export DATABASE_URL="postgres://microsegment_user:secret@localhost:5432/microsegment?sslmode=disable"

# Run all up migrations
migrate -path migrations -database "$DATABASE_URL" up

# Check current version
migrate -path migrations -database "$DATABASE_URL" version

# Rollback last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Force to specific version (use carefully!)
migrate -path migrations -database "$DATABASE_URL" force 1
```

### Method 2: Using psql (Simple)

**Run up migration**:
```bash
psql -h localhost -U microsegment_user -d microsegment -f migrations/001_initial_schema.up.sql
```

**Run down migration** (rollback):
```bash
psql -h localhost -U microsegment_user -d microsegment -f migrations/001_initial_schema.down.sql
```

### Method 3: Automatic on Server Start

The server automatically creates schema on startup via `InitSchema()` in `pkg/storage/postgres.go`.

**Note**: This method doesn't use migration files, but the schema is identical.

## Verifying Migrations

After running migrations, verify the schema:

```bash
psql -h localhost -U microsegment_user -d microsegment
```

```sql
-- List all tables
\dt

-- Check flows table
\d flows

-- Check indexes
\di

-- Verify data
SELECT COUNT(*) FROM flows;
SELECT * FROM policies;
SELECT * FROM agents;

-- Check TimescaleDB (if installed)
SELECT * FROM timescaledb_information.hypertables;
```

## Migration Naming Convention

```
<version>_<description>.<up|down>.sql
```

Examples:
- `001_initial_schema.up.sql`
- `001_initial_schema.down.sql`
- `002_add_flows_indexes.up.sql`
- `003_add_timescaledb_hypertable.up.sql`

## Schema Overview

### Tables Created

1. **flows** - Flow events from agents
   - Primary key: `id` (BIGSERIAL)
   - Time column: `timestamp_ns` (nanoseconds)
   - JSONB columns: `source_labels`, `dest_labels`
   - 7 indexes for query performance

2. **policies** - Network policies
   - Primary key: `rule_id` (INTEGER)
   - JSONB columns: `source_labels`, `dest_labels`
   - Priority-based ordering

3. **policy_version** - Version tracking
   - Single row with global version number
   - Auto-updated via triggers

4. **agents** - Registered agents
   - Primary key: `agent_id` (TEXT)
   - Status tracking: active/inactive/unhealthy
   - Heartbeat monitoring

5. **agent_metrics** - Latest metrics
   - Primary key: `agent_id` (FK to agents)
   - CPU, memory, packet stats

6. **events** - Audit log
   - System events and agent activities
   - JSONB metadata column

### Triggers

- `update_policies_updated_at` - Auto-update `policies.updated_at`
- `update_agents_updated_at` - Auto-update `agents.updated_at`
- `update_policy_version_updated_at` - Auto-update `policy_version.updated_at`

## TimescaleDB (Optional)

The migration includes commented-out TimescaleDB setup. To enable:

1. Install TimescaleDB extension:
   ```bash
   sudo apt-get install postgresql-14-timescaledb
   # Or for macOS:
   brew install timescaledb
   ```

2. Enable in PostgreSQL:
   ```sql
   \c microsegment
   CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
   ```

3. Uncomment TimescaleDB section in `001_initial_schema.up.sql` (lines 165-178)

4. Rerun migration

**Benefits**:
- Automatic time-based partitioning
- Data compression (older than 7 days)
- Automatic retention policy (delete after 90 days)
- 10-100x better performance for time-series queries

## Troubleshooting

### Error: "relation already exists"

The migrations use `CREATE TABLE IF NOT EXISTS`, so it's safe to rerun. If you need a clean slate:

```bash
# Drop all tables
psql -h localhost -U microsegment_user -d microsegment -f migrations/001_initial_schema.down.sql

# Recreate
psql -h localhost -U microsegment_user -d microsegment -f migrations/001_initial_schema.up.sql
```

### Error: "permission denied"

Grant proper permissions:

```sql
GRANT ALL PRIVILEGES ON DATABASE microsegment TO microsegment_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO microsegment_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO microsegment_user;
```

### Error: "migrate: no change"

Migration is already at the latest version. This is normal.

### Check migration status

```bash
migrate -path migrations -database "$DATABASE_URL" version
```

## Adding New Migrations

1. Create new migration files:
   ```bash
   migrate create -ext sql -dir migrations -seq add_new_feature
   ```

2. Edit the generated files:
   - `002_add_new_feature.up.sql` - Add your changes
   - `002_add_new_feature.down.sql` - Add rollback

3. Test locally:
   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```

4. Commit both files to git

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run migrations
  run: |
    migrate -path src/server/migrations \
            -database "${{ secrets.DATABASE_URL }}" \
            up
```

### Docker Entrypoint

```bash
#!/bin/bash
set -e

# Wait for PostgreSQL
until pg_isready -h db -p 5432; do
  echo "Waiting for database..."
  sleep 2
done

# Run migrations
migrate -path /app/migrations -database "$DATABASE_URL" up

# Start server
exec /app/microsegment-server
```

## References

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [TimescaleDB documentation](https://docs.timescale.com/)

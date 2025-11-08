#!/bin/bash

# Database Migration Helper Script
# Usage: ./scripts/migrate.sh [up|down|status|create]

set -e

# Default configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-microsegment_user}"
DB_PASSWORD="${DB_PASSWORD:-secret}"
DB_NAME="${DB_NAME:-microsegment}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

# Construct database URL
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$(cd "${SCRIPT_DIR}/../migrations" && pwd)"

echo "==================================="
echo "Database Migration Tool"
echo "==================================="
echo "Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}"
echo "Migrations: ${MIGRATIONS_DIR}"
echo ""

# Check if migrate tool is installed
check_migrate_tool() {
    if ! command -v migrate &> /dev/null; then
        echo -e "${YELLOW}Warning: 'migrate' tool not found${NC}"
        echo "Install it with:"
        echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        echo ""
        echo "Falling back to psql..."
        return 1
    fi
    return 0
}

# Test database connection
test_connection() {
    echo "Testing database connection..."
    if psql "${DATABASE_URL}" -c "SELECT 1" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Database connection successful${NC}"
        return 0
    else
        echo -e "${RED}✗ Cannot connect to database${NC}"
        echo "Please check your connection settings"
        return 1
    fi
}

# Run migrations up
migrate_up() {
    if check_migrate_tool; then
        echo "Running migrations (UP)..."
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up
        echo -e "${GREEN}✓ Migrations applied successfully${NC}"
    else
        echo "Running migrations with psql..."
        for file in "${MIGRATIONS_DIR}"/*.up.sql; do
            if [ -f "$file" ]; then
                echo "Applying: $(basename "$file")"
                psql "${DATABASE_URL}" -f "$file"
            fi
        done
        echo -e "${GREEN}✓ Migrations applied successfully${NC}"
    fi
}

# Run migrations down
migrate_down() {
    if check_migrate_tool; then
        echo "Rolling back migrations (DOWN)..."
        echo -e "${YELLOW}Warning: This will rollback the last migration${NC}"
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down 1
            echo -e "${GREEN}✓ Migration rolled back${NC}"
        else
            echo "Cancelled"
        fi
    else
        echo "Rolling back with psql..."
        echo -e "${YELLOW}Warning: This will drop all tables!${NC}"
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            for file in "${MIGRATIONS_DIR}"/*.down.sql; do
                if [ -f "$file" ]; then
                    echo "Applying: $(basename "$file")"
                    psql "${DATABASE_URL}" -f "$file"
                fi
            done
            echo -e "${GREEN}✓ Rollback complete${NC}"
        else
            echo "Cancelled"
        fi
    fi
}

# Check migration status
migrate_status() {
    if check_migrate_tool; then
        echo "Migration status:"
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" version
    else
        echo "Checking database schema..."
        psql "${DATABASE_URL}" -c "\dt" 2>/dev/null || echo "No tables found"
    fi
}

# Create new migration
migrate_create() {
    local name=$1
    if [ -z "$name" ]; then
        echo -e "${RED}Error: Migration name required${NC}"
        echo "Usage: $0 create <migration_name>"
        exit 1
    fi

    if check_migrate_tool; then
        migrate create -ext sql -dir "${MIGRATIONS_DIR}" -seq "$name"
        echo -e "${GREEN}✓ Migration files created${NC}"
    else
        # Manual creation
        local next_version=$(ls -1 "${MIGRATIONS_DIR}" | grep -oP '^\d+' | sort -n | tail -1)
        next_version=$((next_version + 1))
        local padded_version=$(printf "%03d" $next_version)

        touch "${MIGRATIONS_DIR}/${padded_version}_${name}.up.sql"
        touch "${MIGRATIONS_DIR}/${padded_version}_${name}.down.sql"

        echo "-- Migration: ${padded_version}_${name}" > "${MIGRATIONS_DIR}/${padded_version}_${name}.up.sql"
        echo "-- Add your UP migration here" >> "${MIGRATIONS_DIR}/${padded_version}_${name}.up.sql"

        echo "-- Migration: ${padded_version}_${name} (ROLLBACK)" > "${MIGRATIONS_DIR}/${padded_version}_${name}.down.sql"
        echo "-- Add your DOWN migration here" >> "${MIGRATIONS_DIR}/${padded_version}_${name}.down.sql"

        echo -e "${GREEN}✓ Created:${NC}"
        echo "  ${MIGRATIONS_DIR}/${padded_version}_${name}.up.sql"
        echo "  ${MIGRATIONS_DIR}/${padded_version}_${name}.down.sql"
    fi
}

# Verify database schema
verify_schema() {
    echo "Verifying database schema..."
    echo ""

    echo "Tables:"
    psql "${DATABASE_URL}" -c "\dt" 2>/dev/null || echo "No tables found"

    echo ""
    echo "Indexes:"
    psql "${DATABASE_URL}" -c "\di" 2>/dev/null | grep -E "idx_|PRIMARY" || echo "No indexes found"

    echo ""
    echo "Row counts:"
    for table in flows policies agents agent_metrics events policy_version; do
        count=$(psql "${DATABASE_URL}" -t -c "SELECT COUNT(*) FROM $table" 2>/dev/null || echo "N/A")
        echo "  $table: $count"
    done
}

# Show help
show_help() {
    cat << EOF
Usage: $0 [command]

Commands:
  up         Apply all pending migrations
  down       Rollback the last migration (requires confirmation)
  status     Show current migration version
  create     Create a new migration file
  verify     Verify database schema and show statistics
  help       Show this help message

Environment Variables:
  DB_HOST      Database host (default: localhost)
  DB_PORT      Database port (default: 5432)
  DB_USER      Database user (default: microsegment_user)
  DB_PASSWORD  Database password (default: secret)
  DB_NAME      Database name (default: microsegment)
  DB_SSLMODE   SSL mode (default: disable)

Examples:
  # Apply all migrations
  ./scripts/migrate.sh up

  # Check migration status
  ./scripts/migrate.sh status

  # Create new migration
  ./scripts/migrate.sh create add_indexes

  # Verify schema
  ./scripts/migrate.sh verify

  # Use custom database
  DB_HOST=prod-db DB_PASSWORD=secure ./scripts/migrate.sh up

EOF
}

# Main command dispatcher
case "${1:-help}" in
    up)
        test_connection && migrate_up
        ;;
    down)
        test_connection && migrate_down
        ;;
    status)
        test_connection && migrate_status
        ;;
    create)
        migrate_create "$2"
        ;;
    verify)
        test_connection && verify_schema
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}Error: Unknown command '$1'${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac

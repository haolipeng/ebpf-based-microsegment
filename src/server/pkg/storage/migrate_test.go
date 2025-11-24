package storage

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationsEmbedded(t *testing.T) {
	// Test that migrations are properly embedded
	entries, err := migrationsFS.ReadDir("migrations")
	require.NoError(t, err, "Should read migrations directory")
	assert.Greater(t, len(entries), 0, "Should have migration files")

	// Verify we have both up and down migrations
	hasUp := false
	hasDown := false
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 7 && name[len(name)-7:] == ".up.sql" {
			hasUp = true
		}
		if len(name) > 9 && name[len(name)-9:] == ".down.sql" {
			hasDown = true
		}
	}
	assert.True(t, hasUp, "Should have .up.sql migrations")
	assert.True(t, hasDown, "Should have .down.sql migrations")

	t.Logf("Found %d migration files", len(entries))
}

func TestMigrationsFileContent(t *testing.T) {
	// Test that initial migration file contains expected content
	content, err := migrationsFS.ReadFile("migrations/001_initial_schema.up.sql")
	require.NoError(t, err, "Should read initial migration file")
	assert.Contains(t, string(content), "CREATE TABLE", "Should contain CREATE TABLE")
	assert.Contains(t, string(content), "flows", "Should create flows table")
	assert.Contains(t, string(content), "policies", "Should create policies table")
	assert.Contains(t, string(content), "agents", "Should create agents table")
}

func TestNewMigratorError(t *testing.T) {
	// Test that newMigrator returns error for invalid connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect connection error when trying to create migrator
	mock.ExpectQuery("SELECT CURRENT_DATABASE()").WillReturnError(sql.ErrConnDone)

	_, err = newMigrator(db)
	assert.Error(t, err, "Should return error for invalid connection")
}

func TestGetMigrationVersionNoMigrations(t *testing.T) {
	// This test would require a real database connection
	// For unit tests, we verify the function signature and basic error handling
	t.Skip("Requires real PostgreSQL database for integration testing")
}

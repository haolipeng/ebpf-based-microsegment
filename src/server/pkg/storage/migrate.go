package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sirupsen/logrus"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateUp runs all pending database migrations.
// It uses embedded SQL files from the migrations directory.
func MigrateUp(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logrus.Info("Database schema is up to date")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	logrus.Infof("Database migrated to version %d (dirty: %v)", version, dirty)
	return nil
}

// MigrateDown rolls back the last migration.
func MigrateDown(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logrus.Info("No migrations to rollback")
			return nil
		}
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	logrus.Infof("Database rolled back to version %d (dirty: %v)", version, dirty)
	return nil
}

// MigrateToVersion migrates to a specific version.
// Use version 0 to rollback all migrations.
func MigrateToVersion(db *sql.DB, version uint) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logrus.Infof("Database already at version %d", version)
			return nil
		}
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	logrus.Infof("Database migrated to version %d", version)
	return nil
}

// GetMigrationVersion returns the current migration version.
func GetMigrationVersion(db *sql.DB) (uint, bool, error) {
	m, err := newMigrator(db)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// MigrateFix forces the migration version without running migrations.
// Use this to fix a dirty database state.
func MigrateFix(db *sql.DB, version int) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}

	logrus.Infof("Migration version forced to %d", version)
	return nil
}

// newMigrator creates a new migrate instance with embedded migrations.
func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	// Create source from embedded filesystem
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return m, nil
}

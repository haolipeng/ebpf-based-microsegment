package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// BunDB wraps the Bun database connection
type BunDB struct {
	*bun.DB
}

// NewBunDB creates a new Bun database connection from DSN
func NewBunDB(dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*BunDB, error) {
	// Create pgdriver connector
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	// Create sql.DB from connector
	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	sqldb.SetConnMaxLifetime(connMaxLifetime)

	// Create Bun DB
	db := bun.NewDB(sqldb, pgdialect.New())

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	logrus.Info("Connected to PostgreSQL database via Bun")
	return &BunDB{DB: db}, nil
}

// NewBunDBFromSQL wraps an existing *sql.DB with Bun
// Useful for reusing existing connection pools or for testing
func NewBunDBFromSQL(sqldb *sql.DB) *BunDB {
	db := bun.NewDB(sqldb, pgdialect.New())
	return &BunDB{DB: db}
}

// Close closes the database connection
func (db *BunDB) Close() error {
	return db.DB.Close()
}

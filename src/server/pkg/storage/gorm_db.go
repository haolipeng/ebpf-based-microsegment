package storage

import (
	"database/sql"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// newGormFromSQL wraps an existing *sql.DB with a GORM instance so that we can
// reuse the same connection pool across storage implementations.
func newGormFromSQL(sqlDB *sql.DB) (*gorm.DB, error) {
	return gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		DisableAutomaticPing: true,
	})
}

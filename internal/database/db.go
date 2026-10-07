package database

import (
	"fmt"
	"log"

	"github.com/ddmas26/inventory/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection using GORM and runs auto-migration.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Connected to PostgreSQL successfully")

	return db, nil
}

// EnableSQLLogging switches the GORM logger to Info mode so all queries
// are printed. Call this after migration / seeding to avoid startup noise.
func EnableSQLLogging(db *gorm.DB) {
	db.Logger = db.Logger.LogMode(logger.Info)
	log.Println("SQL query logging enabled")
}

// legacyIndexes are the single-column unique indexes that the multi-tenant
// change replaces with per-company composite indexes. AutoMigrate never drops
// indexes, so they are removed explicitly before it runs.
var legacyIndexes = []string{
	"idx_products_name",
	"idx_inventories_name",
	"idx_roles_name",
}

// Migrate runs auto-migration for all models.
func Migrate(db *gorm.DB) error {
	for _, idx := range legacyIndexes {
		if err := db.Exec("DROP INDEX IF EXISTS " + idx).Error; err != nil {
			return fmt.Errorf("drop legacy index %s: %w", idx, err)
		}
	}

	err := db.AutoMigrate(
		&Company{},
		&Product{},
		&ProductImage{},
		&Inventory{},
		&Stock{},
		&User{},
		&Role{},
		&Permission{},
		&PlatformUser{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}
	log.Println("Database migration completed")
	return nil
}

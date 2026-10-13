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

// legacyIndexes are indexes that must be dropped before AutoMigrate runs.
// AutoMigrate never drops or alters existing indexes, and "IF NOT EXISTS"
// silently keeps a wrong one in place, so they are removed explicitly.
//
//   - idx_products_name / idx_inventories_name / idx_roles_name were the old
//     single-column global unique indexes, replaced by per-company ones.
//   - idx_*_company_name were briefly created as unique on ("name") alone (a
//     model-tag bug). Dropping them lets AutoMigrate rebuild them correctly as
//     the composite (company_id, name) index.
var legacyIndexes = []string{
	"idx_products_name",
	"idx_inventories_name",
	"idx_roles_name",
	"idx_products_company_name",
	"idx_inventories_company_name",
	"idx_roles_company_name",
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

	if err := migrateProductIdentifiers(db); err != nil {
		return err
	}

	log.Println("Database migration completed")
	return nil
}

// migrateProductIdentifiers guarantees that every product carries a SKU and
// keeps SKU/barcode unique inside a company. Both steps are idempotent.
//
// The uniqueness is deliberately company-scoped: two tenants may legitimately
// use the same SKU or barcode for different products. Partial indexes are used
// (instead of a plain UNIQUE constraint) so that
//   - soft-deleted products do not permanently reserve an identifier, and
//   - an omitted (empty) barcode never collides with another omitted one.
func migrateProductIdentifiers(db *gorm.DB) error {
	// Backfill products that predate the SKU column with a deterministic,
	// collision-free identifier derived from the row UUID (e.g. "LEGACY-3F9A1C7D").
	if err := db.Exec(`
		UPDATE products
		SET sku = 'LEGACY-' || UPPER(SUBSTRING(REPLACE(id::text, '-', '') FROM 1 FOR 8))
		WHERE (sku IS NULL OR sku = '') AND deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("backfill product SKUs: %w", err)
	}

	identifiers := []struct{ index, column string }{
		{"idx_products_company_sku", "sku"},
		{"idx_products_company_barcode", "barcode"},
	}
	for _, ident := range identifiers {
		stmt := fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON products (company_id, %s) WHERE deleted_at IS NULL AND %s <> ''",
			ident.index, ident.column, ident.column,
		)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("create unique index %s: %w", ident.index, err)
		}
	}

	return nil
}

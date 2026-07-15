package database

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =============================================================================
// Repository wraps all database operations.
// =============================================================================

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// =============================================================================
// Product CRUD
// =============================================================================

// CreateProduct inserts a new product.
func (r *Repository) CreateProduct(p *Product) error {
	return r.db.Create(p).Error
}

// GetProductByID retrieves a product by its UUID (non-deleted).
func (r *Repository) GetProductByID(id uuid.UUID) (*Product, error) {
	var p Product
	err := r.db.Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("product %s: %w", id, ErrNotFound)
	}
	return &p, err
}

// GetProductByName retrieves a product by exact name match.
func (r *Repository) GetProductByName(name string) (*Product, error) {
	var p Product
	err := r.db.Where("name = ?", name).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("product %q: %w", name, ErrNotFound)
	}
	return &p, err
}

// ListProducts returns paginated products ordered by creation time.
func (r *Repository) ListProducts(offset, limit int) ([]Product, int64, error) {
	var products []Product
	var total int64

	if err := r.db.Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

// UpdateProduct updates name, description, and price of an existing product.
func (r *Repository) UpdateProduct(p *Product) error {
	result := r.db.Model(p).Select("name", "description", "price").Updates(p)
	if result.RowsAffected == 0 {
		return fmt.Errorf("product %s: %w", p.ID, ErrNotFound)
	}
	return result.Error
}

// DeleteProduct soft-deletes a product by ID.
func (r *Repository) DeleteProduct(id uuid.UUID) error {
	result := r.db.Delete(&Product{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("product %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// =============================================================================
// Inventory CRUD
// =============================================================================

// CreateInventory inserts a new inventory location.
func (r *Repository) CreateInventory(inv *Inventory) error {
	return r.db.Create(inv).Error
}

// GetInventoryByID retrieves an inventory by UUID.
func (r *Repository) GetInventoryByID(id uuid.UUID) (*Inventory, error) {
	var inv Inventory
	err := r.db.Where("id = ?", id).First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("inventory %s: %w", id, ErrNotFound)
	}
	return &inv, err
}

// GetInventoryByName retrieves an inventory by exact name match.
func (r *Repository) GetInventoryByName(name string) (*Inventory, error) {
	var inv Inventory
	err := r.db.Where("name = ?", name).First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("inventory %q: %w", name, ErrNotFound)
	}
	return &inv, err
}

// ListInventories returns paginated inventories.
func (r *Repository) ListInventories(offset, limit int) ([]Inventory, int64, error) {
	var inventories []Inventory
	var total int64

	if err := r.db.Model(&Inventory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&inventories).Error
	return inventories, total, err
}

// UpdateInventory updates the name, address, and coordinates.
func (r *Repository) UpdateInventory(inv *Inventory) error {
	result := r.db.Model(inv).Select("name", "address", "latitude", "longitude").Updates(inv)
	if result.RowsAffected == 0 {
		return fmt.Errorf("inventory %s: %w", inv.ID, ErrNotFound)
	}
	return result.Error
}

// DeleteInventory soft-deletes an inventory by ID.
func (r *Repository) DeleteInventory(id uuid.UUID) error {
	result := r.db.Delete(&Inventory{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("inventory %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// =============================================================================
// Stock Operations
// =============================================================================

// SetProductStock sets an absolute quantity (overwrites).
func (r *Repository) SetProductStock(inventoryID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	ip := Stock{
		InventoryID: inventoryID,
		ProductID:   productID,
		Quantity:    quantity,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "inventory_id"}, {Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity"}),
	}).Create(&ip).Error
}

// RemoveProductFromInventory deletes the inventory-product link entirely.
func (r *Repository) RemoveProductFromInventory(inventoryID, productID uuid.UUID) error {
	result := r.db.Where("inventory_id = ? AND product_id = ?", inventoryID, productID).
		Delete(&Stock{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("stock entry not found: %w", ErrNotFound)
	}
	return result.Error
}

// GetStock returns the quantity of a specific product at a specific inventory.
func (r *Repository) GetStock(inventoryID, productID uuid.UUID) (*Stock, error) {
	var ip Stock
	err := r.db.Where("inventory_id = ? AND product_id = ?", inventoryID, productID).First(&ip).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("stock entry not found: %w", ErrNotFound)
	}
	return &ip, err
}

// ListStock returns stock entries with optional filters, ordering, and pagination.
func (r *Repository) ListStock(filter dtos.ListStockFilter) ([]dtos.StockDto, int64, error) {
	query := r.db.Table("stock").
		Select("stock.*, products.name as product_name, inventories.name as inventory_name").
		Joins("LEFT JOIN products ON products.id = stock.product_id").
		Joins("LEFT JOIN inventories ON inventories.id = stock.inventory_id")

	if filter.InventoryID != nil {
		query = query.Where("stock.inventory_id = ?", *filter.InventoryID)
	}
	if filter.ProductID != nil {
		query = query.Where("stock.product_id = ?", *filter.ProductID)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("stock.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("stock.created_at <= ?", *filter.CreatedTo)
	}
	if filter.Search != "" {
		query = query.Where("products.name ILIKE ?", "%"+filter.Search+"%")
	}

	// Count total before pagination
	var total int64
	countQuery := r.db.Table("stock").
		Joins("LEFT JOIN products ON products.id = stock.product_id")
	if filter.InventoryID != nil {
		countQuery = countQuery.Where("stock.inventory_id = ?", *filter.InventoryID)
	}
	if filter.ProductID != nil {
		countQuery = countQuery.Where("stock.product_id = ?", *filter.ProductID)
	}
	if filter.CreatedFrom != nil {
		countQuery = countQuery.Where("stock.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		countQuery = countQuery.Where("stock.created_at <= ?", *filter.CreatedTo)
	}
	if filter.Search != "" {
		countQuery = countQuery.Where("products.name ILIKE ?", "%"+filter.Search+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Ordering
	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "stock.created_at"
	}
	sort := filter.Sort
	if sort != "asc" && sort != "ASC" {
		sort = "DESC"
	}
	query = query.Order(orderBy + " " + sort)

	// Pagination
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	query = query.Offset(filter.Offset).Limit(filter.Limit)

	type stockRow struct {
		Stock
		ProductName   string `json:"product_name"`
		InventoryName string `json:"inventory_name"`
	}

	var rows []stockRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	dtosItems := make([]dtos.StockDto, 0, len(rows))
	for _, row := range rows {
		dtosItems = append(dtosItems, dtos.StockDto{
			InventoryID:   row.InventoryID,
			ProductID:     row.ProductID,
			ProductName:   row.ProductName,
			InventoryName: row.InventoryName,
			Quantity:      row.Quantity,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}

	return dtosItems, total, nil
}

// =============================================================================
// Transactional Stock Operations (race-condition safe)
// =============================================================================

// DeductStock atomically subtracts quantity from a product at an inventory.
// Returns an error if stock would go negative.
func (r *Repository) DeductStock(inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("deduction amount must be positive")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var ip Stock

		// SELECT ... FOR UPDATE locks the row against concurrent writes
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("inventory_id = ? AND product_id = ?", inventoryID, productID).
			First(&ip).Error
		if err != nil {
			return fmt.Errorf("stock entry not found: %w", ErrNotFound)
		}

		if ip.Quantity < amount {
			return fmt.Errorf("insufficient stock: have %d, need %d", ip.Quantity, amount)
		}

		ip.Quantity -= amount
		return tx.Save(&ip).Error
	})
}

// AddStock atomically adds quantity to a product at an inventory.
// Creates a new stock entry if one doesn't exist yet.
func (r *Repository) AddStock(inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("addition amount must be positive")
	}

	ip := Stock{
		InventoryID: inventoryID,
		ProductID:   productID,
		Quantity:    amount,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "inventory_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"quantity": gorm.Expr("\"stock\".quantity + ?", amount)}),
	}).Create(&ip).Error
}

// TransferStock moves quantity from one inventory to another atomically.
func (r *Repository) TransferStock(fromInv, toInv, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		repo := NewRepository(tx)

		if err := repo.DeductStock(fromInv, productID, amount); err != nil {
			return fmt.Errorf("deduct from source: %w", err)
		}

		if err := repo.AddStock(toInv, productID, amount); err != nil {
			return fmt.Errorf("add to destination: %w", err)
		}

		return nil
	})
}

// =============================================================================
// Errors
// =============================================================================

var ErrNotFound = errors.New("record not found")

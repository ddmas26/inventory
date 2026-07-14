package database

import (
	"errors"
	"fmt"

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
// Stock Operations (InventoryProduct)
// =============================================================================

// AddProductToInventory links a product to an inventory with a quantity.
// Uses ON CONFLICT (upsert) — if the row already exists the quantity is added.
func (r *Repository) AddProductToInventory(inventoryID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	ip := InventoryProduct{
		InventoryID: inventoryID,
		ProductID:   productID,
		Quantity:    quantity,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "inventory_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"quantity": gorm.Expr("inventory_products.quantity + ?", quantity)}),
	}).Create(&ip).Error
}

// SetProductStock sets an absolute quantity (overwrites).
func (r *Repository) SetProductStock(inventoryID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	ip := InventoryProduct{
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
		Delete(&InventoryProduct{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("stock entry not found: %w", ErrNotFound)
	}
	return result.Error
}

// GetStock returns the quantity of a specific product at a specific inventory.
func (r *Repository) GetStock(inventoryID, productID uuid.UUID) (*InventoryProduct, error) {
	var ip InventoryProduct
	err := r.db.Where("inventory_id = ? AND product_id = ?", inventoryID, productID).First(&ip).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("stock entry not found: %w", ErrNotFound)
	}
	return &ip, err
}

// ListProductsAtInventory returns all products stocked at a given inventory with quantities.
func (r *Repository) ListProductsAtInventory(inventoryID uuid.UUID) ([]InventoryProduct, error) {
	var items []InventoryProduct
	err := r.db.Preload("Product").
		Where("inventory_id = ?", inventoryID).
		Find(&items).Error
	return items, err
}

// ListInventoriesForProduct returns all inventories that carry a given product.
func (r *Repository) ListInventoriesForProduct(productID uuid.UUID) ([]InventoryProduct, error) {
	var items []InventoryProduct
	err := r.db.Preload("Inventory").
		Where("product_id = ?", productID).
		Find(&items).Error
	return items, err
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
		var ip InventoryProduct

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
func (r *Repository) AddStock(inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("addition amount must be positive")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var ip InventoryProduct

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("inventory_id = ? AND product_id = ?", inventoryID, productID).
			First(&ip).Error
		if err != nil {
			return fmt.Errorf("stock entry not found: %w", ErrNotFound)
		}

		ip.Quantity += amount
		return tx.Save(&ip).Error
	})
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

		if err := repo.AddProductToInventory(toInv, productID, amount); err != nil {
			return fmt.Errorf("add to destination: %w", err)
		}

		return nil
	})
}

// =============================================================================
// Errors
// =============================================================================

var ErrNotFound = errors.New("record not found")

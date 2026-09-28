package database

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// orderProductImages sorts a product's images so the primary image comes first.
func orderProductImages(db *gorm.DB) *gorm.DB {
	return db.Order("product_images.is_primary DESC, product_images.created_at ASC")
}

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

// GetProductByID retrieves a product by its UUID (non-deleted), including its images.
func (r *Repository) GetProductByID(id uuid.UUID) (*Product, error) {
	var p Product
	err := r.db.Preload("Images", orderProductImages).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("product %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	p.ImageURL = p.PrimaryImageURL()
	return &p, nil
}

// GetProductByName retrieves a product by exact name match, including its images.
func (r *Repository) GetProductByName(name string) (*Product, error) {
	var p Product
	err := r.db.Preload("Images", orderProductImages).Where("name = ?", name).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("product %q: %w", name, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	p.ImageURL = p.PrimaryImageURL()
	return &p, nil
}

// ListProducts returns paginated products ordered by creation time, including their images.
func (r *Repository) ListProducts(offset, limit int) ([]Product, int64, error) {
	var products []Product
	var total int64

	if err := r.db.Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Preload("Images", orderProductImages).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	// Expose the primary image URL for list rendering.
	for i := range products {
		products[i].ImageURL = products[i].PrimaryImageURL()
	}

	return products, total, nil
}

// UpdateProduct updates name, description, price, and the low stock threshold of
// an existing product. Images are managed separately via ReplaceProductImages.
func (r *Repository) UpdateProduct(p *Product) error {
	result := r.db.Model(p).Select("name", "description", "price", "low_stock_threshold").Updates(p)
	if result.RowsAffected == 0 {
		return fmt.Errorf("product %s: %w", p.ID, ErrNotFound)
	}
	return result.Error
}

// ReplaceProductImages replaces the full set of images for a product inside a
// single transaction. Exactly one image is stored as the primary image: the one
// flagged by the caller, or — when none is flagged — one picked at random.
func (r *Repository) ReplaceProductImages(productID uuid.UUID, images []dtos.ProductImageInput) error {
	urls := make([]string, 0, len(images))
	primaryURL := ""

	for _, img := range images {
		if img.URL == "" {
			continue
		}
		urls = append(urls, img.URL)
		if img.IsPrimary && primaryURL == "" {
			primaryURL = img.URL
		}
	}

	// No explicit primary image: choose one at random.
	if primaryURL == "" && len(urls) > 0 {
		primaryURL = urls[rand.IntN(len(urls))]
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Replace-all semantics: clear the current set, then insert the new one.
		if err := tx.Unscoped().Where("product_id = ?", productID).
			Delete(&ProductImage{}).Error; err != nil {
			return err
		}

		if len(urls) == 0 {
			return nil
		}

		rows := make([]ProductImage, 0, len(urls))
		for _, url := range urls {
			rows = append(rows, ProductImage{
				ProductID: productID,
				URL:       url,
				IsPrimary: url == primaryURL,
			})
		}

		return tx.Create(&rows).Error
	})
}

// DeleteProduct soft-deletes a product by ID, together with its images.
func (r *Repository) DeleteProduct(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&Product{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("product %s: %w", id, ErrNotFound)
		}

		// Images belong to the product, so they are hidden along with it.
		return tx.Where("product_id = ?", id).Delete(&ProductImage{}).Error
	})
}

// ListImagesByRefPrefix returns the non-deleted product images whose stored
// reference starts with prefix, oldest first. Used by the storage migration to
// find images still referencing local files.
func (r *Repository) ListImagesByRefPrefix(prefix string) ([]ProductImage, error) {
	var images []ProductImage
	err := r.db.Where("url LIKE ?", prefix+"%").
		Order("created_at ASC").
		Find(&images).Error
	return images, err
}

// UpdateImageRef rewrites the stored reference of a single product image.
func (r *Repository) UpdateImageRef(id uuid.UUID, ref string) error {
	result := r.db.Model(&ProductImage{}).Where("id = ?", id).Update("url", ref)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product image %s: %w", id, ErrNotFound)
	}
	return nil
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

func (r *Repository) ListAllInventories() ([]Inventory, error) {
	var inventories []Inventory

	err := r.db.Order("created_at DESC").Find(&inventories).Error

	if err != nil {
		return nil, err
	}

	return inventories, err
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

	if quantity == 0 {
		result := r.db.Where("inventory_id = ? AND product_id = ?", inventoryID, productID).
			Delete(&Stock{})
		if result.RowsAffected == 0 {
			return fmt.Errorf("stock entry not found: %w", ErrNotFound)
		}
		return nil
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
		Select(`stock.*,
			products.name as product_name,
			inventories.name as inventory_name,
			(SELECT url FROM product_images
			 WHERE product_images.product_id = stock.product_id
			   AND product_images.deleted_at IS NULL
			 ORDER BY product_images.is_primary DESC, product_images.created_at ASC
			 LIMIT 1) AS product_image_url`).
		Joins("LEFT JOIN products ON products.id = stock.product_id").
		Joins("LEFT JOIN inventories ON inventories.id = stock.inventory_id").
		Where("stock.quantity > 0")

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
		Joins("LEFT JOIN products ON products.id = stock.product_id").
		Where("stock.quantity > 0")
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
		ProductName     string `json:"product_name"`
		ProductImageURL string `json:"product_image_url"`
		InventoryName   string `json:"inventory_name"`
	}

	var rows []stockRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	dtosItems := make([]dtos.StockDto, 0, len(rows))
	for _, row := range rows {
		dtosItems = append(dtosItems, dtos.StockDto{
			ID:              row.ID,
			InventoryID:     row.InventoryID,
			ProductID:       row.ProductID,
			ProductName:     row.ProductName,
			ProductImageURL: row.ProductImageURL,
			InventoryName:   row.InventoryName,
			Quantity:        row.Quantity,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}

	return dtosItems, total, nil
}

// GetDashboardData returns aggregated counts and the low-stock items for the dashboard.
// An entry is low stock when its quantity falls below the threshold configured on
// its product; products with a threshold of 0 are never flagged.
func (r *Repository) GetDashboardData() (*dtos.InventoryDashboardDto, error) {
	dto := &dtos.InventoryDashboardDto{}

	// Total value (sum of quantity * price)
	var totalValue float64
	if err := r.db.Table("stock").
		Select("COALESCE(SUM(stock.quantity * products.price), 0)").
		Joins("JOIN products ON products.id = stock.product_id").
		Where("products.deleted_at IS NULL").
		Scan(&totalValue).Error; err != nil {
		return nil, fmt.Errorf("dashboard total value: %w", err)
	}
	dto.Counts.TotalValue = totalValue

	// Active products (distinct products with stock quantity > 0)
	var activeProducts int64
	if err := r.db.Table("stock").
		Select("DISTINCT product_id").
		Where("quantity > 0").
		Count(&activeProducts).Error; err != nil {
		return nil, fmt.Errorf("dashboard active products: %w", err)
	}
	dto.Counts.ActiveProducts = int(activeProducts)

	// Locations count
	var locationsCount int64
	if err := r.db.Model(&Inventory{}).Count(&locationsCount).Error; err != nil {
		return nil, fmt.Errorf("dashboard locations: %w", err)
	}
	dto.Counts.Locations = int(locationsCount)

	// --- Low stock items (quantity below the product's own threshold) ---

	type lowStockRow struct {
		ID              uuid.UUID
		ProductID       uuid.UUID
		ProductName     string
		ProductImageURL string
		InventoryName   string
		Quantity        int
		Threshold       int
		NoStock         bool
	}

	// 1) Stock entries sitting below their product's threshold.
	entries := func() *gorm.DB {
		return r.db.Table("stock").
			Joins("JOIN products ON products.id = stock.product_id").
			Joins("JOIN inventories ON inventories.id = stock.inventory_id").
			Where("products.low_stock_threshold > 0").
			Where("stock.quantity < products.low_stock_threshold").
			Where("products.deleted_at IS NULL").
			Where("inventories.deleted_at IS NULL")
	}

	// 2) Products with no stock anywhere. A stock row is removed once its quantity
	//    reaches zero, so an out-of-stock product has no row at all and would
	//    otherwise never be reported — even though zero is the lowest stock possible.
	noStock := func() *gorm.DB {
		return r.db.Table("products").
			Where("products.low_stock_threshold > 0").
			Where("products.deleted_at IS NULL").
			Where("NOT EXISTS (SELECT 1 FROM stock WHERE stock.product_id = products.id)")
	}

	var entryCount, noStockCount int64
	if err := entries().Count(&entryCount).Error; err != nil {
		return nil, fmt.Errorf("dashboard low stock count: %w", err)
	}
	if err := noStock().Count(&noStockCount).Error; err != nil {
		return nil, fmt.Errorf("dashboard out of stock count: %w", err)
	}
	dto.Counts.LowStocks = int(entryCount + noStockCount)

	// Each row carries the product's primary image so the dashboard can show a
	// thumbnail. Both lists come back lowest-quantity first, so merging their first
	// 10 rows is enough to determine the overall 10 lowest.
	var entryRows []lowStockRow
	if err := entries().
		Select(`stock.id, stock.product_id, stock.quantity,
			products.name as product_name,
			products.low_stock_threshold as threshold,
			inventories.name as inventory_name,
			(SELECT url FROM product_images
			 WHERE product_images.product_id = stock.product_id
			   AND product_images.deleted_at IS NULL
			 ORDER BY product_images.is_primary DESC, product_images.created_at ASC
			 LIMIT 1) AS product_image_url`).
		Order("stock.quantity ASC").
		Limit(10).
		Find(&entryRows).Error; err != nil {
		return nil, fmt.Errorf("dashboard low stock items: %w", err)
	}

	var noStockRows []lowStockRow
	if err := noStock().
		Select(`products.id as id, products.id as product_id, products.name as product_name,
			products.low_stock_threshold as threshold,
			(SELECT url FROM product_images
			 WHERE product_images.product_id = products.id
			   AND product_images.deleted_at IS NULL
			 ORDER BY product_images.is_primary DESC, product_images.created_at ASC
			 LIMIT 1) AS product_image_url`).
		Order("products.name ASC").
		Limit(10).
		Find(&noStockRows).Error; err != nil {
		return nil, fmt.Errorf("dashboard out of stock items: %w", err)
	}
	for i := range noStockRows {
		noStockRows[i].NoStock = true
	}

	// Out-of-stock products (quantity 0) always sort first.
	rows := append(entryRows, noStockRows...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Quantity < rows[j].Quantity })
	if len(rows) > 10 {
		rows = rows[:10]
	}

	dto.Stocks = make([]dtos.LowStockDto, 0, len(rows))
	for _, row := range rows {
		dto.Stocks = append(dto.Stocks, dtos.LowStockDto{
			ID:              row.ID,
			ProductID:       row.ProductID,
			ProductName:     row.ProductName,
			ProductImageURL: row.ProductImageURL,
			InventoryName:   row.InventoryName,
			Quantity:        row.Quantity,
			Threshold:       row.Threshold,
			NoStock:         row.NoStock,
			Status:          dtos.StockStatus_LOW,
		})
	}

	return dto, nil
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

		if ip.Quantity == 0 {
			return tx.Where("inventory_id = ? AND product_id = ?", inventoryID, productID).
				Delete(&Stock{}).Error
		}

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
// User CRUD
// =============================================================================

// CreateUser inserts a new user.
func (r *Repository) CreateUser(u *User) error {
	return r.db.Create(u).Error
}

// GetUserByID retrieves a user by their UUID.
func (r *Repository) GetUserByID(id uuid.UUID) (*User, error) {
	var u User

	err := r.db.Where("id = ?", id).First(&u).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user %s: %w", id, ErrNotFound)
	}

	return &u, err
}

// GetUserWithRoleByID retrieves a user by UUID, including their role and its permissions.
func (r *Repository) GetUserWithRoleByID(id uuid.UUID) (*User, error) {
	var u User

	err := r.db.Preload("Role.Permissions").Where("id = ?", id).First(&u).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user %s: %w", id, ErrNotFound)
	}

	return &u, err
}

// GetUserByEmail retrieves a user by their email address.
func (r *Repository) GetUserByEmail(email string) (*User, error) {
	var u User

	err := r.db.Where("email = ?", email).First(&u).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user %q: %w", email, ErrNotFound)
	}

	return &u, err
}

// ListUsers returns a paginated, searchable list of users.
func (r *Repository) ListUsers(offset, limit int, search string) ([]dtos.UserDto, int64, error) {
	var users []dtos.UserDto
	var total int64

	query := r.db.Model(&User{})

	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Select("id, name, email, is_active, role_id, created_at, updated_at").
		Find(&users).Error

	return users, total, err
}

// UpdateUser updates the name, email, and optionally password of a user.
func (r *Repository) UpdateUser(u *User) error {
	result := r.db.Model(u).Select("name", "email", "password", "is_active", "role_id").Updates(u)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %s: %w", u.ID, ErrNotFound)
	}
	return result.Error
}

// DeleteUser soft-deletes a user by ID.
func (r *Repository) DeleteUser(id uuid.UUID) error {
	result := r.db.Delete(&User{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// SetUserActiveStatus activates or deactivates a user.
func (r *Repository) SetUserActiveStatus(id uuid.UUID, active bool) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Update("is_active", active)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// =============================================================================
// Role CRUD
// =============================================================================

// CreateRole inserts a new role.
func (r *Repository) CreateRole(role *Role) error {
	return r.db.Create(role).Error
}

// GetRoleByID retrieves a role by UUID, including its permissions.
func (r *Repository) GetRoleByID(id uuid.UUID) (*Role, error) {
	var role Role
	err := r.db.Preload("Permissions").Where("id = ?", id).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("role %s: %w", id, ErrNotFound)
	}
	return &role, err
}

// GetRoleByName retrieves a role by name, including its permissions.
func (r *Repository) GetRoleByName(name string) (*Role, error) {
	var role Role
	err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("role %q: %w", name, ErrNotFound)
	}
	return &role, err
}

// ListRoles returns all roles with their permissions.
func (r *Repository) ListRoles() ([]Role, error) {
	var roles []Role
	err := r.db.Preload("Permissions").Order("name ASC").Find(&roles).Error
	return roles, err
}

// GetUserCountByRoleID returns the number of users assigned to a role.
func (r *Repository) GetUserCountByRoleID(roleID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&User{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

// UpdateRole updates name and description of a role.
func (r *Repository) UpdateRole(role *Role) error {
	result := r.db.Model(role).Select("name", "description").Updates(role)
	if result.RowsAffected == 0 {
		return fmt.Errorf("role %s: %w", role.ID, ErrNotFound)
	}
	return result.Error
}

// DeleteRole soft-deletes a role by ID.
func (r *Repository) DeleteRole(id uuid.UUID) error {
	result := r.db.Delete(&Role{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("role %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// AddPermissionToRole associates a permission with a role.
func (r *Repository) AddPermissionToRole(roleID, permissionID uuid.UUID) error {
	role, err := r.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	perm, err := r.GetPermissionByID(permissionID)
	if err != nil {
		return err
	}
	return r.db.Model(role).Association("Permissions").Append(perm)
}

// RemovePermissionFromRole removes a permission association from a role.
func (r *Repository) RemovePermissionFromRole(roleID, permissionID uuid.UUID) error {
	role, err := r.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	perm, err := r.GetPermissionByID(permissionID)
	if err != nil {
		return err
	}
	return r.db.Model(role).Association("Permissions").Delete(perm)
}

// =============================================================================
// Permission CRUD
// =============================================================================

// CreatePermission inserts a new permission.
func (r *Repository) CreatePermission(p *Permission) error {
	return r.db.Create(p).Error
}

// GetPermissionByID retrieves a permission by UUID.
func (r *Repository) GetPermissionByID(id uuid.UUID) (*Permission, error) {
	var p Permission
	err := r.db.Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("permission %s: %w", id, ErrNotFound)
	}
	return &p, err
}

// GetPermissionByCode retrieves a permission by its code.
func (r *Repository) GetPermissionByCode(code string) (*Permission, error) {
	var p Permission
	err := r.db.Where("code = ?", code).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("permission %q: %w", code, ErrNotFound)
	}
	return &p, err
}

// ListPermissions returns all permissions ordered by code.
func (r *Repository) ListPermissions() ([]Permission, error) {
	var permissions []Permission
	err := r.db.Order("code ASC").Find(&permissions).Error
	return permissions, err
}

// UpdatePermission updates code, name, and description of a permission.
func (r *Repository) UpdatePermission(p *Permission) error {
	result := r.db.Model(p).Select("code", "name", "description").Updates(p)
	if result.RowsAffected == 0 {
		return fmt.Errorf("permission %s: %w", p.ID, ErrNotFound)
	}
	return result.Error
}

// DeletePermission deletes a permission by ID.
func (r *Repository) DeletePermission(id uuid.UUID) error {
	result := r.db.Delete(&Permission{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("permission %s: %w", id, ErrNotFound)
	}
	return result.Error
}

// =============================================================================
// Errors
// =============================================================================

var ErrNotFound = errors.New("record not found")

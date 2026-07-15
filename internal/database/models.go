package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product represents a product that can be stocked in inventories.
type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"not null;default:0" json:"price"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Many-to-many: which inventories carry this product and in what quantity
	Inventories []Stock `gorm:"foreignKey:ProductID" json:"inventories,omitempty"`
}

// BeforeCreate hook ensures UUID is set if empty.
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// Inventory represents a physical storage location (warehouse, store, etc.).
type Inventory struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Address   string         `gorm:"type:text" json:"address"`
	Latitude  string         `gorm:"type:varchar(50)" json:"latitude"`
	Longitude string         `gorm:"type:varchar(50)" json:"longitude"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Many-to-many: which products are stocked here and in what quantity
	Products []Stock `gorm:"foreignKey:InventoryID" json:"products,omitempty"`
}

// BeforeCreate hook ensures UUID is set if empty.
func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// Stock represents the quantity of a product at an inventory location.
// It is the join table linking Inventory and Product.
type Stock struct {
	InventoryID uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex:idx_inv_prod" json:"inventory_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex:idx_inv_prod" json:"product_id"`
	Quantity    int       `gorm:"not null;default:0" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Belongs-to relationships
	Inventory Inventory `gorm:"foreignKey:InventoryID;constraint:OnDelete:CASCADE" json:"-"`
	Product   Product   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"-"`
}

func (Stock) TableName() string {
	return "stock"
}

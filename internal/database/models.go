package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyStatus is the lifecycle state of a company (tenant). New sign-ups start
// as "pending" and only become usable once a platform admin approves them.
type CompanyStatus string

const (
	CompanyStatusPending   CompanyStatus = "pending"
	CompanyStatusApproved  CompanyStatus = "approved"
	CompanyStatusRejected  CompanyStatus = "rejected"
	CompanyStatusSuspended CompanyStatus = "suspended"
)

// Company is a tenant. Every tenant-owned record (users, roles, inventories,
// products and stock) belongs to exactly one company, and all access is scoped
// to the caller's company.
type Company struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug       string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"slug"`
	Phone      string         `gorm:"type:varchar(32)" json:"phone"`
	Status     CompanyStatus  `gorm:"type:varchar(20);index" json:"status"`
	ApprovedAt *time.Time     `json:"approved_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// IsApproved reports whether the company may use the application.
func (c *Company) IsApproved() bool {
	return c.Status == CompanyStatusApproved
}

// BeforeCreate hook ensures UUID is set if empty.
func (c *Company) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Product represents a product that can be stocked in inventories.
type Product struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID   uuid.UUID `gorm:"type:uuid;index" json:"company_id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_products_company_name" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"not null;default:0" json:"price"`
	// LowStockThreshold marks the product as low stock whenever its quantity at an
	// inventory drops below this value. Zero disables low-stock monitoring.
	LowStockThreshold int            `gorm:"not null;default:50" json:"low_stock_threshold"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Images attached to this product. Exactly one of them is the primary image
	// whenever the product has at least one image.
	Images []ProductImage `gorm:"foreignKey:ProductID" json:"images,omitempty"`

	// Many-to-many: which inventories carry this product and in what quantity
	Inventories []Stock `gorm:"foreignKey:ProductID" json:"inventories,omitempty"`

	// ImageURL is the URL of the primary image. It is computed from the Images
	// relation when the product is loaded and is never persisted on its own.
	ImageURL string `gorm:"-" json:"image_url"`
}

// PrimaryImageURL returns the URL of the primary image, falling back to the
// first available image, and "" when the product has no images at all.
func (p *Product) PrimaryImageURL() string {
	for i := range p.Images {
		if p.Images[i].IsPrimary {
			return p.Images[i].URL
		}
	}
	if len(p.Images) > 0 {
		return p.Images[0].URL
	}
	return ""
}

// BeforeCreate hook ensures UUID is set if empty.
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ProductImage is one image belonging to a product. A product can have many
// images; at most one of them is flagged as the primary image.
type ProductImage struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	URL       string         `gorm:"type:text;not null" json:"url"`
	IsPrimary bool           `gorm:"not null;default:false" json:"is_primary"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Product Product `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"-"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

// BeforeCreate hook ensures UUID is set if empty.
func (pi *ProductImage) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return nil
}

// Inventory represents a physical storage location (warehouse, store, etc.).
type Inventory struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID uuid.UUID      `gorm:"type:uuid;index" json:"company_id"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_inventories_company_name" json:"name"`
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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID   uuid.UUID `gorm:"type:uuid;index" json:"company_id"`
	InventoryID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_inv_prod" json:"inventory_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_inv_prod;index" json:"product_id"`
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

// BeforeCreate hook ensures UUID is set if empty.
func (s *Stock) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CompanyID uuid.UUID `json:"company_id" gorm:"type:uuid;index"`
	Name      string    `json:"name" gorm:"size:50;not null" validate:"required,min=2,max=50"`
	Email     string    `json:"email" gorm:"uniqueIndex;size:255;not null" validate:"required,email"`
	Phone     string    `json:"phone" gorm:"size:32"`
	Password  string    `json:"-" gorm:"size:255;not null" validate:"required,min=8"`
	// IsRoot marks the company owner (the account created with the company at
	// sign-up). Root users can edit company data regardless of their role.
	IsRoot    bool           `json:"is_root" gorm:"not null;default:false"`
	RoleID    *uuid.UUID     `json:"role_id" gorm:"type:uuid" validate:"omitempty"`
	Role      *Role          `json:"role,omitempty" gorm:"foreignKey:RoleID;constraint:OnDelete:SET NULL"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (s *User) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// Role represents a named group of permissions. Roles are per-company; the
// permission catalog they reference is global.
type Role struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CompanyID   uuid.UUID      `json:"company_id" gorm:"type:uuid;index"`
	Name        string         `json:"name" gorm:"size:100;not null;uniqueIndex:idx_roles_company_name"`
	Description string         `json:"description" gorm:"type:text"`
	Permissions []Permission   `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// Permission represents an individual permission (e.g. "products.create").
type Permission struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Code        string    `json:"code" gorm:"size:100;not null;uniqueIndex"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PlatformUser is an operator of the platform itself — the account that reviews
// and approves companies. Platform users are deliberately kept in their own
// table (and their own auth flow) so they can be extracted into a standalone
// platform service with its own database later. They never belong to a company.
type PlatformUser struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `json:"name" gorm:"size:50;not null"`
	Email     string         `json:"email" gorm:"uniqueIndex;size:255;not null"`
	Phone     string         `json:"phone" gorm:"size:32"`
	Password  string         `json:"-" gorm:"size:255;not null"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (p *PlatformUser) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

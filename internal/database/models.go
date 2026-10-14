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
// to the caller's company. comment for testing deploy
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

// CompanyProfileStatus is the review state of a staged profile change.
type CompanyProfileStatus string

const (
	CompanyProfilePending  CompanyProfileStatus = "pending"
	CompanyProfileApproved CompanyProfileStatus = "approved"
	CompanyProfileRejected CompanyProfileStatus = "rejected"
)

// CompanyProfile is the live, approved business profile of a company. It is kept
// separate from Company (identity + lifecycle status) because profile edits are
// user-driven and must pass through platform review before they take effect.
// There is at most one profile per company.
type CompanyProfile struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"company_id"`
	LegalName   string    `gorm:"type:varchar(255)" json:"legal_name"`
	Email       string    `gorm:"type:varchar(255)" json:"email"`
	Phone       string    `gorm:"type:varchar(32)" json:"phone"`
	Website     string    `gorm:"type:varchar(255)" json:"website"`
	Address     string    `gorm:"type:text" json:"address"`
	City        string    `gorm:"type:varchar(120)" json:"city"`
	Country     string    `gorm:"type:varchar(120)" json:"country"`
	TaxID       string    `gorm:"type:varchar(64)" json:"tax_id"`
	LogoURL     string    `gorm:"type:text" json:"logo_url"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CompanyProfile) TableName() string { return "company_profiles" }

// BeforeCreate hook ensures UUID is set if empty.
func (cp *CompanyProfile) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	return nil
}

// CompanyProfileRequest is a staged profile edit awaiting platform review. Only
// one request per company may be pending at a time; reviewed requests are kept as
// history. Payload holds the complete proposed profile as JSON so a reviewer can
// diff it against the live profile.
type CompanyProfileRequest struct {
	ID          uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID   uuid.UUID            `gorm:"type:uuid;not null;index" json:"company_id"`
	RequestedBy uuid.UUID            `gorm:"type:uuid;index" json:"requested_by"`
	Payload     string               `gorm:"type:text;not null" json:"payload"`
	Status      CompanyProfileStatus `gorm:"type:varchar(20);not null;default:pending;index" json:"status"`
	Reason      string               `gorm:"type:text" json:"reason"`
	ReviewedBy  *uuid.UUID           `gorm:"type:uuid" json:"reviewed_by"`
	ReviewedAt  *time.Time           `json:"reviewed_at"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

func (CompanyProfileRequest) TableName() string { return "company_profile_requests" }

// BeforeCreate hook ensures UUID is set if empty.
func (cpr *CompanyProfileRequest) BeforeCreate(tx *gorm.DB) error {
	if cpr.ID == uuid.Nil {
		cpr.ID = uuid.New()
	}
	return nil
}

// Product represents a product that can be stocked in inventories.
type Product struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// CompanyID and Name share the index name on purpose: GORM builds a composite
	// unique index from every field carrying the same uniqueIndex:<name> tag.
	CompanyID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_products_company_name" json:"company_id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_products_company_name" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"not null;default:0" json:"price"`
	// SKU is the stock keeping unit — a short human-readable identifier for the
	// product (e.g. "MOU-001"). Uniqueness is scoped to the company, not global:
	// two tenants may legitimately use the same SKU for different products.
	// Enforced by the partial unique index idx_products_company_sku (see Migrate).
	SKU string `gorm:"type:varchar(64);index" json:"sku"`
	// Barcode is an optional EAN/UPC code. When present it is unique within the
	// company, enforced by the partial unique index idx_products_company_barcode.
	Barcode string `gorm:"type:varchar(64);index" json:"barcode"`
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
	CompanyID uuid.UUID      `gorm:"type:uuid;uniqueIndex:idx_inventories_company_name" json:"company_id"`
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
	CompanyID   uuid.UUID      `json:"company_id" gorm:"type:uuid;uniqueIndex:idx_roles_company_name"`
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

// Audit actor types recorded on every audit entry.
const (
	AuditActorUser     = "user"     // a company user acting within their tenant
	AuditActorPlatform = "platform" // a platform operator
	AuditActorSystem   = "system"   // automated / background action
)

// AuditLog is an immutable record of a mutating action: who did what, to which
// entity, when, and (when known) the before/after values. CompanyID is nullable
// so platform-level actions not tied to a single tenant can be recorded too.
// Values are stored as JSON text (never queried inside), so a plain text column
// is used instead of jsonb.
type AuditLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID  *uuid.UUID `gorm:"type:uuid;index" json:"company_id,omitempty"`
	ActorType  string     `gorm:"type:varchar(20);not null" json:"actor_type"`
	ActorID    *uuid.UUID `gorm:"type:uuid" json:"actor_id,omitempty"`
	ActorName  string     `gorm:"type:varchar(120)" json:"actor_name"`
	Action     string     `gorm:"type:varchar(80);not null;index" json:"action"`
	EntityType string     `gorm:"type:varchar(60);index" json:"entity_type"`
	EntityID   string     `gorm:"type:varchar(64)" json:"entity_id"`
	OldValues  string     `gorm:"type:text" json:"old_values,omitempty"`
	NewValues  string     `gorm:"type:text" json:"new_values,omitempty"`
	Metadata   string     `gorm:"type:text" json:"metadata,omitempty"`
	IP         string     `gorm:"type:varchar(64)" json:"ip,omitempty"`
	UserAgent  string     `gorm:"type:varchar(255)" json:"user_agent,omitempty"`
	RequestID  string     `gorm:"type:varchar(64);index" json:"request_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// BeforeCreate hook ensures UUID is set if empty.
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

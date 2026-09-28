package dtos

import (
	"time"

	"github.com/google/uuid"
)

type ProductResponse struct {
	ID                uuid.UUID         `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	ImageURL          string            `json:"image_url"`
	Images            []ProductImageDto `json:"images"`
	Price             float64           `json:"price"`
	LowStockThreshold int               `json:"low_stock_threshold"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// ProductImageDto is a single image belonging to a product.
type ProductImageDto struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductImageInput is one image supplied when creating or updating a product.
// When no image in the list has IsPrimary set, the server picks one at random.
type ProductImageInput struct {
	URL       string `json:"url"`
	IsPrimary bool   `json:"is_primary"`
}

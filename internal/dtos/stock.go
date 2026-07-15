package dtos

import (
	"time"

	"github.com/google/uuid"
)

type ListStockFilter struct {
	InventoryID *uuid.UUID
	ProductID   *uuid.UUID
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Search      string // searches product_name
	OrderBy     string // "created_at", "updated_at", "quantity", "product_name", "inventory_name"
	Sort        string // "asc" or "desc"
	Offset      int
	Limit       int
}

type StockDto struct {
	InventoryID   uuid.UUID `json:"inventory_id"`
	ProductID     uuid.UUID `json:"product_id"`
	ProductName   string    `json:"product_name"`
	InventoryName string    `json:"inventory_name"`
	Quantity      int       `json:"quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

package dtos

import "github.com/google/uuid"

type InventoryDashboardDto struct {
	Counts      InventoryCountsDashbaord      `json:"counts"`
	Inventories []InventoryLocationsDashboard `json:"inventories"`
	Stocks      []LowStockDto                 `json:"stocks"`
}

type InventoryCountsDashbaord struct {
	TotalValue     float64 `json:"total_value"`
	ActiveProducts int     `json:"active_products"`
	Locations      int     `json:"locations_count"`
	LowStocks      int     `json:"low_stock"`
}

type InventoryLocationsDashboard struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Latitude  string    `json:"lat"`
	Longitude string    `json:"long"`
}

type LowStockDto struct {
	ID              uuid.UUID `json:"id"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	ProductImageURL string    `json:"product_image_url"`
	InventoryName   string    `json:"inventory_name"`
	Quantity        int       `json:"quantity"`
	Threshold       int       `json:"threshold"`
	// NoStock is true when the product has no stock entries at all. Those rows have
	// no inventory and their ID is the product ID.
	NoStock bool        `json:"no_stock"`
	Status  StockStatus `json:"stats"`
}

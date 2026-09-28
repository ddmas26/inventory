package dtos

// StockStatus classifies how healthy a stock entry is.
//
// Only StockStatus_LOW is currently produced: an entry is considered low when its
// quantity falls below the low stock threshold configured on its product, and only
// low entries are returned by the dashboard.
type StockStatus int

const (
	StockStatus_LOW StockStatus = iota
)

package dtos

type StockStatus int

const (
	StockStatus_LOW StockStatus = iota
	StockStatus_NORMAL
	StockStatus_OK
)

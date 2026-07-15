package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// StockService handles business logic for stock operations.
type StockService struct {
	repo *database.Repository
}

func NewStockService(repo *database.Repository) *StockService {
	return &StockService{repo: repo}
}

// AddStock adds quantity to a product at an inventory. Creates entry if missing.
func (s *StockService) AddStock(inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.AddStock(inventoryID, productID, amount)
}

// DeductStock subtracts quantity. Fails if stock would go negative or entry missing.
func (s *StockService) DeductStock(inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.DeductStock(inventoryID, productID, amount)
}

// SetStock sets absolute quantity (overwrites). Fails if entry missing.
func (s *StockService) SetStock(inventoryID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return s.repo.SetProductStock(inventoryID, productID, quantity)
}

// TransferStock moves quantity from one inventory to another atomically.
func (s *StockService) TransferStock(fromInv, toInv, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}
	if fromInv == toInv {
		return errors.New("source and destination inventories must be different")
	}
	return s.repo.TransferStock(fromInv, toInv, productID, amount)
}

// GetStock retrieves a single stock entry.
func (s *StockService) GetStock(inventoryID, productID uuid.UUID) (*database.Stock, error) {
	stock, err := s.repo.GetStock(inventoryID, productID)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}
	return stock, nil
}

// ListStock returns filtered, paginated stock entries.
func (s *StockService) ListStock(filter dtos.ListStockFilter) ([]dtos.StockDto, int64, error) {
	return s.repo.ListStock(filter)
}

// RemoveStock deletes a stock entry.
func (s *StockService) RemoveStock(inventoryID, productID uuid.UUID) error {
	return s.repo.RemoveProductFromInventory(inventoryID, productID)
}

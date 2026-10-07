package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/ddmas26/inventory/internal/storage"
	"github.com/google/uuid"
)

// StockService handles business logic for stock operations.
type StockService struct {
	repo    *database.Repository
	storage *storage.Store
}

func NewStockService(repo *database.Repository, store *storage.Store) *StockService {
	return &StockService{repo: repo, storage: store}
}

// AddStock adds quantity to a product at an inventory. Creates entry if missing.
func (s *StockService) AddStock(companyID, inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("quantity must be positive")
	}
	if err := s.validateRefs(companyID, inventoryID, productID); err != nil {
		return err
	}
	return s.repo.AddStock(companyID, inventoryID, productID, amount)
}

// DeductStock subtracts quantity. Fails if stock would go negative or entry missing.
func (s *StockService) DeductStock(companyID, inventoryID, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.DeductStock(companyID, inventoryID, productID, amount)
}

// SetStock sets absolute quantity (overwrites). Fails if entry missing.
func (s *StockService) SetStock(companyID, inventoryID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	if err := s.validateRefs(companyID, inventoryID, productID); err != nil {
		return err
	}
	return s.repo.SetProductStock(companyID, inventoryID, productID, quantity)
}

// TransferStock moves quantity from one inventory to another atomically.
func (s *StockService) TransferStock(companyID, fromInv, toInv, productID uuid.UUID, amount int) error {
	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}
	if fromInv == toInv {
		return errors.New("source and destination inventories must be different")
	}
	if err := s.validateRefs(companyID, fromInv, productID); err != nil {
		return err
	}
	if err := s.validateRefs(companyID, toInv, productID); err != nil {
		return err
	}
	return s.repo.TransferStock(companyID, fromInv, toInv, productID, amount)
}

// validateRefs makes sure both the inventory and the product exist inside the
// caller's company, so a stock entry can never link another tenant's records.
func (s *StockService) validateRefs(companyID, inventoryID, productID uuid.UUID) error {
	if _, err := s.repo.GetInventoryByID(companyID, inventoryID); err != nil {
		return errors.New("inventory not found")
	}
	if _, err := s.repo.GetProductByID(companyID, productID); err != nil {
		return errors.New("product not found")
	}
	return nil
}

// GetStock retrieves a single stock entry.
func (s *StockService) GetStock(companyID, inventoryID, productID uuid.UUID) (*database.Stock, error) {
	stock, err := s.repo.GetStock(companyID, inventoryID, productID)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}
	return stock, nil
}

// ListStock returns filtered, paginated stock entries.
func (s *StockService) ListStock(filter dtos.ListStockFilter) ([]dtos.StockDto, int64, error) {
	items, total, err := s.repo.ListStock(filter)
	if err != nil {
		return nil, 0, err
	}

	for i := range items {
		items[i].ProductImageURL = s.storage.URL(items[i].ProductImageURL)
	}

	return items, total, nil
}

// RemoveStock deletes a stock entry.
func (s *StockService) RemoveStock(companyID, inventoryID, productID uuid.UUID) error {
	return s.repo.RemoveProductFromInventory(companyID, inventoryID, productID)
}

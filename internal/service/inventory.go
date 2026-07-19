package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type InventoryService struct {
	repo *database.Repository
}

func NewInventoryService(repo *database.Repository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) CreateInventory(name, address, latitude, longitude string) (*database.Inventory, error) {
	if name == "" {
		return nil, errors.New("inventory name is required")
	}

	inv := &database.Inventory{
		Name:      name,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
	}

	if err := s.repo.CreateInventory(inv); err != nil {
		return nil, fmt.Errorf("create inventory: %w", err)
	}
	return inv, nil
}

func (s *InventoryService) GetByID(id uuid.UUID) (*database.Inventory, error) {
	inv, err := s.repo.GetInventoryByID(id)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	return inv, nil
}

func (s *InventoryService) List(offset, limit int) ([]database.Inventory, int64, error) {
	return s.repo.ListInventories(offset, limit)
}

func (s *InventoryService) Update(inv *database.Inventory) error {
	if inv.Name == "" {
		return errors.New("inventory name is required")
	}
	return s.repo.UpdateInventory(inv)
}

func (s *InventoryService) Delete(id uuid.UUID) error {
	return s.repo.DeleteInventory(id)
}

func (s *InventoryService) ListAll() ([]database.Inventory, error) {
	return s.repo.ListAllInventories()
}

// GetDashboardData returns aggregated dashboard data including counts, low-stock items, and all inventory locations.
func (s *InventoryService) GetDashboardData() (*dtos.InventoryDashboardDto, error) {
	inventories, err := s.repo.ListAllInventories()
	if err != nil {
		return nil, fmt.Errorf("list inventories: %w", err)
	}

	dashData, err := s.repo.GetDashboardData()
	if err != nil {
		return nil, fmt.Errorf("get dashboard data: %w", err)
	}

	dashData.Inventories = make([]dtos.InventoryLocationsDashboard, 0, len(inventories))
	for _, inv := range inventories {
		dashData.Inventories = append(dashData.Inventories, dtos.InventoryLocationsDashboard{
			ID:        inv.ID,
			Name:      inv.Name,
			Latitude:  inv.Latitude,
			Longitude: inv.Longitude,
		})
	}

	return dashData, nil
}

package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type ProductService struct {
	repo *database.Repository
}

func NewProductService(repo *database.Repository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) AddProduct(name, description string, price float64) (*dtos.ProductResponse, error) {
	if name == "" {
		return nil, errors.New("product name is required")
	}
	if len(description) > 2000 {
		return nil, errors.New("product description must be 2000 characters or less")
	}
	if price <= 0 {
		return nil, errors.New("product price must be greater than 0")
	}

	product := database.Product{
		Name:        name,
		Description: description,
		Price:       price,
	}

	if err := s.repo.CreateProduct(&product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	return &dtos.ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}, nil
}

func (s *ProductService) GetByID(id uuid.UUID) (*database.Product, error) {
	product, err := s.repo.GetProductByID(id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return product, nil
}

func (s *ProductService) List(offset, limit int) ([]database.Product, int64, error) {
	return s.repo.ListProducts(offset, limit)
}

func (s *ProductService) Update(product *database.Product) error {
	if product.Name == "" {
		return errors.New("product name is required")
	}
	return s.repo.UpdateProduct(product)
}

func (s *ProductService) Delete(id uuid.UUID) error {
	return s.repo.DeleteProduct(id)
}

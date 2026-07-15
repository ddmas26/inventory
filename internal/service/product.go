package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
)

type ProductService struct {
	repo *database.Repository
}

func NewProductService(repo *database.Repository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) AddProduct(req dtos.CreateProductRequest) (*dtos.ProductResponse, error) {
	if req.Name == "" {
		return nil, errors.New("product name is required")
	}
	if len(req.Description) > 2000 {
		return nil, errors.New("product description must be 2000 characters or less")
	}
	if req.Price <= 0 {
		return nil, errors.New("product price must be greater than 0")
	}

	product := database.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
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

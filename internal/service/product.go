package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/ddmas26/inventory/internal/storage"
	"github.com/google/uuid"
)

// maxProductImages caps how many images a single product can carry.
const maxProductImages = 20

type ProductService struct {
	repo    *database.Repository
	storage *storage.Store
}

func NewProductService(repo *database.Repository, store *storage.Store) *ProductService {
	return &ProductService{repo: repo, storage: store}
}

// normalizeImages trims, de-duplicates and validates an image list, making sure
// at most one image is flagged as primary. Image references are canonicalised to
// object keys so the rows survive a change of storage endpoint.
func (s *ProductService) normalizeImages(images []dtos.ProductImageInput) ([]dtos.ProductImageInput, error) {
	out := make([]dtos.ProductImageInput, 0, len(images))
	seen := make(map[string]bool, len(images))
	primarySet := false

	for _, img := range images {
		ref := s.storage.Ref(img.URL)
		if ref == "" {
			continue
		}
		if len(ref) > 500 {
			return nil, errors.New("image url must be 500 characters or less")
		}
		if seen[ref] {
			continue
		}
		seen[ref] = true

		isPrimary := img.IsPrimary && !primarySet
		if isPrimary {
			primarySet = true
		}

		out = append(out, dtos.ProductImageInput{URL: ref, IsPrimary: isPrimary})
	}

	if len(out) > maxProductImages {
		return nil, fmt.Errorf("a product can have at most %d images", maxProductImages)
	}

	return out, nil
}

// resolveImageRefs turns the stored image references into browser-loadable URLs.
// Idempotent: keys are prefixed, while URLs and site-relative paths pass through.
func (s *ProductService) resolveImageRefs(p *database.Product) {
	for i := range p.Images {
		p.Images[i].URL = s.storage.URL(p.Images[i].URL)
	}
	p.ImageURL = s.storage.URL(p.PrimaryImageURL())
}

// toProductResponse maps a product (with images preloaded) onto its DTO.
func (s *ProductService) toProductResponse(p *database.Product) *dtos.ProductResponse {
	s.resolveImageRefs(p)

	resp := &dtos.ProductResponse{
		ID:                p.ID,
		CompanyID:         p.CompanyID,
		Name:              p.Name,
		Description:       p.Description,
		ImageURL:          p.ImageURL,
		Price:             p.Price,
		LowStockThreshold: p.LowStockThreshold,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		Images:            make([]dtos.ProductImageDto, 0, len(p.Images)),
	}

	for i := range p.Images {
		resp.Images = append(resp.Images, dtos.ProductImageDto{
			ID:        p.Images[i].ID,
			URL:       p.Images[i].URL,
			IsPrimary: p.Images[i].IsPrimary,
			CreatedAt: p.Images[i].CreatedAt,
			UpdatedAt: p.Images[i].UpdatedAt,
		})
	}

	return resp
}

func (s *ProductService) AddProduct(companyID uuid.UUID, name, description string, images []dtos.ProductImageInput, price float64, lowStockThreshold int) (*dtos.ProductResponse, error) {
	if name == "" {
		return nil, errors.New("product name is required")
	}
	if len(description) > 2000 {
		return nil, errors.New("product description must be 2000 characters or less")
	}
	if price <= 0 {
		return nil, errors.New("product price must be greater than 0")
	}
	if lowStockThreshold < 0 {
		return nil, errors.New("low stock threshold cannot be negative")
	}

	imgs, err := s.normalizeImages(images)
	if err != nil {
		return nil, err
	}

	product := database.Product{
		CompanyID:         companyID,
		Name:              name,
		Description:       description,
		Price:             price,
		LowStockThreshold: lowStockThreshold,
	}

	if err := s.repo.CreateProduct(&product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	if err := s.repo.ReplaceProductImages(product.ID, imgs); err != nil {
		return nil, fmt.Errorf("save product images: %w", err)
	}

	created, err := s.repo.GetProductByID(companyID, product.ID)
	if err != nil {
		return nil, fmt.Errorf("load created product: %w", err)
	}

	return s.toProductResponse(created), nil
}

func (s *ProductService) GetByID(companyID, id uuid.UUID) (*database.Product, error) {
	product, err := s.repo.GetProductByID(companyID, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	s.resolveImageRefs(product)
	return product, nil
}

func (s *ProductService) List(companyID uuid.UUID, offset, limit int) ([]database.Product, int64, error) {
	products, total, err := s.repo.ListProducts(companyID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	for i := range products {
		s.resolveImageRefs(&products[i])
	}

	return products, total, nil
}

// Update replaces the product's mutable fields and its full set of images.
func (s *ProductService) Update(companyID uuid.UUID, product *database.Product, images []dtos.ProductImageInput) error {
	if product.Name == "" {
		return errors.New("product name is required")
	}
	if product.LowStockThreshold < 0 {
		return errors.New("low stock threshold cannot be negative")
	}

	imgs, err := s.normalizeImages(images)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProduct(companyID, product); err != nil {
		return err
	}

	return s.repo.ReplaceProductImages(product.ID, imgs)
}

func (s *ProductService) Delete(companyID, id uuid.UUID) error {
	return s.repo.DeleteProduct(companyID, id)
}

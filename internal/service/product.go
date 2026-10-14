package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/ddmas26/inventory/internal/storage"
	"github.com/google/uuid"
)

// maxProductImages caps how many images a single product can carry.
const maxProductImages = 20

// Identifier limits. SKU is mandatory, barcode is optional.
const (
	maxSKULength     = 64
	maxBarcodeLength = 64
)

// normalizeSKU trims and upper-cases a SKU so uniqueness is case-insensitive
// ("mou-001" and "MOU-001" are treated as the same identifier).
func normalizeSKU(sku string) string {
	return strings.ToUpper(strings.TrimSpace(sku))
}

// normalizeBarcode trims a barcode and drops the spaces some scanners inject.
func normalizeBarcode(barcode string) string {
	return strings.ReplaceAll(strings.TrimSpace(barcode), " ", "")
}

// isUniqueViolation reports whether an error is PostgreSQL's unique-constraint
// violation (SQLSTATE 23505). It backs up the pre-insert conflict check so that
// a race between two requests still produces a friendly message instead of a
// raw driver error.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

type ProductService struct {
	repo    *database.Repository
	storage *storage.Store
}

func NewProductService(repo *database.Repository, store *storage.Store) *ProductService {
	return &ProductService{repo: repo, storage: store}
}

// WithRepo returns a copy of the service bound to another repository — used to
// run an operation on a transaction-scoped repository.
func (s *ProductService) WithRepo(repo *database.Repository) *ProductService {
	return &ProductService{repo: repo, storage: s.storage}
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
		SKU:               p.SKU,
		Barcode:           p.Barcode,
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

// checkIdentifierConflict fails when the SKU or barcode is already used by a
// different product inside the same company.
func (s *ProductService) checkIdentifierConflict(companyID uuid.UUID, sku, barcode string, excludeID uuid.UUID) error {
	conflict, err := s.repo.ProductIdentifierConflict(companyID, sku, barcode, excludeID)
	if err != nil {
		return err
	}
	switch conflict {
	case "sku":
		return fmt.Errorf("sku %q is already used by another product", sku)
	case "barcode":
		return fmt.Errorf("barcode %q is already used by another product", barcode)
	}
	return nil
}

func (s *ProductService) AddProduct(companyID uuid.UUID, name, description, sku, barcode string, images []dtos.ProductImageInput, price float64, lowStockThreshold int) (*dtos.ProductResponse, error) {
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

	sku = normalizeSKU(sku)
	barcode = normalizeBarcode(barcode)
	if sku == "" {
		return nil, errors.New("product sku is required")
	}
	if len(sku) > maxSKULength {
		return nil, fmt.Errorf("product sku must be %d characters or less", maxSKULength)
	}
	if len(barcode) > maxBarcodeLength {
		return nil, fmt.Errorf("product barcode must be %d characters or less", maxBarcodeLength)
	}
	if err := s.checkIdentifierConflict(companyID, sku, barcode, uuid.Nil); err != nil {
		return nil, err
	}

	imgs, err := s.normalizeImages(images)
	if err != nil {
		return nil, err
	}

	product := database.Product{
		CompanyID:         companyID,
		Name:              name,
		SKU:               sku,
		Barcode:           barcode,
		Description:       description,
		Price:             price,
		LowStockThreshold: lowStockThreshold,
	}

	if err := s.repo.CreateProduct(&product); err != nil {
		if isUniqueViolation(err) {
			return nil, errors.New("a product with the same name, sku or barcode already exists in this company")
		}
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

func (s *ProductService) List(companyID uuid.UUID, offset, limit int, search string) ([]database.Product, int64, error) {
	products, total, err := s.repo.ListProducts(companyID, offset, limit, strings.TrimSpace(search))
	if err != nil {
		return nil, 0, err
	}

	for i := range products {
		s.resolveImageRefs(&products[i])
	}

	return products, total, nil
}

// GetByCode looks a product up by SKU first, then by barcode. It backs the
// barcode-scanner endpoint: a scanner emits whatever the label encodes, which
// in practice is either our own SKU or the manufacturer's barcode.
func (s *ProductService) GetByCode(companyID uuid.UUID, code string) (*database.Product, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("code is required")
	}

	product, err := s.repo.GetProductBySKU(companyID, normalizeSKU(code))
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("lookup product by sku: %w", err)
		}

		product, err = s.repo.GetProductByBarcode(companyID, normalizeBarcode(code))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, fmt.Errorf("product %q: %w", code, database.ErrNotFound)
			}
			return nil, fmt.Errorf("lookup product by barcode: %w", err)
		}
	}

	s.resolveImageRefs(product)
	return product, nil
}

// Update replaces the product's mutable fields and its full set of images.
func (s *ProductService) Update(companyID uuid.UUID, product *database.Product, images []dtos.ProductImageInput) error {
	if product.Name == "" {
		return errors.New("product name is required")
	}
	if product.LowStockThreshold < 0 {
		return errors.New("low stock threshold cannot be negative")
	}

	product.SKU = normalizeSKU(product.SKU)
	product.Barcode = normalizeBarcode(product.Barcode)

	// A SKU is mandatory, but a client that predates SKUs may omit it: keep the
	// identifier already stored on the product rather than clearing it.
	if product.SKU == "" {
		existing, err := s.repo.GetProductByID(companyID, product.ID)
		if err != nil {
			return err
		}
		product.SKU = existing.SKU
	}
	if len(product.SKU) > maxSKULength {
		return fmt.Errorf("product sku must be %d characters or less", maxSKULength)
	}
	if len(product.Barcode) > maxBarcodeLength {
		return fmt.Errorf("product barcode must be %d characters or less", maxBarcodeLength)
	}
	if err := s.checkIdentifierConflict(companyID, product.SKU, product.Barcode, product.ID); err != nil {
		return err
	}

	imgs, err := s.normalizeImages(images)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProduct(companyID, product); err != nil {
		if isUniqueViolation(err) {
			return errors.New("a product with the same name, sku or barcode already exists in this company")
		}
		return err
	}

	return s.repo.ReplaceProductImages(product.ID, imgs)
}

func (s *ProductService) Delete(companyID, id uuid.UUID) error {
	return s.repo.DeleteProduct(companyID, id)
}

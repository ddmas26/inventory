package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type createProductRequest struct {
	Name              string                   `json:"name"`
	SKU               string                   `json:"sku"`
	Barcode           string                   `json:"barcode"`
	Description       string                   `json:"description"`
	Price             float64                  `json:"price"`
	LowStockThreshold int                      `json:"low_stock_threshold"`
	Images            []dtos.ProductImageInput `json:"images"`
}

type updateProductRequest struct {
	Name              string                   `json:"name"`
	SKU               string                   `json:"sku"`
	Barcode           string                   `json:"barcode"`
	Description       string                   `json:"description"`
	Price             float64                  `json:"price"`
	LowStockThreshold int                      `json:"low_stock_threshold"`
	Images            []dtos.ProductImageInput `json:"images"`
}

// CreateProduct handles POST /api/products
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.create")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	var req createProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var created *dtos.ProductResponse
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType = "product.create", "product"

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		p, err := h.ProductSvc.WithRepo(tx).AddProduct(cid, req.Name, req.Description, req.SKU, req.Barcode, req.Images, req.Price, req.LowStockThreshold)
		if err != nil {
			return err
		}
		created = p
		entry.EntityID, entry.NewValues = p.ID.String(), p
		return nil
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, created)
}

// GetProduct handles GET /api/products/{id}
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.ProductSvc.GetByID(cid, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get product")
		return
	}

	respond(w, http.StatusOK, product)
}

// ScanProduct handles GET /api/products/scan?code={sku-or-barcode}
// It resolves a scanned code (SKU or barcode) to a single product so the
// frontend can act on whatever a barcode scanner emits.
func (h *Handler) ScanProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	product, err := h.ProductSvc.GetByCode(cid, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "no product matches that sku or barcode")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, product)
}

// ListProducts handles GET /api/products
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	pageIndex, _ := strconv.Atoi(r.URL.Query().Get("page_index"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageIndex <= 1 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (pageIndex - 1) * pageSize
	limit := pageSize

	products, total, err := h.ProductSvc.List(cid, offset, limit, r.URL.Query().Get("search"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	if products == nil {
		products = []database.Product{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       products,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// UpdateProduct handles PUT /api/products/{id}
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.edit")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req updateProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Snapshot the current values so the audit entry carries a before/after.
	product := &database.Product{
		ID:                id,
		Name:              req.Name,
		SKU:               req.SKU,
		Barcode:           req.Barcode,
		Description:       req.Description,
		Price:             req.Price,
		LowStockThreshold: req.LowStockThreshold,
	}

	var updated *database.Product
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "product.update", "product", id.String()

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		svc := h.ProductSvc.WithRepo(tx)
		before, _ := svc.GetByID(cid, id)
		if err := svc.Update(cid, product, req.Images); err != nil {
			return err
		}
		after, err := svc.GetByID(cid, id)
		if err != nil {
			after = product
		}
		updated = after
		entry.OldValues, entry.NewValues = before, after
		return nil
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteProduct handles DELETE /api/products/{id}
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "products.delete")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "product.delete", "product", id.String()

	if err := h.auditMutation(&entry, func(tx *database.Repository) error {
		return h.ProductSvc.WithRepo(tx).Delete(cid, id)
	}); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

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
	Description       string                   `json:"description"`
	Price             float64                  `json:"price"`
	LowStockThreshold int                      `json:"low_stock_threshold"`
	Images            []dtos.ProductImageInput `json:"images"`
}

type updateProductRequest struct {
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	Price             float64                  `json:"price"`
	LowStockThreshold int                      `json:"low_stock_threshold"`
	Images            []dtos.ProductImageInput `json:"images"`
}

// CreateProduct handles POST /api/products
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "products.create") {
		return
	}

	var req createProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	created, err := h.ProductSvc.AddProduct(req.Name, req.Description, req.Images, req.Price, req.LowStockThreshold)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, created)
}

// GetProduct handles GET /api/products/{id}
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "products.read") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.ProductSvc.GetByID(id)
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

// ListProducts handles GET /api/products
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "products.read") {
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

	products, total, err := h.ProductSvc.List(offset, limit)
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
	if !h.requirePermission(w, r, "products.edit") {
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

	product := &database.Product{
		ID:                id,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		LowStockThreshold: req.LowStockThreshold,
	}

	if err := h.ProductSvc.Update(product, req.Images); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.ProductSvc.GetByID(id)
	if err != nil {
		respond(w, http.StatusOK, product)
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteProduct handles DELETE /api/products/{id}
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "products.delete") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.ProductSvc.Delete(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

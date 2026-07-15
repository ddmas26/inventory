package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type updateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// CreateProduct handles POST /api/products
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	product := &database.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}

	if err := h.Repo.CreateProduct(product); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "product with this name already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create product")
		return
	}

	respond(w, http.StatusCreated, product)
}

// GetProduct handles GET /api/products/{id}
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.Repo.GetProductByID(id)
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

	products, total, err := h.Repo.ListProducts(offset, limit)
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
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}

	if err := h.Repo.UpdateProduct(product); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "product with this name already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update product")
		return
	}

	// Re-fetch to return the updated record with refreshed timestamps
	updated, err := h.Repo.GetProductByID(id)
	if err != nil {
		respond(w, http.StatusOK, product)
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteProduct handles DELETE /api/products/{id}
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.Repo.DeleteProduct(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// ensure interface compliance
var _ = &gorm.DB{}

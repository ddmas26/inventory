package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type addStockRequest struct {
	InventoryID uuid.UUID `json:"inventory_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int       `json:"quantity"`
}

type deductStockRequest struct {
	InventoryID uuid.UUID `json:"inventory_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int       `json:"quantity"`
}

type setStockRequest struct {
	InventoryID uuid.UUID `json:"inventory_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int       `json:"quantity"`
}

type removeStockRequest struct {
	InventoryID uuid.UUID `json:"inventory_id"`
	ProductID   uuid.UUID `json:"product_id"`
}

type transferStockRequest struct {
	FromInventoryID uuid.UUID `json:"from_inventory_id"`
	ToInventoryID   uuid.UUID `json:"to_inventory_id"`
	ProductID       uuid.UUID `json:"product_id"`
	Quantity        int       `json:"quantity"`
}

// AddStock handles POST /api/stock/add
func (h *Handler) AddStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.create") {
		return
	}

	var req addStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	if err := h.StockSvc.AddStock(req.InventoryID, req.ProductID, req.Quantity); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stock, err := h.StockSvc.GetStock(req.InventoryID, req.ProductID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock added successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// DeductStock handles POST /api/stock/deduct
func (h *Handler) DeductStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.edit") {
		return
	}

	var req deductStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	if err := h.StockSvc.DeductStock(req.InventoryID, req.ProductID, req.Quantity); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stock, err := h.StockSvc.GetStock(req.InventoryID, req.ProductID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock deducted successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// SetStock handles POST /api/stock/set
func (h *Handler) SetStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.edit") {
		return
	}

	var req setStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Quantity < 0 {
		respondError(w, http.StatusBadRequest, "quantity cannot be negative")
		return
	}

	if err := h.StockSvc.SetStock(req.InventoryID, req.ProductID, req.Quantity); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stock, err := h.StockSvc.GetStock(req.InventoryID, req.ProductID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock set successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// ListStock handles GET /api/stock
func (h *Handler) ListStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.read") {
		return
	}

	filter := dtos.ListStockFilter{}

	if invID := r.URL.Query().Get("inventory_id"); invID != "" {
		id, err := uuid.Parse(invID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid inventory id")
			return
		}
		filter.InventoryID = &id
	}

	if prodID := r.URL.Query().Get("product_id"); prodID != "" {
		id, err := uuid.Parse(prodID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid product id")
			return
		}
		filter.ProductID = &id
	}

	if from := r.URL.Query().Get("created_from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid created_from format, use RFC3339")
			return
		}
		filter.CreatedFrom = &t
	}

	if to := r.URL.Query().Get("created_to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid created_to format, use RFC3339")
			return
		}
		filter.CreatedTo = &t
	}

	filter.Search = r.URL.Query().Get("search")
	filter.OrderBy = r.URL.Query().Get("order_by")
	filter.Sort = r.URL.Query().Get("sort")

	pageIndex, _ := strconv.Atoi(r.URL.Query().Get("page_index"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageIndex <= 1 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	filter.Offset = (pageIndex - 1) * pageSize
	filter.Limit = pageSize

	items, total, err := h.StockSvc.ListStock(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list stock")
		return
	}

	if items == nil {
		items = []dtos.StockDto{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       items,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// RemoveStock handles POST /api/stock/remove
func (h *Handler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.delete") {
		return
	}

	var req removeStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.StockSvc.RemoveStock(req.InventoryID, req.ProductID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "stock entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to remove stock")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// TransferStock handles POST /api/stock/transfer
func (h *Handler) TransferStock(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "stock.edit") {
		return
	}

	var req transferStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	if err := h.StockSvc.TransferStock(req.FromInventoryID, req.ToInventoryID, req.ProductID, req.Quantity); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"message": "stock transferred successfully"})
}

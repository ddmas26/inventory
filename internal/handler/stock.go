package handler

import (
	"errors"
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/google/uuid"
)

type addStockRequest struct {
	Quantity int `json:"quantity"`
}

type deductStockRequest struct {
	Quantity int `json:"quantity"`
}

type setStockRequest struct {
	Quantity int `json:"quantity"`
}

type transferStockRequest struct {
	FromInventoryID uuid.UUID `json:"from_inventory_id"`
	ToInventoryID   uuid.UUID `json:"to_inventory_id"`
	ProductID       uuid.UUID `json:"product_id"`
	Quantity        int       `json:"quantity"`
}

// AddStock handles POST /api/inventories/{invID}/products/{prodID}/stock/add
func (h *Handler) AddStock(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
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

	if err := h.Repo.AddStock(invID, prodID, req.Quantity); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "stock entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stock, err := h.Repo.GetStock(invID, prodID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock added successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// DeductStock handles POST /api/inventories/{invID}/products/{prodID}/stock/deduct
func (h *Handler) DeductStock(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
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

	if err := h.Repo.DeductStock(invID, prodID, req.Quantity); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "stock entry not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stock, err := h.Repo.GetStock(invID, prodID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock deducted successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// SetStock handles POST /api/inventories/{invID}/products/{prodID}/stock
func (h *Handler) SetStock(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
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

	if err := h.Repo.SetProductStock(invID, prodID, req.Quantity); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stock, err := h.Repo.GetStock(invID, prodID)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"message": "stock set successfully"})
		return
	}

	respond(w, http.StatusOK, stock)
}

// GetStock handles GET /api/inventories/{invID}/products/{prodID}/stock
func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	stock, err := h.Repo.GetStock(invID, prodID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "stock entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get stock")
		return
	}

	respond(w, http.StatusOK, stock)
}

// RemoveProductFromInventory handles DELETE /api/inventories/{invID}/products/{prodID}
func (h *Handler) RemoveProductFromInventory(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.Repo.RemoveProductFromInventory(invID, prodID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "stock entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to remove product from inventory")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// ListProductsAtInventory handles GET /api/inventories/{invID}/products
func (h *Handler) ListProductsAtInventory(w http.ResponseWriter, r *http.Request) {
	invID, err := uuid.Parse(r.PathValue("invID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	items, err := h.Repo.ListProductsAtInventory(invID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	if items == nil {
		items = []database.InventoryProduct{}
	}

	respond(w, http.StatusOK, items)
}

// ListInventoriesForProduct handles GET /api/products/{prodID}/inventories
func (h *Handler) ListInventoriesForProduct(w http.ResponseWriter, r *http.Request) {
	prodID, err := uuid.Parse(r.PathValue("prodID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	items, err := h.Repo.ListInventoriesForProduct(prodID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list inventories")
		return
	}

	if items == nil {
		items = []database.InventoryProduct{}
	}

	respond(w, http.StatusOK, items)
}

// TransferStock handles POST /api/stock/transfer
func (h *Handler) TransferStock(w http.ResponseWriter, r *http.Request) {
	var req transferStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	if err := h.Repo.TransferStock(req.FromInventoryID, req.ToInventoryID, req.ProductID, req.Quantity); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"message": "stock transferred successfully"})
}

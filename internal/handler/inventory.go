package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/google/uuid"
)

type createInventoryRequest struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type updateInventoryRequest struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// CreateInventory handles POST /api/inventories
func (h *Handler) CreateInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "inventories.create") {
		return
	}

	var req createInventoryRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	inv, err := h.InventorySvc.CreateInventory(req.Name, req.Address, req.Latitude, req.Longitude)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, inv)
}

// GetInventory handles GET /api/inventories/{id}
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "inventories.read") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	inventory, err := h.InventorySvc.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get inventory")
		return
	}

	respond(w, http.StatusOK, inventory)
}

// ListInventories handles GET /api/inventories
func (h *Handler) ListInventories(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "inventories.read") {
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

	inventories, total, err := h.InventorySvc.List(offset, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list inventories")
		return
	}

	if inventories == nil {
		inventories = []database.Inventory{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       inventories,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// UpdateInventory handles PUT /api/inventories/{id}
func (h *Handler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "inventories.edit") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	var req updateInventoryRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	inventory := &database.Inventory{
		ID:        id,
		Name:      req.Name,
		Address:   req.Address,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	if err := h.InventorySvc.Update(inventory); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.InventorySvc.GetByID(id)
	if err != nil {
		respond(w, http.StatusOK, inventory)
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteInventory handles DELETE /api/inventories/{id}
func (h *Handler) DeleteInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "inventories.delete") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	if err := h.InventorySvc.Delete(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete inventory")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) DashboardInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "dashboard.read") {
		return
	}

	dashData, err := h.InventorySvc.GetDashboardData()
	if err != nil {
		respondError(w, http.StatusBadRequest, "error retrieving dashboard data")
		return
	}

	respond(w, http.StatusOK, dashData)
}

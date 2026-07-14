package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	var req createInventoryRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	inventory := &database.Inventory{
		Name:      req.Name,
		Address:   req.Address,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	if err := h.Repo.CreateInventory(inventory); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "inventory with this name already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create inventory")
		return
	}

	respond(w, http.StatusCreated, inventory)
}

// GetInventory handles GET /api/inventories/{id}
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	inventory, err := h.Repo.GetInventoryByID(id)
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
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	inventories, total, err := h.Repo.ListInventories(offset, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list inventories")
		return
	}

	if inventories == nil {
		inventories = []database.Inventory{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":   inventories,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

// UpdateInventory handles PUT /api/inventories/{id}
func (h *Handler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
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

	if err := h.Repo.UpdateInventory(inventory); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "inventory with this name already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update inventory")
		return
	}

	updated, err := h.Repo.GetInventoryByID(id)
	if err != nil {
		respond(w, http.StatusOK, inventory)
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteInventory handles DELETE /api/inventories/{id}
func (h *Handler) DeleteInventory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	if err := h.Repo.DeleteInventory(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete inventory")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

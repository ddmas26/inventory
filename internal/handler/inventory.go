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
	claims, ok := h.authorize(w, r, "inventories.create")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	var req createInventoryRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var inv *database.Inventory
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType = "inventory.create", "inventory"

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		created, err := h.InventorySvc.WithRepo(tx).CreateInventory(cid, req.Name, req.Address, req.Latitude, req.Longitude)
		if err != nil {
			return err
		}
		inv = created
		entry.EntityID, entry.NewValues = created.ID.String(), created
		return nil
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, inv)
}

// GetInventory handles GET /api/inventories/{id}
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "inventories.read")
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
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	inventory, err := h.InventorySvc.GetByID(cid, id)
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
	claims, ok := h.authorize(w, r, "inventories.read")
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

	inventories, total, err := h.InventorySvc.List(cid, offset, limit)
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
	claims, ok := h.authorize(w, r, "inventories.edit")
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

	var updated *database.Inventory
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "inventory.update", "inventory", id.String()

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		svc := h.InventorySvc.WithRepo(tx)
		before, _ := svc.GetByID(cid, id)
		if err := svc.Update(cid, inventory); err != nil {
			return err
		}
		after, err := svc.GetByID(cid, id)
		if err != nil {
			after = inventory
		}
		updated = after
		entry.OldValues, entry.NewValues = before, after
		return nil
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "inventory not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteInventory handles DELETE /api/inventories/{id}
func (h *Handler) DeleteInventory(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "inventories.delete")
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
		respondError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}

	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "inventory.delete", "inventory", id.String()

	if err := h.auditMutation(&entry, func(tx *database.Repository) error {
		return h.InventorySvc.WithRepo(tx).Delete(cid, id)
	}); err != nil {
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
	claims, ok := h.authorize(w, r, "dashboard.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	dashData, err := h.InventorySvc.GetDashboardData(cid)
	if err != nil {
		respondError(w, http.StatusBadRequest, "error retrieving dashboard data")
		return
	}

	respond(w, http.StatusOK, dashData)
}

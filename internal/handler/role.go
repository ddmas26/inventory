package handler

import (
	"errors"
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createPermissionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updatePermissionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rolePermissionRequest struct {
	PermissionID uuid.UUID `json:"permission_id"`
}

// ── Roles ──────────────────────────────────────────────────────────────────

// CreateRole handles POST /api/roles
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.create")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	var req createRoleRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	role, err := h.RoleSvc.CreateRole(cid, req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, role)
}

// GetRole handles GET /api/roles/{id}
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.read")
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
		respondError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	role, err := h.RoleSvc.GetByID(cid, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get role")
		return
	}

	// Map to response with permissions
	userCount, err := h.Repo.GetUserCountByRoleID(cid, role.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get user count")
		return
	}

	resp := dtos.RoleWithPermissionsResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		UserCount:   int(userCount),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		Permissions: make([]dtos.PermissionResponse, 0, len(role.Permissions)),
	}
	for _, p := range role.Permissions {
		resp.Permissions = append(resp.Permissions, dtos.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	respond(w, http.StatusOK, resp)
}

// ListRoles handles GET /api/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	roles, err := h.RoleSvc.List(cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list roles")
		return
	}

	if roles == nil {
		roles = []database.Role{}
	}

	// Map to response with permissions and user count
	resp := make([]dtos.RoleWithPermissionsResponse, 0, len(roles))
	for _, role := range roles {
		userCount, _ := h.Repo.GetUserCountByRoleID(cid, role.ID)

		r := dtos.RoleWithPermissionsResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			UserCount:   int(userCount),
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
			Permissions: make([]dtos.PermissionResponse, 0, len(role.Permissions)),
		}
		for _, p := range role.Permissions {
			r.Permissions = append(r.Permissions, dtos.PermissionResponse{
				ID:          p.ID,
				Code:        p.Code,
				Name:        p.Name,
				Description: p.Description,
				CreatedAt:   p.CreatedAt,
				UpdatedAt:   p.UpdatedAt,
			})
		}
		resp = append(resp, r)
	}

	respond(w, http.StatusOK, resp)
}

// UpdateRole handles PUT /api/roles/{id}
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.edit")
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
		respondError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	var req updateRoleRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	role := &database.Role{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.RoleSvc.Update(cid, role); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.RoleSvc.GetByID(cid, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get updated role")
		return
	}

	// Map to response with permissions
	userCount, err := h.Repo.GetUserCountByRoleID(cid, updated.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get user count")
		return
	}

	resp := dtos.RoleWithPermissionsResponse{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		UserCount:   int(userCount),
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
		Permissions: make([]dtos.PermissionResponse, 0, len(updated.Permissions)),
	}
	for _, p := range updated.Permissions {
		resp.Permissions = append(resp.Permissions, dtos.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	respond(w, http.StatusOK, resp)
}

// DeleteRole handles DELETE /api/roles/{id}
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.delete")
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
		respondError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	if err := h.RoleSvc.Delete(cid, id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete role")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// AddPermissionToRole handles POST /api/roles/{id}/permissions
func (h *Handler) AddPermissionToRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.edit")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	roleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	var req rolePermissionRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	role, err := h.RoleSvc.AddPermissionToRole(cid, roleID, req.PermissionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role or permission not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Map to response
	resp := dtos.RoleWithPermissionsResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		Permissions: make([]dtos.PermissionResponse, 0, len(role.Permissions)),
	}
	for _, p := range role.Permissions {
		resp.Permissions = append(resp.Permissions, dtos.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	respond(w, http.StatusOK, resp)
}

// RemovePermissionFromRole handles DELETE /api/roles/{id}/permissions/{permissionId}
func (h *Handler) RemovePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "roles.edit")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	roleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	permID, err := uuid.Parse(r.PathValue("permissionId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	role, err := h.RoleSvc.RemovePermissionFromRole(cid, roleID, permID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role or permission not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Map to response
	resp := dtos.RoleWithPermissionsResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		Permissions: make([]dtos.PermissionResponse, 0, len(role.Permissions)),
	}
	for _, p := range role.Permissions {
		resp.Permissions = append(resp.Permissions, dtos.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	respond(w, http.StatusOK, resp)
}

// ── Permissions ────────────────────────────────────────────────────────────

// CreatePermission handles POST /api/permissions
func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "permissions.create") {
		return
	}

	var req createPermissionRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	perm, err := h.RoleSvc.CreatePermission(req.Code, req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, perm)
}

// ListPermissions handles GET /api/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "permissions.read") {
		return
	}

	permissions, err := h.RoleSvc.ListPermissions()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list permissions")
		return
	}

	if permissions == nil {
		permissions = []database.Permission{}
	}

	respond(w, http.StatusOK, permissions)
}

// UpdatePermission handles PUT /api/permissions/{id}
func (h *Handler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "permissions.edit") {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	var req updatePermissionRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	perm := &database.Permission{
		ID:          id,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.RoleSvc.UpdatePermission(perm); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "permission not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, perm)
}

// DeletePermission handles DELETE /api/permissions/{id}
func (h *Handler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "permissions.delete") {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	if err := h.RoleSvc.DeletePermission(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "permission not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete permission")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

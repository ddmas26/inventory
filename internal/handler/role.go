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

// setRolePermissionsRequest carries the complete desired permission set for a
// role (bulk update).
type setRolePermissionsRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
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

	var role *dtos.RoleResponse
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType = "role.create", "role"

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		created, err := h.RoleSvc.WithRepo(tx).CreateRole(cid, req.Name, req.Description)
		if err != nil {
			return err
		}
		role = created
		entry.EntityID, entry.NewValues = created.ID.String(), created
		return nil
	})
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

	respawn := dtos.RoleWithPermissionsResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		UserCount:   int(userCount),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		Permissions: make([]dtos.PermissionResponse, 0, len(role.Permissions)),
	}
	for _, p := range role.Permissions {
		respawn.Permissions = append(respawn.Permissions, dtos.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	h.audit(r, claims, "role.view", "role", id.String(), nil, nil, nil)
	respond(w, http.StatusOK, respawn)
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
	respawn := make([]dtos.RoleWithPermissionsResponse, 0, len(roles))
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
		respawn = append(respawn, r)
	}

	h.audit(r, claims, "role.list", "role", "", nil, nil, map[string]any{"count": len(respawn)})
	respond(w, http.StatusOK, respawn)
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

	var updated *database.Role
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "role.update", "role", id.String()

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		svc := h.RoleSvc.WithRepo(tx)
		before, _ := svc.GetByID(cid, id)
		if err := svc.Update(cid, role); err != nil {
			return err
		}
		after, err := svc.GetByID(cid, id)
		if err != nil {
			return err
		}
		updated = after
		entry.OldValues, entry.NewValues = before, after
		return nil
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
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

	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "role.delete", "role", id.String()

	if err := h.auditMutation(&entry, func(tx *database.Repository) error {
		return h.RoleSvc.WithRepo(tx).Delete(cid, id)
	}); err != nil {
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

// SetRolePermissions handles PUT /api/roles/{id}/permissions. It replaces the
// role's entire permission set with the IDs supplied in the body, so the UI can
// stage every checkbox change locally and persist them with a single request.
func (h *Handler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
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

	var req setRolePermissionsRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var role *database.Role
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "role.set_permissions", "role", roleID.String()
	entry.Metadata = map[string]any{"permission_count": len(req.PermissionIDs)}

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		r, err := h.RoleSvc.WithRepo(tx).SetPermissions(cid, roleID, req.PermissionIDs)
		if err != nil {
			return err
		}
		role = r
		return nil
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "role not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	userCount, err := h.Repo.GetUserCountByRoleID(cid, role.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get user count")
		return
	}

	respond(w, http.StatusOK, roleWithPermissions(role, int(userCount)))
}

// roleWithPermissions maps a role (with its permissions preloaded) to the API DTO.
func roleWithPermissions(role *database.Role, userCount int) dtos.RoleWithPermissionsResponse {
	resp := dtos.RoleWithPermissionsResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		UserCount:   userCount,
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
	return resp
}

// ── Permissions ────────────────────────────────────────────────────────────

// CreatePermission handles POST /api/permissions
func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "permissions.create")
	if !ok {
		return
	}

	var req createPermissionRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var perm *dtos.PermissionResponse
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType = "permission.create", "permission"

	err := h.auditMutation(&entry, func(tx *database.Repository) error {
		p, err := h.RoleSvc.WithRepo(tx).CreatePermission(req.Code, req.Name, req.Description)
		if err != nil {
			return err
		}
		perm = p
		entry.EntityID, entry.NewValues = p.ID.String(), p
		return nil
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, perm)
}

// ListPermissions handles GET /api/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "permissions.read")
	if !ok {
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

	h.audit(r, claims, "permission.list", "permission", "", nil, nil, map[string]any{"count": len(permissions)})
	respond(w, http.StatusOK, permissions)
}

// UpdatePermission handles PUT /api/permissions/{id}
func (h *Handler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "permissions.edit")
	if !ok {
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

	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "permission.update", "permission", id.String()
	entry.NewValues = perm

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		return h.RoleSvc.WithRepo(tx).UpdatePermission(perm)
	})
	if err != nil {
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
	claims, ok := h.authorize(w, r, "permissions.delete")
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "permission.delete", "permission", id.String()

	if err := h.auditMutation(&entry, func(tx *database.Repository) error {
		return h.RoleSvc.WithRepo(tx).DeletePermission(id)
	}); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "permission not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete permission")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

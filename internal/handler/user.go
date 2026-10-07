package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type createUserRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	RoleID   *string `json:"role_id"`
}

type updateUserRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password,omitempty"`
	RoleID   *string `json:"role_id"`
}

// CreateUser handles POST /api/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.create")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	var req createUserRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var roleID *uuid.UUID
	if req.RoleID != nil {
		parsed, err := uuid.Parse(*req.RoleID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid role_id")
			return
		}
		roleID = &parsed
	}

	created, err := h.UserSvc.CreateUser(cid, req.Name, req.Email, req.Password, roleID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, created)
}

// GetUser handles GET /api/users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.read")
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
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.UserSvc.GetByID(cid, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	respond(w, http.StatusOK, user)
}

// ListUsers handles GET /api/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.read")
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
	search := r.URL.Query().Get("search")

	if pageIndex <= 1 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (pageIndex - 1) * pageSize
	limit := pageSize

	users, total, err := h.UserSvc.List(cid, offset, limit, search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	if users == nil {
		users = []dtos.UserDto{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       users,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// UpdateUser handles PUT /api/users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.edit")
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
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Fetch existing user to preserve password if not provided
	existing, err := h.UserSvc.GetByID(cid, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	password := existing.Password
	if req.Password != "" {
		password = req.Password
	}

	var roleID *uuid.UUID
	if req.RoleID != nil {
		parsed, err := uuid.Parse(*req.RoleID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid role_id")
			return
		}
		roleID = &parsed
	}

	user := &database.User{
		ID:       id,
		Name:     req.Name,
		Email:    req.Email,
		Password: password,
		RoleID:   roleID,
	}

	if err := h.UserSvc.Update(cid, user); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.UserSvc.GetByID(cid, id)
	if err != nil {
		respond(w, http.StatusOK, user)
		return
	}

	respond(w, http.StatusOK, updated)
}

// DeleteUser handles DELETE /api/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.delete")
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
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.UserSvc.Delete(cid, id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// ActivateUser handles PATCH /api/users/{id}/activate
func (h *Handler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.edit")
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
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.UserSvc.ActivateUser(cid, id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to activate user")
		return
	}

	respond(w, http.StatusOK, map[string]string{"message": "user activated"})
}

// DeactivateUser handles PATCH /api/users/{id}/deactivate
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "users.edit")
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
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.UserSvc.DeactivateUser(cid, id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to deactivate user")
		return
	}

	respond(w, http.StatusOK, map[string]string{"message": "user deactivated"})
}

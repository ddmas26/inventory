package handler

import (
	"net/http"

	"github.com/ddmas26/inventory/internal/dtos"
)

// Register handles POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dtos.RegisterRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.AuthSvc.Register(req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusCreated, user)
}

// Login handles POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dtos.LoginRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.AuthSvc.Login(req)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respond(w, http.StatusOK, resp)
}

// Me handles GET /api/auth/me — returns the current user's claims (including permissions).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, err := h.sessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respond(w, http.StatusOK, dtos.ClaimsResponse{
		UserID:      claims.UserID,
		Name:        claims.Name,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	})
}

// Refresh handles POST /api/auth/refresh — exchanges a refresh token for a new
// access + refresh token pair. The presented refresh token is rotated.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dtos.RefreshRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.AuthSvc.Refresh(req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respond(w, http.StatusOK, resp)
}

// Logout handles POST /api/auth/logout — revokes the caller's access session and,
// when supplied, their refresh token. Always returns 204 so clients can clear
// their local state regardless of the token's condition.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dtos.LogoutRequest
	_ = decode(r, &req) // body is optional

	h.AuthSvc.Logout(bearerToken(r), req.RefreshToken)

	respond(w, http.StatusNoContent, nil)
}

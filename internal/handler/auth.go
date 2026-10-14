package handler

import (
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/ddmas26/inventory/internal/service"
	"github.com/google/uuid"
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

	// Onboarding a new tenant: the company is created pending approval.
	h.Audit.Record(service.AuditEntry{
		CompanyID:  &user.CompanyID,
		ActorType:  database.AuditActorUser,
		ActorID:    &user.ID,
		ActorName:  user.Name,
		Action:     "auth.register",
		EntityType: "company",
		EntityID:   user.CompanyID.String(),
		NewValues:  map[string]any{"company_name": req.CompanyName, "email": user.Email},
		IP:         clientIP(r),
		UserAgent:  r.UserAgent(),
		RequestID:  RequestIDFromContext(r.Context()),
	})

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
		h.Audit.Record(service.AuditEntry{
			ActorType:  database.AuditActorUser,
			Action:     "auth.login_failed",
			EntityType: "user",
			Metadata:   map[string]string{"email": req.Email},
			IP:         clientIP(r),
			UserAgent:  r.UserAgent(),
			RequestID:  RequestIDFromContext(r.Context()),
		})
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	companyID := resp.User.CompanyID
	userID := resp.User.ID
	h.Audit.Record(service.AuditEntry{
		CompanyID:  &companyID,
		ActorType:  database.AuditActorUser,
		ActorID:    &userID,
		ActorName:  resp.User.Name,
		Action:     "auth.login",
		EntityType: "user",
		EntityID:   userID.String(),
		IP:         clientIP(r),
		UserAgent:  r.UserAgent(),
		RequestID:  RequestIDFromContext(r.Context()),
	})

	respond(w, http.StatusOK, resp)
}

// Me handles GET /api/auth/me — returns the current user's claims (including permissions).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, err := h.sessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Read the company's status fresh so the client's "awaiting approval" screen
	// clears as soon as a platform admin approves the company (no re-login needed).
	companyStatus := claims.CompanyStatus
	if cid, err := uuid.Parse(claims.CompanyID); err == nil {
		if company, err := h.Repo.GetCompanyByID(cid); err == nil {
			companyStatus = string(company.Status)
		}
	}

	respond(w, http.StatusOK, dtos.ClaimsResponse{
		UserID:        claims.UserID,
		CompanyID:     claims.CompanyID,
		CompanyName:   claims.CompanyName,
		CompanyStatus: companyStatus,
		IsRoot:        claims.IsRoot,
		Name:          claims.Name,
		Email:         claims.Email,
		Role:          claims.Role,
		Permissions:   claims.Permissions,
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

	// Capture the actor before the session is revoked.
	if claims, err := h.sessionFromRequest(r); err == nil {
		h.audit(r, claims, "auth.logout", "user", claims.UserID, nil, nil, nil)
	}

	h.AuthSvc.Logout(bearerToken(r), req.RefreshToken)

	respond(w, http.StatusNoContent, nil)
}

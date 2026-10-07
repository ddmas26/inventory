package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// ── Platform auth ──────────────────────────────────────────────────────────

// PlatformLogin handles POST /api/platform/auth/login
func (h *Handler) PlatformLogin(w http.ResponseWriter, r *http.Request) {
	var req dtos.PlatformLoginRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.PlatformAuth.Login(req)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respond(w, http.StatusOK, resp)
}

// PlatformLogout handles POST /api/platform/auth/logout
func (h *Handler) PlatformLogout(w http.ResponseWriter, r *http.Request) {
	h.PlatformAuth.Logout(bearerToken(r))
	respond(w, http.StatusNoContent, nil)
}

// PlatformMe handles GET /api/platform/auth/me
func (h *Handler) PlatformMe(w http.ResponseWriter, r *http.Request) {
	claims, err := h.platformSessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	uid, err := uuid.Parse(claims.PlatformUserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	user, err := h.Repo.GetPlatformUserByID(uid)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "platform user not found")
		return
	}

	respond(w, http.StatusOK, dtos.PlatformUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// ── Company administration ──────────────────────────────────────────────────

// PlatformDashboard handles GET /api/platform/dashboard — status counts plus the
// latest pending companies.
func (h *Handler) PlatformDashboard(w http.ResponseWriter, r *http.Request) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	counts, err := h.Repo.CountCompanies()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count companies")
		return
	}

	pending, _, err := h.Repo.ListCompanies(dtos.CompanyListFilter{
		Status: string(database.CompanyStatusPending),
		Limit:  20,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load pending companies")
		return
	}
	if pending == nil {
		pending = []dtos.CompanyResponse{}
	}

	respond(w, http.StatusOK, dtos.PlatformDashboardDto{Counts: *counts, Pending: pending})
}

// ListCompanies handles GET /api/platform/companies with status/search/pagination.
func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
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

	filter := dtos.CompanyListFilter{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
		Offset: (pageIndex - 1) * pageSize,
		Limit:  pageSize,
	}

	companies, total, err := h.Repo.ListCompanies(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list companies")
		return
	}
	if companies == nil {
		companies = []dtos.CompanyResponse{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       companies,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// GetCompany handles GET /api/platform/companies/{id} — company detail with its root user.
func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company id")
		return
	}

	company, err := h.Repo.GetCompanyByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "company not found")
		return
	}

	resp := dtos.CompanyResponse{
		ID:         company.ID,
		Name:       company.Name,
		Slug:       company.Slug,
		Phone:      company.Phone,
		Status:     string(company.Status),
		ApprovedAt: company.ApprovedAt,
		CreatedAt:  company.CreatedAt,
		UpdatedAt:  company.UpdatedAt,
	}
	if root, err := h.Repo.GetRootUserByCompany(company.ID); err == nil {
		resp.RootUserID = &root.ID
		resp.RootUserName = root.Name
		resp.RootUserEmail = root.Email
		resp.RootUserPhone = root.Phone
	}

	respond(w, http.StatusOK, resp)
}

// updateCompanyStatus applies a status transition and returns the new state.
func (h *Handler) updateCompanyStatus(w http.ResponseWriter, r *http.Request, status database.CompanyStatus) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company id")
		return
	}

	company, err := h.Repo.UpdateCompanyStatus(id, status)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "company not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update company")
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"id":          company.ID,
		"status":      company.Status,
		"approved_at": company.ApprovedAt,
	})
}

// ApproveCompany handles PATCH /api/platform/companies/{id}/approve
func (h *Handler) ApproveCompany(w http.ResponseWriter, r *http.Request) {
	h.updateCompanyStatus(w, r, database.CompanyStatusApproved)
}

// RejectCompany handles PATCH /api/platform/companies/{id}/reject
func (h *Handler) RejectCompany(w http.ResponseWriter, r *http.Request) {
	h.updateCompanyStatus(w, r, database.CompanyStatusRejected)
}

// SuspendCompany handles PATCH /api/platform/companies/{id}/suspend
func (h *Handler) SuspendCompany(w http.ResponseWriter, r *http.Request) {
	h.updateCompanyStatus(w, r, database.CompanyStatusSuspended)
}

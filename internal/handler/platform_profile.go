package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// ListProfileRequests handles GET /api/platform/profile-requests — profile change
// requests across all companies, optionally filtered by status.
func (h *Handler) ListProfileRequests(w http.ResponseWriter, r *http.Request) {
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

	reqs, total, err := h.CompanyProfileSvc.ListForPlatform(
		r.URL.Query().Get("status"), (pageIndex-1)*pageSize, pageSize,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list profile requests")
		return
	}
	if reqs == nil {
		reqs = []dtos.CompanyProfileRequestDto{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       reqs,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// GetProfileRequest handles GET /api/platform/profile-requests/{id} — the request
// plus the company's live profile so the reviewer can diff them.
func (h *Handler) GetProfileRequest(w http.ResponseWriter, r *http.Request) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	detail, err := h.CompanyProfileSvc.GetRequestDetail(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "profile request not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load profile request")
		return
	}

	respond(w, http.StatusOK, detail)
}

// reviewProfileRequest applies an approve/reject decision made by a platform admin.
func (h *Handler) reviewProfileRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	claims, err := h.platformSessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	reviewerID, err := uuid.Parse(claims.PlatformUserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	// The review body is optional (approve needs none; reject usually carries a
	// reason). A missing/empty body is not an error.
	var req dtos.ReviewCompanyProfileRequest
	_ = decode(r, &req)

	action := "company_profile.approve"
	meta := map[string]any{}
	if !approve {
		action = "company_profile.reject"
		meta["reason"] = req.Reason
	}

	entry := auditPlatformEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = action, "company_profile_request", id.String()
	entry.Metadata = meta

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		svc := h.CompanyProfileSvc.WithRepo(tx)
		// Attribute the decision to the company the request belongs to.
		if detail, err := svc.GetRequestDetail(id); err == nil {
			companyID := detail.Request.CompanyID
			entry.CompanyID = &companyID
		}
		if approve {
			return svc.Approve(id, reviewerID)
		}
		return svc.Reject(id, reviewerID, req.Reason)
	})
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			respondError(w, http.StatusNotFound, "profile request not found")
		case errors.Is(err, database.ErrNotPending):
			respondError(w, http.StatusConflict, "profile request has already been reviewed")
		default:
			respondError(w, http.StatusInternalServerError, "failed to review profile request")
		}
		return
	}

	detail, err := h.CompanyProfileSvc.GetRequestDetail(id)
	if err != nil {
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	respond(w, http.StatusOK, detail)
}

// ApproveProfileRequest handles PATCH /api/platform/profile-requests/{id}/approve
func (h *Handler) ApproveProfileRequest(w http.ResponseWriter, r *http.Request) {
	h.reviewProfileRequest(w, r, true)
}

// RejectProfileRequest handles PATCH /api/platform/profile-requests/{id}/reject
func (h *Handler) RejectProfileRequest(w http.ResponseWriter, r *http.Request) {
	h.reviewProfileRequest(w, r, false)
}

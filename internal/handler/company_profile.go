package handler

import (
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// GetCompanyProfile handles GET /api/company/profile — the company identity, the
// live (approved) profile, and the pending change request if any.
func (h *Handler) GetCompanyProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "company.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	overview, err := h.CompanyProfileSvc.GetOverview(cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load company profile")
		return
	}

	h.audit(r, claims, "company_profile.view", "company", cid.String(), nil, nil, nil)
	respond(w, http.StatusOK, overview)
}

// UpdateCompanyProfile handles PUT /api/company/profile. It does not change the
// live profile: it stages the proposed values as a change request that a platform
// admin must approve.
func (h *Handler) UpdateCompanyProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "company.edit")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid user in session")
		return
	}

	var req dtos.UpdateCompanyProfileRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var overview *dtos.CompanyProfileResponse
	entry := auditUserEntry(r, claims)
	entry.Action, entry.EntityType, entry.EntityID = "company_profile.submit", "company", cid.String()
	entry.NewValues = req

	err = h.auditMutation(&entry, func(tx *database.Repository) error {
		o, err := h.CompanyProfileSvc.WithRepo(tx).Submit(cid, uid, req)
		if err != nil {
			return err
		}
		overview = o
		return nil
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respond(w, http.StatusOK, overview)
}

// ListCompanyProfileRequests handles GET /api/company/profile/requests — the
// company's own profile-change history.
func (h *Handler) ListCompanyProfileRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "company.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	reqs, err := h.CompanyProfileSvc.History(cid, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load profile requests")
		return
	}
	if reqs == nil {
		reqs = []dtos.CompanyProfileRequestDto{}
	}

	respond(w, http.StatusOK, reqs)
}

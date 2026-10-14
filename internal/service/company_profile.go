package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// CompanyProfileService owns the company profile and its change-request workflow.
// Profile edits never touch the live profile directly: a user submits a change
// request and a platform admin approves or rejects it.
type CompanyProfileService struct {
	repo *database.Repository
}

func NewCompanyProfileService(repo *database.Repository) *CompanyProfileService {
	return &CompanyProfileService{repo: repo}
}

// WithRepo returns a copy of the service bound to another repository — used to
// run an operation on a transaction-scoped repository.
func (s *CompanyProfileService) WithRepo(repo *database.Repository) *CompanyProfileService {
	return &CompanyProfileService{repo: repo}
}

// GetOverview returns the company identity, the live profile and the pending
// change request (when one is under review).
func (s *CompanyProfileService) GetOverview(companyID uuid.UUID) (*dtos.CompanyProfileResponse, error) {
	company, err := s.repo.GetCompanyByID(companyID)
	if err != nil {
		return nil, err
	}

	resp := &dtos.CompanyProfileResponse{
		CompanyID: company.ID,
		Name:      company.Name,
		Slug:      company.Slug,
		Status:    string(company.Status),
	}

	profile, err := s.repo.GetCompanyProfile(companyID)
	if err == nil {
		resp.Profile = profileToDto(profile)
	} else if !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}

	pending, err := s.repo.GetPendingCompanyProfileRequest(companyID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		dto, err := s.toRequestDto(pending)
		if err != nil {
			return nil, err
		}
		resp.Pending = &dto
	}

	return resp, nil
}

// Submit stages a profile change: it replaces any pending request and returns the
// refreshed overview so the UI can immediately show the "under review" state.
func (s *CompanyProfileService) Submit(companyID, userID uuid.UUID, req dtos.UpdateCompanyProfileRequest) (*dtos.CompanyProfileResponse, error) {
	payload, err := json.Marshal(dtos.CompanyProfileDto{
		LegalName:   req.LegalName,
		Email:       req.Email,
		Phone:       req.Phone,
		Website:     req.Website,
		Address:     req.Address,
		City:        req.City,
		Country:     req.Country,
		TaxID:       req.TaxID,
		LogoURL:     req.LogoURL,
		Description: req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("encode profile: %w", err)
	}

	if _, err := s.repo.CreateCompanyProfileRequest(companyID, userID, string(payload)); err != nil {
		return nil, err
	}

	return s.GetOverview(companyID)
}

// History returns a company's profile-change history, newest first.
func (s *CompanyProfileService) History(companyID uuid.UUID, limit int) ([]dtos.CompanyProfileRequestDto, error) {
	reqs, err := s.repo.ListCompanyProfileRequests(companyID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dtos.CompanyProfileRequestDto, 0, len(reqs))
	for i := range reqs {
		dto, err := s.toRequestDto(&reqs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// ListForPlatform returns profile-change requests across all companies.
func (s *CompanyProfileService) ListForPlatform(status string, offset, limit int) ([]dtos.CompanyProfileRequestDto, int64, error) {
	reqs, total, err := s.repo.ListCompanyProfileRequestsForPlatform(status, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	out := make([]dtos.CompanyProfileRequestDto, 0, len(reqs))
	for i := range reqs {
		dto, err := s.toRequestDto(&reqs[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, dto)
	}
	return out, total, nil
}

// GetRequestDetail returns a single request plus the company's live profile.
func (s *CompanyProfileService) GetRequestDetail(id uuid.UUID) (*dtos.CompanyProfileRequestDetailDto, error) {
	req, err := s.repo.GetCompanyProfileRequestByID(id)
	if err != nil {
		return nil, err
	}

	dto, err := s.toRequestDto(req)
	if err != nil {
		return nil, err
	}

	detail := &dtos.CompanyProfileRequestDetailDto{Request: dto}
	if profile, err := s.repo.GetCompanyProfile(req.CompanyID); err == nil {
		detail.Current = profileToDto(profile)
	}
	return detail, nil
}

// Approve applies a pending request to the live profile.
func (s *CompanyProfileService) Approve(requestID, reviewerID uuid.UUID) error {
	return s.repo.ApproveCompanyProfileRequest(requestID, reviewerID)
}

// Reject declines a pending request, recording the reason.
func (s *CompanyProfileService) Reject(requestID, reviewerID uuid.UUID, reason string) error {
	return s.repo.RejectCompanyProfileRequest(requestID, reviewerID, reason)
}

// toRequestDto maps a request model onto the DTO, enriching it with the company
// and requester names and decoding the stored profile payload.
func (s *CompanyProfileService) toRequestDto(req *database.CompanyProfileRequest) (dtos.CompanyProfileRequestDto, error) {
	dto := dtos.CompanyProfileRequestDto{
		ID:         req.ID,
		CompanyID:  req.CompanyID,
		Status:     string(req.Status),
		Reason:     req.Reason,
		ReviewedBy: req.ReviewedBy,
		ReviewedAt: req.ReviewedAt,
		CreatedAt:  req.CreatedAt,
		UpdatedAt:  req.UpdatedAt,
	}

	if req.RequestedBy != uuid.Nil {
		requestedBy := req.RequestedBy
		dto.RequestedBy = &requestedBy
		if u, err := s.repo.GetUserByID(requestedBy); err == nil {
			dto.RequestedByName = u.Name
		}
	}
	if company, err := s.repo.GetCompanyByID(req.CompanyID); err == nil {
		dto.CompanyName = company.Name
	}
	if err := json.Unmarshal([]byte(req.Payload), &dto.Profile); err != nil {
		return dto, fmt.Errorf("decode profile payload: %w", err)
	}
	return dto, nil
}

// profileToDto maps a stored profile onto the DTO.
func profileToDto(p *database.CompanyProfile) dtos.CompanyProfileDto {
	updated := p.UpdatedAt
	return dtos.CompanyProfileDto{
		LegalName:   p.LegalName,
		Email:       p.Email,
		Phone:       p.Phone,
		Website:     p.Website,
		Address:     p.Address,
		City:        p.City,
		Country:     p.Country,
		TaxID:       p.TaxID,
		LogoURL:     p.LogoURL,
		Description: p.Description,
		UpdatedAt:   &updated,
	}
}

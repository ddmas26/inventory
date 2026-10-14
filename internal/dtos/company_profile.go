package dtos

import (
	"time"

	"github.com/google/uuid"
)

// CompanyProfileDto is a company's business profile (both the live/approved copy
// and the proposed copy inside a change request).
type CompanyProfileDto struct {
	LegalName   string     `json:"legal_name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Website     string     `json:"website"`
	Address     string     `json:"address"`
	City        string     `json:"city"`
	Country     string     `json:"country"`
	TaxID       string     `json:"tax_id"`
	LogoURL     string     `json:"logo_url"`
	Description string     `json:"description"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// CompanyProfileResponse is the tenant-facing overview: the company identity, the
// live (approved) profile, and the pending change request if one is under review.
type CompanyProfileResponse struct {
	CompanyID uuid.UUID                 `json:"company_id"`
	Name      string                    `json:"name"`
	Slug      string                    `json:"slug"`
	Status    string                    `json:"status"`
	Profile   CompanyProfileDto         `json:"profile"`
	Pending   *CompanyProfileRequestDto `json:"pending,omitempty"`
}

// UpdateCompanyProfileRequest is the tenant's proposed profile. Submitting it
// does not change the live profile — it creates a change request for review.
type UpdateCompanyProfileRequest struct {
	LegalName   string `json:"legal_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Website     string `json:"website"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Country     string `json:"country"`
	TaxID       string `json:"tax_id"`
	LogoURL     string `json:"logo_url"`
	Description string `json:"description"`
}

// CompanyProfileRequestDto is a staged profile change: pending, approved or
// rejected. It doubles as history for the tenant and as a review item for the
// platform.
type CompanyProfileRequestDto struct {
	ID              uuid.UUID         `json:"id"`
	CompanyID       uuid.UUID         `json:"company_id"`
	CompanyName     string            `json:"company_name,omitempty"`
	RequestedBy     *uuid.UUID        `json:"requested_by,omitempty"`
	RequestedByName string            `json:"requested_by_name,omitempty"`
	Profile         CompanyProfileDto `json:"profile"`
	Status          string            `json:"status"`
	Reason          string            `json:"reason"`
	ReviewedBy      *uuid.UUID        `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time        `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// CompanyProfileRequestDetailDto adds the live profile so a reviewer can diff the
// requested values against the currently approved ones.
type CompanyProfileRequestDetailDto struct {
	Request CompanyProfileRequestDto `json:"request"`
	Current CompanyProfileDto        `json:"current"`
}

// ReviewCompanyProfileRequest is the (optional) body for approve/reject. The
// reason is required by convention when rejecting.
type ReviewCompanyProfileRequest struct {
	Reason string `json:"reason"`
}

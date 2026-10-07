package dtos

import (
	"time"

	"github.com/google/uuid"
)

// CompanyListFilter filters the platform admin's company list.
type CompanyListFilter struct {
	Status string // pending | approved | rejected | suspended | "" (all)
	Search string // matches name, slug or phone
	Offset int
	Limit  int
}

// CompanyResponse is the platform admin's view of a company.
type CompanyResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Slug       string     `json:"slug"`
	Phone      string     `json:"phone"`
	Status     string     `json:"status"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Root user details (populated on the detail endpoint).
	RootUserID    *uuid.UUID `json:"root_user_id,omitempty"`
	RootUserName  string     `json:"root_user_name,omitempty"`
	RootUserEmail string     `json:"root_user_email,omitempty"`
	RootUserPhone string     `json:"root_user_phone,omitempty"`
}

// PlatformCountsDto holds the company counts shown on the platform dashboard.
type PlatformCountsDto struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Approved  int64 `json:"approved"`
	Rejected  int64 `json:"rejected"`
	Suspended int64 `json:"suspended"`
}

// PlatformDashboardDto is returned by the platform dashboard endpoint.
type PlatformDashboardDto struct {
	Counts  PlatformCountsDto `json:"counts"`
	Pending []CompanyResponse `json:"pending"`
}

// UpdateCompanyStatusRequest is the body for approve/reject/suspend actions.
type UpdateCompanyStatusRequest struct {
	// Reason is an optional note recorded for audit (not persisted yet).
	Reason string `json:"reason"`
}

// PlatformUserResponse is the public representation of a platform admin.
type PlatformUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

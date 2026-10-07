package dtos

import (
	"time"

	"github.com/google/uuid"
)

// UserResponse is the public representation of a user (no password).
type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID uuid.UUID  `json:"company_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	IsRoot    bool       `json:"is_root"`
	IsActive  bool       `json:"is_active"`
	RoleID    *uuid.UUID `json:"role_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// UserDto is used for listing users (no password).
type UserDto struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID uuid.UUID  `json:"company_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	IsRoot    bool       `json:"is_root"`
	IsActive  bool       `json:"is_active"`
	RoleID    *uuid.UUID `json:"role_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateUserRequest is the payload for creating a new user.
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// UpdateUserRequest is the payload for updating an existing user.
type UpdateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password,omitempty"`
}

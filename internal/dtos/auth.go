package dtos

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned by login and refresh. ExpiresIn is the access token
// lifetime in seconds.
type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// RefreshRequest is the body of POST /api/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest is the optional body of POST /api/auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RegisterRequest is the body of POST /api/auth/register. It creates a brand new
// company (in the "pending" state, awaiting platform approval) together with its
// root user, who is granted every permission once the company is approved.
type RegisterRequest struct {
	CompanyName  string `json:"company_name"`
	CompanyPhone string `json:"company_phone"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
}

// ClaimsResponse is returned by GET /api/auth/me. CompanyStatus lets the client
// show the "awaiting approval" screen; IsRoot marks the company owner.
type ClaimsResponse struct {
	UserID        string   `json:"user_id"`
	CompanyID     string   `json:"company_id"`
	CompanyName   string   `json:"company_name"`
	CompanyStatus string   `json:"company_status"`
	IsRoot        bool     `json:"is_root"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	Permissions   []string `json:"permissions"`
}

// PlatformLoginRequest is the body of POST /api/platform/auth/login.
type PlatformLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PlatformLoginResponse is returned by a successful platform login.
type PlatformLoginResponse struct {
	Token     string               `json:"token"`
	ExpiresIn int                  `json:"expires_in"`
	User      PlatformUserResponse `json:"user"`
}

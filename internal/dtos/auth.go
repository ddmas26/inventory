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
// company together with its root user, who is granted every permission.
type RegisterRequest struct {
	CompanyName string `json:"company_name"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

// ClaimsResponse is returned by GET /api/auth/me.
type ClaimsResponse struct {
	UserID      string   `json:"user_id"`
	CompanyID   string   `json:"company_id"`
	CompanyName string   `json:"company_name"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

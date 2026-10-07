package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"golang.org/x/crypto/bcrypt"
)

// PlatformAuthService authenticates platform operators (the accounts that review
// and approve companies) and issues platform sessions. It is intentionally
// separate from AuthService so the platform can be split into its own service.
type PlatformAuthService struct {
	repo    *database.Repository
	session *SessionStore
}

func NewPlatformAuthService(repo *database.Repository, session *SessionStore) *PlatformAuthService {
	return &PlatformAuthService{repo: repo, session: session}
}

// Login authenticates a platform user and issues a platform session token.
func (s *PlatformAuthService) Login(req dtos.PlatformLoginRequest) (*dtos.PlatformLoginResponse, error) {
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	user, err := s.repo.GetPlatformUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}
	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.session.CreatePlatformSession(context.Background(), &PlatformSessionClaims{
		PlatformUserID: user.ID.String(),
		Name:           user.Name,
		Email:          user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("create platform session: %w", err)
	}

	return &dtos.PlatformLoginResponse{
		Token:     token,
		ExpiresIn: int(s.session.AccessTTL().Seconds()),
		User: dtos.PlatformUserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Phone:     user.Phone,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

// Logout revokes a platform session. It is best effort and idempotent.
func (s *PlatformAuthService) Logout(token string) {
	if token == "" {
		return
	}
	_ = s.session.DeletePlatformSession(context.Background(), token)
}

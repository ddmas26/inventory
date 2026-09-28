package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic.
type AuthService struct {
	repo    *database.Repository
	session *SessionStore
}

func NewAuthService(repo *database.Repository, session *SessionStore) *AuthService {
	return &AuthService{repo: repo, session: session}
}

// Register creates a new user account.
func (s *AuthService) Register(req dtos.RegisterRequest) (*dtos.UserResponse, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" || len(req.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Check duplicate email
	existing, _ := s.repo.GetUserByEmail(req.Email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &database.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		IsActive: true,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	resp := toUserResponse(user)
	return &resp, nil
}

// Login authenticates a user and issues a Redis-backed access + refresh token pair.
func (s *AuthService) Login(req dtos.LoginRequest) (*dtos.LoginResponse, error) {
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return s.issueTokens(user)
}

// Refresh exchanges a valid refresh token for a brand new token pair. The presented
// refresh token and the access token it was issued with are both revoked, so a
// refresh token can only ever be used once (rotation).
func (s *AuthService) Refresh(refreshToken string) (*dtos.LoginResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	ctx := context.Background()

	record, err := s.session.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}
	if record == nil {
		return nil, errors.New("refresh token is invalid or expired")
	}

	// Rotate: revoke the presented refresh token and its paired access token.
	_ = s.session.DeleteRefreshToken(ctx, refreshToken)
	if record.SessionToken != "" {
		_ = s.session.DeleteSession(ctx, record.SessionToken)
	}

	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		return nil, errors.New("refresh token is invalid")
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user no longer exists")
	}
	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	return s.issueTokens(user)
}

// Logout revokes an access session and a refresh token in Redis. It is idempotent
// and best-effort: unknown or already-expired tokens are simply ignored.
func (s *AuthService) Logout(accessToken, refreshToken string) {
	ctx := context.Background()

	if accessToken != "" {
		_ = s.session.DeleteSession(ctx, accessToken)
	}
	if refreshToken != "" {
		_ = s.session.DeleteRefreshToken(ctx, refreshToken)
	}
}

// issueTokens loads the user's role/permissions and stores a fresh access session
// plus a refresh token in Redis.
func (s *AuthService) issueTokens(user *database.User) (*dtos.LoginResponse, error) {
	fullUser, err := s.repo.GetUserWithRoleByID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("load user claims: %w", err)
	}

	ctx := context.Background()

	accessToken, err := s.session.CreateSession(ctx, sessionClaims(fullUser))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	refreshToken, err := s.session.CreateRefreshToken(ctx, fullUser.ID.String(), accessToken)
	if err != nil {
		_ = s.session.DeleteSession(ctx, accessToken)
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	return &dtos.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.session.AccessTTL().Seconds()),
		User:         toUserResponse(fullUser),
	}, nil
}

// sessionClaims builds the Redis session payload for a user.
func sessionClaims(u *database.User) *SessionClaims {
	perms := make([]string, 0)
	roleName := ""

	if u.Role != nil {
		roleName = u.Role.Name
		for _, p := range u.Role.Permissions {
			perms = append(perms, p.Code)
		}
	}

	return &SessionClaims{
		UserID:      u.ID.String(),
		Name:        u.Name,
		Email:       u.Email,
		Role:        roleName,
		Permissions: perms,
	}
}

// toUserResponse maps a user onto the response DTO.
func toUserResponse(u *database.User) dtos.UserResponse {
	return dtos.UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		IsActive:  u.IsActive,
		RoleID:    u.RoleID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

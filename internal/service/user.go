package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *database.Repository
}

func NewUserService(repo *database.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(name, email, password string, roleID *uuid.UUID) (*dtos.UserResponse, error) {
	if name == "" {
		return nil, errors.New("user name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" || len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Check if email already exists
	existing, _ := s.repo.GetUserByEmail(email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &database.User{
		Name:     name,
		Email:    email,
		Password: string(hashed),
		RoleID:   roleID,
		IsActive: true,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &dtos.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		IsActive:  user.IsActive,
		RoleID:    user.RoleID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *UserService) GetByID(id uuid.UUID) (*database.User, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *UserService) List(offset, limit int, search string) ([]dtos.UserDto, int64, error) {
	return s.repo.ListUsers(offset, limit, search)
}

func (s *UserService) Update(user *database.User) error {
	if user.Name == "" {
		return errors.New("user name is required")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	return s.repo.UpdateUser(user)
}

func (s *UserService) Delete(id uuid.UUID) error {
	return s.repo.DeleteUser(id)
}

func (s *UserService) ActivateUser(id uuid.UUID) error {
	return s.repo.SetUserActiveStatus(id, true)
}

func (s *UserService) DeactivateUser(id uuid.UUID) error {
	return s.repo.SetUserActiveStatus(id, false)
}

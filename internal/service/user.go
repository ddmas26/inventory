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

func (s *UserService) CreateUser(companyID uuid.UUID, name, email, phone, password string, roleID *uuid.UUID) (*dtos.UserResponse, error) {
	if name == "" {
		return nil, errors.New("user name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if phone == "" {
		return nil, errors.New("phone is required")
	}
	if password == "" || len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Check if email already exists (emails are globally unique)
	existing, _ := s.repo.GetUserByEmail(email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// A role can only be assigned from within the caller's company.
	if roleID != nil {
		if _, err := s.repo.GetRoleByID(companyID, *roleID); err != nil {
			return nil, errors.New("role not found")
		}
	}

	// Hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &database.User{
		CompanyID: companyID,
		Name:      name,
		Email:     email,
		Phone:     phone,
		Password:  string(hashed),
		RoleID:    roleID,
		IsActive:  true,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &dtos.UserResponse{
		ID:        user.ID,
		CompanyID: user.CompanyID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		IsRoot:    user.IsRoot,
		IsActive:  user.IsActive,
		RoleID:    user.RoleID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *UserService) GetByID(companyID, id uuid.UUID) (*database.User, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.CompanyID != companyID {
		return nil, fmt.Errorf("get user: %w", database.ErrNotFound)
	}
	return user, nil
}

func (s *UserService) List(companyID uuid.UUID, offset, limit int, search string) ([]dtos.UserDto, int64, error) {
	return s.repo.ListUsers(companyID, offset, limit, search)
}

func (s *UserService) Update(companyID uuid.UUID, user *database.User) error {
	if user.Name == "" {
		return errors.New("user name is required")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	if user.Phone == "" {
		return errors.New("phone is required")
	}
	if user.RoleID != nil {
		if _, err := s.repo.GetRoleByID(companyID, *user.RoleID); err != nil {
			return errors.New("role not found")
		}
	}
	return s.repo.UpdateUser(companyID, user)
}

func (s *UserService) Delete(companyID, id uuid.UUID) error {
	return s.repo.DeleteUser(companyID, id)
}

func (s *UserService) ActivateUser(companyID, id uuid.UUID) error {
	return s.repo.SetUserActiveStatus(companyID, id, true)
}

func (s *UserService) DeactivateUser(companyID, id uuid.UUID) error {
	return s.repo.SetUserActiveStatus(companyID, id, false)
}

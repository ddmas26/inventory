package service

import (
	"errors"
	"fmt"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

type RoleService struct {
	repo *database.Repository
}

func NewRoleService(repo *database.Repository) *RoleService {
	return &RoleService{repo: repo}
}

// ── Role CRUD ──────────────────────────────────────────────────────────────

func (s *RoleService) CreateRole(companyID uuid.UUID, name, description string) (*dtos.RoleResponse, error) {
	if name == "" {
		return nil, errors.New("role name is required")
	}

	role := &database.Role{
		CompanyID:   companyID,
		Name:        name,
		Description: description,
	}

	if err := s.repo.CreateRole(role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	return &dtos.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}, nil
}

func (s *RoleService) GetByID(companyID, id uuid.UUID) (*database.Role, error) {
	role, err := s.repo.GetRoleByID(companyID, id)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return role, nil
}

func (s *RoleService) List(companyID uuid.UUID) ([]database.Role, error) {
	return s.repo.ListRoles(companyID)
}

func (s *RoleService) Update(companyID uuid.UUID, role *database.Role) error {
	if role.Name == "" {
		return errors.New("role name is required")
	}
	return s.repo.UpdateRole(companyID, role)
}

func (s *RoleService) Delete(companyID, id uuid.UUID) error {
	return s.repo.DeleteRole(companyID, id)
}

// ── Permission Management ──────────────────────────────────────────────────

func (s *RoleService) CreatePermission(code, name, description string) (*dtos.PermissionResponse, error) {
	if code == "" {
		return nil, errors.New("permission code is required")
	}
	if name == "" {
		return nil, errors.New("permission name is required")
	}

	perm := &database.Permission{
		Code:        code,
		Name:        name,
		Description: description,
	}

	if err := s.repo.CreatePermission(perm); err != nil {
		return nil, fmt.Errorf("create permission: %w", err)
	}

	return &dtos.PermissionResponse{
		ID:          perm.ID,
		Code:        perm.Code,
		Name:        perm.Name,
		Description: perm.Description,
		CreatedAt:   perm.CreatedAt,
		UpdatedAt:   perm.UpdatedAt,
	}, nil
}

func (s *RoleService) ListPermissions() ([]database.Permission, error) {
	return s.repo.ListPermissions()
}

func (s *RoleService) UpdatePermission(perm *database.Permission) error {
	if perm.Code == "" {
		return errors.New("permission code is required")
	}
	if perm.Name == "" {
		return errors.New("permission name is required")
	}
	return s.repo.UpdatePermission(perm)
}

func (s *RoleService) DeletePermission(id uuid.UUID) error {
	return s.repo.DeletePermission(id)
}

// ── Role-Permission Association ────────────────────────────────────────────

func (s *RoleService) AddPermissionToRole(companyID, roleID, permissionID uuid.UUID) (*database.Role, error) {
	if err := s.repo.AddPermissionToRole(companyID, roleID, permissionID); err != nil {
		return nil, fmt.Errorf("add permission to role: %w", err)
	}
	return s.repo.GetRoleByID(companyID, roleID)
}

func (s *RoleService) RemovePermissionFromRole(companyID, roleID, permissionID uuid.UUID) (*database.Role, error) {
	if err := s.repo.RemovePermissionFromRole(companyID, roleID, permissionID); err != nil {
		return nil, fmt.Errorf("remove permission from role: %w", err)
	}
	return s.repo.GetRoleByID(companyID, roleID)
}

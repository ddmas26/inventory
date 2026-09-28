package database

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// defaultPermissions defines all available permission codes across the system.
var defaultPermissions = []struct {
	Code        string
	Name        string
	Description string
}{
	// Products
	{Code: "products.create", Name: "Create Products", Description: "Can create new products"},
	{Code: "products.read", Name: "Read Products", Description: "Can view products"},
	{Code: "products.edit", Name: "Edit Products", Description: "Can edit products"},
	{Code: "products.delete", Name: "Delete Products", Description: "Can delete products"},

	// Inventories
	{Code: "inventories.create", Name: "Create Inventories", Description: "Can create new inventory locations"},
	{Code: "inventories.read", Name: "Read Inventories", Description: "Can view inventory locations"},
	{Code: "inventories.edit", Name: "Edit Inventories", Description: "Can edit inventory locations"},
	{Code: "inventories.delete", Name: "Delete Inventories", Description: "Can delete inventory locations"},

	// Stock
	{Code: "stock.create", Name: "Add Stock", Description: "Can add stock to inventories"},
	{Code: "stock.read", Name: "Read Stock", Description: "Can view stock levels"},
	{Code: "stock.edit", Name: "Edit Stock", Description: "Can set/override stock quantities"},
	{Code: "stock.delete", Name: "Remove Stock", Description: "Can remove stock entries"},

	// Users
	{Code: "users.create", Name: "Create Users", Description: "Can create new users"},
	{Code: "users.read", Name: "Read Users", Description: "Can view users"},
	{Code: "users.edit", Name: "Edit Users", Description: "Can edit users"},
	{Code: "users.delete", Name: "Delete Users", Description: "Can delete users"},

	// Roles
	{Code: "roles.create", Name: "Create Roles", Description: "Can create new roles"},
	{Code: "roles.read", Name: "Read Roles", Description: "Can view roles"},
	{Code: "roles.edit", Name: "Edit Roles", Description: "Can edit roles"},
	{Code: "roles.delete", Name: "Delete Roles", Description: "Can delete roles"},

	// Permissions
	{Code: "permissions.create", Name: "Create Permissions", Description: "Can create new permissions"},
	{Code: "permissions.read", Name: "Read Permissions", Description: "Can view permissions"},
	{Code: "permissions.edit", Name: "Edit Permissions", Description: "Can edit permissions"},
	{Code: "permissions.delete", Name: "Delete Permissions", Description: "Can delete permissions"},

	// Dashboard
	{Code: "dashboard.read", Name: "View Dashboard", Description: "Can view the dashboard"},
}

// getAdminCredentials returns the admin email and password from environment or defaults.
func getAdminCredentials() (email, password string) {
	email = os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@inventory.com"
	}
	password = os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
	}
	return
}

// Seed creates the default Super Admin role with all permissions
// and an admin user if none exists yet.
func Seed(db *gorm.DB) error {
	log.Println("Seeding database...")

	repo := NewRepository(db)

	// ── 1. Create all permissions if they don't exist ─────────
	permMap := make(map[string]*Permission)
	for _, p := range defaultPermissions {
		existing, err := repo.GetPermissionByCode(p.Code)
		if err != nil {
			// Not found — create it
			perm := &Permission{
				Code:        p.Code,
				Name:        p.Name,
				Description: p.Description,
			}
			if err := repo.CreatePermission(perm); err != nil {
				return fmt.Errorf("seed: create permission %s: %w", p.Code, err)
			}
			permMap[p.Code] = perm
			log.Printf("  ✓ Created permission: %s", p.Code)
		} else {
			permMap[p.Code] = existing
		}
	}

	// ── 2. Create "Super Admin" role with all permissions ────
	superAdminRole, err := repo.GetRoleByName("Super Admin")
	if err != nil {
		role := &Role{
			Name:        "Super Admin",
			Description: "Full system access with all permissions",
		}
		if err := repo.CreateRole(role); err != nil {
			return fmt.Errorf("seed: create super admin role: %w", err)
		}
		superAdminRole = role
		log.Println("  ✓ Created role: Super Admin")

		// Attach all permissions to the new role
		for _, p := range permMap {
			if err := repo.AddPermissionToRole(superAdminRole.ID, p.ID); err != nil {
				return fmt.Errorf("seed: add permission %s to super admin: %w", p.Code, err)
			}
		}
		log.Printf("  ✓ Attached %d permissions to Super Admin role", len(permMap))
	} else {
		// Sync any missing permissions to the existing Super Admin role
		for _, p := range permMap {
			alreadyHas := false
			for _, rp := range superAdminRole.Permissions {
				if rp.ID == p.ID {
					alreadyHas = true
					break
				}
			}
			if !alreadyHas {
				if err := repo.AddPermissionToRole(superAdminRole.ID, p.ID); err != nil {
					return fmt.Errorf("seed: add permission %s to super admin: %w", p.Code, err)
				}
				log.Printf("  ✓ Synced permission to Super Admin: %s", p.Code)
			}
		}
	}

	// ── 3. Create admin user if not exists ───────────────────
	adminEmail, adminPassword := getAdminCredentials()
	_, err = repo.GetUserByEmail(adminEmail)
	if err != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("seed: hash admin password: %w", err)
		}

		adminUser := &User{
			Name:     "Admin",
			Email:    adminEmail,
			Password: string(hashed),
			RoleID:   &superAdminRole.ID,
			IsActive: true,
		}
		if err := repo.CreateUser(adminUser); err != nil {
			return fmt.Errorf("seed: create admin user: %w", err)
		}
		log.Printf("  ✓ Created admin user: %s / %s", adminEmail, adminPassword)
	} else {
		log.Println("  - Admin user already exists, skipping")
	}

	log.Println("Seeding complete.")
	return nil
}

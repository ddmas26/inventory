package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/ddmas26/inventory/internal/auth"
	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/handler"
	"github.com/ddmas26/inventory/internal/service"
	"github.com/ddmas26/inventory/internal/storage"
)

func main() {
	// Load configuration from environment
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Run auto-migration (creates/updates tables)
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Seed default data (admin user, Super Admin role, all permissions)
	if err := database.Seed(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	// Enable SQL query logging for runtime (not needed during migration/seed)
	database.EnableSQLLogging(db)

	// Initialize Redis token store (access sessions + refresh tokens)
	session := auth.NewSessionStore(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL)
	defer session.Close()
	log.Printf("Token store: access TTL %s, refresh TTL %s", cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL)

	// Ensure the directory used for product images exists
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("Failed to create upload directory %q: %v", cfg.UploadDir, err)
	}

	// Initialize the S3-compatible object store for product images
	// (MinIO locally, AWS S3 in production — same code, different env vars).
	store, err := storage.New(context.Background(), cfg.Storage)
	if err != nil {
		log.Fatalf("Storage initialisation failed: %v", err)
	}

	// Initialize repository, services, and handler
	repo := database.NewRepository(db)
	stockSvc := service.NewStockService(repo, store)
	productSvc := service.NewProductService(repo, store)
	inventorySvc := service.NewInventoryService(repo, store)
	userSvc := service.NewUserService(repo)
	authSvc := auth.NewAuthService(repo, session)
	platformAuthSvc := auth.NewPlatformAuthService(repo, session)
	roleSvc := service.NewRoleService(repo)
	h := handler.NewHandler(repo, stockSvc, productSvc, inventorySvc, userSvc, authSvc, platformAuthSvc, roleSvc, session, store, cfg.UploadDir)

	// Set up routes
	mux := http.NewServeMux()

	// ── Uploads ──────────────────────────────────────────────
	mux.HandleFunc("POST /api/uploads/image", h.UploadImage)
	// Serve previously uploaded files (stored on disk).
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	// ── Products ──────────────────────────────────────────────
	mux.HandleFunc("POST /api/products", h.CreateProduct)
	mux.HandleFunc("GET /api/products", h.ListProducts)
	mux.HandleFunc("GET /api/products/{id}", h.GetProduct)
	mux.HandleFunc("PUT /api/products/{id}", h.UpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", h.DeleteProduct)

	// ── Inventories ───────────────────────────────────────────
	mux.HandleFunc("POST /api/inventories", h.CreateInventory)
	mux.HandleFunc("GET /api/inventories", h.ListInventories)
	mux.HandleFunc("GET /api/inventories/{id}", h.GetInventory)
	mux.HandleFunc("PUT /api/inventories/{id}", h.UpdateInventory)
	mux.HandleFunc("DELETE /api/inventories/{id}", h.DeleteInventory)
	mux.HandleFunc("GET /api/inventories/dashboard", h.DashboardInventory)

	// ── Stock Operations ──────────────────────────────────────
	mux.HandleFunc("POST /api/stock/add", h.AddStock)
	mux.HandleFunc("POST /api/stock/deduct", h.DeductStock)
	mux.HandleFunc("POST /api/stock/set", h.SetStock)
	mux.HandleFunc("GET /api/stock", h.ListStock)
	mux.HandleFunc("POST /api/stock/remove", h.RemoveStock)
	mux.HandleFunc("POST /api/stock/transfer", h.TransferStock)

	// ── Users ────────────────────────────────────────────────
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("GET /api/users", h.ListUsers)
	mux.HandleFunc("GET /api/users/{id}", h.GetUser)
	mux.HandleFunc("PUT /api/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/users/{id}", h.DeleteUser)
	mux.HandleFunc("PATCH /api/users/{id}/activate", h.ActivateUser)
	mux.HandleFunc("PATCH /api/users/{id}/deactivate", h.DeactivateUser)

	// ── Auth ──────────────────────────────────────────────────
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/me", h.Me)

	// ── Platform (operator) auth ──────────────────────────────
	mux.HandleFunc("POST /api/platform/auth/login", h.PlatformLogin)
	mux.HandleFunc("POST /api/platform/auth/logout", h.PlatformLogout)
	mux.HandleFunc("GET /api/platform/auth/me", h.PlatformMe)

	// ── Platform (company administration) ─────────────────────
	mux.HandleFunc("GET /api/platform/dashboard", h.PlatformDashboard)
	mux.HandleFunc("GET /api/platform/companies", h.ListCompanies)
	mux.HandleFunc("GET /api/platform/companies/{id}", h.GetCompany)
	mux.HandleFunc("PATCH /api/platform/companies/{id}/approve", h.ApproveCompany)
	mux.HandleFunc("PATCH /api/platform/companies/{id}/reject", h.RejectCompany)
	mux.HandleFunc("PATCH /api/platform/companies/{id}/suspend", h.SuspendCompany)

	// ── Roles ────────────────────────────────────────────────
	mux.HandleFunc("POST /api/roles", h.CreateRole)
	mux.HandleFunc("GET /api/roles", h.ListRoles)
	mux.HandleFunc("GET /api/roles/{id}", h.GetRole)
	mux.HandleFunc("PUT /api/roles/{id}", h.UpdateRole)
	mux.HandleFunc("DELETE /api/roles/{id}", h.DeleteRole)
	mux.HandleFunc("POST /api/roles/{id}/permissions", h.AddPermissionToRole)
	mux.HandleFunc("DELETE /api/roles/{id}/permissions/{permissionId}", h.RemovePermissionFromRole)

	// ── Permissions ──────────────────────────────────────────
	mux.HandleFunc("POST /api/permissions", h.CreatePermission)
	mux.HandleFunc("GET /api/permissions", h.ListPermissions)
	mux.HandleFunc("PUT /api/permissions/{id}", h.UpdatePermission)
	mux.HandleFunc("DELETE /api/permissions/{id}", h.DeletePermission)

	// ── Start Server ──────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "9080"
	}

	log.Printf("Inventory API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

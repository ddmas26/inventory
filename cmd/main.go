package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/handler"
	"github.com/ddmas26/inventory/internal/service"
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

	// Initialize repository, services, and handler
	repo := database.NewRepository(db)
	stockSvc := service.NewStockService(repo)
	productSvc := service.NewProductService(repo)
	inventorySvc := service.NewInventoryService(repo)
	h := handler.NewHandler(repo, stockSvc, productSvc, inventorySvc)

	// Set up routes
	mux := http.NewServeMux()

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

	// ── Dashboard ──────────────────────────────────────

	// ── Start Server ──────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Inventory API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

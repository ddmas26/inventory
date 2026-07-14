package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/handler"
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

	// Initialize repository and handler
	repo := database.NewRepository(db)
	h := handler.NewHandler(repo)

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

	// ── Stock Operations ──────────────────────────────────────
	mux.HandleFunc("POST /api/inventories/{invID}/products/{prodID}/stock/add", h.AddStock)
	mux.HandleFunc("POST /api/inventories/{invID}/products/{prodID}/stock/deduct", h.DeductStock)
	mux.HandleFunc("POST /api/inventories/{invID}/products/{prodID}/stock", h.SetStock)
	mux.HandleFunc("GET /api/inventories/{invID}/products/{prodID}/stock", h.GetStock)
	mux.HandleFunc("DELETE /api/inventories/{invID}/products/{prodID}", h.RemoveProductFromInventory)
	mux.HandleFunc("GET /api/inventories/{invID}/products", h.ListProductsAtInventory)
	mux.HandleFunc("GET /api/products/{prodID}/inventories", h.ListInventoriesForProduct)
	mux.HandleFunc("POST /api/stock/transfer", h.TransferStock)

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

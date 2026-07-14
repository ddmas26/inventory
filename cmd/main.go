package main

import (
	"log"

	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
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

	// Initialize repository
	repo := database.NewRepository(db)

	// --- Demo: create a product ---
	product := &database.Product{
		Name:        "Laptop",
		Description: "High-performance laptop",
		Price:       1299.99,
	}
	if err := repo.CreateProduct(product); err != nil {
		log.Printf("Create product (may already exist): %v", err)
	}
	log.Printf("Created product: %s (%s)\n", product.Name, product.ID)

	// --- Demo: create an inventory location ---
	inventory := &database.Inventory{
		Name:      "Main Warehouse",
		Address:   "123 Storage Blvd",
		Latitude:  "40.7128",
		Longitude: "-74.0060",
	}
	if err := repo.CreateInventory(inventory); err != nil {
		log.Printf("Create inventory (may already exist): %v", err)
	}
	log.Printf("Created inventory: %s (%s)\n", inventory.Name, inventory.ID)

	// --- Demo: add 100 units of laptop to the warehouse ---
	if err := repo.AddProductToInventory(inventory.ID, product.ID, 100); err != nil {
		log.Printf("Add stock: %v", err)
	}
	log.Printf("Added 100 units of %q to %q\n", product.Name, inventory.Name)

	// --- Demo: deduct 5 units ---
	if err := repo.DeductStock(inventory.ID, product.ID, 5); err != nil {
		log.Printf("Deduct stock: %v", err)
	}

	// --- Demo: check current stock ---
	stock, err := repo.GetStock(inventory.ID, product.ID)
	if err != nil {
		log.Printf("Get stock: %v", err)
	} else {
		log.Printf("Current stock: %d units of %q at %q\n", stock.Quantity, product.Name, inventory.Name)
	}

	log.Println("Inventory system is ready!")
}

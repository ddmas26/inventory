package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/service"
)

// Handler holds services and provides common HTTP helpers.
type Handler struct {
	Repo     *database.Repository
	StockSvc *service.StockService
}

func NewHandler(repo *database.Repository, stockSvc *service.StockService) *Handler {
	return &Handler{Repo: repo, StockSvc: stockSvc}
}

// respond writes a JSON response with the given status code.
func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// decode reads and decodes JSON from the request body into dst.
func decode(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ddmas26/inventory/internal/database"
)

// Handler holds a reference to the repository and provides common helpers.
type Handler struct {
	Repo *database.Repository
}

func NewHandler(repo *database.Repository) *Handler {
	return &Handler{Repo: repo}
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

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ddmas26/inventory/internal/auth"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/service"
	"github.com/ddmas26/inventory/internal/storage"
	"github.com/google/uuid"
)

// Handler holds services and provides common HTTP helpers.
type Handler struct {
	Repo         *database.Repository
	StockSvc     *service.StockService
	ProductSvc   *service.ProductService
	InventorySvc *service.InventoryService
	UserSvc      *service.UserService
	AuthSvc      *auth.AuthService
	RoleSvc      *service.RoleService
	Session      *auth.SessionStore
	Storage      *storage.Store
	UploadDir    string
}

func NewHandler(repo *database.Repository, stockSvc *service.StockService, productSvc *service.ProductService, inventorySvc *service.InventoryService, userSvc *service.UserService, authSvc *auth.AuthService, roleSvc *service.RoleService, session *auth.SessionStore, store *storage.Store, uploadDir string) *Handler {
	return &Handler{
		Repo:         repo,
		StockSvc:     stockSvc,
		ProductSvc:   productSvc,
		InventorySvc: inventorySvc,
		UserSvc:      userSvc,
		AuthSvc:      authSvc,
		RoleSvc:      roleSvc,
		Session:      session,
		Storage:      store,
		UploadDir:    uploadDir,
	}
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

// bearerToken returns the token from the Authorization header, or "" when the
// header is missing or malformed.
func bearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

// sessionFromRequest extracts the session token from the Authorization header
// and returns the session claims from Redis.
func (h *Handler) sessionFromRequest(r *http.Request) (*auth.SessionClaims, error) {
	token := bearerToken(r)
	if token == "" {
		return nil, errUnauthorized("missing or invalid authorization header")
	}

	claims, err := h.Session.GetSession(context.Background(), token)
	if err != nil {
		return nil, errUnauthorized("session lookup failed")
	}
	if claims == nil {
		return nil, errUnauthorized("session expired or not found")
	}

	// Refresh TTL on each request
	_ = h.Session.RefreshSession(context.Background(), token)

	return claims, nil
}

// requirePermission checks that the authenticated user has the given permission code.
// Uses Redis session claims instead of querying the database.
func (h *Handler) requirePermission(w http.ResponseWriter, r *http.Request, permissionCode string) bool {
	claims, err := h.sessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return false
	}

	for _, p := range claims.Permissions {
		if p == permissionCode {
			return true
		}
	}

	respondError(w, http.StatusForbidden, "insufficient permissions")
	return false
}

// authorize verifies the session and that the caller holds the given permission.
// On success it returns the session claims, which carry the caller's company ID
// so every tenant-scoped query can be filtered by it.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, permissionCode string) (*auth.SessionClaims, bool) {
	claims, err := h.sessionFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return nil, false
	}

	for _, p := range claims.Permissions {
		if p == permissionCode {
			return claims, true
		}
	}

	respondError(w, http.StatusForbidden, "insufficient permissions")
	return nil, false
}

// companyID parses the caller's company UUID from session claims.
func companyID(claims *auth.SessionClaims) (uuid.UUID, error) {
	return uuid.Parse(claims.CompanyID)
}

// requireAuth checks that the request carries a valid, non-expired session.
// It writes a 401 response and returns false when authentication fails.
func (h *Handler) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if _, err := h.sessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return false
	}
	return true
}

// errUnauthorized creates a simple error for unauthorized requests.
func errUnauthorized(msg string) error {
	return &unauthorizedError{msg: msg}
}

type unauthorizedError struct{ msg string }

func (e *unauthorizedError) Error() string { return e.msg }

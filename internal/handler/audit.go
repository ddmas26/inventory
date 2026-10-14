package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ddmas26/inventory/internal/auth"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/ddmas26/inventory/internal/service"
	"github.com/google/uuid"
)

// ── Audit entry helpers ─────────────────────────────────────────────────────

// auditUserEntry prefills an audit entry for a company-user action: the actor
// (from session claims) plus the request metadata (ip, user-agent, request id).
func auditUserEntry(r *http.Request, claims *auth.SessionClaims) service.AuditEntry {
	entry := service.AuditEntry{
		ActorType: database.AuditActorUser,
		ActorName: claims.Name,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: RequestIDFromContext(r.Context()),
	}
	if cid, err := uuid.Parse(claims.CompanyID); err == nil {
		entry.CompanyID = &cid
	}
	if uid, err := uuid.Parse(claims.UserID); err == nil {
		entry.ActorID = &uid
	}
	return entry
}

// auditPlatformEntry prefills an audit entry for a platform-admin action.
func auditPlatformEntry(r *http.Request, claims *auth.PlatformSessionClaims) service.AuditEntry {
	entry := service.AuditEntry{
		ActorType: database.AuditActorPlatform,
		ActorName: claims.Name,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: RequestIDFromContext(r.Context()),
	}
	if uid, err := uuid.Parse(claims.PlatformUserID); err == nil {
		entry.ActorID = &uid
	}
	return entry
}

// audit records a company-user action in one call.
func (h *Handler) audit(r *http.Request, claims *auth.SessionClaims, action, entityType, entityID string, oldValues, newValues, metadata any) {
	entry := auditUserEntry(r, claims)
	entry.Action = action
	entry.EntityType = entityType
	entry.EntityID = entityID
	entry.OldValues = oldValues
	entry.NewValues = newValues
	entry.Metadata = metadata
	h.Audit.Record(entry)
}

// auditPlatform records a platform-operator action in one call.
func (h *Handler) auditPlatform(r *http.Request, claims *auth.PlatformSessionClaims, action, entityType, entityID string, oldValues, newValues, metadata any) {
	entry := auditPlatformEntry(r, claims)
	entry.Action = action
	entry.EntityType = entityType
	entry.EntityID = entityID
	entry.OldValues = oldValues
	entry.NewValues = newValues
	entry.Metadata = metadata
	h.Audit.Record(entry)
}

// auditMutation runs fn inside a transaction and writes entry on that SAME
// transaction, so the mutation and its audit row commit — or roll back —
// together. entry is a pointer so fn can fill in the before/after values it only
// learns once the mutation has run.
func (h *Handler) auditMutation(entry *service.AuditEntry, fn func(tx *database.Repository) error) error {
	return h.Repo.Tx(func(tx *database.Repository) error {
		if err := fn(tx); err != nil {
			return err
		}
		return h.Audit.RecordTx(tx, *entry)
	})
}

// ── Query endpoints ─────────────────────────────────────────────────────────

// ListAuditLogs handles GET /api/audit-logs — the caller's company audit trail.
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "audit.read")
	if !ok {
		return
	}
	cid, err := companyID(claims)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid company in session")
		return
	}

	filter, pageIndex, pageSize := parseAuditFilter(r)
	logs, total, err := h.Audit.List(&cid, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}
	if logs == nil {
		logs = []dtos.AuditLogDto{}
	}

	h.audit(r, claims, "audit.view", "audit_log", "", nil, nil, map[string]any{"count": len(logs)})

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       logs,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// ListPlatformAuditLogs handles GET /api/platform/audit-logs — every company's
// audit trail, optionally filtered to a single company.
func (h *Handler) ListPlatformAuditLogs(w http.ResponseWriter, r *http.Request) {
	if _, err := h.platformSessionFromRequest(r); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	filter, pageIndex, pageSize := parseAuditFilter(r)

	var companyID *uuid.UUID
	if v := r.URL.Query().Get("company_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			companyID = &id
		}
	}

	logs, total, err := h.Audit.List(companyID, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}
	if logs == nil {
		logs = []dtos.AuditLogDto{}
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"data":       logs,
		"total":      total,
		"page_index": pageIndex,
		"page_size":  pageSize,
	})
}

// parseAuditFilter reads the shared audit-log query parameters.
func parseAuditFilter(r *http.Request) (dtos.AuditLogFilter, int, int) {
	pageIndex, _ := strconv.Atoi(r.URL.Query().Get("page_index"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageIndex <= 1 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}

	filter := dtos.AuditLogFilter{
		Action:     r.URL.Query().Get("action"),
		EntityType: r.URL.Query().Get("entity_type"),
		ActorType:  r.URL.Query().Get("actor_type"),
		Search:     r.URL.Query().Get("search"),
		Offset:     (pageIndex - 1) * pageSize,
		Limit:      pageSize,
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.To = &t
		}
	}

	return filter, pageIndex, pageSize
}

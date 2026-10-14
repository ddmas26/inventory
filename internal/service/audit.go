package service

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/dtos"
	"github.com/google/uuid"
)

// AuditService writes and reads the business audit trail.
type AuditService struct {
	repo *database.Repository
}

func NewAuditService(repo *database.Repository) *AuditService {
	return &AuditService{repo: repo}
}

// AuditEntry describes a business action to record. OldValues, NewValues and
// Metadata are marshalled to JSON; nil values are omitted.
type AuditEntry struct {
	CompanyID  *uuid.UUID
	ActorType  string
	ActorID    *uuid.UUID
	ActorName  string
	Action     string
	EntityType string
	EntityID   string
	OldValues  any
	NewValues  any
	Metadata   any
	IP         string
	UserAgent  string
	RequestID  string
}

// Record writes an audit entry. It is best-effort: a write failure is logged but
// never propagated, so auditing can't break the request that triggered it. Use it
// for events that are not tied to a database mutation (e.g. login/logout).
func (s *AuditService) Record(entry AuditEntry) {
	if err := s.RecordTx(s.repo, entry); err != nil {
		slog.Error("audit: failed to record entry",
			"action", entry.Action, "entity_type", entry.EntityType, "error", err)
	}
}

// RecordTx writes an audit entry using the given (transaction-scoped) repository
// and RETURNS the error, so the caller can roll back the surrounding transaction
// when the entry cannot be written. Use this inside Repository.Tx to make a
// mutation and its audit row atomic.
func (s *AuditService) RecordTx(tx *database.Repository, entry AuditEntry) error {
	if entry.ActorType == "" {
		entry.ActorType = database.AuditActorSystem
	}

	row := &database.AuditLog{
		CompanyID:  entry.CompanyID,
		ActorType:  entry.ActorType,
		ActorID:    entry.ActorID,
		ActorName:  entry.ActorName,
		Action:     entry.Action,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		OldValues:  marshalJSON(entry.OldValues),
		NewValues:  marshalJSON(entry.NewValues),
		Metadata:   marshalJSON(entry.Metadata),
		IP:         entry.IP,
		UserAgent:  entry.UserAgent,
		RequestID:  entry.RequestID,
	}

	if err := tx.CreateAuditLog(row); err != nil {
		return fmt.Errorf("audit: record %s: %w", entry.Action, err)
	}
	return nil
}

// List returns audit entries (company-scoped when companyID is non-nil) plus the
// total number of matching rows.
func (s *AuditService) List(companyID *uuid.UUID, filter dtos.AuditLogFilter) ([]dtos.AuditLogDto, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}

	logs, total, err := s.repo.ListAuditLogs(companyID, filter)
	if err != nil {
		return nil, 0, err
	}

	out := make([]dtos.AuditLogDto, 0, len(logs))
	for i := range logs {
		out = append(out, s.toDto(&logs[i]))
	}
	return out, total, nil
}

// toDto maps a row onto the API DTO, decoding the stored JSON payloads.
func (s *AuditService) toDto(l *database.AuditLog) dtos.AuditLogDto {
	dto := dtos.AuditLogDto{
		ID:         l.ID,
		CompanyID:  l.CompanyID,
		ActorType:  l.ActorType,
		ActorID:    l.ActorID,
		ActorName:  l.ActorName,
		Action:     l.Action,
		EntityType: l.EntityType,
		EntityID:   l.EntityID,
		IP:         l.IP,
		UserAgent:  l.UserAgent,
		RequestID:  l.RequestID,
		CreatedAt:  l.CreatedAt,
	}
	if l.OldValues != "" {
		dto.OldValues = json.RawMessage(l.OldValues)
	}
	if l.NewValues != "" {
		dto.NewValues = json.RawMessage(l.NewValues)
	}
	if l.Metadata != "" {
		dto.Metadata = json.RawMessage(l.Metadata)
	}
	if l.CompanyID != nil {
		if c, err := s.repo.GetCompanyByID(*l.CompanyID); err == nil {
			dto.CompanyName = c.Name
		}
	}
	return dto
}

// marshalJSON encodes a value for storage, returning "" for nil or on error.
func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

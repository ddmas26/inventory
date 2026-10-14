package dtos

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLogDto is a single audit-trail entry returned by the API. OldValues,
// NewValues and Metadata are embedded as raw JSON so the client can render them
// structurally instead of parsing an escaped string.
type AuditLogDto struct {
	ID          uuid.UUID       `json:"id"`
	CompanyID   *uuid.UUID      `json:"company_id,omitempty"`
	CompanyName string          `json:"company_name,omitempty"`
	ActorType   string          `json:"actor_type"`
	ActorID     *uuid.UUID      `json:"actor_id,omitempty"`
	ActorName   string          `json:"actor_name"`
	Action      string          `json:"action"`
	EntityType  string          `json:"entity_type"`
	EntityID    string          `json:"entity_id"`
	OldValues   json.RawMessage `json:"old_values,omitempty"`
	NewValues   json.RawMessage `json:"new_values,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	IP          string          `json:"ip,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
	RequestID   string          `json:"request_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// AuditLogFilter narrows an audit-log query. Company scoping is passed separately.
type AuditLogFilter struct {
	Action     string
	EntityType string
	ActorType  string
	Search     string // matches actor_name, entity_id or action
	From       *time.Time
	To         *time.Time
	Offset     int
	Limit      int
}

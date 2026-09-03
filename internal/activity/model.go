package activity

import "time"

const (
	CategoryGRCWork       = "GRC_WORK"
	CategoryFormsEvidence = "FORMS_EVIDENCE"
	CategoryVendor        = "VENDOR"
	CategoryAI            = "AI"
	CategoryConfiguration = "CONFIGURATION"
	CategorySystem        = "SYSTEM"
	CategoryOther         = "OTHER"

	ActorInternalUser        = "INTERNAL_USER"
	ActorExternalParticipant = "EXTERNAL_PARTICIPANT"
	ActorService             = "SERVICE"
	ActorSystem              = "SYSTEM"
	ActorUnknown             = "UNKNOWN"

	OutcomeSucceeded = "SUCCEEDED"
)

type Event struct {
	TenantID         string    `json:"-"`
	ID               string    `json:"event_id"`
	OccurredAt       time.Time `json:"occurred_at"`
	Category         string    `json:"category"`
	EventType        string    `json:"event_type"`
	Action           string    `json:"action"`
	Outcome          string    `json:"outcome"`
	ActorKind        string    `json:"actor_kind"`
	ActorID          string    `json:"actor_id,omitempty"`
	ActorDisplayName string    `json:"actor_display_name,omitempty"`
	LegalEntityID    string    `json:"legal_entity_id,omitempty"`
	ObjectType       string    `json:"object_type"`
	ObjectID         string    `json:"object_id"`
	RequestID        string    `json:"request_id,omitempty"`
	CorrelationID    string    `json:"correlation_id,omitempty"`
	Source           string    `json:"source"`
}

type Query struct {
	TenantID      string
	Limit         int
	Cursor        string
	From          *time.Time
	To            *time.Time
	Category      string
	EventType     string
	ObjectType    string
	ObjectID      string
	ActorID       string
	ActorQuery    string
	ActorKind     string
	LegalEntityID string
}

type Page struct {
	Items      []Event   `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
	AsOf       time.Time `json:"as_of"`
}

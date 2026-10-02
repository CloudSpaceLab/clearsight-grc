package risk

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusDraft   Status = "DRAFT"
	StatusActive  Status = "ACTIVE"
	StatusRetired Status = "RETIRED"
)

type AssessmentKind string

const (
	AssessmentInherent AssessmentKind = "INHERENT"
	AssessmentCurrent  AssessmentKind = "CURRENT"
	AssessmentResidual AssessmentKind = "RESIDUAL"
	AssessmentTarget   AssessmentKind = "TARGET"
	AssessmentStressed AssessmentKind = "STRESSED"
	AssessmentAccepted AssessmentKind = "ACCEPTED"
)

type AppetitePosition string

const (
	AppetiteWithin      AppetitePosition = "WITHIN"
	AppetiteApproaching AppetitePosition = "APPROACHING"
	AppetiteBreached    AppetitePosition = "BREACHED"
	AppetiteUnknown     AppetitePosition = "UNKNOWN"
)

type AppetiteStatus string

const (
	AppetiteActive  AppetiteStatus = "ACTIVE"
	AppetiteRetired AppetiteStatus = "RETIRED"
)

type Risk struct {
	ID                string          `json:"id"`
	TenantID          string          `json:"tenant_id"`
	LegalEntityID     string          `json:"legal_entity_id"`
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	Category          string          `json:"category"`
	Statement         string          `json:"statement"`
	Cause             string          `json:"cause"`
	Event             string          `json:"event"`
	Impact            string          `json:"impact"`
	Scope             json.RawMessage `json:"scope"`
	OwnerPrincipalID  string          `json:"owner_principal_id,omitempty"`
	Status            Status          `json:"status"`
	Version           int64           `json:"version"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Assessment struct {
	ID                  string           `json:"id"`
	RiskID              string           `json:"risk_id"`
	RiskVersion         int64            `json:"risk_version"`
	Kind                AssessmentKind   `json:"kind"`
	MethodCode          string           `json:"method_code"`
	MethodVersion       string           `json:"method_version"`
	Dimensions          json.RawMessage  `json:"dimensions"`
	Assumptions         json.RawMessage  `json:"assumptions"`
	EvidenceReferences  json.RawMessage  `json:"evidence_references"`
	Confidence          *float64         `json:"confidence,omitempty"`
	AssessedBy          string           `json:"assessed_by,omitempty"`
	AppetiteStatementID string           `json:"appetite_statement_id,omitempty"`
	AppetitePosition    AppetitePosition `json:"appetite_position"`
	AppetiteRationale   string           `json:"appetite_rationale"`
	AssessedAt          time.Time        `json:"assessed_at"`
	CreatedAt           time.Time        `json:"created_at"`
}

type AppetiteStatement struct {
	ID                   string         `json:"id"`
	RiskID               string         `json:"risk_id"`
	RiskVersion          int64          `json:"risk_version"`
	Version              int64          `json:"version"`
	Statement            string         `json:"statement"`
	Rule                 json.RawMessage`json:"rule"`
	Rationale            string         `json:"rationale"`
	OwnerPrincipalID     string         `json:"owner_principal_id,omitempty"`
	AuthorityPrincipalID string         `json:"authority_principal_id,omitempty"`
	Status               AppetiteStatus `json:"status"`
	EffectiveFrom        time.Time      `json:"effective_from"`
	EffectiveUntil       *time.Time     `json:"effective_until,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
}

type Summary struct {
	Risk             Risk               `json:"risk"`
	LatestAssessment *Assessment        `json:"latest_assessment,omitempty"`
	ActiveAppetite   *AppetiteStatement `json:"active_appetite,omitempty"`
}

type Page struct {
	Items      []Summary `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type Aggregate struct {
	Risk        Risk                `json:"risk"`
	Assessments []Assessment        `json:"assessments"`
	Appetite    []AppetiteStatement `json:"appetite"`
}

type Event struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	LegalEntityID    string          `json:"legal_entity_id"`
	RiskID           string          `json:"risk_id"`
	RiskVersion      int64           `json:"risk_version"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ActorID          string          `json:"actor_id,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
}

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type ListFilter struct {
	Status           Status
	Category         string
	OwnerPrincipalID string
	Search           string
	AppetitePosition AppetitePosition
	Cursor           string
	Limit            int
	AsOf             time.Time
}

type CreateInput struct {
	TenantID         string
	LegalEntityID    string
	Code             string
	Name             string
	Category         string
	Statement        string
	Cause            string
	Event            string
	Impact           string
	Scope            json.RawMessage
	OwnerPrincipalID string
	ActorID          string
}

type UpdateInput struct {
	TenantID         string
	LegalEntityID    string
	RiskID           string
	ExpectedVersion  int64
	Name             string
	Category         string
	Statement        string
	Cause            string
	Event            string
	Impact           string
	Scope            json.RawMessage
	OwnerPrincipalID string
	Status           Status
	ActorID          string
}

type AssessmentInput struct {
	TenantID            string
	LegalEntityID       string
	RiskID              string
	ExpectedRiskVersion int64
	Kind                AssessmentKind
	MethodCode          string
	MethodVersion       string
	Dimensions          json.RawMessage
	Assumptions         json.RawMessage
	EvidenceReferences  json.RawMessage
	Confidence          *float64
	AppetiteStatementID string
	AppetitePosition    AppetitePosition
	AppetiteRationale   string
	ActorID              string
}

type AppetiteInput struct {
	TenantID            string
	LegalEntityID       string
	RiskID              string
	ExpectedRiskVersion int64
	Statement           string
	Rule                json.RawMessage
	Rationale           string
	OwnerPrincipalID    string
	ActorID              string
	EffectiveFrom       time.Time
	EffectiveUntil      *time.Time
}

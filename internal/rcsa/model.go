package rcsa

import "time"

type Status string
type TriggerKind string

const (
	StatusDraft             Status = "DRAFT"
	StatusAssessmentOpen    Status = "ASSESSMENT_OPEN"
	StatusAwaitingChallenge Status = "AWAITING_CHALLENGE"
	StatusCompleted         Status = "COMPLETED"
	StatusCancelled         Status = "CANCELLED"
)

const (
	TriggerScheduled TriggerKind = "SCHEDULED"
	TriggerChange    TriggerKind = "CHANGE"
	TriggerManual    TriggerKind = "MANUAL"
)

type Cycle struct {
	ID                          string      `json:"id"`
	TenantID                    string      `json:"tenant_id"`
	LegalEntityID               string      `json:"legal_entity_id"`
	Code                        string      `json:"code"`
	Name                        string      `json:"name"`
	TriggerKind                 TriggerKind `json:"trigger_kind"`
	FirstLineOwnerID            string      `json:"first_line_owner_principal_id"`
	Status                      Status      `json:"status"`
	PopulationChecksum          string      `json:"population_checksum"`
	FirstLineDistributionID     string      `json:"first_line_distribution_id,omitempty"`
	FirstLineResponseRevisionID string      `json:"first_line_response_revision_id,omitempty"`
	ChallengeMatterID           string      `json:"challenge_matter_id,omitempty"`
	Version                     int64       `json:"version"`
	CreatedAt                   time.Time   `json:"created_at"`
	UpdatedAt                   time.Time   `json:"updated_at"`
}

type RiskSnapshot struct {
	CycleID     string `json:"cycle_id"`
	RiskID      string `json:"risk_id"`
	RiskVersion int64  `json:"risk_version"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
}

type ControlSnapshot struct {
	CycleID                      string     `json:"cycle_id"`
	RiskID                       string     `json:"risk_id"`
	RiskVersion                  int64      `json:"risk_version"`
	RiskControlLinkID            string     `json:"risk_control_link_id"`
	CatalogLinkID                string     `json:"catalog_link_id"`
	DefinitionID                 string     `json:"definition_id"`
	DefinitionCode               string     `json:"definition_code"`
	DefinitionName               string     `json:"definition_name"`
	ProgramID                    string     `json:"program_id"`
	ImplementationID             string     `json:"implementation_id"`
	ImplementationVersion        int64      `json:"implementation_version"`
	ImplementationName           string     `json:"implementation_name"`
	ImplementationStatus         string     `json:"implementation_status"`
	ImplementationEffectiveFrom  time.Time  `json:"implementation_effective_from"`
	ImplementationEffectiveUntil *time.Time `json:"implementation_effective_until,omitempty"`
}

type Aggregate struct {
	Cycle    Cycle             `json:"cycle"`
	Risks    []RiskSnapshot    `json:"risks"`
	Controls []ControlSnapshot `json:"controls"`
}

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type CreateInput struct {
	TenantID         string      `json:"tenant_id,omitempty"`
	LegalEntityID    string      `json:"legal_entity_id,omitempty"`
	Code             string      `json:"code"`
	Name             string      `json:"name"`
	TriggerKind      TriggerKind `json:"trigger_kind"`
	RiskIDs          []string    `json:"risk_ids"`
	FirstLineOwnerID string      `json:"first_line_owner_principal_id,omitempty"`
	ActorID          string      `json:"actor_id,omitempty"`
}

type Population struct {
	Risks    []RiskSnapshot
	Controls []ControlSnapshot
}

type Event struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	LegalEntityID string    `json:"legal_entity_id"`
	CycleID       string    `json:"cycle_id"`
	CycleVersion  int64     `json:"cycle_version"`
	Type          string    `json:"type"`
	ActorID       string    `json:"actor_id,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

const (
	EventCycleCreated             = "RCSACycleCreated"
	EventFirstLineDistributionSet = "RCSAFirstLineDistributionSet"
	EventFirstLineCompleted       = "RCSAFirstLineCompleted"
)

type BindFirstLineDistributionInput struct {
	TenantID            string `json:"tenant_id,omitempty"`
	LegalEntityID       string `json:"legal_entity_id,omitempty"`
	CycleID             string `json:"cycle_id,omitempty"`
	ExpectedVersion     int64  `json:"expected_version"`
	DistributionID      string `json:"distribution_id"`
	ActorID             string `json:"actor_id,omitempty"`
}

type CompleteFirstLineInput struct {
	TenantID        string `json:"tenant_id,omitempty"`
	LegalEntityID   string `json:"legal_entity_id,omitempty"`
	CycleID         string `json:"cycle_id,omitempty"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id,omitempty"`
}

package governance

import (
	"encoding/json"
	"errors"
	"time"
)

type PolicyState string

const (
	PolicyDraft           PolicyState = "DRAFT"
	PolicyPendingApproval PolicyState = "PENDING_APPROVAL"
	PolicyActive          PolicyState = "ACTIVE"
	PolicyRetired         PolicyState = "RETIRED"
)

type DelegationState string

const (
	DelegationDraft           DelegationState = "DRAFT"
	DelegationPendingApproval DelegationState = "PENDING_APPROVAL"
	DelegationApproved        DelegationState = "APPROVED"
	DelegationActive          DelegationState = "ACTIVE"
	DelegationRevoked         DelegationState = "REVOKED"
	DelegationExpired         DelegationState = "EXPIRED"
)

var (
	ErrNotFound          = errors.New("governance object not found")
	ErrVersionConflict   = errors.New("governance object version conflict")
	ErrInvalidTransition = errors.New("invalid governance transition")
	ErrMakerChecker      = errors.New("maker and checker must be different principals")
	ErrConflict          = errors.New("segregation or delegation conflict")
	ErrRevisionStale     = errors.New("routing policy revision is stale")
)

type RoutingPolicy struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	LegalEntityID  string          `json:"legal_entity_id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Status         PolicyState     `json:"status"`
	CurrentVersion int             `json:"current_version"`
	Definition     json.RawMessage `json:"definition"`
	Checksum       string          `json:"checksum"`
	MakerID        string          `json:"maker_id"`
	CheckerID      string          `json:"checker_id,omitempty"`
	EffectiveFrom  *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time      `json:"effective_until,omitempty"`
	SubmittedAt    *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt     *time.Time      `json:"approved_at,omitempty"`
	RetiredAt      *time.Time      `json:"retired_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Version        int64           `json:"version"`
}

type RoutingPolicyRevision struct {
	PolicyID       string          `json:"policy_id"`
	TenantID       string          `json:"tenant_id"`
	LegalEntityID  string          `json:"legal_entity_id"`
	Version        int             `json:"version"`
	BaseVersion    int             `json:"base_version"`
	Definition     json.RawMessage `json:"definition"`
	Checksum       string          `json:"checksum"`
	MakerID        string          `json:"maker_id"`
	CreatedAt      time.Time       `json:"created_at"`
	ApprovedBy     string          `json:"approved_by,omitempty"`
	ApprovedAt     *time.Time      `json:"approved_at,omitempty"`
	EffectiveFrom  *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time      `json:"effective_until,omitempty"`
}

type EscalationGuardRevisionInput struct {
	TenantID              string   `json:"tenant_id"`
	LegalEntityID         string   `json:"legal_entity_id"`
	PolicyID              string   `json:"policy_id"`
	SequenceID            string   `json:"sequence_id"`
	StepIndex             int      `json:"step_index"`
	SourceRoles           []string `json:"source_roles"`
	TargetRoles           []string `json:"target_roles"`
	TargetGroupIDs        []string `json:"target_group_ids"`
	ActorID               string   `json:"actor_id"`
	ExpectedPolicyVersion int64    `json:"expected_policy_version"`
}

type ApprovePolicyRevisionInput struct {
	TenantID              string `json:"tenant_id"`
	LegalEntityID         string `json:"legal_entity_id"`
	PolicyID              string `json:"policy_id"`
	RevisionVersion       int    `json:"revision_version"`
	ActorID               string `json:"actor_id"`
	ExpectedPolicyVersion int64  `json:"expected_policy_version"`
	Rationale             string `json:"rationale"`
}

type Delegation struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	LegalEntityID   string          `json:"legal_entity_id"`
	FromPrincipalID string          `json:"from_principal_id"`
	ToPrincipalID   string          `json:"to_principal_id"`
	Responsibility  string          `json:"responsibility"`
	Scope           json.RawMessage `json:"scope"`
	StartsAt        time.Time       `json:"starts_at"`
	EndsAt          time.Time       `json:"ends_at"`
	Status          DelegationState `json:"status"`
	Reason          string          `json:"reason"`
	MakerID         string          `json:"maker_id"`
	ApproverID      string          `json:"approver_id,omitempty"`
	SubmittedAt     *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty"`
	RevokedAt       *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Version         int64           `json:"version"`
}

type ConflictFinding struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

type CreatePolicyInput struct {
	TenantID      string          `json:"tenant_id"`
	LegalEntityID string          `json:"legal_entity_id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	MakerID       string          `json:"maker_id"`
	Definition    json.RawMessage `json:"definition"`
	EffectiveFrom *time.Time      `json:"effective_from,omitempty"`
}

type CreateDelegationInput struct {
	TenantID        string          `json:"tenant_id"`
	LegalEntityID   string          `json:"legal_entity_id"`
	FromPrincipalID string          `json:"from_principal_id"`
	ToPrincipalID   string          `json:"to_principal_id"`
	Responsibility  string          `json:"responsibility"`
	Scope           json.RawMessage `json:"scope"`
	StartsAt        time.Time       `json:"starts_at"`
	EndsAt          time.Time       `json:"ends_at"`
	Reason          string          `json:"reason"`
	MakerID         string          `json:"maker_id"`
}

type TransitionInput struct {
	TenantID        string `json:"tenant_id"`
	LegalEntityID   string `json:"legal_entity_id"`
	ID              string `json:"id,omitempty"`
	ActorID         string `json:"actor_id"`
	ExpectedVersion int64  `json:"expected_version"`
	Rationale       string `json:"rationale"`
}

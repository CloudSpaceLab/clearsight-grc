package formpolicy

import (
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const ActionClassCreateMatter = "FORM_RESPONSE_CREATE_MATTER"

var (
	ErrNotFound             = errors.New("form response policy not found")
	ErrConflict             = errors.New("form response policy version conflict")
	ErrInvalid              = errors.New("form response policy invalid")
	ErrInvalidTransition    = errors.New("form response policy transition invalid")
	ErrMakerChecker         = errors.New("form response policy maker-checker violation")
	ErrPreviewRequired      = errors.New("form response policy simulation required")
	ErrPreviewStale         = errors.New("form response policy simulation is stale")
	ErrFormInactive         = errors.New("form response policy form revision is not active")
	ErrShadowRequired       = errors.New("form response policy requires shadow history")
	ErrAuthorityUnavailable = errors.New("form response policy authority is unavailable")
	ErrActivationAuthority  = errors.New("form response policy activation authority is invalid")
)

type PolicyStatus string

const (
	PolicyDraft           PolicyStatus = "DRAFT"
	PolicyPendingApproval PolicyStatus = "PENDING_APPROVAL"
	PolicyApproved        PolicyStatus = "APPROVED"
	PolicyActive          PolicyStatus = "ACTIVE"
	PolicySuspended       PolicyStatus = "SUSPENDED"
	PolicyRetired         PolicyStatus = "RETIRED"
)

type RolloutMode string

const (
	RolloutShadow  RolloutMode = "SHADOW"
	RolloutEnforce RolloutMode = "ENFORCE"
)

type Actor struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	PrincipalID   string `json:"principal_id"`
}

type Eligibility struct {
	FormTemplateID      string                     `json:"form_template_id"`
	FormTemplateVersion int64                      `json:"form_template_version"`
	SubjectTypes        []string                   `json:"subject_types"`
	CurrentOnly         bool                       `json:"current_only"`
	MinimumCoverage     float64                    `json:"minimum_coverage"`
	Bands               []formcontract.ConcernBand `json:"bands,omitempty"`
	RawBelow            *float64                   `json:"raw_below,omitempty"`
	RawAbove            *float64                   `json:"raw_above,omitempty"`
	AdverseAtLeast      *float64                   `json:"adverse_at_least,omitempty"`
}

type MatterAction struct {
	Type              string `json:"type"`
	Priority          int    `json:"priority"`
	TitleTemplate     string `json:"title_template"`
	SummaryTemplate   string `json:"summary_template"`
	RequestedHandling string `json:"requested_handling"`
}

type BlastRadius struct {
	PerRun int `json:"per_run"`
	PerDay int `json:"per_day"`
}

type OutcomeContract struct {
	ExpectedOutcome   string `json:"expected_outcome"`
	CheckAfterMinutes int    `json:"check_after_minutes"`
	FailureResponse   string `json:"failure_response"`
}

type Policy struct {
	ID                      string          `json:"id"`
	TenantID                string          `json:"tenant_id"`
	LegalEntityID           string          `json:"legal_entity_id"`
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	Purpose                 string          `json:"purpose"`
	ActionClass             string          `json:"action_class"`
	AutomationPolicyID      string          `json:"automation_policy_id"`
	AutomationPolicyVersion int64           `json:"automation_policy_version"`
	Eligibility             Eligibility     `json:"eligibility"`
	Action                  MatterAction    `json:"action"`
	BlastRadius             BlastRadius     `json:"blast_radius"`
	Outcome                 OutcomeContract `json:"outcome_contract"`
	Rollout                 RolloutMode     `json:"rollout"`
	Status                  PolicyStatus    `json:"status"`
	MakerID                 string          `json:"maker_id"`
	CheckerID               string          `json:"checker_id,omitempty"`
	Checksum                string          `json:"checksum"`
	ApprovedSimulationID    string          `json:"approved_simulation_id,omitempty"`
	SupersedesPolicyID      string          `json:"supersedes_policy_id,omitempty"`
	RollbackOfPolicyID      string          `json:"rollback_of_policy_id,omitempty"`
	EffectiveFrom           *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil          *time.Time      `json:"effective_until,omitempty"`
	SubmittedAt             *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt              *time.Time      `json:"approved_at,omitempty"`
	ActivatedAt             *time.Time      `json:"activated_at,omitempty"`
	SuspendedAt             *time.Time      `json:"suspended_at,omitempty"`
	RetiredAt               *time.Time      `json:"retired_at,omitempty"`
	Version                 int64           `json:"version"`
	RecordVersion           int64           `json:"record_version"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	LastActorID             string          `json:"-"`
}

type SimulationReceipt struct {
	ID                      string    `json:"id"`
	TenantID                string    `json:"tenant_id"`
	LegalEntityID           string    `json:"legal_entity_id"`
	PolicyID                string    `json:"policy_id"`
	PolicyVersion           int64     `json:"policy_version"`
	PolicyChecksum          string    `json:"policy_checksum"`
	ActorID                 string    `json:"actor_id"`
	PopulationCount         int       `json:"population_count"`
	EligibleCount           int       `json:"eligible_count"`
	WouldCreateCount        int       `json:"would_create_count"`
	WouldReuseCount         int       `json:"would_reuse_count"`
	BlastSuppressedCount    int       `json:"blast_suppressed_count"`
	RestrictedExcludedCount int       `json:"restricted_excluded_count"`
	PopulationHighWater     string    `json:"population_high_water"`
	PopulationChecksum      string    `json:"population_checksum"`
	ImpactChecksum          string    `json:"impact_checksum"`
	ObservedAt              time.Time `json:"observed_at"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type ExecutionState string

const (
	ExecutionNotMatched      ExecutionState = "NOT_MATCHED"
	ExecutionShadow          ExecutionState = "SHADOW"
	ExecutionApplied         ExecutionState = "APPLIED"
	ExecutionReused          ExecutionState = "REUSED"
	ExecutionBlastSuppressed ExecutionState = "BLAST_SUPPRESSED"
	ExecutionFailed          ExecutionState = "FAILED"
)

type ExecutionReceipt struct {
	ID                      string         `json:"id"`
	TenantID                string         `json:"tenant_id"`
	LegalEntityID           string         `json:"legal_entity_id"`
	PolicyID                string         `json:"policy_id"`
	PolicyVersion           int64          `json:"policy_version"`
	AutomationPolicyID      string         `json:"automation_policy_id"`
	AutomationPolicyVersion int64          `json:"automation_policy_version"`
	ResponseRevisionID      string         `json:"response_revision_id"`
	State                   ExecutionState `json:"state"`
	MatterID                string         `json:"matter_id,omitempty"`
	ReasonCode              string         `json:"reason_code,omitempty"`
	CreatedMatter           bool           `json:"created_matter"`
	CreatedAt               time.Time      `json:"created_at"`
}

type EpisodeState string

const (
	EpisodeOpen   EpisodeState = "OPEN"
	EpisodeClosed EpisodeState = "CLOSED"
)

type AdverseEpisode struct {
	ID                     string       `json:"id"`
	TenantID               string       `json:"tenant_id"`
	LegalEntityID          string       `json:"legal_entity_id"`
	PolicyCode             string       `json:"policy_code"`
	PolicyID               string       `json:"policy_id"`
	PolicyVersion          int64        `json:"policy_version"`
	SubjectType            string       `json:"subject_type"`
	SubjectID              string       `json:"subject_id"`
	State                  EpisodeState `json:"state"`
	MatterID               string       `json:"matter_id,omitempty"`
	LastResponseRevisionID string       `json:"last_response_revision_id"`
	OpenedAt               time.Time    `json:"opened_at"`
	ClosedAt               *time.Time   `json:"closed_at,omitempty"`
	UpdatedAt              time.Time    `json:"updated_at"`
	RecordVersion          int64        `json:"record_version"`
}

type CreateInput struct {
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	Purpose                 string          `json:"purpose"`
	AutomationPolicyID      string          `json:"automation_policy_id"`
	AutomationPolicyVersion int64           `json:"automation_policy_version"`
	Eligibility             Eligibility     `json:"eligibility"`
	Action                  MatterAction    `json:"action"`
	BlastRadius             BlastRadius     `json:"blast_radius"`
	Outcome                 OutcomeContract `json:"outcome_contract"`
	Rollout                 RolloutMode     `json:"rollout"`
	EffectiveFrom           *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil          *time.Time      `json:"effective_until,omitempty"`
}

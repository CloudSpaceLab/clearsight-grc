package autonomy

import (
	"encoding/json"
	"time"
)

type SignalType string

const (
	SignalEvidenceExpired    SignalType = "EVIDENCE_EXPIRED"
	SignalEvidenceAging      SignalType = "EVIDENCE_AGING"
	SignalSourceDegraded     SignalType = "SOURCE_DEGRADED"
	SignalSourceRecovered    SignalType = "SOURCE_RECOVERED"
	SignalRequirementChanged SignalType = "REQUIREMENT_CHANGED"
	SignalRoutingGap         SignalType = "ROUTING_GAP"
	SignalOwnerRemoved       SignalType = "OWNER_REMOVED"
	SignalControlFailed      SignalType = "CONTROL_FAILED"
	SignalVerificationFailed SignalType = "VERIFICATION_FAILED"
	SignalContextChanged     SignalType = "CONTEXT_CHANGED"
)

type Signal struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Type        SignalType        `json:"type"`
	SubjectType string            `json:"subject_type"`
	SubjectID   string            `json:"subject_id"`
	DedupeKey   string            `json:"dedupe_key"`
	Source      string            `json:"source"`
	ObservedAt  time.Time         `json:"observed_at"`
	EffectiveAt time.Time         `json:"effective_at"`
	Payload     map[string]string `json:"payload,omitempty"`
}

type Drift struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SubjectType    string    `json:"subject_type"`
	SubjectID      string    `json:"subject_id"`
	Dimension      string    `json:"dimension"`
	Severity       int       `json:"severity"`
	State          string    `json:"state"`
	Summary        string    `json:"summary"`
	RequiredAction string    `json:"required_action"`
	SignalID       string    `json:"signal_id"`
	DetectedAt     time.Time `json:"detected_at"`
}

type ReadinessDimensions struct {
	Current        int `json:"current"`
	Aging          int `json:"aging"`
	AtRisk         int `json:"at_risk"`
	Unknown        int `json:"unknown"`
	BlockedRouting int `json:"blocked_routing"`
	PendingHuman   int `json:"pending_human"`
}

type Readiness struct {
	TenantID           string              `json:"tenant_id"`
	Status             string              `json:"status"`
	BaselineKnown      bool                `json:"baseline_known"`
	GeneratedAt        time.Time           `json:"generated_at"`
	Dimensions         ReadinessDimensions `json:"dimensions"`
	ActiveDrifts       []Drift             `json:"active_drifts"`
	RecommendedActions []string            `json:"recommended_actions"`
}

// AutomationPolicyState is the lifecycle state of one immutable policy revision.
type AutomationPolicyState string

const (
	AutomationPolicyDraft           AutomationPolicyState = "DRAFT"
	AutomationPolicyPendingApproval AutomationPolicyState = "PENDING_APPROVAL"
	AutomationPolicyApproved        AutomationPolicyState = "APPROVED"
	AutomationPolicyActive          AutomationPolicyState = "ACTIVE"
	AutomationPolicySuspended       AutomationPolicyState = "SUSPENDED"
	AutomationPolicyExpired         AutomationPolicyState = "EXPIRED"
	AutomationPolicyRetired         AutomationPolicyState = "RETIRED"
)

// AutomationPolicy is the governed runtime policy that determines whether a
// bounded action may be automated. The JSON guardrails are preserved exactly
// as approved rather than flattened into frontend-specific fields.
type AutomationPolicy struct {
	ID                   string                `json:"id"`
	TenantID             string                `json:"tenant_id"`
	Code                 string                `json:"code"`
	Name                 string                `json:"name"`
	ActionClass          string                `json:"action_class"`
	Eligibility          json.RawMessage       `json:"eligibility"`
	BlastRadiusLimit     json.RawMessage       `json:"blast_radius_limit"`
	VerificationContract json.RawMessage       `json:"verification_contract"`
	RolloutMode          string                `json:"rollout_mode,omitempty"`
	Checksum             string                `json:"checksum,omitempty"`
	Status               AutomationPolicyState `json:"status"`
	MakerID              string                `json:"maker_id,omitempty"`
	CheckerID            string                `json:"checker_id,omitempty"`
	EffectiveFrom        *time.Time            `json:"effective_from,omitempty"`
	EffectiveUntil       *time.Time            `json:"effective_until,omitempty"`
	SubmittedAt          *time.Time            `json:"submitted_at,omitempty"`
	ApprovedAt           *time.Time            `json:"approved_at,omitempty"`
	ActivatedAt          *time.Time            `json:"activated_at,omitempty"`
	SuspendedAt          *time.Time            `json:"suspended_at,omitempty"`
	RetiredAt            *time.Time            `json:"retired_at,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
	Version              int64                 `json:"version"`
	RecordVersion        int64                 `json:"record_version"`
}

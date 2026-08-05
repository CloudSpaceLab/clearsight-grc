package autonomy

import "time"

type SignalType string

const (
	SignalEvidenceExpired    SignalType = "EVIDENCE_EXPIRED"
	SignalEvidenceAging      SignalType = "EVIDENCE_AGING"
	SignalSourceDegraded     SignalType = "SOURCE_DEGRADED"
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
	GeneratedAt        time.Time           `json:"generated_at"`
	Dimensions         ReadinessDimensions `json:"dimensions"`
	ActiveDrifts       []Drift             `json:"active_drifts"`
	RecommendedActions []string            `json:"recommended_actions"`
}

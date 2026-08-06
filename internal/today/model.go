package today

import "time"

type InterventionClass string

const (
	InterventionReview                 InterventionClass = "REVIEW"
	InterventionDecision               InterventionClass = "DECISION"
	InterventionAuthorization          InterventionClass = "AUTHORIZATION"
	InterventionEvidenceException      InterventionClass = "EVIDENCE_EXCEPTION"
	InterventionEscalation             InterventionClass = "ESCALATION"
	InterventionVerification           InterventionClass = "VERIFICATION"
	InterventionExternalRepresentation InterventionClass = "EXTERNAL_REPRESENTATION"
)

type SourceReference struct {
	ID         string     `json:"id,omitempty"`
	Label      string     `json:"label"`
	Version    string     `json:"version,omitempty"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
}

type VerificationPlan struct {
	State           string     `json:"state,omitempty"`
	ExpectedOutcome string     `json:"expected_outcome,omitempty"`
	Method          string     `json:"method,omitempty"`
	NextCheckAt     *time.Time `json:"next_check_at,omitempty"`
}

type WorkReceipt struct {
	ActorKind      string            `json:"actor_kind"`
	ActorID        string            `json:"actor_id"`
	ActorVersion   string            `json:"actor_version,omitempty"`
	AutonomyLevel  string            `json:"autonomy_level"`
	Summary        string            `json:"summary"`
	Sources        []SourceReference `json:"sources,omitempty"`
	Completed      []string          `json:"completed,omitempty"`
	Blocked        []string          `json:"blocked,omitempty"`
	GeneratedAt    *time.Time        `json:"generated_at,omitempty"`
	AuditReference string            `json:"audit_reference,omitempty"`
}

type GovernedRecommendation struct {
	ProposedAction    string            `json:"proposed_action"`
	Rationale         string            `json:"rationale,omitempty"`
	Alternatives      []string          `json:"alternatives,omitempty"`
	RequiredAuthority string            `json:"required_authority,omitempty"`
	ExpectedNextState string            `json:"expected_next_state,omitempty"`
	Reversible        *bool             `json:"reversible,omitempty"`
	Sources           []SourceReference `json:"sources,omitempty"`
	Assumptions       []string          `json:"assumptions,omitempty"`
	Contradictions    []string          `json:"contradictions,omitempty"`
	Verification      *VerificationPlan `json:"verification,omitempty"`
}

type AttentionItem struct {
	ID                 string                  `json:"id"`
	Type               string                  `json:"type"`
	Title              string                  `json:"title"`
	WhyNow             string                  `json:"why_now"`
	Scope              string                  `json:"scope"`
	State              string                  `json:"state"`
	Evidence           string                  `json:"evidence"`
	Owner              string                  `json:"owner"`
	DueAt              time.Time               `json:"due_at"`
	PrimaryAction      string                  `json:"primary_action"`
	ActionTargetType   string                  `json:"action_target_type,omitempty"`
	ActionTargetID     string                  `json:"action_target_id,omitempty"`
	InterventionClass  InterventionClass       `json:"intervention_class,omitempty"`
	MaterialConclusion string                  `json:"material_conclusion,omitempty"`
	ChangeSummary      string                  `json:"change_summary,omitempty"`
	Recommendation     *GovernedRecommendation `json:"recommendation,omitempty"`
	PreparedWork       *WorkReceipt            `json:"prepared_work,omitempty"`
	Verification       *VerificationPlan       `json:"verification,omitempty"`
}

// InterventionSummary is the actor-facing read projection used by Today.
// Programs, Matters, evidence and operator audit records remain authoritative.
type InterventionSummary = AttentionItem

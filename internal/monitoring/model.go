package monitoring

import (
	"encoding/json"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type RiskBand string

const (
	RiskLow         RiskBand = "LOW"
	RiskModerate    RiskBand = "MODERATE"
	RiskHigh        RiskBand = "HIGH"
	RiskCritical    RiskBand = "CRITICAL"
	RiskNotAssessed RiskBand = "NOT_ASSESSED"
)

type Thresholds struct {
	ModerateFrom int `json:"moderate_from"`
	HighFrom     int `json:"high_from"`
	CriticalFrom int `json:"critical_from"`
}

func DefaultThresholds() Thresholds {
	return Thresholds{ModerateFrom: 25, HighFrom: 50, CriticalFrom: 75}
}

type FormField = formcontract.Scoring

type SourceOperator string

const (
	OperatorEquals         SourceOperator = "EQUALS"
	OperatorNotEquals      SourceOperator = "NOT_EQUALS"
	OperatorGreaterThan    SourceOperator = "GREATER_THAN"
	OperatorGreaterOrEqual SourceOperator = "GREATER_OR_EQUAL"
	OperatorLessThan       SourceOperator = "LESS_THAN"
	OperatorLessOrEqual    SourceOperator = "LESS_OR_EQUAL"
	OperatorPresent        SourceOperator = "PRESENT"
	OperatorMaxAgeMinutes  SourceOperator = "MAX_AGE_MINUTES"
)

type SourceRule struct {
	ID         string         `json:"id"`
	Field      string         `json:"field"`
	Operator   SourceOperator `json:"operator"`
	Expected   string         `json:"expected,omitempty"`
	RiskPoints int            `json:"risk_points"`
	Critical   bool           `json:"critical,omitempty"`
}

type RuleOutcome string

const (
	RulePassed        RuleOutcome = "PASS"
	RuleFailed        RuleOutcome = "FAIL"
	RuleIndeterminate RuleOutcome = "INDETERMINATE"
)

type RuleResult struct {
	RuleID   string      `json:"rule_id,omitempty"`
	FieldID  string      `json:"field_id"`
	Outcome  RuleOutcome `json:"outcome"`
	Points   int         `json:"points"`
	Critical bool        `json:"critical,omitempty"`
	Reason   string      `json:"reason"`
}

type Evaluation struct {
	Score            *float64     `json:"score,omitempty"`
	Band             RiskBand     `json:"band"`
	Coverage         float64      `json:"coverage"`
	CriticalFailures []RuleResult `json:"critical_failures,omitempty"`
	RuleResults      []RuleResult `json:"rule_results"`
}

type LifecycleStatus string

const (
	LifecycleDraft           LifecycleStatus = "DRAFT"
	LifecyclePendingApproval LifecycleStatus = "PENDING_APPROVAL"
	LifecycleActive          LifecycleStatus = "ACTIVE"
	LifecycleRejected        LifecycleStatus = "REJECTED"
	LifecyclePaused          LifecycleStatus = "PAUSED"
	LifecycleRetired         LifecycleStatus = "RETIRED"
)

type Lifecycle struct {
	Status         LifecycleStatus `json:"status"`
	IsCurrent      bool            `json:"is_current"`
	EffectiveFrom  *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time      `json:"effective_until,omitempty"`
	Version        int64           `json:"version"`
	CreatedBy      string          `json:"created_by,omitempty"`
	SubmittedBy    string          `json:"submitted_by,omitempty"`
	ApprovedBy     string          `json:"approved_by,omitempty"`
	RejectedBy     string          `json:"rejected_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type TemplateField = formcontract.Field

type FormTemplate struct {
	ID            string                    `json:"id"`
	TenantID      string                    `json:"tenant_id"`
	LegalEntityID string                    `json:"legal_entity_id"`
	ProgramID     string                    `json:"program_id"`
	Code          string                    `json:"code"`
	Name          string                    `json:"name"`
	Purpose       string                    `json:"purpose"`
	Presentation  formcontract.Presentation `json:"presentation"`
	Sections      []formcontract.Section    `json:"sections"`
	Fields        []TemplateField           `json:"fields"`
	Lifecycle
}

type InputKind string

const (
	InputForm   InputKind = "FORM"
	InputSource InputKind = "SOURCE"
)

type FailureAction string

const (
	FailureReview          FailureAction = "REVIEW"
	FailureRecommendMatter FailureAction = "RECOMMEND_MATTER"
)

type MonitoringCheck struct {
	ID                      string        `json:"id"`
	TenantID                string        `json:"tenant_id"`
	ProgramID               string        `json:"program_id"`
	RequirementID           string        `json:"requirement_id,omitempty"`
	ControlImplementationID string        `json:"control_implementation_id,omitempty"`
	EvidenceContractID      string        `json:"evidence_contract_id,omitempty"`
	Code                    string        `json:"code"`
	Name                    string        `json:"name"`
	Claim                   string        `json:"claim"`
	InputKind               InputKind     `json:"input_kind"`
	FormTemplateID          string        `json:"form_template_id,omitempty"`
	FormTemplateVersion     int64         `json:"form_template_version,omitempty"`
	BindingID               string        `json:"binding_id,omitempty"`
	BindingVersion          int64         `json:"binding_version,omitempty"`
	SourceRules             []SourceRule  `json:"source_rules,omitempty"`
	Thresholds              Thresholds    `json:"thresholds"`
	FreshnessMinutes        int           `json:"freshness_minutes"`
	MinimumCoverage         float64       `json:"minimum_coverage"`
	OwnerPrincipalID        string        `json:"owner_principal_id,omitempty"`
	ReviewerPrincipalID     string        `json:"reviewer_principal_id,omitempty"`
	FailureAction           FailureAction `json:"failure_action"`
	Lifecycle
}

type MonitoringResult struct {
	ID                     string          `json:"id"`
	TenantID               string          `json:"tenant_id"`
	ProgramID              string          `json:"program_id"`
	MonitoringCheckID      string          `json:"monitoring_check_id"`
	MonitoringCheckVersion int64           `json:"monitoring_check_version"`
	InputKind              InputKind       `json:"input_kind"`
	InputReferenceID       string          `json:"input_reference_id"`
	InputReferenceVersion  int64           `json:"input_reference_version"`
	Evaluation             Evaluation      `json:"evaluation"`
	SourceReceipt          json.RawMessage `json:"source_receipt,omitempty"`
	SubmissionProvenance   json.RawMessage `json:"submission_provenance,omitempty"`
	EvaluatedAt            time.Time       `json:"evaluated_at"`
	EvaluatorVersion       string          `json:"evaluator_version"`
	CreatedAt              time.Time       `json:"created_at"`
}

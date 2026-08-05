package continuity

import (
	"encoding/json"
	"time"
)

type ProgramStatus string

type ProgramState string

type RequirementStatus string

type ApplicabilityStatus string

type ControlObjectiveStatus string

type ControlImplementationStatus string

type EvidenceContractStatus string

type EvidenceConclusion string

type MatterType string

type MatterStatus string

type DecisionStatus string

type ActionStatus string

type VerificationStatus string

type VerificationResultStatus string

type ResponseStatus string

type ActorType string

const (
	ProgramDraft   ProgramStatus = "DRAFT"
	ProgramActive  ProgramStatus = "ACTIVE"
	ProgramPaused  ProgramStatus = "PAUSED"
	ProgramRetired ProgramStatus = "RETIRED"
)

const (
	StateCurrent               ProgramState = "CURRENT"
	StateAtRisk                ProgramState = "AT_RISK"
	StateGapIdentified         ProgramState = "GAP_IDENTIFIED"
	StateEvidenceInsufficient  ProgramState = "EVIDENCE_INSUFFICIENT"
	StateImplementationPending ProgramState = "IMPLEMENTATION_PENDING"
	StateOverdue               ProgramState = "OVERDUE"
	StateUnderReview           ProgramState = "UNDER_REVIEW"
	StateNotApplicable         ProgramState = "NOT_APPLICABLE"
	StateUnknown               ProgramState = "UNKNOWN"
)

const (
	RequirementDraft      RequirementStatus = "DRAFT"
	RequirementApproved   RequirementStatus = "APPROVED"
	RequirementSuperseded RequirementStatus = "SUPERSEDED"
	RequirementRetired    RequirementStatus = "RETIRED"
)

const (
	ApplicabilityPotential     ApplicabilityStatus = "POTENTIALLY_APPLICABLE"
	ApplicabilityApplicable    ApplicabilityStatus = "APPLICABLE"
	ApplicabilityPartial       ApplicabilityStatus = "PARTIALLY_APPLICABLE"
	ApplicabilityNotApplicable ApplicabilityStatus = "NOT_APPLICABLE"
	ApplicabilityLater         ApplicabilityStatus = "APPLIES_LATER"
	ApplicabilitySuperseded    ApplicabilityStatus = "SUPERSEDED"
)

const (
	ObjectiveDraft   ControlObjectiveStatus = "DRAFT"
	ObjectiveActive  ControlObjectiveStatus = "ACTIVE"
	ObjectiveRetired ControlObjectiveStatus = "RETIRED"
)

const (
	ImplementationPlanned     ControlImplementationStatus = "PLANNED"
	ImplementationInProgress  ControlImplementationStatus = "IN_PROGRESS"
	ImplementationImplemented ControlImplementationStatus = "IMPLEMENTED"
	ImplementationInactive    ControlImplementationStatus = "INACTIVE"
	ImplementationRetired     ControlImplementationStatus = "RETIRED"
)

const (
	EvidenceContractDraft   EvidenceContractStatus = "DRAFT"
	EvidenceContractActive  EvidenceContractStatus = "ACTIVE"
	EvidenceContractRetired EvidenceContractStatus = "RETIRED"
)

const (
	EvidenceSupported          EvidenceConclusion = "SUPPORTED"
	EvidencePartiallySupported EvidenceConclusion = "PARTIALLY_SUPPORTED"
	EvidenceUnsupported        EvidenceConclusion = "UNSUPPORTED"
	EvidenceContradicted       EvidenceConclusion = "CONTRADICTED"
	EvidenceIndeterminate      EvidenceConclusion = "INDETERMINATE"
	EvidenceExpired            EvidenceConclusion = "EXPIRED"
)

const (
	MatterRegulatoryChange      MatterType = "REGULATORY_CHANGE"
	MatterSupervisoryFinding    MatterType = "SUPERVISORY_FINDING"
	MatterAuthorityRequest      MatterType = "AUTHORITY_REQUEST"
	MatterRiskSituation         MatterType = "RISK_SITUATION"
	MatterControlGap            MatterType = "CONTROL_GAP"
	MatterAuditFinding          MatterType = "AUDIT_FINDING"
	MatterException             MatterType = "EXCEPTION"
	MatterIncident              MatterType = "INCIDENT"
	MatterOperationalLoss       MatterType = "OPERATIONAL_LOSS"
	MatterDataBreach            MatterType = "DATA_BREACH"
	MatterVendorDeficiency      MatterType = "VENDOR_DEFICIENCY"
	MatterCustomerConcern       MatterType = "CUSTOMER_CONCERN"
	MatterOverdueObligation     MatterType = "OVERDUE_OBLIGATION"
	MatterFailedVerification    MatterType = "FAILED_VERIFICATION"
	MatterEvidenceContradiction MatterType = "EVIDENCE_CONTRADICTION"
	MatterKRIBreach             MatterType = "KRI_BREACH"
)

const (
	MatterDraft               MatterStatus = "DRAFT"
	MatterInitialReview       MatterStatus = "TRIAGE"
	MatterAssessment          MatterStatus = "ASSESSMENT"
	MatterDecisionRequired    MatterStatus = "DECISION_REQUIRED"
	MatterActionsInProgress   MatterStatus = "ACTION_IN_PROGRESS"
	MatterResponsePreparation MatterStatus = "RESPONSE_PREPARATION"
	MatterVerification        MatterStatus = "VERIFICATION"
	MatterClosed              MatterStatus = "CLOSED"
	MatterCancelled           MatterStatus = "CANCELLED"
)

const (
	DecisionProposed              DecisionStatus = "PROPOSED"
	DecisionApproved              DecisionStatus = "APPROVED"
	DecisionConditionallyApproved DecisionStatus = "CONDITIONALLY_APPROVED"
	DecisionRejected              DecisionStatus = "REJECTED"
	DecisionReturned              DecisionStatus = "RETURNED"
	DecisionExpired               DecisionStatus = "EXPIRED"
	DecisionSuperseded            DecisionStatus = "SUPERSEDED"
)

const (
	ActionPlanned     ActionStatus = "PLANNED"
	ActionInProgress  ActionStatus = "IN_PROGRESS"
	ActionImplemented ActionStatus = "IMPLEMENTED"
	ActionBlocked     ActionStatus = "BLOCKED"
	ActionCancelled   ActionStatus = "CANCELLED"
)

const (
	VerificationActive  VerificationStatus = "ACTIVE"
	VerificationRetired VerificationStatus = "RETIRED"
)

const (
	VerificationPassed       VerificationResultStatus = "PASS"
	VerificationFailed       VerificationResultStatus = "FAIL"
	VerificationInconclusive VerificationResultStatus = "INCONCLUSIVE"
)

const (
	ResponseDraft        ResponseStatus = "DRAFT"
	ResponseInReview     ResponseStatus = "IN_REVIEW"
	ResponseApproved     ResponseStatus = "APPROVED"
	ResponseTransmitted  ResponseStatus = "TRANSMITTED"
	ResponseAcknowledged ResponseStatus = "ACKNOWLEDGED"
	ResponseRejected     ResponseStatus = "REJECTED"
	ResponseWithdrawn    ResponseStatus = "WITHDRAWN"
)

const (
	ActorPerson  ActorType = "PERSON"
	ActorService ActorType = "SERVICE"
	ActorSystem  ActorType = "SYSTEM"
)

type Program struct {
	ID                   string          `json:"id"`
	TenantID             string          `json:"tenant_id"`
	LegalEntityID        string          `json:"legal_entity_id,omitempty"`
	Code                 string          `json:"code"`
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	Status               ProgramStatus   `json:"status"`
	OwningFunction       string          `json:"owning_function"`
	OwnerPrincipalID     string          `json:"owner_principal_id,omitempty"`
	AuthorityPrincipalID string          `json:"authority_principal_id,omitempty"`
	Jurisdiction         string          `json:"jurisdiction,omitempty"`
	Scope                json.RawMessage `json:"scope"`
	EffectiveFrom        time.Time       `json:"effective_from"`
	EffectiveUntil       *time.Time      `json:"effective_until,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	Version              int64           `json:"version"`
}

type Requirement struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	ProgramID      string            `json:"program_id"`
	SourceID       string            `json:"source_id,omitempty"`
	Code           string            `json:"code"`
	Title          string            `json:"title"`
	Statement      string            `json:"statement"`
	SourceAnchor   string            `json:"source_anchor,omitempty"`
	Modality       string            `json:"modality"`
	Actor          string            `json:"actor,omitempty"`
	Action         string            `json:"action,omitempty"`
	Object         string            `json:"object,omitempty"`
	Status         RequirementStatus `json:"status"`
	EffectiveFrom  time.Time         `json:"effective_from"`
	EffectiveUntil *time.Time        `json:"effective_until,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Version        int64             `json:"version"`
}

type Applicability struct {
	ID             string              `json:"id"`
	TenantID       string              `json:"tenant_id"`
	ProgramID      string              `json:"program_id"`
	RequirementID  string              `json:"requirement_id"`
	Status         ApplicabilityStatus `json:"status"`
	Scope          json.RawMessage     `json:"scope"`
	Rationale      string              `json:"rationale"`
	ApprovedBy     string              `json:"approved_by"`
	EffectiveFrom  time.Time           `json:"effective_from"`
	EffectiveUntil *time.Time          `json:"effective_until,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	Version        int64               `json:"version"`
}

type ControlObjective struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	ProgramID string                 `json:"program_id"`
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Outcome   string                 `json:"outcome"`
	Status    ControlObjectiveStatus `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	Version   int64                  `json:"version"`
}

type ControlImplementation struct {
	ID                 string                      `json:"id"`
	TenantID           string                      `json:"tenant_id"`
	ProgramID          string                      `json:"program_id"`
	ObjectiveID        string                      `json:"objective_id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	ImplementationType string                      `json:"implementation_type"`
	OwnerPrincipalID   string                      `json:"owner_principal_id,omitempty"`
	Scope              json.RawMessage             `json:"scope"`
	Status             ControlImplementationStatus `json:"status"`
	EffectiveFrom      time.Time                   `json:"effective_from"`
	EffectiveUntil     *time.Time                  `json:"effective_until,omitempty"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	Version            int64                       `json:"version"`
}

type RequirementControlLink struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ProgramID        string    `json:"program_id"`
	RequirementID    string    `json:"requirement_id"`
	ImplementationID string    `json:"implementation_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type EvidenceContract struct {
	ID                      string                 `json:"id"`
	TenantID                string                 `json:"tenant_id"`
	ProgramID               string                 `json:"program_id"`
	RequirementID           string                 `json:"requirement_id,omitempty"`
	ControlImplementationID string                 `json:"control_implementation_id,omitempty"`
	Code                    string                 `json:"code"`
	Name                    string                 `json:"name"`
	Claim                   string                 `json:"claim"`
	AcceptableSourceIDs     []string               `json:"acceptable_source_ids"`
	PopulationScope         json.RawMessage        `json:"population_scope"`
	FreshnessMinutes        int                    `json:"freshness_minutes"`
	MinimumCoverage         float64                `json:"minimum_coverage"`
	IndependenceRequired    bool                   `json:"independence_required"`
	ContradictionPolicy     string                 `json:"contradiction_policy"`
	FailureAction           string                 `json:"failure_action"`
	Status                  EvidenceContractStatus `json:"status"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
	Version                 int64                  `json:"version"`
}

type EvidenceAssessment struct {
	ID         string             `json:"id"`
	TenantID   string             `json:"tenant_id"`
	ProgramID  string             `json:"program_id"`
	ContractID string             `json:"contract_id"`
	Conclusion EvidenceConclusion `json:"conclusion"`
	Coverage   float64            `json:"coverage"`
	Basis      json.RawMessage    `json:"basis"`
	ValidUntil *time.Time         `json:"valid_until,omitempty"`
	AssessedBy string             `json:"assessed_by,omitempty"`
	AssessedAt time.Time          `json:"assessed_at"`
	CreatedAt  time.Time          `json:"created_at"`
}

type ComplianceDimensions struct {
	Interpretation         ProgramState `json:"interpretation"`
	Applicability          ProgramState `json:"applicability"`
	ControlDesign          ProgramState `json:"control_design"`
	Implementation         ProgramState `json:"implementation"`
	EvidenceSufficiency    ProgramState `json:"evidence_sufficiency"`
	OperatingEffectiveness ProgramState `json:"operating_effectiveness"`
	Exception              ProgramState `json:"exception"`
	Assurance              ProgramState `json:"assurance"`
	Deadline               ProgramState `json:"deadline"`
	SourceQuality          ProgramState `json:"source_quality"`
}

type StateReason struct {
	Code       string `json:"code"`
	Summary    string `json:"summary"`
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
}

type ProgramStateSnapshot struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	ProgramID       string               `json:"program_id"`
	Overall         ProgramState         `json:"overall"`
	Dimensions      ComplianceDimensions `json:"dimensions"`
	Reasons         []StateReason        `json:"reasons"`
	OpenMatterCount int                  `json:"open_matter_count"`
	TriggerType     string               `json:"trigger_type,omitempty"`
	TriggerID       string               `json:"trigger_id,omitempty"`
	GeneratedAt     time.Time            `json:"generated_at"`
	ProgramVersion  int64                `json:"program_version"`
}

type ProgramAggregate struct {
	StateLabel              string                   `json:"state_label"`
	Program                 Program                  `json:"program"`
	Requirements            []Requirement            `json:"requirements"`
	Applicability           []Applicability          `json:"applicability"`
	ControlObjectives       []ControlObjective       `json:"control_objectives"`
	ControlImplementations  []ControlImplementation  `json:"control_implementations"`
	RequirementControlLinks []RequirementControlLink `json:"requirement_control_links"`
	EvidenceContracts       []EvidenceContract       `json:"evidence_contracts"`
	EvidenceAssessments     []EvidenceAssessment     `json:"evidence_assessments"`
	Triggers                []Trigger                `json:"triggers"`
	CurrentState            *ProgramStateSnapshot    `json:"current_state,omitempty"`
}

type Matter struct {
	ID                string          `json:"id"`
	TenantID          string          `json:"tenant_id"`
	Reference         string          `json:"reference"`
	Type              MatterType      `json:"type"`
	Status            MatterStatus    `json:"status"`
	Priority          int             `json:"priority"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	Scope             json.RawMessage `json:"scope"`
	SourceType        string          `json:"source_type,omitempty"`
	SourceID          string          `json:"source_id,omitempty"`
	TriggerType       string          `json:"trigger_type,omitempty"`
	TriggerID         string          `json:"trigger_id,omitempty"`
	TriggerKey        string          `json:"trigger_key,omitempty"`
	KnownFacts        json.RawMessage `json:"known_facts"`
	MissingFacts      json.RawMessage `json:"missing_facts"`
	Contradictions    json.RawMessage `json:"contradictions"`
	OwnerPrincipalID  string          `json:"owner_principal_id,omitempty"`
	RequiredAuthority string          `json:"required_authority,omitempty"`
	DueAt             *time.Time      `json:"due_at,omitempty"`
	ClosedAt          *time.Time      `json:"closed_at,omitempty"`
	ClosureReason     string          `json:"closure_reason,omitempty"`
	ReopenCount       int             `json:"reopen_count"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Version           int64           `json:"version"`
}

type MatterLink struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	MatterID      string    `json:"matter_id"`
	ProgramID     string    `json:"program_id,omitempty"`
	RequirementID string    `json:"requirement_id,omitempty"`
	ControlID     string    `json:"control_id,omitempty"`
	Relationship  string    `json:"relationship"`
	CreatedAt     time.Time `json:"created_at"`
}

type Decision struct {
	ID                   string          `json:"id"`
	TenantID             string          `json:"tenant_id"`
	MatterID             string          `json:"matter_id"`
	Type                 string          `json:"type"`
	Status               DecisionStatus  `json:"status"`
	Options              json.RawMessage `json:"options"`
	SelectedOption       string          `json:"selected_option,omitempty"`
	Rationale            string          `json:"rationale"`
	Conditions           json.RawMessage `json:"conditions"`
	AuthorityPrincipalID string          `json:"authority_principal_id,omitempty"`
	DecidedAt            *time.Time      `json:"decided_at,omitempty"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	Version              int64           `json:"version"`
}

type Action struct {
	ID               string       `json:"id"`
	TenantID         string       `json:"tenant_id"`
	MatterID         string       `json:"matter_id"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	OwnerPrincipalID string       `json:"owner_principal_id,omitempty"`
	Status           ActionStatus `json:"status"`
	DueAt            *time.Time   `json:"due_at,omitempty"`
	ImplementedAt    *time.Time   `json:"implemented_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	Version          int64        `json:"version"`
}

type VerificationContract struct {
	ID                       string             `json:"id"`
	TenantID                 string             `json:"tenant_id"`
	MatterID                 string             `json:"matter_id"`
	ActionID                 string             `json:"action_id,omitempty"`
	ExpectedOutcome          string             `json:"expected_outcome"`
	Baseline                 json.RawMessage    `json:"baseline"`
	Scope                    json.RawMessage    `json:"scope"`
	MeasurementSourceID      string             `json:"measurement_source_id,omitempty"`
	Threshold                json.RawMessage    `json:"threshold"`
	ObservationPeriodMinutes int                `json:"observation_period_minutes"`
	AuthorityPrincipalID     string             `json:"authority_principal_id,omitempty"`
	FailureResponse          string             `json:"failure_response"`
	Status                   VerificationStatus `json:"status"`
	CreatedAt                time.Time          `json:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at"`
	Version                  int64              `json:"version"`
}

type VerificationResult struct {
	ID                  string                   `json:"id"`
	TenantID            string                   `json:"tenant_id"`
	MatterID            string                   `json:"matter_id"`
	ContractID          string                   `json:"contract_id"`
	Result              VerificationResultStatus `json:"result"`
	Observations        json.RawMessage          `json:"observations"`
	EvidenceReferences  json.RawMessage          `json:"evidence_references"`
	ReviewerPrincipalID string                   `json:"reviewer_principal_id,omitempty"`
	Rationale           string                   `json:"rationale"`
	ObservedAt          time.Time                `json:"observed_at"`
	CreatedAt           time.Time                `json:"created_at"`
}

type ResponsePackage struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	MatterID       string          `json:"matter_id"`
	Purpose        string          `json:"purpose"`
	Audience       string          `json:"audience"`
	Status         ResponseStatus  `json:"status"`
	Manifest       json.RawMessage `json:"manifest"`
	ApprovedBy     string          `json:"approved_by,omitempty"`
	TransmittedAt  *time.Time      `json:"transmitted_at,omitempty"`
	AcknowledgedAt *time.Time      `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Version        int64           `json:"version"`
}

type MatterAggregate struct {
	TypeLabel             string                 `json:"type_label"`
	StatusLabel           string                 `json:"status_label"`
	NextAction            string                 `json:"next_action"`
	Matter                Matter                 `json:"matter"`
	Links                 []MatterLink           `json:"links"`
	Decisions             []Decision             `json:"decisions"`
	Actions               []Action               `json:"actions"`
	VerificationContracts []VerificationContract `json:"verification_contracts"`
	VerificationResults   []VerificationResult   `json:"verification_results"`
	ResponsePackages      []ResponsePackage      `json:"response_packages"`
	Closure               ClosureAssessment      `json:"closure"`
}

type ClosureAssessment struct {
	Ready   bool     `json:"ready"`
	Reasons []string `json:"reasons"`
}

type Trigger struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	ProgramID   string          `json:"program_id"`
	Type        string          `json:"type"`
	SubjectType string          `json:"subject_type,omitempty"`
	SubjectID   string          `json:"subject_id,omitempty"`
	DedupeKey   string          `json:"dedupe_key"`
	Payload     json.RawMessage `json:"payload"`
	ObservedAt  time.Time       `json:"observed_at"`
	Source      string          `json:"source"`
	ActorID     string          `json:"actor_id,omitempty"`
}

type Event struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      string          `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ActorType        ActorType       `json:"actor_type"`
	ActorID          string          `json:"actor_id,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
}

type CreateProgramInput struct {
	TenantID             string          `json:"tenant_id"`
	LegalEntityID        string          `json:"legal_entity_id,omitempty"`
	Code                 string          `json:"code"`
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	OwningFunction       string          `json:"owning_function"`
	OwnerPrincipalID     string          `json:"owner_principal_id,omitempty"`
	AuthorityPrincipalID string          `json:"authority_principal_id,omitempty"`
	Jurisdiction         string          `json:"jurisdiction,omitempty"`
	Scope                json.RawMessage `json:"scope"`
	EffectiveFrom        time.Time       `json:"effective_from"`
	EffectiveUntil       *time.Time      `json:"effective_until,omitempty"`
	ActorID              string          `json:"actor_id,omitempty"`
}

type ProgramTransitionInput struct {
	TenantID        string        `json:"tenant_id"`
	ID              string        `json:"id"`
	ExpectedVersion int64         `json:"expected_version"`
	To              ProgramStatus `json:"to"`
	ActorID         string        `json:"actor_id,omitempty"`
	Rationale       string        `json:"rationale"`
}

type CreateMatterInput struct {
	TenantID          string          `json:"tenant_id"`
	Type              MatterType      `json:"type"`
	Priority          int             `json:"priority"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	Scope             json.RawMessage `json:"scope"`
	SourceType        string          `json:"source_type,omitempty"`
	SourceID          string          `json:"source_id,omitempty"`
	TriggerType       string          `json:"trigger_type,omitempty"`
	TriggerID         string          `json:"trigger_id,omitempty"`
	TriggerKey        string          `json:"trigger_key,omitempty"`
	KnownFacts        json.RawMessage `json:"known_facts"`
	MissingFacts      json.RawMessage `json:"missing_facts"`
	Contradictions    json.RawMessage `json:"contradictions"`
	OwnerPrincipalID  string          `json:"owner_principal_id,omitempty"`
	RequiredAuthority string          `json:"required_authority,omitempty"`
	DueAt             *time.Time      `json:"due_at,omitempty"`
	ProgramID         string          `json:"program_id,omitempty"`
	RequirementID     string          `json:"requirement_id,omitempty"`
	ControlID         string          `json:"control_id,omitempty"`
	ActorID           string          `json:"actor_id,omitempty"`
}

type AddMatterLinkInput struct {
	TenantID        string `json:"tenant_id"`
	MatterID        string `json:"matter_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ProgramID       string `json:"program_id,omitempty"`
	RequirementID   string `json:"requirement_id,omitempty"`
	ControlID       string `json:"control_id,omitempty"`
	Relationship    string `json:"relationship"`
	ActorID         string `json:"actor_id,omitempty"`
}

type TransitionInput struct {
	TenantID        string       `json:"tenant_id"`
	ID              string       `json:"id"`
	ExpectedVersion int64        `json:"expected_version"`
	To              MatterStatus `json:"to"`
	ActorID         string       `json:"actor_id,omitempty"`
	Rationale       string       `json:"rationale"`
}

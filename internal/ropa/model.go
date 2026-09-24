package ropa

import (
	"encoding/json"
	"fmt"
	"time"
)

// ProjectionVersion identifies the dashboard projection contract. A snapshot
// produced by a different version is reported stale rather than current.
const ProjectionVersion = "ropa-v1"

type Status string

const (
	StatusNew    Status = "NEW"
	StatusOpen   Status = "OPEN"
	StatusClosed Status = "CLOSED"
)

func (s Status) String() string {
	switch s {
	case StatusNew:
		return "Not started"
	case StatusOpen:
		return "In progress"
	case StatusClosed:
		return "Complete"
	default:
		return "Unknown"
	}
}

// ValidStatus lets consumers validate against the register's own status
// vocabulary instead of copying the list, which is how a filter could offer a
// status the register cannot store.
func ValidStatus(status Status) bool { return validStatus(status) }

// TransferBasis is the controlled NDPA Article 45 and Schedule 5 safeguard vocabulary.
type TransferBasis string

const (
	TransferBasisAdequacy                TransferBasis = "ADEQUACY"
	TransferBasisApprovedInstrument      TransferBasis = "APPROVED_INSTRUMENT"
	TransferBasisRecognisedLawfulBasis   TransferBasis = "RECOGNISED_LAWFUL_BASIS"
	TransferBasisConsent                 TransferBasis = "CONSENT"
	TransferBasisStandardContractClauses TransferBasis = "STANDARD_CONTRACT_CLAUSES"
	TransferBasisBindingCorporateRules   TransferBasis = "BINDING_CORPORATE_RULES"
	TransferBasisCertification           TransferBasis = "CERTIFICATION"
	TransferBasisNotApplicable           TransferBasis = "NOT_APPLICABLE"
)

// Valid reports whether the value belongs to the controlled safeguard vocabulary.
func (b TransferBasis) Valid() bool {
	switch b {
	case TransferBasisAdequacy,
		TransferBasisApprovedInstrument,
		TransferBasisRecognisedLawfulBasis,
		TransferBasisConsent,
		TransferBasisStandardContractClauses,
		TransferBasisBindingCorporateRules,
		TransferBasisCertification,
		TransferBasisNotApplicable:
		return true
	default:
		return false
	}
}

type ProcessingActivity struct {
	ID                      string     `json:"id"`
	TenantID                string     `json:"tenant_id"`
	LegalEntityID           string     `json:"legal_entity_id"`
	Code                    string     `json:"code"`
	Name                    string     `json:"name"`
	Description             string     `json:"description"`
	Status                  Status     `json:"status"`
	Purpose                 string     `json:"purpose"`
	LawfulBasis             string     `json:"lawful_basis"`
	Controller              string     `json:"controller"`
	Processor               string     `json:"processor"`
	AutomatedDecisionMaking bool       `json:"automated_decision_making"`
	DataSubjectCategories   string     `json:"data_subject_categories"`
	PersonalDataCategories  string     `json:"personal_data_categories"`
	SecurityMeasures        string     `json:"security_measures"`
	RetentionPeriod         string     `json:"retention_period"`
	StartDate               *time.Time `json:"start_date,omitempty"`
	EndDate                 *time.Time `json:"end_date,omitempty"`
	NextReviewDate          *time.Time `json:"next_review_date,omitempty"`
	// OwnerPrincipalID is the accountable owner; required authority is a separate responsibility.
	OwnerPrincipalID             string    `json:"owner_principal_id,omitempty"`
	RequiredAuthorityPrincipalID string    `json:"required_authority_principal_id,omitempty"`
	ProgramID                    string    `json:"program_id,omitempty"`
	Version                      int64     `json:"version"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`

	DataCategories []DataCategory `json:"data_categories,omitempty"`
	Recipients     []Recipient    `json:"recipients,omitempty"`
	Systems        []System       `json:"systems,omitempty"`
	Reviews        []Review       `json:"reviews,omitempty"`
}

// DataCategory, Recipient, System, and Review are nested child values of a
// ProcessingActivity. A child is only ever read or written inside the scope
// of its enclosing ProcessingActivity, which carries tenant and legal-entity
// scope. The repository binds that scope from the parent and never from the
// child.
type DataCategory struct {
	Category    string `json:"category"`
	Sensitivity string `json:"sensitivity"`
}

// Recipient records a disclosed recipient and its cross-border transfer facts.
// CountryCode is present only for cross-border recipients. TransferBasis uses the
// Article 45 / Schedule 5 safeguard vocabulary.
type Recipient struct {
	Recipient     string        `json:"recipient"`
	RecipientKind string        `json:"recipient_kind"`
	CountryCode   string        `json:"country_code,omitempty"`
	IsCrossBorder bool          `json:"is_cross_border"`
	TransferBasis TransferBasis `json:"transfer_basis"`
}

type System struct {
	SystemName string `json:"system_name"`
	SystemKind string `json:"system_kind"`
}

type Review struct {
	ID                  string     `json:"id"`
	CreatedAt           time.Time  `json:"created_at"`
	DueDate             time.Time  `json:"due_date"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	Outcome             string     `json:"outcome,omitempty"`
	ReviewerPrincipalID string     `json:"reviewer_principal_id,omitempty"`
}

type Event struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	LegalEntityID    string          `json:"legal_entity_id"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      string          `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ActorType        string          `json:"actor_type"`
	ActorID          string          `json:"actor_id,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
}

// Coverage separates what the dashboard counted from what it could not. A nil
// pointer means the value is UNKNOWN, not zero.
type Coverage struct {
	Population int  `json:"population"`
	Excluded   *int `json:"excluded,omitempty"`
	Unknown    *int `json:"unknown,omitempty"`
}

type Freshness string

const (
	FreshnessCurrent Freshness = "CURRENT"
	FreshnessStale   Freshness = "STALE"
)

type RegisterCounts struct {
	Total              int `json:"total"`
	New                int `json:"new"`
	Open               int `json:"open"`
	Closed             int `json:"closed"`
	ReviewOverdue      int `json:"review_overdue"`
	MissingLawfulBasis int `json:"missing_lawful_basis"`
	MissingOwner       int `json:"missing_owner"`
	NoDataSubjects     int `json:"no_data_subjects"`
	Retired            int `json:"retired"`
}

type RegisterSummary struct {
	TenantID          string         `json:"-"`
	LegalEntityID     string         `json:"-"`
	GeneratedAt       time.Time      `json:"generated_at"`
	ProjectionVersion string         `json:"projection_version"`
	Freshness         Freshness      `json:"freshness"`
	SourceHighWater   time.Time      `json:"source_high_water"`
	Coverage          Coverage       `json:"coverage"`
	Counts            RegisterCounts `json:"counts"`
}

type Aggregate struct {
	ProcessingActivity
	Events []Event `json:"events,omitempty"`
}

// IsRetired reports whether processing has ceased. An end date removes the
// activity from the live register while its history remains reconstructable.
func (a Aggregate) IsRetired() bool {
	return a.EndDate != nil && !a.EndDate.IsZero()
}

// ReviewOverdue reports whether the next review was strictly before now.
func (a Aggregate) ReviewOverdue(now time.Time) bool {
	return a.NextReviewDate != nil && !a.NextReviewDate.IsZero() && a.NextReviewDate.Before(now)
}

// String returns the activity's working identifier for logs and errors.
func (a Aggregate) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.Code)
}

package evidence

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type AccessPolicy string

const (
	AccessDirectMagicLink AccessPolicy = "DIRECT_MAGIC_LINK"
	AccessSharedEmailOTP  AccessPolicy = "SHARED_LINK_EMAIL_OTP"
	AccessDirectEmailOTP  AccessPolicy = "DIRECT_LINK_EMAIL_OTP"
)

type AccessAssurance string

const (
	AssuranceLinkPossession AccessAssurance = "LINK_POSSESSION"
	AssuranceEmailVerified  AccessAssurance = "EMAIL_VERIFIED"
)

type RecipientRole string

const (
	RecipientTo RecipientRole = "TO"
	RecipientCC RecipientRole = "CC"
)

type DistributionStatus string

const (
	DistributionDraft      DistributionStatus = "DRAFT"
	DistributionReady      DistributionStatus = "READY"
	DistributionOpen       DistributionStatus = "OPEN"
	DistributionLocked     DistributionStatus = "LOCKED"
	DistributionCompleted  DistributionStatus = "COMPLETED"
	DistributionExpired    DistributionStatus = "EXPIRED"
	DistributionRevoked    DistributionStatus = "REVOKED"
	DistributionSuperseded DistributionStatus = "SUPERSEDED"
)

type DistributionRecipientState string

const (
	DistributionRecipientPending   DistributionRecipientState = "PENDING"
	DistributionRecipientDelivered DistributionRecipientState = "DELIVERED"
	DistributionRecipientVerified  DistributionRecipientState = "VERIFIED"
	DistributionRecipientRevoked   DistributionRecipientState = "REVOKED"
	DistributionRecipientCompleted DistributionRecipientState = "COMPLETED"
)

type ResponseWorkspaceStatus string

const (
	ResponseWorkspaceOpen      ResponseWorkspaceStatus = "OPEN"
	ResponseWorkspaceLocked    ResponseWorkspaceStatus = "LOCKED"
	ResponseWorkspaceCompleted ResponseWorkspaceStatus = "COMPLETED"
	ResponseWorkspaceRevoked   ResponseWorkspaceStatus = "REVOKED"
)

type ResponseRevisionState string

const (
	ResponseRevisionProvisional ResponseRevisionState = "PROVISIONAL"
	ResponseRevisionFinal       ResponseRevisionState = "FINAL"
)

type FormDistribution struct {
	ID                  string
	TenantID            string
	LegalEntityID       string
	FormTemplateID      string
	FormTemplateVersion int64
	SubjectType         string
	SubjectID           string
	Title               string
	Purpose             string
	AccessPolicy        AccessPolicy
	Status              DistributionStatus
	Deadline            time.Time
	RouteExpiresAt      time.Time
	ReminderPolicy      map[string]any
	CreatedBy           string
	Version             int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DistributionRecipient is deliberately a safe read model. Protected address
// hashes, ciphertext, key identifiers and full external addresses never leave
// the storage boundary through this type.
type DistributionRecipient struct {
	ID             string
	DistributionID string
	TenantID       string
	LegalEntityID  string
	Role           RecipientRole
	Type           RecipientType
	PrincipalID    string
	RequestID      string
	AudienceHint   string
	ContactLabel   string
	State          DistributionRecipientState
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DistributionRecipientInput struct {
	Role         RecipientRole
	Type         RecipientType
	PrincipalID  string
	Address      string
	AudienceHint string
	ContactLabel string
}

type CreateDistributionInput struct {
	TenantID            string
	LegalEntityID       string
	FormTemplateID      string
	FormTemplateVersion int64
	SubjectType         string
	SubjectID           string
	Title               string
	Purpose             string
	AccessPolicy        AccessPolicy
	EstimatedMinutes    int
	Deadline            time.Time
	RouteExpiresAt      time.Time
	ReminderPolicy      map[string]any
	CreatedBy           string
	Recipients          []DistributionRecipientInput
	// RequestInput carries the workflow-owned, exact request contract for each
	// TO recipient. Distribution scope, form revision, deadline and actor still
	// come from this command and are checked before persistence.
	RequestInput *CreateRequestInput
}

type ResponseWorkspace struct {
	ID             string                  `json:"id"`
	TenantID       string                  `json:"tenant_id"`
	LegalEntityID  string                  `json:"legal_entity_id"`
	DistributionID string                  `json:"distribution_id"`
	Status         ResponseWorkspaceStatus `json:"status"`
	Version        int64                   `json:"version"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type ResponseRevision struct {
	ID                   string
	TenantID             string
	LegalEntityID        string
	DistributionID       string
	WorkspaceID          string
	SubmissionID         string
	Revision             int64
	SupersedesRevisionID string
	AchievedAssurance    AccessAssurance
	SignoffSummary       map[string]any
	Score                *ResponseScoreResult
	ComplianceScore      *float64
	ScoredWeightCoverage float64
	State                ResponseRevisionState
	CriticalFieldResults []map[string]any
	ScoringPolicyVersion string
	Current              bool
	CreatedAt            time.Time
}

type ResponseScoreState string

const (
	ResponseScoreNotConfigured ResponseScoreState = "NOT_CONFIGURED"
	ResponseScoreFinal         ResponseScoreState = "FINAL"
	ResponseScoreProvisional   ResponseScoreState = "PROVISIONAL"
	ResponseScoreFailed        ResponseScoreState = "FAILED"
)

type ResponseScoreResult struct {
	Mode                formcontract.ScoringMode          `json:"mode,omitempty"`
	Direction           formcontract.ScoreDirection       `json:"direction,omitempty"`
	RawScore            *float64                          `json:"raw_score,omitempty"`
	AdverseScore        *float64                          `json:"adverse_score,omitempty"`
	Band                formcontract.ConcernBand          `json:"band,omitempty"`
	Coverage            float64                           `json:"coverage"`
	Final               bool                              `json:"final"`
	State               ResponseScoreState                `json:"state"`
	ProfileVersion      string                            `json:"profile_version,omitempty"`
	ProfileChecksum     string                            `json:"profile_checksum,omitempty"`
	EvaluatorVersion    string                            `json:"evaluator_version,omitempty"`
	FailureCode         string                            `json:"failure_code,omitempty"`
	CalculatedAt        time.Time                         `json:"calculated_at,omitempty"`
	ContributionResults []formcontract.ContributionResult `json:"contribution_results,omitempty"`
	RuleResults         []formcontract.AdvancedRuleResult `json:"rule_results,omitempty"`
}

type DistributionBundle struct {
	Distribution FormDistribution
	Recipients   []DistributionRecipient
	Workspace    ResponseWorkspace
}

package evidence

import "time"

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
	ID            string
	DistributionID string
	TenantID      string
	LegalEntityID string
	Role          RecipientRole
	Type          RecipientType
	PrincipalID   string
	RequestID     string
	AudienceHint  string
	ContactLabel  string
	State         DistributionRecipientState
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
	Deadline            time.Time
	RouteExpiresAt      time.Time
	ReminderPolicy      map[string]any
	CreatedBy           string
	Recipients          []DistributionRecipientInput
}

type ResponseWorkspace struct {
	ID            string
	TenantID      string
	LegalEntityID string
	DistributionID string
	Status        ResponseWorkspaceStatus
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ResponseRevision struct {
	ID                    string
	TenantID              string
	LegalEntityID         string
	DistributionID        string
	WorkspaceID           string
	SubmissionID          string
	Revision              int64
	SupersedesRevisionID  string
	AchievedAssurance     AccessAssurance
	SignoffSummary        map[string]any
	ComplianceScore       *float64
	ScoredWeightCoverage  float64
	State                 ResponseRevisionState
	CriticalFieldResults  []map[string]any
	ScoringPolicyVersion  string
	Current               bool
	CreatedAt             time.Time
}

type DistributionBundle struct {
	Distribution FormDistribution
	Recipients   []DistributionRecipient
	Workspace    ResponseWorkspace
}

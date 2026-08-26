package evidence

import (
	"encoding/json"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type SourceType string
type SourceHealth string
type SourceStatus string

const (
	SourceRegulatory SourceType = "REGULATORY"
	SourceSystem     SourceType = "SYSTEM"
	SourceDocument   SourceType = "DOCUMENT"
	SourceHuman      SourceType = "HUMAN"
	SourceVendor     SourceType = "VENDOR"

	HealthUnknown     SourceHealth = "UNKNOWN"
	HealthCurrent     SourceHealth = "CURRENT"
	HealthDegraded    SourceHealth = "DEGRADED"
	HealthStale       SourceHealth = "STALE"
	HealthUnavailable SourceHealth = "UNAVAILABLE"

	SourceActive  SourceStatus = "ACTIVE"
	SourcePaused  SourceStatus = "PAUSED"
	SourceRetired SourceStatus = "RETIRED"
)

type Source struct {
	ID                       string       `json:"id"`
	TenantID                 string       `json:"tenant_id"`
	LegalEntityID            string       `json:"legal_entity_id,omitempty"`
	Code                     string       `json:"code"`
	Name                     string       `json:"name"`
	Type                     SourceType   `json:"type"`
	AuthorityClass           string       `json:"authority_class"`
	OwnerPrincipalID         string       `json:"owner_principal_id,omitempty"`
	Endpoint                 string       `json:"-"`
	ExpectedFreshnessMinutes int          `json:"expected_freshness_minutes"`
	LastObservedAt           *time.Time   `json:"last_observed_at,omitempty"`
	LastSuccessAt            *time.Time   `json:"last_success_at,omitempty"`
	Health                   SourceHealth `json:"health"`
	Status                   SourceStatus `json:"status"`
	Version                  int64        `json:"version"`
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}

type CreateSourceInput struct {
	TenantID                 string     `json:"tenant_id"`
	LegalEntityID            string     `json:"legal_entity_id,omitempty"`
	Code                     string     `json:"code"`
	Name                     string     `json:"name"`
	Type                     SourceType `json:"type"`
	AuthorityClass           string     `json:"authority_class"`
	OwnerPrincipalID         string     `json:"owner_principal_id,omitempty"`
	Endpoint                 string     `json:"endpoint,omitempty"`
	ExpectedFreshnessMinutes int        `json:"expected_freshness_minutes"`
}

type SourceObservation struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	SourceID          string                 `json:"source_id"`
	Scope             SourceObservationScope `json:"scope,omitempty"`
	ConnectionID      string                 `json:"connection_id,omitempty"`
	ConnectionVersion int64                  `json:"connection_version,omitempty"`
	ViewID            string                 `json:"view_id,omitempty"`
	ViewVersion       int64                  `json:"view_version,omitempty"`
	BindingID         string                 `json:"binding_id,omitempty"`
	BindingVersion    int64                  `json:"binding_version,omitempty"`
	ObservedAt        time.Time              `json:"observed_at"`
	Success           bool                   `json:"success"`
	Unavailable       bool                   `json:"unavailable"`
	LatencyMS         int                    `json:"latency_ms,omitempty"`
	Detail            string                 `json:"detail,omitempty"`
	RecordedBy        string                 `json:"recorded_by,omitempty"`
}

type RequestStatus string
type RecipientType string
type RecipientState string

const (
	RequestDraft      RequestStatus = "DRAFT"
	RequestReady      RequestStatus = "READY"
	RequestInProgress RequestStatus = "IN_PROGRESS"
	RequestSubmitted  RequestStatus = "SUBMITTED"
	RequestCancelled  RequestStatus = "CANCELLED"
	RequestExpired    RequestStatus = "EXPIRED"

	RecipientInternalPrincipal RecipientType = "INTERNAL_PRINCIPAL"
	RecipientExternalAudience  RecipientType = "EXTERNAL_AUDIENCE"

	RecipientStateAssigned             RecipientState = "ASSIGNED"
	RecipientStateReassignmentRequired RecipientState = "REASSIGNMENT_REQUIRED"
	RecipientStateLegacyUnassigned     RecipientState = "LEGACY_UNASSIGNED"
)

type BindingUseMode string
type LookupValueSource string
type SourceResolutionState string
type AnswerOrigin string

const (
	BindingUsePrefill  BindingUseMode = "PREFILL"
	BindingUseOptions  BindingUseMode = "OPTIONS"
	BindingUseValidate BindingUseMode = "VALIDATE"
	BindingUseEvidence BindingUseMode = "EVIDENCE"

	LookupValueSubjectID LookupValueSource = "SUBJECT_ID"
	LookupValueKnownFact LookupValueSource = "KNOWN_FACT"

	SourceResolutionCurrent     SourceResolutionState = "CURRENT"
	SourceResolutionPartial     SourceResolutionState = "PARTIAL"
	SourceResolutionStale       SourceResolutionState = "STALE"
	SourceResolutionNotFound    SourceResolutionState = "NOT_FOUND"
	SourceResolutionAmbiguous   SourceResolutionState = "AMBIGUOUS"
	SourceResolutionSchemaDrift SourceResolutionState = "SCHEMA_DRIFT"
	SourceResolutionUnavailable SourceResolutionState = "UNAVAILABLE"
	SourceResolutionInvalid     SourceResolutionState = "INVALID"

	AnswerSourcePrefilled     AnswerOrigin = "SOURCE_PREFILLED"
	AnswerRespondentEntered   AnswerOrigin = "RESPONDENT_ENTERED"
	AnswerRespondentCorrected AnswerOrigin = "RESPONDENT_CORRECTED"
)

type LookupValueReference struct {
	Source LookupValueSource `json:"source"`
	Key    string            `json:"key,omitempty"`
}

type FieldBindingReference struct {
	BindingID      string                `json:"binding_id"`
	BindingVersion int64                 `json:"binding_version"`
	Mode           BindingUseMode        `json:"mode"`
	ValueField     string                `json:"value_field,omitempty"`
	LookupValue    *LookupValueReference `json:"lookup_value,omitempty"`
}

type SourceResolution struct {
	Mode           BindingUseMode                 `json:"mode"`
	BindingID      string                         `json:"binding_id"`
	BindingVersion int64                          `json:"binding_version"`
	BindingName    string                         `json:"binding_name"`
	SourceID       string                         `json:"source_id"`
	State          SourceResolutionState          `json:"state"`
	Value          *sourceaccess.Scalar           `json:"value,omitempty"`
	Records        []sourceaccess.Record          `json:"records,omitempty"`
	Receipt        *sourceaccess.OperationReceipt `json:"receipt,omitempty"`
	FailureCode    string                         `json:"failure_code,omitempty"`
}

type RequestBindingReference struct {
	BindingID      string               `json:"binding_id"`
	BindingVersion int64                `json:"binding_version"`
	LookupValue    LookupValueReference `json:"lookup_value"`
	Resolution     *SourceResolution    `json:"resolution,omitempty"`
}

type AnswerProvenance struct {
	Origin         AnswerOrigin                   `json:"origin"`
	BindingID      string                         `json:"binding_id,omitempty"`
	BindingVersion int64                          `json:"binding_version,omitempty"`
	SourceValue    *sourceaccess.Scalar           `json:"source_value,omitempty"`
	SourceReceipt  *sourceaccess.OperationReceipt `json:"source_receipt,omitempty"`
	Validations    []SourceResolution             `json:"validations,omitempty"`
}

type Field struct {
	ID                string                  `json:"id"`
	Label             string                  `json:"label"`
	Type              string                  `json:"type"`
	Required          bool                    `json:"required"`
	Description       string                  `json:"description,omitempty"`
	Options           []string                `json:"options,omitempty"`
	AcceptedFormats   []string                `json:"accepted_formats,omitempty"`
	Bindings          []FieldBindingReference `json:"bindings,omitempty"`
	SourceResolutions []SourceResolution      `json:"source_resolutions,omitempty"`
}

type Recipient struct {
	Type         RecipientType  `json:"type,omitempty"`
	PrincipalID  string         `json:"principal_id,omitempty"`
	AudienceHint string         `json:"audience_hint,omitempty"`
	State        RecipientState `json:"state,omitempty"`
	Revision     int64          `json:"revision,omitempty"`
	IssueReason  string         `json:"issue_reason,omitempty"`
	AudienceHash []byte         `json:"-"`
}

type RecipientInput struct {
	Type        RecipientType `json:"type"`
	PrincipalID string        `json:"principal_id,omitempty"`
	Audience    string        `json:"audience,omitempty"`
}

type Request struct {
	ID                    string                    `json:"id"`
	TenantID              string                    `json:"tenant_id"`
	LegalEntityID         string                    `json:"legal_entity_id"`
	SubjectType           string                    `json:"subject_type"`
	SubjectID             string                    `json:"subject_id"`
	Title                 string                    `json:"title"`
	Purpose               string                    `json:"purpose"`
	WhyYou                string                    `json:"why_you"`
	Sensitivity           string                    `json:"sensitivity"`
	AudienceType          string                    `json:"audience_type"`
	Recipient             Recipient                 `json:"recipient"`
	EstimatedMinutes      int                       `json:"estimated_minutes"`
	Deadline              time.Time                 `json:"deadline"`
	KnownFacts            map[string]string         `json:"known_facts"`
	Fields                []Field                   `json:"fields"`
	SourceBindings        []RequestBindingReference `json:"source_bindings,omitempty"`
	FormTemplateID        string                    `json:"form_template_id,omitempty"`
	FormTemplateVersion   int64                     `json:"form_template_version,omitempty"`
	CollectionPeriodStart *time.Time                `json:"collection_period_start,omitempty"`
	CollectionPeriodEnd   *time.Time                `json:"collection_period_end,omitempty"`
	Status                RequestStatus             `json:"status"`
	CreatedBy             string                    `json:"created_by,omitempty"`
	Version               int64                     `json:"version"`
	CreatedAt             time.Time                 `json:"created_at"`
	UpdatedAt             time.Time                 `json:"updated_at"`
}

type CreateRequestInput struct {
	TenantID              string                    `json:"tenant_id"`
	LegalEntityID         string                    `json:"legal_entity_id,omitempty"`
	SubjectType           string                    `json:"subject_type"`
	SubjectID             string                    `json:"subject_id"`
	Title                 string                    `json:"title"`
	Purpose               string                    `json:"purpose"`
	WhyYou                string                    `json:"why_you"`
	Sensitivity           string                    `json:"sensitivity"`
	AudienceType          string                    `json:"audience_type"`
	Recipient             RecipientInput            `json:"recipient"`
	EstimatedMinutes      int                       `json:"estimated_minutes"`
	Deadline              time.Time                 `json:"deadline"`
	KnownFacts            map[string]string         `json:"known_facts"`
	Fields                []Field                   `json:"fields"`
	SourceBindings        []RequestBindingReference `json:"source_bindings,omitempty"`
	FormTemplateID        string                    `json:"form_template_id,omitempty"`
	FormTemplateVersion   int64                     `json:"form_template_version,omitempty"`
	CollectionPeriodStart *time.Time                `json:"collection_period_start,omitempty"`
	CollectionPeriodEnd   *time.Time                `json:"collection_period_end,omitempty"`
	CreatedBy             string                    `json:"created_by,omitempty"`
}

type DeclareWrongRecipientInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
	Reason           string `json:"reason"`
	ExpectedVersion  int64  `json:"expected_version"`
}

type ReassignRecipientInput struct {
	TenantID         string         `json:"tenant_id"`
	LegalEntityID    string         `json:"legal_entity_id,omitempty"`
	RequestID        string         `json:"request_id"`
	ActorPrincipalID string         `json:"actor_principal_id,omitempty"`
	Recipient        RecipientInput `json:"recipient"`
	Reason           string         `json:"reason"`
	ExpectedVersion  int64          `json:"expected_version"`
}

type Submission struct {
	ID               string                      `json:"id"`
	TenantID         string                      `json:"tenant_id"`
	LegalEntityID    string                      `json:"legal_entity_id,omitempty"`
	RequestID        string                      `json:"request_id"`
	SessionID        string                      `json:"session_id,omitempty"`
	SubmittedBy      string                      `json:"submitted_by,omitempty"`
	Channel          string                      `json:"channel"`
	Answers          map[string]string           `json:"answers"`
	AnswerProvenance map[string]AnswerProvenance `json:"answer_provenance,omitempty"`
	ExpectedVersion  int64                       `json:"expected_version"`
	SubmittedAt      time.Time                   `json:"submitted_at"`
}

type SubmissionReceipt struct {
	SubmissionID string        `json:"submission_id"`
	RequestID    string        `json:"request_id"`
	Status       RequestStatus `json:"status"`
	SubmittedAt  time.Time     `json:"submitted_at"`
	Version      int64         `json:"version"`
}

type Invitation struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	RequestID      string     `json:"request_id"`
	TokenHash      []byte     `json:"-"`
	AudienceHash   []byte     `json:"-"`
	AudienceHint   string     `json:"audience_hint"`
	Purpose        string     `json:"purpose"`
	ExpiresAt      time.Time  `json:"expires_at"`
	MaxRedemptions int        `json:"max_redemptions"`
	Redemptions    int        `json:"redemptions"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedBy      string     `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type IssueInvitationInput struct {
	TenantID      string        `json:"tenant_id"`
	LegalEntityID string        `json:"legal_entity_id,omitempty"`
	RequestID     string        `json:"request_id"`
	Audience      string        `json:"audience"`
	Purpose       string        `json:"purpose"`
	TTL           time.Duration `json:"-"`
	TTLMinutes    int           `json:"ttl_minutes"`
	CreatedBy     string        `json:"created_by,omitempty"`
}

type IssuedInvitation struct {
	InvitationID string    `json:"invitation_id"`
	Token        string    `json:"token"`
	AudienceHint string    `json:"audience_hint"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// InvitationMetadata is safe for requester administration responses. Token
// material, token hashes, audience hashes, and raw audience addresses are
// deliberately not part of this type.
type InvitationMetadata struct {
	ID             string     `json:"id"`
	RequestID      string     `json:"request_id"`
	AudienceHint   string     `json:"audience_hint"`
	Purpose        string     `json:"purpose"`
	ExpiresAt      time.Time  `json:"expires_at"`
	MaxRedemptions int        `json:"max_redemptions"`
	Redemptions    int        `json:"redemptions"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ManageInvitationsInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
}

type RevokeInvitationAsRequesterInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	InvitationID     string `json:"invitation_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
}

type RevokeSessionAsRequesterInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	SessionID        string `json:"session_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
}

type ReplaceInvitationInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	InvitationID     string `json:"invitation_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
	Audience         string `json:"audience"`
	Purpose          string `json:"purpose"`
	TTLMinutes       int    `json:"ttl_minutes"`
}

type Session struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	RequestID    string     `json:"request_id"`
	InvitationID string     `json:"-"`
	AudienceHint string     `json:"audience_hint"`
	TokenHash    []byte     `json:"-"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type RedeemedSession struct {
	SessionID    string    `json:"session_id"`
	SessionToken string    `json:"session_token"`
	RequestID    string    `json:"request_id"`
	AudienceHint string    `json:"audience_hint"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type RedeemInput struct {
	InvitationTokenHash []byte
	AudienceHash        []byte
	SessionTokenHash    []byte
	SessionID           string
	Now                 time.Time
	SessionExpiresAt    time.Time
}

type ArtifactStatus string

const (
	ArtifactStoredUnscanned ArtifactStatus = "STORED_UNSCANNED"
	ArtifactAvailable       ArtifactStatus = "AVAILABLE"
	ArtifactQuarantined     ArtifactStatus = "QUARANTINED"
	ArtifactDeleted         ArtifactStatus = "DELETED"
)

type Artifact struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	RequestID    string         `json:"request_id"`
	SubmissionID string         `json:"submission_id,omitempty"`
	FileName     string         `json:"file_name"`
	MediaType    string         `json:"media_type"`
	SizeBytes    int64          `json:"size_bytes"`
	SHA256       string         `json:"sha256"`
	StorageKey   string         `json:"storage_key"`
	Status       ArtifactStatus `json:"status"`
	CreatedBy    string         `json:"created_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type ArtifactInput struct {
	TenantID     string
	RequestID    string
	SubmissionID string
	FileName     string
	MediaType    string
	CreatedBy    string
	SessionToken string
}

type ObjectInfo struct {
	Key       string
	SizeBytes int64
	SHA256    string
}

type sourceHealthChange struct {
	SourceID string
	From     SourceHealth
	To       SourceHealth
}

type JSONMap = map[string]json.RawMessage

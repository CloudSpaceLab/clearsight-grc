package thirdparty

import "time"

type AssessmentReviewKind string

const AssessmentReviewOnboarding AssessmentReviewKind = "ONBOARDING"

type AssessmentStatus string

const (
	AssessmentSetupPending AssessmentStatus = "SETUP_PENDING"
	AssessmentReadyToSend  AssessmentStatus = "READY_TO_SEND"
	AssessmentCollecting   AssessmentStatus = "COLLECTING"
	AssessmentSubmitted    AssessmentStatus = "SUBMITTED"
	AssessmentUnderReview  AssessmentStatus = "UNDER_REVIEW"
	AssessmentCompleted    AssessmentStatus = "COMPLETED"
	AssessmentCancelled    AssessmentStatus = "CANCELLED"
)

type AssessmentConclusion string

const (
	AssessmentSatisfactory               AssessmentConclusion = "SATISFACTORY"
	AssessmentSatisfactoryWithConditions AssessmentConclusion = "SATISFACTORY_WITH_CONDITIONS"
	AssessmentUnsatisfactory             AssessmentConclusion = "UNSATISFACTORY"
	AssessmentInconclusive               AssessmentConclusion = "INCONCLUSIVE"
)

type AssessmentRequestPurpose string

const (
	AssessmentRequestInitial       AssessmentRequestPurpose = "INITIAL"
	AssessmentRequestClarification AssessmentRequestPurpose = "CLARIFICATION"
	AssessmentRequestOrigin                                 = "THIRD_PARTY_ASSESSMENT"
)

type Assessment struct {
	ID                      string               `json:"id"`
	TenantID                string               `json:"tenant_id"`
	LegalEntityID           string               `json:"legal_entity_id"`
	RelationshipID          string               `json:"relationship_id"`
	ReviewKind              AssessmentReviewKind `json:"review_kind"`
	StableEpisodeKey        string               `json:"stable_episode_key"`
	Status                  AssessmentStatus     `json:"status"`
	FormTemplateID          string               `json:"form_template_id"`
	FormTemplateVersion     int64                `json:"form_template_version"`
	CurrentRequestID        string               `json:"current_request_id,omitempty"`
	SubmissionID            string               `json:"submission_id,omitempty"`
	ReviewMatterID          string               `json:"review_matter_id,omitempty"`
	ReviewDueAt             time.Time            `json:"review_due_at"`
	StartedByPrincipalID    string               `json:"started_by_principal_id"`
	StartedAt               time.Time            `json:"started_at"`
	SubmittedAt             *time.Time           `json:"submitted_at,omitempty"`
	ReviewStartedAt         *time.Time           `json:"review_started_at,omitempty"`
	CompletedAt             *time.Time           `json:"completed_at,omitempty"`
	ReviewerPrincipalID     string               `json:"reviewer_principal_id,omitempty"`
	Conclusion              AssessmentConclusion `json:"conclusion,omitempty"`
	ConclusionUncertainty   string               `json:"conclusion_uncertainty,omitempty"`
	ConclusionRationale     string               `json:"conclusion_rationale,omitempty"`
	NextReviewRecommendedAt *time.Time           `json:"next_review_recommended_at,omitempty"`
	CancellationReason      string               `json:"cancellation_reason,omitempty"`
	Version                 int64                `json:"version"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
}

type AssessmentRequestLink struct {
	TenantID       string                   `json:"tenant_id"`
	LegalEntityID  string                   `json:"legal_entity_id"`
	AssessmentID   string                   `json:"assessment_id"`
	RequestID      string                   `json:"request_id"`
	Purpose        AssessmentRequestPurpose `json:"purpose"`
	Sequence       int                      `json:"sequence"`
	OriginType     string                   `json:"origin_type"`
	OriginID       string                   `json:"origin_id"`
	OriginSequence int                      `json:"origin_sequence"`
	InvitationID   string                   `json:"invitation_id"`
	CreatedAt      time.Time                `json:"created_at"`
}

type AssessmentListFilter struct {
	Scope
	Status AssessmentStatus `json:"status,omitempty"`
	Limit  int              `json:"limit,omitempty"`
	Cursor string           `json:"cursor,omitempty"`
}

type AssessmentPage struct {
	Items      []Assessment `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type AssessmentEvent struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	AssessmentID      string                 `json:"assessment_id"`
	AssessmentVersion int64                  `json:"assessment_version"`
	Type              string                 `json:"type"`
	Payload           map[string]interface{} `json:"payload"`
	OccurredAt        time.Time              `json:"occurred_at"`
}

type StartAssessmentInput struct {
	RelationshipVersion int64     `json:"relationship_version"`
	FormTemplateID      string    `json:"form_template_id"`
	FormTemplateVersion int64     `json:"form_template_version"`
	ReviewDueAt         time.Time `json:"review_due_at"`
}

type AssessmentSetupCompletedInput struct {
	Scope
	AssessmentID    string `json:"assessment_id"`
	ExpectedVersion int64  `json:"expected_version"`
	CausationID     string `json:"causation_id"`
	SetupJobID      string `json:"setup_job_id"`
	ReviewMatterID  string `json:"review_matter_id"`
}

type AssessmentSubmittedInput struct {
	Scope
	AssessmentID    string `json:"assessment_id"`
	ExpectedVersion int64  `json:"expected_version"`
	CausationID     string `json:"causation_id"`
	EventID         string `json:"event_id"`
	RequestID       string `json:"request_id"`
	SubmissionID    string `json:"submission_id"`
}

type RecordRequestIssuedInput struct {
	ExpectedVersion int64                    `json:"expected_version"`
	RequestID       string                   `json:"request_id"`
	Purpose         AssessmentRequestPurpose `json:"purpose"`
	OriginType      string                   `json:"origin_type"`
	OriginID        string                   `json:"origin_id"`
	OriginSequence  int                      `json:"origin_sequence"`
	InvitationID    string                   `json:"invitation_id"`
}

type RecordRequestIssuedOutcome struct {
	Assessment Assessment            `json:"assessment"`
	Link       AssessmentRequestLink `json:"link"`
}

type CompleteAssessmentInput struct {
	ExpectedVersion         int64                `json:"expected_version"`
	Conclusion              AssessmentConclusion `json:"conclusion"`
	Uncertainty             string               `json:"uncertainty,omitempty"`
	Rationale               string               `json:"rationale"`
	NextReviewRecommendedAt *time.Time           `json:"next_review_recommended_at,omitempty"`
}

type CancelAssessmentInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason"`
}

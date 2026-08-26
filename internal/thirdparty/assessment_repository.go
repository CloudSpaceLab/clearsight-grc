package thirdparty

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

var (
	ErrInvalidAssessmentTransition    = errors.New("invalid third-party assessment transition")
	ErrAssessmentAuthorityUnavailable = errors.New("third-party assessment authority service is unavailable")
	ErrAssessmentIdentityMismatch     = errors.New("assessment command identity does not match verified context")
)

type AssessmentRepository interface {
	Repository
	CreateAssessment(context.Context, CreateAssessmentRecord) (Assessment, error)
	GetAssessment(context.Context, Scope, string) (Assessment, error)
	GetCurrentAssessment(context.Context, Scope, string, AssessmentReviewKind) (Assessment, error)
	ListAssessments(context.Context, AssessmentListFilter) (AssessmentPage, error)
	TransitionAssessment(context.Context, AssessmentTransitionRecord) (Assessment, error)
	ApplyAssessmentReaction(context.Context, AssessmentReactionRecord) (Assessment, error)
	RequeueAssessmentSetup(context.Context, RequeueAssessmentSetupRecord) (AssessmentSetupJob, Assessment, error)
	PrepareAssessmentRequest(context.Context, PrepareAssessmentRequestRecord) (AssessmentRequestLink, Assessment, error)
	RecordRequestIssued(context.Context, RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error)
	GetCurrentAssessmentRequestLink(context.Context, Scope, string) (AssessmentRequestLink, error)
	PrepareRequestReissue(context.Context, PrepareRequestReissueRecord) (AssessmentRequestLink, Assessment, error)
	FinalizeRequestReissue(context.Context, FinalizeRequestReissueRecord) (AssessmentRequestLink, Assessment, error)
	ListAssessmentRequestLinks(context.Context, Scope, string) ([]AssessmentRequestLink, error)
	ListAssessmentMatterLinks(context.Context, Scope, string, int) ([]AssessmentMatterLink, error)
	ResolveAssessmentRequest(context.Context, string, evidence.RequestOrigin, string) (AssessmentSubmissionTarget, error)
}

type AssessmentReactionKind string

const (
	AssessmentReactionSetupCompleted AssessmentReactionKind = "SETUP_COMPLETED"
	AssessmentReactionSubmitted      AssessmentReactionKind = "SUBMITTED"
)

type CreateAssessmentRecord struct {
	Scope
	RelationshipID      string
	RelationshipVersion int64
	Assessment          Assessment
}

type AssessmentTransitionRecord struct {
	Scope
	ID                      string
	ExpectedVersion         int64
	From                    []AssessmentStatus
	To                      AssessmentStatus
	At                      time.Time
	ActorPrincipalID        string
	ReviewMatterID          string
	Conclusion              AssessmentConclusion
	ConclusionUncertainty   string
	ConclusionRationale     string
	NextReviewRecommendedAt *time.Time
	CancellationReason      string
}

type AssessmentReactionRecord struct {
	Scope
	AssessmentID    string
	ExpectedVersion int64
	Kind            AssessmentReactionKind
	CausationID     string
	JobID           string
	EventID         string
	MatterID        string
	RequestID       string
	SubmissionID    string
	At              time.Time
}

type RequeueAssessmentSetupRecord struct {
	Scope
	AssessmentID     string
	ExpectedVersion  int64
	ActorPrincipalID string
	QueuedAt         time.Time
}

type RecordRequestIssuedRecord struct {
	Scope
	AssessmentID     string
	ExpectedVersion  int64
	ActorPrincipalID string
	RequestID        string
	Purpose          AssessmentRequestPurpose
	OriginType       string
	OriginID         string
	OriginSequence   int
	InvitationID     string
	IssuedAt         time.Time
}

type PrepareAssessmentRequestRecord struct {
	Scope
	AssessmentID     string
	ExpectedVersion  int64
	ActorPrincipalID string
	RequestID        string
	Purpose          AssessmentRequestPurpose
	OriginType       string
	OriginID         string
	OriginSequence   int
	PreparedAt       time.Time
}

type PrepareRequestReissueRecord struct {
	Scope
	AssessmentID         string
	ExpectedVersion      int64
	ActorPrincipalID     string
	RequestID            string
	ExpectedInvitationID string
	PreparedAt           time.Time
}

type FinalizeRequestReissueRecord struct {
	Scope
	AssessmentID     string
	ExpectedVersion  int64
	ActorPrincipalID string
	RequestID        string
	InvitationID     string
	ReissuedAt       time.Time
}

package thirdparty

import (
	"context"
	"errors"
	"time"
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
	TransitionAssessment(context.Context, AssessmentTransitionRecord) (Assessment, error)
	ApplyAssessmentReaction(context.Context, AssessmentReactionRecord) (Assessment, error)
	RecordRequestIssued(context.Context, RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error)
	ListAssessmentRequestLinks(context.Context, Scope, string) ([]AssessmentRequestLink, error)
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

type RecordRequestIssuedRecord struct {
	Scope
	AssessmentID    string
	ExpectedVersion int64
	RequestID       string
	Purpose         AssessmentRequestPurpose
	OriginType      string
	OriginID        string
	OriginSequence  int
	InvitationID    string
	IssuedAt        time.Time
}

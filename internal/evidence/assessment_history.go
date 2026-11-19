package evidence

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type AssessmentRevisionSummary struct {
	ID           string    `json:"id"`
	Revision     int64     `json:"revision"`
	CorrectionOf string    `json:"correction_of,omitempty"`
	SubmittedAt  time.Time `json:"submitted_at"`
	SubmittedBy  string    `json:"submitted_by,omitempty"`
	ReviewStatus string    `json:"review_status,omitempty"`
	CanRead      bool      `json:"can_read"`
}

type AssessmentHistoryQuery struct {
	ProgramID, RequestID, SubmissionID string
	BeforeRevision, BeforeReview       int64
}

type AssessmentHistory struct {
	Revisions           []AssessmentRevisionSummary `json:"revisions"`
	NextBeforeRevision  int64                       `json:"next_before_revision,omitempty"`
	LatestReviewVersion int64                       `json:"latest_review_version"`
	CanReview           bool                        `json:"can_review"`
	Correction          *AssessmentCorrectionWindow `json:"correction,omitempty"`
	Submission          *Submission                 `json:"submission,omitempty"`
	Reviews             []SubmissionReview          `json:"reviews,omitempty"`
	NextBeforeReview    int64                       `json:"next_before_review,omitempty"`
}

type assessmentHistoryStore interface {
	AssessmentHistory(context.Context, identity.Actor, AssessmentHistoryQuery) (AssessmentHistory, error)
}

func (s *Service) ProgramAssessmentHistory(ctx context.Context, query AssessmentHistoryQuery) (AssessmentHistory, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return AssessmentHistory{}, ErrRecipientMismatch
	}
	store, ok := s.repo.(assessmentHistoryStore)
	if !ok {
		return AssessmentHistory{}, ErrNotFound
	}
	return store.AssessmentHistory(ctx, actor, query)
}

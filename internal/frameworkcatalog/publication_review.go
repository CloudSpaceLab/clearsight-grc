package frameworkcatalog

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var ErrPublicationReview = errors.New("framework publication review is invalid")

type PublicationReviewStatus string

const (
	PublicationReviewDraft    PublicationReviewStatus = "DRAFT"
	PublicationReviewPending  PublicationReviewStatus = "PENDING"
	PublicationReviewApproved PublicationReviewStatus = "APPROVED"
	PublicationReviewRejected PublicationReviewStatus = "REJECTED"
)

// PublicationReview binds the exact verified release to its maker and independent
// reviewer. This is domain state only: callers must bind verified identity,
// resolve current platform authority, and persist changes and audit atomically.
type PublicationReview struct {
	Publication   Publication             `json:"publication"`
	ReleaseSHA256 string                  `json:"release_sha256"`
	SourceBytes   int64                   `json:"source_bytes"`
	MakerID       string                  `json:"maker_id"`
	ReviewerID    string                  `json:"reviewer_id,omitempty"`
	Rationale     string                  `json:"rationale,omitempty"`
	Status        PublicationReviewStatus `json:"status"`
	Version       int64                   `json:"version"`
	PreparedAt    time.Time               `json:"prepared_at"`
	DecidedAt     *time.Time              `json:"decided_at,omitempty"`
}

// PreparePublicationReview verifies the supplied source bytes against the
// manifest. Source verification is not a review, rights grant or approval.
func PreparePublicationReview(manifest, source io.Reader, maker string, now time.Time) (PublicationReview, error) {
	maker = strings.TrimSpace(maker)
	if maker == "" || len(maker) > 128 || now.IsZero() {
		return PublicationReview{}, ErrPublicationReview
	}
	p, digest, size, err := VerifyPublicationSource(manifest, source)
	if err != nil {
		return PublicationReview{}, err
	}
	return PublicationReview{Publication: p, ReleaseSHA256: digest, SourceBytes: size, MakerID: maker, Status: PublicationReviewDraft, Version: 1, PreparedAt: now.UTC()}, nil
}

// SubmitPublicationReview freezes the validated release digest for review.
func SubmitPublicationReview(current PublicationReview, maker string, expectedVersion int64) (PublicationReview, error) {
	if current.Status != PublicationReviewDraft || current.Version != expectedVersion || strings.TrimSpace(maker) == "" || strings.TrimSpace(maker) != current.MakerID || current.SourceBytes <= 0 {
		return PublicationReview{}, ErrPublicationReview
	}
	digest, err := ValidatePublication(current.Publication)
	if err != nil || digest != current.ReleaseSHA256 {
		return PublicationReview{}, fmt.Errorf("%w: release changed after verification", ErrPublicationReview)
	}
	current.Status = PublicationReviewPending
	current.Version++
	return current, nil
}

// DecidePublicationReview accepts only a different reviewer and a rationale for
// the same source and content that the maker submitted. It does not publish.
func DecidePublicationReview(current PublicationReview, reviewer, rationale string, expectedVersion int64, approve bool, now time.Time) (PublicationReview, error) {
	reviewer, rationale = strings.TrimSpace(reviewer), strings.TrimSpace(rationale)
	if current.Status != PublicationReviewPending || current.Version != expectedVersion || reviewer == "" || len(reviewer) > 128 || reviewer == current.MakerID || rationale == "" || len(rationale) > 2000 || now.IsZero() || now.Before(current.PreparedAt) || current.SourceBytes <= 0 {
		return PublicationReview{}, ErrPublicationReview
	}
	digest, err := ValidatePublication(current.Publication)
	if err != nil || digest != current.ReleaseSHA256 {
		return PublicationReview{}, fmt.Errorf("%w: submitted release changed", ErrPublicationReview)
	}
	if approve {
		current.Status = PublicationReviewApproved
	} else {
		current.Status = PublicationReviewRejected
	}
	current.ReviewerID, current.Rationale = reviewer, rationale
	decision := now.UTC()
	current.DecidedAt = &decision
	current.Version++
	return current, nil
}

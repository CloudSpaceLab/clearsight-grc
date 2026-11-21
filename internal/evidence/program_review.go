package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var ErrProgramReviewUnavailable = errors.New("program reviews unavailable")

type ProgramReviewQuery struct {
	TenantID      string
	LegalEntityID string
	Cursor        string
	Limit         int
}

type ProgramReviewPage struct {
	Items      []CompletedResponseSummary `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
	Coverage   string                     `json:"coverage"`
}

type programReviewCursor struct {
	Scope       string    `json:"scope"`
	CompletedAt time.Time `json:"completed_at"`
	ID          string    `json:"id"`
}

type programReviewRepository interface {
	listPendingProgramReviews(context.Context, identity.Actor, ProgramReviewQuery) (ProgramReviewPage, error)
}

func programReviewScope(actor identity.Actor, entity string) string {
	sum := sha256.Sum256([]byte(actor.TenantID + "\x00" + entity + "\x00" + actor.PrincipalID))
	return hex.EncodeToString(sum[:])
}

func parseProgramReviewCursor(value string, actor identity.Actor, entity string) (programReviewCursor, error) {
	if value == "" {
		return programReviewCursor{}, nil
	}
	if len(value) > 1024 {
		return programReviewCursor{}, ErrDistributionInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return programReviewCursor{}, ErrDistributionInvalid
	}
	var cursor programReviewCursor
	if json.Unmarshal(raw, &cursor) != nil || cursor.Scope != programReviewScope(actor, entity) || cursor.ID == "" || cursor.CompletedAt.IsZero() {
		return programReviewCursor{}, ErrDistributionInvalid
	}
	return cursor, nil
}

func makeProgramReviewCursor(actor identity.Actor, entity string, last CompletedResponseSummary) string {
	raw, _ := json.Marshal(programReviewCursor{Scope: programReviewScope(actor, entity), CompletedAt: last.CompletedAt, ID: last.ID})
	return base64.RawURLEncoding.EncodeToString(raw)
}

// ListPendingProgramReviews returns a bounded actor-scoped review population.
// No in-memory fallback may infer authority from a list of readable responses.
func (s *DistributionService) ListPendingProgramReviews(ctx context.Context, query ProgramReviewQuery) (ProgramReviewPage, error) {
	actor, ok := identity.FromContext(ctx)
	if s == nil || !ok || actor.Valid(s.currentTime()) != nil || actor.Kind != "PERSON" || actor.TenantID != query.TenantID || actor.LegalEntityID != query.LegalEntityID || query.LegalEntityID == "*" || strings.TrimSpace(query.TenantID) == "" || strings.TrimSpace(query.LegalEntityID) == "" || query.Limit < 1 || query.Limit > 100 {
		return ProgramReviewPage{}, ErrDistributionInvalid
	}
	if _, err := parseProgramReviewCursor(query.Cursor, actor, query.LegalEntityID); err != nil {
		return ProgramReviewPage{}, err
	}
	repo, ok := s.store.(programReviewRepository)
	if !ok || s.responseReviewAuthorizer == nil {
		return ProgramReviewPage{}, ErrProgramReviewUnavailable
	}
	return repo.listPendingProgramReviews(ctx, actor, query)
}

package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type ProgramActivityQuery struct {
	Search    string `json:"search"`
	EventType string `json:"event_type"`
	Category  string `json:"category"`
	From      string `json:"from"`
	To        string `json:"to"`
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit"`
}
type ProgramActivityEvent struct {
	ID           string         `json:"id"`
	Category     string         `json:"category"`
	RecordID     string         `json:"record_id"`
	RequestID    string         `json:"request_id,omitempty"`
	SubmissionID string         `json:"submission_id,omitempty"`
	ArtifactID   string         `json:"artifact_id,omitempty"`
	FieldID      string         `json:"field_id,omitempty"`
	Title        string         `json:"title"`
	Event        string         `json:"event"`
	Actor        string         `json:"actor,omitempty"`
	ActorName    string         `json:"actor_name"`
	At           time.Time      `json:"at"`
	Details      map[string]any `json:"details"`
}
type ProgramActivityPage struct {
	Items      []ProgramActivityEvent `json:"items"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Through    time.Time              `json:"through"`
}
type programActivityCursor struct {
	At      time.Time `json:"at"`
	ID      string    `json:"id"`
	Through time.Time `json:"through"`
	Scope   string    `json:"scope"`
}

func activityQuery(actor identity.Actor, program string, q ProgramActivityQuery, now time.Time) (ProgramActivityQuery, programActivityCursor, error) {
	q.Search = strings.TrimSpace(q.Search)
	if len(q.Search) > 200 || len(q.EventType) > 128 || q.Limit < 0 || q.Limit > 100 {
		return q, programActivityCursor{}, ErrDraftInvalid
	}
	if q.Limit == 0 {
		q.Limit = 25
	}
	switch q.Category {
	case "", "PROGRAM", "ISSUE", "ASSESSMENT", "DOCUMENT", "SCHEDULE":
	default:
		return q, programActivityCursor{}, ErrDraftInvalid
	}
	for _, d := range []string{q.From, q.To} {
		if d != "" {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return q, programActivityCursor{}, ErrDraftInvalid
			}
		}
	}
	if q.From != "" && q.To != "" && q.From > q.To {
		return q, programActivityCursor{}, ErrDraftInvalid
	}
	scope := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join([]string{actor.TenantID, actor.LegalEntityID, actor.PrincipalID, program, q.Search, q.Category, q.EventType, q.From, q.To}, "\x00"))))
	c := programActivityCursor{Through: now.UTC(), Scope: scope}
	if q.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
		if err != nil || len(raw) > 2048 {
			return q, c, ErrDraftInvalid
		}
		if json.Unmarshal(raw, &c) != nil || c.Scope != scope || c.ID == "" || c.At.IsZero() || c.Through.IsZero() || c.At.After(c.Through) || c.Through.After(now.Add(time.Minute)) {
			return q, c, ErrDraftInvalid
		}
	}
	return q, c, nil
}
func (s *Service) ProgramActivity(ctx context.Context, program string, q ProgramActivityQuery) (ProgramActivityPage, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return ProgramActivityPage{}, ErrRecipientMismatch
	}
	repo, ok := s.repo.(interface {
		ProgramActivity(context.Context, identity.Actor, string, ProgramActivityQuery, time.Time) (ProgramActivityPage, error)
	})
	if !ok {
		return ProgramActivityPage{}, ErrNotFound
	}
	return repo.ProgramActivity(ctx, actor, program, q, s.now().UTC())
}

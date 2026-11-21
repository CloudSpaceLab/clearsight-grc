package sourceaccess

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

var ErrQuestionTestRateLimit = errors.New("question test rate limit reached")

// The audit contract deliberately excludes endpoint, sample, response and
// credential fields. A reservation is durable before network access begins.
type QuestionTestActor struct{ TenantID, LegalEntityID, PrincipalID, AuthorityRuleID, AuthorityPolicyVersion string }
type QuestionTestAudit interface {
	Reserve(context.Context, QuestionTestActor, string, int64, string, string, time.Time) (string, error)
	Complete(context.Context, QuestionTestActor, string, QuestionTestOutcome, time.Time) error
}

type questionTestAttempt struct {
	id            string
	actor         QuestionTestActor
	formID        string
	version       int64
	fieldID, mode string
	when          time.Time
	finished      bool
	outcome       QuestionTestOutcome
	completedAt   time.Time
}
type MemoryQuestionTestAudit struct {
	mu       sync.Mutex
	attempts []questionTestAttempt
}

func NewMemoryQuestionTestAudit() *MemoryQuestionTestAudit { return &MemoryQuestionTestAudit{} }

func validQuestionTestAudit(actor QuestionTestActor, formID string, version int64, fieldID, mode string, when time.Time) bool {
	return strings.TrimSpace(actor.TenantID) != "" && strings.TrimSpace(actor.LegalEntityID) != "" && actor.LegalEntityID != "*" && strings.TrimSpace(actor.PrincipalID) != "" && strings.TrimSpace(actor.AuthorityRuleID) != "" && len(actor.AuthorityRuleID) <= 128 && strings.TrimSpace(actor.AuthorityPolicyVersion) != "" && len(actor.AuthorityPolicyVersion) <= 128 && !containsControl(actor.AuthorityRuleID) && !containsControl(actor.AuthorityPolicyVersion) && strings.TrimSpace(formID) != "" && version > 0 && strings.TrimSpace(fieldID) != "" && (mode == "LIVE" || mode == "SAMPLE") && !when.IsZero()
}
func (s *MemoryQuestionTestAudit) Reserve(ctx context.Context, actor QuestionTestActor, formID string, version int64, fieldID, mode string, when time.Time) (string, error) {
	if s == nil || !validQuestionTestAudit(actor, formID, version, fieldID, mode, when) {
		return "", ErrDefinitionInvalid
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entity, principal := 0, 0
	for _, attempt := range s.attempts {
		if attempt.actor.TenantID == actor.TenantID && attempt.actor.LegalEntityID == actor.LegalEntityID && attempt.when.After(when.Add(-time.Minute)) && !attempt.when.After(when) {
			entity++
			if attempt.actor.PrincipalID == actor.PrincipalID {
				principal++
			}
		}
	}
	if entity >= 60 || principal >= 10 {
		return "", ErrQuestionTestRateLimit
	}
	value, err := id.NewUUIDv7()
	if err != nil {
		return "", err
	}
	s.attempts = append(s.attempts, questionTestAttempt{id: value, actor: actor, formID: formID, version: version, fieldID: fieldID, mode: mode, when: when.UTC()})
	return value, nil
}
func (s *MemoryQuestionTestAudit) Complete(ctx context.Context, actor QuestionTestActor, attemptID string, outcome QuestionTestOutcome, when time.Time) error {
	if s == nil || ctx.Err() != nil || (outcome != QuestionTestValid && outcome != QuestionTestInvalid && outcome != QuestionTestUnavailable) || when.IsZero() {
		return ErrDefinitionInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.attempts {
		attempt := &s.attempts[i]
		if attempt.id == attemptID && attempt.actor == actor && !attempt.finished && !when.Before(attempt.when) {
			attempt.finished = true
			attempt.outcome = outcome
			attempt.completedAt = when.UTC()
			return nil
		}
	}
	return ErrDefinitionInvalid
}

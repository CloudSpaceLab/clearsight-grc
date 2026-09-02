package operations

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	QueueOutbox = "outbox-delivery"
	QueueTimers = "workflow-timers"
)

var (
	ErrRecoveryInvalid  = errors.New("background job recovery request is invalid")
	ErrRecoveryConflict = errors.New("background job is no longer the expected terminal attempt")
)

type RetryInput struct {
	TenantID         string `json:"tenant_id"`
	Queue            string `json:"queue"`
	JobID            string `json:"job_id"`
	ExpectedAttempts int    `json:"expected_attempts"`
	ActorPrincipalID string `json:"actor_principal_id"`
	Rationale        string `json:"rationale"`
}

type RecoveryReceipt struct {
	JobID            string    `json:"job_id"`
	Queue            string    `json:"queue"`
	PreviousAttempts int       `json:"previous_attempts"`
	State            string    `json:"state"`
	RetriedAt        time.Time `json:"retried_at"`
}

type RecoverySource interface {
	RetryTerminalJob(context.Context, RetryInput) (RecoveryReceipt, error)
}

func (s *Service) RetryTerminalJob(ctx context.Context, input RetryInput) (RecoveryReceipt, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Queue = strings.TrimSpace(input.Queue)
	input.JobID = strings.TrimSpace(input.JobID)
	input.ActorPrincipalID = strings.TrimSpace(input.ActorPrincipalID)
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.TenantID == "" || input.JobID == "" || input.ActorPrincipalID == "" || input.ExpectedAttempts < 1 || len(input.Rationale) < 20 || len(input.Rationale) > 2000 || (input.Queue != QueueOutbox && input.Queue != QueueTimers) {
		return RecoveryReceipt{}, ErrRecoveryInvalid
	}
	for _, source := range s.sources {
		recovery, ok := source.(RecoverySource)
		if !ok {
			continue
		}
		receipt, err := recovery.RetryTerminalJob(ctx, input)
		if errors.Is(err, ErrRecoveryConflict) {
			continue
		}
		return receipt, err
	}
	return RecoveryReceipt{}, ErrRecoveryConflict
}

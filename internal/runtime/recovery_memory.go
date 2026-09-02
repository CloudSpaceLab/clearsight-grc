package runtime

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func (r *MemoryRepository) RetryTerminalJob(_ context.Context, input operations.RetryInput) (operations.RecoveryReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	switch input.Queue {
	case operations.QueueOutbox:
		value, ok := r.outbox[input.JobID]
		if !ok || value.TenantID != input.TenantID || value.DeadLetteredAt == nil || value.Attempts != input.ExpectedAttempts {
			return operations.RecoveryReceipt{}, operations.ErrRecoveryConflict
		}
		previous := value.Attempts
		value.Attempts = 0
		value.DeadLetteredAt = nil
		value.NextAttemptAt = &now
		value.LockedBy, value.LeaseUntil = "", nil
		r.outbox[input.JobID] = value
		return operations.RecoveryReceipt{JobID: input.JobID, Queue: input.Queue, PreviousAttempts: previous, State: "READY", RetriedAt: now}, nil
	case operations.QueueTimers:
		value, ok := r.timers[input.JobID]
		if !ok || value.TenantID != input.TenantID || value.State != TimerFailed || value.FailedAt == nil || value.Attempts != input.ExpectedAttempts {
			return operations.RecoveryReceipt{}, operations.ErrRecoveryConflict
		}
		previous := value.Attempts
		value.Attempts = 0
		value.State = TimerReady
		value.FailedAt = nil
		value.DueAt = now
		value.LockedBy, value.LeaseUntil = "", nil
		r.timers[input.JobID] = value
		return operations.RecoveryReceipt{JobID: input.JobID, Queue: input.Queue, PreviousAttempts: previous, State: "READY", RetriedAt: now}, nil
	default:
		return operations.RecoveryReceipt{}, operations.ErrRecoveryInvalid
	}
}

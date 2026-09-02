package operations

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recoverySourceStub struct {
	receipt RecoveryReceipt
	err     error
	input   RetryInput
}

func (s *recoverySourceStub) BackgroundJobs(context.Context, string, int) (Snapshot, error) {
	return Snapshot{}, nil
}

func (s *recoverySourceStub) RetryTerminalJob(_ context.Context, input RetryInput) (RecoveryReceipt, error) {
	s.input = input
	return s.receipt, s.err
}

func TestRetryTerminalJobRequiresExactGovernedTarget(t *testing.T) {
	source := &recoverySourceStub{receipt: RecoveryReceipt{JobID: "job-1", Queue: QueueOutbox, PreviousAttempts: 5, State: "READY", RetriedAt: time.Now().UTC()}}
	service := NewService(source)
	input := RetryInput{TenantID: "bank", Queue: QueueOutbox, JobID: "job-1", ExpectedAttempts: 5, ActorPrincipalID: "admin-1", Rationale: "The tenant-scope defect is fixed and the original event can be delivered safely."}

	receipt, err := service.RetryTerminalJob(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.JobID != "job-1" || source.input != input {
		t.Fatalf("receipt=%#v input=%#v", receipt, source.input)
	}
}

func TestRetryTerminalJobFailsClosedForInvalidOrUnavailableTarget(t *testing.T) {
	service := NewService(&recoverySourceStub{err: ErrRecoveryConflict})
	if _, err := service.RetryTerminalJob(t.Context(), RetryInput{}); !errors.Is(err, ErrRecoveryInvalid) {
		t.Fatalf("invalid retry error = %v", err)
	}
	_, err := service.RetryTerminalJob(t.Context(), RetryInput{TenantID: "bank", Queue: QueueOutbox, JobID: "job-1", ExpectedAttempts: 5, ActorPrincipalID: "admin-1", Rationale: "The root cause was corrected before this retry."})
	if !errors.Is(err, ErrRecoveryConflict) {
		t.Fatalf("stale retry error = %v", err)
	}
}

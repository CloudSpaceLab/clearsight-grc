package runtime

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func TestMemoryRecoveryRetriesExactTerminalOutboxWithoutDeletingHistoryState(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	repository.outbox["job-1"] = OutboxEvent{ID: "job-1", TenantID: "bank", Attempts: 5, DeadLetteredAt: &now, LastError: "safe failure"}
	input := operations.RetryInput{TenantID: "bank", Queue: operations.QueueOutbox, JobID: "job-1", ExpectedAttempts: 5, ActorPrincipalID: "admin-1", Rationale: "The root cause was corrected before this retry."}

	receipt, err := repository.RetryTerminalJob(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	stored := repository.outbox["job-1"]
	if receipt.PreviousAttempts != 5 || stored.DeadLetteredAt != nil || stored.Attempts != 0 || stored.NextAttemptAt == nil || stored.LastError != "safe failure" {
		t.Fatalf("receipt=%#v stored=%#v", receipt, stored)
	}
	if _, err := repository.RetryTerminalJob(t.Context(), input); err != operations.ErrRecoveryConflict {
		t.Fatalf("duplicate recovery error = %v", err)
	}
}

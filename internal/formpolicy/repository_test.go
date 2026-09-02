package formpolicy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutionReceiptAndOpenEpisodeAreIdempotent(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	receipt := ExecutionReceipt{ID: "execution-a", TenantID: "bank", LegalEntityID: "entity", PolicyID: "policy-a", PolicyVersion: 1, AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, ResponseRevisionID: "response-a", State: ExecutionShadow, ReasonCode: "POLICY_MATCHED", CreatedAt: now}
	created, inserted, err := repo.CreateExecution(ctx, receipt)
	if err != nil || !inserted || created.ID != receipt.ID {
		t.Fatalf("first execution = %#v, inserted=%v, err=%v", created, inserted, err)
	}
	replayed := receipt
	replayed.ID = "execution-replay"
	stored, inserted, err := repo.CreateExecution(ctx, replayed)
	if err != nil || inserted || stored.ID != receipt.ID {
		t.Fatalf("replayed execution = %#v, inserted=%v, err=%v", stored, inserted, err)
	}

	episode := AdverseEpisode{ID: "episode-a", TenantID: "bank", LegalEntityID: "entity", PolicyCode: "poor-response", PolicyID: "policy-a", PolicyVersion: 1, SubjectType: "VENDOR", SubjectID: "vendor-a", State: EpisodeOpen, LastResponseRevisionID: "response-a", OpenedAt: now, UpdatedAt: now, RecordVersion: 1}
	opened, inserted, err := repo.OpenEpisode(ctx, episode)
	if err != nil || !inserted || opened.ID != episode.ID {
		t.Fatalf("first episode = %#v, inserted=%v, err=%v", opened, inserted, err)
	}
	replayedEpisode := episode
	replayedEpisode.ID = "episode-replay"
	replayedEpisode.LastResponseRevisionID = "response-b"
	storedEpisode, inserted, err := repo.OpenEpisode(ctx, replayedEpisode)
	if err != nil || inserted || storedEpisode.ID != episode.ID {
		t.Fatalf("replayed episode = %#v, inserted=%v, err=%v", storedEpisode, inserted, err)
	}

	conflict := receipt
	conflict.State = ExecutionApplied
	if _, _, err := repo.CreateExecution(ctx, conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed replay err = %v", err)
	}
}

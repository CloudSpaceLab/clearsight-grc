package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProgramReviewCheckpointIsActorScopedAndDerivedFromCanonicalVersions(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID:       "bank",
		Code:           "AML",
		Name:           "AML Programme",
		Type:           "REGULATORY",
		OwningFunction: "Compliance",
		Scope:          json.RawMessage(`{}`),
		EffectiveFrom:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil || program.CurrentState.ProjectionVersion != 1 {
		t.Fatalf("expected canonical first projection, got %#v", program.CurrentState)
	}

	before, err := service.ProgramReviewDigest(ctx, "bank", program.Program.ID, "reviewer-a")
	if err != nil {
		t.Fatal(err)
	}
	if before.State != "NO_BASELINE" || !before.ReviewRequired || before.Checkpoint != nil || before.CurrentProjectionVersion < 1 {
		t.Fatalf("unexpected first-review digest: %#v", before)
	}

	accepted, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "bank",
		ProgramID:                 program.Program.ID,
		PrincipalID:               "reviewer-a",
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: before.CurrentProjectionVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.State != "CURRENT" || accepted.ReviewRequired || accepted.Checkpoint == nil {
		t.Fatalf("expected accepted current baseline, got %#v", accepted)
	}
	if accepted.Checkpoint.ProgramVersion != program.Program.Version || accepted.Checkpoint.ProjectionVersion != accepted.CurrentProjectionVersion {
		t.Fatalf("checkpoint did not expose actor-safe versions: %#v", accepted.Checkpoint)
	}

	otherActor, err := service.ProgramReviewDigest(ctx, "bank", program.Program.ID, "reviewer-b")
	if err != nil {
		t.Fatal(err)
	}
	if otherActor.State != "NO_BASELINE" || otherActor.Checkpoint != nil {
		t.Fatalf("one actor's acknowledgement leaked to another actor: %#v", otherActor)
	}

	now = now.Add(time.Minute)
	updated, err := service.AddRequirement(ctx, AddRequirementInput{
		TenantID:        "bank",
		ProgramID:       program.Program.ID,
		ExpectedVersion: program.Program.Version,
		Code:            "CDD",
		Title:           "Maintain current customer due diligence",
		Statement:       "Customer due diligence must remain current.",
		Modality:        "MUST",
		Status:          RequirementApproved,
		EffectiveFrom:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentState == nil || updated.CurrentState.ProjectionVersion <= program.CurrentState.ProjectionVersion {
		t.Fatalf("expected a newer canonical projection, got %#v", updated.CurrentState)
	}

	changed, err := service.ProgramReviewDigest(ctx, "bank", program.Program.ID, "reviewer-a")
	if err != nil {
		t.Fatal(err)
	}
	if changed.State != "CHANGED" || !changed.ReviewRequired || changed.ChangesTotal == 0 || changed.CurrentProjectionVersion == accepted.CurrentProjectionVersion {
		t.Fatalf("expected changed digest, got %#v", changed)
	}
	foundRequirement := false
	for _, change := range changed.Changes {
		if change.Kind == "REQUIREMENT" && strings.Contains(change.Summary, "customer due diligence") {
			foundRequirement = true
		}
	}
	if !foundRequirement {
		t.Fatalf("expected canonical requirement event in change digest: %#v", changed.Changes)
	}
	if changed.ResolvedExceptionsTotal == 0 {
		t.Fatalf("expected initial setup exception to be resolved: %#v", changed)
	}

	_, err = service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "bank",
		ProgramID:                 program.Program.ID,
		PrincipalID:               "reviewer-a",
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: accepted.CurrentProjectionVersion,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale screen must not become the review baseline, got %v", err)
	}

	current, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "bank",
		ProgramID:                 updated.Program.ID,
		PrincipalID:               "reviewer-a",
		ExpectedProgramVersion:    updated.Program.Version,
		ExpectedProjectionVersion: changed.CurrentProjectionVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if current.State != "CURRENT" || current.ReviewRequired || current.ChangesTotal != 0 {
		t.Fatalf("accepting current versions should clear the derived change delta: %#v", current)
	}
}

func TestDiffStateReasonsUsesStableCanonicalIdentity(t *testing.T) {
	before := []StateReason{{Code: "EVIDENCE_EXPIRED", Summary: "Old wording", ObjectType: "EVIDENCE_CONTRACT", ObjectID: "contract-1"}}
	after := []StateReason{{Code: "EVIDENCE_EXPIRED", Summary: "New wording", ObjectType: "EVIDENCE_CONTRACT", ObjectID: "contract-1"}, {Code: "SOURCE_DEGRADED", Summary: "Source unavailable"}}
	added, resolved := diffStateReasons(before, after)
	if len(added) != 1 || added[0].Code != "SOURCE_DEGRADED" || len(resolved) != 0 {
		t.Fatalf("reason wording must not manufacture a new exception: added=%#v resolved=%#v", added, resolved)
	}
}

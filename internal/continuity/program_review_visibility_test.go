package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestProgramReviewDigestIgnoresHiddenMatterOnlyProjectionChange(t *testing.T) {
	ctx := WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a")
	repo := NewMemoryRepository()
	repo.RegisterLegalEntity("bank", "entity-a", "BANK")
	service := NewService(repo)
	t1 := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	service.now = func() time.Time { return t2 }

	program := Program{ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "REVIEW", Name: "Review visibility", Type: "COMPLIANCE", Status: ProgramActive, OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: t1, CreatedAt: t1, UpdatedAt: t1, Version: 1}
	programEvent, err := newEvent("bank", "PROGRAM", program.ID, 1, EventProgramCreated, program, ActorSystem, "", t1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateProgram(ctx, program, programEvent); err != nil {
		t.Fatal(err)
	}

	baseline := ProgramStateSnapshot{ID: "baseline", TenantID: "bank", ProgramID: program.ID, Overall: StateCurrent, Dimensions: allCurrentDimensions(), Reasons: []StateReason{}, OpenMatterCount: 0, GeneratedAt: t1, ProgramVersion: 1}
	if _, err := repo.SaveProgramState(ctx, "bank", program.ID, 1, baseline); err != nil {
		t.Fatal(err)
	}
	for _, principalID := range []string{"person-a", "person-b"} {
		checkpoint := ProgramReviewCheckpoint{ID: "checkpoint-" + principalID, TenantID: "bank", ProgramID: program.ID, PrincipalID: principalID, ProgramVersion: 1, ProjectionVersion: 1, AcceptedAt: t1}
		reviewEvent, err := newEvent("bank", "PROGRAM_REVIEW", checkpoint.ID, 1, EventProgramReviewAccepted, checkpoint, actorFor(principalID), principalID, t1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.RecordProgramReview(ctx, checkpoint, reviewEvent); err != nil {
			t.Fatal(err)
		}
	}

	matter := Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-a", Reference: "MAT-1", Type: MatterAuthorityRequest, Status: MatterAssessment, Priority: 4, Title: "Restricted issue", Summary: "Restricted issue", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"]}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: t2, UpdatedAt: t2, Version: 1}
	matterEvent, err := newEvent("bank", "MATTER", matter.ID, 1, EventMatterCreated, matter, ActorSystem, "", t2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateMatter(ctx, matter, matterEvent); err != nil {
		t.Fatal(err)
	}
	link := MatterLink{ID: "link-1", TenantID: "bank", MatterID: matter.ID, ProgramID: program.ID, Relationship: "AFFECTS", CreatedAt: t2}
	linkEvent, err := newEvent("bank", "MATTER", matter.ID, 2, EventMatterLinked, link, ActorSystem, "", t2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyMatterEvent(ctx, "bank", matter.ID, 1, linkEvent); err != nil {
		t.Fatal(err)
	}

	current := ProgramStateSnapshot{ID: "current", TenantID: "bank", ProgramID: program.ID, Overall: StateAtRisk, Dimensions: allCurrentDimensions(), Reasons: []StateReason{{Code: "OPEN_MATTERS", Summary: "1 open issue(s) or change(s) affect this program."}}, OpenMatterCount: 1, TriggerType: EventMatterLinked, TriggerID: matter.ID, GeneratedAt: t2, ProgramVersion: 1}
	current.Dimensions.Exception = StateAtRisk
	if _, err := repo.SaveProgramState(ctx, "bank", program.ID, 1, current); err != nil {
		t.Fatal(err)
	}

	hiddenDigest, err := service.ProgramReviewDigest(ctx, "bank", program.ID, "person-a")
	if err != nil {
		t.Fatal(err)
	}
	if hiddenDigest.State != "CURRENT" || hiddenDigest.ReviewRequired || hiddenDigest.OpenMatterCount != 0 || hiddenDigest.OpenMatterDelta != 0 || hiddenDigest.ChangesTotal != 0 {
		t.Fatalf("hidden Matter changed actor A review digest: %#v", hiddenDigest)
	}
	if hiddenDigest.CurrentOverall != StateCurrent || hasReasonCode(hiddenDigest.CurrentExceptions, "OPEN_MATTERS") {
		t.Fatalf("hidden Matter leaked through actor A review state: %#v", hiddenDigest)
	}
	if hiddenDigest.Checkpoint == nil || hiddenDigest.Checkpoint.ProjectionVersion != hiddenDigest.CurrentProjectionVersion {
		t.Fatalf("actor A semantic checkpoint/version diverged: %#v", hiddenDigest)
	}

	visibleDigest, err := service.ProgramReviewDigest(ctx, "bank", program.ID, "person-b")
	if err != nil {
		t.Fatal(err)
	}
	if visibleDigest.State != "CHANGED" || !visibleDigest.ReviewRequired || visibleDigest.OpenMatterCount != 1 || visibleDigest.OpenMatterDelta != 1 {
		t.Fatalf("authorized actor B lost visible Matter review change: %#v", visibleDigest)
	}
	if visibleDigest.CurrentOverall != StateAtRisk || !hasReasonCode(visibleDigest.CurrentExceptions, "OPEN_MATTERS") {
		t.Fatalf("actor B visible Matter state missing: %#v", visibleDigest)
	}
}

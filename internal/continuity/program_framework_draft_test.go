package continuity

import (
	"context"
	"testing"
	"time"
)

func TestFrameworkDraftCanBeSavedAndSubmittedWithoutSkippingReview(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	s := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	p := createProgramForCalculationPolicy(t, s, ctx, now)
	s.ConfigureFrameworkEditionLookup(fixtureEditionLookup{value: FrameworkEditionReference{ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY", EditionCode: "2026", ReleaseSHA256: "checksum"}})
	input := ProposeFrameworkAdoptionInput{TenantID: "bank", ProgramID: p.Program.ID, EditionID: "edition-1", ExpectedVersion: p.Program.Version, ActorID: "program-owner", Rationale: "Initial scope"}
	first, err := s.SaveFrameworkAdoptionDraft(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.FrameworkAdoptions) != 1 || first.FrameworkAdoptions[0].Status != FrameworkAdoptionDraft {
		t.Fatalf("draft missing: %+v", first.FrameworkAdoptions)
	}
	id := first.FrameworkAdoptions[0].ID
	input.ExpectedVersion = first.Program.Version
	input.Rationale = "Updated scope"
	second, err := s.SaveFrameworkAdoptionDraft(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if second.FrameworkAdoptions[0].ID != id || second.FrameworkAdoptions[0].Version != 2 || second.FrameworkAdoptions[0].Rationale != "Updated scope" {
		t.Fatalf("draft update failed: %+v", second.FrameworkAdoptions)
	}
	if _, err = s.SaveFrameworkAdoptionDraft(ctx, input); err == nil {
		t.Fatal("stale Program version accepted")
	}
	input.ExpectedVersion = second.Program.Version
	submitted, err := s.ProposeFrameworkAdoption(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(submitted.FrameworkAdoptions) != 1 || submitted.FrameworkAdoptions[0].ID != id || submitted.FrameworkAdoptions[0].Status != FrameworkAdoptionPending || submitted.FrameworkAdoptions[0].Version != 3 {
		t.Fatalf("submitted draft not pending: %+v", submitted.FrameworkAdoptions)
	}
	if _, err = s.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{TenantID: "bank", ProgramID: p.Program.ID, AdoptionID: id, ExpectedVersion: submitted.Program.Version, ActorID: "program-owner", Rationale: "Self approval"}); err == nil {
		t.Fatal("maker approved own draft")
	}
}

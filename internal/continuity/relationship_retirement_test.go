package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRetireRequirementControlLinkPreservesHistoryAndRemovesCurrentCoverage(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repository := NewMemoryRepository()
	service := NewService(repository)
	createdAt := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return createdAt }
	program, link := createLinkedRequirementControl(t, ctx, service, createdAt)
	linkedVersion := program.Program.Version
	checkpoint := createdAt.Add(time.Minute)
	service.now = func() time.Time { return checkpoint.Add(time.Minute) }

	retired, err := service.RetireRequirementControlLink(ctx, RetireRequirementControlLinkInput{
		TenantID: "bank", ProgramID: program.Program.ID, LinkID: link.ID, ExpectedVersion: linkedVersion,
		ActorID: "owner-1", Rationale: "The safeguard was mapped to the wrong requirement.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(retired.RequirementControlLinks) != 0 {
		t.Fatalf("retired coverage link remained current: %#v", retired.RequirementControlLinks)
	}
	coverage := CurrentRequirementCoverage(retired, checkpoint.Add(time.Minute))[link.RequirementID]
	if len(coverage.ControlIDs) != 0 || coverage.ControlImplemented {
		t.Fatalf("retired safeguard still contributes to current coverage: %#v", coverage)
	}
	historical, err := service.ProgramAt(ctx, "bank", program.Program.ID, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if len(historical.RequirementControlLinks) != 1 || historical.RequirementControlLinks[0].ID != link.ID {
		t.Fatalf("point-in-time reconstruction lost the former link: %#v", historical.RequirementControlLinks)
	}
	events, err := repository.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if last.Type != EventRequirementControlLinkRetired || last.ActorID != "owner-1" {
		t.Fatalf("retirement was not append-only and actor-backed: %#v", last)
	}
	var recorded RequirementControlLink
	if err := json.Unmarshal(last.Payload, &recorded); err != nil {
		t.Fatal(err)
	}
	if recorded.ID != link.ID || recorded.RetiredAt == nil || recorded.RetiredBy != "owner-1" || recorded.RetirementReason == "" {
		t.Fatalf("retirement record is incomplete: %#v", recorded)
	}
}

func TestRetireRequirementControlLinkIsOptimisticAndFailureLeavesNoEvent(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repository := NewMemoryRepository()
	service := NewService(repository)
	now := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, link := createLinkedRequirementControl(t, ctx, service, now)
	before, _ := repository.ProgramEvents(ctx, "bank", program.Program.ID, nil)

	_, err := service.RetireRequirementControlLink(ctx, RetireRequirementControlLinkInput{
		TenantID: "bank", ProgramID: program.Program.ID, LinkID: link.ID, ExpectedVersion: program.Program.Version - 1,
		ActorID: "owner-1", Rationale: "The relationship is incorrect.",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale retirement returned %v", err)
	}
	after, _ := repository.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	current, _ := service.GetProgram(ctx, "bank", program.Program.ID)
	if len(after) != len(before) || len(current.RequirementControlLinks) != 1 {
		t.Fatalf("failed retirement changed authoritative state: events %d->%d links=%#v", len(before), len(after), current.RequirementControlLinks)
	}
}

func TestRetireLastMatterProgramLinkIsAllowedAndExcludesCurrentQueries(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repository := NewMemoryRepository()
	service := NewService(repository)
	createdAt := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return createdAt }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY",
		OwningFunction: "Privacy", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1", Scope: json.RawMessage(`{}`), EffectiveFrom: createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 3,
		Title: "Correct the safeguard mapping", Summary: "The issue was linked to the wrong Program.", Scope: json.RawMessage(`{}`),
		ProgramID: program.Program.ID, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	link := matter.Links[0]
	checkpoint := createdAt.Add(time.Minute)
	service.now = func() time.Time { return checkpoint.Add(time.Minute) }

	retired, err := service.RetireMatterLink(ctx, RetireMatterLinkInput{
		TenantID: "bank", MatterID: matter.Matter.ID, LinkID: link.ID, ExpectedVersion: matter.Matter.Version,
		ActorID: "owner-1", Rationale: "This issue belongs outside the Privacy Program.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(retired.Links) != 0 {
		t.Fatalf("the retired last Program link remained current: %#v", retired.Links)
	}
	programIDs, err := repository.LinkedProgramIDs(ctx, "bank", matter.Matter.ID)
	if err != nil || len(programIDs) != 0 {
		t.Fatalf("retired link remained in exact current query: ids=%#v err=%v", programIDs, err)
	}
	open, err := repository.OpenMatterCount(ctx, "bank", program.Program.ID)
	if err != nil || open != 0 {
		t.Fatalf("retired link remained in Program open issue count: count=%d err=%v", open, err)
	}
	historical, err := service.MatterAt(ctx, "bank", matter.Matter.ID, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if len(historical.Links) != 1 || historical.Links[0].ID != link.ID {
		t.Fatalf("point-in-time Matter reconstruction lost the former link: %#v", historical.Links)
	}
	events, _ := repository.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	last := events[len(events)-1]
	if last.Type != EventMatterLinkRetired || last.ActorID != "owner-1" {
		t.Fatalf("Matter link retirement is not append-only and actor-backed: %#v", last)
	}
}

func createLinkedRequirementControl(t *testing.T, ctx context.Context, service *Service, now time.Time) (ProgramAggregate, RequirementControlLink) {
	t.Helper()
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY",
		OwningFunction: "Privacy", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "ACCESS", Title: "Review privileged access", Statement: "Privileged access must be reviewed.", Status: RequirementApproved, EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "ACCESS-REVIEW", Name: "Current access review", Outcome: "Every privileged account is reviewed.", Status: ObjectiveActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: program.ControlObjectives[0].ID,
		Name: "Quarterly access review", Description: "Account owners confirm privileged access.", ImplementationType: "REVIEW",
		EffectiveFrom: now, Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []ControlImplementationStatus{ImplementationInProgress, ImplementationImplemented} {
		implementation := program.ControlImplementations[0]
		program, err = service.TransitionControlImplementation(ctx, TransitionControlImplementationInput{
			TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: implementation.ID,
			ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: implementation.Version,
			To: target, Rationale: "Progress the safeguard through its governed lifecycle.", ActorID: "owner-1",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	program, err = service.LinkRequirementControl(ctx, LinkRequirementControlInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		RequirementID: program.Requirements[0].ID, ImplementationID: program.ControlImplementations[0].ID, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.RequirementControlLinks[0]
}

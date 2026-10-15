package continuity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type fixtureEditionLookup struct {
	value FrameworkEditionReference
	err   error
}

func (f fixtureEditionLookup) GetFrameworkEdition(_ context.Context, id string) (FrameworkEditionReference, error) {
	if f.err != nil {
		return FrameworkEditionReference{}, f.err
	}
	if id != f.value.ID {
		return FrameworkEditionReference{}, ErrNotFound
	}
	return f.value, nil
}

func TestFrameworkAdoptionGovernedEventAndHistory(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program := createProgramForCalculationPolicy(t, service, ctx, now)
	service.ConfigureFrameworkEditionLookup(fixtureEditionLookup{value: FrameworkEditionReference{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", ReleaseSHA256: "published-checksum",
	}})
	makerAt := now.Add(time.Minute)
	service.now = func() time.Time { return makerAt }
	input := ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID, EditionID: "edition-1",
		ExpectedVersion: program.Program.Version, ActorID: "program-owner",
		Rationale: "Adopt the approved security edition.",
	}
	proposed, err := service.ProposeFrameworkAdoption(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposed.FrameworkAdoptions) != 1 ||
		proposed.FrameworkAdoptions[0].Status != FrameworkAdoptionPending ||
		proposed.FrameworkAdoptions[0].ProposedBy != "program-owner" {
		t.Fatalf("proposal not retained: %+v", proposed.FrameworkAdoptions)
	}
	if len(proposed.Requirements) != 0 || proposed.Program.CalculationPolicy != nil {
		t.Fatal("framework proposal mutated requirements or scoring policy")
	}
	if _, err := service.ProposeFrameworkAdoption(ctx, ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: proposed.Program.Version, EditionID: "edition-1",
		ActorID: "program-owner", Rationale: "Duplicate proposed standard.",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("duplicate pending edition permitted: %v", err)
	}
	id := proposed.FrameworkAdoptions[0].ID
	checkerAt := makerAt.Add(time.Minute)
	service.now = func() time.Time { return checkerAt }
	if _, err := service.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID, AdoptionID: id,
		ExpectedVersion: proposed.Program.Version, ActorID: "program-owner",
		Rationale: "Self-approval.",
	}); !errors.Is(err, ErrInvalidState) && !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker approved own adoption: %v", err)
	}
	before, err := service.ProgramAt(ctx, "bank", program.Program.ID, makerAt)
	if err != nil || len(before.FrameworkAdoptions) != 1 ||
		before.FrameworkAdoptions[0].Status != FrameworkAdoptionPending {
		t.Fatalf("historical proposal not reconstructable: %+v %v", before.FrameworkAdoptions, err)
	}
	approved, err := service.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID, AdoptionID: id,
		ExpectedVersion: proposed.Program.Version, ActorID: "independent-checker",
		Rationale: "Verified the edition and intended scope.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(approved.FrameworkAdoptions) != 1 ||
		approved.FrameworkAdoptions[0].Status != FrameworkAdoptionActive ||
		approved.FrameworkAdoptions[0].Version != 2 ||
		approved.FrameworkAdoptions[0].ReviewedBy != "independent-checker" ||
		approved.FrameworkAdoptions[0].EffectiveFrom == nil {
		t.Fatalf("approval not governed: %+v", approved.FrameworkAdoptions)
	}
	history, err := service.ProgramAt(ctx, "bank", program.Program.ID, makerAt)
	if err != nil || history.FrameworkAdoptions[0].Status != FrameworkAdoptionPending {
		t.Fatalf("approval rewrote earlier event: %+v %v", history, err)
	}
	events, err := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil || events[len(events)-1].Type != EventProgramFrameworkAdoptionApproved ||
		events[len(events)-1].ActorID != "independent-checker" {
		t.Fatalf("approval event lost actor: %+v %v", events, err)
	}
	if len(approved.Requirements) != 0 || approved.Program.CalculationPolicy != nil {
		t.Fatal("approved edition created fictional compliance items")
	}
}

func TestFrameworkAdoptionRejectedAndInvalidInputsLeaveNoEvents(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program := createProgramForCalculationPolicy(t, service, ctx, now)
	input := ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: program.Program.Version, EditionID: "edition-1",
		ActorID: "program-owner", Rationale: "Review edition.",
	}
	if _, err := service.ProposeFrameworkAdoption(ctx, input); !errors.Is(err, ErrFrameworkEditionUnavailable) {
		t.Fatalf("adoption succeeded without trusted catalog: %v", err)
	}
	service.ConfigureFrameworkEditionLookup(fixtureEditionLookup{value: FrameworkEditionReference{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", ReleaseSHA256: "published-checksum",
	}})
	input.ActorID = "independent-checker"
	if _, err := service.ProposeFrameworkAdoption(ctx, input); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("non-owner proposal accepted: %v", err)
	}
	input.ActorID = "program-owner"
	proposed, err := service.ProposeFrameworkAdoption(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	count, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if _, err := service.RejectFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID:      proposed.FrameworkAdoptions[0].ID,
		ExpectedVersion: proposed.Program.Version - 1,
		ActorID:         "independent-checker", Rationale: "Old version.",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale decision accepted: %v", err)
	}
	unchanged, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if len(count) != len(unchanged) {
		t.Fatal("failed command generated an event")
	}
	service.now = func() time.Time { return now.Add(time.Second) }
	rejected, err := service.RejectFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID:      proposed.FrameworkAdoptions[0].ID,
		ExpectedVersion: proposed.Program.Version,
		ActorID:         "independent-checker", Rationale: "Edition not applicable.",
	})
	if err != nil || rejected.FrameworkAdoptions[0].Status != FrameworkAdoptionRejected ||
		rejected.FrameworkAdoptions[0].EffectiveFrom != nil {
		t.Fatalf("rejected proposal became active: %+v %v", rejected, err)
	}
}

func TestFrameworkAdoptionAuthenticatedActorCannotSpoofOwner(t *testing.T) {
	trusted := WithTrustedSystemScope(context.Background())
	repo := NewMemoryRepository()
	service := NewService(repo)
	program := createProgramForCalculationPolicy(t, service, trusted, time.Now().UTC())
	service.ConfigureFrameworkEditionLookup(fixtureEditionLookup{value: FrameworkEditionReference{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", ReleaseSHA256: "published-checksum",
	}})
	ctx := identity.WithActor(context.Background(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "attacker",
	})
	_, err := service.ProposeFrameworkAdoption(ctx, ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: program.Program.Version, EditionID: "edition-1",
		ActorID: "program-owner", Rationale: "Spoof the Program owner.",
	})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("body actor spoofing succeeded: %v", err)
	}
}

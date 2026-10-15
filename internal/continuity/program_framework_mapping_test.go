package continuity

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mappingTestCatalog struct {
	fixtureEditionLookup
	clauseID string
}

func (c mappingTestCatalog) GetFrameworkClause(_ context.Context, editionID, clauseID string) (FrameworkClauseReference, error) {
	if editionID != c.value.ID || clauseID != c.clauseID {
		return FrameworkClauseReference{}, ErrNotFound
	}
	return FrameworkClauseReference{ID: clauseID, EditionID: editionID}, nil
}

type mappingMemoryRepo struct{ *MemoryRepository }

func (r *mappingMemoryRepo) GetFrameworkMapping(ctx context.Context, tenant, programID, mappingID string) (FrameworkClauseMapping, error) {
	events, err := r.ProgramEvents(ctx, tenant, programID, nil)
	if err != nil {
		return FrameworkClauseMapping{}, err
	}
	var latest FrameworkClauseMapping
	for _, event := range events {
		if event.Type != EventFrameworkMappingProposed &&
			event.Type != EventFrameworkMappingApproved &&
			event.Type != EventFrameworkMappingRejected {
			continue
		}
		mapping, err := frameworkMappingFromEvent(event)
		if err != nil {
			return FrameworkClauseMapping{}, err
		}
		if mapping.ID == mappingID {
			latest = mapping
		}
	}
	if latest.ID == "" {
		return FrameworkClauseMapping{}, ErrNotFound
	}
	return latest, nil
}

func setupFrameworkMappingService(t *testing.T) (context.Context, *Service, *mappingMemoryRepo, ProgramAggregate, time.Time) {
	t.Helper()
	ctx := WithTrustedSystemScope(context.Background())
	repo := &mappingMemoryRepo{MemoryRepository: NewMemoryRepository()}
	service := NewService(repo)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program := createProgramForCalculationPolicy(t, service, ctx, now)
	var err error
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: program.Program.Version,
		Code:            "ACCESS", Title: "Access reviews", Statement: "Review user access",
		SourceAnchor: "Clause 1", Modality: "MUST", Status: RequirementApproved,
		Actor: "Security", Action: "Review", Object: "Accounts",
		EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureFrameworkEditionLookup(mappingTestCatalog{
		fixtureEditionLookup: fixtureEditionLookup{value: FrameworkEditionReference{
			ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SEC",
			EditionCode: "2026", ReleaseSHA256: "approved-content",
		}},
		clauseID: "clause-1",
	})
	service.now = func() time.Time { return now.Add(time.Minute) }
	program, err = service.ProposeFrameworkAdoption(ctx, ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		EditionID: "edition-1", ExpectedVersion: program.Program.Version,
		ActorID: "program-owner", Rationale: "Assess this edition.",
	})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(2 * time.Minute) }
	program, err = service.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID:      program.FrameworkAdoptions[0].ID,
		ExpectedVersion: program.Program.Version,
		ActorID:         "independent-checker", Rationale: "Reference validated.",
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, service, repo, program, now
}

func TestFrameworkMappingReviewHistoryAndNoDuplicateScoring(t *testing.T) {
	ctx, service, repo, program, now := setupFrameworkMappingService(t)
	service.now = func() time.Time { return now.Add(3 * time.Minute) }
	proposal := ProposeFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID: program.FrameworkAdoptions[0].ID,
		ClauseID:   "clause-1", RequirementID: program.Requirements[0].ID,
		MappingKind:     FrameworkMappingPartial,
		ExpectedVersion: program.Program.Version, ActorID: "program-owner",
		Rationale: "Partially covers this source clause.",
	}
	pending, updated, err := service.ProposeFrameworkMapping(ctx, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != FrameworkMappingPending || pending.MappingKind != FrameworkMappingPartial ||
		updated.Program.Version != program.Program.Version+1 ||
		len(updated.Requirements) != 1 || updated.Program.CalculationPolicy != nil {
		t.Fatalf("proposal mutated scorable Program population: %+v %+v", pending, updated.Program)
	}
	if _, _, err := service.ApproveFrameworkMapping(ctx, DecideFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID, MappingID: pending.ID,
		ExpectedVersion: updated.Program.Version, ActorID: "program-owner",
		Rationale: "Self review",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("maker approved own proposed map: %v", err)
	}
	countBefore, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if _, _, err := service.RejectFrameworkMapping(ctx, DecideFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID, MappingID: pending.ID,
		ExpectedVersion: updated.Program.Version - 1, ActorID: "independent-checker",
		Rationale: "Stale review.",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale review was accepted: %v", err)
	}
	countAfter, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if len(countBefore) != len(countAfter) {
		t.Fatal("failed decision committed a Program event")
	}
	service.now = func() time.Time { return now.Add(4 * time.Minute) }
	approved, result, err := service.ApproveFrameworkMapping(ctx, DecideFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID, MappingID: pending.ID,
		ExpectedVersion: updated.Program.Version, ActorID: "independent-checker",
		Rationale: "Independent source comparison passed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != FrameworkMappingApproved || approved.ReviewedBy != "independent-checker" ||
		approved.EffectiveFrom == nil || approved.Version != 2 ||
		len(result.Requirements) != 1 || result.Program.CalculationPolicy != nil {
		t.Fatalf("approved mapping changed scoring inputs: %+v %+v", approved, result.Program)
	}
	prior, err := service.FrameworkMappingAt(ctx, "bank", program.Program.ID, pending.ID, now.Add(3*time.Minute))
	if err != nil || prior.Status != FrameworkMappingPending {
		t.Fatalf("mapping approval overwrote historical proposal: %+v %v", prior, err)
	}
	current, err := service.FrameworkMappingAt(ctx, "bank", program.Program.ID, pending.ID, now.Add(4*time.Minute))
	if err != nil || current.Status != FrameworkMappingApproved {
		t.Fatalf("mapping decision missing from history: %+v %v", current, err)
	}
}

func TestFrameworkMappingRejectAndMissingClause(t *testing.T) {
	ctx, service, _, program, now := setupFrameworkMappingService(t)
	service.now = func() time.Time { return now.Add(3 * time.Minute) }
	bad := ProposeFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID: program.FrameworkAdoptions[0].ID,
		ClauseID:   "other-clause", RequirementID: program.Requirements[0].ID,
		MappingKind: FrameworkMappingExact, ActorID: "program-owner",
		ExpectedVersion: program.Program.Version, Rationale: "Wrong edition clause.",
	}
	if _, _, err := service.ProposeFrameworkMapping(ctx, bad); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other edition clause accepted: %v", err)
	}
	bad.ClauseID = "clause-1"
	bad.MappingKind = "INVALID"
	if _, _, err := service.ProposeFrameworkMapping(ctx, bad); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("invalid mapping kind accepted: %v", err)
	}
	bad.MappingKind = FrameworkMappingSupporting
	pending, next, err := service.ProposeFrameworkMapping(ctx, bad)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(4 * time.Minute) }
	rejected, _, err := service.RejectFrameworkMapping(ctx, DecideFrameworkMappingInput{
		TenantID: "bank", ProgramID: program.Program.ID, MappingID: pending.ID,
		ExpectedVersion: next.Program.Version, ActorID: "independent-checker",
		Rationale: "Insufficient control coverage.",
	})
	if err != nil || rejected.Status != FrameworkMappingRejected || rejected.EffectiveFrom != nil {
		t.Fatalf("rejected mapping became active: %+v %v", rejected, err)
	}
}

package rcsa

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testPopulationResolver struct {
	population Population
	err        error
}

func (r testPopulationResolver) ResolvePopulation(context.Context, Scope, []string, time.Time) (Population, error) {
	return r.population, r.err
}

func TestCreateFreezesExactRiskControlPopulation(t *testing.T) {
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), testPopulationResolver{population: Population{
		Risks: []RiskSnapshot{
			{RiskID: "risk-b", RiskVersion: 4, Code: "RB", Name: "Payments risk", Category: "Operational"},
			{RiskID: "risk-a", RiskVersion: 2, Code: "RA", Name: "Identity risk", Category: "Technology"},
		},
		Controls: []ControlSnapshot{
			{RiskID: "risk-a", RiskVersion: 2, RiskControlLinkID: "risk-control-1", CatalogLinkID: "catalog-1",
				DefinitionID: "definition-1", DefinitionCode: "IAM-01", DefinitionName: "Access review",
				ProgramID: "program-1", ImplementationID: "implementation-1", ImplementationVersion: 3, ImplementationName: "Quarterly access review",
				ImplementationStatus: "IMPLEMENTED", ImplementationEffectiveFrom: now.Add(-24 * time.Hour)},
		},
	}})
	service.Now = func() time.Time { return now }

	created, err := service.Create(context.Background(), CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: " q4-rcsa ", Name: "Q4 RCSA",
		TriggerKind: TriggerScheduled, RiskIDs: []string{"risk-b", "risk-a", "risk-a"},
		FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Cycle.Code != "Q4-RCSA" || created.Cycle.Status != StatusDraft || created.Cycle.Version != 1 ||
		len(created.Cycle.PopulationChecksum) != 64 {
		t.Fatalf("cycle=%#v", created.Cycle)
	}
	if len(created.Risks) != 2 || len(created.Controls) != 1 {
		t.Fatalf("population=%#v", created)
	}
	if created.Risks[0].CycleID != created.Cycle.ID || created.Controls[0].CycleID != created.Cycle.ID {
		t.Fatal("cycle identity was not applied to population snapshots")
	}

	read, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Cycle.PopulationChecksum != created.Cycle.PopulationChecksum {
		t.Fatalf("read checksum=%q want=%q", read.Cycle.PopulationChecksum, created.Cycle.PopulationChecksum)
	}
}

func TestFirstLineLifecyclePinsServerResolvedFinalResponse(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), testPopulationResolver{population: Population{
		Risks: []RiskSnapshot{{RiskID: "risk-a", RiskVersion: 2, Code: "RA", Name: "Risk A"}},
	}})
	service.Now = func() time.Time { return now }
	validatedDistribution := ""
	resolvedCycle := ""
	service.ConfigureFirstLine(
		func(_ context.Context, scope Scope, cycle Cycle, distributionID string) error {
			if scope != (Scope{TenantID: "bank", LegalEntityID: "entity-a"}) ||
				cycle.FirstLineOwnerID != "owner-1" || distributionID != "distribution-1" {
				return ErrInvalid
			}
			validatedDistribution = distributionID
			return nil
		},
		func(_ context.Context, scope Scope, cycle Cycle) (string, error) {
			if scope != (Scope{TenantID: "bank", LegalEntityID: "entity-a"}) ||
				cycle.FirstLineDistributionID != "distribution-1" || cycle.Status != StatusAssessmentOpen {
				return "", ErrInvalid
			}
			resolvedCycle = cycle.ID
			return "response-final-2", nil
		},
	)
	created, err := service.Create(context.Background(), CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-Q4", Name: "Q4 RCSA",
		TriggerKind: TriggerScheduled, RiskIDs: []string{"risk-a"}, FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	opened, err := service.BindFirstLineDistribution(context.Background(), BindFirstLineDistributionInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: 1, DistributionID: "distribution-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if validatedDistribution != "distribution-1" || opened.Status != StatusAssessmentOpen ||
		opened.FirstLineDistributionID != "distribution-1" || opened.Version != 2 ||
		opened.FirstLineResponseRevisionID != "" {
		t.Fatalf("opened=%#v", opened)
	}

	now = now.Add(time.Minute)
	challengeReady, err := service.CompleteFirstLine(context.Background(), CompleteFirstLineInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: 2, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolvedCycle != created.Cycle.ID || challengeReady.Status != StatusAwaitingChallenge ||
		challengeReady.FirstLineResponseRevisionID != "response-final-2" || challengeReady.Version != 3 {
		t.Fatalf("challengeReady=%#v", challengeReady)
	}
	read, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Cycle != challengeReady {
		t.Fatalf("read cycle=%#v want=%#v", read.Cycle, challengeReady)
	}
}

func TestFirstLineLifecycleRejectsWrongOwnerAndIncompleteResponse(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 30, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), testPopulationResolver{population: Population{
		Risks: []RiskSnapshot{{RiskID: "risk-a", RiskVersion: 1, Code: "RA", Name: "Risk A"}},
	}})
	service.Now = func() time.Time { return now }
	service.ConfigureFirstLine(
		func(context.Context, Scope, Cycle, string) error { return nil },
		func(context.Context, Scope, Cycle) (string, error) { return "", ErrInvalid },
	)
	created, err := service.Create(context.Background(), CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-Q3", Name: "Q3 RCSA",
		TriggerKind: TriggerManual, RiskIDs: []string{"risk-a"}, FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.BindFirstLineDistribution(context.Background(), BindFirstLineDistributionInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: 1, DistributionID: "distribution-1", ActorID: "other-owner",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong-owner bind error=%v", err)
	}
	unchanged, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Cycle.Version != 1 || unchanged.Cycle.Status != StatusDraft {
		t.Fatalf("wrong-owner bind changed cycle=%#v", unchanged.Cycle)
	}

	opened, err := service.BindFirstLineDistribution(context.Background(), BindFirstLineDistributionInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: 1, DistributionID: "distribution-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CompleteFirstLine(context.Background(), CompleteFirstLineInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: opened.Version, ActorID: "owner-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("incomplete response error=%v", err)
	}
	after, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Cycle.Version != 2 || after.Cycle.Status != StatusAssessmentOpen || after.Cycle.FirstLineResponseRevisionID != "" {
		t.Fatalf("failed completion changed cycle=%#v", after.Cycle)
	}
}

func TestCreateRejectsIncompleteOrCrossScopePopulation(t *testing.T) {
	service := NewService(NewMemoryRepository(), testPopulationResolver{population: Population{
		Risks: []RiskSnapshot{{RiskID: "risk-a", RiskVersion: 1, Code: "RA", Name: "Risk A"}},
	}})
	service.Now = func() time.Time { return time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC) }
	_, err := service.Create(context.Background(), CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-1", Name: "RCSA",
		TriggerKind: TriggerManual, RiskIDs: []string{"risk-a", "risk-b"}, FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("population error=%v", err)
	}
}

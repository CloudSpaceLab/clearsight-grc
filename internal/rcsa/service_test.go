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

func (r testPopulationResolver) ResolvePopulation(context.Context, Scope, []string) (Population, error) {
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
				ProgramID: "program-1", ImplementationID: "implementation-1", ImplementationVersion: 3, ImplementationName: "Quarterly access review"},
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

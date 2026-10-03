package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
)

type rcsaChallengePopulation struct{}

func (rcsaChallengePopulation) ResolvePopulation(context.Context, rcsa.Scope, []string, time.Time) (rcsa.Population, error) {
	return rcsa.Population{Risks: []rcsa.RiskSnapshot{{RiskID: "risk-1", RiskVersion: 1, Code: "R1", Name: "Risk 1"}}}, nil
}

func TestRCSAChallengeBridgeUsesOneCanonicalMatterAndDecision(t *testing.T) {
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a")
	matters := continuity.NewService(continuity.NewMemoryRepository())
	cycles := rcsa.NewService(rcsa.NewMemoryRepository(), rcsaChallengePopulation{})
	now := time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)
	cycles.Now = func() time.Time { return now }
	cycles.ConfigureFirstLine(
		func(context.Context, rcsa.Scope, rcsa.Cycle, string) error { return nil },
		func(context.Context, rcsa.Scope, rcsa.Cycle) (string, error) { return "response-final-1", nil },
	)
	configureRCSAChallenge(cycles, matters)

	created, err := cycles.Create(ctx, rcsa.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-1", Name: "Quarterly RCSA",
		TriggerKind: rcsa.TriggerScheduled, RiskIDs: []string{"risk-1"}, FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := cycles.BindFirstLineDistribution(ctx, rcsa.BindFirstLineDistributionInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: 1, DistributionID: "distribution-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := cycles.CompleteFirstLine(ctx, rcsa.CompleteFirstLineInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: opened.Version, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	challenging, err := cycles.StartChallenge(ctx, rcsa.StartChallengeInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: ready.Version, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if challenging.ChallengeMatterID == "" {
		t.Fatal("challenge Matter was not linked")
	}

	matter, err := matters.MatterByTriggerKey(ctx, "bank", "rcsa-challenge:"+created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.ID != challenging.ChallengeMatterID || matter.Matter.Type != continuity.MatterRiskSituation ||
		matter.Matter.SourceType != "RCSA_CYCLE" || matter.Matter.SourceID != created.Cycle.ID ||
		matter.Matter.OwnerPrincipalID != "reviewer-1" {
		t.Fatalf("challenge Matter=%#v", matter.Matter)
	}
	var scope struct {
		CycleID            string `json:"rcsa_cycle_id"`
		PopulationChecksum string `json:"population_checksum"`
		ResponseRevision   string `json:"first_line_response_revision_id"`
	}
	if err := json.Unmarshal(matter.Matter.Scope, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.CycleID != created.Cycle.ID || scope.PopulationChecksum != created.Cycle.PopulationChecksum ||
		scope.ResponseRevision != "response-final-1" {
		t.Fatalf("challenge scope=%#v", scope)
	}
	decision := continuity.CurrentDecisionForType(matter.Decisions, rcsaChallengeDecisionType)
	if decision == nil || decision.Status != continuity.DecisionProposed || decision.ProposedBy != "reviewer-1" {
		t.Fatalf("challenge decision=%#v", decision)
	}

	matter, err = matters.RecordDecisionLifecycle(ctx, continuity.AddDecisionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Type: rcsaChallengeDecisionType, Status: continuity.DecisionApproved,
		Options: rcsaChallengeOptions, SelectedOption: "ACCEPT_FIRST_LINE",
		Rationale:  "Independent challenge accepted the first-line assessment.",
		Conditions: json.RawMessage(`[]`), AuthorityPrincipalID: "authorizer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := cycles.CompleteChallenge(ctx, rcsa.CompleteChallengeInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: challenging.Version, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != rcsa.StatusCompleted {
		t.Fatalf("completed cycle=%#v", completed)
	}
	after, err := matters.GetMatter(ctx, "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Matter.Status == continuity.MatterClosed || after.Matter.Status == continuity.MatterCancelled {
		t.Fatalf("RCSA completion incorrectly closed challenge Matter: %#v", after.Matter)
	}
}

func TestRCSAChallengeCompletionRejectsFirstLineAuthorityDecision(t *testing.T) {
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a")
	matters := continuity.NewService(continuity.NewMemoryRepository())
	cycles := rcsa.NewService(rcsa.NewMemoryRepository(), rcsaChallengePopulation{})
	cycles.Now = func() time.Time { return time.Date(2026, 10, 3, 16, 0, 0, 0, time.UTC) }
	cycles.ConfigureFirstLine(
		func(context.Context, rcsa.Scope, rcsa.Cycle, string) error { return nil },
		func(context.Context, rcsa.Scope, rcsa.Cycle) (string, error) { return "response-final-1", nil },
	)
	configureRCSAChallenge(cycles, matters)
	created, err := cycles.Create(ctx, rcsa.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-2", Name: "RCSA",
		TriggerKind: rcsa.TriggerManual, RiskIDs: []string{"risk-1"}, FirstLineOwnerID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := cycles.BindFirstLineDistribution(ctx, rcsa.BindFirstLineDistributionInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID, ExpectedVersion: 1,
		DistributionID: "distribution-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := cycles.CompleteFirstLine(ctx, rcsa.CompleteFirstLineInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: opened.Version, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	challenging, err := cycles.StartChallenge(ctx, rcsa.StartChallengeInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: ready.Version, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := matters.GetMatter(ctx, "bank", challenging.ChallengeMatterID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = matters.RecordDecisionLifecycle(ctx, continuity.AddDecisionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Type: rcsaChallengeDecisionType, Status: continuity.DecisionRejected,
		Options: rcsaChallengeOptions, SelectedOption: "REQUIRE_CHANGES",
		Rationale: "First-line outcome requires correction.", Conditions: json.RawMessage(`[]`),
		AuthorityPrincipalID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cycles.CompleteChallenge(ctx, rcsa.CompleteChallengeInput{
		TenantID: "bank", LegalEntityID: "entity-a", CycleID: created.Cycle.ID,
		ExpectedVersion: challenging.Version, ActorID: "reviewer-1",
	})
	if !errors.Is(err, rcsa.ErrInvalid) {
		t.Fatalf("first-line authority challenge completion error=%v", err)
	}
}

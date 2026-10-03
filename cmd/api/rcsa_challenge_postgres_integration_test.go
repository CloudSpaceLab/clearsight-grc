//go:build postgres && postgresintegration

package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRCSAChallengeBridgeUsesCanonicalMatterTruth(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tenantID := challengeID(t)
	entityID := challengeID(t)
	firstLineID := challengeID(t)
	reviewerID := challengeID(t)
	authorizerID := challengeID(t)
	suffix := tenantID[len(tenantID)-8:]
	now := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, "rcsa-challenge-"+suffix, "RCSA Challenge "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'Challenge Entity','NG',$4::timestamptz)
	`, entityID, tenantID, "RCC-"+suffix, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$2::uuid,'PERSON','First line owner','ACTIVE',$5::timestamptz),
			($3::uuid,$2::uuid,'PERSON','Independent reviewer','ACTIVE',$5::timestamptz),
			($4::uuid,$2::uuid,'PERSON','Challenge authorizer','ACTIVE',$5::timestamptz)
	`, firstLineID, tenantID, reviewerID, authorizerID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	matters := continuity.NewService(continuity.NewReliablePostgresRepository(pool))
	cycles := rcsa.NewService(rcsa.NewMemoryRepository(), rcsaChallengePopulation{})
	cycles.Now = func() time.Time { return now }
	cycles.ConfigureFirstLine(
		func(context.Context, rcsa.Scope, rcsa.Cycle, string) error { return nil },
		func(context.Context, rcsa.Scope, rcsa.Cycle) (string, error) { return "response-final-postgres", nil },
	)
	configureRCSAChallenge(cycles, matters)
	trusted := continuity.WithTrustedSystemEntityScope(ctx, tenantID, entityID)

	created, err := cycles.Create(trusted, rcsa.CreateInput{
		TenantID: tenantID, LegalEntityID: entityID, Code: "RCSA-" + suffix, Name: "Durable challenge RCSA",
		TriggerKind: rcsa.TriggerManual, RiskIDs: []string{"risk-1"}, FirstLineOwnerID: firstLineID, ActorID: firstLineID,
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := cycles.BindFirstLineDistribution(trusted, rcsa.BindFirstLineDistributionInput{
		TenantID: tenantID, LegalEntityID: entityID, CycleID: created.Cycle.ID,
		ExpectedVersion: created.Cycle.Version, DistributionID: "distribution-postgres", ActorID: firstLineID,
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := cycles.CompleteFirstLine(trusted, rcsa.CompleteFirstLineInput{
		TenantID: tenantID, LegalEntityID: entityID, CycleID: created.Cycle.ID,
		ExpectedVersion: opened.Version, ActorID: firstLineID,
	})
	if err != nil {
		t.Fatal(err)
	}
	challenging, err := cycles.StartChallenge(trusted, rcsa.StartChallengeInput{
		TenantID: tenantID, LegalEntityID: entityID, CycleID: created.Cycle.ID,
		ExpectedVersion: ready.Version, ActorID: reviewerID,
	})
	if err != nil {
		t.Fatal(err)
	}

	matter, err := matters.MatterByTriggerKey(trusted, tenantID, "rcsa-challenge:"+created.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.ID != challenging.ChallengeMatterID || matter.Matter.OwnerPrincipalID != reviewerID {
		t.Fatalf("challenge Matter=%#v cycle=%#v", matter.Matter, challenging)
	}
	var matterCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM matters
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND trigger_key=$3
	`, tenantID, entityID, "rcsa-challenge:"+created.Cycle.ID).Scan(&matterCount); err != nil {
		t.Fatal(err)
	}
	if matterCount != 1 {
		t.Fatalf("challenge Matter count=%d want=1", matterCount)
	}

	matter, err = matters.RecordDecisionLifecycle(trusted, continuity.AddDecisionInput{
		TenantID: tenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Type: rcsaChallengeDecisionType, Status: continuity.DecisionConditionallyApproved,
		Options: rcsaChallengeOptions, SelectedOption: "DEFICIENCY_CONFIRMED",
		Rationale:            "Independent challenge confirmed a deficiency requiring governed remediation.",
		Conditions:           json.RawMessage(`["Track remediation through this Matter"]`),
		AuthorityPrincipalID: authorizerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := cycles.CompleteChallenge(trusted, rcsa.CompleteChallengeInput{
		TenantID: tenantID, LegalEntityID: entityID, CycleID: created.Cycle.ID,
		ExpectedVersion: challenging.Version, ActorID: reviewerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != rcsa.StatusCompleted || completed.ChallengeMatterID != matter.Matter.ID {
		t.Fatalf("completed cycle=%#v", completed)
	}
	persisted, err := matters.GetMatter(trusted, tenantID, matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	decision := continuity.CurrentDecisionForType(persisted.Decisions, rcsaChallengeDecisionType)
	if decision == nil || decision.Status != continuity.DecisionConditionallyApproved ||
		decision.SelectedOption != "DEFICIENCY_CONFIRMED" || decision.AuthorityPrincipalID != authorizerID {
		t.Fatalf("persisted challenge decision=%#v", decision)
	}
	if persisted.Matter.Status == continuity.MatterClosed || persisted.Matter.Status == continuity.MatterCancelled {
		t.Fatalf("challenge Matter was closed by RCSA completion: %#v", persisted.Matter)
	}
}

func challengeID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

//go:build postgres && postgresintegration

package risk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRiskLifecycleIsScopedVersionedAndAtomic(t *testing.T) {
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

	tenantID := mustRiskID(t)
	entityA := mustRiskID(t)
	entityB := mustRiskID(t)
	ownerID := mustRiskID(t)
	reviewerID := mustRiskID(t)
	authorizerID := mustRiskID(t)
	missingPrincipal := mustRiskID(t)
	suffix := tenantID[len(tenantID)-8:]
	entityACode := "RISK-A-" + suffix
	entityBCode := "RISK-B-" + suffix
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, "risk-"+suffix, "Risk Test "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$2::uuid,$3,'Risk Entity A','NG',$6),
			($4::uuid,$2::uuid,$5,'Risk Entity B','GH',$6)
	`, entityA, tenantID, entityACode, entityB, entityBCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$2::uuid,'PERSON','Risk owner','ACTIVE',$5),
			($3::uuid,$2::uuid,'PERSON','Risk reviewer','ACTIVE',$5),
			($4::uuid,$2::uuid,'PERSON','Risk authorizer','ACTIVE',$5)
	`, ownerID, tenantID, reviewerID, authorizerID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	service := NewService(repository)
	service.Now = func() time.Time { return now }

	created, err := service.Create(ctx, CreateInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, Code: "NET-" + suffix,
		Name: "Network resilience", Category: "Operational resilience",
		Statement: "Critical network service may exceed approved recovery tolerance.",
		Cause:     "Primary and recovery paths can become unavailable.", Event: "Network service interruption",
		Impact: "Customers cannot access critical services within the approved tolerance.",
		Scope:  json.RawMessage(`{"service":"critical-network"}`), OwnerPrincipalID: ownerID, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID || created.LegalEntityID != entityA || created.Version != 1 {
		t.Fatalf("created risk = %#v", created)
	}
	if _, err := service.Get(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityBCode}, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity risk read error = %v", err)
	}

	now = now.Add(time.Minute)
	riskWithAppetite, appetite, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 1,
		Statement: "Keep disruption below 30 minutes.", Rule: json.RawMessage(`{"max_minutes":30}`),
		Rationale: "Protect critical customer service.", OwnerPrincipalID: ownerID, ActorID: authorizerID,
		EffectiveFrom: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if riskWithAppetite.Version != 2 || appetite.Version != 1 || appetite.RiskVersion != 2 {
		t.Fatalf("appetite result risk=%#v appetite=%#v", riskWithAppetite, appetite)
	}
	currentAppetite, err := repository.CurrentAppetite(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, created.ID, now)
	if err != nil || currentAppetite == nil || currentAppetite.ID != appetite.ID {
		t.Fatalf("current appetite=%#v err=%v", currentAppetite, err)
	}

	now = now.Add(time.Minute)
	_, _, err = service.AddAssessment(ctx, AssessmentInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 2,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: appetite.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Recovery exceeds tolerance.", ActorID: missingPrincipal,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid assessor error = %v", err)
	}
	afterFailure, err := service.Get(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.Risk.Version != 2 || len(afterFailure.Assessments) != 0 {
		t.Fatalf("failed assessment changed risk: %#v", afterFailure)
	}

	now = now.Add(time.Minute)
	updated, assessment, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 2,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: appetite.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Recovery exceeds tolerance.", ActorID: reviewerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 3 || assessment.RiskVersion != 3 {
		t.Fatalf("assessment version risk=%d assessment=%d", updated.Version, assessment.RiskVersion)
	}

	page, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Risk.ID != created.ID ||
		page.Items[0].LatestAssessment == nil || page.Items[0].LatestAssessment.ID != assessment.ID ||
		page.Items[0].ActiveAppetite == nil || page.Items[0].ActiveAppetite.ID != appetite.ID {
		t.Fatalf("risk page = %#v", page)
	}

	for table, want := range map[string]int{"risk_revisions": 3, "risk_events": 3} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND risk_id=$3::uuid"
		if err := pool.QueryRow(ctx, query, tenantID, entityA, created.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s count=%d want=%d", table, count, want)
		}
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM outbox_events
		WHERE tenant_id=$1::uuid AND aggregate_type='RISK' AND aggregate_id=$2::uuid
	`, tenantID, created.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 3 {
		t.Fatalf("risk outbox count=%d want=3", outboxCount)
	}

	now = now.Add(time.Minute)
	updatedAfterAssessment, err := service.Update(ctx, UpdateInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedVersion: updated.Version,
		Name: updated.Name, Category: updated.Category, Statement: updated.Statement,
		Cause: updated.Cause, Event: updated.Event, Impact: updated.Impact, Scope: updated.Scope,
		OwnerPrincipalID: updated.OwnerPrincipalID, Status: StatusActive, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	staleFiltered, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedAfterAssessment.Version != assessment.RiskVersion+1 || len(staleFiltered.Items) != 0 {
		t.Fatalf("stale assessment drove current appetite filter: risk=%#v page=%#v", updatedAfterAssessment, staleFiltered)
	}
}

func mustRiskID(t *testing.T) string {
	t.Helper()
	value, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

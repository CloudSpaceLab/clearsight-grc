//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNigerianBankReferenceJourneys(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}

	const (
		tenantID    = "77777777-7777-7777-8777-777777777771"
		entityID    = "77777777-7777-7777-8777-777777777772"
		actorID     = "77777777-7777-7777-8777-777777777773"
		ownerID     = "77777777-7777-7777-8777-777777777774"
		reviewerID  = "77777777-7777-7777-8777-777777777775"
		signatoryID = "77777777-7777-7777-8777-777777777776"
	)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'vertical-bank','Reference Bank Nigeria')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'BANK-NG','Reference Bank Nigeria','Nigeria')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	ctx = continuity.WithTrustedSystemEntityScope(ctx, "vertical-bank", entityID)
	for _, principal := range []struct{ id, name string }{{actorID, "Amaka Okafor"}, {ownerID, "Data Protection Owner"}, {reviewerID, "Control Assurance Reviewer"}, {signatoryID, "Executive Signatory"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON',$3)`, principal.id, tenantID, principal.name); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	continuityRepo := continuity.NewPostgresRepository(pool)
	continuityService := continuity.NewServiceWithClock(continuityRepo, clock)
	evidenceService := evidence.NewServiceWithClock(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore(), clock)
	service := bankverticals.NewService(continuityService, evidenceService)
	config := bankverticals.SeedConfig{TenantID: "vertical-bank", LegalEntityID: entityID, BankName: "Reference Bank Nigeria", ActorID: actorID, OwnerPrincipalID: ownerID, ReviewerPrincipalID: reviewerID, SignatoryPrincipalID: signatoryID, Now: now}

	journeys, err := service.SeedSample(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	projectionAt := now.Add(time.Hour)
	maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "vertical-test-worker", Now: func() time.Time { return projectionAt }}
	for {
		completed, maintainErr := maintainer.Maintain(ctx, projectionAt, 20)
		if maintainErr != nil {
			t.Fatal(maintainErr)
		}
		if completed == 0 {
			break
		}
	}
	journeys, err = service.List(ctx, "vertical-bank")
	if err != nil {
		t.Fatal(err)
	}
	if len(journeys) != 4 {
		t.Fatalf("expected four journeys, got %d", len(journeys))
	}

	var programs, requirements, evidenceChecks, matters, requests int
	queries := []struct {
		query string
		out   *int
	}{
		{`SELECT count(*) FROM programs WHERE tenant_id=$1::uuid`, &programs},
		{`SELECT count(*) FROM program_requirements WHERE tenant_id=$1::uuid`, &requirements},
		{`SELECT count(*) FROM evidence_contracts WHERE tenant_id=$1::uuid`, &evidenceChecks},
		{`SELECT count(*) FROM matters WHERE tenant_id=$1::uuid`, &matters},
		{`SELECT count(*) FROM capture_requests WHERE tenant_id=$1::uuid`, &requests},
	}
	for _, check := range queries {
		if err := pool.QueryRow(ctx, check.query, tenantID).Scan(check.out); err != nil {
			t.Fatal(err)
		}
	}
	if programs != 1 || requirements != 5 || evidenceChecks != 5 || matters != 3 || requests != 2 {
		t.Fatalf("unexpected vertical records programs=%d requirements=%d checks=%d matters=%d requests=%d", programs, requirements, evidenceChecks, matters, requests)
	}

	var acknowledged, closedFindings, passedChecks int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM response_packages WHERE tenant_id=$1::uuid AND status='ACKNOWLEDGED'`, tenantID).Scan(&acknowledged); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND matter_type='AUDIT_FINDING' AND status='CLOSED'`, tenantID).Scan(&closedFindings); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM verification_results WHERE tenant_id=$1::uuid AND result='PASS'`, tenantID).Scan(&passedChecks); err != nil {
		t.Fatal(err)
	}
	if acknowledged != 1 || closedFindings != 1 || passedChecks != 1 {
		t.Fatalf("journeys did not reach their required outcomes acknowledged=%d closed_findings=%d passed_checks=%d", acknowledged, closedFindings, passedChecks)
	}

	if _, err := service.SeedSample(ctx, config); err != nil {
		t.Fatalf("repeat seed failed: %v", err)
	}
	var repeatedMatters int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid`, tenantID).Scan(&repeatedMatters); err != nil {
		t.Fatal(err)
	}
	if repeatedMatters != matters {
		t.Fatalf("repeat seed created duplicate matters: before=%d after=%d", matters, repeatedMatters)
	}
}

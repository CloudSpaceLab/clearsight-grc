//go:build postgres && postgresintegration

package governance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDelegationCoversPlannedAndEmergencyAbsenceFailClosed(t *testing.T) {
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

	const (
		tenantID    = "8a999999-9999-7999-8999-999999999901"
		entityID    = "8a999999-9999-7999-8999-999999999902"
		giverID     = "8a999999-9999-7999-8999-999999999903"
		recipientID = "8a999999-9999-7999-8999-999999999904"
		makerID     = "8a999999-9999-7999-8999-999999999905"
		checkerID   = "8a999999-9999-7999-8999-999999999906"
		giverPosID  = "8a999999-9999-7999-8999-999999999907"
		targetPosID = "8a999999-9999-7999-8999-999999999908"
	)
	cleanup := func(cleanCtx context.Context) {
		for _, query := range []string{
			"DELETE FROM outbox_events WHERE tenant_id=$1::uuid",
			"DELETE FROM governance_decisions WHERE tenant_id=$1::uuid",
			"DELETE FROM delegations WHERE tenant_id=$1::uuid",
			"DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid",
			"DELETE FROM org_positions WHERE tenant_id=$1::uuid",
			"DELETE FROM principals WHERE tenant_id=$1::uuid",
			"DELETE FROM legal_entities WHERE tenant_id=$1::uuid",
			"DELETE FROM tenants WHERE id=$1::uuid",
		} {
			_, _ = pool.Exec(cleanCtx, query, tenantID)
		}
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	mustGovernanceExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'absence-governance-test','Absence governance test')`, tenantID)
	mustGovernanceExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour))
	mustGovernanceExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$5::uuid,'PERSON','Responsibility holder','ACTIVE',$6),
			($2::uuid,$5::uuid,'PERSON','Acting holder','ACTIVE',$6),
			($3::uuid,$5::uuid,'PERSON','Maker','ACTIVE',$6),
			($4::uuid,$5::uuid,'PERSON','Checker','ACTIVE',$6)`,
		giverID, recipientID, makerID, checkerID, tenantID, now.Add(-time.Hour))
	mustGovernanceExec(t, ctx, pool, `
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES
			($1::uuid,$3::uuid,$4::uuid,'OWNER','Owner',$5::uuid,$7),
			($2::uuid,$3::uuid,$4::uuid,'ACTING','Acting owner',$6::uuid,$7)`,
		giverPosID, targetPosID, tenantID, entityID, giverID, recipientID, now.Add(-time.Hour))
	for _, responsibility := range []string{"REVIEWER", "PERFORMER", "AUTHORIZER"} {
		mustGovernanceExec(t, ctx, pool, `
			INSERT INTO responsibility_assignments(
				tenant_id,legal_entity_id,principal_id,responsibility,object_type,priority,valid_from,valid_until,policy_version
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'LEGAL_ENTITY',100,$5,$6,'absence-test:v1')`,
			tenantID, entityID, giverID, responsibility, now.Add(-time.Hour), now.Add(24 * time.Hour))
	}

	repo := NewPostgresRepository(pool)
	service := NewService(repo)
	service.now = func() time.Time { return now }
	scope := json.RawMessage(`{"legal_entity_id":"` + entityID + `"}`)

	planned, err := service.CreateDelegation(ctx, CreateDelegationInput{
		TenantID: tenantID, LegalEntityID: entityID, FromPrincipalID: giverID, ToPrincipalID: recipientID,
		Responsibility: "REVIEWER", Scope: scope, StartsAt: now.Add(time.Hour), EndsAt: now.Add(3 * time.Hour),
		Reason: "planned leave", MakerID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	planned, err = service.SubmitDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: planned.ID, ActorID: makerID, ExpectedVersion: planned.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	planned, err = service.ApproveDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: planned.ID, ActorID: checkerID,
		ExpectedVersion: planned.Version, Rationale: "coverage reviewed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if planned.Status != DelegationApproved {
		t.Fatalf("planned absence activated early: %#v", planned)
	}
	if count, err := repo.ActivateDueDelegations(ctx, now.Add(30 * time.Minute), 10); err != nil || count != 0 {
		t.Fatalf("early planned activation count=%d err=%v", count, err)
	}
	if count, err := repo.ActivateDueDelegations(ctx, now.Add(time.Hour), 10); err != nil || count != 1 {
		t.Fatalf("due planned activation count=%d err=%v", count, err)
	}
	planned, err = repo.GetDelegationForEntity(ctx, tenantID, entityID, planned.ID)
	if err != nil || planned.Status != DelegationActive {
		t.Fatalf("planned delegation after start=%#v err=%v", planned, err)
	}
	if count, err := repo.ExpireDueDelegations(ctx, now.Add(3 * time.Hour), 10); err != nil || count != 1 {
		t.Fatalf("planned expiry count=%d err=%v", count, err)
	}
	planned, err = repo.GetDelegationForEntity(ctx, tenantID, entityID, planned.ID)
	if err != nil || planned.Status != DelegationExpired {
		t.Fatalf("planned delegation after end=%#v err=%v", planned, err)
	}

	emergency, err := service.CreateDelegation(ctx, CreateDelegationInput{
		TenantID: tenantID, LegalEntityID: entityID, FromPrincipalID: giverID, ToPrincipalID: recipientID,
		Responsibility: "PERFORMER", Scope: scope, StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour),
		Reason: "emergency leave", MakerID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	emergency, err = service.SubmitDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: emergency.ID, ActorID: makerID, ExpectedVersion: emergency.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	emergency, err = service.ApproveDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: emergency.ID, ActorID: checkerID,
		ExpectedVersion: emergency.Version, Rationale: "emergency coverage reviewed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if emergency.Status != DelegationActive {
		t.Fatalf("emergency delegation was not active immediately: %#v", emergency)
	}

	future, err := service.CreateDelegation(ctx, CreateDelegationInput{
		TenantID: tenantID, LegalEntityID: entityID, FromPrincipalID: giverID, ToPrincipalID: recipientID,
		Responsibility: "AUTHORIZER", Scope: scope, StartsAt: now.Add(2 * time.Hour), EndsAt: now.Add(4 * time.Hour),
		Reason: "future leave", MakerID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	future, err = service.SubmitDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: future.ID, ActorID: makerID, ExpectedVersion: future.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	future, err = service.ApproveDelegation(ctx, TransitionInput{
		TenantID: tenantID, LegalEntityID: entityID, ID: future.ID, ActorID: checkerID,
		ExpectedVersion: future.Version, Rationale: "future coverage reviewed",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustGovernanceExec(t, ctx, pool, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, giverID)
	if count, err := repo.ActivateDueDelegations(ctx, now.Add(2 * time.Hour), 10); !errors.Is(err, ErrDelegationEligibility) || count != 0 {
		t.Fatalf("ineligible future activation count=%d err=%v", count, err)
	}
	future, err = repo.GetDelegationForEntity(ctx, tenantID, entityID, future.ID)
	if err != nil || future.Status != DelegationApproved {
		t.Fatalf("failed future activation mutated delegation=%#v err=%v", future, err)
	}
}

func mustGovernanceExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

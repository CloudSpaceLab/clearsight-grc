//go:build postgres && postgresintegration

package oploss

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresOperationalLossIsScopedVersionedAndRecoverySafe(t *testing.T) {
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

	tenantID := mustLossID(t)
	entityA := mustLossID(t)
	entityB := mustLossID(t)
	scopeA := mustLossID(t)
	scopeChildA := mustLossID(t)
	scopeSiblingA := mustLossID(t)
	scopeB := mustLossID(t)
	ownerID := mustLossID(t)
	riskA := mustLossID(t)
	riskB := mustLossID(t)
	matterA := mustLossID(t)
	matterB := mustLossID(t)
	suffix := tenantID[len(tenantID)-8:]
	tenantSlug := "loss-" + suffix
	entityACode, entityBCode := "LOSS-A-"+suffix, "LOSS-B-"+suffix
	now := time.Date(2026, 10, 3, 18, 15, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, tenantSlug, "Loss Test "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$2::uuid,$3,'Loss Entity A','NG',$6::timestamptz),
			($4::uuid,$2::uuid,$5,'Loss Entity B','GH',$6::timestamptz)
	`, entityA, tenantID, entityACode, entityB, entityBCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from,created_at,updated_at,version) VALUES
			($1::uuid,$2::uuid,$3::uuid,'BR-A','Branch A','BRANCH',ARRAY['BRANCH A'],'MANAGED','ACTIVE',$6,$6,$6,1),
			($4::uuid,$2::uuid,$5::uuid,'BR-B','Branch B','BRANCH',ARRAY['BRANCH B'],'MANAGED','ACTIVE',$6,$6,$6,1)
	`, scopeA, tenantID, entityA, scopeB, entityB, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from,created_at,updated_at,version) VALUES
			($1::uuid,$2::uuid,$3::uuid,$4::uuid,'BR-A-OPS','Branch A Operations','DEPARTMENT',ARRAY['BRANCH A','OPERATIONS'],'MANAGED','ACTIVE',$6,$6,$6,1),
			($5::uuid,$2::uuid,$3::uuid,NULL,'FIN-A','Finance A','DEPARTMENT',ARRAY['FINANCE A'],'MANAGED','ACTIVE',$6,$6,$6,1)
	`, scopeChildA, tenantID, entityA, scopeA, scopeSiblingA, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Loss owner','ACTIVE',$3::timestamptz)
	`, ownerID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO risks(
			id,tenant_id,legal_entity_id,code,name,statement,impact,scope,owner_principal_id,status,version,created_at,updated_at) VALUES
			($1::uuid,$2::uuid,$3::uuid,$4,'Risk A','Loss may occur.','Material financial loss.','{}'::jsonb,$8::uuid,'ACTIVE',1,$9,$9),
			($5::uuid,$2::uuid,$6::uuid,$7,'Risk B','Loss may occur.','Material financial loss.','{}'::jsonb,$8::uuid,'ACTIVE',1,$9,$9)
	`, riskA, tenantID, entityA, "RISK-A-"+suffix, riskB, entityB, "RISK-B-"+suffix, ownerID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(
			id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,
			known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at,version) VALUES
			($1::uuid,$2::uuid,$3::uuid,$4,'OPERATIONAL_LOSS','DRAFT',2,'Loss A','Loss A follow-up','{}'::jsonb,
			 '{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$8::uuid,$9,$9,1),
			($5::uuid,$2::uuid,$6::uuid,$7,'OPERATIONAL_LOSS','DRAFT',2,'Loss B','Loss B follow-up','{}'::jsonb,
			 '{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$8::uuid,$9,$9,1)
	`, matterA, tenantID, entityA, "LOSS-M-A-"+suffix, matterB, entityB, "LOSS-M-B-"+suffix, ownerID, now); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewPostgresRepository(pool))
	service.Now = func() time.Time { return now }
	created, err := service.Create(ctx, CreateInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, OrganizationScopeID: scopeA,
		Code: "LOSS-" + suffix, Title: "Duplicate settlement loss",
		EventType: EventExecutionDeliveryProcess, Cause: "Duplicate settlement instruction.",
		Description:      "Duplicate settlement completed before correction.",
		GrossAmountMinor: 500_000_000, Currency: "NGN",
		OccurredAt: now.Add(-2 * time.Hour), DiscoveredAt: now.Add(-time.Hour),
		RiskID: riskA, MatterID: matterA, OwnerPrincipalID: ownerID, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID || created.LegalEntityID != entityA ||
		created.OrganizationScopeID != scopeA || created.RiskID != riskA || created.MatterID != matterA ||
		created.Version != 1 {
		t.Fatalf("created=%#v", created)
	}
	for _, value := range []struct {
		code  string
		scope string
	}{
		{code: "LOSS-CHILD-" + suffix, scope: scopeChildA},
		{code: "LOSS-SIBLING-" + suffix, scope: scopeSiblingA},
	} {
		if _, err := service.Create(ctx, CreateInput{
			TenantID: tenantSlug, LegalEntityID: entityACode, OrganizationScopeID: value.scope,
			Code: value.code, Title: value.code, EventType: EventOther, Cause: "Scoped loss.",
			GrossAmountMinor: 10_000, Currency: "NGN",
			OccurredAt: now.Add(-2 * time.Hour), DiscoveredAt: now.Add(-time.Hour),
			OwnerPrincipalID: ownerID, ActorID: ownerID,
		}); err != nil {
			t.Fatal(err)
		}
	}
	scopedPage, err := service.List(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityACode}, ListFilter{
		OrganizationScopeID: scopeA, OrganizationScopeIDs: []string{scopeA, scopeChildA}, Limit: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scopedPage.OrganizationScopeID != scopeA || len(scopedPage.Items) != 2 {
		t.Fatalf("scoped loss page=%#v", scopedPage)
	}
	for _, item := range scopedPage.Items {
		if item.Loss.OrganizationScopeID != scopeA && item.Loss.OrganizationScopeID != scopeChildA {
			t.Fatalf("scope membership leaked %q: %#v", item.Loss.OrganizationScopeID, scopedPage)
		}
	}
	if _, err := service.Get(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityBCode}, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity read error=%v", err)
	}

	now = now.Add(time.Minute)
	recoveredLoss, recovery, err := service.AddRecovery(ctx, RecoveryInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, LossID: created.ID, ExpectedVersion: 1,
		Kind: RecoveryCash, AmountMinor: 200_000_000, Reference: "Insurer settlement",
		RecoveredAt: now, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if recoveredLoss.Version != 2 || recovery.LossVersion != 2 {
		t.Fatalf("recovery loss=%#v recovery=%#v", recoveredLoss, recovery)
	}
	aggregate, err := service.Get(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityACode}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.Totals.RecoveredAmountMinor != 200_000_000 || aggregate.Totals.NetLossMinor != 300_000_000 {
		t.Fatalf("totals=%#v", aggregate.Totals)
	}

	now = now.Add(time.Minute)
	_, _, err = service.AddRecovery(ctx, RecoveryInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, LossID: created.ID, ExpectedVersion: 2,
		Kind: RecoveryCash, AmountMinor: 400_000_000, Reference: "Invalid over recovery",
		RecoveredAt: now, ActorID: ownerID,
	})
	if !errors.Is(err, ErrRecoveryLimit) {
		t.Fatalf("over recovery error=%v", err)
	}
	afterFailure, err := service.Get(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityACode}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.Loss.Version != 2 || len(afterFailure.Recoveries) != 1 ||
		afterFailure.Totals.RecoveredAmountMinor != 200_000_000 {
		t.Fatalf("failed recovery mutated loss=%#v", afterFailure)
	}

	now = now.Add(time.Minute)
	reversedLoss, _, err := service.AddRecovery(ctx, RecoveryInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, LossID: created.ID, ExpectedVersion: 2,
		Kind: RecoveryReversal, AmountMinor: 50_000_000, Reference: "Recovery correction",
		RecoveredAt: now, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.Get(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityACode}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reversedLoss.Version != 3 || aggregate.Totals.RecoveredAmountMinor != 150_000_000 ||
		aggregate.Totals.NetLossMinor != 350_000_000 {
		t.Fatalf("reversal aggregate=%#v", aggregate)
	}

	for table, want := range map[string]int{
		"operational_loss_revisions":  3,
		"operational_loss_events":     3,
		"operational_loss_recoveries": 2,
	} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND loss_id=$3::uuid"
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
		WHERE tenant_id=$1::uuid AND aggregate_type='OPERATIONAL_LOSS' AND aggregate_id=$2::uuid
	`, tenantID, created.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 3 {
		t.Fatalf("loss outbox count=%d want=3", outboxCount)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE operational_loss_recoveries SET reference='tampered'
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND loss_id=$3::uuid
	`, tenantID, entityA, created.ID); err == nil {
		t.Fatal("immutable recovery update unexpectedly succeeded")
	}

	badLossID := mustLossID(t)
	_, err = pool.Exec(ctx, `
		INSERT INTO operational_losses(
			id,tenant_id,legal_entity_id,organization_scope_id,code,title,event_type,cause,description,
			gross_amount_minor,currency,occurred_at,discovered_at,risk_id,matter_id,owner_principal_id,
			status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'Wrong scope','OTHER','Cause.','',
		       100,'NGN',$6,$6,$7::uuid,$8::uuid,$9::uuid,'ACTIVE',1,$6,$6)
	`, badLossID, tenantID, entityA, scopeB, "BAD-"+suffix, now, riskB, matterB, ownerID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-entity loss insert error=%v, want foreign-key rejection", err)
	}
}

func mustLossID(t *testing.T) string {
	t.Helper()
	value, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

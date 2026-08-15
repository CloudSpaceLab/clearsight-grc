//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	scopedHealthTenantID        = "7e111111-1111-7111-8111-111111111111"
	scopedHealthActorID         = "7e222222-2222-7222-8222-222222222222"
	scopedHealthSourceID        = "7e333333-3333-7333-8333-333333333333"
	scopedHealthConnectionID    = "7e444444-4444-7444-8444-444444444444"
	scopedHealthConnection2ID   = "7e455555-5555-7555-8555-555555555555"
	scopedHealthViewID          = "7e555555-5555-7555-8555-555555555555"
	scopedHealthBindingID       = "7e666666-6666-7666-8666-666666666666"
	scopedHealthForeignTenantID = "7e911111-1111-7111-8111-111111111111"
	scopedHealthForeignActorID  = "7e922222-2222-7222-8222-222222222222"
	scopedHealthForeignSourceID = "7e933333-3333-7333-8333-333333333333"
)

func TestPostgresScopedSourceHealthAggregatesExactResourceState(t *testing.T) {
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
	cleanupScopedHealthFixture(ctx, pool)
	t.Cleanup(func() { cleanupScopedHealthFixture(context.Background(), pool) })

	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	seedScopedHealthFixture(t, ctx, pool, now)
	repository := NewPostgresRepository(pool)
	service := NewService(repository, NewMemoryObjectStore())
	service.now = func() time.Time { return now }

	// A Source with no observations must still be a valid maintenance candidate.
	if changed, err := service.Maintain(ctx, now, 20); err != nil || changed != 0 {
		t.Fatalf("empty health rollup changed=%d err=%v", changed, err)
	}

	connectionObservation := SourceObservation{
		TenantID: scopedHealthTenantID, SourceID: scopedHealthSourceID, Scope: ObservationScopeConnection,
		ConnectionID: scopedHealthConnectionID, ConnectionVersion: 1,
		ObservedAt: now, Success: true, RecordedBy: scopedHealthActorID,
	}
	updated, err := service.RecordSourceObservation(ctx, connectionObservation)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthCurrent {
		t.Fatalf("healthy Connection did not produce CURRENT Source health: %#v", updated)
	}

	now = now.Add(time.Minute)
	bindingFailure := SourceObservation{
		TenantID: scopedHealthTenantID, SourceID: scopedHealthSourceID, Scope: ObservationScopeBinding,
		ConnectionID: scopedHealthConnectionID, ConnectionVersion: 1,
		ViewID: scopedHealthViewID, ViewVersion: 1,
		BindingID: scopedHealthBindingID, BindingVersion: 1,
		ObservedAt: now, Unavailable: true, RecordedBy: scopedHealthActorID,
	}
	updated, err = service.RecordSourceObservation(ctx, bindingFailure)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthUnavailable {
		t.Fatalf("Binding outage was hidden by healthy Connection: %#v", updated)
	}

	now = now.Add(time.Minute)
	recovery := bindingFailure
	recovery.ObservedAt = now
	recovery.Unavailable = false
	recovery.Success = true
	updated, err = service.RecordSourceObservation(ctx, recovery)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthCurrent {
		t.Fatalf("Source did not recover after the Binding recovered: %#v", updated)
	}

	scopes, err := service.ListSourceScopeHealth(ctx, scopedHealthTenantID, scopedHealthSourceID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 {
		t.Fatalf("scope count=%d want=2: %#v", len(scopes), scopes)
	}
	var sawConnection, sawBinding bool
	for _, scope := range scopes {
		switch scope.Scope {
		case ObservationScopeConnection:
			sawConnection = scope.ConnectionID == scopedHealthConnectionID && scope.ConnectionVersion == 1 && scope.Health == HealthCurrent
		case ObservationScopeBinding:
			sawBinding = scope.ConnectionID == scopedHealthConnectionID && scope.ViewID == scopedHealthViewID && scope.BindingID == scopedHealthBindingID && scope.Health == HealthCurrent
		}
	}
	if !sawConnection || !sawBinding {
		t.Fatalf("exact scoped health lineage missing: %#v", scopes)
	}

	// All individual FKs below are valid, but the View/Binding belongs to the first
	// Connection. The hierarchy trigger must reject the internally inconsistent scope.
	_, err = pool.Exec(ctx, `
		INSERT INTO source_observations(
			id,tenant_id,source_id,observed_at,success,unavailable,recorded_by,
			scope_kind,connection_id,connection_version,view_id,view_version,binding_id,binding_version
		) VALUES (
			'7e777777-7777-7777-8777-777777777777'::uuid,$1::uuid,$2::uuid,$3,true,false,$4::uuid,
			'BINDING',$5::uuid,1,$6::uuid,1,$7::uuid,1
		)`, scopedHealthTenantID, scopedHealthSourceID, now.Add(time.Minute), scopedHealthActorID,
		scopedHealthConnection2ID, scopedHealthViewID, scopedHealthBindingID)
	if err == nil {
		t.Fatal("database accepted a Binding health observation with mismatched Connection lineage")
	}

	// A raw SOURCE observation must not pair one tenant with another tenant's Source.
	_, err = pool.Exec(ctx, `
		INSERT INTO source_observations(
			id,tenant_id,source_id,observed_at,success,unavailable,recorded_by,scope_kind
		) VALUES (
			'7e788888-8888-7888-8888-888888888888'::uuid,$1::uuid,$2::uuid,$3,true,false,$4::uuid,'SOURCE'
		)`, scopedHealthTenantID, scopedHealthForeignSourceID, now.Add(time.Minute), scopedHealthActorID)
	if err == nil {
		t.Fatal("database accepted a Source observation under the wrong tenant")
	}

	// recorded_by is provenance, not decoration; it must be a principal in the same tenant.
	_, err = pool.Exec(ctx, `
		INSERT INTO source_observations(
			id,tenant_id,source_id,observed_at,success,unavailable,recorded_by,scope_kind
		) VALUES (
			'7e799999-9999-7999-8999-999999999999'::uuid,$1::uuid,$2::uuid,$3,true,false,$4::uuid,'SOURCE'
		)`, scopedHealthTenantID, scopedHealthSourceID, now.Add(time.Minute), scopedHealthForeignActorID)
	if err == nil {
		t.Fatal("database accepted a source observation recorded by another tenant's principal")
	}

	// Database writes bypassing the service get the same bounded clock-skew rule.
	_, err = pool.Exec(ctx, `
		INSERT INTO source_observations(
			id,tenant_id,source_id,observed_at,success,unavailable,recorded_by,scope_kind
		) VALUES (
			'7e7aaaaa-aaaa-7aaa-8aaa-aaaaaaaaaaaa'::uuid,$1::uuid,$2::uuid,clock_timestamp()+interval '6 minutes',true,false,$3::uuid,'SOURCE'
		)`, scopedHealthTenantID, scopedHealthSourceID, scopedHealthActorID)
	if err == nil {
		t.Fatal("database accepted a far-future source observation")
	}
}

func seedScopedHealthFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-health-test','Source health test')`, scopedHealthTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-health-actor','Source health actor')`, scopedHealthActorID, scopedHealthTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'HEALTH-SOURCE','Health source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,30,'UNKNOWN','ACTIVE',1,$4,$4)`,
		scopedHealthSourceID, scopedHealthTenantID, scopedHealthActorID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	for revisionID, connectionID := range map[string]string{
		"7e811111-1111-7111-8111-111111111111": scopedHealthConnectionID,
		"7e822222-2222-7222-8222-222222222222": scopedHealthConnection2ID,
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO source_connections(
				revision_id,connection_id,tenant_id,source_id,code,name,adapter_kind,adapter_version,secret_ref,definition,
				declared_capabilities,verified_capabilities,owner_principal_id,status,is_current,effective_from,version,created_by,created_at,updated_at
			) VALUES (
				$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$5,'POSTGRES','postgres-v1','vault://health/reader','{}'::jsonb,
				'["PAGE"]'::jsonb,'["PAGE"]'::jsonb,$6::uuid,'ACTIVE',true,$7,1,$6::uuid,$7,$7
			)`, revisionID, connectionID, scopedHealthTenantID, scopedHealthSourceID, "CONNECTION_"+connectionID[len(connectionID)-4:], scopedHealthActorID, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_views(
			revision_id,view_id,tenant_id,source_id,connection_id,connection_version,code,name,definition,output_kind,
			stable_keys,native_schema,schema_fingerprint,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7e833333-3333-7333-8333-333333333333'::uuid,$1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'ACCOUNTS','Accounts',
			'{"query":"SELECT account_id FROM accounts"}'::jsonb,'RECORDS','["account_id"]'::jsonb,
			'[{"name":"account_id","native_type":"uuid","nullable":false}]'::jsonb,$5,'ACTIVE',true,$6,1,$7::uuid,$6,$6
		)`, scopedHealthViewID, scopedHealthTenantID, scopedHealthSourceID, scopedHealthConnectionID, strings.Repeat("a", 64), now.Add(-time.Hour), scopedHealthActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_bindings(
			revision_id,binding_id,tenant_id,source_id,view_id,view_version,code,name,purpose,operations,selected_fields,key_fields,
			page_rows,response_bytes,lookup_values,timeout_ms,mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,
			sensitivity_handling,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7e844444-4444-7444-8444-444444444444'::uuid,$1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'ACCOUNT_PAGE','Account page','health-test',
			'["PAGE"]'::jsonb,'["account_id"]'::jsonb,'["account_id"]'::jsonb,25,65536,10,2000,
			'{}'::jsonb,'{}'::jsonb,'{}'::jsonb,30,'REQUIRE_COMPLETE','{}'::jsonb,'ACTIVE',true,$5,1,$6::uuid,$5,$5
		)`, scopedHealthBindingID, scopedHealthTenantID, scopedHealthSourceID, scopedHealthViewID, now.Add(-time.Hour), scopedHealthActorID); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-health-foreign','Source health foreign')`, scopedHealthForeignTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-health-foreign-actor','Source health foreign actor')`, scopedHealthForeignActorID, scopedHealthForeignTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'FOREIGN-SOURCE','Foreign source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,30,'UNKNOWN','ACTIVE',1,$4,$4)`,
		scopedHealthForeignSourceID, scopedHealthForeignTenantID, scopedHealthForeignActorID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func cleanupScopedHealthFixture(ctx context.Context, pool *pgxpool.Pool) {
	for _, tenantID := range []string{scopedHealthTenantID, scopedHealthForeignTenantID} {
		for _, statement := range []string{
			`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
			`DELETE FROM source_binding_checkpoints WHERE tenant_id=$1::uuid`,
			`DELETE FROM source_observations WHERE tenant_id=$1::uuid`,
			`DELETE FROM source_bindings WHERE tenant_id=$1::uuid`,
			`DELETE FROM source_views WHERE tenant_id=$1::uuid`,
			`DELETE FROM source_connections WHERE tenant_id=$1::uuid`,
			`DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`,
			`DELETE FROM principals WHERE tenant_id=$1::uuid`,
			`DELETE FROM tenants WHERE id=$1::uuid`,
		} {
			_, _ = pool.Exec(ctx, statement, tenantID)
		}
	}
}

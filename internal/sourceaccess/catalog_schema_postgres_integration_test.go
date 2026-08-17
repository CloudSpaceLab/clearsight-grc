//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	schemaGuardTenantID             = "7a111111-1111-7111-8111-111111111111"
	schemaGuardActorID              = "7a222222-2222-7222-8222-222222222222"
	schemaGuardSourceID             = "7a333333-3333-7333-8333-333333333333"
	schemaGuardConnectionRevisionID = "7a444444-4444-7444-8444-444444444444"
	schemaGuardConnectionID         = "7a555555-5555-7555-8555-555555555555"
	schemaGuardViewRevisionID       = "7a888888-8888-7888-8888-888888888888"
	schemaGuardViewID               = "7a999999-9999-7999-8999-999999999999"
)

func TestPostgresCatalogRejectsSchemaDriftAtStorageBoundary(t *testing.T) {
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
	cleanupSchemaGuardFixture(ctx, pool)
	t.Cleanup(func() { cleanupSchemaGuardFixture(context.Background(), pool) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-schema-guard','Source schema guard')`, schemaGuardTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-schema-guard-actor','Source schema guard actor')`, schemaGuardActorID, schemaGuardTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'SCHEMA-GUARD','Schema guard source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, schemaGuardSourceID, schemaGuardTenantID, schemaGuardActorID); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	repository := NewPostgresCatalogRepository(pool)
	connection := catalogConnectionRevision(now)
	connection.RevisionID = schemaGuardConnectionRevisionID
	connection.ConnectionID = schemaGuardConnectionID
	connection.TenantID = schemaGuardTenantID
	connection.SourceID = schemaGuardSourceID
	connection.OwnerPrincipalID = schemaGuardActorID
	connection.CreatedBy = schemaGuardActorID
	createdConnection, err := repository.CreateConnectionRevision(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO source_views(
			revision_id,view_id,tenant_id,source_id,connection_id,connection_version,
			code,name,definition,output_kind,stable_keys,native_schema,schema_fingerprint,
			status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7a666666-6666-7666-8666-666666666666'::uuid,
			'7a777777-7777-7777-8777-777777777777'::uuid,
			$1::uuid,$2::uuid,$3::uuid,$4,
			'INVALID_STABLE_KEYS','Invalid stable keys',
			'{"query":"SELECT account_id FROM active_accounts"}'::jsonb,'RECORDS',
			'["missing_key"]'::jsonb,
			'[{"name":"account_id","native_type":"uuid","nullable":false}]'::jsonb,
			$5,'ACTIVE',true,$6,1,$7::uuid,$6,$6
		)`, schemaGuardTenantID, schemaGuardSourceID, schemaGuardConnectionID, createdConnection.Version, strings.Repeat("a", 64), now, schemaGuardActorID); err == nil {
		t.Fatal("database accepted a stable key outside the view native schema")
	}

	view := catalogViewRevision(now)
	view.RevisionID = schemaGuardViewRevisionID
	view.ViewID = schemaGuardViewID
	view.TenantID = schemaGuardTenantID
	view.SourceID = schemaGuardSourceID
	view.ConnectionID = schemaGuardConnectionID
	view.ConnectionVersion = createdConnection.Version
	view.CreatedBy = schemaGuardActorID
	createdView, err := repository.CreateViewRevision(ctx, view)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		revisionID string
		bindingID  string
		code       string
		operations string
		selected   string
		keys       string
	}{
		{
			name:       "selected field outside native schema",
			revisionID: "7b111111-1111-7111-8111-111111111111",
			bindingID:  "7b222222-2222-7222-8222-222222222222",
			code:       "INVALID_SELECTED_FIELD",
			operations: `["AGGREGATE"]`,
			selected:   `["missing_field"]`,
			keys:       `[]`,
		},
		{
			name:       "key is not a stable selected key",
			revisionID: "7b333333-3333-7333-8333-333333333333",
			bindingID:  "7b444444-4444-7444-8444-444444444444",
			code:       "INVALID_KEY_FIELD",
			operations: `["LOOKUP"]`,
			selected:   `["account_id","status"]`,
			keys:       `["status"]`,
		},
		{
			name:       "page operation without key",
			revisionID: "7b555555-5555-7555-8555-555555555555",
			bindingID:  "7b666666-6666-7666-8666-666666666666",
			code:       "PAGE_WITHOUT_KEY",
			operations: `["PAGE"]`,
			selected:   `["account_id"]`,
			keys:       `[]`,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := insertSchemaGuardBinding(ctx, pool, createdView, now, testCase.revisionID, testCase.bindingID, testCase.code, testCase.operations, testCase.selected, testCase.keys); err == nil {
				t.Fatalf("database accepted invalid binding: %s", testCase.name)
			}
		})
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO source_connections(
			revision_id,connection_id,tenant_id,source_id,code,name,adapter_kind,adapter_version,
			secret_ref,definition,declared_capabilities,verified_capabilities,owner_principal_id,
			status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7b777777-7777-7777-8777-777777777777'::uuid,
			'7b888888-8888-7888-8888-888888888888'::uuid,
			$1::uuid,$2::uuid,'BAD_REFERENCE','Bad reference','REFERENCE','reference-v1','',
			$3::jsonb,'[]'::jsonb,'[]'::jsonb,$4::uuid,
			'ACTIVE',true,$5,1,$4::uuid,$5,$5
		)`, schemaGuardTenantID, schemaGuardSourceID, `{"endpoint":"https://example.invalid/source\nvalue"}`, schemaGuardActorID, now); err == nil {
		t.Fatal("database accepted a reference endpoint with a control character")
	}
}

func insertSchemaGuardBinding(ctx context.Context, pool *pgxpool.Pool, view ViewRevision, now time.Time, revisionID, bindingID, code, operations, selected, keys string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO source_bindings(
			revision_id,binding_id,tenant_id,source_id,view_id,view_version,code,name,purpose,
			operations,selected_fields,key_fields,page_rows,response_bytes,lookup_values,timeout_ms,
			mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,
			sensitivity_handling,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$7,'schema-bound-access',
			$8::jsonb,$9::jsonb,$10::jsonb,25,65536,10,2000,
			'{}'::jsonb,'{}'::jsonb,'{}'::jsonb,15,'REQUIRE_COMPLETE',
			'{}'::jsonb,'ACTIVE',true,$11,1,$12::uuid,$11,$11
		)`, revisionID, bindingID, schemaGuardTenantID, schemaGuardSourceID, view.ViewID, view.Version, code, operations, selected, keys, now, schemaGuardActorID)
	return err
}

func cleanupSchemaGuardFixture(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, schemaGuardTenantID)
}

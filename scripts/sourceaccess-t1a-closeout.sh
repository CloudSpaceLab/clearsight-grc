#!/usr/bin/env bash
set -euo pipefail

python - <<'PY'
from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if new in text:
        return text
    if old not in text:
        raise SystemExit(f"{label} anchor not found")
    return text.replace(old, new, 1)


postgres = Path("internal/evidence/postgres.go")
text = postgres.read_text()
text = replace_once(
    text,
    "jsonb_build_object('endpoint',$7)",
    "jsonb_build_object('endpoint',$7::text)",
    "reference endpoint parameter",
)
postgres.write_text(text)

validation = Path("internal/sourceaccess/catalog_validation.go")
text = validation.read_text()
stable_key_anchor = '''\tif value.SchemaFingerprint != "" && !isLowerHex(value.SchemaFingerprint, 64) {\n'''
stable_key_guard = '''\tfor _, key := range value.StableKeys {\n\t\tif _, exists := seen[key]; !exists {\n\t\t\treturn ViewRevision{}, catalogInvalid("stable keys must exist in the native schema")\n\t\t}\n\t}\n\tif value.SchemaFingerprint != "" && !isLowerHex(value.SchemaFingerprint, 64) {\n'''
text = replace_once(text, stable_key_anchor, stable_key_guard, "stable-key validation")
validation.write_text(text)

migration = Path("migrations/000030_source_access_catalog.up.sql")
text = migration.read_text()
text = replace_once(
    text,
    "AND NULLIF(btrim(definition->>'endpoint'),'') IS NOT NULL));",
    "AND NULLIF(btrim(definition->>'endpoint'),'') IS NOT NULL AND NOT ((definition->>'endpoint') ~ '[[:cntrl:]]')));",
    "reference endpoint constraint",
)

view_anchor = '''    IF parent_kind='REFERENCE' THEN
        RAISE EXCEPTION 'reference connections cannot own executable views' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
'''
view_guard = '''    IF parent_kind='REFERENCE' THEN
        RAISE EXCEPTION 'reference connections cannot own executable views' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.stable_keys) AS keys(key_name)
         WHERE NOT EXISTS (
             SELECT 1
               FROM jsonb_array_elements(NEW.native_schema) AS fields(field)
              WHERE fields.field->>'name'=keys.key_name
         )
    ) THEN
        RAISE EXCEPTION 'source view stable keys must exist in the native schema' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
'''
text = replace_once(text, view_anchor, view_guard, "source-view storage guard")

binding_declaration = '''CREATE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    parent_current boolean;
BEGIN
'''
binding_declaration_guard = '''CREATE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    parent_current boolean;
    parent_stable_keys jsonb;
    parent_native_schema jsonb;
BEGIN
'''
text = replace_once(text, binding_declaration, binding_declaration_guard, "source-binding declaration")

binding_anchor = '''    SELECT is_current INTO parent_current
      FROM source_views
     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version;
    IF NEW.is_current AND NOT COALESCE(parent_current,false) THEN
        RAISE EXCEPTION 'current source binding requires its current view revision' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
'''
binding_guard = '''    SELECT is_current,stable_keys,native_schema
      INTO parent_current,parent_stable_keys,parent_native_schema
      FROM source_views
     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version;
    IF NEW.is_current AND NOT COALESCE(parent_current,false) THEN
        RAISE EXCEPTION 'current source binding requires its current view revision' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.selected_fields) AS selected(field_name)
         WHERE NOT EXISTS (
             SELECT 1
               FROM jsonb_array_elements(parent_native_schema) AS fields(field)
              WHERE fields.field->>'name'=selected.field_name
         )
    ) THEN
        RAISE EXCEPTION 'source binding selected fields must exist in the view schema' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.key_fields) AS keys(key_name)
         WHERE NOT (NEW.selected_fields ? keys.key_name)
            OR NOT (parent_stable_keys ? keys.key_name)
    ) THEN
        RAISE EXCEPTION 'source binding key fields must be selected stable view keys' USING ERRCODE='23514';
    END IF;
    IF (NEW.operations ? 'PAGE' OR NEW.operations ? 'LOOKUP') AND jsonb_array_length(NEW.key_fields)=0 THEN
        RAISE EXCEPTION 'page and lookup bindings require a stable key' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
'''
text = replace_once(text, binding_anchor, binding_guard, "source-binding storage guard")

legacy_anchor = '''    IF EXISTS (
        SELECT 1 FROM evidence_sources
         WHERE btrim(endpoint)<>''
           AND octet_length(jsonb_build_object('endpoint',btrim(endpoint))::text)>32768
    ) THEN
        RAISE EXCEPTION 'legacy evidence source endpoint exceeds the source connection definition limit';
    END IF;
'''
legacy_guard = '''    IF EXISTS (
        SELECT 1 FROM evidence_sources
         WHERE btrim(endpoint)<>''
           AND (
               octet_length(jsonb_build_object('endpoint',btrim(endpoint))::text)>32768
               OR endpoint ~ '[[:cntrl:]]'
           )
    ) THEN
        RAISE EXCEPTION 'legacy evidence source endpoint is invalid for source connection migration';
    END IF;
'''
text = replace_once(text, legacy_anchor, legacy_guard, "legacy endpoint migration guard")
migration.write_text(text)

rollback = Path("migrations/000030_source_access_catalog.down.sql")
text = rollback.read_text()
rollback_anchor = "            OR NULLIF(btrim(definition->>'endpoint'),'') IS NULL\n"
rollback_guard = "            OR NULLIF(btrim(definition->>'endpoint'),'') IS NULL\n            OR definition <> jsonb_build_object('endpoint',definition->>'endpoint')\n"
text = replace_once(text, rollback_anchor, rollback_guard, "rollback reference shape guard")
rollback.write_text(text)

Path("internal/sourceaccess/catalog_schema_validation_test.go").write_text(r'''package sourceaccess

import (
	"errors"
	"testing"
	"time"
)

func TestCatalogViewStableKeysMustExistInNativeSchema(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	view := catalogViewRevision(now)
	view.StableKeys = []string{"missing_key"}
	if _, err := normalizeViewRevision(view); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("stable key outside native schema should fail, got %v", err)
	}
}
''')

Path("internal/sourceaccess/catalog_schema_postgres_integration_test.go").write_text(r'''//go:build postgres && postgresintegration

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
''')
PY

gofmt -w \
  internal/evidence/postgres.go \
  internal/sourceaccess/catalog_validation.go \
  internal/sourceaccess/catalog_schema_validation_test.go \
  internal/sourceaccess/catalog_schema_postgres_integration_test.go

git diff --check
go mod download
go test ./internal/sourceaccess ./internal/evidence
go test -tags postgres ./internal/sourceaccess ./internal/evidence

for migration in migrations/*.up.sql; do
  PGPASSWORD=clearsight psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -f "$migration"
done
TEST_DATABASE_URL=postgres://clearsight:clearsight@localhost:5432/clearsight?sslmode=disable \
  go test -p 1 -tags "postgres postgresintegration" ./internal/sourceaccess ./internal/evidence

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git rm .github/workflows/sourceaccess-t1a-closeout.yml scripts/sourceaccess-t1a-closeout.sh
git add \
  internal/evidence/postgres.go \
  internal/sourceaccess/catalog_validation.go \
  internal/sourceaccess/catalog_schema_validation_test.go \
  internal/sourceaccess/catalog_schema_postgres_integration_test.go \
  migrations/000030_source_access_catalog.up.sql \
  migrations/000030_source_access_catalog.down.sql
git commit -m "fix(sourceaccess): close T1a catalog invariants"
git push origin HEAD:codex/issue-61-sourceaccess-t1a
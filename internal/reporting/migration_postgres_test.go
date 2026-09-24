//go:build postgres

package reporting

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReportMigrationRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured; real PostgreSQL migration round trip was not run")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()

	schema := fmt.Sprintf("reporting_migration_%d", time.Now().UnixNano())
	if _, err := connection.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	}()
	if _, err := connection.Exec(ctx, "SET search_path TO "+schema+", public"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = connection.Exec(context.Background(), "RESET search_path") }()

	if _, err := connection.Exec(ctx, `
		CREATE FUNCTION uuidv7() RETURNS uuid LANGUAGE sql VOLATILE AS
		'SELECT md5(random()::text || clock_timestamp()::text)::uuid';
		CREATE TABLE tenants(id uuid PRIMARY KEY);
		CREATE TABLE legal_entities(
			id uuid NOT NULL,
			tenant_id uuid NOT NULL,
			PRIMARY KEY (id, tenant_id)
		);
		CREATE TABLE principals(
			id uuid NOT NULL,
			tenant_id uuid NOT NULL,
			PRIMARY KEY (id, tenant_id)
		);
		CREATE TABLE programs(
			id uuid NOT NULL,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			PRIMARY KEY (id, tenant_id, legal_entity_id)
		);
		CREATE TABLE matters(
			id uuid NOT NULL,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			PRIMARY KEY (id, tenant_id, legal_entity_id)
		);
		CREATE TABLE ropa_processing_activities(
			id uuid PRIMARY KEY,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			code text NOT NULL,
			name text NOT NULL,
			controller text NOT NULL,
			version bigint NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			UNIQUE (id, tenant_id, legal_entity_id)
		);
		CREATE TABLE ropa_processing_activity_reviews(
			id uuid PRIMARY KEY,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			activity_id uuid NOT NULL,
			completed_at timestamptz,
			outcome text,
			FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
				REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
		);
	`); err != nil {
		t.Fatal(err)
	}

	up := readReportingMigration(t, "../../migrations/000093_report_builder.up.sql")
	down := readReportingMigration(t, "../../migrations/000093_report_builder.down.sql")
	if _, err := connection.Exec(ctx, up); err != nil {
		t.Fatalf("apply migration 000093: %v", err)
	}

	const (
		tenantID      = "00000000-0000-7000-8000-000000000001"
		entityID      = "00000000-0000-7000-8000-000000000002"
		matterID      = "00000000-0000-7000-8000-000000000003"
		programID     = "00000000-0000-7000-8000-000000000004"
		activityID    = "00000000-0000-7000-8000-000000000005"
		definitionID  = "00000000-0000-7000-8000-000000000006"
		runID         = "00000000-0000-7000-8000-000000000007"
		otherEntityID = "00000000-0000-7000-8000-000000000008"
	)
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := connection.Exec(ctx, `
		INSERT INTO tenants(id) VALUES($1::uuid);
		INSERT INTO legal_entities(id, tenant_id) VALUES($2::uuid,$1::uuid),($3::uuid,$1::uuid);
		INSERT INTO programs(id, tenant_id, legal_entity_id) VALUES($4::uuid,$1::uuid,$2::uuid);
		INSERT INTO matters(id, tenant_id, legal_entity_id) VALUES($5::uuid,$1::uuid,$2::uuid);
		INSERT INTO ropa_processing_activities
			(id, tenant_id, legal_entity_id, code, name, controller, version, created_at, updated_at, matter_id)
			VALUES($6::uuid,$1::uuid,$2::uuid,'PA-001','Customer onboarding','Fidelity Bank',1,$7,$7,$5::uuid);
	`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, otherEntityID, programID, matterID, activityID, now); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `
		INSERT INTO report_definitions
			(id, tenant_id, legal_entity_id, code, name, dataset, scope_kind, format, status,
			 current_version, checksum, maker_id, version, created_at, updated_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,'ROPA-EXCEPTIONS','Open processing exceptions',
				'PROCESSING_ACTIVITY_EXCEPTIONS','LEGAL_ENTITY','CSV','DRAFT',1,$4,'maker-1',1,
				clock_timestamp(),clock_timestamp());
		INSERT INTO report_definition_revisions
			(definition_id, tenant_id, legal_entity_id, version, base_version, dataset, scope_kind,
			 format, filter, checksum, maker_id, decision)
			VALUES($1::uuid,$2::uuid,$3::uuid,1,0,'PROCESSING_ACTIVITY_EXCEPTIONS','LEGAL_ENTITY','CSV',
				'{}'::jsonb,$5,'maker-1','PROPOSED');
	`, pgx.QueryExecModeSimpleProtocol, definitionID, tenantID, entityID, strings.Repeat("a", 64), strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `
		INSERT INTO report_definition_revisions
			(definition_id, tenant_id, legal_entity_id, version, base_version, dataset, scope_kind,
			 format, filter, checksum, maker_id, approved_by, approved_at, decision)
			VALUES($1::uuid,$2::uuid,$3::uuid,2,1,'PROCESSING_ACTIVITY_EXCEPTIONS','LEGAL_ENTITY','CSV',
				'{}'::jsonb,$4,'maker-1','checker-1',$5,'APPROVED');
		UPDATE report_definitions
		SET current_version=2, status='ACTIVE', checker_id='checker-1', approved_at=$5, effective_from=$5,
			updated_at=clock_timestamp(), version=version+1
		WHERE id=$1::uuid;
		INSERT INTO report_runs
			(id, tenant_id, legal_entity_id, definition_id, definition_version, requested_by_ref,
			 as_of, filter, dataset, format, status, created_at, expires_at)
			VALUES($6::uuid,$7::uuid,$8::uuid,$1::uuid,2,'request-1',$5,'{}'::jsonb,
				'PROCESSING_ACTIVITY_EXCEPTIONS','CSV','QUEUED',$5,$5::timestamptz + interval '1 day');
		UPDATE report_runs
		SET status='READY', data_object_key='reports/run/data.csv', data_sha256=$9,
			manifest_object_key='reports/run/manifest.json', manifest_sha256=$10,
			completed_at=$5
		WHERE id=$6::uuid;
	`, pgx.QueryExecModeSimpleProtocol, definitionID, tenantID, entityID, strings.Repeat("c", 64), now, runID, tenantID, entityID, strings.Repeat("e", 64), strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `UPDATE report_definition_revisions SET decision_note='changed' WHERE definition_id=$1::uuid AND version=1`, definitionID); err == nil {
		t.Fatal("immutable report definition revision unexpectedly accepted an update")
	}
	if _, err := connection.Exec(ctx, `UPDATE report_runs SET status='RUNNING' WHERE id=$1::uuid`, runID); err == nil {
		t.Fatal("terminal report run unexpectedly re-opened")
	}

	if _, err := connection.Exec(ctx, down); err == nil {
		t.Fatal("down migration unexpectedly erased populated report history")
	} else if !strings.Contains(err.Error(), "refusing to erase report history") {
		t.Fatalf("down migration returned an unexpected refusal: %v", err)
	}
	_, _ = connection.Exec(ctx, "ROLLBACK")
	if _, err := connection.Exec(ctx, `TRUNCATE report_runs, report_definition_revisions, report_definitions`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, down); err != nil {
		t.Fatalf("apply migration 000093 down after cleanup: %v", err)
	}

	var reportTable *string
	if err := connection.QueryRow(ctx, `SELECT to_regclass($1)::text`, schema+".report_definitions").Scan(&reportTable); err != nil {
		t.Fatal(err)
	}
	if reportTable != nil {
		t.Fatalf("report_definitions still exists after down migration: %q", *reportTable)
	}
	var matterColumn bool
	if err := connection.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema=$1 AND table_name='ropa_processing_activities' AND column_name='matter_id'
	)`, schema).Scan(&matterColumn); err != nil {
		t.Fatal(err)
	}
	if matterColumn {
		t.Fatal("matter_id still exists after down migration")
	}
}

func readReportingMigration(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(body)
}

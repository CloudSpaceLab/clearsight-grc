//go:build postgres

package reporting

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		t.Fatalf("apply migration 000093: %v%s", err, migrationErrorLocation(up, err))
	}

	const (
		tenantID          = "00000000-0000-7000-8000-000000000001"
		entityID          = "00000000-0000-7000-8000-000000000002"
		matterID          = "00000000-0000-7000-8000-000000000003"
		programID         = "00000000-0000-7000-8000-000000000004"
		activityID        = "00000000-0000-7000-8000-000000000005"
		definitionID      = "00000000-0000-7000-8000-000000000006"
		runID             = "00000000-0000-7000-8000-000000000007"
		otherEntityID     = "00000000-0000-7000-8000-000000000008"
		makerID           = "00000000-0000-7000-8000-000000000009"
		reviewerID        = "00000000-0000-7000-8000-000000000010"
		authorizerID      = "00000000-0000-7000-8000-000000000011"
		invalidDefinition = "00000000-0000-7000-8000-000000000012"
	)
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := connection.Exec(ctx, `
		INSERT INTO tenants(id) VALUES($1::uuid);
		INSERT INTO legal_entities(id, tenant_id) VALUES($2::uuid,$1::uuid),($3::uuid,$1::uuid);
		INSERT INTO principals(id, tenant_id)
			VALUES($4::uuid,$1::uuid),($5::uuid,$1::uuid),($6::uuid,$1::uuid);
		INSERT INTO programs(id, tenant_id, legal_entity_id) VALUES($7::uuid,$1::uuid,$2::uuid);
		INSERT INTO matters(id, tenant_id, legal_entity_id) VALUES($8::uuid,$1::uuid,$2::uuid);
		INSERT INTO ropa_processing_activities
			(id, tenant_id, legal_entity_id, code, name, controller, version, created_at, updated_at, matter_id)
			VALUES($9::uuid,$1::uuid,$2::uuid,'PA-001','Customer onboarding','Fidelity Bank',1,$10,$10,$8::uuid);
	`, pgx.QueryExecModeSimpleProtocol,
		tenantID, entityID, otherEntityID, makerID, reviewerID, authorizerID,
		programID, matterID, activityID, now,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `
		INSERT INTO report_definitions
			(id, tenant_id, legal_entity_id, code, name, dataset, scope_kind, format, status,
			 current_version, checksum, maker_id, version, created_at, updated_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,'PROGRAM-HEALTH','Program health across the entity',
				'PROGRAMS','LEGAL_ENTITY','CSV','DRAFT',1,$4,$5::uuid,1,
				clock_timestamp(),clock_timestamp());
		INSERT INTO report_definition_revisions
			(definition_id, tenant_id, legal_entity_id, version, base_version, dataset, scope_kind,
			 format, filter, checksum, maker_id, decision)
			VALUES($1::uuid,$2::uuid,$3::uuid,1,0,'PROGRAMS','LEGAL_ENTITY','CSV',
				'{}'::jsonb,$4,$5::uuid,'PROPOSED');
	`, pgx.QueryExecModeSimpleProtocol,
		definitionID, tenantID, entityID, strings.Repeat("a", 64), makerID,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `
		INSERT INTO report_definitions
			(id, tenant_id, legal_entity_id, code, name, dataset, scope_kind, format, status,
			 current_version, checksum, maker_id, version, created_at, updated_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,'INVALID-ACTOR','Invalid actor reference',
				'PROGRAMS','LEGAL_ENTITY','CSV','DRAFT',1,$4,'maker-1',1,
				clock_timestamp(),clock_timestamp())
	`, pgx.QueryExecModeSimpleProtocol, invalidDefinition, tenantID, entityID, strings.Repeat("1", 64)); err == nil {
		t.Fatal("report definition accepted a free-text maker instead of a tenant-scoped principal")
	}

	if _, err := connection.Exec(ctx, `
		UPDATE report_definition_revisions
		SET decision='REVIEWED', reviewed_by=$2::uuid, reviewed_at=$3, decision_note='Scope and filter checked.'
		WHERE definition_id=$1::uuid AND version=1
	`, pgx.QueryExecModeSimpleProtocol, definitionID, reviewerID, now); err != nil {
		t.Fatalf("record review on immutable revision content: %v", err)
	}
	if _, err := connection.Exec(ctx, `
		UPDATE report_definition_revisions
		SET filter='{"kind":"group","operator":"and","children":[{"kind":"condition","field":"forged","operator":"is","value":"x"}]}'::jsonb
		WHERE definition_id=$1::uuid AND version=1
	`, definitionID); err == nil {
		t.Fatal("immutable report definition revision accepted changed content")
	}
	if _, err := connection.Exec(ctx, `
		UPDATE report_definition_revisions
		SET decision='APPROVED', approved_by=$2::uuid, approved_at=$3, decision_note='Authorized for the recorded revision.'
		WHERE definition_id=$1::uuid AND version=1
	`, pgx.QueryExecModeSimpleProtocol, definitionID, authorizerID, now); err != nil {
		t.Fatalf("record approval after review: %v", err)
	}
	if _, err := connection.Exec(ctx, `
		UPDATE report_definition_revisions SET decision='REVIEWED'
		WHERE definition_id=$1::uuid AND version=1
	`, definitionID); err == nil {
		t.Fatal("decided report definition revision unexpectedly moved backwards")
	}
	if _, err := connection.Exec(ctx, `DELETE FROM report_definition_revisions WHERE definition_id=$1::uuid`, definitionID); err == nil {
		t.Fatal("report definition revision was deleted")
	}

	if _, err := connection.Exec(ctx, `
		UPDATE report_definitions
		SET status='REVIEWED', reviewer_id=$3::uuid, reviewer_note='Scope and filter checked.',
			updated_at=clock_timestamp(), version=version+1
		WHERE id=$1::uuid;
		UPDATE report_definitions
		SET status='ACTIVE', checker_id=$5::uuid, approved_at=$4, effective_from=$4,
			updated_at=clock_timestamp(), version=version+1
		WHERE id=$1::uuid;
		INSERT INTO report_runs
			(id, tenant_id, legal_entity_id, definition_id, definition_version, definition_code,
			 definition_checksum, scope_kind, requested_by_ref, as_of, source_boundary,
			 filter, dataset, format, status, created_at, expires_at)
			VALUES($6::uuid,$2::uuid,$7::uuid,$1::uuid,1,'PROGRAM-HEALTH',$8,
				'LEGAL_ENTITY','request-1',$4,
				jsonb_build_object(
					'captured_at',$4,
					'projection_version','program-summary.v1',
					'source_high_water',jsonb_build_object('programs',$4),
					'population',1,
					'population_complete',true
				),
				'{}'::jsonb,'PROGRAMS','CSV','QUEUED',$4,$4::timestamptz + interval '1 day');
		UPDATE report_runs
		SET status='READY', data_object_key='reports/run/data.csv', data_sha256=$9,
			manifest_object_key='reports/run/manifest.json', manifest_sha256=$10,
			completed_at=$4
		WHERE id=$6::uuid;
	`, pgx.QueryExecModeSimpleProtocol,
		definitionID, tenantID, reviewerID, now, authorizerID, runID, entityID,
		strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64),
	); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, `UPDATE report_runs SET status='RUNNING' WHERE id=$1::uuid`, runID); err == nil {
		t.Fatal("terminal report run unexpectedly re-opened")
	}

	if _, err := connection.Exec(ctx, down); err == nil {
		t.Fatal("down migration unexpectedly erased populated report history")
	} else if !strings.Contains(err.Error(), "refusing to erase report history") {
		t.Fatalf("down migration returned an unexpected refusal: %v", err)
	}
	if _, err := connection.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `TRUNCATE report_runs, report_definition_revisions, report_definitions`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, down); err != nil {
		t.Fatalf("apply migration 000093 down after cleanup: %v", err)
	}
	if _, err := connection.Exec(ctx, up); err != nil {
		t.Fatalf("reapply amended migration 000093: %v", err)
	}
	if _, err := connection.Exec(ctx, down); err != nil {
		t.Fatalf("roll back reapplied migration 000093: %v", err)
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

func migrationErrorLocation(sql string, err error) string {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Position <= 0 || int(postgresError.Position) > len(sql) {
		return ""
	}
	position := int(postgresError.Position) - 1
	line := 1 + strings.Count(sql[:position], "\n")
	return fmt.Sprintf(" (server position %d, line %d)", postgresError.Position, line)
}

func readReportingMigration(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(body)
}

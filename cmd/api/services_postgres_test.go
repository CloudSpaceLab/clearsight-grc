//go:build postgres

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
	"github.com/jackc/pgx/v5"
)

type postgresDemoCounts struct {
	Activities          int64
	ActivityRevisions   int64
	ActivityRevisionMax string
	Definitions         int64
	DefinitionRevisions int64
	Runs                int64
}

func TestPostgresCompositionInstallsRopaAndReportingDemoOnlyInDemoMode(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	databaseURL := createPostgresDemoDatabase(t, baseURL)
	seedPostgresDemoPrerequisites(t, databaseURL)
	cfg := postgresCompositionTestConfig(t, databaseURL)

	// Demo mode off must seed nothing. The config is copied per case because
	// reusing one value would run both builds in demo mode and prove nothing.
	offConfig := postgresCompositionTestConfig(t, databaseURL)
	offConfig.DemoMode = false
	off, err := buildServices(t.Context(), offConfig, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	offCounts := readPostgresDemoCounts(t, databaseURL)
	off.Close()
	if offCounts.Activities != 0 || offCounts.Definitions != 0 || offCounts.Runs != 0 {
		t.Fatalf("demo-off composition populated data: %+v", offCounts)
	}

	on, err := buildServices(t.Context(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := assertPostgresDemoComposition(t, on, databaseURL)
	on.Close()

	secondServices, err := buildServices(t.Context(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	second := readPostgresDemoCounts(t, databaseURL)
	secondServices.Close()
	if second != first {
		t.Fatalf("second PostgreSQL demo install changed counts or revision history: first=%+v second=%+v", first, second)
	}
}

func createPostgresDemoDatabase(t *testing.T, baseURL string) string {
	t.Helper()
	ctx := t.Context()
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminURL := *parsed
	adminURL.Path = "/postgres"
	admin, err := pgx.Connect(ctx, adminURL.String())
	if err != nil {
		t.Fatal(err)
	}
	databaseName := fmt.Sprintf("ropa_demo_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatal(err)
	}
	if err = admin.Close(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cleanupErr := pgx.Connect(context.Background(), adminURL.String())
		if cleanupErr != nil {
			t.Logf("connect for demo database cleanup: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, cleanupErr = cleanup.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)"); cleanupErr != nil {
			t.Logf("drop demo database %s: %v", databaseName, cleanupErr)
		}
	})

	databaseURL := *parsed
	databaseURL.Path = "/" + databaseName
	return databaseURL.String()
}

func seedPostgresDemoPrerequisites(t *testing.T, databaseURL string) {
	t.Helper()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	migrations, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(migrations)
	for _, migration := range migrations {
		body, readErr := os.ReadFile(migration)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, execErr := conn.Exec(ctx, string(body)); execErr != nil {
			t.Fatalf("apply %s: %v", filepath.Base(migration), execErr)
		}
	}

	foundationBytes, err := os.ReadFile(filepath.Join("..", "..", "deploy", "scripts", "seed-demo-foundation.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// The checked-out script may carry CRLF line endings, so normalise before
	// locating the heredoc rather than assuming LF.
	foundation := strings.ReplaceAll(string(foundationBytes), "\r\n", "\n")
	startMarker := "<<'SQL'\n"
	start := strings.Index(foundation, startMarker)
	if start < 0 {
		t.Fatal("demo foundation SQL heredoc start not found")
	}
	start += len(startMarker)
	end := strings.Index(foundation[start:], "\nSQL")
	if end < 0 {
		t.Fatal("demo foundation SQL heredoc end not found")
	}
	foundation = strings.ReplaceAll(foundation[start:start+end], ":'demo_staff_email'", "''")
	if _, err = conn.Exec(ctx, foundation); err != nil {
		t.Fatalf("seed demo foundation: %v", err)
	}

	programID, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	matterID, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	ownerID := demoPrincipalID(t, conn, "demo-program-owner")
	authorityID := demoPrincipalID(t, conn, "demo-cro")
	if _, err = conn.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,
			owner_principal_id,authority_principal_id,jurisdiction,scope,effective_from)
		VALUES($1::uuid,'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',
			'NDPA-2023','Nigeria data protection','PRIVACY','ACTIVE','Data Protection Office',$2::uuid,$3::uuid,
			'Nigeria','{"sample":true}'::jsonb,clock_timestamp())`,
		programID, ownerID, authorityID); err != nil {
		t.Fatalf("seed demo Program prerequisite: %v", err)
	}
	if _, err = conn.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,
			scope,trigger_type,trigger_key,owner_principal_id,required_authority,due_at)
		VALUES($1::uuid,'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',
			'DEMO-GAID-2025','REGULATORY_CHANGE','ASSESSMENT',4,'Implement the annual privacy return requirements',
			'Record the current return requirements and evidence owners.','{"sample":true}'::jsonb,
			'REQUIREMENT_CHANGED','sample:gaid-2025-car',$2::uuid,'AUTHORIZER',clock_timestamp()+interval '30 days')`,
		matterID, ownerID); err != nil {
		t.Fatalf("seed demo Matter prerequisite: %v", err)
	}
}

func demoPrincipalID(t *testing.T, conn *pgx.Conn, externalRef string) string {
	t.Helper()
	var id string
	if err := conn.QueryRow(t.Context(), `SELECT id::text FROM principals WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND external_ref=$1 AND status='ACTIVE' AND valid_until IS NULL`, externalRef).Scan(&id); err != nil {
		t.Fatalf("resolve %s: %v", externalRef, err)
	}
	return id
}

func postgresCompositionTestConfig(t *testing.T, databaseURL string) config.Config {
	t.Helper()
	return config.Config{
		Environment:                          "development",
		DatabaseURL:                          databaseURL,
		DatabaseMinConns:                     0,
		DatabaseMaxConns:                     4,
		QueryTimeout:                         10 * time.Second,
		ArtifactRoot:                         t.TempDir(),
		MaxArtifactBytes:                     1 << 20,
		CaptureSessionTTL:                    30 * time.Minute,
		DemoMode:                             true,
		DemoAllowUnscannedArtifacts:          true,
		DocumentImportAllowUnscannedAnalysis: true,
		DemoTenantID:                         identity.DurableDemoTenantID,
		DemoLegalEntityID:                    identity.DurableDemoLegalEntityID,
		DemoPrincipalID:                      identity.DurableDemoPrincipalCRO,
	}
}

func assertPostgresDemoComposition(t *testing.T, services serviceSet, databaseURL string) postgresDemoCounts {
	t.Helper()
	scope := ropa.ActivityScope{TenantID: identity.DurableDemoTenantID, LegalEntityID: identity.DurableDemoLegalEntityID}
	page, err := services.Ropa.ListActivities(t.Context(), scope, ropa.ListActivitiesFilter{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 7 {
		t.Fatalf("PostgreSQL demo ROPA live rows = %d, want 7", len(page.Rows))
	}
	namedOwner := false
	for _, activity := range page.Rows {
		if activity.OwnerPrincipalID == "" {
			continue
		}
		namedOwner = true
		if strings.TrimSpace(activity.OwnerDisplayName) == "" {
			t.Fatalf("PostgreSQL demo ROPA owner %s was not resolved to a display name", activity.OwnerPrincipalID)
		}
	}
	if !namedOwner {
		t.Fatal("PostgreSQL demo ROPA page has no owned activity to validate")
	}

	now := time.Now().UTC()
	actor := identity.Actor{TenantID: identity.DurableDemoTenantID, LegalEntityID: identity.DurableDemoLegalEntityID,
		PrincipalID: identity.DurableDemoPrincipalCRO, Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	reportScope := reporting.ReportScope{TenantID: identity.DurableDemoTenantID, LegalEntityID: identity.DurableDemoLegalEntityID}
	ctx := identity.WithActor(t.Context(), actor)
	definitions, err := services.Reporting.ListDefinitions(ctx, reportScope, true)
	if err != nil {
		t.Fatal(err)
	}
	runs, err := services.Reporting.ListRuns(ctx, reportScope, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	history, err := services.Reporting.ListRunHistory(ctx, reportScope, "", "", 50)
	if err != nil {
		t.Fatalf("list first PostgreSQL report history page without cursor: %v", err)
	}
	if len(history.Items) != len(runs) || len(history.Items) == 0 || history.Items[0].ID != runs[0].ID {
		t.Fatalf("PostgreSQL report history first page = %#v, runs = %#v", history.Items, runs)
	}
	if len(definitions) != 4 || len(runs) != 1 {
		t.Fatalf("PostgreSQL demo reports definitions=%d runs=%d, want 4 and 1", len(definitions), len(runs))
	}
	if runs[0].Status != reporting.RunFailed || runs[0].FailureCode != reporting.FailureRowLimitExceeded {
		t.Fatalf("PostgreSQL demo bounded-stop run = %#v", runs[0])
	}

	counts := readPostgresDemoCounts(t, databaseURL)
	// Log the counts so a CI run records what the deployed demo actually holds,
	// rather than only that an assertion happened to pass.
	t.Logf("postgres demo counts: activities=%d activity_revisions=%d definitions=%d definition_revisions=%d runs=%d",
		counts.Activities, counts.ActivityRevisions, counts.Definitions, counts.DefinitionRevisions, counts.Runs)
	if counts.Activities != 8 || counts.Definitions != 4 || counts.Runs != 1 || counts.DefinitionRevisions < 4 {
		t.Fatalf("PostgreSQL demo counts = %+v, want 8 activities, 4 definitions, at least 4 definition revisions and 1 run", counts)
	}
	return counts
}

func readPostgresDemoCounts(t *testing.T, databaseURL string) postgresDemoCounts {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(t.Context())
	var counts postgresDemoCounts
	if err := conn.QueryRow(t.Context(), `
		SELECT
			(SELECT count(*) FROM ropa_processing_activities WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002'),
			(SELECT count(*) FROM ropa_processing_activity_revisions WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002'),
			(SELECT COALESCE(max(recorded_at)::text,'never') FROM ropa_processing_activity_revisions WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002'),
			(SELECT count(*) FROM report_definitions WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002'),
			(SELECT count(*) FROM report_definition_revisions WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002'),
			(SELECT count(*) FROM report_runs WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002')`,
	).Scan(
		&counts.Activities,
		&counts.ActivityRevisions,
		&counts.ActivityRevisionMax,
		&counts.Definitions,
		&counts.DefinitionRevisions,
		&counts.Runs,
	); err != nil {
		t.Fatal(err)
	}
	return counts
}

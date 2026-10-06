//go:build postgres && postgresintegration

package rcsa

import (
	"context"
	"os"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCycleListIsScopedFilteredAndPaginated(t *testing.T) {
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

	tenantID := mustRCSAIntegrationID(t)
	entityID := mustRCSAIntegrationID(t)
	ownerID := mustRCSAIntegrationID(t)
	firstID := mustRCSAIntegrationID(t)
	secondID := mustRCSAIntegrationID(t)
	completedID := mustRCSAIntegrationID(t)
	suffix := tenantID[len(tenantID)-8:]
	tenantSlug := "rcsa-" + suffix
	entityCode := "RCSA-" + suffix
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, tenantSlug, "RCSA test "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'RCSA Entity','NG',$4)
	`, entityID, tenantID, entityCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','RCSA owner','ACTIVE',$3)
	`, ownerID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM rcsa_cycles WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})

	for _, cycle := range []struct {
		id      string
		code    string
		status  Status
		updated time.Time
	}{
		{id: firstID, code: "DRAFT-OLD", status: StatusDraft, updated: now},
		{id: secondID, code: "DRAFT-NEW", status: StatusDraft, updated: now.Add(time.Minute)},
		{id: completedID, code: "COMPLETE", status: StatusCompleted, updated: now.Add(2 * time.Minute)},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO rcsa_cycles(
				id,tenant_id,legal_entity_id,code,name,trigger_kind,first_line_owner_principal_id,
				status,population_checksum,version,created_at,updated_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$4,'SCHEDULED',$5::uuid,$6,$7,1,$8,$9)
		`, cycle.id, tenantID, entityID, cycle.code, ownerID, cycle.status,
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", now.Add(-time.Hour), cycle.updated); err != nil {
			t.Fatal(err)
		}
	}

	service := NewService(NewPostgresRepository(pool), nil)
	first, err := service.List(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityCode}, CycleFilter{Status: StatusDraft, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].Cycle.ID != secondID || first.NextCursor == "" || first.Items[0].RiskCount != 0 || first.Items[0].ControlCount != 0 {
		t.Fatalf("first page=%#v", first)
	}
	second, err := service.List(ctx, Scope{TenantID: tenantSlug, LegalEntityID: entityCode}, CycleFilter{Status: StatusDraft, Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Cycle.ID != firstID || second.NextCursor != "" {
		t.Fatalf("second page=%#v", second)
	}
}

func mustRCSAIntegrationID(t *testing.T) string {
	t.Helper()
	value, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

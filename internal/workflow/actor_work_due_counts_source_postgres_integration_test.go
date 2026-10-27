//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Counts and cursor traversal must reconcile on the same authorized source.
// Restricted Matters, other entities and other principals must not affect
// either population, including assignments beyond the first 50.
func TestActorWorkPostgresCountsReconcileWithVisibleCursorPages(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID         = "9f888888-8888-4888-8888-888888888801"
		entityID         = "9f888888-8888-4888-8888-888888888802"
		otherEntityID    = "9f888888-8888-4888-8888-888888888803"
		principalID      = "9f888888-8888-4888-8888-888888888804"
		otherPrincipal   = "9f888888-8888-4888-8888-888888888805"
		visibleMatter    = "9f888888-8888-4888-8888-888888888806"
		restrictedMatter = "9f888888-8888-4888-8888-888888888807"
		entityMatter     = "9f888888-8888-4888-8888-888888888808"
		otherMatter      = "9f888888-8888-4888-8888-888888888809"
		visibleFlow      = "9f888888-8888-4888-8888-888888888810"
		restrictedFlow   = "9f888888-8888-4888-8888-888888888811"
		entityFlow       = "9f888888-8888-4888-8888-888888888812"
		otherFlow        = "9f888888-8888-4888-8888-888888888813"
	)
	cleanup := func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		for _, stmt := range []string{
			"DELETE FROM workflow_tasks WHERE tenant_id=$1::uuid",
			"DELETE FROM workflow_events WHERE tenant_id=$1::uuid",
			"DELETE FROM workflow_instances WHERE tenant_id=$1::uuid",
			"DELETE FROM matters WHERE tenant_id=$1::uuid",
			"DELETE FROM principals WHERE tenant_id=$1::uuid",
			"DELETE FROM legal_entities WHERE tenant_id=$1::uuid",
			"DELETE FROM tenants WHERE id=$1::uuid",
		} {
			if _, cleanupErr := pool.Exec(cleanupCtx, stmt, tenantID); cleanupErr != nil {
				t.Logf("fixture cleanup: %v", cleanupErr)
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	now := time.Now().UTC()
	old := now.Add(-10 * time.Minute)
	late := now.Add(-24 * time.Hour)
	soon := now.Add(24 * time.Hour)
	later := now.Add(14 * 24 * time.Hour)

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'actor-counts-parity','Actor Counts Parity')`, tenantID)
	for _, entity := range []struct{ id, code string }{
		{entityID, "COUNT-A"}, {otherEntityID, "COUNT-B"},
	} {
		mustExec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
			VALUES($1::uuid,$2::uuid,$3,$3,'NG',$4)`, entity.id, tenantID, entity.code, old)
	}
	for _, principal := range []struct{ id, name string }{
		{principalID, "Counting reviewer"}, {otherPrincipal, "Other reviewer"},
	} {
		mustExec(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
			VALUES($1::uuid,$2::uuid,'PERSON',$3,'ACTIVE',$4)`, principal.id, tenantID, principal.name, old)
	}

	matters := []struct{ id, entity, reference, scope string }{
		{visibleMatter, entityID, "COUNT-VISIBLE", `{"access":"INTERNAL"}`},
		{restrictedMatter, entityID, "COUNT-RESTRICTED",
			`{"access":"RESTRICTED","allowed_principal_ids":["9f888888-8888-4888-8888-888888888805"]}`},
		{entityMatter, otherEntityID, "COUNT-OTHER-ENTITY", `{"access":"INTERNAL"}`},
		{otherMatter, entityID, "COUNT-OTHER-PRINCIPAL", `{"access":"INTERNAL"}`},
	}
	for _, matter := range matters {
		mustExec(`INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,
			priority,title,summary,scope,created_at,updated_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4,'CONTROL_GAP','TRIAGE',
			3,$4,'Count source parity',$5::jsonb,$6,$6)`,
			matter.id, tenantID, matter.entity, matter.reference, matter.scope, old)
	}
	for _, binding := range []struct{ id, matter string }{
		{visibleFlow, visibleMatter}, {restrictedFlow, restrictedMatter},
		{entityFlow, entityMatter}, {otherFlow, otherMatter},
	} {
		mustExec(`INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,
			state,policy_version,created_at,updated_at)
			VALUES($1::uuid,$2::uuid,'MATTER_LIFECYCLE','MATTER',$3::uuid,
			'ACTIVE','count-test-v1',$4,$4)`, binding.id, tenantID, binding.matter, old)
	}
	mustExec(`
		INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,
			title,status,due_at,created_at,updated_at)
		SELECT $1::uuid,$2::uuid,'review-'||series::text,'REVIEWER',$3::uuid,
			'Review '||series::text,'READY',
			CASE WHEN series<=15 THEN $4::timestamptz
			     WHEN series<=25 THEN $5::timestamptz
			     WHEN series<=77 THEN $6::timestamptz
			     ELSE NULL END,
			$7,$7
		FROM generate_series(1,81) AS series`,
		tenantID, visibleFlow, principalID, late, soon, later, old)
	for _, excluded := range []struct {
		flow, owner, step, status string
	}{
		{visibleFlow, principalID, "completed", "COMPLETED"},
		{restrictedFlow, principalID, "restricted", "READY"},
		{entityFlow, principalID, "other-entity", "READY"},
		{otherFlow, otherPrincipal, "other-principal", "READY"},
	} {
		mustExec(`INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,
			title,status,due_at,created_at,updated_at)
			VALUES($1::uuid,$2::uuid,$3,'REVIEWER',$4::uuid,$3,$5,$6,$7,$7)`,
			tenantID, excluded.flow, excluded.step, excluded.owner, excluded.status, late, old)
	}

	service := NewService(NewPostgresRepository(pool))
	counts, err := service.ReadActorWorkDueCounts(ctx, "actor-counts-parity", entityID, principalID)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Total != 81 || counts.Overdue != 15 || counts.DueSoon != 10 || counts.Undated != 4 {
		t.Fatalf("unauthorized or terminal assignments changed source counts: %#v", counts)
	}
	seen := map[string]struct{}{}
	query := ActorWorkPageQuery{
		TenantID: "actor-counts-parity", LegalEntityID: entityID, PrincipalID: principalID, Limit: 17,
	}
	for i := 0; i < 7; i++ {
		page, pageErr := service.ListActorWorkPage(ctx, query)
		if pageErr != nil {
			t.Fatal(pageErr)
		}
		for _, task := range page.Items {
			if _, duplicate := seen[task.ID]; duplicate {
				t.Fatalf("duplicate row across keyset pages: %s", task.ID)
			}
			seen[task.ID] = struct{}{}
			if task.MatterID != visibleMatter {
				t.Fatalf("unauthorized Matter reached page: %s", task.MatterID)
			}
		}
		if !page.HasMore {
			if int64(len(seen)) != counts.Total {
				t.Fatal(fmt.Sprintf("authorized pages %d != source count %d", len(seen), counts.Total))
			}
			return
		}
		query.Cursor = page.NextCursor
	}
	t.Fatalf("cursor traversal did not terminate: %d records", len(seen))
}

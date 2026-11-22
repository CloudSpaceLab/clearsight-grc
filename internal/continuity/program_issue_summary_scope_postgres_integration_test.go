//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresVisibleProgramIssueCountMatchesOpenScopedList(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	const tenant = "97777777-7777-7777-8777-777777777771"
	const principal = "97777777-7777-7777-8777-777777777772"
	const program = "97777777-7777-7777-8777-777777777773"
	const visible = "97777777-7777-7777-8777-777777777774"
	const retired = "97777777-7777-7777-8777-777777777775"
	const hidden = "97777777-7777-7777-8777-777777777776"
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-issue-scope','Program issue scope')`, tenant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, table := range []string{"framework_starter_requirement_links", "monitoring_form_templates", "matter_links", "matters", "programs", "principals", "legal_entities"} {
			if _, err := pool.Exec(context.Background(), "DELETE FROM "+table+" WHERE tenant_id=$1::uuid", tenant); err != nil {
				t.Errorf("cleanup %s: %v", table, err)
			}
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenant); err != nil {
			t.Errorf("cleanup tenant: %v", err)
		}
		pool.Close()
	})
	entity := seedPostgresTestLegalEntity(t, ctx, pool, tenant, "ENTITY-A")
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Reader')`, principal, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'P','Program','ASSURANCE','ACTIVE','Risk',clock_timestamp())`, program, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope) VALUES($1::uuid,$4::uuid,$5::uuid,'MAT-A','CONTROL_GAP','ASSESSMENT',3,'Visible','Visible','{"access":"INTERNAL"}'::jsonb),($2::uuid,$4::uuid,$5::uuid,'MAT-B','CONTROL_GAP','ASSESSMENT',3,'Retired link','Retired link','{"access":"INTERNAL"}'::jsonb),($3::uuid,$4::uuid,$5::uuid,'MAT-C','CONTROL_GAP','ASSESSMENT',3,'Restricted','Restricted','{"access":"RESTRICTED","allowed_principal_ids":["another"]}'::jsonb)`, visible, retired, hidden, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id) VALUES($1::uuid,$2::uuid,$5::uuid),($1::uuid,$3::uuid,$5::uuid),($1::uuid,$4::uuid,$5::uuid)`, tenant, visible, retired, hidden, program); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matter_links SET retired_at=clock_timestamp(),retired_by=$3::uuid,retirement_reason='Review complete' WHERE tenant_id=$1::uuid AND matter_id=$2::uuid`, tenant, retired, principal); err != nil {
		t.Fatal(err)
	}
	actor := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: principal, ExpiresAt: time.Now().Add(time.Hour)})
	svc := NewService(NewCurrentPostgresRepository(pool))
	programs, err := svc.ListProgramSummaries(actor, tenant, SummaryQuery{Limit: 20})
	if err != nil || len(programs.Items) != 1 {
		t.Fatalf("programs: %#v %v", programs, err)
	}
	if programs.Items[0].OpenMatterCount != 1 {
		t.Fatalf("visible count=%d want 1", programs.Items[0].OpenMatterCount)
	}
	matters, err := svc.ListMatterSummaries(actor, tenant, SummaryQuery{ProgramID: program, Status: "OPEN", Limit: 20})
	if err != nil || len(matters.Items) != 1 || matters.Items[0].Matter.ID != visible {
		t.Fatalf("visible issues: %#v %v", matters, err)
	}
	if _, err = svc.ListMatterSummaries(actor, tenant, SummaryQuery{ProgramID: program, Status: "OPEN", Cursor: "bad", Limit: 20}); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid cursor error: %v", err)
	}
	// A deferred constraint allows a deliberately malformed link to be read
	// inside this transaction; rollback prevents it ever being committed.
	otherEntity := seedPostgresTestLegalEntity(t, ctx, pool, tenant, "ENTITY-B")
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET CONSTRAINTS matter_links_legal_entity_match DEFERRED`); err != nil {
		t.Fatal(err)
	}
	const foreign = "97777777-7777-7777-8777-777777777777"
	if _, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope) VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-FOREIGN','CONTROL_GAP','ASSESSMENT',3,'Foreign','Foreign','{}'::jsonb)`, foreign, tenant, otherEntity); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id) VALUES($1::uuid,$2::uuid,$3::uuid)`, tenant, foreign, program); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.Query(ctx, matterSummarySQL, tenant, "OPEN", "", true, principal, tenant, false, 0, time.Time{}, "", 21, program, true, tenant, "*", "", 0, "", time.Now(), false, "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, values[0].(string))
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if len(ids) != 1 || ids[0] != visible {
		t.Fatalf("tenant-wide program issue IDs = %#v", ids)
	}

}

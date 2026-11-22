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

func TestPostgresVisibleInterventionEnforcesScopeAndMatterVisibility(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	const tenant = "96666666-6666-7666-8666-666666666661"
	const owner = "96666666-6666-7666-8666-666666666662"
	const allowed = "96666666-6666-7666-8666-666666666663"
	const other = "96666666-6666-7666-8666-666666666664"
	const program = "96666666-6666-7666-8666-666666666665"
	const matter = "96666666-6666-7666-8666-666666666666"
	const link = "96666666-6666-7666-8666-666666666667"
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'visible-intervention-test','Visible intervention')`, tenant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, table := range []string{"framework_starter_requirement_links", "monitoring_form_templates", "matter_links", "matter_actions", "matters", "programs", "principals", "legal_entities"} {
			_, _ = pool.Exec(context.Background(), "DELETE FROM "+table+" WHERE tenant_id=$1::uuid", tenant)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenant)
		pool.Close()
	})
	entity := seedPostgresTestLegalEntity(t, ctx, pool, tenant, "ENTITY-A")
	entityB := seedPostgresTestLegalEntity(t, ctx, pool, tenant, "ENTITY-B")
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$4::uuid,'PERSON','Owner'),($2::uuid,$4::uuid,'PERSON','Allowed'),($3::uuid,$4::uuid,'PERSON','Other')`, owner, allowed, other, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'P','Program','ASSURANCE','ACTIVE','Risk',clock_timestamp())`, program, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,owner_principal_id,trigger_key) VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-SECRET','CONTROL_GAP','DRAFT',3,'Issue','Issue',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($4::text)),$5::uuid,'monitoring-check-adverse:check-1')`, matter, tenant, entity, allowed, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matter_links(id,tenant_id,matter_id,program_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, link, tenant, matter, program); err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewCurrentPostgresRepository(pool))
	actor := func(id, legal string) context.Context {
		return identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: legal, PrincipalID: id, ExpiresAt: time.Now().Add(time.Hour)})
	}
	lookup := func(c context.Context, legal, p string) (VisibleIntervention, error) {
		return svc.VisibleIntervention(c, tenant, legal, "monitoring-check-adverse:check-1", p)
	}
	for _, id := range []string{owner, allowed} {
		v, err := lookup(actor(id, entity), entity, program)
		if err != nil || v.ID != matter || v.Reference != "MAT-SECRET" {
			t.Fatalf("visible %s: %#v %v", id, v, err)
		}
	}
	if _, err = lookup(actor(other, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted actor: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matter_actions(tenant_id,matter_id,title,description,owner_principal_id) VALUES($1::uuid,$2::uuid,'Review','Review',$3::uuid)`, tenant, matter, other); err != nil {
		t.Fatal(err)
	}
	if value, err := lookup(actor(other, entity), entity, program); err != nil || value.ID != matter {
		t.Fatalf("assigned action owner: %#v %v", value, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matter_actions SET owner_principal_id=$3::uuid WHERE tenant_id=$1::uuid AND matter_id=$2::uuid`, tenant, matter, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(other, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked action owner: %v", err)
	}
	const tenantAlias = "96666666-6666-7666-8666-666666666669"
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Alias collision')`, tenantAlias, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(owner, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ambiguous tenant alias: %v", err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantAlias); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(owner, entityB), entityB, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong entity: %v", err)
	}
	if _, err = lookup(actor(owner, entity), entity, "96666666-6666-7666-8666-666666666668"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong program: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2,'Ambiguous alias','NG')`, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(owner, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ambiguous entity alias: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE legal_entities SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid AND code=$2`, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(owner, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inactive owner: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matter_links SET retired_at=clock_timestamp(),retired_by=$2::uuid,retirement_reason='Review complete' WHERE id=$1::uuid`, link, allowed); err != nil {
		t.Fatal(err)
	}
	if _, err = lookup(actor(allowed, entity), entity, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired link: %v", err)
	}
}

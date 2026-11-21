//go:build postgres && postgresintegration

package apiquestion

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdoptionFormAccessUsesScopedPostgresLibrary(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ids := make([]string, 5)
	for i := range ids {
		ids[i], err = id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
	}
	tenant, entity, otherEntity, principal, form := ids[0], ids[1], ids[2], ids[3], ids[4]
	now := time.Now().UTC()
	slug := "form-access-" + tenant
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid; DELETE FROM principals WHERE tenant_id=$1::uuid; DELETE FROM legal_entities WHERE tenant_id=$1::uuid; DELETE FROM tenants WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
	}()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$6,'Form access test'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'ONE','One','NG',$7),($3::uuid,$1::uuid,'TWO','Two','NG',$7); INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($4::uuid,$1::uuid,'PERSON','Test owner','ACTIVE',$7); INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at) VALUES($5::uuid,$1::uuid,$2::uuid,'FORM','Sample','Sample','{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,'[{"id":"general","title":"Questions"}]'::jsonb,'[{"id":"q","section_id":"general","label":"Access?","type":"yes_no"}]'::jsonb,'ACTIVE',true,$7,1,$4::uuid,$7,$7)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, otherEntity, principal, form, slug, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.Actor{TenantID: slug, LegalEntityID: entity, PrincipalID: principal, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	access := NewAdoptionFormAccess(monitoring.NewService(monitoring.NewPostgresRepository(pool), nil))
	if err = access.AuthorizeAdoptionForm(identity.WithActor(ctx, actor), slug, entity, "program", form, 1); err != nil {
		t.Fatalf("scoped active form: %v", err)
	}
	actor.LegalEntityID = otherEntity
	if err = access.AuthorizeAdoptionForm(identity.WithActor(ctx, actor), slug, otherEntity, "program", form, 1); err == nil {
		t.Fatal("other entity read was permitted")
	}
}

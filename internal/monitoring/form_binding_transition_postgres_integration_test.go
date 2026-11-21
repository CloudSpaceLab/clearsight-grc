//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type failingBindingTransitionHook struct{}

func (failingBindingTransitionHook) ApplyFormBindingTransition(ctx context.Context, tx pgx.Tx, _, _ FormTemplate, _ LifecycleTransition) error {
	_, err := tx.Exec(ctx, `UPDATE monitoring_form_templates SET purpose='changed' WHERE id='00000000-0000-0000-0000-000000000000'::uuid`)
	if err != nil {
		return err
	}
	return errors.New("binding transition unavailable")
}

func TestPostgresFormBindingTransitionFailsClosedAndRollsBack(t *testing.T) {
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
	ids := make([]string, 4)
	for i := range ids {
		ids[i], err = id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
	}
	tenant, entity, actor, form := ids[0], ids[1], ids[2], ids[3]
	now := time.Now().UTC()
	slug := "form-binding-" + tenant
	defer func() {
		for _, query := range []string{`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, `DELETE FROM monitoring_events WHERE tenant_id=$1::uuid`, `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			_, _ = pool.Exec(context.Background(), query, tenant)
		}
	}()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$5,'Binding transition'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Bank','NG',$6); INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Maker','ACTIVE',$6); INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,presentation,sections,fields,status,is_current,version,created_by,created_at,updated_at) VALUES($4::uuid,$1::uuid,$2::uuid,'FORM','Form','Check access','{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,'[{"id":"general","title":"Questions"}]'::jsonb,'[{"id":"q","section_id":"general","label":"Access?","type":"yes_no"}]'::jsonb,'DRAFT',false,1,$3::uuid,$6,$6)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actor, form, slug, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	transition := LifecycleTransition{TenantID: slug, LegalEntityID: entity, ID: form, ExpectedVersion: 1, To: LifecyclePendingApproval, ActorID: actor, At: now, BindingImpact: &FormBindingTransition{ApprovalIDs: []string{"selected"}}}
	if _, err = repo.TransitionForm(ctx, transition); !errors.Is(err, ErrBindingTransitionUnavailable) {
		t.Fatalf("missing binding transition hook: %v", err)
	}
	repo.ConfigureFormBindingTransition(failingBindingTransitionHook{})
	if _, err = repo.TransitionForm(ctx, transition); err == nil {
		t.Fatal("failing hook succeeded")
	}
	var revisions, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid),(SELECT count(*) FROM monitoring_events WHERE tenant_id=$1::uuid),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid)`, tenant).Scan(&revisions, &events, &outbox); err != nil || revisions != 1 || events != 0 || outbox != 0 {
		t.Fatalf("partial transition revisions=%d events=%d outbox=%d err=%v", revisions, events, outbox, err)
	}
}

//go:build postgres && postgresintegration

package bankverticals

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDefaultComplianceFormPreservesScopedHistory(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	rowID := func(sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	tenant := rowID(`INSERT INTO tenants(slug,name) VALUES('compliance-form-'||uuidv7()::text,'Compliance form test') RETURNING id::text`)
	defer func() {
		for _, table := range []string{"outbox_events", "monitoring_events", "monitoring_form_templates", "programs", "principals", "legal_entities"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id=$1::uuid", tenant); err != nil {
				t.Error(err)
			}
		}
		if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenant); err != nil {
			t.Error(err)
		}
	}()
	entity := rowID(`INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,'COMPLIANCE','Compliance test','NG') RETURNING id::text`, tenant)
	person := func(name string) string {
		return rowID(`INSERT INTO principals(tenant_id,kind,display_name) VALUES($1::uuid,'PERSON',$2) RETURNING id::text`, tenant, name)
	}
	maker, checker := person("Form maker"), person("Independent checker")
	program := rowID(`INSERT INTO programs(tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from) VALUES($1::uuid,$2::uuid,'COMPLIANCE-TEST','Compliance test','COMPLIANCE','DRAFT','Compliance','NG','{}',clock_timestamp()) RETURNING id::text`, tenant, entity)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.TenantID = tenant
	config.LegalEntityID = entity
	config.ActorID = maker
	config.ReviewerPrincipalID = checker
	forms := monitoring.NewService(monitoring.NewPostgresRepository(pool), nil)
	service := NewService(nil, nil)
	service.ConfigureMonitoring(forms)
	if err := service.ensureVendorAcceptanceForms(ctx, config, program); err != nil {
		t.Fatal(err)
	}
	actor := monitoring.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: maker}
	form, err := forms.LatestFormByCode(ctx, actor, program, vendorComplianceFormCode)
	if err != nil {
		t.Fatal(err)
	}
	if form.Status != monitoring.LifecycleActive || form.SubmittedBy != maker || form.ApprovedBy != checker || form.Version != 3 {
		t.Fatalf("governed activation lost: %+v", form.Lifecycle)
	}
	paused, err := forms.TransitionForm(ctx, actor, monitoring.TransitionInput{ID: form.ID, ProgramID: program, LegalEntityID: entity, ExpectedVersion: form.Version, To: monitoring.LifecyclePaused})
	if err != nil {
		t.Fatal(err)
	}
	// Fill the old prefix-limited page ahead of the reference code.
	if _, err := pool.Exec(ctx, `INSERT INTO monitoring_form_templates(tenant_id,legal_entity_id,program_id,code,name,purpose,fields,status,version) SELECT $1::uuid,$2::uuid,$3::uuid,'A-FILLER-'||n,'Other form','Collect a separate service answer.','[{"id":"answer","label":"Answer","type":"short_text","required":false}]'::jsonb,'DRAFT',1 FROM generate_series(1,110) n`, tenant, entity, program); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := service.ensureGovernedVendorForm(ctx, config, ReferenceThirdPartyRiskComplianceForm(program, entity), "third-party risk compliance"); err != nil {
			t.Fatal(err)
		}
	}
	after, err := forms.LatestFormByCode(ctx, actor, program, vendorComplianceFormCode)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paused, after) {
		t.Fatal("repeat install changed stored reference history")
	}
	var identities int
	if err := pool.QueryRow(ctx, `SELECT count(DISTINCT id) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND program_id=$3::uuid AND code=$4`, tenant, entity, program, vendorComplianceFormCode).Scan(&identities); err != nil || identities != 1 {
		t.Fatalf("reference form duplicated: %d %v", identities, err)
	}
	wrong := actor
	wrong.TenantID = "unknown-tenant"
	if _, err := forms.LatestFormByCode(ctx, wrong, program, vendorComplianceFormCode); !errors.Is(err, monitoring.ErrNotFound) {
		t.Fatalf("tenant scope escaped: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO monitoring_form_templates(tenant_id,legal_entity_id,program_id,code,name,purpose,fields,status,version) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'Conflicting draft','Test ambiguous form identity.','[{"id":"answer","label":"Answer","type":"short_text","required":false}]','DRAFT',1)`, tenant, entity, program, vendorComplianceFormCode); err != nil {
		t.Fatal(err)
	}
	if err := service.ensureGovernedVendorForm(ctx, config, ReferenceThirdPartyRiskComplianceForm(program, entity), "third-party risk compliance"); !errors.Is(err, monitoring.ErrInvalid) {
		t.Fatalf("duplicate code did not fail closed: %v", err)
	}
}

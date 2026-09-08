//go:build postgres && postgresintegration

package main

import (
	"context"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestPostgresAssessmentAuthorityUsesCommandTransactionAndDelegatedRoleCode(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	newID := func() string {
		v, e := id.NewUUIDv7()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	tenant, entity, reviewer, delegate, policy, version, role, position, binding, response := newID(), newID(), newID(), newID(), newID(), newID(), newID(), newID(), newID(), newID()
	approver := newID()
	now := time.Now().UTC().Add(-time.Minute)
	definition := fmt.Sprintf(`{"rules":[{"id":"review","legal_entity_id":"%s","object_type":"FORM_RESPONSE","object_id":"%s","responsibility":"REVIEWER","decision_type":"forms.response.assess","min_materiality":0,"priority":100,"selector":{"kind":"ROLE","ref":"CISO"}}]}`, entity, response)
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$1,'Assessment authority test');
 INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Test bank','NG',$10);
 INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Security reviewer','ACTIVE',$10),($4::uuid,$1::uuid,'PERSON','Delegated reviewer','ACTIVE',$10),($12::uuid,$1::uuid,'PERSON','Authority approver','ACTIVE',$10);
 INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,valid_from) VALUES($7::uuid,$1::uuid,'CISO','Chief information security officer','',ARRAY['REVIEWER'],$10);
 INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($8::uuid,$1::uuid,$2::uuid,'SECURITY','Security lead',$3::uuid,$10);
 INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,priority,valid_from) VALUES($9::uuid,$1::uuid,$8::uuid,$7::uuid,100,$10);
 INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version) VALUES($5::uuid,$1::uuid,$2::uuid,'REVIEW','Response review','DRAFT',1,$10,1);
 INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at) VALUES($6::uuid,$5::uuid,$2::uuid,1,$11::jsonb,'test',$10,$10);
 UPDATE routing_policies SET status='ACTIVE' WHERE id=$5::uuid;
 INSERT INTO delegations(tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by,approved_by,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$2::text,'decision_type','forms.response.assess'),$10,$10::timestamptz+interval '1 hour','ACTIVE',$3::uuid,$12::uuid,$10,1)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, reviewer, delegate, policy, version, role, position, binding, now, definition, approver)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenant) }()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	ctx = authority.WithPostgresTransaction(ctx, tx)
	router := authority.NewPostgresService(pool)
	actor := identity.Actor{TenantID: tenant, LegalEntityID: "*", PrincipalID: delegate}
	summary := evidence.CompletedResponseSummary{ID: response, TenantID: tenant, LegalEntityID: entity}
	field := formcontract.Field{Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, ReviewerRole: "CISO"}}
	authorize := formAssessmentAuthorizer(router)
	if _, err = authorize(ctx, actor, summary, field); err != nil {
		t.Fatalf("current delegated role in command transaction: %v", err)
	}
	outcomes, err := router.(authority.BatchResolver).ResolveMany(ctx, []authority.ResolveInput{{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_RESPONSE", ObjectID: response, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3}})
	if err != nil || len(outcomes) != 1 || outcomes[0].Err != nil || !outcomes[0].Resolution.AllowsPrincipalWithRole(delegate, "CISO") {
		t.Fatalf("batch delegated role: %+v %v", outcomes, err)
	}
	if err = formResponseDiscoveryAuthorizer(router)(ctx, actor, summary); err != nil {
		t.Fatalf("review discovery: %v", err)
	}
	field.Assessment.ReviewerRole = "CRO"
	if _, err = authorize(ctx, actor, summary, field); err == nil {
		t.Fatal("different field role authorized")
	}
	field.Assessment.ReviewerRole = "CISO"
	if _, err = tx.Exec(ctx, `UPDATE delegations SET status='EXPIRED' WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = authorize(ctx, actor, summary, field); err == nil {
		t.Fatal("uncommitted delegation revocation was ignored")
	}
	if err = formResponseDiscoveryAuthorizer(router)(ctx, actor, summary); err == nil {
		t.Fatal("revoked delegate retained discovery")
	}
}

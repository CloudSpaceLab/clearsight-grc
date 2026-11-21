//go:build postgres && postgresintegration

package authority

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This is a narrow parity check for whole-response reviewer routes. It does
// not establish parity for all route types or authorize an approval inbox.
func TestPostgresResponseReviewerReadRouteParity(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var tenant, entity, response, reviewer, delegate, other string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &response, &reviewer, &delegate, &other); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Review parity')`, tenant, "review-parity-"+tenant)
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM segregation_rules WHERE tenant_id=$1::uuid`, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`,
			`DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, `DELETE FROM delegations WHERE tenant_id=$1::uuid`,
			`DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`,
			`DELETE FROM role_templates WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`,
			`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`,
		} {
			_, _ = pool.Exec(context.Background(), query, tenant)
		}
	})
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'PARITY','Parity','NG',$3)`, entity, tenant, now.Add(-time.Hour))
	for _, principal := range []string{reviewer, delegate, other} {
		exec(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Reviewer','ACTIVE',$3)`, principal, tenant, now.Add(-time.Hour))
	}
	service := NewPostgresService(pool)
	input := ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_RESPONSE", ObjectID: response, Responsibility: ResponsibilityReviewer, DecisionType: "forms.response.review", Materiality: 3, At: now}
	predicate := PostgresReadRouteSQL("source", "id", "FORM_RESPONSE", "FORMS.RESPONSE.REVIEW", "REVIEWER", 4, 5)
	check := func(label, principal string, expected bool) {
		t.Helper()
		var permitted bool
		err := pool.QueryRow(ctx, `SELECT `+predicate+` FROM (SELECT $1::uuid AS tenant_id,$2::uuid AS legal_entity_id,$3::uuid AS id) source`, tenant, entity, response, principal, now).Scan(&permitted)
		if err != nil {
			t.Fatalf("%s predicate: %v", label, err)
		}
		resolution, resolveErr := service.Resolve(ctx, input)
		allowed := resolveErr == nil && resolution.AllowsPrincipal(principal)
		if resolveErr != nil && !errors.Is(resolveErr, ErrNoRoute) && !errors.Is(resolveErr, ErrAmbiguousRoute) {
			t.Fatalf("%s resolver: %v", label, resolveErr)
		}
		batch, batchErr := service.(BatchResolver).ResolveMany(ctx, []ResolveInput{input})
		if batchErr != nil || len(batch) != 1 {
			t.Fatalf("%s batch: %+v %v", label, batch, batchErr)
		}
		batchAllowed := batch[0].Err == nil && batch[0].Resolution.AllowsPrincipal(principal)
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		current, evaluated, currentErr := service.(*postgresService).ResolveCurrent(WithPostgresTransaction(ctx, tx), input)
		_ = tx.Rollback(ctx)
		currentAllowed := currentErr == nil && current.AllowsPrincipal(principal)
		if permitted != expected || allowed != expected || batchAllowed != expected || currentAllowed != expected {
			t.Fatalf("%s expected=%v predicate=%v resolver=%v (%v) batch=%v (%v) current=%v (%v)", label, expected, permitted, allowed, resolveErr, batchAllowed, batch[0].Err, currentAllowed, currentErr)
		}
		if currentErr == nil && (evaluated.Before(now) || current.RuleID != resolution.RuleID || current.PolicyVersion != resolution.PolicyVersion || current.SourceType != resolution.SourceType || current.SourceID != resolution.SourceID || current.SourceID == "") {
			t.Fatalf("%s current authority provenance differs: current=%+v at=%v single=%+v", label, current, evaluated, resolution)
		}
	}
	check("missing route", reviewer, false)
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','FORM_RESPONSE',$4::uuid,'forms.response.review',10,$5,'review:v1')`, tenant, entity, reviewer, response, now.Add(-time.Hour))
	check("named reviewer", reviewer, true)
	check("unassigned reviewer", other, false)
	exec(`INSERT INTO delegations(tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$2::text),$5,$6,'ACTIVE',$3::uuid)`, tenant, entity, reviewer, delegate, now.Add(-time.Hour), now.Add(time.Hour))
	check("delegate", delegate, true)
	exec(`UPDATE delegations SET scope=scope||'{"max_materiality":2}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("delegation maximum excludes materiality three", delegate, false)
	exec(`UPDATE delegations SET scope=(scope-'max_materiality')||'{"min_materiality":4}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("delegation minimum excludes materiality three", delegate, false)
	exec(`UPDATE delegations SET scope=(scope-'min_materiality')||'{"min_materiality":2,"max_materiality":4}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("delegation range includes materiality three", delegate, true)
	exec(`UPDATE delegations SET scope=scope-'min_materiality'-'max_materiality' WHERE tenant_id=$1::uuid`, tenant)
	exec(`UPDATE delegations SET scope=scope||'{"decision_type":"forms.response.assess"}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("other decision delegation", delegate, false)
	exec(`UPDATE delegations SET scope=(scope-'decision_type')||'{"decision_type":"forms.response.review"}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("matching decision delegation", delegate, true)
	exec(`UPDATE delegations SET scope=(scope-'decision_type')||'{"object_type":"PROGRAM"}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("other object type delegation", delegate, false)
	exec(`UPDATE delegations SET scope=scope-'object_type' WHERE tenant_id=$1::uuid`, tenant)
	exec(`INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'forms.response.review',$4,'grant:v1')`, tenant, entity, delegate, now.Add(-time.Hour))
	check("grant excludes origin", reviewer, false)
	check("grant permits delegate", delegate, true)
	exec(`UPDATE authority_grants SET limits='{"max_materiality":2}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("grant maximum excludes materiality three", delegate, false)
	exec(`UPDATE authority_grants SET limits='{"min_materiality":4}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("grant minimum excludes materiality three", delegate, false)
	exec(`UPDATE authority_grants SET limits='{"min_materiality":2,"max_materiality":4}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("grant range permits materiality three", delegate, true)
	for _, valid := range []string{`{}`, `{"min_materiality":0,"max_materiality":5}`, `{"min_materiality":3,"max_materiality":3}`} {
		exec(`UPDATE authority_grants SET limits=$2::jsonb WHERE tenant_id=$1::uuid`, tenant, valid)
		check("valid grant "+valid, delegate, true)
	}
	for _, invalid := range []string{`{"max_materiality":null}`, `{"max_materiality":"4"}`, `{"max_materiality":""}`, `{"max_materiality":6}`, `{"min_materiality":-1}`, `{"min_materiality":2.5}`, `{"maximum_amount":10}`, `[]`, `null`, `42`} {
		exec(`UPDATE authority_grants SET limits=$2::jsonb WHERE tenant_id=$1::uuid`, tenant, invalid)
		check("malformed grant "+invalid, delegate, false)
	}
	exec(`DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant)
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','FORM_RESPONSE',$4::uuid,'forms.response.review',10,$5,'review:v1')`, tenant, entity, other, response, now.Add(-time.Hour))
	check("ambiguous route", reviewer, false)
	exec(`DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenant, other)
	exec(`UPDATE delegations SET ends_at=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Minute))
	check("expired delegation", delegate, false)
	exec(`DELETE FROM delegations WHERE tenant_id=$1::uuid`, tenant)
	exec(`DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant)
	var role, position, binding, otherEntity string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&role, &position, &binding, &otherEntity); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'OTHER','Other','NG',$3)`, otherEntity, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($1::uuid,$2::uuid,'RISK_REVIEWER','Risk reviewer',$3)`, role, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','Reviewer',$4::uuid,$5)`, position, tenant, entity, reviewer, now.Add(-time.Hour))
	exec(`INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,scope,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,jsonb_build_object('legal_entity_id',$5::text),$6)`, binding, tenant, position, role, entity, now.Add(-time.Hour))
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,role_template_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','FORM_RESPONSE',$4::uuid,'forms.response.review',10,$5,'role:v1')`, tenant, entity, role, response, now.Add(-time.Hour))
	check("entity role reviewer", reviewer, true)
	exec(`UPDATE position_role_bindings SET scope='{"legal_entity_id":"*"}'::jsonb WHERE id=$1::uuid`, binding)
	check("wildcard binding in entity role", reviewer, true)
	exec(`UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$2::text) WHERE id=$1::uuid`, binding, entity)
	var otherRole, otherPosition, otherBinding string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&otherRole, &otherPosition, &otherBinding); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($1::uuid,$2::uuid,'OTHER_ENTITY_ROLE','Other entity role',$3)`, otherRole, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'OTHER_POSITION','Other position',$4::uuid,$5)`, otherPosition, tenant, otherEntity, reviewer, now.Add(-time.Hour))
	exec(`INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,scope,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,jsonb_build_object('legal_entity_id',$5::text),$6)`, otherBinding, tenant, otherPosition, otherRole, otherEntity, now.Add(-time.Hour))
	exec(`INSERT INTO authority_grants(tenant_id,legal_entity_id,role_template_id,decision_type,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'forms.response.review',$4,'grant:v1')`, tenant, entity, otherRole, now.Add(-time.Hour))
	check("other entity role cannot satisfy grant", reviewer, false)
	exec(`UPDATE position_role_bindings SET scope='{"legal_entity_id":"*"}'::jsonb WHERE id=$1::uuid`, otherBinding)
	check("wildcard other entity role cannot satisfy grant", reviewer, false)
	exec(`DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant)
	exec(`UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$2::text) WHERE id=$1::uuid`, binding, otherEntity)
	check("other entity role binding", reviewer, false)
	exec(`UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$2::text) WHERE id=$1::uuid`, binding, entity)
	exec(`INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES($1::uuid,'NO_REVIEW','REVIEWER','RISK_REVIEWER','ACTIVE',$2)`, tenant, now.Add(-time.Hour))
	check("segregated reviewer", reviewer, false)
}

func TestPostgresDelegationMaterialityRejectsMalformedLimits(t *testing.T) {
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
	for _, tc := range []struct {
		scope   string
		allowed bool
	}{
		{`{}`, true}, {`{"min_materiality":2,"max_materiality":4}`, true},
		{`{"max_materiality":2}`, false}, {`{"min_materiality":4}`, false},
		{`{"min_materiality":"2"}`, false}, {`{"max_materiality":null}`, false},
		{`{"max_materiality":6}`, false}, {`{"min_materiality":2.5}`, false},
		{`[]`, false},
	} {
		var allowed bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM (SELECT $1::jsonb AS scope) d WHERE true`+delegationMaterialitySQL("$2")+`)`, tc.scope, 3).Scan(&allowed)
		if err != nil || allowed != tc.allowed {
			t.Errorf("scope=%s allowed=%v want=%v err=%v", tc.scope, allowed, tc.allowed, err)
		}
	}
}

// The same scope predicate must serve the individual, batch and read paths;
// both matching and excluding conditions affect whether a delegated decision
// is available.
func TestPostgresDelegationTargetScope(t *testing.T) {
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
	for _, tc := range []struct {
		scope   string
		allowed bool
	}{
		{`{}`, true},
		{`{"object_type":"form_response","object_id":"response-1","decision_type":"FORMS.RESPONSE.REVIEW","min_materiality":2,"max_materiality":4}`, true},
		{`{"object_type":"PROGRAM"}`, false},
		{`{"object_id":"response-2"}`, false},
		{`{"decision_type":"forms.response.assess"}`, false},
		{`{"max_materiality":2}`, false},
		{`{"object_type":null}`, false},
	} {
		var allowed bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM (SELECT $1::jsonb AS scope) d WHERE true`+delegationTargetScopeSQL("$2::text", "$3::text", "$4::text", "$5::int")+`)`, tc.scope, "FORM_RESPONSE", "response-1", "forms.response.review", 3).Scan(&allowed)
		if err != nil || allowed != tc.allowed {
			t.Errorf("scope=%s allowed=%v want=%v err=%v", tc.scope, allowed, tc.allowed, err)
		}
	}
}

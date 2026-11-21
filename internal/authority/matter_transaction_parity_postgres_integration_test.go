//go:build postgres && postgresintegration

package authority

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This verifies a prerequisite only; it does not authorize Matter decisions.
func TestMatterBatchAuthoritySeesRevocationAfterSubjectLock(t *testing.T) {
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
	var tenant, entity, principal, subject string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &principal, &subject); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Matter authority')`, tenant, "matter-auth-"+tenant)
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			_, _ = pool.Exec(context.Background(), sql, tenant)
		}
	})
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK','Bank','NG',$3)`, entity, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Reviewer','ACTIVE',$3)`, principal, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','DECISION',$4::uuid,'matter.decision.record',10,$5,'review:v1')`, tenant, entity, principal, subject, now.Add(-time.Hour))
	resolver := NewPostgresService(pool).(BatchResolver)
	input := ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "DECISION", ObjectID: subject, Responsibility: ResponsibilityReviewer, DecisionType: "matter.decision.record", Materiality: 4, At: now}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM legal_entities WHERE id=$1::uuid FOR UPDATE`, entity).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	bound := WithPostgresTransaction(ctx, tx)
	initial, err := resolver.ResolveMany(bound, []ResolveInput{input})
	if err != nil || len(initial) != 1 || initial[0].Err != nil || !initial[0].Resolution.AllowsPrincipal(principal) {
		t.Fatalf("initial route: %+v %v", initial, err)
	}
	input.Materiality = 5
	high, err := resolver.ResolveMany(bound, []ResolveInput{input})
	if err != nil || len(high) != 1 || high[0].Err != nil || !high[0].Resolution.AllowsPrincipal(principal) {
		t.Fatalf("high materiality route: %+v %v", high, err)
	}
	current := NewPostgresService(pool).(*postgresService)
	repeatable, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := current.ResolveCurrent(WithPostgresTransaction(ctx, repeatable), input); err == nil {
		_ = repeatable.Rollback(ctx)
		t.Fatal("current authority accepted an earlier transaction snapshot")
	}
	_ = repeatable.Rollback(ctx)
	exec(`UPDATE responsibility_assignments SET valid_until=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Second))
	after, err := resolver.ResolveMany(bound, []ResolveInput{input})
	if err != nil || len(after) != 1 || !errors.Is(after[0].Err, ErrNoRoute) {
		t.Fatalf("revoked route must deny: %+v %v", after, err)
	}
	// A route restored inside this transaction is invisible to the pool until
	// commit. Seeing it proves the resolver used the bound transaction.
	if _, err := tx.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	inside, err := resolver.ResolveMany(bound, []ResolveInput{input})
	if err != nil || len(inside) != 1 || inside[0].Err != nil || !inside[0].Resolution.AllowsPrincipal(principal) {
		t.Fatalf("transaction-local route: %+v %v", inside, err)
	}
	if _, _, err := current.ResolveCurrent(ctx, input); err == nil {
		t.Fatal("current authority must require the caller's transaction")
	}
	input.At = now.Add(-time.Hour)
	if _, err := tx.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := current.ResolveCurrent(bound, input); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("current authority accepted an expired route at caller time: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	resolved, evaluated, err := current.ResolveCurrent(bound, input)
	if err != nil || !resolved.AllowsPrincipal(principal) || resolved.RuleID == "" || resolved.PolicyVersion == "" || resolved.SourceType != "ASSIGNMENT" || resolved.SourceID == "" || evaluated.Before(now) {
		t.Fatalf("current route and database instant: %+v at=%v error=%v", resolved, evaluated, err)
	}
	if _, err = tx.Exec(ctx, `CREATE TEMP TABLE review_decision_probe(source_id text NOT NULL,rule_id text NOT NULL,policy_version text NOT NULL,decided_at timestamptz NOT NULL) ON COMMIT DROP`); err != nil {
		t.Fatal(err)
	}
	encoded, err := PostgresCurrentRouteRequest(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PostgresCurrentRouteRequest(ResolveInput{}); err == nil {
		t.Fatal("invalid authority request accepted")
	}
	wildcard := input
	wildcard.ObjectID = "*"
	if _, err := PostgresCurrentRouteRequest(wildcard); err == nil {
		t.Fatal("wildcard authority request accepted")
	}
	statement := PostgresCurrentRouteCTEs() + ` INSERT INTO review_decision_probe(source_id,rule_id,policy_version,decided_at)
	SELECT DISTINCT s.source_id,s.rule_id,s.policy_version,statement_timestamp() FROM selected s JOIN effective e USING(request_index)
	WHERE NOT s.ambiguous AND e.principal_id=$2::uuid RETURNING source_id,rule_id,policy_version,decided_at`
	var sourceID, ruleID, policy string
	var decidedAt time.Time
	err = tx.QueryRow(ctx, statement, string(encoded), principal).Scan(&sourceID, &ruleID, &policy, &decidedAt)
	if err != nil || sourceID != "assignment:"+resolved.SourceID || ruleID != resolved.RuleID || policy != resolved.PolicyVersion || decidedAt.Before(evaluated) {
		t.Fatalf("atomic authority insert: source=%q rule=%q policy=%q at=%v err=%v", sourceID, ruleID, policy, decidedAt, err)
	}
	if _, err = tx.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, statement, string(encoded), principal).Scan(&sourceID, &ruleID, &policy, &decidedAt); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("revoked route inserted a decision: %v", err)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM review_decision_probe`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("decision count=%d err=%v", count, err)
	}
}

// These checks establish limited resolver parity, not command authorization.
func TestMatterDecisionBatchResolverParity(t *testing.T) {
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
	var tenant, entity, owner, delegate, other, matter, decision string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &owner, &delegate, &other, &matter, &decision); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Matter parity')`, tenant, "matter-parity-"+tenant)
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM effective_authority_routes WHERE tenant_id=$1::uuid`, `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, `DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, `DELETE FROM delegations WHERE tenant_id=$1::uuid`, `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			_, _ = pool.Exec(context.Background(), sql, tenant)
		}
	})
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK','Bank','NG',$3)`, entity, tenant, now.Add(-time.Hour))
	for _, principal := range []string{owner, delegate, other} {
		exec(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Reviewer','ACTIVE',$3)`, principal, tenant, now.Add(-time.Hour))
	}
	service := NewPostgresService(pool)
	input := ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "DECISION", ObjectID: decision, Responsibility: ResponsibilityReviewer, DecisionType: "matter.decision.record", Materiality: 4, At: now}
	check := func(label, principal string, want bool) {
		t.Helper()
		for _, level := range []int{4, 5} {
			input.Materiality = level
			single, singleErr := service.Resolve(ctx, input)
			batch, batchErr := service.(BatchResolver).ResolveMany(ctx, []ResolveInput{input})
			if batchErr != nil || len(batch) != 1 {
				t.Fatalf("%s materiality=%d: batch=%+v error=%v", label, level, batch, batchErr)
			}
			singleAllowed := singleErr == nil && single.AllowsPrincipal(principal)
			batchAllowed := batch[0].Err == nil && batch[0].Resolution.AllowsPrincipal(principal)
			if singleAllowed != want || batchAllowed != want {
				t.Fatalf("%s materiality=%d want=%t single=%t (%v) batch=%t (%v)", label, level, want, singleAllowed, singleErr, batchAllowed, batch[0].Err)
			}
			for _, routeErr := range []error{ErrNoRoute, ErrAmbiguousRoute} {
				if errors.Is(singleErr, routeErr) != errors.Is(batch[0].Err, routeErr) {
					t.Fatalf("%s materiality=%d route failures differ: single=%v batch=%v", label, level, singleErr, batch[0].Err)
				}
			}
			if singleErr == nil && batch[0].Err == nil && (single.RuleID != batch[0].Resolution.RuleID || single.PolicyVersion != batch[0].Resolution.PolicyVersion) {
				t.Fatalf("%s materiality=%d route provenance differs: single=%+v batch=%+v", label, level, single, batch[0].Resolution)
			}
			if singleErr == nil && batch[0].Err == nil && (single.SourceType == "" || single.SourceID == "" || single.SourceType != batch[0].Resolution.SourceType || single.SourceID != batch[0].Resolution.SourceID) {
				t.Fatalf("%s materiality=%d source identity differs: single=%+v batch=%+v", label, level, single, batch[0].Resolution)
			}
		}
	}
	check("missing route", owner, false)
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','DECISION',$4::uuid,'matter.decision.record',10,$5,'review:v1')`, tenant, entity, owner, decision, now.Add(-time.Hour))
	check("named reviewer", owner, true)
	var assignmentID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant).Scan(&assignmentID); err != nil {
		t.Fatal(err)
	}
	assignment, err := service.Resolve(ctx, input)
	if err != nil || assignment.SourceType != "ASSIGNMENT" || assignment.SourceID != assignmentID {
		t.Fatalf("wrong assignment receipt: %+v %v", assignment, err)
	}
	check("unassigned reviewer", other, false)
	if _, err = pool.Exec(ctx, `INSERT INTO delegations(tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$2::text,'object_type','DECISION','object_id',$5::text),$6,$7,'ACTIVE',$3::uuid)`, tenant, entity, owner, delegate, decision, now.Add(-time.Hour), now.Add(time.Hour)); err == nil {
		t.Fatal("exact DECISION delegation unexpectedly accepted by current schema")
	}
	exec(`INSERT INTO delegations(tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$2::text,'decision_type','matter.decision.record'),$5,$6,'ACTIVE',$3::uuid)`, tenant, entity, owner, delegate, now.Add(-time.Hour), now.Add(time.Hour))
	check("entity and decision-type delegated reviewer", delegate, true)
	exec(`UPDATE delegations SET scope=scope||jsonb_build_object('object_type','MATTER','object_id',$2::text) WHERE tenant_id=$1::uuid`, tenant, matter)
	check("different object delegation", delegate, false)
	exec(`UPDATE delegations SET scope=(scope-'object_id'-'object_type')||'{"max_materiality":3}'::jsonb WHERE tenant_id=$1::uuid`, tenant)
	check("materiality exceeds delegation", delegate, false)
	exec(`UPDATE delegations SET scope=scope-'max_materiality' WHERE tenant_id=$1::uuid`, tenant)
	exec(`INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,limits,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.decision.record','{"max_materiality":3}'::jsonb,$4,'grant:v1')`, tenant, entity, delegate, now.Add(-time.Hour))
	check("materiality exceeds grant", delegate, false)
	exec(`DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant)
	var otherEntity, role, position, binding string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&otherEntity, &role, &position, &binding); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'OTHER','Other','NG',$3)`, otherEntity, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($1::uuid,$2::uuid,'OTHER_ROLE','Other role',$3)`, role, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'OTHER','Other',$4::uuid,$5)`, position, tenant, otherEntity, owner, now.Add(-time.Hour))
	exec(`INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,scope,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,jsonb_build_object('legal_entity_id',$5::text),$6)`, binding, tenant, position, role, otherEntity, now.Add(-time.Hour))
	exec(`INSERT INTO authority_grants(tenant_id,legal_entity_id,role_template_id,decision_type,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.decision.record',$4,'grant:v1')`, tenant, entity, role, now.Add(-time.Hour))
	check("other entity role cannot satisfy entity grant", owner, false)
	exec(`UPDATE org_positions SET legal_entity_id=$2::uuid WHERE id=$1::uuid`, position, entity)
	exec(`UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$2::text) WHERE id=$1::uuid`, binding, entity)
	check("matching entity role satisfies grant", owner, true)
	exec(`UPDATE position_role_bindings SET scope='{"legal_entity_id":"*"}'::jsonb WHERE id=$1::uuid`, binding)
	check("wildcard binding for matching entity satisfies grant", owner, true)
	exec(`UPDATE org_positions SET legal_entity_id=$2::uuid WHERE id=$1::uuid`, position, otherEntity)
	check("wildcard binding cannot cross position entity", owner, false)
	exec(`UPDATE org_positions SET legal_entity_id=$2::uuid WHERE id=$1::uuid`, position, entity)
	exec(`UPDATE position_role_bindings SET scope='{"legal_entity_id":null}'::jsonb WHERE id=$1::uuid`, binding)
	check("malformed binding cannot satisfy grant", owner, false)
	exec(`DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant)
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','DECISION',$4::uuid,'matter.decision.record',10,$5,'review:v1')`, tenant, entity, other, decision, now.Add(-time.Hour))
	check("conflicting top routes", owner, false)
	exec(`DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant)
	exec(`DELETE FROM delegations WHERE tenant_id=$1::uuid`, tenant)
	var firstPolicy, secondPolicy string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&firstPolicy, &secondPolicy); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status) VALUES($1::uuid,$3::uuid,$4::uuid,'FIRST','First','DRAFT'),($2::uuid,$3::uuid,$4::uuid,'SECOND','Second','DRAFT')`, firstPolicy, secondPolicy, tenant, entity)
	exec(`INSERT INTO effective_authority_routes(tenant_id,source_policy_id,source_rule_id,policy_version,legal_entity_ref,object_type,object_id,responsibility,decision_type,priority,selector_kind,selector_ref,valid_from) VALUES($1::uuid,$2::uuid,'shared','collision:v1',$4,'DECISION',$5,'REVIEWER','matter.decision.record',100,'PRINCIPAL',$6,$8),($1::uuid,$3::uuid,'shared','collision:v1',$4,'DECISION',$5,'REVIEWER','matter.decision.record',10,'PRINCIPAL',$7,$8)`, tenant, firstPolicy, secondPolicy, entity, decision, owner, other, now.Add(-time.Hour))
	check("lower ranked policy cannot add reviewer", other, false)
	check("highest ranked policy keeps reviewer", owner, true)
	selected, err := service.Resolve(ctx, input)
	if err != nil || selected.SourceType != "POLICY" || selected.SourceID != firstPolicy {
		t.Fatalf("wrong policy receipt: %+v %v", selected, err)
	}
	exec(`UPDATE effective_authority_routes SET priority=100 WHERE source_policy_id=$1::uuid`, secondPolicy)
	_, err = service.Resolve(ctx, input)
	batch, batchErr := service.(BatchResolver).ResolveMany(ctx, []ResolveInput{input})
	if !errors.Is(err, ErrAmbiguousRoute) || batchErr != nil || len(batch) != 1 || !errors.Is(batch[0].Err, ErrAmbiguousRoute) {
		t.Fatalf("same-ranked policies must be ambiguous: single=%v batch=%+v error=%v", err, batch, batchErr)
	}
	exec(`UPDATE effective_authority_routes SET selector_ref=$2 WHERE source_policy_id=$1::uuid`, secondPolicy, owner)
	check("same-ranked equivalent policies have stable source", owner, true)
}

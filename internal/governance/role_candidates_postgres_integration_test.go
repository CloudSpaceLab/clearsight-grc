//go:build postgres && postgresintegration

package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRoleCandidatePolicyApproval(t *testing.T) {
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
	tenant := rowID(`INSERT INTO tenants(slug,name) VALUES('role-candidates-' || uuidv7()::text,'Role candidates test') RETURNING id::text`)
	defer cleanupGovernanceRevisionFixture(ctx, pool, tenant)
	foreignTenant := rowID(`INSERT INTO tenants(slug,name) VALUES('role-candidates-foreign-' || uuidv7()::text,'Other tenant') RETURNING id::text`)
	defer cleanupGovernanceRevisionFixture(ctx, pool, foreignTenant)
	foreignEntity := rowID(`INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,'FOREIGN','Other tenant bank','NG') RETURNING id::text`, foreignTenant)
	entity := rowID(`INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,'TEST','Test bank','NG') RETURNING id::text`, tenant)
	otherEntity := rowID(`INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,'OTHER','Other bank','NG') RETURNING id::text`, tenant)
	principal := func(name string) string {
		return rowID(`INSERT INTO principals(tenant_id,kind,display_name,external_ref,valid_from) VALUES($1::uuid,'PERSON',$2,$2,now()-interval '1 day') RETURNING id::text`, tenant, name)
	}
	maker, checker := principal("Policy maker"), principal("Independent checker")
	first, second := principal("First author"), principal("Second author")
	role := rowID(`INSERT INTO role_templates(tenant_id,code,name,responsibilities,valid_from) VALUES($1::uuid,'FORM_AUTHOR','Form author',ARRAY['ACCOUNTABLE_OWNER'],now()-interval '1 day') RETURNING id::text`, tenant)
	position := func(code, person string) string {
		return rowID(`INSERT INTO org_positions(tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3,$3,$4::uuid,now()-interval '1 day') RETURNING id::text`, tenant, entity, code, person)
	}
	firstPos, secondPos := position("FIRST", first), position("SECOND", second)
	for _, pos := range []string{firstPos, secondPos} {
		mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,scope,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,jsonb_build_object('legal_entity_id',$4::text),now()-interval '1 day')`, tenant, pos, role, entity)
	}
	definition := func(kind, ref, scope string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{"rules":[{"id":"form-author","legal_entity_id":%q,"object_type":"FORM_TEMPLATE","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","decision_type":"forms.template.revise","selector":{"kind":%q,"ref":%q}}]}`, scope, kind, ref))
	}
	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	policy, err := svc.CreatePolicy(ctx, CreatePolicyInput{TenantID: tenant, LegalEntityID: entity, Code: "FORMS-AUTHOR", Name: "Forms author", MakerID: maker, Definition: definition("ROLE", "FORM_AUTHOR", entity)})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.SubmitPolicy(ctx, TransitionInput{TenantID: tenant, LegalEntityID: entity, ID: policy.ID, ActorID: maker, ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	approval := TransitionInput{TenantID: tenant, LegalEntityID: entity, ID: policy.ID, ActorID: maker, ExpectedVersion: policy.Version, Rationale: "Review author candidates"}
	if _, err := svc.ApprovePolicy(ctx, approval); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("same maker approved: %v", err)
	}
	approval.ActorID = checker
	approved, err := svc.ApprovePolicy(ctx, approval)
	if err != nil {
		t.Errorf("two eligible role candidates must pass independent approval: %v", err)
	} else {
		if approved.Status != PolicyActive || approved.CheckerID != checker {
			t.Fatalf("approval provenance lost: %+v", approved)
		}
		resolution, err := authority.NewPostgresService(pool).Resolve(ctx, authority.ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_TEMPLATE", ObjectID: role, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2, At: time.Now().UTC()})
		if err != nil || !resolution.AllowsPrincipal(first) || !resolution.AllowsPrincipal(second) || resolution.AllowsPrincipal(checker) {
			t.Fatalf("approved candidate route differs from execution: %+v %v", resolution, err)
		}
		for _, test := range []struct {
			name, sql string
			args      []any
		}{
			{"binding excludes entity", `UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$1::text) WHERE position_id=$2::uuid`, []any{otherEntity, secondPos}},
			{"binding belongs to foreign tenant", `UPDATE position_role_bindings SET tenant_id=$1::uuid WHERE position_id=$2::uuid`, []any{foreignTenant, secondPos}},
			{"position belongs to foreign tenant", `UPDATE org_positions SET tenant_id=$1::uuid WHERE id=$2::uuid`, []any{foreignTenant, secondPos}},
			{"occupant belongs to foreign tenant", `UPDATE principals SET tenant_id=$1::uuid WHERE id=$2::uuid`, []any{foreignTenant, second}},
			{"position has no approved entity", `UPDATE org_positions SET legal_entity_id=NULL WHERE id=$1::uuid`, []any{secondPos}},
		} {
			t.Run("execution/"+test.name, func(t *testing.T) {
				tx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(ctx)
				if _, err := tx.Exec(ctx, test.sql, test.args...); err != nil {
					t.Fatal(err)
				}
				runtimeCtx := authority.WithPostgresTransaction(ctx, tx)
				resolver := authority.NewPostgresService(pool)
				input := authority.ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_TEMPLATE", ObjectID: role, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2, At: time.Now().UTC()}
				result, err := resolver.Resolve(runtimeCtx, input)
				if err != nil || !result.AllowsPrincipal(first) || result.AllowsPrincipal(second) {
					t.Errorf("single resolution admitted excluded holder: %+v %v", result, err)
				}
				outcomes, err := resolver.(authority.BatchResolver).ResolveMany(runtimeCtx, []authority.ResolveInput{input})
				if err != nil || len(outcomes) != 1 || outcomes[0].Err != nil || !outcomes[0].Resolution.AllowsPrincipal(first) || outcomes[0].Resolution.AllowsPrincipal(second) {
					t.Errorf("batch resolution admitted excluded holder: %+v %v", outcomes, err)
				}
			})
		}
	}
	// Each preview uses real PostgreSQL and rolls back eligibility changes.
	for _, test := range []struct {
		name, sql, kind, ref, entity string
		args                         []any
		conflict                     bool
	}{
		{name: "empty role", kind: "ROLE", ref: "MISSING", conflict: true},
		{name: "other legal entity", kind: "ROLE", ref: "FORM_AUTHOR", entity: otherEntity, conflict: true},
		{name: "foreign legal entity", kind: "ROLE", ref: "FORM_AUTHOR", entity: foreignEntity, conflict: true},
		{name: "binding belongs to another tenant", sql: `UPDATE position_role_bindings SET tenant_id=$1::uuid WHERE role_template_id=$2::uuid`, args: []any{foreignTenant, role}, conflict: true},
		{name: "position belongs to another tenant", sql: `UPDATE org_positions SET tenant_id=$1::uuid WHERE tenant_id=$2::uuid`, args: []any{foreignTenant, tenant}, conflict: true},
		{name: "occupants belong to another tenant", sql: `UPDATE principals SET tenant_id=$1::uuid WHERE id IN($2::uuid,$3::uuid)`, args: []any{foreignTenant, first, second}, conflict: true},
		{name: "bindings exclude requested entity", sql: `UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$1::text) WHERE role_template_id=$2::uuid`, args: []any{otherEntity, role}, conflict: true},
		{name: "duplicate binding counts one principal", sql: `INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,now()-interval '1 day')`, args: []any{tenant, firstPos, role}},
		{name: "active role with future expiry", sql: `UPDATE role_templates SET valid_until=now()+interval '1 day' WHERE id=$1::uuid`, args: []any{role}},
		{name: "active binding with future expiry", sql: `UPDATE position_role_bindings SET valid_until=now()+interval '1 day' WHERE role_template_id=$1::uuid`, args: []any{role}},
		{name: "active position with future expiry", sql: `UPDATE org_positions SET valid_until=now()+interval '1 day' WHERE tenant_id=$1::uuid`, args: []any{tenant}},
		{name: "active principal with future expiry", sql: `UPDATE principals SET valid_until=now()+interval '1 day' WHERE id IN($1::uuid,$2::uuid)`, args: []any{first, second}},
		{name: "inactive occupants", sql: `UPDATE principals SET status='INACTIVE' WHERE id IN($1::uuid,$2::uuid)`, args: []any{first, second}, conflict: true},
		{name: "future role", sql: `UPDATE role_templates SET valid_from=now()+interval '1 day' WHERE id=$1::uuid`, args: []any{role}, conflict: true},
		{name: "future bindings", sql: `UPDATE position_role_bindings SET valid_from=now()+interval '1 day' WHERE role_template_id=$1::uuid`, args: []any{role}, conflict: true},
		{name: "future positions", sql: `UPDATE org_positions SET valid_from=now()+interval '1 day' WHERE tenant_id=$1::uuid`, args: []any{tenant}, conflict: true},
		{name: "future occupants", sql: `UPDATE principals SET valid_from=now()+interval '1 day' WHERE id IN($1::uuid,$2::uuid)`, args: []any{first, second}, conflict: true},
		{name: "expired bindings", sql: `UPDATE position_role_bindings SET valid_until=now()-interval '1 hour' WHERE role_template_id=$1::uuid`, args: []any{role}, conflict: true},
		{name: "direct principal duplicate external reference", kind: "PRINCIPAL", ref: "First author", sql: `UPDATE principals SET external_ref='First author' WHERE id=$1::uuid`, args: []any{second}, conflict: true},
		{name: "direct principal valid", kind: "PRINCIPAL", ref: first},
		{name: "direct principal inactive", kind: "PRINCIPAL", ref: first, sql: `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, args: []any{first}, conflict: true},
		{name: "direct position valid", kind: "POSITION", ref: "FIRST"},
		{name: "direct position ambiguous", kind: "POSITION", ref: "FIRST", sql: `UPDATE org_positions SET code='FIRST',valid_until=now()+interval '1 day' WHERE id=$1::uuid`, args: []any{secondPos}, conflict: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if test.sql != "" {
				if _, err := tx.Exec(ctx, test.sql, test.args...); err != nil {
					t.Fatal(err)
				}
			}
			kind, ref, scope := test.kind, test.ref, test.entity
			if kind == "" {
				kind = "ROLE"
			}
			if ref == "" {
				ref = "FORM_AUTHOR"
			}
			if scope == "" {
				scope = entity
			}
			probe := policy
			probe.Definition = definition(kind, ref, scope)
			findings, err := policyConflicts(ctx, tx, probe)
			if err != nil {
				t.Fatal(err)
			}
			if (len(findings) > 0) != test.conflict {
				t.Fatalf("conflict=%v, expected %v: %+v", len(findings) > 0, test.conflict, findings)
			}
		})
	}
	// A role from another tenant cannot make an otherwise empty route eligible.
	probe := policy
	probe.TenantID = foreignTenant
	probe.LegalEntityID = foreignEntity
	probe.Definition = definition("ROLE", "FORM_AUTHOR", foreignEntity)
	if findings, err := repo.PolicyConflicts(ctx, probe); err != nil || len(findings) == 0 {
		t.Fatalf("tenant scope escaped: %+v %v", findings, err)
	}
}

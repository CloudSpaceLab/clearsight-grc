//go:build postgres && postgresintegration

package authority

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exercise the actual deploy fixture in a rollback-only transaction, including
// the production resolver. No shell or fixed test database name is required.
func TestPostgresDemoFormsAuthorFixture(t *testing.T) {
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	ctx = WithPostgresTransaction(ctx, tx)
	data, err := os.ReadFile("../../deploy/scripts/seed-demo-foundation.sh")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ReplaceAll(string(data), "\r\n", "\n")
	_, sql, ok := strings.Cut(sql, "<<'SQL'\nBEGIN;\n")
	if !ok {
		t.Fatal("seed transaction start missing")
	}
	sql, _, ok = strings.Cut(sql, "\nCOMMIT;\nSQL")
	if !ok {
		t.Fatal("seed transaction end missing")
	}
	sql = strings.ReplaceAll(sql, ":'demo_staff_email'", "''")
	seed := func() {
		t.Helper()
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	seed()
	const tenant = "00000000-0000-4000-8000-000000000001"
	const entity = "00000000-0000-4000-8000-000000000002"
	const cro = "00000000-0000-4000-8000-000000000101"
	const owner = "00000000-0000-4000-8000-000000000107"
	const reviewer = "00000000-0000-4000-8000-000000000106"
	service := NewPostgresService(pool)
	input := ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_TEMPLATE", ObjectID: "00000000-0000-4000-8000-000000000999", Responsibility: ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2, At: time.Now().UTC().Add(time.Second)}
	for _, command := range []string{"forms.template.create", "forms.template.revise", "forms.template.transition"} {
		t.Run(command, func(t *testing.T) {
			in := input
			in.DecisionType = command
			if command == "forms.template.create" {
				in.ObjectType = "LEGAL_ENTITY"
				in.ObjectID = entity
			}
			simulation, err := service.Simulate(ctx, in)
			if err != nil || simulation.Selected == nil {
				t.Fatalf("simulation: %+v %v", simulation, err)
			}
			result, err := service.Resolve(ctx, in)
			if err != nil || !result.AllowsPrincipal(cro) || !result.AllowsPrincipal(owner) || result.AllowsPrincipal(reviewer) {
				t.Fatalf("CRO and Program Owner must author, independent reviewer must not: %+v %v", result, err)
			}
			if result.PolicyVersion != "CLEARSIGHT-DEMO-FORMS-AUTHOR:v1" {
				t.Fatalf("unexpected policy: %+v", result)
			}
			outcomes, err := service.(BatchResolver).ResolveMany(ctx, []ResolveInput{in})
			if err != nil || len(outcomes) != 1 || outcomes[0].Err != nil || !outcomes[0].Resolution.AllowsPrincipal(cro) || !outcomes[0].Resolution.AllowsPrincipal(owner) {
				t.Fatalf("batch author resolution: %+v %v", outcomes, err)
			}
		})
	}
	for _, in := range []ResolveInput{
		{TenantID: tenant, LegalEntityID: entity, ObjectType: "PROGRAM", ObjectID: input.ObjectID, Responsibility: ResponsibilityOwner, DecisionType: "program.update", Materiality: 2, At: input.At},
		{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_TEMPLATE", ObjectID: input.ObjectID, Responsibility: ResponsibilityOwner, DecisionType: "program.monitoring.collect", Materiality: 2, At: input.At},
	} {
		result, err := service.Resolve(ctx, in)
		if err != nil || !result.AllowsPrincipal(owner) || result.AllowsPrincipal(cro) {
			t.Fatalf("unrelated ownership changed: %+v %v", result, err)
		}
	}
	review := input
	review.DecisionType = "forms.template.transition"
	review.Responsibility = ResponsibilityReviewer
	review.Materiality = 3
	result, err := service.Resolve(ctx, review)
	if err != nil || !result.AllowsPrincipal(reviewer) || result.AllowsPrincipal(cro) || result.AllowsPrincipal(owner) {
		t.Fatalf("independent approval changed: %+v %v", result, err)
	}
	for _, change := range []func(*ResolveInput){func(in *ResolveInput) { in.TenantID = "missing-tenant" }, func(in *ResolveInput) { in.LegalEntityID = "missing-entity" }} {
		in := input
		change(&in)
		if result, err := service.Resolve(ctx, in); err == nil && result.AllowsPrincipal(cro) {
			t.Fatalf("scope escaped: %+v", result)
		}
	}
	// Repeat installation leaves the new managed records and their receipts exact.
	snapshot := func() string {
		t.Helper()
		var value string
		err := tx.QueryRow(ctx, `SELECT jsonb_build_object(
			'policy',(SELECT to_jsonb(p) FROM routing_policies p WHERE id='00000000-0000-4000-8000-000000000206'),
			'version',(SELECT to_jsonb(v) FROM routing_policy_versions v WHERE id='00000000-0000-4000-8000-000000000207'),
			'role',(SELECT to_jsonb(r) FROM role_templates r WHERE id='00000000-0000-4000-8000-000000000409'),
			'bindings',(SELECT jsonb_agg(to_jsonb(b) ORDER BY id) FROM position_role_bindings b WHERE role_template_id='00000000-0000-4000-8000-000000000409'),
			'decisions',(SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM governance_decisions d WHERE object_id='00000000-0000-4000-8000-000000000206'),
			'outbox',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM outbox_events e WHERE aggregate_id='00000000-0000-4000-8000-000000000206'))::text`).Scan(&value)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	seed()
	if after := snapshot(); before != after {
		t.Fatalf("repeat installation rewrote managed records")
	}
	var receiptCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM governance_decisions WHERE object_id='00000000-0000-4000-8000-000000000206' AND actor_type='SYSTEM' AND actor_id IS NULL AND from_state='UNINSTALLED' AND to_state='ACTIVE' AND rationale LIKE '%no live bank approval%'`).Scan(&receiptCount); err != nil || receiptCount != 1 {
		t.Fatalf("expected one explicitly sample installation receipt, got %d: %v", receiptCount, err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id='00000000-0000-4000-8000-000000000206' AND event_type='RoutingPolicyStateChanged' AND payload->>'sample_fixture'='true'`).Scan(&receiptCount); err != nil || receiptCount != 1 {
		t.Fatalf("expected one transactional sample outbox event, got %d: %v", receiptCount, err)
	}
	for _, mutation := range []string{
		`UPDATE routing_policy_versions SET definition='{"rules":[]}' WHERE id='00000000-0000-4000-8000-000000000207'`,
		`UPDATE routing_policy_versions SET effective_until='2027-01-01' WHERE id='00000000-0000-4000-8000-000000000207'`,
		`UPDATE routing_policies SET status='RETIRED' WHERE id='00000000-0000-4000-8000-000000000206'`,
		`UPDATE position_role_bindings SET scope='{}' WHERE id='00000000-0000-4000-8000-000000000510'`,
		`UPDATE role_templates SET capabilities=ARRAY['manage:program'] WHERE id='00000000-0000-4000-8000-000000000409'`,
		`UPDATE routing_policy_versions SET checksum=repeat('0',64) WHERE id='00000000-0000-4000-8000-000000000207'`,
		`UPDATE routing_policy_versions SET approved_by='00000000-0000-4000-8000-000000000101' WHERE id='00000000-0000-4000-8000-000000000207'`,
		`UPDATE routing_policies SET current_version=2 WHERE id='00000000-0000-4000-8000-000000000206'`,
		`INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,scope,valid_from) VALUES('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000306','00000000-0000-4000-8000-000000000409','{}','2020-01-01')`,
	} {
		if _, err := tx.Exec(ctx, "SAVEPOINT mismatch"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, mutation); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, sql); err == nil {
			t.Fatalf("managed mismatch was silently accepted: %s", mutation)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT mismatch"); err != nil {
			t.Fatal(err)
		}
	}
	// A conflict removes only the affected author through the current resolver.
	if _, err := tx.Exec(ctx, "SAVEPOINT conflict"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES('00000000-0000-4000-8000-000000000001','DEMO-FORMS-CRO-CONFLICT','ACCOUNTABLE_OWNER','CRO','ACTIVE','2020-01-01')`); err != nil {
		t.Fatal(err)
	}
	result, err = service.Resolve(ctx, input)
	if err != nil || result.AllowsPrincipal(cro) || !result.AllowsPrincipal(owner) {
		t.Fatalf("conflicted author retained CRO: %+v %v", result, err)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT conflict"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE position_role_bindings SET valid_until=clock_timestamp() WHERE id='00000000-0000-4000-8000-000000000510'`); err != nil {
		t.Fatal(err)
	}
	result, err = service.Resolve(ctx, input)
	if err != nil || result.AllowsPrincipal(cro) || !result.AllowsPrincipal(owner) {
		t.Fatalf("revoked author binding retained CRO: %+v %v", result, err)
	}
}

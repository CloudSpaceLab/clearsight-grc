//go:build postgres && postgresintegration

package authority

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresEffectiveAuthorityDelegationGrantAndSegregation(t *testing.T) {
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

	const (
		tenantID   = "84444444-4444-7444-8444-444444444441"
		entityID   = "84444444-4444-7444-8444-444444444442"
		ownerID    = "84444444-4444-7444-8444-444444444443"
		delegateID = "84444444-4444-7444-8444-444444444444"
		blockedID  = "84444444-4444-7444-8444-444444444445"
		policyID   = "84444444-4444-7444-8444-444444444446"
		versionID  = "84444444-4444-7444-8444-444444444447"
		roleID     = "84444444-4444-7444-8444-444444444448"
		positionID = "84444444-4444-7444-8444-444444444449"
		bindingID  = "84444444-4444-7444-8444-444444444450"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'authority-effective-test','Authority Effective Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Original owner','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Delegated owner','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','Segregated owner','ACTIVE',$5)`, ownerID, delegateID, blockedID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	definition, _ := json.Marshal(map[string]any{"rules": []map[string]any{{
		"id": "owner-route", "legal_entity_id": "BANK-NG", "object_type": "MATTER", "object_id": "*",
		"responsibility": "ACCOUNTABLE_OWNER", "decision_type": "matter.action.add", "min_materiality": 0, "priority": 100,
		"selector": map[string]any{"kind": "PRINCIPAL", "ref": ownerID},
	}}})
	if _, err := pool.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,'OWNER','Owner route','DRAFT',1,$3,1)`, policyID, tenantID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO routing_policy_versions(id,policy_id,version,definition,checksum,effective_from,approved_at) VALUES($1::uuid,$2::uuid,1,$3::jsonb,'test',$4,$4)`, versionID, policyID, string(definition), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policyID); err != nil {
		t.Fatal(err)
	}
	assertAuthorityCount(t, ctx, pool, `SELECT count(*) FROM effective_authority_routes WHERE tenant_id=$1::uuid AND source_rule_id='owner-route'`, 1, tenantID)

	if _, err := pool.Exec(ctx, `INSERT INTO delegations(tenant_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by,approved_by,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'ACCOUNTABLE_OWNER',$4::jsonb,$5,$6,'ACTIVE',$2::uuid,$7::uuid,$5,1)`, tenantID, ownerID, delegateID, `{"legal_entity_id":"BANK-NG","object_type":"MATTER"}`, now.Add(-time.Minute), now.Add(time.Hour), blockedID); err != nil {
		t.Fatal(err)
	}

	service := NewPostgresService(pool)
	input := ResolveInput{TenantID: "authority-effective-test", LegalEntityID: "BANK-NG", ObjectType: "MATTER", ObjectID: "84444444-4444-7444-8444-444444444499", Responsibility: ResponsibilityOwner, DecisionType: "matter.action.add", Materiality: 3, At: now}
	resolution, err := service.Resolve(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.AllowsPrincipal(ownerID) || !resolution.AllowsPrincipal(delegateID) {
		t.Fatalf("active delegation did not expand the effective route: %#v", resolution)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,limits,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.action.add','{"max_materiality":4}'::jsonb,$4,'grant:v1')`, tenantID, entityID, ownerID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	resolution, err = service.Resolve(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.AllowsPrincipal(delegateID) {
		t.Fatalf("delegate did not inherit the origin grant: %#v", resolution)
	}
	input.Materiality = 5
	if _, err := service.Resolve(ctx, input); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("grant materiality limit did not fail closed: %v", err)
	}
	input.Materiality = 3

	if _, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,valid_from) VALUES($1::uuid,$2::uuid,'CONFLICT_ROLE','Conflict role','',ARRAY['ACCOUNTABLE_OWNER'],$3)`, roleID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'CONFLICT-POS','Conflict position',$4::uuid,$5)`, positionID, tenantID, entityID, delegateID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,priority,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,100,$5)`, bindingID, tenantID, positionID, roleID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES($1::uuid,'NO-CONFLICT-OWNER','ACCOUNTABLE_OWNER','CONFLICT_ROLE','ACTIVE',$2)`, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	resolution, err = service.Resolve(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.AllowsPrincipal(delegateID) || !resolution.AllowsPrincipal(ownerID) {
		t.Fatalf("segregation did not remove only the conflicted delegate: %#v", resolution)
	}

	if _, err := pool.Exec(ctx, `UPDATE delegations SET status='EXPIRED' WHERE tenant_id=$1::uuid`, tenantID); err != nil {
		t.Fatal(err)
	}
	resolution, err = service.Resolve(ctx, input)
	if err != nil || resolution.AllowsPrincipal(delegateID) {
		t.Fatalf("expired delegation remained effective: %#v err=%v", resolution, err)
	}
}

func TestPostgresEffectiveAuthorityAssignmentAndAmbiguity(t *testing.T) {
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

	const (
		tenantID = "85555555-5555-7555-8555-555555555551"
		entityID = "85555555-5555-7555-8555-555555555552"
		firstID  = "85555555-5555-7555-8555-555555555553"
		secondID = "85555555-5555-7555-8555-555555555554"
		objectID = "85555555-5555-7555-8555-555555555555"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'authority-assignment-test','Authority Assignment Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'ENTITY','Entity','NG',$3)`, entityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$3::uuid,'PERSON','First','ACTIVE',$4),($2::uuid,$3::uuid,'PERSON','Second','ACTIVE',$4)`, firstID, secondID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','MATTER',$4::uuid,100,$5,'assignment:v1','matter.outcome.record')`, tenantID, entityID, firstID, objectID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	service := NewPostgresService(pool)
	input := ResolveInput{TenantID: "authority-assignment-test", LegalEntityID: "ENTITY", ObjectType: "MATTER", ObjectID: objectID, Responsibility: ResponsibilityReviewer, DecisionType: "matter.outcome.record", Materiality: 4, At: now}
	resolution, err := service.Resolve(ctx, input)
	if err != nil || !resolution.AllowsPrincipal(firstID) {
		t.Fatalf("assignment did not resolve: %#v err=%v", resolution, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','MATTER',$4::uuid,100,$5,'assignment:v2','matter.outcome.record')`, tenantID, entityID, secondID, objectID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(ctx, input); !errors.Is(err, ErrAmbiguousRoute) {
		t.Fatalf("same-rank conflicting assignments did not fail closed: %v", err)
	}
}

func assertAuthorityCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count=%d want=%d query=%s", got, want, query)
	}
}

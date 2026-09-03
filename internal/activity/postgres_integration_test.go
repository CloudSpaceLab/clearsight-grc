//go:build postgres && postgresintegration

package activity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresActivityFederatesIdentityDecisionsWithoutLeakingOtherHistory(t *testing.T) {
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
		tenantA          = "8b000000-0000-7000-8000-000000000001"
		entityA          = "8b000000-0000-7000-8000-000000000002"
		actorA           = "8b000000-0000-7000-8000-000000000003"
		roleA            = "8b000000-0000-7000-8000-000000000004"
		sourceA          = "8b000000-0000-7000-8000-000000000005"
		groupA           = "8b000000-0000-7000-8000-000000000006"
		bindingA         = "8b000000-0000-7000-8000-000000000007"
		bindingDecisionA = "8b000000-0000-7000-8000-000000000008"
		scimDecisionA    = "8b000000-0000-7000-8000-000000000009"
		outboxA          = "8b000000-0000-7000-8000-000000000010"
		matterA          = "8b000000-0000-7000-8000-000000000011"
		routingDecisionA = "8b000000-0000-7000-8000-000000000012"
		routingPolicyA   = "8b000000-0000-7000-8000-000000000013"
		tenantB          = "8b000000-0000-7000-8000-000000000014"
		actorB           = "8b000000-0000-7000-8000-000000000015"
		scimDecisionB    = "8b000000-0000-7000-8000-000000000016"
		sourceB          = "8b000000-0000-7000-8000-000000000017"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantA, tenantB)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustExecActivity(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'activity-a','Activity A'),($2::uuid,'activity-b','Activity B')`, tenantA, tenantB)
	mustExecActivity(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-A','Bank A','NG',$3)`, entityA, tenantA, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Identity administrator','ACTIVE',$4),($3::uuid,$5::uuid,'PERSON','Other tenant administrator','ACTIVE',$4)`, actorA, tenantA, actorB, now.Add(-time.Hour), tenantB)
	mustExecActivity(t, ctx, pool, `INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,capabilities,valid_from) VALUES($1::uuid,$2::uuid,'IDENTITY_REVIEWER','Identity reviewer',ARRAY['REVIEWER'],ARRAY['IDENTITY_READ'],$3)`, roleA, tenantA, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO scim_sources(id,tenant_id,code,token_hash,status,created_at,updated_at) VALUES($1::uuid,$2::uuid,'ENTRA',decode(repeat('ab',32),'hex'),'ACTIVE',$3,$3)`, sourceA, tenantA, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO directory_groups(id,tenant_id,source_id,display_name,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,'Risk reviewers',$4,$4)`, groupA, tenantA, sourceA, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO directory_group_role_bindings(id,tenant_id,group_id,role_template_id,legal_entity_id,valid_from,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$6)`, bindingA, tenantA, groupA, roleA, entityA, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,$2::uuid,'MATTER',$3::uuid,'MATTER_CREATED',jsonb_build_object('actor_id',$4::text,'legal_entity_id',$5::text),$6,$6,$6)`, outboxA, tenantA, matterA, actorA, entityA, now)
	mustExecActivity(t, ctx, pool, `INSERT INTO governance_decisions(id,tenant_id,object_type,object_id,from_state,to_state,actor_type,actor_id,rationale,decided_at) VALUES
		($1::uuid,$2::uuid,'DIRECTORY_GROUP_ROLE_BINDING',$3::uuid,'NONE','ACTIVE','PRINCIPAL',$4::uuid,'DIRECTORY_GROUP_ROLE_BOUND',$5),
		($6::uuid,$2::uuid,'SCIM_SOURCE',$7::uuid,'NONE','ACTIVE','PRINCIPAL',$4::uuid,'SCIM_SOURCE_CREATED',$8),
		($9::uuid,$2::uuid,'ROUTING_POLICY',$10::uuid,'DRAFT','ACTIVE','PRINCIPAL',$4::uuid,'ROUTING_POLICY_APPROVED',$11),
		($12::uuid,$13::uuid,'SCIM_SOURCE',$14::uuid,'NONE','ACTIVE','PRINCIPAL',$15::uuid,'SCIM_SOURCE_CREATED',$8)`,
		bindingDecisionA, tenantA, bindingA, actorA, now.Add(-time.Minute),
		scimDecisionA, sourceA, now.Add(-2*time.Minute),
		routingDecisionA, routingPolicyA, now.Add(-3*time.Minute),
		scimDecisionB, tenantB, sourceB, actorB)

	service := NewService(NewPostgresRepository(pool))
	page, err := service.List(ctx, Query{TenantID: tenantA, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	bindingEventID := "decision:" + bindingDecisionA
	scimEventID := "decision:" + scimDecisionA
	if len(page.Items) != 2 || page.Items[0].ID != outboxA || page.Items[1].ID != bindingEventID || page.NextCursor != bindingEventID {
		t.Fatalf("unexpected first federated page: %#v", page)
	}
	bindingEvent := page.Items[1]
	if bindingEvent.Source != "GOVERNANCE_DECISION" || bindingEvent.EventType != "DIRECTORY_GROUP_ROLE_BOUND" || bindingEvent.Category != CategoryConfiguration || bindingEvent.ActorID != actorA || bindingEvent.ActorDisplayName != "Identity administrator" || bindingEvent.ActorKind != ActorInternalUser || bindingEvent.LegalEntityID != entityA {
		t.Fatalf("identity decision was not normalized correctly: %#v", bindingEvent)
	}

	next, err := service.List(ctx, Query{TenantID: tenantA, Limit: 2, Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Items) != 1 || next.Items[0].ID != scimEventID || next.Items[0].Source != "GOVERNANCE_DECISION" || next.NextCursor != "" {
		t.Fatalf("unexpected second federated page: %#v", next)
	}

	filtered, err := service.List(ctx, Query{TenantID: tenantA, Category: CategoryConfiguration, LegalEntityID: entityA})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != bindingEventID {
		t.Fatalf("legal-entity filter returned unrelated identity history: %#v", filtered)
	}

	got, err := service.Get(ctx, tenantA, bindingEventID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != bindingEventID || got.ObjectID != bindingA || got.Source != "GOVERNANCE_DECISION" {
		t.Fatalf("unexpected federated event detail: %#v", got)
	}
}

func mustExecActivity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMatterLifecycleProjectorRoutesPolicySelectedResponsibilityWithoutPredeterminingOutcome(t *testing.T) {
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
		tenantID      = "98888888-8888-7888-8888-888888888881"
		entityID      = "98888888-8888-7888-8888-888888888882"
		reviewerID    = "98888888-8888-7888-8888-888888888883"
		authorizerID  = "98888888-8888-7888-8888-888888888884"
		matterID      = "98888888-8888-7888-8888-888888888885"
		decisionID    = "98888888-8888-7888-8888-888888888886"
		reviewPolicy  = "98888888-8888-7888-8888-888888888887"
		reviewVersion = "98888888-8888-7888-8888-888888888888"
		authPolicy    = "98888888-8888-7888-8888-888888888889"
		authVersion   = "98888888-8888-7888-8888-888888888890"
	)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'sequence-work-test','Sequence Work Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Independent reviewer','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Entity authorizer','ACTIVE',$4)`, reviewerID, authorizerID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MAT-SEQ-1','EXCEPTION','DECISION_REQUIRED',4,'Approve temporary exception','Material exception requires governed review','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$3,$3)`, matterID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matter_decisions(id,tenant_id,matter_id,decision_type,status,options,rationale,conditions,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'EXCEPTION_APPROVAL','PROPOSED','[]'::jsonb,'Await governed review','[]'::jsonb,$4,$4,1)`, decisionID, tenantID, matterID, now.Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	activateSequencePolicy(t, ctx, pool, sequencePolicyFixture{
		TenantID: tenantID, PolicyID: reviewPolicy, VersionID: reviewVersion, Code: "EXCEPTION-REVIEW", RuleID: "review-gate",
		Responsibility: "REVIEWER", PrincipalID: reviewerID, Priority: 50, LifecycleState: "PROPOSED", LifecycleSubtype: "EXCEPTION_APPROVAL", At: now,
	})

	repo := NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	projector := &MatterLifecycleProjector{
		Repo: repo, Continuity: continuityService, Authority: authority.NewEffectivePostgresService(pool),
		Sequence: governance.NewService(governance.NewPostgresRepository(pool)), Now: func() time.Time { return now },
	}
	if err := projector.ReconcileMatter(ctx, "sequence-work-test", matterID, now); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo)
	tasks, err := service.List(ctx, ListFilter{TenantID: "sequence-work-test", PrincipalID: reviewerID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one reviewer packet, got %#v", tasks)
	}
	packet := tasks[0]
	if packet.StepKey != "decision:"+decisionID+":gate:reviewer" || packet.Responsibility != "REVIEWER" || packet.PrincipalID != reviewerID || packet.Status != StatusReady {
		t.Fatalf("unexpected reviewer packet: %#v", packet)
	}
	if packet.Context["target_status"] != "" || packet.Context["allowed_targets"] != "IN_REVIEW,RETURNED" {
		t.Fatalf("review packet pre-decided an outcome or lost its legal choices: %#v", packet.Context)
	}
	if packet.Context["sequence_rule_id"] != "EXCEPTION-REVIEW/review-gate" || packet.Context["sequence_policy_version"] != "EXCEPTION-REVIEW:v1" {
		t.Fatalf("sequence provenance missing from packet: %#v", packet.Context)
	}

	// A later, higher-priority governed sequence policy changes the next gate to
	// AUTHORIZER. Reconciliation must retire the reviewer packet and re-resolve
	// current authority without requiring a new Matter event.
	activateSequencePolicy(t, ctx, pool, sequencePolicyFixture{
		TenantID: tenantID, PolicyID: authPolicy, VersionID: authVersion, Code: "EXCEPTION-AUTH", RuleID: "authorize-gate",
		Responsibility: "AUTHORIZER", PrincipalID: authorizerID, Priority: 100, LifecycleState: "PROPOSED", LifecycleSubtype: "EXCEPTION_APPROVAL", At: now.Add(time.Minute),
	})
	if err := projector.ReconcileMatter(ctx, "sequence-work-test", matterID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	reviewerTasks, err := service.List(ctx, ListFilter{TenantID: "sequence-work-test", PrincipalID: reviewerID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(reviewerTasks) != 0 {
		t.Fatalf("obsolete reviewer packet remained active: %#v", reviewerTasks)
	}
	authorizerTasks, err := service.List(ctx, ListFilter{TenantID: "sequence-work-test", PrincipalID: authorizerID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(authorizerTasks) != 1 {
		t.Fatalf("expected one authorizer packet after policy change, got %#v", authorizerTasks)
	}
	authorizerPacket := authorizerTasks[0]
	if authorizerPacket.StepKey != "decision:"+decisionID+":gate:authorizer" || authorizerPacket.Context["target_status"] != "" {
		t.Fatalf("authorizer packet pre-decided an outcome: %#v", authorizerPacket)
	}
	if authorizerPacket.Context["allowed_targets"] != "APPROVED,CONDITIONALLY_APPROVED,REJECTED,SUPERSEDED" {
		t.Fatalf("authorizer packet lost current legal outcomes: %#v", authorizerPacket.Context)
	}
}

type sequencePolicyFixture struct {
	TenantID         string
	PolicyID         string
	VersionID        string
	Code             string
	RuleID           string
	Responsibility   string
	PrincipalID      string
	Priority         int
	LifecycleState   string
	LifecycleSubtype string
	At               time.Time
}

func activateSequencePolicy(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture sequencePolicyFixture) {
	t.Helper()
	definition, err := json.Marshal(map[string]any{"rules": []map[string]any{{
		"id": fixture.RuleID, "legal_entity_id": "BANK-NG", "object_type": "MATTER", "object_id": "*",
		"responsibility": fixture.Responsibility, "decision_type": "matter.decision.record", "min_materiality": 0, "priority": fixture.Priority,
		"selector": map[string]any{"kind": "PRINCIPAL", "ref": fixture.PrincipalID},
		"lifecycle_type": "DECISION", "lifecycle_state": fixture.LifecycleState, "lifecycle_subtype": fixture.LifecycleSubtype,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,code,name,status,current_version,approved_at,version)
		VALUES($1::uuid,$2::uuid,$3,$3,'DRAFT',1,$4,1)`, fixture.PolicyID, fixture.TenantID, fixture.Code, fixture.At); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO routing_policy_versions(id,policy_id,version,definition,checksum,effective_from,approved_at)
		VALUES($1::uuid,$2::uuid,1,$3::jsonb,'sequence-test',$4,$4)`, fixture.VersionID, fixture.PolicyID, string(definition), fixture.At.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, fixture.PolicyID); err != nil {
		t.Fatal(err)
	}
	var routes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM effective_authority_routes WHERE tenant_id=$1::uuid AND source_rule_id=$2`, fixture.TenantID, fixture.RuleID).Scan(&routes); err != nil {
		t.Fatal(err)
	}
	if routes != 1 {
		t.Fatalf("sequence authority route was not materialized: %d", routes)
	}
}

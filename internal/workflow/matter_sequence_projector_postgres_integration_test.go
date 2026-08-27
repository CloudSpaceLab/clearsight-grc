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

	const tenantID = "98888888-8888-7888-8888-888888888881"
	const entityID = "98888888-8888-7888-8888-888888888882"
	const reviewerID = "98888888-8888-7888-8888-888888888883"
	const authorizerID = "98888888-8888-7888-8888-888888888884"
	const matterID = "98888888-8888-7888-8888-888888888885"
	const decisionID = "98888888-8888-7888-8888-888888888886"
	const reviewPolicyID = "98888888-8888-7888-8888-888888888887"
	const reviewVersionID = "98888888-8888-7888-8888-888888888888"
	const authorizerPolicyID = "98888888-8888-7888-8888-888888888889"
	const authorizerVersionID = "98888888-8888-7888-8888-888888888890"
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})
	mustExecSequence(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'sequence-work-test','Sequence Work Test')`, tenantID)
	mustExecSequence(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour))
	mustExecSequence(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Independent reviewer','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Entity authorizer','ACTIVE',$4)`, reviewerID, authorizerID, tenantID, now.Add(-24*time.Hour))
	mustExecSequence(t, ctx, pool, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-SEQ-1','EXCEPTION','DECISION_REQUIRED',4,'Approve temporary exception','Material exception requires governed review','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4,$4)`, matterID, tenantID, entityID, now.Add(-time.Hour))
	mustExecSequence(t, ctx, pool, `INSERT INTO matter_decisions(id,tenant_id,matter_id,decision_type,status,options,rationale,conditions,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'EXCEPTION_APPROVAL','PROPOSED','[]'::jsonb,'Await governed review','[]'::jsonb,$4,$4,1)`, decisionID, tenantID, matterID, now.Add(-30*time.Minute))

	// Sequence rules are selector-free and therefore cannot materialize actor
	// authority. A separate ordinary authority rule resolves who holds the
	// selected responsibility. The sequence rule deliberately stores the legal
	// entity UUID while the lifecycle projector derives the canonical code.
	activateSequencePolicy(t, ctx, pool, tenantID, entityID, reviewPolicyID, reviewVersionID, "EXCEPTION-REVIEW", "review-gate", "REVIEWER", reviewerID, 50, now)

	repo := NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	projector := &MatterLifecycleProjector{
		Repo:       repo,
		Continuity: continuityService,
		Authority:  authority.NewEffectivePostgresService(pool),
		Sequence:   governance.NewService(governance.NewPostgresRepository(pool)),
		Now:        func() time.Time { return now },
	}
	if err := projector.ReconcileMatter(ctx, "sequence-work-test", matterID, now); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo)
	reviewerTasks, err := service.List(ctx, ListFilter{TenantID: "sequence-work-test", PrincipalID: reviewerID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(reviewerTasks) != 1 {
		t.Fatalf("expected one reviewer packet, got %#v", reviewerTasks)
	}
	packet := reviewerTasks[0]
	if packet.StepKey != "decision:"+decisionID+":gate:reviewer" || packet.Responsibility != "REVIEWER" || packet.PrincipalID != reviewerID || packet.Status != StatusReady {
		t.Fatalf("unexpected reviewer packet: %#v", packet)
	}
	if packet.Context["target_status"] != "" || packet.Context["allowed_targets"] != "IN_REVIEW,RETURNED" {
		t.Fatalf("review packet pre-decided an outcome or lost legal choices: %#v", packet.Context)
	}
	if packet.Context["sequence_rule_id"] != "EXCEPTION-REVIEW/review-gate" || packet.Context["sequence_policy_version"] != "EXCEPTION-REVIEW:v1" {
		t.Fatalf("sequence provenance missing from packet: %#v", packet.Context)
	}

	activateSequencePolicy(t, ctx, pool, tenantID, entityID, authorizerPolicyID, authorizerVersionID, "EXCEPTION-AUTH", "authorize-gate", "AUTHORIZER", authorizerID, 100, now.Add(time.Minute))
	if err := projector.ReconcileMatter(ctx, "sequence-work-test", matterID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	reviewerTasks, err = service.List(ctx, ListFilter{TenantID: "sequence-work-test", PrincipalID: reviewerID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
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

func activateSequencePolicy(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, legalEntityRef, policyID, versionID, code, ruleID, responsibility, principalID string, priority int, at time.Time) {
	t.Helper()
	authorityRuleID := ruleID + "-authority"
	definition, err := json.Marshal(map[string]any{
		"rules": []map[string]any{
			{
				"id":                ruleID,
				"legal_entity_id":   legalEntityRef,
				"object_type":       "MATTER",
				"object_id":         "*",
				"responsibility":    responsibility,
				"decision_type":     "matter.decision.record",
				"min_materiality":   0,
				"priority":          priority,
				"lifecycle_type":    "DECISION",
				"lifecycle_state":   "PROPOSED",
				"lifecycle_subtype": "EXCEPTION_APPROVAL",
			},
			{
				"id":              authorityRuleID,
				"legal_entity_id": legalEntityRef,
				"object_type":     "MATTER",
				"object_id":       "*",
				"responsibility":  responsibility,
				"decision_type":   "matter.decision.record",
				"min_materiality": 0,
				"priority":        priority,
				"selector":        map[string]any{"kind": "PRINCIPAL", "ref": principalID},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustExecSequence(t, ctx, pool, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$4,'DRAFT',1,$5,1)`, policyID, tenantID, legalEntityRef, code, at)
	mustExecSequence(t, ctx, pool, `INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,1,$4::jsonb,'sequence-test',$5,$5)`, versionID, policyID, legalEntityRef, string(definition), at.Add(-time.Second))
	mustExecSequence(t, ctx, pool, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policyID)

	var sequenceRoutes, authorityRoutes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM effective_authority_routes WHERE tenant_id=$1::uuid AND source_rule_id=$2`, tenantID, ruleID).Scan(&sequenceRoutes); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM effective_authority_routes WHERE tenant_id=$1::uuid AND source_rule_id=$2`, tenantID, authorityRuleID).Scan(&authorityRoutes); err != nil {
		t.Fatal(err)
	}
	if sequenceRoutes != 0 {
		t.Fatalf("selector-free lifecycle sequence rule granted actor authority: %d routes", sequenceRoutes)
	}
	if authorityRoutes != 1 {
		t.Fatalf("ordinary authority rule did not materialize exactly once: %d routes", authorityRoutes)
	}
}

func mustExecSequence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

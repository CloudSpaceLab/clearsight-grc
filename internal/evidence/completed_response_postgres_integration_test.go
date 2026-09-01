//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCompletedResponsesUseBoundedScoreIndexAndEntityScope(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID      = "9e111111-1111-7111-8111-111111111111"
		entityID      = "9e111111-1111-7111-8111-111111111112"
		otherEntityID = "9e111111-1111-7111-8111-111111111113"
		actorID       = "9e111111-1111-7111-8111-111111111114"
		formID        = "9e111111-1111-7111-8111-111111111115"
		otherFormID   = "9e111111-1111-7111-8111-111111111116"
	)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cleanupResponseWorkspaceTenant(ctx, pool, tenantID)
	defer cleanupResponseWorkspaceTenant(context.Background(), pool, tenantID)
	setupResponseWorkspaceFixture(t, ctx, pool, tenantID, "completed-response-load", entityID, actorID, formID, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'OTHER','Other completed-response entity','NG',$5);
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,code,name,purpose,presentation,scoring_mode,score_profile,sections,fields,
			status,is_current,effective_from,version,created_by,created_at,updated_at
		)
		SELECT $3::uuid,tenant_id,$1::uuid,'OTHER-SCORE-FORM',name,purpose,presentation,scoring_mode,score_profile,sections,fields,
		       status,is_current,effective_from,version,created_by,created_at,updated_at
		FROM monitoring_form_templates WHERE tenant_id=$2::uuid AND id=$4::uuid AND version=1`,
		pgx.QueryExecModeSimpleProtocol, otherEntityID, tenantID, otherFormID, formID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	seedCompletedResponseRows(t, ctx, pool, tenantID, entityID, actorID, formID, "primary", 10000, now)
	seedCompletedResponseRows(t, ctx, pool, tenantID, otherEntityID, actorID, otherFormID, "other", 12, now.Add(time.Hour))
	if _, err := pool.Exec(ctx, `ANALYZE capture_response_revisions; ANALYZE capture_form_distributions`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}

	var plan string
	if err := pool.QueryRow(ctx, `
		EXPLAIN (FORMAT JSON)
		SELECT r.id
		FROM capture_response_revisions r
		JOIN capture_form_distributions d
		  ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid
		  AND r.is_current AND r.score_state IN ('FINAL','PROVISIONAL') AND r.adverse_score >= 0
		ORDER BY r.adverse_score DESC NULLS LAST,r.created_at DESC,r.id DESC
		LIMIT 38`, tenantID, entityID).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan, "capture_response_revisions_current_adverse_idx") {
		t.Fatalf("completed-response query did not use the bounded score index: %s", plan)
	}

	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	minimum := 0.0
	query := CompletedResponseQuery{
		TenantID: tenantID, LegalEntityID: entityID, PrincipalID: actorID,
		States: []ResponseScoreState{ResponseScoreFinal, ResponseScoreProvisional}, AdverseMinimum: &minimum,
		CurrentOnly: true, Sort: ResponseSortConcern, Limit: 37,
	}
	if _, err := store.ListCompletedResponses(ctx, query); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	first, err := store.ListCompletedResponses(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed >= 500*time.Millisecond {
		t.Fatalf("warm completed-response query took %s; target is under 500ms", elapsed)
	}
	if len(first.Items) != query.Limit || first.NextCursor == "" {
		t.Fatalf("first page was not bounded: items=%d cursor=%q", len(first.Items), first.NextCursor)
	}
	seen := make(map[string]struct{}, len(first.Items))
	for _, item := range first.Items {
		seen[item.ID] = struct{}{}
		if item.LegalEntityID != entityID || strings.Contains(item.Title, "other") {
			t.Fatalf("completed-response page crossed legal-entity scope: %+v", item)
		}
	}
	query.Cursor = first.NextCursor
	second, err := store.ListCompletedResponses(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range second.Items {
		if _, duplicate := seen[item.ID]; duplicate {
			t.Fatalf("stable keyset pagination returned %s twice", item.ID)
		}
	}
}

func seedCompletedResponseRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, entityID, actorID, formID, prefix string, count int, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_form_distributions(
			id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,title,purpose,
			access_policy,status,deadline,route_expires_at,created_by,version,created_at,updated_at
		)
		SELECT md5($5||':distribution:'||g)::uuid,$1::uuid,$2::uuid,$4::uuid,1,'VENDOR',md5($5||':subject:'||g)::uuid,
		       $5||' scored response '||g,'Review the completed scored response.','DIRECT_MAGIC_LINK','COMPLETED',$7::timestamptz + interval '30 days',$7::timestamptz + interval '7 days',$3::uuid,1,
		       $7::timestamptz - (g % 3000) * interval '1 second',$7::timestamptz - (g % 3000) * interval '1 second'
		FROM generate_series(1,$6) g;

		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,distribution_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			estimated_minutes,deadline,known_facts,presentation,scoring_mode,sections,fields,source_bindings,form_template_id,form_template_version,
			status,created_by,version,created_at,updated_at
		)
		SELECT md5($5||':request:'||g)::uuid,$1::uuid,$2::uuid,md5($5||':distribution:'||g)::uuid,'VENDOR',md5($5||':subject:'||g),
		       $5||' request '||g,'Review the response.','Review the response.','INTERNAL','INTERNAL',5,$7::timestamptz + interval '30 days','{}'::jsonb,
		       '{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,'RISK','[{"id":"general","title":"General"}]'::jsonb,
		       '[{"id":"score","section_id":"general","label":"Score","type":"number","required":true}]'::jsonb,'[]'::jsonb,$4::uuid,1,
		       'SUBMITTED',$3::uuid,1,$7,$7
		FROM generate_series(1,$6) g;

		INSERT INTO capture_response_workspaces(id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at)
		SELECT md5($5||':workspace:'||g)::uuid,$1::uuid,$2::uuid,md5($5||':distribution:'||g)::uuid,'COMPLETED',1,$7,$7
		FROM generate_series(1,$6) g;

		INSERT INTO capture_submissions(id,tenant_id,request_id,submitted_by,channel,answers,submitted_at,created_at,distribution_id)
		SELECT md5($5||':submission:'||g)::uuid,$1::uuid,md5($5||':request:'||g)::uuid,$3::uuid,'INTERNAL',
		       jsonb_build_object('score',g % 101),$7::timestamptz - (g % 3000) * interval '1 second',$7::timestamptz - (g % 3000) * interval '1 second',md5($5||':distribution:'||g)::uuid
		FROM generate_series(1,$6) g;

		INSERT INTO capture_response_revisions(
			id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,achieved_assurance,signoff_summary,
			compliance_score,scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at,
			score_mode,score_direction,raw_score,adverse_score,concern_band,score_state,score_result,score_profile_checksum,score_calculated_at
		)
		SELECT md5($5||':response:'||g)::uuid,$1::uuid,$2::uuid,md5($5||':distribution:'||g)::uuid,md5($5||':workspace:'||g)::uuid,
		       md5($5||':submission:'||g)::uuid,1,'EMAIL_VERIFIED','{}'::jsonb,(g % 101)::numeric,100,'FINAL','[]'::jsonb,'advanced-v1',true,
		       $7::timestamptz - (g % 3000) * interval '1 second','RISK','HIGH_IS_POOR',(g % 101)::numeric,(g % 101)::numeric,
		       CASE WHEN g % 101 >= 75 THEN 'CRITICAL' WHEN g % 101 >= 50 THEN 'HIGH' WHEN g % 101 >= 25 THEN 'MODERATE' ELSE 'LOW' END,
		       'FINAL',jsonb_build_object('profile_version','advanced-v1','evaluator_version','advanced-score-v1'),'load-proof',$7
		FROM generate_series(1,$6) g`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, actorID, formID, prefix, count, now); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCompletedResponseScoreStateIndexPredicateIsClosed(t *testing.T) {
	if got := postgresCompletedResponseScoreStateIndexPredicate([]ResponseScoreState{ResponseScoreFinal, ResponseScoreProvisional}); got != " AND r.score_state IN ('FINAL','PROVISIONAL')" {
		t.Fatalf("score-state index predicate = %q", got)
	}
	if got := postgresCompletedResponseScoreStateIndexPredicate([]ResponseScoreState{ResponseScoreFailed}); got != "" {
		t.Fatalf("failed score state generated an unsafe index predicate: %q", got)
	}
}

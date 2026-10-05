//go:build postgres && postgresintegration

package attention

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEpisodeProjectorDeduplicatesPersistsUnknownAndNoticesWorseningAndClear(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tenantID := mustAttentionID(t)
	entityID := mustAttentionID(t)
	principalID := mustAttentionID(t)
	riskID := mustAttentionID(t)
	memberID := mustAttentionID(t)
	now := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)

	mustAttentionExec(t, ctx, pool, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Attention test')`,
		tenantID, "attention-"+tenantID[len(tenantID)-8:])
	mustAttentionExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'ATT-NG','Attention Nigeria','NG',$3)`,
		entityID, tenantID, now.Add(-24*time.Hour))
	mustAttentionExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Risk owner','ACTIVE',$3)`,
		principalID, tenantID, now.Add(-24*time.Hour))
	mustAttentionExec(t, ctx, pool, `
		INSERT INTO risks(
			id,tenant_id,legal_entity_id,code,name,category,statement,impact,
			owner_principal_id,status,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,'ATT-RISK','Attention risk','Operational',
			'Operational exposure','Material impact',$4::uuid,'ACTIVE',1,$5,$5
		)`, riskID, tenantID, entityID, principalID, now.Add(-time.Hour))

	projector := NewEpisodeProjector(pool)
	insertSource := func(at time.Time, state string, unknown int) workflowruntime.OutboxEvent {
		t.Helper()
		sourceID := mustAttentionID(t)
		eventID := mustAttentionID(t)
		completeness := "COMPLETE"
		if unknown > 0 {
			completeness = "PARTIAL"
		}
		mustAttentionExec(t, ctx, pool, `
			INSERT INTO domain_metric_snapshots(
				id,tenant_id,legal_entity_id,definition_revision,source_revision,
				source_high_water,bucket_start,generated_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,'enterprise-domain-v1','enterprise-domain-v1',
			         '{}'::jsonb,$4,$4)`, sourceID, tenantID, entityID, at)
		value := 0
		condition := "CLEAR"
		if state != "" {
			value = 1
			condition = "ATTENTION"
			mustAttentionExec(t, ctx, pool, `
				INSERT INTO domain_metric_snapshot_memberships(
					source_id,metric_id,definition_revision,member_id,target_type,target_id,target_title,state
				) VALUES(
					$1::uuid,'indicator_breaches','enterprise-domain-v1',$2::uuid,'RISK',$3::uuid,'Retained title',$4
				)`, sourceID, memberID, riskID, state)
		}
		for _, metricID := range []string{
			"risks_outside_appetite", "indicator_breaches", "assurance_failures", "losses_without_issue",
		} {
			metricValue, metricUnknown, metricPopulation, metricCompleteness, metricCondition := 0, 0, 0, "COMPLETE", "CLEAR"
			if metricID == "indicator_breaches" {
				metricValue, metricUnknown, metricPopulation = value, unknown, 1
				metricCompleteness, metricCondition = completeness, condition
			}
			mustAttentionExec(t, ctx, pool, `
				INSERT INTO metric_observations(
					tenant_id,legal_entity_id,metric_id,definition_revision,source_kind,source_id,
					source_revision,source_high_water,generated_at,period_start,period_end,posture_as_of,
					value,condition,freshness,completeness,population,excluded,unknown
				) VALUES(
					$1::uuid,$2::uuid,$3,'enterprise-domain-v1','DOMAIN_SNAPSHOT',$4::uuid,
					'enterprise-domain-v1','{}'::jsonb,$5,$5,$5,$5,
					$6,$7,'CURRENT',$8,$9,0,$10
				)`,
				tenantID, entityID, metricID, sourceID, at,
				metricValue, metricCondition, metricCompleteness, metricPopulation, metricUnknown)
		}
		payload, err := json.Marshal(SourceEvent{
			SourceID: sourceID, LegalEntityID: entityID,
			DefinitionRevision: "enterprise-domain-v1", SourceRevision: "enterprise-domain-v1",
		})
		if err != nil {
			t.Fatal(err)
		}
		mustAttentionExec(t, ctx, pool, `
			INSERT INTO outbox_events(
				id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
			) VALUES($1::uuid,$2::uuid,$3,$4::uuid,$5,$6::jsonb,$7,$7,$7)`,
			eventID, tenantID, SourceAggregateType, sourceID, SourceEventType, payload, at)
		return workflowruntime.OutboxEvent{
			ID: eventID, TenantID: tenantID, AggregateType: SourceAggregateType, AggregateID: sourceID,
			EventType: SourceEventType, Payload: payload, OccurredAt: at,
		}
	}

	first := insertSource(now, "HIGH", 0)
	if err := projector.Publish(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := projector.Publish(ctx, first); err != nil {
		t.Fatal(err)
	}
	assertAttentionState(t, ctx, pool, tenantID, entityID, "OPEN", "HIGH", 1, 1)

	partial := insertSource(now.Add(5*time.Minute), "", 1)
	if err := projector.Publish(ctx, partial); err != nil {
		t.Fatal(err)
	}
	assertAttentionState(t, ctx, pool, tenantID, entityID, "OPEN", "HIGH", 1, 1)

	persistent := insertSource(now.Add(10*time.Minute), "HIGH", 0)
	if err := projector.Publish(ctx, persistent); err != nil {
		t.Fatal(err)
	}
	assertAttentionState(t, ctx, pool, tenantID, entityID, "OPEN", "HIGH", 1, 1)

	worsened := insertSource(now.Add(15*time.Minute), "CRITICAL", 0)
	if err := projector.Publish(ctx, worsened); err != nil {
		t.Fatal(err)
	}
	assertAttentionState(t, ctx, pool, tenantID, entityID, "OPEN", "CRITICAL", 2, 2)

	cleared := insertSource(now.Add(20*time.Minute), "", 0)
	if err := projector.Publish(ctx, cleared); err != nil {
		t.Fatal(err)
	}
	assertAttentionState(t, ctx, pool, tenantID, entityID, "CLEARED", "CRITICAL", 3, 3)

	rows, err := pool.Query(ctx, `
		SELECT event_type,payload->>'principal_id',payload->>'condition_state',payload->>'notice_sequence'
		FROM outbox_events
		WHERE tenant_id=$1::uuid AND aggregate_type=$2
		ORDER BY occurred_at,id`, tenantID, EpisodeAggregateType)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var eventType, recipient, state, sequence string
		if err := rows.Scan(&eventType, &recipient, &state, &sequence); err != nil {
			t.Fatal(err)
		}
		if recipient != principalID {
			t.Fatalf("recipient=%q want %q", recipient, principalID)
		}
		got = append(got, eventType+":"+state+":"+sequence)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		EventEpisodeOpened + ":HIGH:1",
		EventEpisodeWorsened + ":CRITICAL:2",
		EventEpisodeCleared + ":CRITICAL:3",
	}
	if len(got) != len(want) {
		t.Fatalf("episode intents=%v want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("episode intents=%v want %v", got, want)
		}
	}
}

func assertAttentionState(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID, entityID, state, conditionState string,
	sequence, intentCount int,
) {
	t.Helper()
	var gotState, gotCondition string
	var gotSequence, gotIntents int
	if err := pool.QueryRow(ctx, `
		SELECT state,last_condition_state,notice_sequence
		FROM attention_episodes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		ORDER BY created_at DESC,id DESC
		LIMIT 1`, tenantID, entityID).Scan(&gotState, &gotCondition, &gotSequence); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM outbox_events
		WHERE tenant_id=$1::uuid AND aggregate_type=$2`,
		tenantID, EpisodeAggregateType).Scan(&gotIntents); err != nil {
		t.Fatal(err)
	}
	if gotState != state || gotCondition != conditionState || gotSequence != sequence || gotIntents != intentCount {
		t.Fatalf("episode=(%s,%s,%d) intents=%d want=(%s,%s,%d) intents=%d",
			gotState, gotCondition, gotSequence, gotIntents, state, conditionState, sequence, intentCount)
	}
}

func mustAttentionExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func mustAttentionID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

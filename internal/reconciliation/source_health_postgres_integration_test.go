//go:build postgres && postgresintegration

package reconciliation

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSourceHealthPostgresReplayAndRecovery(t *testing.T) {
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

	const tenantID = "83333333-3333-7333-8333-333333333331"
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-replay-test','Source Replay Test')`, tenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	evidenceRepo := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepo, nil)
	source, err := evidenceService.CreateSource(ctx, evidence.CreateSourceInput{
		TenantID:                 "source-replay-test",
		Code:                     "CORE-SOURCE",
		Name:                     "Core governed source",
		Type:                     evidence.SourceSystem,
		AuthorityClass:           "SYSTEM_OF_RECORD",
		ExpectedFreshnessMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE evidence_sources SET health='CURRENT',last_observed_at=$2,last_success_at=$2 WHERE id=$1::uuid`, source.ID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	continuityRepo := continuity.NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepo)
	program, err := continuityService.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID:             "source-replay-test",
		Code:                 "SOURCE-REPLAY",
		Name:                 "Source replay program",
		Type:                 "CONTINUOUS",
		OwningFunction:       "Compliance",
		OwnerPrincipalID:     "",
		AuthorityPrincipalID: "",
		EffectiveFrom:        now.Add(-time.Hour),
		ActorID:              "",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = continuityService.AddRequirement(ctx, continuity.AddRequirementInput{
		TenantID:        "source-replay-test",
		ProgramID:       program.Program.ID,
		ExpectedVersion: program.Program.Version,
		SourceID:        source.ID,
		Code:            "REQ-SOURCE",
		Title:           "Maintain governed source",
		Statement:       "The governed source must remain current.",
		Modality:        "MUST",
		Status:          continuity.RequirementApproved,
		EffectiveFrom:   now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	projection := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "source-replay-projection"}
	drainProjection(t, ctx, projection, now)
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET published_at=$2 WHERE tenant_id=$1::uuid AND published_at IS NULL`, tenantID, now); err != nil {
		t.Fatal(err)
	}

	autonomyService := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	runtimeRepo := workflowruntime.NewPostgresRepository(pool)
	consumer := &SourceHealthConsumer{Inbox: runtimeRepo, Dependencies: continuityRepo, Signals: autonomyService, Programs: continuityService}

	if _, err := evidenceService.RecordSourceObservation(ctx, evidence.SourceObservation{
		TenantID:   "source-replay-test",
		SourceID:   source.ID,
		ObservedAt: now.Add(time.Second),
		Success:    false,
		Detail:     "source degraded",
	}); err != nil {
		t.Fatal(err)
	}
	degradedEvent := deliverOutbox(t, ctx, runtimeRepo, consumer, now.Add(2*time.Second), "SourceHealthChanged")
	if degradedEvent.ID == "" {
		t.Fatal("source degradation event was not delivered")
	}
	drainProjection(t, ctx, projection, now.Add(3*time.Second))

	assertCount(t, ctx, pool, `SELECT count(*) FROM inbox_receipts WHERE tenant_id=$1::uuid AND consumer='source-health-reconciliation' AND event_id=$2`, 1, tenantID, degradedEvent.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM compliance_signals WHERE tenant_id=$1::uuid AND signal_type='SOURCE_DEGRADED' AND subject_id=$2`, 1, tenantID, source.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM drift_assessments WHERE tenant_id=$1::uuid AND subject_type='EVIDENCE_SOURCE' AND subject_id=$2 AND dimension='source_quality' AND state='ACTIVE'`, 1, tenantID, source.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND trigger_type='SOURCE_DEGRADED'`, 1, tenantID, program.Program.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND trigger_type='SOURCE_DEGRADED'`, 1, tenantID)
	assertProgramSourceQuality(t, ctx, pool, tenantID, program.Program.ID, "AT_RISK")

	if err := consumer.Publish(ctx, degradedEvent); err != nil {
		t.Fatalf("duplicate delivery failed: %v", err)
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM compliance_signals WHERE tenant_id=$1::uuid AND signal_type='SOURCE_DEGRADED' AND subject_id=$2`, 1, tenantID, source.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND trigger_type='SOURCE_DEGRADED'`, 1, tenantID, program.Program.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND trigger_type='SOURCE_DEGRADED'`, 1, tenantID)

	if _, err := evidenceService.RecordSourceObservation(ctx, evidence.SourceObservation{
		TenantID:   "source-replay-test",
		SourceID:   source.ID,
		ObservedAt: now.Add(4 * time.Second),
		Success:    true,
		Detail:     "source recovered",
	}); err != nil {
		t.Fatal(err)
	}
	recoveredEvent := deliverOutbox(t, ctx, runtimeRepo, consumer, now.Add(5*time.Second), "SourceHealthChanged")
	if recoveredEvent.ID == "" || recoveredEvent.ID == degradedEvent.ID {
		t.Fatalf("source recovery event was not delivered: %#v", recoveredEvent)
	}
	drainProjection(t, ctx, projection, now.Add(6*time.Second))

	assertCount(t, ctx, pool, `SELECT count(*) FROM drift_assessments WHERE tenant_id=$1::uuid AND subject_type='EVIDENCE_SOURCE' AND subject_id=$2 AND dimension='source_quality' AND state='ACTIVE'`, 0, tenantID, source.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM drift_assessments WHERE tenant_id=$1::uuid AND subject_type='EVIDENCE_SOURCE' AND subject_id=$2 AND dimension='source_quality' AND state='RESOLVED'`, 1, tenantID, source.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND trigger_type='SOURCE_RECOVERED'`, 1, tenantID, program.Program.ID)
	assertCount(t, ctx, pool, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND trigger_type='SOURCE_DEGRADED'`, 1, tenantID)
	assertProgramSourceQuality(t, ctx, pool, tenantID, program.Program.ID, "CURRENT")
}

func deliverOutbox(t *testing.T, ctx context.Context, repo *workflowruntime.PostgresRepository, consumer *SourceHealthConsumer, at time.Time, captureType string) workflowruntime.OutboxEvent {
	t.Helper()
	var captured workflowruntime.OutboxEvent
	for iteration := 0; iteration < 20; iteration++ {
		events, err := repo.ClaimOutbox(ctx, "source-replay-worker", at.Add(time.Duration(iteration)*time.Millisecond), time.Minute, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) == 0 {
			break
		}
		for _, event := range events {
			if event.EventType == captureType {
				captured = event
			}
			if err := consumer.Publish(ctx, event); err != nil {
				t.Fatalf("publish %s: %v", event.EventType, err)
			}
			if err := repo.MarkPublished(ctx, event, at); err != nil {
				t.Fatalf("mark %s published: %v", event.EventType, err)
			}
		}
	}
	return captured
}

func drainProjection(t *testing.T, ctx context.Context, maintainer *continuity.ProjectionMaintainer, at time.Time) {
	t.Helper()
	for iteration := 0; iteration < 20; iteration++ {
		count, err := maintainer.Maintain(ctx, at.Add(time.Duration(iteration)*time.Millisecond), 100)
		if err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			return
		}
	}
	t.Fatal("projection queue did not drain")
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %s", got, want, query)
	}
}

func assertProgramSourceQuality(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, programID, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(ctx, `SELECT dimensions->>'source_quality' FROM program_state_snapshots WHERE tenant_id=$1::uuid AND program_id=$2::uuid ORDER BY generated_at DESC,id DESC LIMIT 1`, tenantID, programID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("source_quality = %q, want %q", got, want)
	}
}

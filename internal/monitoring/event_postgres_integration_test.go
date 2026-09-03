//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConcurrentReplacementApprovalBindsCurrentAndRecordsRetirement(t *testing.T) {
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
		tenantID  = "9c666666-6666-7666-8666-666666666661"
		entityID  = "9c666666-6666-7666-8666-666666666662"
		programID = "9c666666-6666-7666-8666-666666666663"
		currentID = "9c666666-6666-7666-8666-666666666664"
		firstID   = "9c666666-6666-7666-8666-666666666665"
		secondID  = "9c666666-6666-7666-8666-666666666666"
		actorID   = "9c666666-6666-7666-8666-666666666667"
	)
	cleanupMonitoringEventTenant(ctx, pool, tenantID)
	defer cleanupMonitoringEventTenant(context.Background(), pool, tenantID)
	now := time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'monitoring-concurrency-test','Monitoring Concurrency Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'NG','Nigeria','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($4::uuid,$1::uuid,'PERSON','Independent Reviewer');
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from)
		VALUES($3::uuid,$1::uuid,$2::uuid,'MON-CONCURRENT','Monitoring concurrency','COMPLIANCE','ACTIVE','Compliance','NG','{}'::jsonb,$5)`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, programID, actorID, now); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	base := MonitoringCheck{ID: currentID, TenantID: "monitoring-concurrency-test", ProgramID: programID, Code: "ENCRYPTION", Name: "Encryption check", Claim: "Encryption remains enabled.", InputKind: InputSource, BindingID: "9c666666-6666-7666-8666-666666666668", BindingVersion: 1, SourceRules: []SourceRule{{ID: "enabled", Field: "enabled", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}}, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecyclePaused, IsCurrent: true, Version: 3, EffectiveFrom: &now, CreatedAt: now, UpdatedAt: now}}
	first := base
	first.ID = firstID
	first.Lifecycle = Lifecycle{Status: LifecyclePendingApproval, Version: 2, SubmittedBy: actorID, CreatedAt: now, UpdatedAt: now}
	second := first
	second.ID = secondID
	for _, check := range []MonitoringCheck{base, first, second} {
		if _, err = repo.CreateCheckRevision(ctx, check); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []string{firstID, secondID} {
		wg.Add(1)
		go func(candidateID string) {
			defer wg.Done()
			<-start
			_, transitionErr := repo.TransitionCheck(ctx, LifecycleTransition{TenantID: tenantID, ID: candidateID, ExpectedVersion: 2, ExpectedCurrentID: currentID, ExpectedCurrentVersion: 3, To: LifecycleActive, ActorID: actorID, At: now.Add(time.Minute)})
			results <- transitionErr
		}(id)
	}
	close(start)
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for transitionErr := range results {
		if transitionErr == nil {
			successes++
		} else if errors.Is(transitionErr, ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected transition error: %v", transitionErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes/conflicts = %d/%d, want 1/1", successes, conflicts)
	}
	var currentCount, retirementEvents, retirementOutbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_checks WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND code='ENCRYPTION' AND is_current`, tenantID, programID).Scan(&currentCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND aggregate_version=4 AND event_type=$3`, tenantID, currentID, EventMonitoringCheckStateChanged).Scan(&retirementEvents); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3 AND payload->>'version'='4'`, tenantID, currentID, EventMonitoringCheckStateChanged).Scan(&retirementOutbox); err != nil {
		t.Fatal(err)
	}
	if currentCount != 1 || retirementEvents != 1 || retirementOutbox != 1 {
		t.Fatalf("current/event/outbox = %d/%d/%d, want 1/1/1", currentCount, retirementEvents, retirementOutbox)
	}
}

func TestPostgresMonitoringResultRowEventAndOutboxAreAtomicAndIdempotent(t *testing.T) {
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
		tenantID  = "9c444444-4444-7444-8444-444444444441"
		entityID  = "9c444444-4444-7444-8444-444444444442"
		programID = "9c444444-4444-7444-8444-444444444443"
		checkID   = "9c444444-4444-7444-8444-444444444444"
		resultID  = "9c444444-4444-7444-8444-444444444445"
	)
	cleanupMonitoringFailureTrigger(ctx, pool)
	cleanupMonitoringEventTenant(ctx, pool, tenantID)
	defer func() { cleanupMonitoringEventTenant(context.Background(), pool, tenantID) }()
	now := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'monitoring-events-test','Monitoring Events Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'NG','Nigeria','Nigeria');
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from)
		VALUES($3::uuid,$1::uuid,$2::uuid,'MON-EVENT','Monitoring events','COMPLIANCE','ACTIVE','Compliance','NG','{}'::jsonb,$4)`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, programID, now); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	check := MonitoringCheck{ID: checkID, TenantID: "monitoring-events-test", ProgramID: programID, Code: "EVENT", Name: "Event check", Claim: "The result is reconstructable.", InputKind: InputSource, BindingID: "9c444444-4444-7444-8444-444444444446", BindingVersion: 1, SourceRules: []SourceRule{{ID: "healthy", Field: "healthy", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}}, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedAt: now, UpdatedAt: now}}
	if _, err := repo.CreateCheckRevision(ctx, check); err != nil {
		t.Fatal(err)
	}
	score := 80.0
	result := MonitoringResult{ID: resultID, TenantID: "monitoring-events-test", ProgramID: programID, MonitoringCheckID: checkID, MonitoringCheckVersion: 1, InputKind: InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1, Evaluation: Evaluation{Score: &score, Band: RiskCritical, Coverage: 1}, EvaluatedAt: now, EvaluatorVersion: "risk-v1", CreatedAt: now}
	if _, err := repo.AppendResult(ctx, result); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AppendResult(ctx, result); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate result error = %v, want conflict", err)
	}
	var rows, events, outbox int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_results WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, resultID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_events WHERE tenant_id=$1::uuid AND aggregate_type='MONITORING_RESULT' AND aggregate_id=$2::uuid`, tenantID, resultID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='MONITORING_RESULT' AND aggregate_id=$2::uuid`, tenantID, resultID).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || events != 1 || outbox != 1 {
		t.Fatalf("result row/event/outbox = %d/%d/%d, want 1/1/1", rows, events, outbox)
	}
}

func TestPostgresMonitoringRollsBackAuthoritativeRowWhenOutboxInsertFails(t *testing.T) {
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
		tenantID  = "9c555555-5555-7555-8555-555555555551"
		entityID  = "9c555555-5555-7555-8555-555555555552"
		programID = "9c555555-5555-7555-8555-555555555553"
		checkID   = "9c555555-5555-7555-8555-555555555554"
	)
	cleanupMonitoringFailureTrigger(ctx, pool)
	cleanupMonitoringEventTenant(ctx, pool, tenantID)
	defer func() {
		cleanupMonitoringFailureTrigger(context.Background(), pool)
		cleanupMonitoringEventTenant(context.Background(), pool, tenantID)
	}()
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'monitoring-rollback-test','Monitoring Rollback Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'NG','Nigeria','Nigeria');
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from)
		VALUES($3::uuid,$1::uuid,$2::uuid,'MON-ROLLBACK','Monitoring rollback','COMPLIANCE','ACTIVE','Compliance','NG','{}'::jsonb,$4);
		CREATE FUNCTION monitoring_outbox_failure_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_type='MONITORING_CHECK' THEN RAISE EXCEPTION 'forced monitoring outbox failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER monitoring_outbox_failure_test BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION monitoring_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, programID, now); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	check := MonitoringCheck{ID: checkID, TenantID: "monitoring-rollback-test", ProgramID: programID, Code: "ROLLBACK", Name: "Rollback check", Claim: "No partial command commits.", InputKind: InputSource, BindingID: "9c555555-5555-7555-8555-555555555555", BindingVersion: 1, SourceRules: []SourceRule{{ID: "healthy", Field: "healthy", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}}, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedAt: now, UpdatedAt: now}}
	if _, err := repo.CreateCheckRevision(ctx, check); err == nil {
		t.Fatal("expected forced outbox failure")
	}
	var rows, events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_checks WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, checkID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid`, tenantID, checkID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || events != 0 {
		t.Fatalf("failed command left row/event = %d/%d", rows, events)
	}
}

func cleanupMonitoringEventTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}

func cleanupMonitoringFailureTrigger(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DROP TRIGGER IF EXISTS monitoring_outbox_failure_test ON outbox_events`)
	_, _ = pool.Exec(ctx, `DROP FUNCTION IF EXISTS monitoring_outbox_failure_test()`)
}

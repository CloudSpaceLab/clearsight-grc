//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAdverseEpisodeIsScopedVersionedAndOutboxBacked(t *testing.T) {
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

	tenantID := adverseEpisodeTestID(t)
	entityA := adverseEpisodeTestID(t)
	entityB := adverseEpisodeTestID(t)
	programID := adverseEpisodeTestID(t)
	checkID := adverseEpisodeTestID(t)
	bindingID := adverseEpisodeTestID(t)
	episodeID := adverseEpisodeTestID(t)
	tamperEpisodeID := adverseEpisodeTestID(t)
	suffix := tenantID[len(tenantID)-8:]
	tenantSlug := "episode-" + suffix
	entityACode := "EPA-" + suffix
	entityBCode := "EPB-" + suffix
	now := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, tenantSlug, "Episode "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$2::uuid,$3,'Episode Entity A','NG',$6::timestamptz),
			($4::uuid,$2::uuid,$5,'Episode Entity B','GH',$6::timestamptz)
	`, entityA, tenantID, entityACode, entityB, entityBCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(
			id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,
			effective_from,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,'Episode Program','ASSURANCE','ACTIVE','Risk','{}'::jsonb,
		       $5::timestamptz,$5::timestamptz,$5::timestamptz,1)
	`, programID, tenantID, entityA, "EPP-"+suffix, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	effectiveFrom := now.Add(-time.Hour)
	check, err := repo.CreateCheckRevision(ctx, MonitoringCheck{
		ID: checkID, TenantID: tenantSlug, ProgramID: programID,
		Code: "KRI-" + suffix, Name: "Critical failure rate", Claim: "Critical failures remain within tolerance.",
		InputKind: InputSource, BindingID: bindingID, BindingVersion: 1,
		SourceRules: []SourceRule{{ID: "failure", Field: "state", Operator: OperatorEquals, Expected: "healthy", RiskPoints: 100}},
		Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 0.9,
		FailureAction: FailureRecommendMatter,
		Lifecycle: Lifecycle{
			Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &effectiveFrom, Version: 1,
			CreatedAt: effectiveFrom, UpdatedAt: effectiveFrom,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	score := 60.0
	first, err := repo.AppendResult(ctx, MonitoringResult{
		ID: adverseEpisodeTestID(t), TenantID: tenantSlug, ProgramID: programID,
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation: Evaluation{Score: &score, Band: RiskHigh, Coverage: 1},
		EvaluatedAt: now, EvaluatorVersion: "risk-v1", CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	observation := AdverseEpisodeObservation{
		TenantID: tenantSlug, LegalEntityID: entityACode, ProgramID: programID,
		Check: check, Result: first,
	}
	opened, change, err := repo.OpenOrUpdateAdverseEpisode(ctx, observation, episodeID, now)
	if err != nil || change != AdverseEpisodeOpened {
		t.Fatalf("open episode=%#v change=%s err=%v", opened, change, err)
	}
	if opened.TenantID != tenantID || opened.LegalEntityID != entityA || opened.RecordVersion != 1 {
		t.Fatalf("opened episode scope/version=%#v", opened)
	}

	replayed, change, err := repo.OpenOrUpdateAdverseEpisode(ctx, observation, adverseEpisodeTestID(t), now.Add(time.Second))
	if err != nil || change != AdverseEpisodeNoChange || replayed.ID != opened.ID || replayed.RecordVersion != 1 {
		t.Fatalf("replay episode=%#v change=%s err=%v", replayed, change, err)
	}

	score = 90
	secondAt := now.Add(time.Minute)
	second, err := repo.AppendResult(ctx, MonitoringResult{
		ID: adverseEpisodeTestID(t), TenantID: tenantSlug, ProgramID: programID,
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputSource, InputReferenceID: "receipt-2", InputReferenceVersion: 2,
		Evaluation: Evaluation{Score: &score, Band: RiskCritical, Coverage: 1},
		EvaluatedAt: secondAt, EvaluatorVersion: "risk-v1", CreatedAt: secondAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	observation.Result = second
	worsened, change, err := repo.OpenOrUpdateAdverseEpisode(ctx, observation, adverseEpisodeTestID(t), secondAt)
	if err != nil || change != AdverseEpisodeWorsened || worsened.ID != opened.ID || worsened.RecordVersion != 2 {
		t.Fatalf("worsened episode=%#v change=%s err=%v", worsened, change, err)
	}

	score = 5
	clearAt := now.Add(2 * time.Minute)
	clear, err := repo.AppendResult(ctx, MonitoringResult{
		ID: adverseEpisodeTestID(t), TenantID: tenantSlug, ProgramID: programID,
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputSource, InputReferenceID: "receipt-3", InputReferenceVersion: 3,
		Evaluation: Evaluation{Score: &score, Band: RiskLow, Coverage: 1},
		EvaluatedAt: clearAt, EvaluatorVersion: "risk-v1", CreatedAt: clearAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	observation.Result = clear
	closed, change, err := repo.CloseAdverseEpisode(ctx, observation, clearAt)
	if err != nil || change != AdverseEpisodeCleared || closed == nil || closed.State != AdverseEpisodeClosed || closed.RecordVersion != 3 {
		t.Fatalf("closed episode=%#v change=%s err=%v", closed, change, err)
	}
	if _, err := repo.OpenAdverseEpisode(ctx, tenantSlug, entityACode, check.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("closed episode remained open: %v", err)
	}
	if _, err := repo.OpenAdverseEpisode(ctx, tenantSlug, entityBCode, check.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity episode read error=%v", err)
	}

	var eventCount, outboxCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM monitoring_events
		WHERE tenant_id=$1::uuid AND aggregate_type='MONITORING_ADVERSE_EPISODE' AND aggregate_id=$2::uuid
	`, tenantID, episodeID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM outbox_events
		WHERE tenant_id=$1::uuid AND aggregate_type='MONITORING_ADVERSE_EPISODE' AND aggregate_id=$2::uuid
	`, tenantID, episodeID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 3 || outboxCount != 3 {
		t.Fatalf("episode event/outbox counts=%d/%d want=3/3", eventCount, outboxCount)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO monitoring_adverse_episodes(
			id,tenant_id,legal_entity_id,program_id,monitoring_check_id,state,
			first_result_id,last_result_id,last_check_version,last_band,last_coverage,opened_at,updated_at,record_version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'OPEN',
		       $6::uuid,$6::uuid,1,'HIGH',1,$7::timestamptz,$7::timestamptz,1)
	`, tamperEpisodeID, tenantID, entityB, programID, check.ID, first.ID, now)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-entity episode SQL error=%v want foreign-key rejection", err)
	}
}

func adverseEpisodeTestID(t *testing.T) string {
	t.Helper()
	value, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

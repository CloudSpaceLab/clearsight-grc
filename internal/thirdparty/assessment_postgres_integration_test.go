//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	assessmentTemplateID = "33333333-3333-7333-8333-333333333341"
	assessmentOneID      = "33333333-3333-7333-8333-333333333342"
	assessmentTwoID      = "33333333-3333-7333-8333-333333333343"
	assessmentThreeID    = "33333333-3333-7333-8333-333333333344"
)

func TestPostgresAssessmentStartCommitsOneEpisodeEventOutboxAndJob(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed card processing")
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	record := postgresAssessmentRecord(assessmentOneID, relationship, now)
	repo := NewPostgresRepository(pool)

	created, err := repo.CreateAssessment(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != assessmentOneID || created.Status != AssessmentSetupPending || created.StartedByPrincipalID != thirdPartyPrincipal {
		t.Fatalf("unexpected assessment: %#v", created)
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)
	assertAssessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT", 1)
	assertAssessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT", 1)
	events, err := repo.ListAssessmentEvents(ctx, record.Scope, created.ID, created.Version)
	if err != nil || len(events) != 1 || events[0].AssessmentVersion != 1 || events[0].Payload["status"] != string(AssessmentSetupPending) {
		t.Fatalf("assessment history=%#v err=%v", events, err)
	}

	replay := record
	replay.Assessment.ID = assessmentTwoID
	replay.RelationshipVersion++
	replayed, err := repo.CreateAssessment(ctx, replay)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID {
		t.Fatalf("stable replay created another episode: first=%s replay=%s", created.ID, replayed.ID)
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)
	assertAssessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT", 1)
	assertAssessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT", 1)
}

func TestPostgresAssessmentConcurrentStartDeduplicatesAndScopeFailuresWriteNothing(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Payment tokenization")
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	recordA := postgresAssessmentRecord(assessmentOneID, relationship, now)
	recordB := postgresAssessmentRecord(assessmentTwoID, relationship, now)
	repo := NewPostgresRepository(pool)

	var wg sync.WaitGroup
	results := make(chan Assessment, 2)
	errs := make(chan error, 2)
	for _, record := range []CreateAssessmentRecord{recordA, recordB} {
		wg.Add(1)
		go func(value CreateAssessmentRecord) {
			defer wg.Done()
			created, err := repo.CreateAssessment(ctx, value)
			results <- created
			errs <- err
		}(record)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent start: %v", err)
		}
	}
	var episodeID string
	for value := range results {
		if episodeID == "" {
			episodeID = value.ID
		}
		if value.ID != episodeID {
			t.Fatalf("concurrent starts returned different episodes: %s and %s", episodeID, value.ID)
		}
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)

	beforeEvents := assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT")
	beforeOutbox := assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT")
	outside := recordA
	outside.LegalEntityID = thirdPartyEntityB
	outside.Assessment.LegalEntityID = thirdPartyEntityB
	outside.Assessment.StableEpisodeKey = assessmentEpisodeKey(outside.Scope, relationship.Relationship.ID, AssessmentReviewOnboarding)
	if _, err := repo.CreateAssessment(ctx, outside); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity start returned %v", err)
	}
	if assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT") != beforeEvents || assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT") != beforeOutbox {
		t.Fatal("failed scoped start wrote event or outbox state")
	}

	staleRelationship := seedAssessmentRelationship(t, pool, "Customer notification delivery")
	stale := postgresAssessmentRecord(assessmentThreeID, staleRelationship, now)
	stale.RelationshipVersion++
	if _, err := repo.CreateAssessment(ctx, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale relationship start returned %v", err)
	}
	if assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT") != beforeEvents || assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT") != beforeOutbox {
		t.Fatal("stale start wrote event or outbox state")
	}
}

func assessmentPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'third-party-bank','Third Party Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
			($2::uuid,$1::uuid,'ENTITY-A','Entity A','Nigeria'),($3::uuid,$1::uuid,'ENTITY-B','Entity B','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($4::uuid,$1::uuid,'PERSON','Vendor Owner','ACTIVE');
		INSERT INTO monitoring_form_templates(id,tenant_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by)
		VALUES($5::uuid,$1::uuid,'VENDOR-DD','Vendor due diligence','Collect current vendor control information.',
			'{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
			'[{"id":"company","title":"Company details"}]'::jsonb,
			'[{"id":"confirmed","section_id":"company","label":"Confirm the supplied details","type":"yes_no","required":true}]'::jsonb,
			'ACTIVE',true,'2026-08-26T09:00:00Z',3,$4::uuid)`,
		thirdPartyTenantID, thirdPartyEntityA, thirdPartyEntityB, thirdPartyPrincipal, assessmentTemplateID); err != nil {
		t.Fatal(err)
	}
	return pool
}

func seedAssessmentRelationship(t *testing.T, pool *pgxpool.Pool, serviceName string) Aggregate {
	t.Helper()
	service := NewService(NewPostgresRepository(pool))
	input := validPostgresCreateInput(serviceName)
	input.ExternalRef = serviceName
	value, err := service.CreateRelationship(context.Background(), Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal}, input)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func postgresAssessmentRecord(id string, relationship Aggregate, now time.Time) CreateAssessmentRecord {
	scope := Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}
	return CreateAssessmentRecord{
		Scope: scope, RelationshipID: relationship.Relationship.ID, RelationshipVersion: relationship.Relationship.Version,
		Assessment: Assessment{
			ID: id, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: relationship.Relationship.ID,
			ReviewKind: AssessmentReviewOnboarding, StableEpisodeKey: assessmentEpisodeKey(scope, relationship.Relationship.ID, AssessmentReviewOnboarding),
			Status: AssessmentSetupPending, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
			ReviewDueAt: now.Add(14 * 24 * time.Hour), StartedByPrincipalID: thirdPartyPrincipal, StartedAt: now,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
}

func assertAssessmentCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	allowed := map[string]bool{"third_party_assessments": true, "third_party_assessment_jobs": true}
	if !allowed[table] {
		t.Fatalf("unsupported assessment table %q", table)
	}
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count=%d want=%d", table, got, want)
	}
}

func assertAssessmentTypedCount(t *testing.T, pool *pgxpool.Pool, table, aggregateType string, want int) {
	t.Helper()
	if got := assessmentTypedCount(t, pool, table, aggregateType); got != want {
		t.Fatalf("%s %s count=%d want=%d", table, aggregateType, got, want)
	}
}

func assessmentTypedCount(t *testing.T, pool *pgxpool.Pool, table, aggregateType string) int {
	t.Helper()
	if table != "third_party_events" && table != "outbox_events" {
		t.Fatalf("unsupported event table %q", table)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE aggregate_type=$1`, aggregateType).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

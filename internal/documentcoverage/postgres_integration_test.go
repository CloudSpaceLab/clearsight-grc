//go:build postgres && postgresintegration

package documentcoverage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCoverageRoundTripReviewConflictAndQueue(t *testing.T) {
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
		tenantID      = "96666666-6666-7666-8666-666666666661"
		legalEntityID = "96666666-6666-7666-8666-666666666662"
		principalID   = "96666666-6666-7666-8666-666666666663"
		documentID    = "96666666-6666-7666-8666-666666666664"
		assessmentID  = "96666666-6666-7666-8666-666666666665"
		programID     = "96666666-6666-7666-8666-666666666666"
		requirementID = "96666666-6666-7666-8666-666666666667"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	setup := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'coverage-postgres-test','Coverage PostgreSQL Test')`, []any{tenantID}},
		{`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'NG','Nigeria Entity','Nigeria')`, []any{tenantID, legalEntityID}},
		{`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($2::uuid,$1::uuid,'PERSON','Coverage Reviewer')`, []any{tenantID, principalID}},
		{`INSERT INTO document_imports(
			id,tenant_id,legal_entity_id,file_name,media_type,purpose,source_type,size_bytes,sha256,storage_key,
			artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,created_by)
		VALUES($4::uuid,$1::uuid,$2::uuid,'ndpc.pdf','application/pdf','Coverage review','REGULATORY',100,
		       repeat('a',64),'document-imports/test/ndpc.pdf','STORED_UNSCANNED','EXTRACTED','POPPLER_TEXT_V1','REVIEW_REQUIRED','STRUCTURED_OBLIGATION_V1',$3::uuid)`, []any{tenantID, legalEntityID, principalID, documentID}},
	}
	for _, statement := range setup {
		if _, err := tx.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	repository := NewPostgresRepository(pool)
	assessment := Assessment{
		ID: assessmentID, TenantID: "coverage-postgres-test", LegalEntityID: legalEntityID, DocumentID: documentID,
		DocumentSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Status: AssessmentComparing,
		AnalyzerVersion: AnalyzerVersion, MatcherVersion: MatcherVersion, ScoringPolicyVersion: ScoringPolicyVersion,
		ProgramSnapshotHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Metrics:             Metrics{}, Candidates: []Candidate{}, Reviews: []ReviewRecord{}, Suggestions: []Suggestion{}, Limitations: []string{},
		AssessedAt: now, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	started, err := repository.BeginVersion(ctx, assessment)
	if err != nil {
		t.Fatal(err)
	}
	match := Match{
		ID: "match-1", ProgramID: programID, RequirementID: requirementID, Score: .91, Band: MatchStrong,
		Coverage: continuity.RequirementCoverage{RequirementID: requirementID, Applicable: true, ControlImplemented: true, EvidenceSupported: true, Complete: true},
	}
	started.Status = AssessmentReady
	started.Candidates = []Candidate{{
		ID: "candidate-1", Fingerprint: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Eligible: true, Statement: "A controller must retain records.", Classification: ClassificationPartialMatch, Matches: []Match{match},
	}}
	started.Suggestions = []Suggestion{{
		ID: "suggestion-1", CandidateID: "candidate-1", Type: SuggestionLinkRequirement,
		Status: SuggestionProposed, Title: "Review match", ProgramID: programID, RequirementID: requirementID,
	}}
	started.Metrics.EstimatedVerified = CountMetric{Numerator: 1, Denominator: 1}
	completed, err := repository.CompleteVersion(ctx, started, started.Version)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != AssessmentReady || len(completed.Candidates) != 1 || len(completed.Suggestions) != 1 {
		t.Fatalf("coverage did not round trip: %#v", completed)
	}

	completed.Candidates[0].Review = &ReviewDecision{Decision: DecisionAccept, MatchID: "match-1", ReviewerID: principalID, ReviewedAt: now.Add(time.Minute)}
	completed.Candidates[0].Classification = ClassificationVerified
	completed.Reviews = append(completed.Reviews, ReviewRecord{CandidateID: "candidate-1", Decision: DecisionAccept, MatchID: "match-1", ReviewerID: principalID, ReviewedAt: now.Add(time.Minute)})
	completed.Metrics.Verified = CountMetric{Numerator: 1, Denominator: 1}
	completed.UpdatedAt = now.Add(time.Minute)
	reviewed, err := repository.Review(ctx, completed, completed.Version)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Version != 2 || len(reviewed.Reviews) != 1 || reviewed.Metrics.Verified.Numerator != 1 {
		t.Fatalf("review did not round trip: %#v", reviewed)
	}
	if _, err := repository.Review(ctx, completed, completed.Version); err != ErrVersionConflict {
		t.Fatalf("stale review should conflict, got %v", err)
	}

	if err := repository.QueueRecompare(ctx, "coverage-postgres-test", documentID); err != nil {
		t.Fatal(err)
	}
	if err := repository.QueueRecompare(ctx, "coverage-postgres-test", documentID); err != nil {
		t.Fatal(err)
	}
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3 AND published_at IS NULL`, tenantID, documentID, EventCoverageComparisonRequested).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("recompare queue must be idempotent, got %d", queued)
	}
}

//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAssessmentRestartMigrationRollsBackAndReappliesWithHistoryIntact(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed onboarding restart")
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	record := postgresAssessmentRecord(assessmentOneID, relationship, now)
	record.Assessment.SourceTrigger = "RESTART:" + assessmentTwoID
	record.Assessment.StableEpisodeKey = assessmentEpisodeKey(record.Scope, relationship.Relationship.ID, AssessmentReviewOnboarding, record.Assessment.SourceTrigger)
	created, err := NewPostgresRepository(pool).CreateAssessment(ctx, record)
	if err != nil {
		t.Fatal(err)
	}

	down, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("rollback with restart history: %v", err)
	}
	assertRestartAssessmentHistory(t, pool, created.ID, record.Assessment.SourceTrigger)

	up, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply with restart history: %v", err)
	}
	assertRestartAssessmentHistory(t, pool, created.ID, record.Assessment.SourceTrigger)
}

func assertRestartAssessmentHistory(t *testing.T, pool *pgxpool.Pool, assessmentID, wantTrigger string) {
	t.Helper()
	var reviewKind, sourceTrigger string
	if err := pool.QueryRow(context.Background(), `SELECT review_kind,source_trigger FROM third_party_assessments WHERE id=$1::uuid`, assessmentID).Scan(&reviewKind, &sourceTrigger); err != nil {
		t.Fatal(err)
	}
	if reviewKind != string(AssessmentReviewOnboarding) || sourceTrigger != wantTrigger {
		t.Fatalf("restart history changed: kind=%q trigger=%q", reviewKind, sourceTrigger)
	}
}

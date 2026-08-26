//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestPostgresAssessmentProvisionerLeaseFencing(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	relationship := seedAssessmentRelationship(t, pool, "Payment authorization")
	repository := NewPostgresRepository(pool)
	assessment, err := repository.CreateAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now))
	if err != nil {
		t.Fatal(err)
	}
	first, err := repository.ClaimAssessmentSetupJobs(ctx, "worker-a", now, time.Minute, 3, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim = (%#v, %v)", first, err)
	}
	second, err := repository.ClaimAssessmentSetupJobs(ctx, "worker-b", now.Add(2*time.Minute), time.Minute, 3, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("reclaim = (%#v, %v)", second, err)
	}
	if _, err := repository.FailAssessmentSetupJob(ctx, first[0], 3, AssessmentSetupFailureMatter, now.Add(2*time.Minute), now.Add(3*time.Minute)); !errors.Is(err, ErrAssessmentJobLeaseLost) {
		t.Fatalf("stale release error = %v", err)
	}
	if _, err := repository.CompleteAssessmentSetupJob(ctx, first[0], assessment.Version, "33333333-3333-7333-8333-333333333399", now.Add(2*time.Minute)); !errors.Is(err, ErrAssessmentJobLeaseLost) {
		t.Fatalf("stale completion error = %v", err)
	}
	failed, err := repository.FailAssessmentSetupJob(ctx, second[0], 2, AssessmentSetupFailureMatter, now.Add(2*time.Minute), now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != AssessmentJobFailed || failed.Attempts != 2 || failed.LastFailureCode != AssessmentSetupFailureMatter {
		t.Fatalf("terminal job = %#v", failed)
	}
	stored, err := repository.GetAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now).Scope, assessment.ID)
	if err != nil || stored.Status != AssessmentSetupPending {
		t.Fatalf("assessment after terminal failure = (%#v, %v)", stored, err)
	}
}

func TestPostgresAssessmentProvisionerCompletionIsAtomic(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	relationship := seedAssessmentRelationship(t, pool, "Card processing")
	repository := NewPostgresRepository(pool)
	record := postgresAssessmentRecord(assessmentOneID, relationship, now)
	assessment, err := repository.CreateAssessment(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	triggerKey := "thirdparty-assessment:" + assessment.ID
	dueAt := assessment.ReviewDueAt
	matter, err := continuity.NewService(continuity.NewPostgresRepository(pool)).CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: record.TenantID, Type: continuity.MatterVendorReview, Priority: 5,
		Title: "Review vendor due diligence", Summary: "Review the response and supporting evidence.",
		Scope:       json.RawMessage(`{"assessment_id":"` + assessment.ID + `"}`),
		TriggerType: "VENDOR_DUE_DILIGENCE_STARTED", TriggerID: assessment.ID, TriggerKey: triggerKey,
		KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: thirdPartyPrincipal, RequiredAuthority: "REVIEWER", DueAt: &dueAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := repository.ClaimAssessmentSetupJobs(ctx, "worker-a", now, time.Minute, 3, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = (%#v, %v)", claimed, err)
	}
	ready, err := repository.CompleteAssessmentSetupJob(ctx, claimed[0], assessment.Version, matter.Matter.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != AssessmentReadyToSend || ready.ReviewMatterID != matter.Matter.ID || ready.Version != assessment.Version+1 {
		t.Fatalf("ready assessment = %#v", ready)
	}
	jobs, err := repository.ListAssessmentSetupJobs(ctx, record.Scope, assessment.ID)
	if err != nil || len(jobs) != 1 || jobs[0].State != AssessmentJobCompleted {
		t.Fatalf("completed jobs = (%#v, %v)", jobs, err)
	}
	var links, reactions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_assessment_matter_links WHERE assessment_id=$1::uuid AND matter_id=$2::uuid`, assessment.ID, matter.Matter.ID).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_assessment_reactions WHERE assessment_id=$1::uuid AND reaction_kind='SETUP_COMPLETED'`, assessment.ID).Scan(&reactions); err != nil {
		t.Fatal(err)
	}
	if links != 1 || reactions != 1 {
		t.Fatalf("links=%d reactions=%d", links, reactions)
	}
	assertAssessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT", 2)
	assertAssessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT", 2)
}

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestAssessmentProvisionerReusesMatterAfterWorkerCrash(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	recording := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}}
	crashedClaim, err := repository.ClaimAssessmentSetupJobs(context.Background(), "worker-crashed", now, time.Minute, 3, 1)
	if err != nil || len(crashedClaim) != 1 {
		t.Fatalf("crashed worker claim = (%#v, %v)", crashedClaim, err)
	}
	relationship, err := repository.GetRelationship(context.Background(), Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.RelationshipID)
	if err != nil {
		t.Fatal(err)
	}
	triggerKey := "thirdparty-assessment:" + assessment.ID
	if _, err := recording.CreateMatter(context.Background(), assessmentMatterInput(assessment, relationship, triggerKey)); err != nil {
		t.Fatal(err)
	}
	// The worker stops after the canonical Matter commits, leaving its setup job
	// leased. A later worker must reclaim the job and reuse that Matter.
	provisioner := NewAssessmentProvisioner(repository, recording, "worker-b")
	provisioner.Configure(time.Minute, 3, time.Second)

	if processed, err := provisioner.Maintain(context.Background(), now.Add(time.Minute), 10); err != nil || processed != 1 {
		t.Fatalf("retry = (%d, %v)", processed, err)
	}
	ready, err := repository.GetAssessment(context.Background(), Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != AssessmentReadyToSend || ready.ReviewMatterID == "" {
		t.Fatalf("assessment = %#v", ready)
	}
	if recording.created != 1 || recording.createCalls != 2 || recording.lookupCalls != 1 {
		t.Fatalf("matter calls create=%d created=%d lookup=%d", recording.createCalls, recording.created, recording.lookupCalls)
	}
	if got := recording.inputs[0].TriggerKey; got != "thirdparty-assessment:"+assessment.ID {
		t.Fatalf("trigger key = %q", got)
	}
	job := onlyAssessmentSetupJob(t, repository, assessment)
	if job.State != AssessmentJobCompleted || job.LeaseToken != "" || job.LeaseExpiresAt != nil {
		t.Fatalf("completed job = %#v", job)
	}
}

func TestAssessmentProvisionerReleasesMatterFailureForRetry(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	recording := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}, createFailures: 1}
	provisioner := NewAssessmentProvisioner(repository, recording, "worker-a")
	provisioner.Configure(time.Minute, 3, 2*time.Second)

	if processed, err := provisioner.Maintain(context.Background(), now, 10); processed != 1 || err == nil {
		t.Fatalf("first pass = (%d, %v), want one retriable failure", processed, err)
	}
	job := onlyAssessmentSetupJob(t, repository, assessment)
	if job.State != AssessmentJobReady || !job.AvailableAt.Equal(now.Add(time.Second)) || job.LastFailureCode != AssessmentSetupFailureMatter {
		t.Fatalf("released job = %#v", job)
	}
	if processed, err := provisioner.Maintain(context.Background(), now.Add(500*time.Millisecond), 10); err != nil || processed != 0 {
		t.Fatalf("early retry = (%d, %v)", processed, err)
	}
	if processed, err := provisioner.Maintain(context.Background(), now.Add(time.Second), 10); err != nil || processed != 1 {
		t.Fatalf("due retry = (%d, %v)", processed, err)
	}
	ready, err := repository.GetAssessment(context.Background(), Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.ID)
	if err != nil || ready.Status != AssessmentReadyToSend {
		t.Fatalf("ready assessment = (%#v, %v)", ready, err)
	}
	completedJob := onlyAssessmentSetupJob(t, repository, assessment)
	if completedJob.State != AssessmentJobCompleted || completedJob.LastFailureCode != "" {
		t.Fatalf("completed retry retained failure state: %#v", completedJob)
	}
}

func TestAssessmentProvisionerLeaseFencingRejectsStaleWorker(t *testing.T) {
	now := time.Date(2026, 8, 26, 11, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	first, err := repository.ClaimAssessmentSetupJobs(context.Background(), "worker-a", now, time.Minute, 3, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim = (%#v, %v)", first, err)
	}
	second, err := repository.ClaimAssessmentSetupJobs(context.Background(), "worker-b", now.Add(2*time.Minute), time.Minute, 3, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("second claim = (%#v, %v)", second, err)
	}
	if first[0].LeaseToken == second[0].LeaseToken {
		t.Fatal("reclaimed job retained its lease token")
	}
	if _, err := repository.CompleteAssessmentSetupJob(context.Background(), first[0], assessment.Version, "matter-1", now.Add(2*time.Minute)); !errors.Is(err, ErrAssessmentJobLeaseLost) {
		t.Fatalf("stale completion error = %v", err)
	}
	if _, err := repository.FailAssessmentSetupJob(context.Background(), first[0], 3, AssessmentSetupFailureMatter, now.Add(2*time.Minute), now.Add(3*time.Minute)); !errors.Is(err, ErrAssessmentJobLeaseLost) {
		t.Fatalf("stale failure error = %v", err)
	}
}

func TestAssessmentProvisionerCrashAtAttemptLimitBecomesTerminal(t *testing.T) {
	now := time.Date(2026, 8, 26, 11, 30, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	claimed, err := repository.ClaimAssessmentSetupJobs(context.Background(), "worker-crashed", now, time.Minute, 1, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = (%#v, %v)", claimed, err)
	}
	provisioner := NewAssessmentProvisioner(repository, &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}}, "worker-b")
	provisioner.Configure(time.Minute, 1, time.Second)
	if processed, err := provisioner.Maintain(context.Background(), now.Add(time.Minute), 10); err != nil || processed != 0 {
		t.Fatalf("expired final claim = (%d, %v)", processed, err)
	}
	job := onlyAssessmentSetupJob(t, repository, assessment)
	if job.State != AssessmentJobFailed || job.Attempts != 1 || job.LastFailureCode != AssessmentSetupFailureAttemptsExhausted {
		t.Fatalf("terminal crashed job = %#v", job)
	}
}

func TestAssessmentProvisionerTerminalFailureRemainsVisibleAndPending(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	recording := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}, alwaysFail: true}
	provisioner := NewAssessmentProvisioner(repository, recording, "worker-a")
	provisioner.Configure(time.Minute, 2, time.Second)

	for _, at := range []time.Time{now, now.Add(time.Second)} {
		if processed, err := provisioner.Maintain(context.Background(), at, 10); processed != 1 || err == nil {
			t.Fatalf("failure pass at %s = (%d, %v)", at, processed, err)
		}
	}
	job := onlyAssessmentSetupJob(t, repository, assessment)
	if job.State != AssessmentJobFailed || job.Attempts != 2 || job.LastFailureCode != AssessmentSetupFailureMatter {
		t.Fatalf("terminal job = %#v", job)
	}
	stored, err := repository.GetAssessment(context.Background(), Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.ID)
	if err != nil || stored.Status != AssessmentSetupPending || stored.ReviewMatterID != "" {
		t.Fatalf("assessment after terminal failure = (%#v, %v)", stored, err)
	}
	if processed, err := provisioner.Maintain(context.Background(), now.Add(time.Hour), 10); err != nil || processed != 0 {
		t.Fatalf("terminal job was reclaimed = (%d, %v)", processed, err)
	}
}

func TestAssessmentProvisionerCreatesCanonicalVendorReviewMatter(t *testing.T) {
	now := time.Date(2026, 8, 26, 13, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	recording := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}}
	provisioner := NewAssessmentProvisioner(repository, recording, "worker-a")
	provisioner.Configure(time.Minute, 3, time.Second)

	if _, err := provisioner.Maintain(context.Background(), now, 10); err != nil {
		t.Fatal(err)
	}
	if len(recording.inputs) != 1 {
		t.Fatalf("matter inputs = %d", len(recording.inputs))
	}
	input := recording.inputs[0]
	if input.Type != continuity.MatterVendorReview || input.OwnerPrincipalID != "owner-1" || input.RequiredAuthority != "REVIEWER" {
		t.Fatalf("matter routing = %#v", input)
	}
	if input.Title != "Review Acme Payments for card processing" || input.Summary == "" || input.Priority != 5 {
		t.Fatalf("matter content = %#v", input)
	}
	if input.DueAt == nil || !input.DueAt.Equal(assessment.ReviewDueAt) {
		t.Fatalf("matter due date = %#v", input.DueAt)
	}
	var scope map[string]any
	if err := json.Unmarshal(input.Scope, &scope); err != nil {
		t.Fatal(err)
	}
	if scope["assessment_id"] != assessment.ID || scope["relationship_id"] != assessment.RelationshipID {
		t.Fatalf("matter scope = %#v", scope)
	}
}

type recordingAssessmentMatterService struct {
	inputs         []continuity.CreateMatterInput
	matters        map[string]continuity.MatterAggregate
	createCalls    int
	lookupCalls    int
	created        int
	createFailures int
	alwaysFail     bool
}

func (s *recordingAssessmentMatterService) CreateMatter(_ context.Context, input continuity.CreateMatterInput) (continuity.MatterAggregate, error) {
	s.createCalls++
	s.inputs = append(s.inputs, input)
	if s.alwaysFail || s.createFailures > 0 {
		if s.createFailures > 0 {
			s.createFailures--
		}
		return continuity.MatterAggregate{}, errors.New("matter service unavailable")
	}
	if _, ok := s.matters[input.TriggerKey]; ok {
		return continuity.MatterAggregate{}, continuity.ErrDuplicate
	}
	s.created++
	matter := continuity.Matter{ID: "matter-1", TenantID: input.TenantID, Type: input.Type, TriggerKey: input.TriggerKey, OwnerPrincipalID: input.OwnerPrincipalID, DueAt: input.DueAt, Version: 1}
	aggregate := continuity.MatterAggregate{Matter: matter}
	s.matters[input.TriggerKey] = aggregate
	return aggregate, nil
}

func (s *recordingAssessmentMatterService) MatterByTriggerKey(_ context.Context, _ string, triggerKey string) (continuity.MatterAggregate, error) {
	s.lookupCalls++
	value, ok := s.matters[triggerKey]
	if !ok {
		return continuity.MatterAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}

func assessmentProvisionerFixture(t *testing.T, now time.Time) (*MemoryAssessmentRepository, Assessment) {
	t.Helper()
	repository := NewMemoryAssessmentRepository()
	scope := Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}
	_, err := repository.CreateRelationship(context.Background(), CreateRecord{
		Vendor:       Vendor{ID: "vendor-1", TenantID: scope.TenantID, LegalName: "Acme Payments", Status: VendorActive, CreatedAt: now, UpdatedAt: now, Version: 1},
		Relationship: Relationship{ID: "relationship-1", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, VendorID: "vendor-1", ServiceName: "card processing", BusinessOwnerPrincipalID: "owner-1", Criticality: CriticalityCritical, PrivacyRole: PrivacyProcessor, Status: RelationshipProposed, CreatedAt: now, UpdatedAt: now, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	assessment := Assessment{ID: "assessment-1", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: "relationship-1", ReviewKind: AssessmentReviewOnboarding, StableEpisodeKey: "episode-1", Status: AssessmentSetupPending, FormTemplateID: "form-1", FormTemplateVersion: 1, ReviewDueAt: now.Add(7 * 24 * time.Hour), StartedByPrincipalID: "owner-1", StartedAt: now, Version: 1, CreatedAt: now, UpdatedAt: now}
	created, err := repository.CreateAssessment(context.Background(), CreateAssessmentRecord{Scope: scope, RelationshipID: assessment.RelationshipID, RelationshipVersion: 1, Assessment: assessment})
	if err != nil {
		t.Fatal(err)
	}
	return repository, created
}

func onlyAssessmentSetupJob(t *testing.T, repository *MemoryAssessmentRepository, assessment Assessment) AssessmentSetupJob {
	t.Helper()
	jobs, err := repository.ListAssessmentSetupJobs(context.Background(), Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("setup jobs = %#v", jobs)
	}
	return jobs[0]
}

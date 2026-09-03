package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestCollectionDatesClampMonthEnd(t *testing.T) {
	tests := []struct {
		name      string
		submitted time.Time
		months    int
		want      time.Time
	}{
		{name: "January month end", submitted: time.Date(2027, 1, 31, 10, 32, 0, 0, time.UTC), months: 1, want: time.Date(2027, 2, 28, 10, 32, 0, 0, time.UTC)},
		{name: "leap day annual", submitted: time.Date(2028, 2, 29, 10, 32, 0, 0, time.UTC), months: 12, want: time.Date(2029, 2, 28, 10, 32, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := CollectionPolicy{ValidityMonths: tc.months, RenewalWindowDays: 20, ReminderCount: 3}
			expires, opens, reminders := CollectionDates(tc.submitted, policy)
			if !expires.Equal(tc.want) || !opens.Equal(tc.want.AddDate(0, 0, -20)) || len(reminders) != 3 {
				t.Fatalf("dates = %s %s %#v", expires, opens, reminders)
			}
		})
	}
}

func TestCollectionDatesCreateOrderedUniqueRemindersBeforeExpiry(t *testing.T) {
	submitted := time.Date(2027, 1, 10, 10, 32, 0, 0, time.UTC)
	for count := 1; count <= 5; count++ {
		expires, opens, reminders := CollectionDates(submitted, CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: count})
		if len(reminders) != count {
			t.Fatalf("count %d reminders = %#v", count, reminders)
		}
		previous := opens
		for _, reminder := range reminders {
			if !reminder.After(previous) || !reminder.Before(expires) {
				t.Fatalf("count %d reminders are not ordered and unique: %#v", count, reminders)
			}
			previous = reminder
		}
	}
}

func TestCollectionConsumerProjectsSubmissionOnce(t *testing.T) {
	ctx := context.Background()
	submittedAt := time.Date(2027, 1, 31, 10, 32, 0, 0, time.UTC)
	monitoringRepo := NewMemoryRepository()
	check := activeCollectionCheck(submittedAt)
	if _, err := monitoringRepo.CreateCheckRevision(ctx, check); err != nil {
		t.Fatal(err)
	}
	evidenceRepo := evidence.NewMemoryRepository(nil, nil)
	evidenceService := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	request, err := evidenceService.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: "bank-a", SubjectType: "PROGRAM", SubjectID: check.ProgramID, Title: "Vendor security review",
		Purpose: "Collect the current vendor response.", WhyYou: "You are responsible for this vendor response.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: "respondent-1"}, EstimatedMinutes: 5,
		Deadline: submittedAt.Add(24 * time.Hour), Fields: []evidence.Field{{ID: "answer", Label: "Answer", Type: "text", Required: true}},
		Origin: evidence.RequestOrigin{Type: evidence.OriginMonitoringCollection, ID: check.ID, Version: 1}, CreatedBy: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := evidenceRepo.Submit(ctx, evidence.Submission{ID: "submission-1", TenantID: request.TenantID, RequestID: request.ID, SubmittedBy: "respondent-1", Channel: "INTERNAL", Answers: formcontract.TextAnswers(map[string]string{"answer": "Confirmed"}), ExpectedVersion: request.Version, SubmittedAt: submittedAt})
	if err != nil {
		t.Fatal(err)
	}

	inbox := &failFirstRecordInbox{MemoryRepository: workflowruntime.NewMemoryRepository()}
	cycleIDs := []string{"cycle-1", "cycle-2"}
	consumer := &CollectionConsumer{Inbox: inbox, Repository: monitoringRepo, Evidence: evidenceService, Now: func() time.Time { return submittedAt }, NewID: func() (string, error) {
		value := cycleIDs[0]
		cycleIDs = cycleIDs[1:]
		return value, nil
	}}
	payload, _ := json.Marshal(map[string]string{"submission_id": receipt.SubmissionID, "channel": "INTERNAL"})
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "bank-a", AggregateType: "EVIDENCE_REQUEST", AggregateID: request.ID, EventType: "EvidenceResponseSubmitted", Payload: payload, OccurredAt: submittedAt}
	if err := consumer.Publish(ctx, event); err == nil {
		t.Fatal("expected inbox interruption after the idempotent cycle write")
	}
	if err := consumer.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}

	summaries, err := monitoringRepo.ListCollectionSummaries(ctx, "bank-a", check.ProgramID, 10)
	if err != nil || len(summaries) != 1 {
		t.Fatalf("summaries = %#v, err = %v", summaries, err)
	}
	wantExpiry := time.Date(2027, 2, 28, 10, 32, 0, 0, time.UTC)
	got := summaries[0]
	if got.CycleID != "cycle-1" || got.Sequence != 1 || got.LatestSubmissionID != receipt.SubmissionID || got.LatestSubmittedAt == nil || !got.LatestSubmittedAt.Equal(submittedAt) || !got.ExpiresAt.Equal(wantExpiry) || got.Recipient.PrincipalID != "respondent-1" {
		t.Fatalf("summary = %#v", got)
	}
	processed, err := inbox.InboxProcessed(ctx, "bank-a", collectionSubmissionConsumer, event.ID)
	if err != nil || !processed {
		t.Fatalf("inbox processed = %v, err = %v", processed, err)
	}
}

func TestCollectionConsumerCancelsOpenCycleWhenCheckIsRetired(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 2, 1, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	retired := activeCollectionCheck(now)
	retired.Status = LifecycleRetired
	retired.IsCurrent = false
	retired.Version = 3
	if _, err := repo.CreateCheckRevision(ctx, retired); err != nil {
		t.Fatal(err)
	}
	cycle := collectionCycleFixture("cycle-1", "bank-a", retired.ProgramID, retired.ID, 1, now.Add(time.Hour))
	if _, err := repo.UpsertCollectionCycle(ctx, cycle); err != nil {
		t.Fatal(err)
	}
	request := evidence.Request{ID: "request-2", TenantID: "bank-a", SubjectType: "PROGRAM", SubjectID: retired.ProgramID, Origin: evidence.RequestOrigin{Type: evidence.OriginMonitoringCollection, ID: retired.ID, Version: 2}}
	submission := evidence.Submission{ID: "submission-2", TenantID: "bank-a", RequestID: request.ID, SubmittedAt: now}
	inbox := workflowruntime.NewMemoryRepository()
	consumer := &CollectionConsumer{Inbox: inbox, Repository: repo, Evidence: staticCollectionEvidence{request: request, submission: submission}, Now: func() time.Time { return now }}
	payload, _ := json.Marshal(map[string]string{"submission_id": submission.ID})
	event := workflowruntime.OutboxEvent{ID: "event-retired", TenantID: "bank-a", AggregateType: "EVIDENCE_REQUEST", AggregateID: request.ID, EventType: "EvidenceResponseSubmitted", Payload: payload, OccurredAt: now}
	if err := consumer.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.CollectionCycle(ctx, "bank-a", cycle.ID)
	if err != nil || stored.State != CycleCancelled || stored.NextActionAt != nil {
		t.Fatalf("retired cycle = %#v, err = %v", stored, err)
	}
}

type staticCollectionEvidence struct {
	request    evidence.Request
	submission evidence.Submission
}

func (r staticCollectionEvidence) GetRequest(_ context.Context, tenant, id string) (evidence.Request, error) {
	if r.request.TenantID != tenant || r.request.ID != id {
		return evidence.Request{}, evidence.ErrNotFound
	}
	return r.request, nil
}

func (r staticCollectionEvidence) GetSubmission(_ context.Context, tenant, id string) (evidence.Submission, error) {
	if r.submission.TenantID != tenant || r.submission.ID != id {
		return evidence.Submission{}, evidence.ErrNotFound
	}
	return r.submission, nil
}

type failFirstRecordInbox struct {
	*workflowruntime.MemoryRepository
	failed bool
}

func (i *failFirstRecordInbox) RecordInbox(ctx context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {
	if !i.failed {
		i.failed = true
		return false, errors.New("simulated inbox interruption")
	}
	return i.MemoryRepository.RecordInbox(ctx, tenant, consumer, eventID, at)
}

func activeCollectionCheck(at time.Time) MonitoringCheck {
	return MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "VENDOR-SECURITY", Name: "Vendor security review", Claim: "The vendor security response is current.",
		InputKind: InputForm, FormTemplateID: "form-1", FormTemplateVersion: 1, CollectionPolicy: &CollectionPolicy{ValidityMonths: 1, RenewalWindowDays: 20, ReminderCount: 3},
		Thresholds: DefaultThresholds(), FreshnessMinutes: 43200, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 2, CreatedAt: at.Add(-time.Hour), UpdatedAt: at.Add(-time.Hour)},
	}
}

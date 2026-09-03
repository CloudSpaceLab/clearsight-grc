package monitoring

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestCollectionMaintainerReusesSuccessorAfterInterruption(t *testing.T) {
	fixture := newRenewalFixture(t, 3)
	fixture.maintainer.afterRequest = func() error { return errors.New("simulated interruption") }
	if _, err := fixture.maintainer.Maintain(context.Background(), fixture.openAt, 10); err != nil {
		t.Fatalf("interrupted renewal should be durably retried: %v", err)
	}
	fixture.maintainer.afterRequest = nil
	if _, err := fixture.maintainer.Maintain(context.Background(), fixture.openAt.Add(time.Minute), 10); err != nil {
		t.Fatal(err)
	}
	requests, err := fixture.evidence.ListRequests(context.Background(), "bank-a", 20)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, request := range requests {
		if request.Origin.ID == fixture.check.ID && request.Origin.Version == 2 {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("successor count = %d", count)
	}
	cycle, err := fixture.repo.CollectionCycle(context.Background(), "bank-a", fixture.cycle.ID)
	if err != nil || cycle.State != CycleAwaitingResponse || cycle.CurrentRequestID == fixture.predecessor.ID || cycle.DeliveryState != DeliveryAssigned {
		t.Fatalf("renewed cycle = %#v, err = %v", cycle, err)
	}
}

func TestCollectionMaintainerSendsConfiguredReminderCount(t *testing.T) {
	for _, count := range []int{1, 3, 5} {
		t.Run(fmt.Sprintf("%d reminders", count), func(t *testing.T) {
			fixture := newRenewalFixture(t, count)
			if _, err := fixture.maintainer.Maintain(context.Background(), fixture.openAt, 10); err != nil {
				t.Fatal(err)
			}
			_, _, reminders := CollectionDates(fixture.submission.SubmittedAt, fixture.cycle.Policy)
			for _, at := range reminders {
				if _, err := fixture.maintainer.Maintain(context.Background(), at, 10); err != nil {
					t.Fatal(err)
				}
			}
			if fixture.dispatcher.reminders != count {
				t.Fatalf("reminders sent = %d, want %d", fixture.dispatcher.reminders, count)
			}
			cycle, err := fixture.repo.CollectionCycle(context.Background(), "bank-a", fixture.cycle.ID)
			if err != nil || cycle.RemindersSent != count || cycle.NextActionAt == nil || !cycle.NextActionAt.Equal(cycle.ExpiresAt) {
				t.Fatalf("cycle after reminders = %#v, err = %v", cycle, err)
			}
		})
	}
}

func TestCollectionMaintainerBlocksExternalRouteWithoutAdapter(t *testing.T) {
	fixture := newRenewalFixture(t, 3, RecipientRoute{Type: RouteExternalContact, ContactRef: "contact-opaque-1", SafeHint: "Vendor security contact"})
	fixture.maintainer.Dispatcher = &CanonicalCollectionDispatcher{Requests: fixture.evidence}
	if _, err := fixture.maintainer.Maintain(context.Background(), fixture.openAt, 10); err != nil {
		t.Fatal(err)
	}
	cycle, err := fixture.repo.CollectionCycle(context.Background(), "bank-a", fixture.cycle.ID)
	if err != nil || cycle.State != CycleBlocked || cycle.DeliveryState != DeliveryBlocked || cycle.NextActionAt != nil || cycle.SafeError == "" {
		t.Fatalf("blocked external cycle = %#v, err = %v", cycle, err)
	}
}

func TestCollectionMaintainerBoundsTerminalFailure(t *testing.T) {
	fixture := newRenewalFixture(t, 3)
	fixture.maintainer.Dispatcher = failingCollectionDispatcher{}
	for attempt := 0; attempt < 3; attempt++ {
		at := fixture.openAt.Add(time.Duration(attempt) * time.Minute)
		if _, err := fixture.maintainer.Maintain(context.Background(), at, 10); err != nil {
			t.Fatal(err)
		}
	}
	cycle, err := fixture.repo.CollectionCycle(context.Background(), "bank-a", fixture.cycle.ID)
	if err != nil || cycle.State != CycleFailed || cycle.DeliveryState != DeliveryFailed || cycle.Attempts != 3 || cycle.NextActionAt != nil {
		t.Fatalf("terminal cycle = %#v, err = %v", cycle, err)
	}
}

func TestCanonicalCollectionDispatcherUsesConfirmedExternalReceipt(t *testing.T) {
	external := &recordingCollectionDispatcher{}
	dispatcher := &CanonicalCollectionDispatcher{External: external}
	route := RecipientRoute{Type: RouteExternalContact, ContactRef: "contact-opaque-1", SafeHint: "Vendor security contact"}
	if err := dispatcher.ValidateRoute(context.Background(), "bank-a", route); err != nil {
		t.Fatal(err)
	}
	receipt, err := dispatcher.DispatchRequest(context.Background(), evidence.Request{ID: "request-2", TenantID: "bank-a"}, route)
	if err != nil || receipt.State != DeliveryDelivered || receipt.Reference == "" {
		t.Fatalf("external receipt = %#v, err = %v", receipt, err)
	}
}

func TestCollectionMaintainerStopsAfterRecipientReassignmentIsRequired(t *testing.T) {
	fixture := newRenewalFixture(t, 3)
	ctx := context.Background()
	if _, err := fixture.maintainer.Maintain(ctx, fixture.openAt, 10); err != nil {
		t.Fatal(err)
	}
	cycle, err := fixture.repo.CollectionCycle(ctx, "bank-a", fixture.cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	request, err := fixture.evidence.GetRequest(ctx, "bank-a", cycle.CurrentRequestID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.evidence.DeclareWrongRecipient(ctx, evidence.DeclareWrongRecipientInput{TenantID: "bank-a", LegalEntityID: "entity-a", RequestID: request.ID, ActorPrincipalID: "respondent-1", Reason: "The vendor owner changed.", ExpectedVersion: request.Version}); err != nil {
		t.Fatal(err)
	}
	_, _, reminders := CollectionDates(fixture.submission.SubmittedAt, fixture.cycle.Policy)
	if _, err := fixture.maintainer.Maintain(ctx, reminders[0], 10); err != nil {
		t.Fatal(err)
	}
	cycle, err = fixture.repo.CollectionCycle(ctx, "bank-a", fixture.cycle.ID)
	if err != nil || cycle.State != CycleBlocked || fixture.dispatcher.reminders != 0 {
		t.Fatalf("reassignment cycle = %#v, reminders = %d, err = %v", cycle, fixture.dispatcher.reminders, err)
	}
}

type renewalFixture struct {
	repo        *MemoryRepository
	evidence    *evidence.Service
	dispatcher  *recordingCollectionDispatcher
	maintainer  *CollectionMaintainer
	check       MonitoringCheck
	predecessor evidence.Request
	submission  evidence.Submission
	cycle       CollectionCycle
	openAt      time.Time
}

func newRenewalFixture(t *testing.T, reminders int, routes ...RecipientRoute) renewalFixture {
	t.Helper()
	ctx := context.Background()
	submittedAt := time.Date(2026, 9, 3, 10, 32, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "VENDOR-SECURITY", Name: "Vendor security review", Purpose: "Collect the current vendor security response.",
		Fields:    []TemplateField{{ID: "answer", Label: "Security controls confirmed", Type: "text", Required: true}},
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 2, CreatedAt: submittedAt.Add(-time.Hour), UpdatedAt: submittedAt.Add(-time.Hour)},
	}
	if _, err := repo.CreateFormRevision(ctx, form); err != nil {
		t.Fatal(err)
	}
	check := activeCollectionCheck(submittedAt)
	check.FormTemplateVersion = form.Version
	check.CollectionPolicy = &CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: reminders}
	if _, err := repo.CreateCheckRevision(ctx, check); err != nil {
		t.Fatal(err)
	}
	evidenceRepo := &renewalEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil)}
	evidenceService := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	predecessor, err := evidenceService.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", SubjectType: "PROGRAM", SubjectID: check.ProgramID, Title: form.Name, Purpose: form.Purpose,
		WhyYou: "You are responsible for this vendor response.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: "respondent-1"}, EstimatedMinutes: 5,
		Deadline: submittedAt.Add(24 * time.Hour), Fields: []evidence.Field{{ID: "answer", Label: "Security controls confirmed", Type: "text", Required: true}},
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, Origin: evidence.RequestOrigin{Type: evidence.OriginMonitoringCollection, ID: check.ID, Version: 1}, CreatedBy: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := evidenceRepo.Submit(ctx, evidence.Submission{ID: "submission-1", TenantID: "bank-a", RequestID: predecessor.ID, SubmittedBy: "respondent-1", Channel: "INTERNAL", Answers: formcontract.TextAnswers(map[string]string{"answer": "Confirmed"}), ExpectedVersion: predecessor.Version, SubmittedAt: submittedAt})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := evidenceService.GetSubmission(ctx, "bank-a", receipt.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	expires, opens, _ := CollectionDates(submittedAt, *check.CollectionPolicy)
	route := RecipientRoute{Type: RouteInternalPrincipal, PrincipalID: "respondent-1"}
	delivery := DeliveryAssigned
	if len(routes) > 0 {
		route = routes[0]
		delivery = DeliveryDelivered
	}
	cycle := CollectionCycle{
		ID: "cycle-1", TenantID: "bank-a", ProgramID: check.ProgramID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version, Sequence: 1,
		Policy: *check.CollectionPolicy, CurrentRequestID: predecessor.ID, LatestSubmissionID: submission.ID, LatestSubmittedAt: &submittedAt,
		ExpiresAt: expires, RenewalOpensAt: opens, NextActionAt: &opens, Recipient: route,
		DeliveryState: delivery, State: CycleScheduled, CreatedAt: submittedAt, UpdatedAt: submittedAt,
	}
	if _, err := repo.UpsertCollectionCycle(ctx, cycle); err != nil {
		t.Fatal(err)
	}
	dispatcher := &recordingCollectionDispatcher{}
	maintainer := &CollectionMaintainer{Repository: repo, Requests: evidenceService, Dispatcher: dispatcher, WorkerID: "worker-1", Lease: time.Minute, MaxAttempts: 3}
	return renewalFixture{repo: repo, evidence: evidenceService, dispatcher: dispatcher, maintainer: maintainer, check: check, predecessor: predecessor, submission: submission, cycle: cycle, openAt: opens}
}

type recordingCollectionDispatcher struct {
	requests  int
	reminders int
}

type renewalEvidenceRepository struct {
	*evidence.MemoryRepository
}

func (r *renewalEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	if tenant != "bank-a" || subjectType != "PROGRAM" || subjectID != "program-1" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: "entity-a", SubjectType: subjectType, SubjectID: subjectID}, nil
}

type failingCollectionDispatcher struct{}

func (failingCollectionDispatcher) ValidateRoute(context.Context, string, RecipientRoute) error {
	return errors.New("temporary delivery failure")
}

func (failingCollectionDispatcher) DispatchRequest(context.Context, evidence.Request, RecipientRoute) (DeliveryReceipt, error) {
	return DeliveryReceipt{}, errors.New("temporary delivery failure")
}

func (failingCollectionDispatcher) DispatchReminder(context.Context, evidence.Request, RecipientRoute, int) (DeliveryReceipt, error) {
	return DeliveryReceipt{}, errors.New("temporary delivery failure")
}

func (d *recordingCollectionDispatcher) ValidateRoute(_ context.Context, _ string, route RecipientRoute) error {
	return validateRecipientRoute(route)
}

func (d *recordingCollectionDispatcher) DispatchRequest(_ context.Context, _ evidence.Request, route RecipientRoute) (DeliveryReceipt, error) {
	d.requests++
	if route.Type == RouteExternalContact {
		return DeliveryReceipt{State: DeliveryDelivered, Reference: "provider-receipt-1"}, nil
	}
	return DeliveryReceipt{State: DeliveryAssigned, Reference: "assignment-confirmed"}, nil
}

func (d *recordingCollectionDispatcher) DispatchReminder(_ context.Context, _ evidence.Request, route RecipientRoute, _ int) (DeliveryReceipt, error) {
	d.reminders++
	if route.Type == RouteExternalContact {
		return DeliveryReceipt{State: DeliveryDelivered, Reference: fmt.Sprintf("provider-reminder-%d", d.reminders)}, nil
	}
	return DeliveryReceipt{State: DeliveryAssigned, Reference: fmt.Sprintf("reminder-%d", d.reminders)}, nil
}

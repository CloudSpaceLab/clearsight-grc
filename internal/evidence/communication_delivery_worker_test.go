package evidence

import (
	"context"
	"testing"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type communicationDeliveryRepositoryStub struct {
	bundle       communicationDeliveryBundle
	prior        CommunicationDeliveryAttempt
	priorFound   bool
	records      int
	lastFailure  string
	lastReceipt  InvitationDeliveryReceipt
}

func (stub *communicationDeliveryRepositoryStub) LoadCommunicationDelivery(context.Context, string, string) (communicationDeliveryBundle, error) {
	return stub.bundle, nil
}

func (stub *communicationDeliveryRepositoryStub) GetCommunicationDeliveryAttempt(context.Context, string, string, CommunicationAction) (CommunicationDeliveryAttempt, bool, error) {
	return stub.prior, stub.priorFound, nil
}

func (stub *communicationDeliveryRepositoryStub) RecordCommunicationDeliveryAttempt(_ context.Context, _ workflowruntime.OutboxEvent, _ communicationDeliveryBundle, _ communicationDeliveryRecipient, _ CommunicationTemplate, receipt InvitationDeliveryReceipt, failure string, _ time.Time) error {
	stub.records++
	stub.lastFailure = failure
	stub.lastReceipt = receipt
	return nil
}

func TestCommunicationDeliveryWorkerSkipsAlreadyFinalAttempt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	communications := activeCommunicationServiceForWorker(now, CommunicationInvitation)
	repository := &communicationDeliveryRepositoryStub{
		bundle: communicationDeliveryBundle{
			Distribution: FormDistribution{ID: "distribution", TenantID: "tenant", LegalEntityID: "entity", Status: DistributionOpen, Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour)},
			Recipients: []communicationDeliveryRecipient{{DistributionRecipient: DistributionRecipient{ID: "recipient", Role: RecipientTo, Type: RecipientExternalAudience, State: DistributionRecipientDelivered}}},
		},
		prior: CommunicationDeliveryAttempt{Status: "DELIVERED", AttemptedAt: now.Add(-time.Minute)},
		priorFound: true,
	}
	deliveryCalls := 0
	delivery := NewInvitationDeliveryService(invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		deliveryCalls++
		return InvitationDeliveryReceipt{}, nil
	}))
	worker, err := NewCommunicationDeliveryWorker(repository, communications, &DistributionAccessService{}, delivery, "https://capture.example")
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	worker.now = func() time.Time { return now }
	if err := worker.Publish(context.Background(), workflowruntime.OutboxEvent{ID: "event", TenantID: "tenant", AggregateType: "FORM_DISTRIBUTION", AggregateID: "distribution", EventType: "FORM_DISTRIBUTION_OPEN"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if deliveryCalls != 0 || repository.records != 0 {
		t.Fatalf("final attempt was retried: deliveries=%d records=%d", deliveryCalls, repository.records)
	}
}

func TestCommunicationDeliveryWorkerStopsAfterDeadline(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	repository := &communicationDeliveryRepositoryStub{
		bundle: communicationDeliveryBundle{
			Distribution: FormDistribution{ID: "distribution", TenantID: "tenant", LegalEntityID: "entity", Status: DistributionOpen, Deadline: now.Add(-time.Minute), RouteExpiresAt: now.Add(time.Hour)},
			Recipients: []communicationDeliveryRecipient{{DistributionRecipient: DistributionRecipient{ID: "recipient", Role: RecipientTo, Type: RecipientExternalAudience, State: DistributionRecipientDelivered}}},
		},
	}
	deliveryCalls := 0
	delivery := NewInvitationDeliveryService(invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		deliveryCalls++
		return InvitationDeliveryReceipt{}, nil
	}))
	worker, err := NewCommunicationDeliveryWorker(repository, NewCommunicationService(NewMemoryCommunicationStore()), &DistributionAccessService{}, delivery, "https://capture.example")
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	worker.now = func() time.Time { return now }
	if err := worker.Publish(context.Background(), workflowruntime.OutboxEvent{ID: "event", TenantID: "tenant", AggregateType: "FORM_DISTRIBUTION", AggregateID: "distribution", EventType: "FORM_DISTRIBUTION_OPEN"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if deliveryCalls != 0 || repository.records != 1 || repository.lastFailure != "DISTRIBUTION_NOT_DELIVERABLE" {
		t.Fatalf("deadline stop failed: deliveries=%d records=%d failure=%q", deliveryCalls, repository.records, repository.lastFailure)
	}
}

func activeCommunicationServiceForWorker(now time.Time, action CommunicationAction) *CommunicationService {
	store := NewMemoryCommunicationStore()
	store.profiles[communicationScopeKey("tenant", "entity")] = []CommunicationProfile{{
		ID: "profile", TenantID: "tenant", LegalEntityID: "entity", Version: 1, DefaultLocale: "en",
		BankName: "Bank", Status: CommunicationActive, EffectiveFrom: now.Add(-time.Hour), MakerID: "maker",
	}}
	store.templates[communicationTemplateKey("tenant", "entity", action, "en")] = []CommunicationTemplate{{
		ID: "template", TenantID: "tenant", LegalEntityID: "entity", Action: action, Locale: "en", Version: 1,
		SubjectTemplate: "Complete {{form_title}}", Status: CommunicationActive, EffectiveFrom: now.Add(-time.Hour), MakerID: "maker",
		Document: []CommunicationNode{{Type: "paragraph", Text: "Please complete {{form_title}}."}, {Type: "primary-action", Text: "Open form", Href: "{{secure_form_link}}"}},
	}}
	service := NewCommunicationService(store)
	service.now = func() time.Time { return now }
	return service
}

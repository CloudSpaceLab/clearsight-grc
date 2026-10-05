package attention

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type criticalEmailRepoStub struct {
	value       CriticalEmailContext
	claims      []EmailDeliveryRecord
	records     []EmailDeliveryRecord
	claimResult bool
}

func (s *criticalEmailRepoStub) LoadCriticalEmailContext(context.Context, workflowruntime.OutboxEvent, Intent) (CriticalEmailContext, error) {
	return s.value, nil
}

func (s *criticalEmailRepoStub) ClaimCriticalEmail(_ context.Context, _ workflowruntime.OutboxEvent, _ Intent, record EmailDeliveryRecord) (bool, error) {
	s.claims = append(s.claims, record)
	return s.claimResult, nil
}

func (s *criticalEmailRepoStub) RecordCriticalEmail(_ context.Context, _ workflowruntime.OutboxEvent, _ Intent, record EmailDeliveryRecord) error {
	s.records = append(s.records, record)
	return nil
}

type criticalEmailDeliveryStub struct {
	calls   int
	request evidence.InvitationDeliveryRequest
	receipt evidence.InvitationDeliveryReceipt
	err     error
}

func (s *criticalEmailDeliveryStub) DeliverGoverned(_ context.Context, request evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error) {
	s.calls++
	s.request = request
	return s.receipt, s.err
}

func attentionEmailEvent(eventType, state string, sequence int) workflowruntime.OutboxEvent {
	payload := fmt.Sprintf(
		"{\"episode_id\":\"33333333-3333-4333-8333-333333333333\",\"legal_entity_id\":\"44444444-4444-4444-8444-444444444444\",\"principal_id\":\"55555555-5555-4555-8555-555555555555\",\"condition\":\"indicator_breaches\",\"condition_state\":%q,\"subject_type\":\"RISK\",\"subject_id\":\"66666666-6666-4666-8666-666666666666\",\"source_id\":\"77777777-7777-4777-8777-777777777777\",\"notice_sequence\":%d}",
		state, sequence,
	)
	return workflowruntime.OutboxEvent{
		ID:            "11111111-1111-4111-8111-111111111111",
		TenantID:      "22222222-2222-4222-8222-222222222222",
		AggregateType: EpisodeAggregateType,
		AggregateID:   "33333333-3333-4333-8333-333333333333",
		EventType:     eventType,
		Payload:       []byte(payload),
	}
}

func TestCriticalEmailSendsOnlyCriticalOpenOrWorsen(t *testing.T) {
	now := time.Now().UTC()
	repo := &criticalEmailRepoStub{
		value: CriticalEmailContext{
			LegalEntityID:         "44444444-4444-4444-8444-444444444444",
			BrandName:             "Meridian Bank",
			RecipientName:         "Risk Officer",
			RecipientAddress:      "risk@example.test",
			CurrentNoticeSequence: 2,
		},
		claimResult: true,
	}
	delivery := &criticalEmailDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{
		Status:      evidence.InvitationDelivered,
		DeliveredAt: &now,
	}}
	consumer := NewCriticalEmailConsumer(repo, delivery, "https://clearsight.example.test")
	consumer.now = func() time.Time { return now }

	if err := consumer.Publish(context.Background(), attentionEmailEvent(EventEpisodeWorsened, "CRITICAL", 2)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 || len(repo.claims) != 1 || repo.claims[0].Status != "DELIVERY_STARTED" {
		t.Fatalf("calls=%d claims=%#v", delivery.calls, repo.claims)
	}
	if len(repo.records) != 1 || repo.records[0].Status != "DELIVERED" {
		t.Fatalf("records=%#v", repo.records)
	}

	if err := consumer.Publish(context.Background(), attentionEmailEvent(EventEpisodeOpened, "HIGH", 2)); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), attentionEmailEvent(EventEpisodeCleared, "CRITICAL", 2)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 {
		t.Fatalf("routine/clear transitions sent immediate email: calls=%d", delivery.calls)
	}
}

func TestCriticalEmailClaimPreventsDuplicateAndUnknownOutcomeIsTerminal(t *testing.T) {
	repo := &criticalEmailRepoStub{
		value: CriticalEmailContext{
			BrandName:             "Meridian Bank",
			RecipientName:         "Risk Officer",
			RecipientAddress:      "risk@example.test",
			CurrentNoticeSequence: 1,
		},
		claimResult: false,
	}
	delivery := &criticalEmailDeliveryStub{}
	consumer := NewCriticalEmailConsumer(repo, delivery, "https://clearsight.example.test")
	if err := consumer.Publish(context.Background(), attentionEmailEvent(EventEpisodeOpened, "CRITICAL", 1)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 {
		t.Fatal("duplicate claim reached SMTP")
	}

	repo.claimResult = true
	delivery.receipt = evidence.InvitationDeliveryReceipt{
		Status:      evidence.InvitationDeliveryFailed,
		FailureCode: evidence.InvitationFailureOutcomeUnknown,
	}
	delivery.err = errors.New("unknown outcome")
	if err := consumer.Publish(context.Background(), attentionEmailEvent(EventEpisodeOpened, "CRITICAL", 1)); err != nil {
		t.Fatalf("unknown delivery outcome must not auto-retry: %v", err)
	}
	if got := repo.records[len(repo.records)-1].Status; got != "DELIVERY_OUTCOME_UNKNOWN" {
		t.Fatalf("status=%s", got)
	}
}

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type assignmentNotificationRepositoryStub struct {
	context     assignmentNotificationContext
	prior       assignmentNotificationRecord
	found       bool
	recorded    assignmentNotificationRecord
	loadCalls   int
	claimCalls  int
	claimDenied bool
	recordErr   error
}

func (stub *assignmentNotificationRepositoryStub) LoadAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) (assignmentNotificationContext, error) {
	stub.loadCalls++
	return stub.context, nil
}

func (stub *assignmentNotificationRepositoryStub) GetAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) (assignmentNotificationRecord, bool, error) {
	return stub.prior, stub.found, nil
}

func (stub *assignmentNotificationRepositoryStub) ClaimAssignmentNotification(_ context.Context, _ workflowruntime.OutboxEvent, _ assignmentNotificationEvent, record assignmentNotificationRecord) (bool, error) {
	stub.claimCalls++
	if stub.claimDenied {
		return false, nil
	}
	stub.prior = record
	stub.found = true
	return true, nil
}

func (stub *assignmentNotificationRepositoryStub) RecordAssignmentNotification(_ context.Context, _ workflowruntime.OutboxEvent, _ assignmentNotificationEvent, record assignmentNotificationRecord) error {
	stub.recorded = record
	if stub.recordErr != nil {
		return stub.recordErr
	}
	stub.prior = record
	stub.found = true
	return nil
}

type assignmentDeliveryStub struct {
	receipt evidence.InvitationDeliveryReceipt
	err     error
	calls   int
	request evidence.InvitationDeliveryRequest
}

func (stub *assignmentDeliveryStub) DeliverGoverned(_ context.Context, request evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error) {
	stub.calls++
	stub.request = request
	return stub.receipt, stub.err
}

func matterOwnerAssignmentEvent(t *testing.T) workflowruntime.OutboxEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"matter": map[string]any{
			"id":                 "00000000-0000-4000-8000-000000000701",
			"tenant_id":          "bank-1",
			"legal_entity_id":    "00000000-0000-4000-8000-000000000702",
			"owner_principal_id": "00000000-0000-4000-8000-000000000703",
		},
		"previous_owner_principal_id": "00000000-0000-4000-8000-000000000704",
		"owner_principal_id":          "00000000-0000-4000-8000-000000000703",
	})
	if err != nil {
		t.Fatal(err)
	}
	return workflowruntime.OutboxEvent{
		ID: "00000000-0000-4000-8000-000000000705", TenantID: "bank-1", AggregateType: "MATTER",
		AggregateID: "00000000-0000-4000-8000-000000000701", EventType: continuity.EventMatterOwnerChanged,
		Payload: payload, OccurredAt: time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC),
	}
}

func actionPerformerAssignmentEvent(t *testing.T) workflowruntime.OutboxEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"action": map[string]any{
			"id":                 "00000000-0000-4000-8000-000000000706",
			"matter_id":          "00000000-0000-4000-8000-000000000701",
			"owner_principal_id": "00000000-0000-4000-8000-000000000703",
		},
		"previous_owner_principal_id": "00000000-0000-4000-8000-000000000704",
		"owner_principal_id":          "00000000-0000-4000-8000-000000000703",
	})
	if err != nil {
		t.Fatal(err)
	}
	return workflowruntime.OutboxEvent{
		ID: "00000000-0000-4000-8000-000000000707", TenantID: "bank-1", AggregateType: "MATTER",
		AggregateID: "00000000-0000-4000-8000-000000000701", EventType: continuity.EventActionAssigned,
		Payload: payload, OccurredAt: time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC),
	}
}

func deliverableAssignmentContext() assignmentNotificationContext {
	return assignmentNotificationContext{
		LegalEntityID: "00000000-0000-4000-8000-000000000702",
		BankName:      "Clear Bank Nigeria", RecipientName: "Program Owner", RecipientAddress: "staff@example.test",
		CurrentPrincipalID: "00000000-0000-4000-8000-000000000703",
		MatterID:           "00000000-0000-4000-8000-000000000701", MatterTitle: "Verify vendor address",
		WorkTitle: "Confirm scope and owner", DueAt: time.Date(2026, 9, 5, 16, 0, 0, 0, time.UTC),
	}
}

func TestAssignmentNotificationDeliversMatterOwnerEmailAndRecordsFinalReceipt(t *testing.T) {
	repo := &assignmentNotificationRepositoryStub{context: deliverableAssignmentContext()}
	deliveredAt := time.Date(2026, 8, 31, 10, 1, 0, 0, time.UTC)
	delivery := &assignmentDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDelivered, ProviderMessageID: "provider-1", DeliveredAt: &deliveredAt}}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 || delivery.request.RecipientAddress != "staff@example.test" {
		t.Fatalf("delivery calls=%d request=%#v", delivery.calls, delivery.request)
	}
	if repo.recorded.Status != assignmentNotificationDelivered || repo.recorded.ProviderMessageID != "provider-1" || len(repo.recorded.RecipientFingerprint) != 32 {
		t.Fatalf("recorded receipt = %#v", repo.recorded)
	}
	if got := delivery.request.String(); got != "InvitationDeliveryRequest{protected}" {
		t.Fatalf("protected request formatted as %q", got)
	}
}

func TestAssignmentNotificationDeliversExactActionPerformerWork(t *testing.T) {
	notificationContext := deliverableAssignmentContext()
	notificationContext.WorkTitle = "Confirm the registered address"
	repo := &assignmentNotificationRepositoryStub{context: notificationContext}
	deliveredAt := time.Date(2026, 8, 31, 10, 1, 0, 0, time.UTC)
	delivery := &assignmentDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDelivered, ProviderMessageID: "provider-2", DeliveredAt: &deliveredAt}}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), actionPerformerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 || !strings.Contains(delivery.request.PlainText, "Assigned performer") || !strings.Contains(delivery.request.PlainText, notificationContext.WorkTitle) {
		t.Fatalf("action assignment request = %#v", delivery.request)
	}
}

func TestAssignmentNotificationRecordsMissingContactWithoutRetryingAssignment(t *testing.T) {
	context := deliverableAssignmentContext()
	context.RecipientAddress = ""
	repo := &assignmentNotificationRepositoryStub{context: context}
	delivery := &assignmentDeliveryStub{}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 || repo.recorded.Status != assignmentNotificationContactUnavailable {
		t.Fatalf("delivery calls=%d record=%#v", delivery.calls, repo.recorded)
	}
}

func TestAssignmentNotificationSkipsSupersededAssignment(t *testing.T) {
	context := deliverableAssignmentContext()
	context.CurrentPrincipalID = "00000000-0000-4000-8000-000000000706"
	repo := &assignmentNotificationRepositoryStub{context: context}
	delivery := &assignmentDeliveryStub{}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 || repo.recorded.Status != assignmentNotificationSuperseded {
		t.Fatalf("delivery calls=%d record=%#v", delivery.calls, repo.recorded)
	}
}

func TestAssignmentNotificationRetriesTemporaryDeliveryAndFinalReceiptIsIdempotent(t *testing.T) {
	repo := &assignmentNotificationRepositoryStub{context: deliverableAssignmentContext()}
	delivery := &assignmentDeliveryStub{err: evidence.ErrInvitationDeliveryFailed, receipt: evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDeliveryFailed, FailureCode: evidence.InvitationFailureTemporary}}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); !errors.Is(err, ErrAssignmentNotificationRetry) {
		t.Fatalf("temporary failure error = %v", err)
	}
	if repo.recorded.Status != assignmentNotificationTemporaryFailure {
		t.Fatalf("temporary receipt = %#v", repo.recorded)
	}

	deliveredAt := time.Date(2026, 8, 31, 10, 2, 0, 0, time.UTC)
	delivery.err = nil
	delivery.receipt = evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDelivered, ProviderMessageID: "provider-retry", DeliveredAt: &deliveredAt}
	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 2 || repo.claimCalls != 2 || repo.recorded.Status != assignmentNotificationDelivered {
		t.Fatalf("retry delivery calls=%d claims=%d receipt=%#v", delivery.calls, repo.claimCalls, repo.recorded)
	}

	delivery.calls = 0
	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 || repo.loadCalls != 2 {
		t.Fatalf("final replay delivery calls=%d context loads=%d", delivery.calls, repo.loadCalls)
	}
}

func TestAssignmentNotificationDoesNotRedeliverWhenFinalReceiptWriteFails(t *testing.T) {
	recordErr := errors.New("receipt store unavailable")
	repo := &assignmentNotificationRepositoryStub{context: deliverableAssignmentContext(), recordErr: recordErr}
	deliveredAt := time.Date(2026, 8, 31, 10, 1, 0, 0, time.UTC)
	delivery := &assignmentDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDelivered, ProviderMessageID: "provider-1", DeliveredAt: &deliveredAt}}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); !errors.Is(err, recordErr) {
		t.Fatalf("receipt write error = %v", err)
	}
	if repo.prior.Status != assignmentNotificationDeliveryStarted || delivery.calls != 1 {
		t.Fatalf("claim=%#v delivery calls=%d", repo.prior, delivery.calls)
	}
	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 || repo.claimCalls != 1 {
		t.Fatalf("replay delivery calls=%d claims=%d", delivery.calls, repo.claimCalls)
	}
}

func TestAssignmentNotificationSkipsDeliveryWhenAnotherWorkerOwnsClaim(t *testing.T) {
	repo := &assignmentNotificationRepositoryStub{context: deliverableAssignmentContext(), claimDenied: true}
	delivery := &assignmentDeliveryStub{}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 || repo.claimCalls != 1 {
		t.Fatalf("delivery calls=%d claims=%d", delivery.calls, repo.claimCalls)
	}
}

func TestAssignmentNotificationDoesNotRetryUnknownPostAcceptanceOutcome(t *testing.T) {
	repo := &assignmentNotificationRepositoryStub{context: deliverableAssignmentContext()}
	delivery := &assignmentDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{
		Status: evidence.InvitationDeliveryFailed, FailureCode: evidence.InvitationFailureOutcomeUnknown,
	}}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}

	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if repo.recorded.Status != assignmentNotificationOutcomeUnknown || delivery.calls != 1 {
		t.Fatalf("receipt=%#v delivery calls=%d", repo.recorded, delivery.calls)
	}
	if err := consumer.Publish(t.Context(), matterOwnerAssignmentEvent(t)); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 1 {
		t.Fatalf("unknown outcome was redelivered: calls=%d", delivery.calls)
	}
}

func TestAssignmentNotificationIgnoresUnrelatedEventsAndRejectsInsecureBaseURL(t *testing.T) {
	repo := &assignmentNotificationRepositoryStub{}
	delivery := &assignmentDeliveryStub{}
	consumer, err := NewAssignmentNotificationConsumer(repo, delivery, "https://clearsight.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(t.Context(), workflowruntime.OutboxEvent{EventType: "MATTER_UPDATED"}); err != nil || repo.loadCalls != 0 {
		t.Fatalf("unrelated event error=%v loads=%d", err, repo.loadCalls)
	}
	if _, err := NewAssignmentNotificationConsumer(repo, delivery, "http://clearsight.example.test/"); err == nil {
		t.Fatal("insecure application base URL accepted")
	}
}

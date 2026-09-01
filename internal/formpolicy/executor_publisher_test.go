package formpolicy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type scoredResponseHandlerStub struct {
	events []ScoredResponseEvent
	err    error
}

func (stub *scoredResponseHandlerStub) Handle(_ context.Context, event ScoredResponseEvent) ([]ExecutionReceipt, error) {
	stub.events = append(stub.events, event)
	return nil, stub.err
}

func TestScoredResponsePublisherConsumesOnlyBoundedScoredEvents(t *testing.T) {
	handler := &scoredResponseHandlerStub{}
	publisher := ScoredResponsePublisher{Handler: handler}
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]any{"version": 2, "response_revision_id": "response-a", "form_template_id": "form-a", "form_template_version": 3, "score_state": "FINAL"})
	event := workflowruntime.OutboxEvent{ID: "event-a", TenantID: "bank", AggregateType: "FORM_DISTRIBUTION", AggregateID: "distribution-a", EventType: "FORM_RESPONSE_SCORED", Payload: payload, OccurredAt: now}
	if err := publisher.Publish(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	if len(handler.events) != 1 || handler.events[0].ResponseRevisionID != "response-a" || handler.events[0].TenantID != "bank" {
		t.Fatalf("handled events = %#v", handler.events)
	}
	if err := publisher.Publish(t.Context(), workflowruntime.OutboxEvent{EventType: "MATTER_CREATED"}); err != nil || len(handler.events) != 1 {
		t.Fatalf("unrelated event was consumed: events=%#v err=%v", handler.events, err)
	}
	oversized, _ := json.Marshal(map[string]any{"response_revision_id": string(make([]byte, 513)), "form_template_id": "form-a", "form_template_version": 3, "score_state": "FINAL"})
	event.Payload = oversized
	if err := publisher.Publish(t.Context(), event); err == nil {
		t.Fatal("expected oversized scored event to fail closed")
	}
}

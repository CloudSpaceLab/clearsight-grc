package formpolicy

import (
	"context"
	"encoding/json"
	"strings"
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

func TestPublisherDispatchesFinalBankAssessment(t *testing.T) {
	handler := &scoredResponseHandlerStub{}
	publisher := ScoredResponsePublisher{Handler: handler}
	event := workflowruntime.OutboxEvent{ID: "assessment-event", TenantID: "bank", EventType: "FORM_RESPONSE_ASSESSED", OccurredAt: time.Now(), Payload: json.RawMessage(`{"version":1,"response_revision_id":"response-a","form_template_id":"form-a","form_template_version":3,"assessment_version":2,"result_basis":"BANK_ASSESSED","score_state":"FINAL"}`)}
	if err := publisher.Publish(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	if len(handler.events) != 1 {
		t.Fatal("bank assessment event was ignored")
	}
}

func TestScoredResponsePublisherAcceptsRequestVersionProvenance(t *testing.T) {
	handler := &scoredResponseHandlerStub{}
	event := workflowruntime.OutboxEvent{ID: "event-source", TenantID: "bank", EventType: "FORM_RESPONSE_SCORED", OccurredAt: time.Now(), Payload: json.RawMessage(`{"version":2,"response_revision_id":"response-a","request_id":"request-a","request_version":7,"form_template_id":"form-a","form_template_version":3,"score_state":"FINAL"}`)}
	if err := (ScoredResponsePublisher{Handler: handler}).Publish(t.Context(), event); err != nil {
		t.Fatalf("producer provenance rejected: %v", err)
	}
	if len(handler.events) != 1 {
		t.Fatal("scored response was not delivered")
	}
	if handler.events[0].RequestID != "request-a" || handler.events[0].RequestVersion != 7 {
		t.Fatal("request provenance was not preserved")
	}
}

func TestScoredResponsePublisherRejectsInvalidProvenanceAndUnknownFields(t *testing.T) {
	for _, test := range []struct {
		name  string
		extra map[string]any
	}{
		{"missing version", map[string]any{"request_id": "request-a"}},
		{"missing request", map[string]any{"request_version": 7}},
		{"blank request", map[string]any{"request_id": " ", "request_version": 7}},
		{"oversized request", map[string]any{"request_id": strings.Repeat("a", 513), "request_version": 7}},
		{"zero version", map[string]any{"request_id": "request-a", "request_version": 0}},
		{"negative version", map[string]any{"request_id": "request-a", "request_version": -1}},
		{"fractional version", map[string]any{"request_id": "request-a", "request_version": 1.5}},
		{"text version", map[string]any{"request_id": "request-a", "request_version": "7"}},
		{"null version", map[string]any{"request_id": "request-a", "request_version": nil}},
		{"unknown field", map[string]any{"request_id": "request-a", "request_version": 7, "approve_vendor": true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := &scoredResponseHandlerStub{}
			payload := map[string]any{"version": 2, "response_revision_id": "response-a", "form_template_id": "form-a", "form_template_version": 3, "score_state": "FINAL"}
			for k, v := range test.extra {
				payload[k] = v
			}
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			event := workflowruntime.OutboxEvent{ID: "event-source", TenantID: "bank", EventType: "FORM_RESPONSE_SCORED", OccurredAt: time.Now(), Payload: encoded}
			if err = (ScoredResponsePublisher{Handler: handler}).Publish(t.Context(), event); err == nil || len(handler.events) != 0 {
				t.Fatal("malformed event reached policy executor")
			}
		})
	}
}

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestAssessmentCancellationConsumerRevokesCurrentRequestCapabilities(t *testing.T) {
	revoker := &assessmentCancellationRevokerStub{}
	consumer := NewAssessmentCancellationConsumer(revoker)
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "bank-a", AggregateType: "THIRD_PARTY_ASSESSMENT", AggregateID: "assessment-1", EventType: "AssessmentCancelled", Payload: json.RawMessage(`{"request_id":"request-1"}`), OccurredAt: time.Now().UTC()}

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if revoker.tenant != "bank-a" || revoker.requestID != "request-1" {
		t.Fatalf("revocation = (%q,%q)", revoker.tenant, revoker.requestID)
	}
}

func TestAssessmentCancellationConsumerRetriesRevocationFailure(t *testing.T) {
	revoker := &assessmentCancellationRevokerStub{err: errors.New("evidence unavailable")}
	consumer := NewAssessmentCancellationConsumer(revoker)
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "bank-a", AggregateType: "THIRD_PARTY_ASSESSMENT", AggregateID: "assessment-1", EventType: "AssessmentCancelled", Payload: json.RawMessage(`{"request_id":"request-1"}`), OccurredAt: time.Now().UTC()}

	if err := consumer.Publish(context.Background(), event); err == nil {
		t.Fatal("revocation failure was acknowledged")
	}
}

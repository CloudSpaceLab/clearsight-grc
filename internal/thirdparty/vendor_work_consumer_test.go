package thirdparty

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestVendorWorkConsumerRecordsCurrentCaptureSubmissionExactlyOnce(t *testing.T) {
	inbox := &assessmentConsumerInboxFake{seen: map[string]bool{}}
	requests := &assessmentConsumerRequestFake{request: evidence.Request{ID: "request-1", TenantID: "bank-a", Origin: evidence.RequestOrigin{Type: VendorWorkOrigin, ID: "work-1", Version: 2}}}
	resolver := &vendorWorkResolverFake{target: VendorWorkSubmissionTarget{Scope: Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}, WorkRequestID: "work-1", WorkVersion: 5, RequestID: "request-1"}}
	reaction := &vendorWorkReactionFake{}
	consumer := &VendorWorkConsumer{Inbox: inbox, Requests: requests, Resolver: resolver, Reactions: reaction}
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "bank-a", AggregateType: assessmentSubmissionAggregateType, AggregateID: "request-1", EventType: assessmentSubmissionEventType, Payload: json.RawMessage(`{"submission_id":"submission-1","channel":"EXTERNAL"}`), OccurredAt: time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)}

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if reaction.calls != 1 || reaction.input.WorkRequestID != "work-1" || reaction.input.RequestID != "request-1" || reaction.input.SubmissionID != "submission-1" || reaction.input.CausationID != event.ID {
		t.Fatalf("reaction = %#v calls=%d", reaction.input, reaction.calls)
	}
}

func TestVendorWorkConsumerIgnoresNonVendorRequestAcrossTenantAliases(t *testing.T) {
	inbox := &assessmentConsumerInboxFake{seen: map[string]bool{}}
	requests := &assessmentConsumerRequestFake{request: evidence.Request{ID: "request-1", TenantID: "bank-a"}}
	resolver := &vendorWorkResolverFake{}
	reaction := &vendorWorkReactionFake{}
	consumer := &VendorWorkConsumer{Inbox: inbox, Requests: requests, Resolver: resolver, Reactions: reaction}
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "00000000-0000-4000-8000-000000000001", AggregateType: assessmentSubmissionAggregateType, AggregateID: "request-1", EventType: assessmentSubmissionEventType, Payload: json.RawMessage(`{"submission_id":"submission-1","channel":"INTERNAL"}`), OccurredAt: time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)}

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if requests.tenant != event.TenantID || resolver.calls != 0 || reaction.calls != 0 || inbox.records != 0 {
		t.Fatalf("non-vendor alias handling request tenant=%q resolver=%d reaction=%d inbox=%d", requests.tenant, resolver.calls, reaction.calls, inbox.records)
	}
}

func TestVendorWorkConsumerRejectsWrongRequestFromScopedRead(t *testing.T) {
	inbox := &assessmentConsumerInboxFake{seen: map[string]bool{}}
	requests := &assessmentConsumerRequestFake{request: evidence.Request{ID: "request-other", TenantID: "bank-a", Origin: evidence.RequestOrigin{Type: VendorWorkOrigin, ID: "work-1", Version: 2}}}
	resolver := &vendorWorkResolverFake{}
	reaction := &vendorWorkReactionFake{}
	consumer := &VendorWorkConsumer{Inbox: inbox, Requests: requests, Resolver: resolver, Reactions: reaction}
	event := workflowruntime.OutboxEvent{ID: "event-1", TenantID: "bank-a", AggregateType: assessmentSubmissionAggregateType, AggregateID: "request-1", EventType: assessmentSubmissionEventType, Payload: json.RawMessage(`{"submission_id":"submission-1","channel":"EXTERNAL"}`), OccurredAt: time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)}

	if err := consumer.Publish(context.Background(), event); err == nil {
		t.Fatal("wrong request identity was accepted")
	}
	if resolver.calls != 0 || reaction.calls != 0 || inbox.records != 0 {
		t.Fatalf("wrong request caused effects: resolver=%d reaction=%d inbox=%d", resolver.calls, reaction.calls, inbox.records)
	}
}

type vendorWorkResolverFake struct {
	target VendorWorkSubmissionTarget
	calls  int
}

func (f *vendorWorkResolverFake) ResolveVendorWorkCapture(_ context.Context, _ string, _ evidence.RequestOrigin, _ string) (VendorWorkSubmissionTarget, error) {
	f.calls++
	return f.target, nil
}

type vendorWorkReactionFake struct {
	input VendorWorkSubmissionInput
	calls int
}

func (f *vendorWorkReactionFake) RecordSubmission(_ context.Context, input VendorWorkSubmissionInput) (VendorWorkRequest, error) {
	f.calls++
	f.input = input
	return VendorWorkRequest{ID: input.WorkRequestID}, nil
}

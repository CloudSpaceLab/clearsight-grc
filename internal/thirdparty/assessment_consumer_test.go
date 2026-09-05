package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestAssessmentSubmissionConsumerAdvancesExactlyOnceOnReplay(t *testing.T) {
	consumer, inbox, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	event := assessmentSubmissionEvent()

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if requests.calls != 1 || resolver.calls != 1 || reactions.calls != 1 {
		t.Fatalf("replay calls request=%d resolver=%d reaction=%d", requests.calls, resolver.calls, reactions.calls)
	}
	if !inbox.processed(assessmentSubmissionConsumerName, event.ID) || inbox.records != 1 {
		t.Fatalf("inbox state = %#v", inbox)
	}
}

func TestAssessmentSubmissionConsumerIgnoresUnrelatedEvents(t *testing.T) {
	consumer, inbox, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	for _, event := range []workflowruntime.OutboxEvent{
		{AggregateType: "MATTER", EventType: assessmentSubmissionEventType},
		{AggregateType: assessmentSubmissionAggregateType, EventType: "EvidenceRequestExpired"},
	} {
		if err := consumer.Publish(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	if requests.calls != 0 || resolver.calls != 0 || reactions.calls != 0 || inbox.records != 0 {
		t.Fatalf("unrelated event caused work: request=%d resolver=%d reaction=%d inbox=%d", requests.calls, resolver.calls, reactions.calls, inbox.records)
	}
}

func TestAssessmentSubmissionConsumerIgnoresRequestWithoutAssessmentOrigin(t *testing.T) {
	consumer, inbox, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	requests.request.Origin = evidence.RequestOrigin{Type: "MONITORING_CHECK", ID: "check-1", Version: 1}

	if err := consumer.Publish(context.Background(), assessmentSubmissionEvent()); err != nil {
		t.Fatal(err)
	}
	if requests.calls != 1 || resolver.calls != 0 || reactions.calls != 0 || inbox.records != 0 {
		t.Fatalf("non-assessment request caused work: request=%d resolver=%d reaction=%d inbox=%d", requests.calls, resolver.calls, reactions.calls, inbox.records)
	}
}

func TestAssessmentSubmissionConsumerAcceptsTenantAliasFromScopedRequestRead(t *testing.T) {
	consumer, _, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	event := assessmentSubmissionEvent()
	event.TenantID = "00000000-0000-4000-8000-000000000001"
	requests.request.TenantID = "bank-a"
	resolver.target.TenantID = event.TenantID

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if requests.tenant != event.TenantID || resolver.calls != 1 || reactions.calls != 1 {
		t.Fatalf("tenant alias handling request tenant=%q resolver=%d reaction=%d", requests.tenant, resolver.calls, reactions.calls)
	}
}

func TestAssessmentSubmissionConsumerRejectsWrongRequestFromScopedRead(t *testing.T) {
	consumer, inbox, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	requests.request.ID = "request-other"

	if err := consumer.Publish(context.Background(), assessmentSubmissionEvent()); err == nil {
		t.Fatal("wrong request identity was accepted")
	}
	if resolver.calls != 0 || reactions.calls != 0 || inbox.records != 0 {
		t.Fatalf("wrong request caused effects: resolver=%d reaction=%d inbox=%d", resolver.calls, reactions.calls, inbox.records)
	}
}

func TestAssessmentSubmissionConsumerRejectsMalformedOrUnsafePayload(t *testing.T) {
	cases := []json.RawMessage{
		json.RawMessage(`{`),
		json.RawMessage(`{}`),
		json.RawMessage(`{"submission_id":"submission-1","answers":{"secret":"value"}}`),
		json.RawMessage(`{"submission_id":" "}`),
	}
	for _, payload := range cases {
		consumer, inbox, _, resolver, reactions := assessmentSubmissionConsumerFixture()
		event := assessmentSubmissionEvent()
		event.Payload = payload
		if err := consumer.Publish(context.Background(), event); err == nil {
			t.Fatalf("payload %s was accepted", payload)
		}
		if resolver.calls != 0 || reactions.calls != 0 || inbox.records != 0 {
			t.Fatalf("malformed payload caused effects: resolver=%d reaction=%d inbox=%d", resolver.calls, reactions.calls, inbox.records)
		}
	}
}

func TestAssessmentSubmissionConsumerDoesNotRecordInboxWhenReactionFails(t *testing.T) {
	consumer, inbox, _, _, reactions := assessmentSubmissionConsumerFixture()
	reactions.err = errors.New("assessment store unavailable")
	event := assessmentSubmissionEvent()

	if err := consumer.Publish(context.Background(), event); err == nil {
		t.Fatal("reaction failure was not returned")
	}
	if inbox.records != 0 || inbox.processed(assessmentSubmissionConsumerName, event.ID) {
		t.Fatalf("failed reaction recorded inbox: %#v", inbox)
	}
	reactions.err = nil
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if reactions.calls != 2 || inbox.records != 1 {
		t.Fatalf("retry calls=%d inbox=%d", reactions.calls, inbox.records)
	}
}

func TestAssessmentSubmissionConsumerPropagatesExactTenantScopeAndCausation(t *testing.T) {
	consumer, inbox, requests, resolver, reactions := assessmentSubmissionConsumerFixture()
	event := assessmentSubmissionEvent()
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if requests.tenant != event.TenantID || requests.requestID != event.AggregateID {
		t.Fatalf("request read scope = tenant %q request %q", requests.tenant, requests.requestID)
	}
	if resolver.tenant != event.TenantID || resolver.requestID != event.AggregateID || resolver.origin != requests.request.Origin {
		t.Fatalf("resolver input = tenant %q request %q origin %#v", resolver.tenant, resolver.requestID, resolver.origin)
	}
	input := reactions.input
	if input.TenantID != event.TenantID || input.LegalEntityID != "entity-a" || input.AssessmentID != "assessment-1" || input.ExpectedVersion != 4 || input.RequestID != event.AggregateID || input.SubmissionID != "submission-1" || input.EventID != event.ID || input.CausationID != event.ID {
		t.Fatalf("reaction input = %#v", input)
	}
	if inbox.tenant != event.TenantID || inbox.consumer != assessmentSubmissionConsumerName || inbox.eventID != event.ID || !inbox.at.Equal(event.OccurredAt) {
		t.Fatalf("inbox input = %#v", inbox)
	}
}

type assessmentConsumerInboxFake struct {
	seen                      map[string]bool
	records                   int
	tenant, consumer, eventID string
	at                        time.Time
}

func (f *assessmentConsumerInboxFake) InboxProcessed(_ context.Context, tenant, consumer, eventID string) (bool, error) {
	return f.seen[tenant+":"+consumer+":"+eventID], nil
}

func (f *assessmentConsumerInboxFake) RecordInbox(_ context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {
	key := tenant + ":" + consumer + ":" + eventID
	if f.seen[key] {
		return false, nil
	}
	f.seen[key] = true
	f.records++
	f.tenant, f.consumer, f.eventID, f.at = tenant, consumer, eventID, at
	return true, nil
}

func (f *assessmentConsumerInboxFake) processed(consumer, eventID string) bool {
	return f.seen["bank-a:"+consumer+":"+eventID]
}

type assessmentConsumerRequestFake struct {
	request           evidence.Request
	err               error
	calls             int
	tenant, requestID string
}

func (f *assessmentConsumerRequestFake) GetRequest(_ context.Context, tenant, requestID string) (evidence.Request, error) {
	f.calls++
	f.tenant, f.requestID = tenant, requestID
	return f.request, f.err
}

type assessmentConsumerResolverFake struct {
	target            AssessmentSubmissionTarget
	err               error
	calls             int
	tenant, requestID string
	origin            evidence.RequestOrigin
}

func (f *assessmentConsumerResolverFake) ResolveAssessmentRequest(_ context.Context, tenant string, origin evidence.RequestOrigin, requestID string) (AssessmentSubmissionTarget, error) {
	f.calls++
	f.tenant, f.origin, f.requestID = tenant, origin, requestID
	return f.target, f.err
}

type assessmentConsumerReactionFake struct {
	input AssessmentSubmittedInput
	err   error
	calls int
}

func (f *assessmentConsumerReactionFake) RecordAssessmentSubmitted(_ context.Context, input AssessmentSubmittedInput) (Assessment, error) {
	f.calls++
	f.input = input
	return Assessment{ID: input.AssessmentID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Status: AssessmentSubmitted, Version: input.ExpectedVersion + 1}, f.err
}

func assessmentSubmissionConsumerFixture() (*AssessmentConsumer, *assessmentConsumerInboxFake, *assessmentConsumerRequestFake, *assessmentConsumerResolverFake, *assessmentConsumerReactionFake) {
	inbox := &assessmentConsumerInboxFake{seen: map[string]bool{}}
	request := &assessmentConsumerRequestFake{request: evidence.Request{
		ID: "request-1", TenantID: "bank-a",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1},
	}}
	resolver := &assessmentConsumerResolverFake{target: AssessmentSubmissionTarget{
		Scope: Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}, AssessmentID: "assessment-1", AssessmentVersion: 4, RequestID: "request-1",
	}}
	reactions := &assessmentConsumerReactionFake{}
	consumer := &AssessmentConsumer{Inbox: inbox, Requests: request, Resolver: resolver, Reactions: reactions}
	return consumer, inbox, request, resolver, reactions
}

func assessmentSubmissionEvent() workflowruntime.OutboxEvent {
	return workflowruntime.OutboxEvent{
		ID: "event-1", TenantID: "bank-a", AggregateType: assessmentSubmissionAggregateType,
		AggregateID: "request-1", EventType: assessmentSubmissionEventType,
		Payload:    json.RawMessage(`{"submission_id":"submission-1","channel":"EXTERNAL"}`),
		OccurredAt: time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC),
	}
}

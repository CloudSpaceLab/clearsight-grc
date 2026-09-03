package thirdparty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	assessmentSubmissionConsumerName  = "third-party-assessment-submission"
	assessmentSubmissionAggregateType = "EVIDENCE_REQUEST"
	assessmentSubmissionEventType     = "EvidenceResponseSubmitted"
)

type AssessmentSubmissionInbox interface {
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type AssessmentSubmissionRequestReader interface {
	GetRequest(context.Context, string, string) (evidence.Request, error)
}

type AssessmentSubmissionTarget struct {
	Scope
	AssessmentID      string
	AssessmentVersion int64
	RequestID         string
}

type AssessmentSubmissionResolver interface {
	ResolveAssessmentRequest(context.Context, string, evidence.RequestOrigin, string) (AssessmentSubmissionTarget, error)
}

type AssessmentSubmissionReaction interface {
	RecordAssessmentSubmitted(context.Context, AssessmentSubmittedInput) (Assessment, error)
}

type AssessmentConsumer struct {
	Inbox     AssessmentSubmissionInbox
	Requests  AssessmentSubmissionRequestReader
	Resolver  AssessmentSubmissionResolver
	Reactions AssessmentSubmissionReaction
}

type assessmentSubmissionPayload struct {
	SubmissionID string `json:"submission_id"`
	Channel      string `json:"channel,omitempty"`
}

func (c *AssessmentConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != assessmentSubmissionAggregateType || event.EventType != assessmentSubmissionEventType {
		return nil
	}
	if c == nil || c.Inbox == nil || c.Requests == nil || c.Resolver == nil || c.Reactions == nil {
		return errors.New("third-party assessment submission consumer is not configured")
	}
	if !validAssessmentIdentifiers(event.ID, event.TenantID, event.AggregateID) || event.OccurredAt.IsZero() {
		return errors.New("evidence submission event identity is invalid")
	}
	processed, err := c.Inbox.InboxProcessed(ctx, event.TenantID, assessmentSubmissionConsumerName, event.ID)
	if err != nil {
		return fmt.Errorf("check assessment submission inbox: %w", err)
	}
	if processed {
		return nil
	}
	payload, err := decodeAssessmentSubmissionPayload(event.Payload)
	if err != nil {
		return err
	}
	request, err := c.Requests.GetRequest(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return fmt.Errorf("read submitted evidence request: %w", err)
	}
	// GetRequest is already tenant-scoped and accepts either the canonical tenant
	// ID or its slug. The returned projection may use the other representation,
	// so comparing those two strings would reject an otherwise correctly scoped
	// request before this consumer can ignore non-assessment submissions.
	if request.ID != event.AggregateID {
		return errors.New("submitted evidence request does not match the event scope")
	}
	origin := evidence.RequestOrigin{
		Type: strings.ToUpper(strings.TrimSpace(request.Origin.Type)),
		ID:   strings.TrimSpace(request.Origin.ID), Version: request.Origin.Version,
	}
	if origin.Type != AssessmentRequestOrigin {
		return nil
	}
	if !validAssessmentIdentifier(origin.ID) || origin.Version < 1 {
		return errors.New("assessment request origin is invalid")
	}
	target, err := c.Resolver.ResolveAssessmentRequest(ctx, event.TenantID, origin, request.ID)
	if err != nil {
		return fmt.Errorf("resolve assessment request link: %w", err)
	}
	if target.TenantID != event.TenantID || !validAssessmentIdentifiers(target.LegalEntityID, target.AssessmentID, target.RequestID) ||
		target.AssessmentID != origin.ID || target.RequestID != request.ID || target.AssessmentVersion < 1 {
		return errors.New("assessment request link does not match the event scope")
	}
	_, err = c.Reactions.RecordAssessmentSubmitted(ctx, AssessmentSubmittedInput{
		Scope: target.Scope, AssessmentID: target.AssessmentID, ExpectedVersion: target.AssessmentVersion,
		CausationID: event.ID, EventID: event.ID, RequestID: target.RequestID, SubmissionID: payload.SubmissionID,
	})
	if err != nil {
		return fmt.Errorf("record assessment submission: %w", err)
	}
	if _, err := c.Inbox.RecordInbox(ctx, event.TenantID, assessmentSubmissionConsumerName, event.ID, event.OccurredAt.UTC()); err != nil {
		return fmt.Errorf("record assessment submission inbox: %w", err)
	}
	return nil
}

func decodeAssessmentSubmissionPayload(raw json.RawMessage) (assessmentSubmissionPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var payload assessmentSubmissionPayload
	if err := decoder.Decode(&payload); err != nil {
		return assessmentSubmissionPayload{}, fmt.Errorf("decode evidence submission event: %w", err)
	}
	if err := ensureAssessmentPayloadEOF(decoder); err != nil {
		return assessmentSubmissionPayload{}, err
	}
	payload.SubmissionID = strings.TrimSpace(payload.SubmissionID)
	payload.Channel = strings.TrimSpace(payload.Channel)
	if !validAssessmentIdentifier(payload.SubmissionID) || len(payload.Channel) > 64 {
		return assessmentSubmissionPayload{}, errors.New("evidence submission event payload is invalid")
	}
	return payload, nil
}

func ensureAssessmentPayloadEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("decode evidence submission event: %w", err)
	}
	return errors.New("evidence submission event contains multiple values")
}

var _ interface {
	Publish(context.Context, workflowruntime.OutboxEvent) error
} = (*AssessmentConsumer)(nil)

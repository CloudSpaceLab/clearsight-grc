package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const vendorWorkSubmissionConsumerName = "third-party-work-submission"

type VendorWorkSubmissionResolver interface {
	ResolveVendorWorkCapture(context.Context, string, evidence.RequestOrigin, string) (VendorWorkSubmissionTarget, error)
}

type VendorWorkSubmissionReaction interface {
	RecordSubmission(context.Context, VendorWorkSubmissionInput) (VendorWorkRequest, error)
}

type vendorWorkSubmissionRecorder struct {
	repo VendorWorkRepository
	now  func() time.Time
}

func NewVendorWorkSubmissionRecorder(repo VendorWorkRepository) VendorWorkSubmissionReaction {
	return &vendorWorkSubmissionRecorder{repo: repo, now: time.Now}
}

func (r *vendorWorkSubmissionRecorder) RecordSubmission(ctx context.Context, input VendorWorkSubmissionInput) (VendorWorkRequest, error) {
	if r == nil || r.repo == nil {
		return VendorWorkRequest{}, errors.New("vendor work submission repository is not configured")
	}
	return r.repo.RecordVendorWorkSubmission(ctx, input, r.now().UTC())
}

type VendorWorkConsumer struct {
	Inbox     AssessmentSubmissionInbox
	Requests  AssessmentSubmissionRequestReader
	Resolver  VendorWorkSubmissionResolver
	Reactions VendorWorkSubmissionReaction
}

func (c *VendorWorkConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != assessmentSubmissionAggregateType || event.EventType != assessmentSubmissionEventType {
		return nil
	}
	if c == nil || c.Inbox == nil || c.Requests == nil || c.Resolver == nil || c.Reactions == nil {
		return errors.New("vendor work submission consumer is not configured")
	}
	if !validAssessmentIdentifiers(event.ID, event.TenantID, event.AggregateID) || event.OccurredAt.IsZero() {
		return errors.New("evidence submission event identity is invalid")
	}
	processed, err := c.Inbox.InboxProcessed(ctx, event.TenantID, vendorWorkSubmissionConsumerName, event.ID)
	if err != nil {
		return fmt.Errorf("check vendor work submission inbox: %w", err)
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
		return fmt.Errorf("read submitted vendor request: %w", err)
	}
	// GetRequest has already enforced the event tenant at the repository
	// boundary. Its projection may return the tenant slug when the event carries
	// the canonical ID, so only the request identity needs a second comparison.
	if request.ID != event.AggregateID {
		return errors.New("submitted vendor request does not match the event scope")
	}
	origin := evidence.RequestOrigin{Type: strings.ToUpper(strings.TrimSpace(request.Origin.Type)), ID: strings.TrimSpace(request.Origin.ID), Version: request.Origin.Version}
	if origin.Type != VendorWorkOrigin {
		return nil
	}
	if !validAssessmentIdentifier(origin.ID) || origin.Version < 1 {
		return errors.New("vendor work request origin is invalid")
	}
	target, err := c.Resolver.ResolveVendorWorkCapture(ctx, event.TenantID, origin, request.ID)
	if err != nil {
		return fmt.Errorf("resolve vendor work capture: %w", err)
	}
	if target.TenantID != event.TenantID || target.WorkRequestID != origin.ID || target.RequestID != request.ID || target.LegalEntityID == "" {
		return errors.New("vendor work capture does not match the event scope")
	}
	_, err = c.Reactions.RecordSubmission(ctx, VendorWorkSubmissionInput{TenantID: target.TenantID, WorkRequestID: target.WorkRequestID, RequestID: target.RequestID, SubmissionID: payload.SubmissionID, CausationID: event.ID})
	if err != nil {
		return fmt.Errorf("record vendor work submission: %w", err)
	}
	if _, err := c.Inbox.RecordInbox(ctx, event.TenantID, vendorWorkSubmissionConsumerName, event.ID, event.OccurredAt.UTC()); err != nil {
		return fmt.Errorf("record vendor work submission inbox: %w", err)
	}
	return nil
}

var _ interface {
	Publish(context.Context, workflowruntime.OutboxEvent) error
} = (*VendorWorkConsumer)(nil)

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const collectionSubmissionConsumer = "monitoring-collection-submission"

type collectionInbox interface {
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type collectionEvidenceReader interface {
	GetRequest(context.Context, string, string) (evidence.Request, error)
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
}

type collectionRepository interface {
	Repository
	CollectionCycleRepository
}

type CollectionConsumer struct {
	Inbox      collectionInbox
	Repository collectionRepository
	Evidence   collectionEvidenceReader
	Now        func() time.Time
	NewID      func() (string, error)
}

type collectionSubmissionPayload struct {
	SubmissionID string `json:"submission_id"`
}

func (c *CollectionConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != "EVIDENCE_REQUEST" || event.EventType != "EvidenceResponseSubmitted" {
		return nil
	}
	if c == nil || c.Inbox == nil || c.Repository == nil || c.Evidence == nil {
		return fmt.Errorf("collection submission consumer is not configured")
	}
	processed, err := c.Inbox.InboxProcessed(ctx, event.TenantID, collectionSubmissionConsumer, event.ID)
	if err != nil {
		return fmt.Errorf("check collection submission inbox: %w", err)
	}
	if processed {
		return nil
	}
	var payload collectionSubmissionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode collection submission event: %w", err)
	}
	payload.SubmissionID = strings.TrimSpace(payload.SubmissionID)
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.TenantID) == "" || strings.TrimSpace(event.AggregateID) == "" || payload.SubmissionID == "" {
		return fmt.Errorf("collection submission event identifiers are required")
	}
	request, err := c.Evidence.GetRequest(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return fmt.Errorf("load collection request: %w", err)
	}
	submission, err := c.Evidence.GetSubmission(ctx, event.TenantID, payload.SubmissionID)
	if err != nil {
		return fmt.Errorf("load collection submission: %w", err)
	}
	if submission.RequestID != request.ID || request.TenantID != event.TenantID {
		return fmt.Errorf("collection submission does not match the event request")
	}
	if request.Origin.Type != evidence.OriginMonitoringCollection {
		return c.recordInbox(ctx, event)
	}
	check, err := currentCheckForCollection(ctx, c.Repository, event.TenantID, request.SubjectID, request.Origin.ID)
	if err != nil {
		return err
	}
	now := c.currentTime()
	if check.Status == LifecyclePaused || check.Status == LifecycleRetired {
		if _, err := c.Repository.CancelCollectionCyclesByCheck(ctx, event.TenantID, check.ID, now); err != nil {
			return fmt.Errorf("cancel inactive collection cycles: %w", err)
		}
		return c.recordInbox(ctx, event)
	}
	if check.Status != LifecycleActive || !check.IsCurrent || check.InputKind != InputForm || check.CollectionPolicy == nil || request.SubjectType != "PROGRAM" || request.SubjectID != check.ProgramID {
		return fmt.Errorf("collection request does not match an active Program form check")
	}
	policy, err := normalizeCollectionPolicy(check.CollectionPolicy)
	if err != nil {
		return err
	}
	route, delivery, err := collectionRoute(request)
	if err != nil {
		return err
	}
	expires, opens, _ := CollectionDates(submission.SubmittedAt, policy)
	if expires.IsZero() || opens.IsZero() {
		return fmt.Errorf("calculate collection renewal dates")
	}
	if _, err := c.Repository.CompleteCollectionCyclesBeforeSequence(ctx, event.TenantID, check.ID, request.Origin.Version, now); err != nil {
		return fmt.Errorf("complete prior collection cycles: %w", err)
	}
	cycleID, err := c.nextID()
	if err != nil {
		return fmt.Errorf("create collection cycle id: %w", err)
	}
	submittedAt := submission.SubmittedAt.UTC()
	nextAction := opens
	cycle := CollectionCycle{
		ID: cycleID, TenantID: event.TenantID, ProgramID: check.ProgramID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		Sequence: request.Origin.Version, Policy: policy, CurrentRequestID: request.ID, PredecessorRequestID: request.PredecessorRequestID,
		LatestSubmissionID: submission.ID, LatestSubmittedAt: &submittedAt, LatestRespondentLabel: strings.TrimSpace(submission.SubmittedBy), ExpiresAt: expires, RenewalOpensAt: opens, NextActionAt: &nextAction,
		Recipient: route, DeliveryState: delivery, State: CycleScheduled, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := c.Repository.UpsertCollectionCycle(ctx, cycle); err != nil {
		return fmt.Errorf("project collection cycle: %w", err)
	}
	return c.recordInbox(ctx, event)
}

func (c *CollectionConsumer) recordInbox(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if _, err := c.Inbox.RecordInbox(ctx, event.TenantID, collectionSubmissionConsumer, event.ID, c.currentTime()); err != nil {
		return fmt.Errorf("record collection submission inbox: %w", err)
	}
	return nil
}

func (c *CollectionConsumer) currentTime() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *CollectionConsumer) nextID() (string, error) {
	if c.NewID != nil {
		return c.NewID()
	}
	return id.NewUUIDv7()
}

func currentCheckForCollection(ctx context.Context, repo Repository, tenant, programID, checkID string) (MonitoringCheck, error) {
	checks, err := repo.ListCheckRevisions(ctx, tenant, programID, 500)
	if err != nil {
		return MonitoringCheck{}, fmt.Errorf("list collection check revisions: %w", err)
	}
	var latest *MonitoringCheck
	for _, check := range checks {
		if check.ID != checkID {
			continue
		}
		if check.IsCurrent {
			return check, nil
		}
		if latest == nil || check.Version > latest.Version {
			candidate := check
			latest = &candidate
		}
	}
	if latest != nil {
		return *latest, nil
	}
	return MonitoringCheck{}, fmt.Errorf("collection check %s is unavailable", checkID)
}

func collectionRoute(request evidence.Request) (RecipientRoute, DeliveryState, error) {
	switch request.Recipient.Type {
	case evidence.RecipientInternalPrincipal:
		if strings.TrimSpace(request.Recipient.PrincipalID) == "" {
			return RecipientRoute{}, "", fmt.Errorf("collection request has no current internal recipient")
		}
		return RecipientRoute{Type: RouteInternalPrincipal, PrincipalID: request.Recipient.PrincipalID}, DeliveryAssigned, nil
	case evidence.RecipientExternalAudience:
		return RecipientRoute{}, "", fmt.Errorf("collection request has no opaque external recipient route")
	default:
		return RecipientRoute{}, "", fmt.Errorf("collection request recipient route is unavailable")
	}
}

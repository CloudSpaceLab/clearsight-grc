package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

var ErrAssignmentNotificationRetry = errors.New("staff assignment notification should be retried")

const (
	assignmentNotificationDelivered          = "DELIVERED"
	assignmentNotificationContactUnavailable = "CONTACT_UNAVAILABLE"
	assignmentNotificationSuperseded         = "ASSIGNMENT_SUPERSEDED"
	assignmentNotificationRecipientRejected  = "RECIPIENT_REJECTED"
	assignmentNotificationPermanentFailure   = "PERMANENT_FAILURE"
	assignmentNotificationTemporaryFailure   = "TEMPORARY_FAILURE"
	assignmentNotificationDeliveryStarted    = "DELIVERY_STARTED"

	matterOwnerNotificationKind     = "MATTER_OWNER_ASSIGNED"
	actionPerformerNotificationKind = "ACTION_PERFORMER_ASSIGNED"
)

type assignmentNotificationEvent struct {
	NotificationKind string
	PrincipalID      string
	PreviousOwnerID  string
	ActionID         string
}

type assignmentNotificationContext struct {
	LegalEntityID      string
	BankName           string
	RecipientName      string
	RecipientAddress   string
	CurrentPrincipalID string
	MatterID           string
	MatterTitle        string
	WorkTitle          string
	DueAt              time.Time
}

type assignmentNotificationRecord struct {
	Status               string
	FailureCode          string
	ProviderMessageID    string
	RecipientFingerprint []byte
	AttemptedAt          time.Time
	DeliveredAt          *time.Time
}

type assignmentNotificationRepository interface {
	LoadAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) (assignmentNotificationContext, error)
	GetAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) (assignmentNotificationRecord, bool, error)
	ClaimAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent, assignmentNotificationRecord) (bool, error)
	RecordAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent, assignmentNotificationRecord) error
}

type assignmentNotificationDelivery interface {
	DeliverGoverned(context.Context, evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error)
}

type AssignmentNotificationConsumer struct {
	repository assignmentNotificationRepository
	delivery   assignmentNotificationDelivery
	baseURL    string
	now        func() time.Time
}

func NewAssignmentNotificationConsumer(repository assignmentNotificationRepository, delivery assignmentNotificationDelivery, applicationBaseURL string) (*AssignmentNotificationConsumer, error) {
	applicationBaseURL = strings.TrimSpace(applicationBaseURL)
	parsed, err := url.Parse(applicationBaseURL)
	if repository == nil || delivery == nil || err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ContainsAny(applicationBaseURL, "\r\n") {
		return nil, fmt.Errorf("invalid staff assignment notification configuration")
	}
	return &AssignmentNotificationConsumer{repository: repository, delivery: delivery, baseURL: strings.TrimRight(applicationBaseURL, "/"), now: time.Now}, nil
}

func (consumer *AssignmentNotificationConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	assignment, relevant, err := decodeAssignmentNotificationEvent(event)
	if err != nil {
		return err
	}
	if !relevant {
		return nil
	}
	prior, found, err := consumer.repository.GetAssignmentNotification(ctx, event, assignment)
	if err != nil {
		return err
	}
	if found && assignmentNotificationFinal(prior.Status) {
		return nil
	}
	notificationContext, err := consumer.repository.LoadAssignmentNotification(ctx, event, assignment)
	if err != nil {
		return err
	}
	now := consumer.currentTime()
	if strings.TrimSpace(notificationContext.CurrentPrincipalID) != assignment.PrincipalID {
		return consumer.repository.RecordAssignmentNotification(ctx, event, assignment, assignmentNotificationRecord{Status: assignmentNotificationSuperseded, AttemptedAt: now})
	}
	address := strings.TrimSpace(notificationContext.RecipientAddress)
	if !validStaffMailbox(address) {
		return consumer.repository.RecordAssignmentNotification(ctx, event, assignment, assignmentNotificationRecord{Status: assignmentNotificationContactUnavailable, AttemptedAt: now})
	}

	responsibility := "ACCOUNTABLE_OWNER"
	if assignment.NotificationKind == actionPerformerNotificationKind {
		responsibility = "PERFORMER"
	}
	request, err := evidence.BuildOperationalNotificationRequest(address, evidence.OperationalNotificationContext{
		BankName: notificationContext.BankName, RecipientName: notificationContext.RecipientName,
		MatterTitle: notificationContext.MatterTitle, WorkTitle: notificationContext.WorkTitle,
		Responsibility: responsibility, DueAt: notificationContext.DueAt,
		IssueURL: consumer.baseURL + "/#work/matters/" + url.PathEscape(notificationContext.MatterID),
	})
	if err != nil {
		return fmt.Errorf("build staff assignment notification: %w", err)
	}
	// SMTP has no portable provider-side idempotency key. Persist the claim before
	// delivery and treat a stranded claim as an unknown outcome so an outbox retry
	// cannot send a second message after a process crash.
	claimed, err := consumer.repository.ClaimAssignmentNotification(ctx, event, assignment, assignmentNotificationRecord{
		Status: assignmentNotificationDeliveryStarted, RecipientFingerprint: assignmentRecipientFingerprint(address), AttemptedAt: now,
	})
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	receipt, deliveryErr := consumer.delivery.DeliverGoverned(ctx, request)
	record := assignmentNotificationRecord{
		Status: assignmentNotificationStatus(receipt, deliveryErr), FailureCode: string(receipt.FailureCode),
		ProviderMessageID: receipt.ProviderMessageID, RecipientFingerprint: assignmentRecipientFingerprint(address),
		AttemptedAt: now, DeliveredAt: receipt.DeliveredAt,
	}
	if err := consumer.repository.RecordAssignmentNotification(ctx, event, assignment, record); err != nil {
		return err
	}
	if record.Status == assignmentNotificationTemporaryFailure {
		return ErrAssignmentNotificationRetry
	}
	return nil
}

func validStaffMailbox(address string) bool {
	parsed, err := mail.ParseAddress(strings.TrimSpace(address))
	return err == nil && strings.EqualFold(parsed.Address, strings.TrimSpace(address))
}

func decodeAssignmentNotificationEvent(event workflowruntime.OutboxEvent) (assignmentNotificationEvent, bool, error) {
	if event.AggregateType != "MATTER" || (event.EventType != continuity.EventMatterOwnerChanged && event.EventType != continuity.EventActionAssigned) {
		return assignmentNotificationEvent{}, false, nil
	}
	var envelope struct {
		Matter struct {
			ID string `json:"id"`
		} `json:"matter"`
		Action struct {
			ID       string `json:"id"`
			MatterID string `json:"matter_id"`
		} `json:"action"`
		PreviousOwnerID  string `json:"previous_owner_principal_id"`
		OwnerPrincipalID string `json:"owner_principal_id"`
	}
	if err := json.Unmarshal(event.Payload, &envelope); err != nil {
		return assignmentNotificationEvent{}, true, fmt.Errorf("decode staff assignment event: %w", err)
	}
	assignment := assignmentNotificationEvent{PrincipalID: strings.TrimSpace(envelope.OwnerPrincipalID), PreviousOwnerID: strings.TrimSpace(envelope.PreviousOwnerID)}
	if event.EventType == continuity.EventMatterOwnerChanged {
		assignment.NotificationKind = matterOwnerNotificationKind
		if strings.TrimSpace(envelope.Matter.ID) != strings.TrimSpace(event.AggregateID) {
			return assignmentNotificationEvent{}, true, fmt.Errorf("staff assignment event Matter does not match aggregate")
		}
	} else {
		assignment.NotificationKind = actionPerformerNotificationKind
		assignment.ActionID = strings.TrimSpace(envelope.Action.ID)
		if strings.TrimSpace(envelope.Action.MatterID) != strings.TrimSpace(event.AggregateID) || assignment.ActionID == "" {
			return assignmentNotificationEvent{}, true, fmt.Errorf("staff assignment event Action does not match aggregate")
		}
	}
	if assignment.PrincipalID == "" || assignment.PrincipalID == assignment.PreviousOwnerID {
		return assignmentNotificationEvent{}, true, fmt.Errorf("staff assignment event does not identify a changed owner")
	}
	return assignment, true, nil
}

func assignmentNotificationStatus(receipt evidence.InvitationDeliveryReceipt, deliveryErr error) string {
	if deliveryErr != nil || receipt.Status == evidence.InvitationLinkCreatedEmailNotSent || receipt.FailureCode == evidence.InvitationFailureTemporary || receipt.FailureCode == evidence.InvitationFailureProviderError {
		return assignmentNotificationTemporaryFailure
	}
	if receipt.Status == evidence.InvitationDelivered {
		return assignmentNotificationDelivered
	}
	if receipt.FailureCode == evidence.InvitationFailureRecipientRejected {
		return assignmentNotificationRecipientRejected
	}
	return assignmentNotificationPermanentFailure
}

func assignmentNotificationFinal(status string) bool {
	return status != "" && status != assignmentNotificationTemporaryFailure
}

func assignmentRecipientFingerprint(address string) []byte {
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(address))))
	return digest[:]
}

func (consumer *AssignmentNotificationConsumer) currentTime() time.Time {
	if consumer.now == nil {
		return time.Now().UTC()
	}
	return consumer.now().UTC()
}

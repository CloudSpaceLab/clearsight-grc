package attention

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const criticalEmailDeliveryClass = "ATTENTION_CRITICAL"

type CriticalEmailContext struct {
	LegalEntityID         string
	BrandName             string
	RecipientName         string
	RecipientAddress      string
	CurrentNoticeSequence int
	StillEligible         bool
}

type EmailDeliveryRecord struct {
	Status               string
	RecipientFingerprint []byte
	FailureCode          string
	ProviderMessageID    string
	AttemptedAt          time.Time
	DeliveredAt          *time.Time
}

type CriticalEmailRepository interface {
	LoadCriticalEmailContext(context.Context, workflowruntime.OutboxEvent, Intent) (CriticalEmailContext, error)
	ClaimCriticalEmail(context.Context, workflowruntime.OutboxEvent, Intent, EmailDeliveryRecord) (bool, error)
	RecordCriticalEmail(context.Context, workflowruntime.OutboxEvent, Intent, EmailDeliveryRecord) error
}

type governedEmailDelivery interface {
	DeliverGoverned(context.Context, evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error)
}

type CriticalEmailConsumer struct {
	repository     CriticalEmailRepository
	delivery       governedEmailDelivery
	applicationURL string
	now            func() time.Time
}

func NewCriticalEmailConsumer(repository CriticalEmailRepository, delivery governedEmailDelivery, applicationURL string) *CriticalEmailConsumer {
	return &CriticalEmailConsumer{
		repository:     repository,
		delivery:       delivery,
		applicationURL: strings.TrimRight(strings.TrimSpace(applicationURL), "/"),
		now:            time.Now,
	}
}

func (c *CriticalEmailConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	intent, relevant, err := DecodeIntent(event)
	if err != nil || !relevant || !criticalEmailIntent(event.EventType, intent) {
		return err
	}
	if c == nil || c.repository == nil || c.delivery == nil || c.applicationURL == "" {
		return fmt.Errorf("critical attention email delivery is unavailable")
	}
	value, err := c.repository.LoadCriticalEmailContext(ctx, event, intent)
	if err != nil {
		return err
	}
	if value.CurrentNoticeSequence != intent.NoticeSequence {
		return nil
	}
	now := c.now().UTC()
	if !value.StillEligible {
		claimed, err := c.repository.ClaimCriticalEmail(ctx, event, intent, EmailDeliveryRecord{
			Status: "NOTICE_SUPERSEDED", AttemptedAt: now,
		})
		if err != nil || !claimed {
			return err
		}
		return nil
	}
	address := strings.TrimSpace(value.RecipientAddress)
	if !canonicalMailbox(address) {
		claimed, err := c.repository.ClaimCriticalEmail(ctx, event, intent, EmailDeliveryRecord{
			Status: "CONTACT_UNAVAILABLE", AttemptedAt: now,
		})
		if err != nil || !claimed {
			return err
		}
		return nil
	}
	fingerprint := emailFingerprint(address)
	claimed, err := c.repository.ClaimCriticalEmail(ctx, event, intent, EmailDeliveryRecord{
		Status: "DELIVERY_STARTED", RecipientFingerprint: fingerprint, AttemptedAt: now,
	})
	if err != nil || !claimed {
		return err
	}

	request, err := evidence.BuildAttentionNotificationRequest(address, evidence.AttentionNotificationContext{
		BrandName:       value.BrandName,
		RecipientName:   value.RecipientName,
		ConditionLabel:  conditionEmailLabel(intent.Condition),
		TransitionLabel: transitionEmailLabel(event.EventType),
		RecordURL:       c.recordURL(intent),
	})
	if err != nil {
		_ = c.repository.RecordCriticalEmail(ctx, event, intent, EmailDeliveryRecord{
			Status: "PERMANENT_FAILURE", RecipientFingerprint: fingerprint,
			FailureCode: "INVALID_MESSAGE", AttemptedAt: now,
		})
		return nil
	}

	receipt, deliveryErr := c.delivery.DeliverGoverned(ctx, request)
	record := emailRecordFromReceipt(receipt, deliveryErr, fingerprint, now)
	if err := c.repository.RecordCriticalEmail(ctx, event, intent, record); err != nil {
		return err
	}
	if record.Status == "TEMPORARY_FAILURE" {
		return errors.New("critical attention email delivery should be retried")
	}
	return nil
}

func criticalEmailIntent(eventType string, intent Intent) bool {
	if strings.ToUpper(strings.TrimSpace(intent.ConditionState)) != "CRITICAL" {
		return false
	}
	return eventType == EventEpisodeOpened || eventType == EventEpisodeWorsened
}

func (c *CriticalEmailConsumer) recordURL(intent Intent) string {
	switch intent.SubjectType {
	case "RISK":
		return c.applicationURL + "/#risks/" + intent.SubjectID
	case "LOSS":
		return c.applicationURL + "/#losses/" + intent.SubjectID
	default:
		return ""
	}
}

func conditionEmailLabel(condition string) string {
	switch condition {
	case "indicator_breaches":
		return "Risk indicator"
	case "risks_outside_appetite":
		return "Risk appetite"
	case "assurance_failures":
		return "Assurance"
	case "losses_without_issue":
		return "Operational loss"
	default:
		return "Risk"
	}
}

func transitionEmailLabel(eventType string) string {
	if eventType == EventEpisodeWorsened {
		return "Worsened"
	}
	return "Opened"
}

func canonicalMailbox(address string) bool {
	parsed, err := mail.ParseAddress(address)
	return err == nil && parsed.Address == address
}

func emailFingerprint(address string) []byte {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(address))))
	return sum[:]
}

func emailRecordFromReceipt(receipt evidence.InvitationDeliveryReceipt, deliveryErr error, fingerprint []byte, attemptedAt time.Time) EmailDeliveryRecord {
	record := EmailDeliveryRecord{
		RecipientFingerprint: fingerprint,
		FailureCode:          strings.TrimSpace(string(receipt.FailureCode)),
		ProviderMessageID:    strings.TrimSpace(receipt.ProviderMessageID),
		AttemptedAt:          attemptedAt,
		DeliveredAt:          receipt.DeliveredAt,
	}
	switch {
	case receipt.FailureCode == evidence.InvitationFailureOutcomeUnknown:
		record.Status = "DELIVERY_OUTCOME_UNKNOWN"
	case receipt.Status == evidence.InvitationDelivered && deliveryErr == nil:
		record.Status = "DELIVERED"
	case receipt.FailureCode == evidence.InvitationFailureRecipientRejected:
		record.Status = "RECIPIENT_REJECTED"
	case receipt.FailureCode == evidence.InvitationFailurePermanent || receipt.FailureCode == evidence.InvitationFailureInvalidReceipt:
		record.Status = "PERMANENT_FAILURE"
	default:
		record.Status = "TEMPORARY_FAILURE"
	}
	return record
}

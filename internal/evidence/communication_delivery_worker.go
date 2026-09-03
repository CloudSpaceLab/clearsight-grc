package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

var ErrCommunicationDeliveryRetry = errors.New("communication delivery should be retried")

type CommunicationDeliveryAttempt struct {
	Status          string
	FailureCode     InvitationDeliveryFailureCode
	ProviderMessage string
	AttemptedAt     time.Time
}

func communicationDeliveryAttemptStatus(receipt InvitationDeliveryReceipt, failureCode string) string {
	if receipt.Status == InvitationDelivered {
		return "DELIVERED"
	}
	switch failureCode {
	case "DISTRIBUTION_NOT_DELIVERABLE", "RECIPIENT_NOT_DELIVERABLE", "SUPERSEDED_COMMUNICATION", "WORKFLOW_OWNED_COMMUNICATION":
		return "SKIPPED"
	default:
		return "FAILED"
	}
}

type communicationDeliveryRecipient struct {
	DistributionRecipient
	ProtectedAddress protectedRecipientAddress
}

type communicationDeliveryBundle struct {
	Distribution FormDistribution
	OriginType   string
	Recipients   []communicationDeliveryRecipient
}

type communicationDeliveryRepository interface {
	LoadCommunicationDelivery(context.Context, string, string) (communicationDeliveryBundle, error)
	GetCommunicationDeliveryAttempt(context.Context, string, string, CommunicationAction) (CommunicationDeliveryAttempt, bool, error)
	RecordCommunicationDeliveryAttempt(context.Context, workflowruntime.OutboxEvent, communicationDeliveryBundle, communicationDeliveryRecipient, CommunicationTemplate, InvitationDeliveryReceipt, string, time.Time) error
}

type CommunicationDeliveryWorker struct {
	repository     communicationDeliveryRepository
	communications *CommunicationService
	access         *DistributionAccessService
	delivery       *InvitationDeliveryService
	captureBaseURL string
	now            func() time.Time
}

func NewCommunicationDeliveryWorker(repository communicationDeliveryRepository, communications *CommunicationService, access *DistributionAccessService, delivery *InvitationDeliveryService, captureBaseURL string) (*CommunicationDeliveryWorker, error) {
	captureBaseURL = strings.TrimSpace(captureBaseURL)
	if repository == nil || communications == nil || access == nil || delivery == nil || validateCommunicationCaptureBaseURL(captureBaseURL) != nil {
		return nil, ErrCommunicationUnavailable
	}
	return &CommunicationDeliveryWorker{
		repository: repository, communications: communications, access: access, delivery: delivery,
		captureBaseURL: captureBaseURL, now: time.Now,
	}, nil
}

func (worker *CommunicationDeliveryWorker) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if worker == nil || worker.repository == nil || worker.communications == nil || worker.access == nil || worker.delivery == nil {
		return ErrCommunicationUnavailable
	}
	action, ok := communicationActionForOutboxEvent(event)
	if !ok {
		return nil
	}
	bundle, err := worker.repository.LoadCommunicationDelivery(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return err
	}
	now := worker.currentTime()
	if communicationOwnedByOriginWorkflow(bundle.OriginType) {
		return worker.skipRecipients(ctx, event, bundle, action, "WORKFLOW_OWNED_COMMUNICATION", now)
	}
	if communicationSecureLinkEventSuperseded(event, bundle.Distribution, action) {
		return worker.skipRecipients(ctx, event, bundle, action, "SUPERSEDED_COMMUNICATION", now)
	}
	if !communicationActionDeliverable(bundle.Distribution, action, now) {
		return worker.skipRecipients(ctx, event, bundle, action, "DISTRIBUTION_NOT_DELIVERABLE", now)
	}
	template, err := worker.communications.ResolveTemplate(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, action, "", now)
	if err != nil {
		return fmt.Errorf("%w: active %s template unavailable", ErrCommunicationDeliveryRetry, action)
	}
	profile, err := worker.communications.effectiveProfile(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, now)
	if err != nil {
		return fmt.Errorf("%w: communication profile unavailable", ErrCommunicationDeliveryRetry)
	}

	var retryErrors []error
	var sharedRoute *IssuedAccessRoute
	for _, recipient := range bundle.Recipients {
		if !communicationRecipientDeliverable(recipient, action) {
			continue
		}
		prior, found, err := worker.repository.GetCommunicationDeliveryAttempt(ctx, event.ID, recipient.ID, action)
		if err != nil {
			return err
		}
		if found && communicationAttemptFinal(prior) {
			continue
		}

		address, err := worker.access.revealer.RevealRecipientAddress(ctx, bundle.Distribution.TenantID, bundle.Distribution.ID, recipient.ID, recipient.ProtectedAddress)
		if err != nil {
			receipt := invitationFailureReceipt(recipient.AudienceHint, InvitationFailurePermanent)
			if recordErr := worker.repository.RecordCommunicationDeliveryAttempt(ctx, event, bundle, recipient, template, receipt, "ADDRESS_UNAVAILABLE", now); recordErr != nil {
				return recordErr
			}
			continue
		}

		link := ""
		linkExpiry := ""
		if recipient.Role == RecipientTo && communicationRequiresSecureLink(action) {
			issued, issueErr := worker.deliveryRoute(ctx, bundle, recipient, &sharedRoute)
			if issueErr != nil {
				retryErrors = append(retryErrors, issueErr)
				continue
			}
			link, issueErr = buildCommunicationAccessLink(worker.captureBaseURL, issued.Selector)
			if issueErr != nil {
				retryErrors = append(retryErrors, issueErr)
				continue
			}
			linkExpiry = issued.ExpiresAt.UTC().Format(time.RFC3339)
		}

		messageContext := CommunicationContext{
			RecipientName:      communicationRecipientName(recipient),
			BankName:           profile.BankName,
			FormTitle:          bundle.Distribution.Title,
			TaskSummary:        bundle.Distribution.Purpose,
			DueTime:            bundle.Distribution.Deadline.UTC().Format(time.RFC3339),
			LinkExpiry:         linkExpiry,
			AccessInstructions: communicationAccessInstructions(bundle.Distribution.AccessPolicy, recipient.Role),
			SupportContact:     profile.SupportContact,
			SecureFormLink:     ProtectCommunicationString(link),
		}
		var rendered RenderedMessage
		if recipient.Role == RecipientCC {
			rendered, err = renderCommunicationWithoutResponseRoute(template, messageContext)
		} else {
			rendered, err = RenderCommunication(template, messageContext)
		}
		if err != nil {
			return fmt.Errorf("%w: render %s", ErrCommunicationDeliveryRetry, action)
		}

		receipt, deliveryErr := worker.delivery.DeliverGoverned(ctx, InvitationDeliveryRequest{
			RecipientAddress: address,
			InvitationLink:   link,
			Subject:          rendered.Subject.value,
			PlainText:        rendered.PlainText.value,
			HTML:             rendered.HTML.value,
		})
		if deliveryErr != nil {
			receipt = invitationFailureReceipt(recipient.AudienceHint, InvitationFailureProviderError)
		}
		failureCode := string(receipt.FailureCode)
		if receipt.Status == InvitationLinkCreatedEmailNotSent {
			receipt = invitationFailureReceipt(recipient.AudienceHint, InvitationFailureTemporary)
			failureCode = string(InvitationFailureTemporary)
		}
		if err := worker.repository.RecordCommunicationDeliveryAttempt(ctx, event, bundle, recipient, template, receipt, failureCode, now); err != nil {
			return err
		}
		if deliveryErr != nil || receipt.FailureCode == InvitationFailureTemporary || receipt.FailureCode == InvitationFailureProviderError {
			retryErrors = append(retryErrors, ErrCommunicationDeliveryRetry)
		}
	}
	if len(retryErrors) != 0 {
		return errors.Join(retryErrors...)
	}
	return nil
}

func communicationOwnedByOriginWorkflow(originType string) bool {
	switch strings.ToUpper(strings.TrimSpace(originType)) {
	case "THIRD_PARTY_ASSESSMENT", "THIRD_PARTY_WORK":
		return true
	default:
		return false
	}
}

func communicationSecureLinkEventSuperseded(event workflowruntime.OutboxEvent, distribution FormDistribution, action CommunicationAction) bool {
	if !communicationRequiresSecureLink(action) || distribution.Version < 2 {
		return false
	}
	var payload struct {
		Version int64 `json:"version"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil || payload.Version < 1 {
		return false
	}
	return payload.Version < distribution.Version
}

func (worker *CommunicationDeliveryWorker) deliveryRoute(ctx context.Context, bundle communicationDeliveryBundle, recipient communicationDeliveryRecipient, shared **IssuedAccessRoute) (IssuedAccessRoute, error) {
	if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP && *shared != nil {
		return **shared, nil
	}
	recipientID := recipient.ID
	hint := recipient.AudienceHint
	if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP {
		recipientID = ""
		hint = ""
	}

	active, err := worker.access.store.ListActiveAccessRoutes(
		ctx,
		bundle.Distribution.TenantID,
		bundle.Distribution.LegalEntityID,
		bundle.Distribution.ID,
		worker.currentTime(),
	)
	if err != nil {
		return IssuedAccessRoute{}, fmt.Errorf("%w: inspect access routes", ErrCommunicationDeliveryRetry)
	}
	for _, current := range active {
		if current.Policy != bundle.Distribution.AccessPolicy || current.RecipientID != recipientID {
			continue
		}
		issued, rotateErr := worker.access.RotateDistributionAccessRoute(
			ctx,
			bundle.Distribution.TenantID,
			bundle.Distribution.LegalEntityID,
			bundle.Distribution.ID,
			current.ID,
			bundle.Distribution.CreatedBy,
		)
		if rotateErr != nil {
			return IssuedAccessRoute{}, fmt.Errorf("%w: rotate access route", ErrCommunicationDeliveryRetry)
		}
		if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP {
			copy := issued
			*shared = &copy
		}
		return issued, nil
	}

	route, issued, err := worker.access.engine.IssueRoute(AccessRouteInput{
		TenantID: bundle.Distribution.TenantID, LegalEntityID: bundle.Distribution.LegalEntityID,
		DistributionID: bundle.Distribution.ID, RecipientID: recipientID,
		Policy: bundle.Distribution.AccessPolicy, AudienceHint: hint,
		RouteExpiresAt: bundle.Distribution.RouteExpiresAt, Deadline: bundle.Distribution.Deadline,
		CreatedBy: bundle.Distribution.CreatedBy,
	})
	if err != nil {
		return IssuedAccessRoute{}, fmt.Errorf("%w: issue access route", ErrCommunicationDeliveryRetry)
	}
	if err := worker.access.store.CreateAccessRoutes(ctx, []AccessRoute{route}); err != nil {
		return IssuedAccessRoute{}, fmt.Errorf("%w: persist access route", ErrCommunicationDeliveryRetry)
	}
	if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP {
		copy := issued
		*shared = &copy
	}
	return issued, nil
}

func (worker *CommunicationDeliveryWorker) skipRecipients(ctx context.Context, event workflowruntime.OutboxEvent, bundle communicationDeliveryBundle, action CommunicationAction, reason string, now time.Time) error {
	template := CommunicationTemplate{TenantID: bundle.Distribution.TenantID, LegalEntityID: bundle.Distribution.LegalEntityID, Action: action}
	for _, recipient := range bundle.Recipients {
		if !communicationRecipientDeliverable(recipient, action) {
			continue
		}
		prior, found, err := worker.repository.GetCommunicationDeliveryAttempt(ctx, event.ID, recipient.ID, action)
		if err != nil {
			return err
		}
		if found && communicationAttemptFinal(prior) {
			continue
		}
		receipt := InvitationDeliveryReceipt{Status: InvitationDeliveryFailed, RecipientHint: recipient.AudienceHint}
		if err := worker.repository.RecordCommunicationDeliveryAttempt(ctx, event, bundle, recipient, template, receipt, reason, now); err != nil {
			return err
		}
	}
	return nil
}

func (worker *CommunicationDeliveryWorker) currentTime() time.Time {
	if worker != nil && worker.now != nil {
		return worker.now().UTC()
	}
	return time.Now().UTC()
}

func communicationAttemptFinal(attempt CommunicationDeliveryAttempt) bool {
	if attempt.Status == "DELIVERED" || attempt.Status == "SKIPPED" {
		return true
	}
	if attempt.Status != "FAILED" {
		return false
	}
	return attempt.FailureCode != InvitationFailureTemporary && attempt.FailureCode != InvitationFailureProviderError
}

var _ workflowruntime.Publisher = (*CommunicationDeliveryWorker)(nil)

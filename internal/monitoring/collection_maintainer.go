package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

const CollectionRenewalWorkClass = "monitoring-collection-renewal"

var ErrDeliveryUnavailable = errors.New("collection delivery is unavailable")
var ErrRenewalBlocked = errors.New("collection renewal is blocked")

type DeliveryReceipt struct {
	State     DeliveryState
	Reference string
}

type CollectionDispatcher interface {
	ValidateRoute(context.Context, string, RecipientRoute) error
	DispatchRequest(context.Context, evidence.Request, RecipientRoute) (DeliveryReceipt, error)
	DispatchReminder(context.Context, evidence.Request, RecipientRoute, int) (DeliveryReceipt, error)
}

type collectionRequestService interface {
	CreateRequest(context.Context, evidence.CreateRequestInput) (evidence.Request, error)
	GetRequest(context.Context, string, string) (evidence.Request, error)
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
}

type externalCollectionRequestCreator interface {
	CreateExternalCollectionRequest(context.Context, evidence.CreateRequestInput, RecipientRoute) (evidence.Request, error)
}

// CanonicalCollectionDispatcher handles internal assignment directly and
// delegates external delivery only when a deployment adapter is configured.
type CanonicalCollectionDispatcher struct {
	Requests interface {
		GetRequest(context.Context, string, string) (evidence.Request, error)
	}
	External CollectionDispatcher
}

func (d *CanonicalCollectionDispatcher) ValidateRoute(ctx context.Context, tenant string, route RecipientRoute) error {
	if err := validateRecipientRoute(route); err != nil {
		return err
	}
	if route.Type == RouteExternalContact {
		if d == nil || d.External == nil {
			return ErrDeliveryUnavailable
		}
		return d.External.ValidateRoute(ctx, tenant, route)
	}
	return nil
}

func (d *CanonicalCollectionDispatcher) DispatchRequest(ctx context.Context, request evidence.Request, route RecipientRoute) (DeliveryReceipt, error) {
	if route.Type == RouteExternalContact {
		if d == nil || d.External == nil {
			return DeliveryReceipt{}, ErrDeliveryUnavailable
		}
		return d.External.DispatchRequest(ctx, request, route)
	}
	if d == nil || d.Requests == nil {
		return DeliveryReceipt{}, ErrDeliveryUnavailable
	}
	current, err := d.Requests.GetRequest(ctx, request.TenantID, request.ID)
	if err != nil {
		return DeliveryReceipt{}, err
	}
	if current.Recipient.Type != evidence.RecipientInternalPrincipal || current.Recipient.State != evidence.RecipientStateAssigned || current.Recipient.PrincipalID != route.PrincipalID {
		return DeliveryReceipt{}, fmt.Errorf("internal collection assignment is not current")
	}
	return DeliveryReceipt{State: DeliveryAssigned, Reference: request.ID}, nil
}

func (d *CanonicalCollectionDispatcher) DispatchReminder(ctx context.Context, request evidence.Request, route RecipientRoute, reminder int) (DeliveryReceipt, error) {
	if reminder < 1 {
		return DeliveryReceipt{}, fmt.Errorf("collection reminder number is invalid")
	}
	if route.Type == RouteExternalContact {
		if d == nil || d.External == nil {
			return DeliveryReceipt{}, ErrDeliveryUnavailable
		}
		return d.External.DispatchReminder(ctx, request, route, reminder)
	}
	return d.DispatchRequest(ctx, request, route)
}

type CollectionMaintainer struct {
	Repository   collectionRepository
	Requests     collectionRequestService
	Dispatcher   CollectionDispatcher
	WorkerID     string
	Lease        time.Duration
	MaxAttempts  int
	afterRequest func() error
}

func (m *CollectionMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Repository == nil || m.Requests == nil || m.Dispatcher == nil || strings.TrimSpace(m.WorkerID) == "" {
		return 0, fmt.Errorf("collection renewal maintainer is not configured")
	}
	lease := m.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	maxAttempts := m.MaxAttempts
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	now = now.UTC()
	claims, err := m.Repository.ClaimDueCollectionCycles(ctx, m.WorkerID, now, lease, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, claim := range claims {
		processed++
		err := m.processClaim(ctx, claim, now)
		if err == nil {
			continue
		}
		if errors.Is(err, ErrDeliveryUnavailable) || errors.Is(err, ErrRenewalBlocked) {
			safeError := "Collection renewal is blocked because its current form or recipient route is unavailable."
			if errors.Is(err, ErrDeliveryUnavailable) {
				safeError = "External collection delivery is not configured for this recipient route."
			}
			if _, completeErr := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleBlocked, DeliveryState: DeliveryBlocked, SafeError: safeError, At: now}); completeErr != nil {
				return processed, completeErr
			}
			continue
		}
		retryAt := now.Add(time.Minute)
		if _, failErr := m.Repository.FailCollectionAction(ctx, claim, "Collection renewal could not be completed.", &retryAt, maxAttempts, now); failErr != nil {
			return processed, failErr
		}
	}
	return processed, nil
}

func (m *CollectionMaintainer) processClaim(ctx context.Context, claim CollectionCycle, now time.Time) error {
	request, err := m.Requests.GetRequest(ctx, claim.TenantID, claim.CurrentRequestID)
	if err != nil {
		return err
	}
	if request.Origin.Type != evidence.OriginMonitoringCollection || request.Origin.ID != claim.MonitoringCheckID {
		return fmt.Errorf("collection request origin does not match the cycle")
	}
	if request.Origin.Version == claim.Sequence {
		return m.openRenewal(ctx, claim, request, now)
	}
	if request.Origin.Version == claim.Sequence+1 {
		return m.sendReminderOrClose(ctx, claim, request, now)
	}
	return fmt.Errorf("collection request sequence does not match the cycle")
}

func (m *CollectionMaintainer) openRenewal(ctx context.Context, claim CollectionCycle, predecessor evidence.Request, now time.Time) error {
	if claim.LatestSubmittedAt == nil || claim.LatestSubmissionID == "" {
		return fmt.Errorf("collection cycle has no predecessor submission")
	}
	check, err := currentCheckForCollection(ctx, m.Repository, claim.TenantID, claim.ProgramID, claim.MonitoringCheckID)
	if err != nil {
		return err
	}
	if check.Status == LifecyclePaused || check.Status == LifecycleRetired {
		_, err := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleCancelled, At: now})
		return err
	}
	if check.Status != LifecycleActive || !check.IsCurrent || check.InputKind != InputForm || check.CollectionPolicy == nil {
		return ErrRenewalBlocked
	}
	form, err := m.Repository.FormRevision(ctx, claim.TenantID, predecessor.LegalEntityID, claim.ProgramID, check.FormTemplateID, check.FormTemplateVersion)
	if err != nil || form.Status != LifecycleActive || !form.IsCurrent {
		return ErrRenewalBlocked
	}
	if err := m.Dispatcher.ValidateRoute(ctx, claim.TenantID, claim.Recipient); err != nil {
		return err
	}
	submission, err := m.Requests.GetSubmission(ctx, claim.TenantID, claim.LatestSubmissionID)
	if err != nil || submission.RequestID != predecessor.ID {
		return fmt.Errorf("load predecessor collection submission: %w", err)
	}
	fields := evidenceFields(form.Fields)
	input := evidence.CreateRequestInput{
		TenantID: claim.TenantID, LegalEntityID: predecessor.LegalEntityID, SubjectType: "PROGRAM", SubjectID: claim.ProgramID, Title: form.Name, Purpose: form.Purpose,
		WhyYou: predecessor.WhyYou, Sensitivity: predecessor.Sensitivity, AudienceType: predecessor.AudienceType,
		EstimatedMinutes: estimateMinutes(len(fields)), Deadline: claim.ExpiresAt, KnownFacts: copyStringMap(predecessor.KnownFacts),
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: fields,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, CollectionPeriodStart: predecessor.CollectionPeriodStart, CollectionPeriodEnd: predecessor.CollectionPeriodEnd,
		Origin: evidence.RequestOrigin{Type: evidence.OriginMonitoringCollection, ID: check.ID, Version: claim.Sequence + 1}, PredecessorRequestID: predecessor.ID,
		PreviousResponses: evidence.BuildPreviousResponsePrefill(predecessor, submission, fields), CreatedBy: predecessor.CreatedBy,
	}
	var successor evidence.Request
	switch claim.Recipient.Type {
	case RouteInternalPrincipal:
		input.AudienceType = "INTERNAL"
		input.Recipient = evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: claim.Recipient.PrincipalID}
		successor, err = m.Requests.CreateRequest(ctx, input)
	case RouteExternalContact:
		creator, ok := m.Requests.(externalCollectionRequestCreator)
		if !ok {
			return ErrDeliveryUnavailable
		}
		successor, err = creator.CreateExternalCollectionRequest(ctx, input, claim.Recipient)
	default:
		return ErrDeliveryUnavailable
	}
	if err != nil {
		return err
	}
	if m.afterRequest != nil {
		if err := m.afterRequest(); err != nil {
			return err
		}
	}
	receipt, err := m.Dispatcher.DispatchRequest(ctx, successor, claim.Recipient)
	if err != nil {
		return err
	}
	if err := validateDeliveryReceipt(receipt, claim.Recipient); err != nil {
		return err
	}
	_, _, reminders := CollectionDates(*claim.LatestSubmittedAt, claim.Policy)
	nextAction := claim.ExpiresAt
	if len(reminders) > 0 {
		nextAction = reminders[0]
	}
	_, err = m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{
		State: CycleAwaitingResponse, CurrentRequestID: successor.ID, DeliveryState: receipt.State, DeliveryReference: receipt.Reference, NextActionAt: &nextAction, At: now,
	})
	return err
}

func (m *CollectionMaintainer) sendReminderOrClose(ctx context.Context, claim CollectionCycle, request evidence.Request, now time.Time) error {
	check, err := currentCheckForCollection(ctx, m.Repository, claim.TenantID, claim.ProgramID, claim.MonitoringCheckID)
	if err != nil {
		return err
	}
	if check.Status == LifecyclePaused || check.Status == LifecycleRetired {
		_, err := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleCancelled, At: now})
		return err
	}
	if check.Status != LifecycleActive || !check.IsCurrent {
		return ErrRenewalBlocked
	}
	if !now.Before(claim.ExpiresAt) || request.Status == evidence.RequestSubmitted || request.Status == evidence.RequestCancelled || request.Status == evidence.RequestExpired {
		_, err := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleComplete, At: now})
		return err
	}
	if request.Recipient.State != evidence.RecipientStateAssigned {
		_, err := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleBlocked, DeliveryState: DeliveryBlocked, SafeError: "The assigned respondent must be updated before reminders can continue.", At: now})
		return err
	}
	if err := m.Dispatcher.ValidateRoute(ctx, claim.TenantID, claim.Recipient); err != nil {
		return err
	}
	reminderNumber := claim.RemindersSent + 1
	if reminderNumber > claim.Policy.ReminderCount {
		_, err := m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{State: CycleComplete, At: now})
		return err
	}
	receipt, err := m.Dispatcher.DispatchReminder(ctx, request, claim.Recipient, reminderNumber)
	if err != nil {
		return err
	}
	if err := validateDeliveryReceipt(receipt, claim.Recipient); err != nil {
		return err
	}
	_, _, reminders := CollectionDates(*claim.LatestSubmittedAt, claim.Policy)
	nextAction := claim.ExpiresAt
	if reminderNumber < len(reminders) {
		nextAction = reminders[reminderNumber]
	}
	_, err = m.Repository.CompleteCollectionAction(ctx, claim, CollectionActionCompletion{
		State: CycleAwaitingResponse, DeliveryState: receipt.State, DeliveryReference: receipt.Reference, NextActionAt: &nextAction, RemindersSent: &reminderNumber, At: now,
	})
	return err
}

func validateDeliveryReceipt(receipt DeliveryReceipt, route RecipientRoute) error {
	switch route.Type {
	case RouteInternalPrincipal:
		if receipt.State != DeliveryAssigned {
			return fmt.Errorf("internal collection assignment was not confirmed")
		}
	case RouteExternalContact:
		if receipt.State != DeliveryDelivered || strings.TrimSpace(receipt.Reference) == "" {
			return fmt.Errorf("external collection delivery was not confirmed")
		}
	default:
		return ErrDeliveryUnavailable
	}
	return nil
}

func evidenceFields(fields []TemplateField) []evidence.Field {
	result := make([]evidence.Field, len(fields))
	for index, field := range fields {
		result[index] = evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring}
	}
	return result
}

func copyStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

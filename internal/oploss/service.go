package oploss

import (
	"context"
	"regexp"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	EventLossCreated          = "OperationalLossCreated"
	EventLossUpdated          = "OperationalLossUpdated"
	EventLossRecoveryRecorded = "OperationalLossRecoveryRecorded"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type ReferenceValidator func(context.Context, Scope, Loss) error

type Service struct {
	repo      Repository
	validate  ReferenceValidator
	Now       func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ConfigureReferenceValidator(validator ReferenceValidator) {
	s.validate = validator
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Loss, error) {
	if s == nil || s.repo == nil {
		return Loss{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Loss{}, err
	}
	now := s.now()
	loss := Loss{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		OrganizationScopeID: strings.TrimSpace(input.OrganizationScopeID),
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), Title: strings.TrimSpace(input.Title),
		EventType: input.EventType, Cause: strings.TrimSpace(input.Cause), Description: strings.TrimSpace(input.Description),
		GrossAmountMinor: input.GrossAmountMinor, Currency: strings.ToUpper(strings.TrimSpace(input.Currency)),
		OccurredAt: input.OccurredAt.UTC(), DiscoveredAt: input.DiscoveredAt.UTC(),
		RiskID: strings.TrimSpace(input.RiskID), MatterID: strings.TrimSpace(input.MatterID),
		OwnerPrincipalID: strings.TrimSpace(input.OwnerPrincipalID), Status: StatusActive,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if loss.OwnerPrincipalID == "" {
		loss.OwnerPrincipalID = strings.TrimSpace(input.ActorID)
	}
	if err := validateLoss(loss); err != nil {
		return Loss{}, err
	}
	if s.validate != nil {
		if err := s.validate(ctx, scope, loss); err != nil {
			return Loss{}, err
		}
	}
	loss.ID, err = newID()
	if err != nil {
		return Loss{}, err
	}
	event, err := newEvent(loss, EventLossCreated, input.ActorID, now)
	if err != nil {
		return Loss{}, err
	}
	return s.repo.Create(ctx, loss, event)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Loss, error) {
	if s == nil || s.repo == nil {
		return Loss{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Loss{}, err
	}
	current, err := s.repo.Get(ctx, scope, strings.TrimSpace(input.LossID))
	if err != nil {
		return Loss{}, err
	}
	if input.ExpectedVersion < 1 || current.Loss.Version != input.ExpectedVersion {
		return Loss{}, ErrVersionConflict
	}
	next := current.Loss
	next.OrganizationScopeID = strings.TrimSpace(input.OrganizationScopeID)
	next.Title = strings.TrimSpace(input.Title)
	next.EventType = input.EventType
	next.Cause = strings.TrimSpace(input.Cause)
	next.Description = strings.TrimSpace(input.Description)
	next.GrossAmountMinor = input.GrossAmountMinor
	next.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	next.OccurredAt = input.OccurredAt.UTC()
	next.DiscoveredAt = input.DiscoveredAt.UTC()
	next.RiskID = strings.TrimSpace(input.RiskID)
	next.MatterID = strings.TrimSpace(input.MatterID)
	if strings.TrimSpace(input.OwnerPrincipalID) != "" && strings.TrimSpace(input.OwnerPrincipalID) != current.Loss.OwnerPrincipalID {
		return Loss{}, ErrInvalid
	}
	next.OwnerPrincipalID = current.Loss.OwnerPrincipalID
	next.Status = input.Status
	if next.Status == "" {
		next.Status = current.Loss.Status
	}
	next.Version++
	next.UpdatedAt = s.now()
	if next.Currency != current.Loss.Currency {
		return Loss{}, ErrInvalid
	}
	if current.Totals.RecoveredAmountMinor > next.GrossAmountMinor {
		return Loss{}, ErrRecoveryLimit
	}
	if err := validateLoss(next); err != nil {
		return Loss{}, err
	}
	if s.validate != nil {
		if err := s.validate(ctx, scope, next); err != nil {
			return Loss{}, err
		}
	}
	event, err := newEvent(next, EventLossUpdated, input.ActorID, next.UpdatedAt)
	if err != nil {
		return Loss{}, err
	}
	return s.repo.Update(ctx, scope, next, input.ExpectedVersion, event)
}

func (s *Service) AddRecovery(ctx context.Context, input RecoveryInput) (Loss, Recovery, error) {
	if s == nil || s.repo == nil {
		return Loss{}, Recovery{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	current, err := s.repo.Get(ctx, scope, strings.TrimSpace(input.LossID))
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	if input.ExpectedVersion < 1 || current.Loss.Version != input.ExpectedVersion ||
		current.Loss.Status != StatusActive || !validRecoveryKind(input.Kind) || input.AmountMinor <= 0 {
		return Loss{}, Recovery{}, ErrInvalid
	}
	nextRecovered := current.Totals.RecoveredAmountMinor
	if input.Kind == RecoveryCash {
		nextRecovered += input.AmountMinor
	} else {
		nextRecovered -= input.AmountMinor
	}
	if nextRecovered < 0 || nextRecovered > current.Loss.GrossAmountMinor {
		return Loss{}, Recovery{}, ErrRecoveryLimit
	}
	now := s.now()
	recovery := Recovery{
		LossID: current.Loss.ID, LossVersion: current.Loss.Version + 1,
		Kind: input.Kind, AmountMinor: input.AmountMinor, Currency: current.Loss.Currency,
		Reference: strings.TrimSpace(input.Reference), RecoveredAt: input.RecoveredAt.UTC(),
		ActorID: strings.TrimSpace(input.ActorID), CreatedAt: now,
	}
	if recovery.RecoveredAt.IsZero() {
		recovery.RecoveredAt = now
	}
	recovery.ID, err = newID()
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	event, err := newEventWithVersion(current.Loss, current.Loss.Version+1, EventLossRecoveryRecorded, input.ActorID, now)
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	return s.repo.AddRecovery(ctx, scope, current.Loss.ID, input.ExpectedVersion, recovery, event)
}

func (s *Service) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if s == nil || s.repo == nil {
		return Aggregate{}, ErrInvalid
	}
	normalized, err := normalizeScope(scope)
	if err != nil || strings.TrimSpace(id) == "" {
		return Aggregate{}, ErrInvalid
	}
	return s.repo.Get(ctx, normalized, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, scope Scope, filter ListFilter) (Page, error) {
	if s == nil || s.repo == nil {
		return Page{}, ErrInvalid
	}
	normalized, err := normalizeScope(scope)
	if err != nil {
		return Page{}, err
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.EventType = EventType(strings.ToUpper(strings.TrimSpace(string(filter.EventType))))
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	filter.OrganizationScopeID = strings.TrimSpace(filter.OrganizationScopeID)
	filter.RiskID = strings.TrimSpace(filter.RiskID)
	filter.RecoveryStatus = strings.ToUpper(strings.TrimSpace(filter.RecoveryStatus))
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Status != "" && filter.Status != StatusActive && filter.Status != StatusVoided {
		return Page{}, ErrInvalid
	}
	if filter.EventType != "" && !validEventType(filter.EventType) {
		return Page{}, ErrInvalid
	}
	if filter.Currency != "" && !currencyPattern.MatchString(filter.Currency) {
		return Page{}, ErrInvalid
	}
	if filter.RecoveryStatus != "" && filter.RecoveryStatus != "NONE" &&
		filter.RecoveryStatus != "PARTIAL" && filter.RecoveryStatus != "FULL" {
		return Page{}, ErrInvalid
	}
	if _, err := decodeListCursor(filter.Cursor); err != nil {
		return Page{}, err
	}
	return s.repo.List(ctx, normalized, filter)
}

func (s *Service) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return "", ErrInvalid
	}
	return s.repo.ResolveLegalEntity(ctx, strings.TrimSpace(tenant), strings.TrimSpace(id))
}

func validateLoss(value Loss) error {
	if value.TenantID == "" || value.LegalEntityID == "" || value.Code == "" || value.Title == "" ||
		!validEventType(value.EventType) || value.Cause == "" || value.GrossAmountMinor <= 0 ||
		!currencyPattern.MatchString(value.Currency) || value.OccurredAt.IsZero() || value.DiscoveredAt.IsZero() ||
		value.DiscoveredAt.Before(value.OccurredAt) || value.OwnerPrincipalID == "" ||
		(value.Status != StatusActive && value.Status != StatusVoided) || value.Version < 1 ||
		value.CreatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return ErrInvalid
	}
	return nil
}

func validEventType(value EventType) bool {
	switch value {
	case EventInternalFraud, EventExternalFraud, EventEmploymentPractices, EventClientProductsBusinessPractices,
		EventDamageToPhysicalAssets, EventBusinessDisruptionSystems, EventExecutionDeliveryProcess, EventOther:
		return true
	default:
		return false
	}
}

func validRecoveryKind(value RecoveryKind) bool {
	return value == RecoveryCash || value == RecoveryReversal
}

func normalizeScope(scope Scope) (Scope, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.LegalEntityID == "*" {
		return Scope{}, ErrInvalid
	}
	return scope, nil
}

func totals(loss Loss, recoveries []Recovery) (Totals, error) {
	recovered := int64(0)
	for _, recovery := range recoveries {
		if recovery.Currency != loss.Currency || recovery.AmountMinor <= 0 || !validRecoveryKind(recovery.Kind) {
			return Totals{}, ErrInvalid
		}
		if recovery.Kind == RecoveryCash {
			recovered += recovery.AmountMinor
		} else {
			recovered -= recovery.AmountMinor
		}
	}
	if recovered < 0 || recovered > loss.GrossAmountMinor {
		return Totals{}, ErrRecoveryLimit
	}
	status := "NONE"
	if recovered > 0 && recovered < loss.GrossAmountMinor {
		status = "PARTIAL"
	} else if recovered == loss.GrossAmountMinor {
		status = "FULL"
	}
	return Totals{
		GrossAmountMinor: loss.GrossAmountMinor, RecoveredAmountMinor: recovered,
		NetLossMinor: loss.GrossAmountMinor - recovered, Currency: loss.Currency, RecoveryStatus: status,
	}, nil
}

func newID() (string, error) {
	return platformid.NewUUIDv7()
}

func newEvent(loss Loss, eventType, actorID string, at time.Time) (Event, error) {
	return newEventWithVersion(loss, loss.Version, eventType, actorID, at)
}

func newEventWithVersion(loss Loss, version int64, eventType, actorID string, at time.Time) (Event, error) {
	id, err := newID()
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID: id, TenantID: loss.TenantID, LegalEntityID: loss.LegalEntityID,
		LossID: loss.ID, LossVersion: version, Type: eventType,
		ActorID: strings.TrimSpace(actorID), OccurredAt: at.UTC(),
	}, nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

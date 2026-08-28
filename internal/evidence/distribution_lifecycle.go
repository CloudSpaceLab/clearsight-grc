package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrDistributionInvalid  = errors.New("form distribution is invalid")
	ErrDistributionConflict = errors.New("form distribution has changed")
)

type DistributionPage struct {
	Items      []FormDistribution `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type AmendDistributionInput struct {
	ExpectedVersion    int64
	Deadline           *time.Time
	RouteExpiresAt     *time.Time
	ReminderPolicy     *map[string]any
	AddRecipients      []DistributionRecipientInput
	RevokeRecipientIDs []string
	ActorID            string
}

type DistributionImpact struct {
	CurrentVersion        int64     `json:"current_version"`
	NextVersion           int64     `json:"next_version"`
	DeadlineChanged       bool      `json:"deadline_changed"`
	RouteExpiryChanged    bool      `json:"route_expiry_changed"`
	ReminderPolicyChanged bool      `json:"reminder_policy_changed"`
	RecipientsAdded       int       `json:"recipients_added"`
	RecipientsRevoked     int       `json:"recipients_revoked"`
	EffectiveDeadline     time.Time `json:"effective_deadline"`
	EffectiveRouteExpiry  time.Time `json:"effective_route_expiry"`
	AffectedRecipients    int       `json:"affected_recipients"`
}

type AmendDistributionResult struct {
	Bundle DistributionBundle `json:"bundle"`
	Impact DistributionImpact `json:"impact"`
}

type TransitionDistributionInput struct {
	ExpectedVersion int64
	To              DistributionStatus
	ActorID         string
}

type distributionLifecycleStore interface {
	DistributionStore
	AmendDistribution(context.Context, string, string, string, AmendDistributionInput, time.Time) (AmendDistributionResult, error)
	TransitionDistribution(context.Context, string, string, string, TransitionDistributionInput, time.Time) (DistributionBundle, error)
}

type DistributionService struct {
	store distributionLifecycleStore
	now   func() time.Time
}

func NewDistributionService(store distributionLifecycleStore) *DistributionService {
	return &DistributionService{store: store, now: time.Now}
}

func (service *DistributionService) Create(ctx context.Context, input CreateDistributionInput) (DistributionBundle, error) {
	if service == nil || service.store == nil {
		return DistributionBundle{}, ErrDistributionInvalid
	}
	bundle, err := service.store.CreateDistribution(ctx, input)
	if err != nil {
		return DistributionBundle{}, normalizeDistributionError(err)
	}
	opened, err := service.store.TransitionDistribution(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, TransitionDistributionInput{
		ExpectedVersion: bundle.Distribution.Version,
		To:              DistributionOpen,
		ActorID:         input.CreatedBy,
	}, service.currentTime())
	if err != nil {
		// A safely persisted DRAFT is recoverable; never report it as dispatched.
		return DistributionBundle{}, normalizeDistributionError(err)
	}
	return opened, nil
}

func (service *DistributionService) Get(ctx context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	if service == nil || service.store == nil {
		return DistributionBundle{}, ErrNotFound
	}
	bundle, err := service.store.GetDistribution(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(distributionID))
	if err != nil {
		return DistributionBundle{}, normalizeDistributionError(err)
	}
	return bundle, nil
}

func (service *DistributionService) List(ctx context.Context, query DistributionListQuery) (DistributionPage, error) {
	if service == nil || service.store == nil {
		return DistributionPage{}, ErrDistributionInvalid
	}
	if query.Limit == 0 {
		query.Limit = 25
	}
	if !normalizeDistributionListQuery(&query, service.currentTime()) {
		return DistributionPage{}, fmt.Errorf("%w: distribution filters are invalid", ErrDistributionInvalid)
	}
	values, err := service.store.ListDistributions(ctx, query)
	if err != nil {
		return DistributionPage{}, normalizeDistributionError(err)
	}
	page := DistributionPage{Items: values}
	if len(values) == query.Limit {
		page.NextCursor = encodeDistributionCursor(values[len(values)-1])
	}
	return page, nil
}

func (service *DistributionService) Amend(ctx context.Context, tenantID, legalEntityID, distributionID string, input AmendDistributionInput) (AmendDistributionResult, error) {
	if service == nil || service.store == nil || input.ExpectedVersion < 1 || strings.TrimSpace(input.ActorID) == "" {
		return AmendDistributionResult{}, fmt.Errorf("%w: expected_version and actor are required", ErrDistributionInvalid)
	}
	input.ActorID = strings.TrimSpace(input.ActorID)
	if len(input.AddRecipients)+len(input.RevokeRecipientIDs) > 500 {
		return AmendDistributionResult{}, fmt.Errorf("%w: too many recipient changes", ErrDistributionInvalid)
	}
	seen := make(map[string]struct{}, len(input.RevokeRecipientIDs))
	for index, recipientID := range input.RevokeRecipientIDs {
		recipientID = strings.TrimSpace(recipientID)
		if recipientID == "" {
			return AmendDistributionResult{}, fmt.Errorf("%w: recipient id is required", ErrDistributionInvalid)
		}
		if _, duplicate := seen[recipientID]; duplicate {
			return AmendDistributionResult{}, fmt.Errorf("%w: duplicate recipient revocation", ErrDistributionInvalid)
		}
		seen[recipientID] = struct{}{}
		input.RevokeRecipientIDs[index] = recipientID
	}
	value, err := service.store.AmendDistribution(ctx, tenantID, legalEntityID, distributionID, input, service.currentTime())
	if err != nil {
		return AmendDistributionResult{}, normalizeDistributionError(err)
	}
	return value, nil
}

func (service *DistributionService) Lock(ctx context.Context, tenantID, legalEntityID, distributionID string, expectedVersion int64, actorID string) (DistributionBundle, error) {
	return service.transition(ctx, tenantID, legalEntityID, distributionID, expectedVersion, DistributionLocked, actorID)
}

func (service *DistributionService) Reopen(ctx context.Context, tenantID, legalEntityID, distributionID string, expectedVersion int64, actorID string) (DistributionBundle, error) {
	return service.transition(ctx, tenantID, legalEntityID, distributionID, expectedVersion, DistributionOpen, actorID)
}

func (service *DistributionService) Revoke(ctx context.Context, tenantID, legalEntityID, distributionID string, expectedVersion int64, actorID string) (DistributionBundle, error) {
	return service.transition(ctx, tenantID, legalEntityID, distributionID, expectedVersion, DistributionRevoked, actorID)
}

func (service *DistributionService) transition(ctx context.Context, tenantID, legalEntityID, distributionID string, expectedVersion int64, to DistributionStatus, actorID string) (DistributionBundle, error) {
	if service == nil || service.store == nil || expectedVersion < 1 || strings.TrimSpace(actorID) == "" {
		return DistributionBundle{}, fmt.Errorf("%w: expected_version and actor are required", ErrDistributionInvalid)
	}
	bundle, err := service.store.TransitionDistribution(ctx, tenantID, legalEntityID, distributionID, TransitionDistributionInput{ExpectedVersion: expectedVersion, To: to, ActorID: strings.TrimSpace(actorID)}, service.currentTime())
	if err != nil {
		return DistributionBundle{}, normalizeDistributionError(err)
	}
	return bundle, nil
}

func (service *DistributionService) currentTime() time.Time {
	if service != nil && service.now != nil {
		return service.now().UTC()
	}
	return time.Now().UTC()
}

func normalizeDistributionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrDistributionConflict) || errors.Is(err, ErrDistributionInvalid) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrDistributionInvalid, err)
}

func validDistributionTransition(from, to DistributionStatus, now, deadline time.Time) bool {
	switch to {
	case DistributionLocked:
		return from == DistributionOpen || from == DistributionReady || from == DistributionDraft
	case DistributionOpen:
		return (from == DistributionDraft || from == DistributionReady || from == DistributionLocked) && deadline.After(now)
	case DistributionRevoked:
		return from != DistributionRevoked && from != DistributionCompleted && from != DistributionSuperseded
	default:
		return false
	}
}

package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type RecipientRouteType string

const (
	RouteInternalPrincipal RecipientRouteType = "INTERNAL_PRINCIPAL"
	RouteExternalContact   RecipientRouteType = "EXTERNAL_CONTACT"
)

type RecipientRoute struct {
	Type        RecipientRouteType `json:"type"`
	PrincipalID string             `json:"principal_id,omitempty"`
	ContactRef  string             `json:"contact_ref,omitempty"`
	SafeHint    string             `json:"safe_hint,omitempty"`
}

type DeliveryState string

const (
	DeliveryNotDispatched DeliveryState = "NOT_DISPATCHED"
	DeliveryAssigned      DeliveryState = "ASSIGNED"
	DeliveryDelivered     DeliveryState = "DELIVERED"
	DeliveryBlocked       DeliveryState = "BLOCKED"
	DeliveryFailed        DeliveryState = "FAILED"
)

type CollectionCycleState string

const (
	CycleScheduled        CollectionCycleState = "SCHEDULED"
	CycleClaimed          CollectionCycleState = "CLAIMED"
	CycleAwaitingResponse CollectionCycleState = "AWAITING_RESPONSE"
	CycleComplete         CollectionCycleState = "COMPLETE"
	CycleCancelled        CollectionCycleState = "CANCELLED"
	CycleBlocked          CollectionCycleState = "BLOCKED"
	CycleFailed           CollectionCycleState = "FAILED"
)

type CollectionCycle struct {
	ID                     string               `json:"id"`
	TenantID               string               `json:"tenant_id"`
	ProgramID              string               `json:"program_id"`
	MonitoringCheckID      string               `json:"monitoring_check_id"`
	MonitoringCheckVersion int64                `json:"monitoring_check_version"`
	Sequence               int64                `json:"sequence"`
	Policy                 CollectionPolicy     `json:"collection_policy"`
	CurrentRequestID       string               `json:"current_request_id,omitempty"`
	PredecessorRequestID   string               `json:"predecessor_request_id,omitempty"`
	LatestSubmissionID     string               `json:"latest_submission_id,omitempty"`
	LatestSubmittedAt      *time.Time           `json:"latest_submitted_at,omitempty"`
	ExpiresAt              time.Time            `json:"expires_at"`
	RenewalOpensAt         time.Time            `json:"renewal_opens_at"`
	NextActionAt           *time.Time           `json:"next_action_at,omitempty"`
	RemindersSent          int                  `json:"reminders_sent"`
	Recipient              RecipientRoute       `json:"recipient"`
	DeliveryState          DeliveryState        `json:"delivery_state"`
	DeliveryReference      string               `json:"delivery_reference,omitempty"`
	State                  CollectionCycleState `json:"state"`
	LeaseOwner             string               `json:"lease_owner,omitempty"`
	LeaseToken             string               `json:"lease_token,omitempty"`
	LeaseUntil             *time.Time           `json:"lease_until,omitempty"`
	Attempts               int                  `json:"attempts"`
	SafeError              string               `json:"safe_error,omitempty"`
	CreatedAt              time.Time            `json:"created_at"`
	UpdatedAt              time.Time            `json:"updated_at"`
}

type CollectionActionCompletion struct {
	State             CollectionCycleState
	CurrentRequestID  string
	DeliveryState     DeliveryState
	DeliveryReference string
	NextActionAt      *time.Time
	RemindersSent     *int
	At                time.Time
}

type CollectionSummary struct {
	CycleID                string               `json:"cycle_id"`
	TenantID               string               `json:"tenant_id"`
	ProgramID              string               `json:"program_id"`
	MonitoringCheckID      string               `json:"monitoring_check_id"`
	MonitoringCheckVersion int64                `json:"monitoring_check_version"`
	Sequence               int64                `json:"sequence"`
	Policy                 CollectionPolicy     `json:"collection_policy"`
	CurrentRequestID       string               `json:"current_request_id,omitempty"`
	LatestSubmissionID     string               `json:"latest_submission_id,omitempty"`
	LatestSubmittedAt      *time.Time           `json:"latest_submitted_at,omitempty"`
	ExpiresAt              time.Time            `json:"expires_at"`
	RenewalOpensAt         time.Time            `json:"renewal_opens_at"`
	NextActionAt           *time.Time           `json:"next_action_at,omitempty"`
	RemindersSent          int                  `json:"reminders_sent"`
	Recipient              RecipientRoute       `json:"recipient"`
	DeliveryState          DeliveryState        `json:"delivery_state"`
	State                  CollectionCycleState `json:"state"`
	SafeError              string               `json:"safe_error,omitempty"`
	GeneratedAt            time.Time            `json:"generated_at"`
}

type CollectionCycleRepository interface {
	UpsertCollectionCycle(context.Context, CollectionCycle) (CollectionCycle, error)
	CollectionCycle(context.Context, string, string) (CollectionCycle, error)
	CollectionCycleForSequence(context.Context, string, string, int64) (CollectionCycle, error)
	ClaimDueCollectionCycles(context.Context, string, time.Time, time.Duration, int) ([]CollectionCycle, error)
	CompleteCollectionAction(context.Context, CollectionCycle, CollectionActionCompletion) (CollectionCycle, error)
	FailCollectionAction(context.Context, CollectionCycle, string, *time.Time, int, time.Time) (CollectionCycle, error)
	CancelCollectionCyclesByCheck(context.Context, string, string, time.Time) (int, error)
	CompleteCollectionCyclesBeforeSequence(context.Context, string, string, int64, time.Time) (int, error)
	ListCollectionSummaries(context.Context, string, string, int) ([]CollectionSummary, error)
}

func validateCollectionCycle(value CollectionCycle) (CollectionCycle, error) {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.TenantID) == "" || strings.TrimSpace(value.ProgramID) == "" || strings.TrimSpace(value.MonitoringCheckID) == "" || value.MonitoringCheckVersion < 1 || value.Sequence < 1 {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection cycle identity is required"))
	}
	policy, err := normalizeCollectionPolicy(&value.Policy)
	if err != nil {
		return CollectionCycle{}, err
	}
	value.Policy = policy
	if value.ExpiresAt.IsZero() || value.RenewalOpensAt.IsZero() || !value.RenewalOpensAt.Before(value.ExpiresAt) {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection expiry and renewal opening are required"))
	}
	if value.NextActionAt != nil && value.NextActionAt.After(value.ExpiresAt) {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("next collection action cannot follow expiry"))
	}
	if value.RemindersSent < 0 || value.RemindersSent > policy.ReminderCount || value.Attempts < 0 || value.Attempts > 20 {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection progress is invalid"))
	}
	if err := validateRecipientRoute(value.Recipient); err != nil {
		return CollectionCycle{}, err
	}
	if value.DeliveryState == "" {
		value.DeliveryState = DeliveryNotDispatched
	}
	switch value.DeliveryState {
	case DeliveryNotDispatched, DeliveryAssigned, DeliveryDelivered, DeliveryBlocked, DeliveryFailed:
	default:
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection delivery state is invalid"))
	}
	switch value.State {
	case CycleScheduled, CycleClaimed, CycleAwaitingResponse, CycleComplete, CycleCancelled, CycleBlocked, CycleFailed:
	default:
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection cycle state is invalid"))
	}
	if len(strings.TrimSpace(value.SafeError)) > 1000 || len(strings.TrimSpace(value.DeliveryReference)) > 512 {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection operational detail is too long"))
	}
	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return CollectionCycle{}, errors.Join(ErrInvalid, fmt.Errorf("collection timestamps are invalid"))
	}
	return value, nil
}

func validateRecipientRoute(route RecipientRoute) error {
	switch route.Type {
	case RouteInternalPrincipal:
		if strings.TrimSpace(route.PrincipalID) == "" || route.ContactRef != "" || route.SafeHint != "" {
			return errors.Join(ErrInvalid, fmt.Errorf("internal collection route requires one principal"))
		}
	case RouteExternalContact:
		if route.PrincipalID != "" || strings.TrimSpace(route.ContactRef) == "" || len(strings.TrimSpace(route.ContactRef)) > 512 || strings.TrimSpace(route.SafeHint) == "" || len(strings.TrimSpace(route.SafeHint)) > 256 {
			return errors.Join(ErrInvalid, fmt.Errorf("external collection route requires an opaque contact and safe hint"))
		}
	default:
		return errors.Join(ErrInvalid, fmt.Errorf("collection recipient route is invalid"))
	}
	return nil
}

func collectionSummary(value CollectionCycle, generatedAt time.Time) CollectionSummary {
	return CollectionSummary{
		CycleID: value.ID, TenantID: value.TenantID, ProgramID: value.ProgramID, MonitoringCheckID: value.MonitoringCheckID,
		MonitoringCheckVersion: value.MonitoringCheckVersion, Sequence: value.Sequence, Policy: value.Policy,
		CurrentRequestID: value.CurrentRequestID, LatestSubmissionID: value.LatestSubmissionID, LatestSubmittedAt: value.LatestSubmittedAt,
		ExpiresAt: value.ExpiresAt, RenewalOpensAt: value.RenewalOpensAt, NextActionAt: value.NextActionAt,
		RemindersSent: value.RemindersSent, Recipient: value.Recipient, DeliveryState: value.DeliveryState, State: value.State, SafeError: value.SafeError,
		GeneratedAt: generatedAt.UTC(),
	}
}

func sameCollectionSchedule(left, right CollectionCycle) bool {
	return left.TenantID == right.TenantID && left.ProgramID == right.ProgramID && left.MonitoringCheckID == right.MonitoringCheckID &&
		left.MonitoringCheckVersion == right.MonitoringCheckVersion && left.Sequence == right.Sequence && left.Policy == right.Policy &&
		left.CurrentRequestID == right.CurrentRequestID && left.PredecessorRequestID == right.PredecessorRequestID &&
		left.LatestSubmissionID == right.LatestSubmissionID && equalOptionalTime(left.LatestSubmittedAt, right.LatestSubmittedAt) &&
		left.ExpiresAt.Equal(right.ExpiresAt) && left.RenewalOpensAt.Equal(right.RenewalOpensAt) && left.Recipient == right.Recipient
}

func equalOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

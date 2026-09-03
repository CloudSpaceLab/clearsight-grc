package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultRenewalWindowDays = 30
	defaultReminderCount     = 3
)

type CollectionPolicy struct {
	ValidityMonths    int `json:"validity_months"`
	RenewalWindowDays int `json:"renewal_window_days"`
	ReminderCount     int `json:"reminder_count"`
}

// CollectionDates uses calendar-month validity and evenly spaces reminders
// inside the renewal window. Month-end submissions clamp to the target month's
// last valid day.
func CollectionDates(submitted time.Time, policy CollectionPolicy) (time.Time, time.Time, []time.Time) {
	normalized, err := normalizeCollectionPolicy(&policy)
	if err != nil || submitted.IsZero() {
		return time.Time{}, time.Time{}, nil
	}
	expires := addMonthsClamped(submitted.UTC(), normalized.ValidityMonths)
	opens := expires.AddDate(0, 0, -normalized.RenewalWindowDays)
	window := expires.Sub(opens)
	reminders := make([]time.Time, normalized.ReminderCount)
	for index := range reminders {
		reminders[index] = opens.Add(window * time.Duration(index+1) / time.Duration(normalized.ReminderCount+1))
	}
	return expires, opens, reminders
}

func addMonthsClamped(value time.Time, months int) time.Time {
	targetMonth := time.Date(value.Year(), value.Month(), 1, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location()).AddDate(0, months, 0)
	lastDay := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location()).Day()
	day := value.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonth.Year(), targetMonth.Month(), day, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
}

type UpdateCollectionPolicyInput struct {
	ID              string           `json:"id"`
	ExpectedVersion int64            `json:"expected_version"`
	Policy          CollectionPolicy `json:"collection_policy"`
}

func normalizeCollectionPolicy(policy *CollectionPolicy) (CollectionPolicy, error) {
	if policy == nil {
		return CollectionPolicy{}, errors.Join(ErrInvalid, fmt.Errorf("collection policy is required"))
	}
	normalized := *policy
	if normalized.RenewalWindowDays == 0 {
		normalized.RenewalWindowDays = defaultRenewalWindowDays
	}
	if normalized.ReminderCount == 0 {
		normalized.ReminderCount = defaultReminderCount
	}
	if normalized.ValidityMonths < 1 || normalized.ValidityMonths > 120 {
		return CollectionPolicy{}, errors.Join(ErrInvalid, fmt.Errorf("response expiry must be between 1 and 120 months"))
	}
	if normalized.RenewalWindowDays < 1 || normalized.RenewalWindowDays > 90 || normalized.RenewalWindowDays > normalized.ValidityMonths*28-1 {
		return CollectionPolicy{}, errors.Join(ErrInvalid, fmt.Errorf("renewal period must end before the earliest response expiry"))
	}
	if normalized.ReminderCount < 1 || normalized.ReminderCount > 5 {
		return CollectionPolicy{}, errors.Join(ErrInvalid, fmt.Errorf("reminders must be between 1 and 5"))
	}
	return normalized, nil
}

func (s *Service) UpdateCollectionPolicy(ctx context.Context, actor Actor, input UpdateCollectionPolicyInput) (MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringCheck{}, err
	}
	if strings.TrimSpace(input.ID) == "" || input.ExpectedVersion < 1 {
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring check and expected version are required"))
	}
	policy, err := normalizeCollectionPolicy(&input.Policy)
	if err != nil {
		return MonitoringCheck{}, err
	}
	current, err := s.repo.CheckRevision(ctx, actor.TenantID, strings.TrimSpace(input.ID), input.ExpectedVersion)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if current.InputKind != InputForm || current.Status != LifecycleActive || !current.IsCurrent {
		return MonitoringCheck{}, ErrInactive
	}
	return s.repo.ReviseCheck(ctx, CheckRevisionUpdate{
		TenantID: actor.TenantID, ID: current.ID, ExpectedVersion: input.ExpectedVersion,
		Policy: policy, ActorID: actor.PrincipalID, At: s.now().UTC(),
	})
}

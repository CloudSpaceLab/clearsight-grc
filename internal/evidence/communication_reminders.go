package evidence

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

type communicationReminderSpec struct {
	Action      CommunicationAction
	HoursBefore int
}

type communicationReminderSchedulerRepository interface {
	ScheduleDueCommunicationReminders(context.Context, time.Time, int) (int, error)
}

type CommunicationReminderScheduler struct {
	repository communicationReminderSchedulerRepository
}

func NewCommunicationReminderScheduler(repository communicationReminderSchedulerRepository) *CommunicationReminderScheduler {
	return &CommunicationReminderScheduler{repository: repository}
}

func (scheduler *CommunicationReminderScheduler) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if scheduler == nil || scheduler.repository == nil {
		return 0, ErrCommunicationUnavailable
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	return scheduler.repository.ScheduleDueCommunicationReminders(ctx, now.UTC(), limit)
}

func normalizeDistributionReminderPolicy(policy map[string]any) (map[string]any, error) {
	if len(policy) == 0 {
		return map[string]any{}, nil
	}
	specs, err := communicationReminderSpecs(policy)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	for _, action := range []CommunicationAction{CommunicationReminder, CommunicationDueSoon} {
		values := make([]int, 0)
		for _, spec := range specs {
			if spec.Action == action {
				values = append(values, spec.HoursBefore)
			}
		}
		if len(values) == 0 {
			continue
		}
		key := "reminder_hours_before"
		if action == CommunicationDueSoon {
			key = "due_soon_hours_before"
		}
		result[key] = values
	}
	return result, nil
}

func communicationReminderSpecs(policy map[string]any) ([]communicationReminderSpec, error) {
	if len(policy) == 0 {
		return nil, nil
	}
	allowed := map[string]CommunicationAction{
		"reminder_hours_before": CommunicationReminder,
		"due_soon_hours_before": CommunicationDueSoon,
	}
	for key := range policy {
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("%w: unsupported reminder policy key %q", ErrDistributionInvalid, key)
		}
	}

	result := make([]communicationReminderSpec, 0, 8)
	seen := map[communicationReminderSpec]struct{}{}
	for key, action := range allowed {
		value, ok := policy[key]
		if !ok {
			continue
		}
		hours, err := reminderHours(value)
		if err != nil {
			return nil, err
		}
		maximum := 720
		if action == CommunicationDueSoon {
			maximum = 168
		}
		for _, hour := range hours {
			if hour < 1 || hour > maximum {
				return nil, fmt.Errorf("%w: reminder offset is outside the supported window", ErrDistributionInvalid)
			}
			spec := communicationReminderSpec{Action: action, HoursBefore: hour}
			if _, duplicate := seen[spec]; duplicate {
				continue
			}
			seen[spec] = struct{}{}
			result = append(result, spec)
		}
	}
	if len(result) > 12 {
		return nil, fmt.Errorf("%w: at most 12 reminder offsets are supported", ErrDistributionInvalid)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Action != result[j].Action {
			return result[i].Action < result[j].Action
		}
		return result[i].HoursBefore > result[j].HoursBefore
	})
	return result, nil
}

func reminderHours(value any) ([]int, error) {
	values, ok := value.([]any)
	if !ok {
		if typed, typedOK := value.([]int); typedOK {
			return append([]int(nil), typed...), nil
		}
		return nil, fmt.Errorf("%w: reminder offsets must be an array", ErrDistributionInvalid)
	}
	result := make([]int, 0, len(values))
	for _, raw := range values {
		switch number := raw.(type) {
		case float64:
			if number != math.Trunc(number) || number > math.MaxInt32 || number < math.MinInt32 {
				return nil, fmt.Errorf("%w: reminder offsets must be whole hours", ErrDistributionInvalid)
			}
			result = append(result, int(number))
		case int:
			result = append(result, number)
		case int64:
			if number > math.MaxInt32 || number < math.MinInt32 {
				return nil, fmt.Errorf("%w: reminder offset is invalid", ErrDistributionInvalid)
			}
			result = append(result, int(number))
		default:
			return nil, fmt.Errorf("%w: reminder offsets must be numeric", ErrDistributionInvalid)
		}
	}
	return result, nil
}

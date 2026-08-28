package evidence

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type reminderSchedulerRepositoryFunc func(context.Context, time.Time, int) (int, error)

func (fn reminderSchedulerRepositoryFunc) ScheduleDueCommunicationReminders(ctx context.Context, now time.Time, limit int) (int, error) {
	return fn(ctx, now, limit)
}

func TestCommunicationReminderPolicyNormalizesAndRejectsUnsafeShapes(t *testing.T) {
	t.Parallel()

	policy, err := normalizeDistributionReminderPolicy(map[string]any{
		"reminder_hours_before": []any{float64(48), float64(24), float64(48)},
		"due_soon_hours_before": []int{12, 6},
	})
	if err != nil {
		t.Fatalf("normalize reminder policy: %v", err)
	}
	want := map[string]any{
		"reminder_hours_before": []int{48, 24},
		"due_soon_hours_before": []int{12, 6},
	}
	if !reflect.DeepEqual(policy, want) {
		t.Fatalf("policy = %#v, want %#v", policy, want)
	}

	for name, invalid := range map[string]map[string]any{
		"unknown key": {"weekly": []any{float64(24)}},
		"fraction":    {"reminder_hours_before": []any{1.5}},
		"too far":     {"due_soon_hours_before": []any{float64(169)}},
		"not array":   {"reminder_hours_before": 24},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeDistributionReminderPolicy(invalid); !errors.Is(err, ErrDistributionInvalid) {
				t.Fatalf("error = %v, want distribution invalid", err)
			}
		})
	}
}

func TestCommunicationReminderSchedulerBoundsWork(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	called := 0
	scheduler := NewCommunicationReminderScheduler(reminderSchedulerRepositoryFunc(func(_ context.Context, got time.Time, limit int) (int, error) {
		called++
		if !got.Equal(fixed) {
			t.Fatalf("time = %v, want %v", got, fixed)
		}
		if limit != 500 {
			t.Fatalf("limit = %d, want bounded 500", limit)
		}
		return 3, nil
	}))
	count, err := scheduler.Maintain(context.Background(), fixed, 9999)
	if err != nil || count != 3 || called != 1 {
		t.Fatalf("maintain = (%d, %v), calls=%d", count, err, called)
	}
}

package notificationprefs

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultsKeepCriticalEmailMandatory(t *testing.T) {
	service := NewService(NewMemoryRepository())
	value, err := service.Get(context.Background(), "tenant", "principal")
	if err != nil {
		t.Fatal(err)
	}
	if !value.DailyDigestEnabled || value.DigestMinute != DefaultDigestMinute || value.TimeZone != DefaultTimeZone {
		t.Fatalf("unexpected defaults: %#v", value)
	}
	if !value.CriticalEmailRequired {
		t.Fatal("critical email must remain organization-mandated")
	}
	if value.Version != 0 {
		t.Fatalf("default version=%d want 0", value.Version)
	}
}

func TestUpdateValidatesVersionTimezoneAndQuietHours(t *testing.T) {
	service := NewService(NewMemoryRepository())
	input := UpdateInput{
		DailyDigestEnabled: true,
		DigestMinute:       8*60 + 30,
		TimeZone:           "Africa/Lagos",
		QuietHoursEnabled:  true,
		QuietStartMinute:   22 * 60,
		QuietEndMinute:     6 * 60,
		ExpectedVersion:    0,
	}
	saved, err := service.Update(context.Background(), "tenant", "principal", input)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 || saved.DigestMinute != input.DigestMinute || saved.TimeZone != input.TimeZone {
		t.Fatalf("unexpected saved preferences: %#v", saved)
	}
	input.ExpectedVersion = 0
	if _, err := service.Update(context.Background(), "tenant", "principal", input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale update err=%v want version conflict", err)
	}

	input.ExpectedVersion = 1
	input.TimeZone = "Not/A_Real_Timezone"
	if _, err := service.Update(context.Background(), "tenant", "principal", input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid timezone err=%v want invalid", err)
	}

	input.TimeZone = "UTC"
	input.QuietStartMinute = input.QuietEndMinute
	if _, err := service.Update(context.Background(), "tenant", "principal", input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid quiet window err=%v want invalid", err)
	}
}

func TestDigestMayBeDisabledWithoutChangingCriticalPolicy(t *testing.T) {
	service := NewService(NewMemoryRepository())
	saved, err := service.Update(context.Background(), "tenant", "principal", UpdateInput{
		DailyDigestEnabled: false,
		DigestMinute:       DefaultDigestMinute,
		TimeZone:           "UTC",
		QuietStartMinute:   DefaultQuietStartMinute,
		QuietEndMinute:     DefaultQuietEndMinute,
		ExpectedVersion:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.DailyDigestEnabled {
		t.Fatal("daily digest should be optional")
	}
	if !saved.CriticalEmailRequired {
		t.Fatal("critical email policy must not be user-disableable")
	}
}

package notificationprefs

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid         = errors.New("notification preference is invalid")
	ErrNotFound        = errors.New("notification preference not found")
	ErrVersionConflict = errors.New("notification preference changed")
)

const (
	DefaultDigestMinute     = 7 * 60
	DefaultQuietStartMinute = 22 * 60
	DefaultQuietEndMinute   = 7 * 60
	DefaultTimeZone         = "UTC"
)

type Preferences struct {
	TenantID              string    `json:"tenant_id"`
	PrincipalID           string    `json:"principal_id"`
	DailyDigestEnabled    bool      `json:"daily_digest_enabled"`
	DigestMinute          int       `json:"digest_minute"`
	TimeZone              string    `json:"time_zone"`
	QuietHoursEnabled     bool      `json:"quiet_hours_enabled"`
	QuietStartMinute      int       `json:"quiet_start_minute"`
	QuietEndMinute        int       `json:"quiet_end_minute"`
	CriticalEmailRequired bool      `json:"critical_email_required"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
	Version               int64     `json:"version"`
}

type UpdateInput struct {
	DailyDigestEnabled bool   `json:"daily_digest_enabled"`
	DigestMinute       int    `json:"digest_minute"`
	TimeZone           string `json:"time_zone"`
	QuietHoursEnabled  bool   `json:"quiet_hours_enabled"`
	QuietStartMinute   int    `json:"quiet_start_minute"`
	QuietEndMinute     int    `json:"quiet_end_minute"`
	ExpectedVersion    int64  `json:"expected_version"`
}

type Stored struct {
	TenantID           string
	PrincipalID        string
	DailyDigestEnabled bool
	DigestMinute       int
	TimeZone           string
	QuietHoursEnabled  bool
	QuietStartMinute   int
	QuietEndMinute     int
	UpdatedAt          time.Time
	Version            int64
}

type Repository interface {
	Get(context.Context, string, string) (Stored, error)
	Upsert(context.Context, Stored, int64) (Stored, error)
}

func Default(tenantID, principalID string) Stored {
	return Stored{
		TenantID: tenantID, PrincipalID: principalID,
		DailyDigestEnabled: true, DigestMinute: DefaultDigestMinute, TimeZone: DefaultTimeZone,
		QuietStartMinute: DefaultQuietStartMinute, QuietEndMinute: DefaultQuietEndMinute,
	}
}

func validate(value Stored) bool {
	value.TenantID = strings.TrimSpace(value.TenantID)
	value.PrincipalID = strings.TrimSpace(value.PrincipalID)
	value.TimeZone = strings.TrimSpace(value.TimeZone)
	if value.TenantID == "" || value.PrincipalID == "" || len(value.TimeZone) == 0 || len(value.TimeZone) > 64 ||
		strings.ContainsAny(value.TimeZone, "\r\n\x00") ||
		value.DigestMinute < 0 || value.DigestMinute > 1439 ||
		value.QuietStartMinute < 0 || value.QuietStartMinute > 1439 ||
		value.QuietEndMinute < 0 || value.QuietEndMinute > 1439 ||
		(value.QuietHoursEnabled && value.QuietStartMinute == value.QuietEndMinute) {
		return false
	}
	_, err := time.LoadLocation(value.TimeZone)
	return err == nil
}

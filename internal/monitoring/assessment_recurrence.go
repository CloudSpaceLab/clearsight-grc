package monitoring

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// AssessmentRecurrence describes calendar periods in the schedule's time zone.
// Monthly periods start on the first day of the month. Day intervals can start on any date.
type AssessmentRecurrence struct {
	FirstPeriodStart time.Time
	IntervalMonths   int
	IntervalDays     int
	ResponseDays     int
}

// AssessmentOccurrence identifies one reporting period and its response window.
type AssessmentOccurrence struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	OpensAt     time.Time
	Deadline    time.Time
}

// Occurrence computes dates from the original anchor. ResponseDays counts
// local calendar days, including the opening day.
func (r AssessmentRecurrence) Occurrence(index int) (AssessmentOccurrence, error) {
	anchor := r.FirstPeriodStart
	if anchor.IsZero() || anchor.Hour() != 0 || anchor.Minute() != 0 || anchor.Second() != 0 || anchor.Nanosecond() != 0 {
		return AssessmentOccurrence{}, fmt.Errorf("%w: choose a valid reporting start date", ErrInvalid)
	}
	monthly := r.IntervalMonths >= 1 && r.IntervalMonths <= 24 && r.IntervalDays == 0
	daily := r.IntervalDays >= 1 && r.IntervalDays <= 365 && r.IntervalMonths == 0
	if !monthly && !daily {
		return AssessmentOccurrence{}, fmt.Errorf("%w: choose 1–24 months or 1–365 days between assessments", ErrInvalid)
	}
	if monthly && anchor.Day() != 1 {
		return AssessmentOccurrence{}, fmt.Errorf("%w: choose the first day of a month for a monthly interval", ErrInvalid)
	}
	if r.ResponseDays < 1 || r.ResponseDays > 365 || index < 0 || index > 120000 {
		return AssessmentOccurrence{}, fmt.Errorf("%w: choose a valid occurrence and response window", ErrInvalid)
	}
	start := anchor.AddDate(0, index*r.IntervalMonths, index*r.IntervalDays)
	opens := anchor.AddDate(0, (index+1)*r.IntervalMonths, (index+1)*r.IntervalDays)
	if opens.AddDate(0, 0, r.ResponseDays).Year() > 9999 {
		return AssessmentOccurrence{}, fmt.Errorf("%w: the response deadline exceeds the supported calendar", ErrInvalid)
	}
	return AssessmentOccurrence{
		PeriodStart: start,
		PeriodEnd:   opens.Add(-time.Microsecond),
		OpensAt:     opens,
		Deadline:    opens.AddDate(0, 0, r.ResponseDays).Add(-time.Microsecond),
	}, nil
}

// AssessmentKey gives the existing origin deduplication a stable UUID for
// this schedule, occurrence, and recipient. The schedule ID must be immutable.
func (o AssessmentOccurrence) AssessmentKey(scheduleID, respondentID string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(scheduleID) + ":" +
		o.PeriodStart.UTC().Format(time.RFC3339) + ":" + strings.ToLower(respondentID)))
	sum[6] = (sum[6] & 0x0f) | 0x80
	sum[8] = (sum[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

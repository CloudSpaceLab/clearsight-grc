package monitoring

import (
	"testing"
	"time"
)

func TestAssessmentRecurrenceQuarterlyBoundaries(t *testing.T) {
	lagos := time.FixedZone("WAT", 3600)
	r := AssessmentRecurrence{time.Date(2026, 10, 1, 0, 0, 0, 0, lagos), 3, 0, 15}
	o, err := r.Occurrence(0)
	if err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]struct{ got, want string }{
		"start":    {o.PeriodStart.Format(time.RFC3339Nano), "2026-10-01T00:00:00+01:00"},
		"end":      {o.PeriodEnd.Format(time.RFC3339Nano), "2026-12-31T23:59:59.999999+01:00"},
		"dispatch": {o.OpensAt.Format(time.RFC3339Nano), "2027-01-01T00:00:00+01:00"},
		"deadline": {o.Deadline.Format(time.RFC3339Nano), "2027-01-15T23:59:59.999999+01:00"},
	} {
		if check.got != check.want {
			t.Errorf("%s = %s, want %s", name, check.got, check.want)
		}
	}
	next, err := r.Occurrence(1)
	if err != nil || !next.PeriodStart.Equal(o.OpensAt) {
		t.Fatalf("next period does not start at the preceding boundary: %+v, %v", next, err)
	}
	key := o.AssessmentKey("schedule-a", "respondent-a")
	if !formLinkUUID.MatchString(key) || key != o.AssessmentKey("schedule-a", "respondent-a") ||
		key == o.AssessmentKey("schedule-b", "respondent-a") || key == o.AssessmentKey("schedule-a", "respondent-b") ||
		key == next.AssessmentKey("schedule-a", "respondent-a") {
		t.Fatal("assessment key must be stable for retries and distinct per schedule, period and respondent")
	}
}

func TestAssessmentRecurrenceKeepsLeapDay(t *testing.T) {
	r := AssessmentRecurrence{time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC), 1, 0, 1}
	o, err := r.Occurrence(0)
	if err != nil || o.PeriodEnd.Day() != 29 || o.Deadline.Day() != 1 || o.Deadline.Month() != time.March {
		t.Fatalf("leap-year occurrence = %+v, %v", o, err)
	}
	r.FirstPeriodStart = r.FirstPeriodStart.AddDate(0, 0, 1)
	if _, err := r.Occurrence(0); err == nil {
		t.Fatal("non-month boundary was accepted")
	}
}

func TestAssessmentRecurrenceCustomFortnight(t *testing.T) {
	r := AssessmentRecurrence{
		FirstPeriodStart: time.Date(2026, 12, 22, 0, 0, 0, 0, time.UTC),
		IntervalDays:     14,
		ResponseDays:     5,
	}
	occurrence, err := r.Occurrence(0)
	if err != nil || occurrence.OpensAt.Format(time.DateOnly) != "2027-01-05" || occurrence.Deadline.Format(time.DateOnly) != "2027-01-09" {
		t.Fatalf("custom occurrence = %+v, %v", occurrence, err)
	}
	r.IntervalMonths = 2
	if _, err := r.Occurrence(0); err == nil {
		t.Fatal("two interval units were accepted")
	}
	r.IntervalDays = 0
	r.FirstPeriodStart = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	occurrence, err = r.Occurrence(0)
	if err != nil || occurrence.OpensAt.Format(time.DateOnly) != "2027-02-01" {
		t.Fatalf("two-month occurrence = %+v, %v", occurrence, err)
	}
}

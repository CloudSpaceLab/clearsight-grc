package httpapi

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestRiskIndicatorMovementSkipsSamePeriodAmendment(t *testing.T) {
	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC)
	priorStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	priorEnd := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	measurement := func(value string, start, end *time.Time) *monitoring.NativeMeasurement {
		return &monitoring.NativeMeasurement{
			Field: "availability", Unit: monitoring.MeasurementPercent, Precision: 2, Value: value,
			ReportingPeriodStart: start, ReportingPeriodEnd: end,
		}
	}
	results := []monitoring.MonitoringResult{
		{EvaluatedAt: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC), Evaluation: monitoring.Evaluation{Measurement: measurement("99.70", &periodStart, &periodEnd)}},
		{EvaluatedAt: time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC), Evaluation: monitoring.Evaluation{Measurement: measurement("99.60", &periodStart, &periodEnd)}},
		{EvaluatedAt: time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC), Evaluation: monitoring.Evaluation{Measurement: measurement("99.50", &priorStart, &priorEnd)}},
	}
	movement, ok := riskIndicatorMovement(results)
	if !ok || movement.Direction != monitoring.MeasurementMovementUp || movement.Delta != "+0.20" ||
		!movement.PreviousEvaluatedAt.Equal(results[2].EvaluatedAt) {
		t.Fatalf("movement=%#v ok=%v", movement, ok)
	}
}

func TestRiskIndicatorMovementUsesObservationOrderWithoutPeriods(t *testing.T) {
	results := []monitoring.MonitoringResult{
		{EvaluatedAt: time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC), Evaluation: monitoring.Evaluation{Measurement: &monitoring.NativeMeasurement{Field: "exceptions", Unit: monitoring.MeasurementCount, Value: "4"}}},
		{EvaluatedAt: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC), Evaluation: monitoring.Evaluation{Measurement: &monitoring.NativeMeasurement{Field: "exceptions", Unit: monitoring.MeasurementCount, Value: "6"}}},
	}
	movement, ok := riskIndicatorMovement(results)
	if !ok || movement.Direction != monitoring.MeasurementMovementDown || movement.Delta != "-2" {
		t.Fatalf("movement=%#v ok=%v", movement, ok)
	}
}

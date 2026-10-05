package httpapi

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestCurrentRiskIndicatorLinksCollapseSupersededCheckRevisions(t *testing.T) {
	values := []risk.IndicatorLink{
		{ID: "link-1", RiskVersion: 2, MonitoringCheckID: "check-1", MonitoringCheckVersion: 1, Kind: risk.IndicatorKRI},
		{ID: "link-2", RiskVersion: 4, MonitoringCheckID: "check-1", MonitoringCheckVersion: 2, Kind: risk.IndicatorKRI},
		{ID: "link-3", RiskVersion: 3, MonitoringCheckID: "check-2", MonitoringCheckVersion: 1, Kind: risk.IndicatorKCI},
	}
	current := currentRiskIndicatorLinks(values)
	if len(current) != 2 {
		t.Fatalf("current indicators=%#v", current)
	}
	if current[0].ID != "link-2" || current[1].ID != "link-3" {
		t.Fatalf("current Indicator order/revision=%#v", current)
	}
}

func TestCurrentRiskIndicatorStatePreservesUnknownSemantics(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	check := monitoring.MonitoringCheck{
		ID: "check-1", TenantID: "bank", ProgramID: "program-1",
		Lifecycle:        monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3},
		FreshnessMinutes: 60, MinimumCoverage: 0.9,
	}
	score := 20.0
	base := monitoring.MonitoringResult{
		ID: "result-1", TenantID: "bank", ProgramID: "program-1",
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		Evaluation:  monitoring.Evaluation{Score: &score, Band: monitoring.RiskLow, Coverage: 1},
		EvaluatedAt: now.Add(-30 * time.Minute),
	}

	tests := []struct {
		name   string
		check  monitoring.MonitoringCheck
		result monitoring.MonitoringResult
		state  riskIndicatorState
	}{
		{name: "low is normal", check: check, result: base, state: riskIndicatorNormal},
		{name: "moderate is watch", check: check, result: withIndicatorBand(base, monitoring.RiskModerate), state: riskIndicatorWatch},
		{name: "high is breach", check: check, result: withIndicatorBand(base, monitoring.RiskHigh), state: riskIndicatorBreach},
		{name: "critical is breach", check: check, result: withIndicatorBand(base, monitoring.RiskCritical), state: riskIndicatorBreach},
		{name: "not assessed is unknown", check: check, result: withIndicatorBand(base, monitoring.RiskNotAssessed), state: riskIndicatorUnknown},
		{name: "stale is unknown", check: check, result: withIndicatorTime(base, now.Add(-61*time.Minute)), state: riskIndicatorUnknown},
		{name: "insufficient coverage is unknown", check: check, result: withIndicatorCoverage(base, 0.89), state: riskIndicatorUnknown},
		{name: "wrong revision is unknown", check: check, result: withIndicatorVersion(base, 2), state: riskIndicatorUnknown},
		{name: "paused check is unknown", check: withIndicatorStatus(check, monitoring.LifecyclePaused, false), result: base, state: riskIndicatorUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, reason := currentRiskIndicatorState(tt.check, tt.result, now)
			if state != tt.state {
				t.Fatalf("state=%s want=%s reason=%q", state, tt.state, reason)
			}
			if reason == "" {
				t.Fatal("indicator state reason is empty")
			}
		})
	}
}

func TestCurrentRiskIndicatorStateUsesNativeLimitBeforeConcernBand(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	check := monitoring.MonitoringCheck{
		ID: "check-native",
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 2},
		FreshnessMinutes: 60,
		MinimumCoverage:  1,
	}
	result := monitoring.MonitoringResult{
		MonitoringCheckID: check.ID,
		MonitoringCheckVersion: check.Version,
		EvaluatedAt: now.Add(-time.Minute),
		Evaluation: monitoring.Evaluation{
			Band: monitoring.RiskLow,
			Coverage: 1,
			Measurement: &monitoring.NativeMeasurement{
				Unit: monitoring.MeasurementPercent,
				Value: "98.70",
				Limits: []monitoring.MeasurementLimit{{Operator: monitoring.OperatorGreaterOrEqual, Expected: "99.50"}},
			},
		},
	}
	state, reason := currentRiskIndicatorState(check, result, now)
	if state != riskIndicatorBreach || reason != "Latest native measurement is outside its approved limit." {
		t.Fatalf("native breach state=%s reason=%q", state, reason)
	}

	result.Evaluation.Band = monitoring.RiskCritical
	result.Evaluation.Measurement.Value = "99.70"
	state, reason = currentRiskIndicatorState(check, result, now)
	if state != riskIndicatorNormal || reason != "Latest native measurement is within its approved limit." {
		t.Fatalf("native within state=%s reason=%q", state, reason)
	}
}

func withIndicatorBand(value monitoring.MonitoringResult, band monitoring.RiskBand) monitoring.MonitoringResult {
	value.Evaluation.Band = band
	return value
}

func withIndicatorTime(value monitoring.MonitoringResult, at time.Time) monitoring.MonitoringResult {
	value.EvaluatedAt = at
	return value
}

func withIndicatorCoverage(value monitoring.MonitoringResult, coverage float64) monitoring.MonitoringResult {
	value.Evaluation.Coverage = coverage
	return value
}

func withIndicatorVersion(value monitoring.MonitoringResult, version int64) monitoring.MonitoringResult {
	value.MonitoringCheckVersion = version
	return value
}

func withIndicatorStatus(value monitoring.MonitoringCheck, status monitoring.LifecycleStatus, current bool) monitoring.MonitoringCheck {
	value.Status = status
	value.IsCurrent = current
	return value
}

func TestRiskIndicatorMatterLinkedToProgramRequiresActiveProgramLink(t *testing.T) {
	linked := continuity.MatterAggregate{
		Links: []continuity.MatterLink{{ProgramID: "program-1"}},
	}
	if !riskIndicatorMatterLinkedToProgram(linked, "program-1") {
		t.Fatal("active Program link was not recognized")
	}

	retiredAt := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	linked.Links[0].RetiredAt = &retiredAt
	if riskIndicatorMatterLinkedToProgram(linked, "program-1") {
		t.Fatal("retired Program link was treated as current intervention")
	}
	if riskIndicatorMatterLinkedToProgram(linked, "program-2") {
		t.Fatal("wrong Program was treated as current intervention")
	}
}

func TestCurrentRiskIndicatorNativeMeasurementPrefersObservedValue(t *testing.T) {
	check := monitoring.MonitoringCheck{
		Measurement: &monitoring.MeasurementSpec{Field: "success_rate", Label: "Success rate", Unit: monitoring.MeasurementPercent, Precision: 2},
		SourceRules: []monitoring.SourceRule{{
			ID: "minimum", Field: "success_rate", Operator: monitoring.OperatorGreaterOrEqual, Expected: "99.50", RiskPoints: 100,
		}},
	}
	configured := currentRiskIndicatorNativeMeasurement(check, nil)
	if configured == nil || configured.Value != "" || len(configured.Limits) != 1 || configured.Limits[0].Expected != "99.50" {
		t.Fatalf("configured measurement=%#v", configured)
	}

	result := monitoring.MonitoringResult{Evaluation: monitoring.Evaluation{Measurement: &monitoring.NativeMeasurement{
		Field: "success_rate", Label: "Success rate", Unit: monitoring.MeasurementPercent, Precision: 2, Value: "98.70",
		Limits: []monitoring.MeasurementLimit{{Operator: monitoring.OperatorGreaterOrEqual, Expected: "99.50"}},
	}}}
	observed := currentRiskIndicatorNativeMeasurement(check, &result)
	if observed == nil || observed.Value != "98.70" || len(observed.Limits) != 1 {
		t.Fatalf("observed measurement=%#v", observed)
	}
	observed.Limits[0].Expected = "mutated"
	if result.Evaluation.Measurement.Limits[0].Expected != "99.50" {
		t.Fatal("indicator read mutated the retained monitoring result")
	}
}

package httpapi

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

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

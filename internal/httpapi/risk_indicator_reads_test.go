package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
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

func TestRiskIndicatorReadIncludesVisibleOpenMonitoringMatter(t *testing.T) {
	now := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	ctx := context.Background()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(ctx), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RESILIENCE", Name: "Network resilience",
		Type: "ASSURANCE", OwningFunction: "Technology", OwnerPrincipalID: "owner-1",
		AuthorityPrincipalID: "authorizer-1", Scope: json.RawMessage(`{}`), EffectiveFrom: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	monitorRepo := monitoring.NewMemoryRepository()
	monitorService := monitoring.NewService(monitorRepo, nil)
	check, err := monitorRepo.CreateCheckRevision(ctx, monitoring.MonitoringCheck{
		ID: "check-1", TenantID: "bank", ProgramID: program.Program.ID, Code: "FAILOVER", Name: "Failover health",
		Claim: "Failover remains within approved bounds.", InputKind: monitoring.InputSource,
		BindingID: "binding-1", BindingVersion: 1, Thresholds: monitoring.DefaultThresholds(),
		FreshnessMinutes: 60, MinimumCoverage: 0.95, OwnerPrincipalID: "owner-1", ReviewerPrincipalID: "reviewer-1",
		FailureAction: monitoring.FailureRecommendMatter,
		Lifecycle:     monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 2, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	score := 80.0
	result, err := monitorRepo.AppendResult(ctx, monitoring.MonitoringResult{
		ID: "result-1", TenantID: "bank", ProgramID: program.Program.ID,
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation:  monitoring.Evaluation{Score: &score, Band: monitoring.RiskCritical, Coverage: 1},
		EvaluatedAt: now, EvaluatorVersion: "risk-v1", CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"monitoring_result_id":     result.ID,
		"monitoring_check_id":      check.ID,
		"monitoring_check_version": check.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, matter, inserted, err := continuityService.ApplyTrigger(continuity.WithTrustedSystemScope(ctx), continuity.Trigger{
		TenantID: "bank", ProgramID: program.Program.ID, Type: "MONITORING_RESULT_ADVERSE",
		SubjectType: "MONITORING_RESULT", SubjectID: result.ID, DedupeKey: "monitoring-result-adverse:" + result.ID,
		Payload: payload, ObservedAt: now, Source: "monitoring-result-review", ActorID: "owner-1",
	})
	if err != nil || !inserted || matter == nil {
		t.Fatalf("create monitoring Matter inserted=%v matter=%#v err=%v", inserted, matter, err)
	}

	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	actorCtx := identity.WithActor(ctx, actor)
	api := &API{deps: Dependencies{Continuity: continuityService, Monitoring: monitorService}}
	read := api.riskAggregateWithDetails(actorCtx, actor, risk.Aggregate{
		Risk: risk.Risk{
			ID: "risk-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-1", Name: "Network resilience",
			Statement: "Network service may exceed tolerance.", Impact: "Critical service disruption.",
			Status: risk.StatusActive, Version: 2, CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
		},
		Indicators: []risk.IndicatorLink{{
			ID: "indicator-1", RiskID: "risk-1", RiskVersion: 2, ProgramID: program.Program.ID,
			MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version, Kind: risk.IndicatorKRI,
			Measurement: risk.IndicatorMonitoringRiskScore, CreatedAt: now,
		}},
	})
	if !read.IndicatorDetailsComplete || len(read.IndicatorDetails) != 1 {
		t.Fatalf("Indicator read incomplete: %#v", read)
	}
	detail := read.IndicatorDetails[0]
	if detail.OpenMatterID != matter.ID || detail.OpenMatterReference != matter.Reference || detail.OpenMatterStatus != matter.Status {
		t.Fatalf("open Matter read-through=%#v want=%#v", detail, matter)
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

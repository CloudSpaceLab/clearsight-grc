package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestIndicatorInsightsReturnsScopedNativeIndicatorOnce(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	programs := continuity.NewService(continuity.NewMemoryRepository())
	program, err := programs.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "CHANNELS", Name: "Channel assurance", Type: "CHANNEL",
		OwningFunction: "Technology", OwnerPrincipalID: "owner-a", AuthorityPrincipalID: "reviewer-a",
		Scope: json.RawMessage(`{}`), EffectiveFrom: now.Add(-24 * time.Hour), ActorID: "owner-a",
	})
	if err != nil {
		t.Fatal(err)
	}

	riskRepo := risk.NewMemoryRepository()
	current, err := riskRepo.Create(t.Context(), risk.Risk{
		ID: "risk-a", TenantID: "bank-a", LegalEntityID: "entity-a", Code: "CHANNEL-RISK", Name: "Channel availability",
		Statement: "Channel success may fall below the approved service level.", Impact: "Customer transactions may fail.",
		Scope: json.RawMessage(`{}`), Status: risk.StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now,
	}, risk.Event{RiskID: "risk-a", RiskVersion: 1, Type: risk.EventRiskCreated, OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = riskRepo.AddIndicator(t.Context(), risk.Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}, current.ID, current.Version, risk.IndicatorLink{
		ID: "indicator-link", RiskID: current.ID, RiskVersion: 2, ProgramID: program.Program.ID,
		MonitoringCheckID: "check-success", MonitoringCheckVersion: 3, Kind: risk.IndicatorKRI,
		Measurement: risk.IndicatorMonitoringRiskScore, CreatedAt: now,
	}, risk.Event{RiskID: current.ID, RiskVersion: 2, Type: risk.EventIndicatorLinked, OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}

	monitorRepo := monitoring.NewMemoryRepository()
	effectiveFrom := now.Add(-time.Hour)
	score := float64(100)
	check := monitoring.MonitoringCheck{
		ID: "check-success", TenantID: "bank-a", ProgramID: program.Program.ID, Code: "MOBILE-SUCCESS", Name: "Mobile success rate",
		Claim: "Mobile transaction success remains at or above the approved limit.", InputKind: monitoring.InputSource,
		BindingID: "binding-1", BindingVersion: 1,
		SourceRules: []monitoring.SourceRule{{ID: "minimum", Field: "success_rate", Operator: monitoring.OperatorGreaterOrEqual, Expected: "99.50", RiskPoints: 100}},
		Measurement: &monitoring.MeasurementSpec{Field: "success_rate", Label: "Success rate", Unit: monitoring.MeasurementPercent, Precision: 2},
		Thresholds: monitoring.DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: monitoring.FailureReview,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, EffectiveFrom: &effectiveFrom, Version: 3},
	}
	if _, err := monitorRepo.CreateCheckRevision(t.Context(), check); err != nil {
		t.Fatal(err)
	}
	if _, err := monitorRepo.AppendResult(t.Context(), monitoring.MonitoringResult{
		ID: "result-success", TenantID: "bank-a", ProgramID: program.Program.ID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation: monitoring.Evaluation{
			Score: &score, Band: monitoring.RiskCritical, Coverage: 1,
			Measurement: &monitoring.NativeMeasurement{
				Field: "success_rate", Label: "Success rate", Unit: monitoring.MeasurementPercent, Precision: 2,
				Value: "98.70", Limits: []monitoring.MeasurementLimit{{Operator: monitoring.OperatorGreaterOrEqual, Expected: "99.50"}},
				Condition: monitoring.MeasurementConditionBreached,
			},
		},
		EvaluatedAt: now.Add(-10 * time.Minute), EvaluatorVersion: "monitoring-risk-v1", CreatedAt: now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-a", "viewer", "entity-a", "CRO"),
		Risk: risk.NewService(riskRepo),
		Monitoring: monitoring.NewService(monitorRepo, nil),
		Continuity: programs,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/insights/indicators?kind=KRI&limit=25", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("indicator insights returned %d: %s", response.Code, response.Body.String())
	}
	var page indicatorInsightsPageRead
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if !page.Complete || len(page.Items) != 1 {
		t.Fatalf("indicator insights page = %#v", page)
	}
	item := page.Items[0]
	if item.Kind != risk.IndicatorKRI || item.RiskCount != 1 || item.CheckID != check.ID || item.State != riskIndicatorBreach {
		t.Fatalf("indicator insight = %#v", item)
	}
	if item.NativeMeasurement == nil || item.NativeMeasurement.Value != "98.70" || item.NativeMeasurement.Condition != monitoring.MeasurementConditionBreached {
		t.Fatalf("native measurement = %#v", item.NativeMeasurement)
	}
}

func TestIndicatorInsightsRequiresOversightPermission(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-a", "viewer", "entity-a"),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/insights/indicators", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("indicator insights without oversight permission returned %d: %s", response.Code, response.Body.String())
	}
}

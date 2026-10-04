package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

func TestHomeMetricsUseVerifiedActorLegalEntityAndPreserveCoverage(t *testing.T) {
	now := time.Now().UTC()
	unknown := 2
	excluded := 1
	repo := oversight.NewMemoryRepository([]oversight.Snapshot{
		{
			TenantID: "bank", LegalEntityID: "bank-ng", GeneratedAt: now,
			ProjectionVersion: oversight.ProjectionVersion,
			Coverage:          oversight.Coverage{Population: 20, Unknown: &unknown, Excluded: &excluded},
			Counts:            oversight.Counts{CriticalHigh: 4, Overdue: 2, RoutingFailures: 1, OutcomeFailures: 3},
		},
		{
			TenantID: "bank", LegalEntityID: "bank-gh", GeneratedAt: now,
			ProjectionVersion: oversight.ProjectionVersion,
			Coverage:          oversight.Coverage{Population: 99, Unknown: &unknown, Excluded: &excluded},
			Counts:            oversight.Counts{CriticalHigh: 99},
		},
	})
	handler := New(Dependencies{
		Logger:    slog.Default(),
		Identity:  identity.NewDevelopmentAuthenticator("bank", "cro-1", "bank-ng", "CRO"),
		Oversight: oversight.NewService(repo),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	var bundle metricview.Bundle
	if err := json.Unmarshal(response.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.ScopeID != "bank-ng" || bundle.Population != 20 || bundle.Completeness != metricview.CompletenessPartial {
		t.Fatalf("unexpected metric bundle: %#v", bundle)
	}
	if len(bundle.Items) != 4 || bundle.Items[0].Value != 4 || bundle.Items[0].Unknown == nil || *bundle.Items[0].Unknown != 2 {
		t.Fatalf("unexpected metric items: %#v", bundle.Items)
	}
}

func TestHomeMetricsDoNotGrantOversightToSystemAdministrator(t *testing.T) {
	handler := New(Dependencies{
		Logger:    slog.Default(),
		Identity:  identity.NewDevelopmentAuthenticator("bank", "admin-1", "bank-ng", "SYSTEM_ADMIN"),
		Oversight: oversight.NewService(oversight.NewMemoryRepository(nil)),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHomeMetricRecoveryStatesAreExplicit(t *testing.T) {
	for _, tt := range []struct {
		name          string
		service       *oversight.Service
		code, message string
	}{
		{"unconfigured", nil, "metrics_unavailable", "Risk metrics are unavailable. Try again."},
		{"uncalculated", oversight.NewService(oversight.NewMemoryRepository(nil)), "metrics_not_ready", "Risk metrics have not been calculated for this legal entity."},
		{"failed read", oversight.NewService(nil), "metrics_unavailable", "Risk metrics are unavailable. Try again."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{deps: Dependencies{Oversight: tt.service}}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home", nil)
			request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
				TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: time.Now().Add(time.Hour),
			}))
			response := httptest.NewRecorder()
			api.homeMetrics(response, request)
			assertAPIError(t, response, http.StatusServiceUnavailable, tt.code, tt.message)
		})
	}
}

func TestHomeMetricsShareRequestedPeriodAndMarkHeadlineMetricsCurrentPosture(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)
	repo := oversight.NewMemoryRepository(nil).WithPeriodBuilder(func(_ context.Context, scope oversight.Scope, start, end time.Time) (oversight.Snapshot, error) {
		unknown := 0
		return oversight.Snapshot{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, GeneratedAt: end,
			PeriodStart: start, PeriodEnd: end, PostureAsOf: end, ProjectionVersion: oversight.ProjectionVersion,
			Coverage: oversight.Coverage{Population: 8, Unknown: &unknown},
			Counts:   oversight.Counts{CriticalHigh: 3, Overdue: 2},
		}, nil
	})
	service := oversight.NewService(repo)
	service.Now = func() time.Time { return now }
	api := &API{deps: Dependencies{Oversight: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home?start_date=2026-08-02&end_date=2026-10-02", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetrics(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var bundle metricview.Bundle
	if err := json.Unmarshal(response.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.ReportingPeriod.StartDate != "2026-08-02" || bundle.ReportingPeriod.EndDate != "2026-10-02" {
		t.Fatalf("reporting period=%#v", bundle.ReportingPeriod)
	}
	for _, metric := range bundle.Items {
		if metric.Basis != metricview.MetricBasisCurrentPosture || metric.Drill.Consistency != metricview.DrillCurrentState {
			t.Fatalf("headline metric lost current-posture semantics: %#v", metric)
		}
	}
}


func TestHomeMetricsRetainOnDemandPopulationForExactDrill(t *testing.T) {
	now := time.Date(2026, 10, 4, 14, 30, 0, 0, time.UTC)
	repo := oversight.NewMemoryRepository(nil).WithPeriodBuilder(func(_ context.Context, scope oversight.Scope, start, end time.Time) (oversight.Snapshot, error) {
		unknown := 0
		return oversight.Snapshot{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, GeneratedAt: end,
			PeriodStart: start, PeriodEnd: end, PostureAsOf: end, ProjectionVersion: oversight.ProjectionVersion,
			Coverage: oversight.Coverage{Population: 3, Unknown: &unknown},
			Counts:   oversight.Counts{CriticalHigh: 1},
			MetricMembers: []oversight.MetricMember{{
				MetricID: "critical_high_open", MemberID: "8f620000-0000-4000-8000-000000000001",
				TargetType: "MATTER", TargetID: "8f620000-0000-4000-8000-000000000002",
				TargetTitle: "Review critical issue", State: "ASSESSMENT",
			}},
		}, nil
	})
	service := oversight.NewService(repo)
	service.Now = func() time.Time { return now }
	members := &metricMembershipReaderStub{retainSourceID: "8f620000-0000-4000-8000-000000000010"}
	api := &API{deps: Dependencies{Oversight: service, MetricMembership: members}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home?start_date=2026-09-01&end_date=2026-10-04", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetrics(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var bundle metricview.Bundle
	if err := json.Unmarshal(response.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.SourceID != members.retainSourceID || bundle.DefinitionRevision != metricview.HomeDefinitionRevision {
		t.Fatalf("exact runtime bundle=%#v", bundle)
	}
	for _, metric := range bundle.Items {
		if metric.Drill.Consistency != metricview.DrillSourceSnapshot ||
			metric.DefinitionRevision != metricview.HomeDefinitionRevision {
			t.Fatalf("metric did not bind exact runtime source: %#v", metric)
		}
	}
	if members.retained.TenantID != "bank" || members.retained.LegalEntityID != "bank-ng" ||
		members.retained.SnapshotID != "" || len(members.retained.MetricMembers) != 1 ||
		members.retained.MetricMembers[0].MetricID != "critical_high_open" {
		t.Fatalf("retained snapshot=%#v", members.retained)
	}
}

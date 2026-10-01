package httpapi

import (
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

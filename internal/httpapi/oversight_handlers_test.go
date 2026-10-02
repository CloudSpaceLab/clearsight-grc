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
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

func TestOversightReturnsOnlyVerifiedActorLegalEntitySnapshot(t *testing.T) {
	now := time.Now().UTC()
	repo := oversight.NewMemoryRepository([]oversight.Snapshot{
		{TenantID: "bank", LegalEntityID: "bank-ng", GeneratedAt: now, ProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{Overdue: 4}},
		{TenantID: "bank", LegalEntityID: "bank-gh", GeneratedAt: now, ProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{Overdue: 99}},
	})
	handler := New(Dependencies{
		Logger:    slog.Default(),
		Identity:  identity.NewDevelopmentAuthenticator("bank", "cro-1", "bank-ng", "CRO"),
		Oversight: oversight.NewService(repo),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value oversight.Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Counts.Overdue != 4 {
		t.Fatalf("cross-entity snapshot selected: %#v", value.Counts)
	}
}

func TestSystemAdministratorDoesNotGainRiskOversightFromPlatformAdministration(t *testing.T) {
	handler := New(Dependencies{
		Logger:    slog.Default(),
		Identity:  identity.NewDevelopmentAuthenticator("bank", "admin-1", "bank-ng", "SYSTEM_ADMIN"),
		Oversight: oversight.NewService(oversight.NewMemoryRepository(nil)),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOversightRecoveryDescribesUnavailableAndUncalculatedStates(t *testing.T) {
	for _, tt := range []struct {
		name          string
		service       *oversight.Service
		code, message string
	}{
		{"unconfigured", nil, "oversight_unavailable", "Oversight unavailable. Try again."},
		{"uncalculated", oversight.NewService(oversight.NewMemoryRepository(nil)), "oversight_not_ready", "Oversight has not been calculated for this legal entity."},
		{"failed read", oversight.NewService(nil), "oversight_unavailable", "Oversight unavailable. Try again."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{deps: Dependencies{Oversight: tt.service}}
			r := httptest.NewRequest(http.MethodGet, "/api/v1/oversight", nil)
			r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: time.Now().Add(time.Hour)}))
			w := httptest.NewRecorder()
			api.oversightSnapshot(w, r)
			assertAPIError(t, w, 503, tt.code, tt.message)
		})
	}
}

func TestOversightAcceptsBoundedCurrentReportingPeriod(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)
	repo := oversight.NewMemoryRepository(nil).WithPeriodBuilder(func(_ context.Context, scope oversight.Scope, start, end time.Time) (oversight.Snapshot, error) {
		return oversight.Snapshot{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			GeneratedAt: end, PeriodStart: start, PeriodEnd: end, PostureAsOf: end,
			ProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{Overdue: 4},
		}, nil
	})
	service := oversight.NewService(repo)
	service.Now = func() time.Time { return now }
	api := &API{deps: Dependencies{Oversight: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight?start_date=2026-09-01&end_date=2026-10-02", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.oversightSnapshot(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value oversight.Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.ReportingPeriod.StartDate != "2026-09-01" || value.ReportingPeriod.EndDate != "2026-10-02" || value.Counts.Overdue != 4 {
		t.Fatalf("snapshot=%#v", value)
	}
}

func TestOversightRejectsHistoricalReportingEnd(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)
	service := oversight.NewService(oversight.NewMemoryRepository(nil))
	service.Now = func() time.Time { return now }
	api := &API{deps: Dependencies{Oversight: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight?start_date=2026-09-01&end_date=2026-10-01", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.oversightSnapshot(response, request)
	assertAPIError(t, response, http.StatusBadRequest, "historical_period_end_unsupported", "Historical end dates are not available. Use the current reporting date as the end date.")
}

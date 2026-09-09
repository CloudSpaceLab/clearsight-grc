package httpapi

import (
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

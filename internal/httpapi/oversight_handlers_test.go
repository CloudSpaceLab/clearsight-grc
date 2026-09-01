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

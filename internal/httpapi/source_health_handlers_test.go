package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestSourceHealthRouteRequiresConfigRead(t *testing.T) {
	found := false
	for _, route := range (&API{}).routes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/config/sources/{source_id}/health" {
			found = true
			if route.Permission != identity.PermissionConfigRead {
				t.Fatalf("health route permission=%q want=%q", route.Permission, identity.PermissionConfigRead)
			}
		}
	}
	if !found {
		t.Fatal("scoped source health route is not registered")
	}
}

func TestSourceObservationIgnoresClientTenantAndRecorder(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	repo := evidence.NewMemoryRepository([]evidence.Source{{
		ID: "source-1", TenantID: "bank-a", Code: "CORE", Name: "Core", Type: evidence.SourceSystem,
		AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 30, Health: evidence.HealthUnknown,
		Status: evidence.SourceActive, Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}}, nil)
	service := evidence.NewService(repo, evidence.NewMemoryObjectStore())
	api := &API{deps: Dependencies{Evidence: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/sources/source-1/observations", strings.NewReader(`{
		"tenant_id":"bank-b","source_id":"other","scope":"SOURCE","success":true,"recorded_by":"spoofed"
	}`))
	request.SetPathValue("id", "source-1")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank-a", PrincipalID: "actor-a", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.recordEvidenceSourceObservation(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	values, err := service.ListSources(request.Context(), "bank-a", 10)
	if err != nil || len(values) != 1 || values[0].Health != evidence.HealthCurrent {
		t.Fatalf("verified tenant observation not recorded: values=%#v err=%v", values, err)
	}
	if foreign, _ := service.ListSources(request.Context(), "bank-b", 10); len(foreign) != 0 {
		t.Fatalf("client tenant affected another tenant: %#v", foreign)
	}
}

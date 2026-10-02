package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestRiskRoutesUseGovernedAuthorityContracts(t *testing.T) {
	want := map[string]struct {
		responsibility authority.Responsibility
		materiality    int
		bindEntity     bool
	}{
		"POST /api/v1/risks":                  {authority.ResponsibilityOwner, 3, true},
		"POST /api/v1/risks/{id}":             {authority.ResponsibilityOwner, 3, false},
		"POST /api/v1/risks/{id}/assessments": {authority.ResponsibilityReviewer, 3, false},
		"POST /api/v1/risks/{id}/appetite":    {authority.ResponsibilityAuthorizer, 4, false},
		"POST /api/v1/risks/{id}/controls":    {authority.ResponsibilityOwner, 3, false},
	}
	for _, route := range (&API{}).riskRoutes() {
		key := route.Method + " " + route.Path
		expected, ok := want[key]
		if !ok {
			if route.Class != routeAuthenticatedRead {
				t.Fatalf("unexpected risk route: %#v", route)
			}
			continue
		}
		if route.Class != routeMaterialCommand || route.Command == nil {
			t.Fatalf("%s is not a material command: %#v", key, route)
		}
		policy := route.Command.Policy
		if policy.ObjectType != "RISK" || policy.Responsibility != expected.responsibility ||
			policy.Materiality != expected.materiality || policy.BindLegalEntity != expected.bindEntity ||
			policy.ActorField != noActorField {
			t.Fatalf("%s policy = %#v", key, policy)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing risk routes: %#v", want)
	}
}

func TestRiskHTTPBindsVerifiedScopeAndActor(t *testing.T) {
	service := risk.NewService(risk.NewMemoryRepository())
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "risk-owner", "entity-a"),
		Risk:     service,
	})

	createBody := `{
		"code":"NET-RES-01",
		"name":"Network resilience",
		"category":"Operational resilience",
		"statement":"Critical network service may exceed approved recovery tolerance.",
		"impact":"Customers cannot access critical services within the approved tolerance.",
		"scope":{"service":"critical-network"},
		"owner_principal_id":"spoofed-owner",
		"actor_id":"spoofed-actor"
	}`
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/risks", strings.NewReader(createBody)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created risk.Risk
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "bank" || created.LegalEntityID != "entity-a" || created.OwnerPrincipalID != "risk-owner" {
		t.Fatalf("created scope/owner = %#v", created)
	}

	now = now.Add(time.Minute)
	appetiteBody := `{
		"expected_risk_version":1,
		"statement":"Keep recovery below 30 minutes.",
		"rule":{"max_minutes":30},
		"rationale":"Protect critical customer services.",
		"actor_id":"spoofed-authorizer"
	}`
	appetite := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/risks/"+created.ID+"/appetite", strings.NewReader(appetiteBody))
	handler.ServeHTTP(appetite, req)
	if appetite.Code != http.StatusCreated {
		t.Fatalf("appetite status=%d body=%s", appetite.Code, appetite.Body.String())
	}
	var appetiteResponse struct {
		Risk     risk.Risk              `json:"risk"`
		Appetite risk.AppetiteStatement `json:"appetite"`
	}
	if err := json.Unmarshal(appetite.Body.Bytes(), &appetiteResponse); err != nil {
		t.Fatal(err)
	}
	if appetiteResponse.Appetite.AuthorityPrincipalID != "risk-owner" {
		t.Fatalf("body actor redirected appetite authority: %#v", appetiteResponse.Appetite)
	}

	now = now.Add(time.Minute)
	assessmentBody := fmt.Sprintf(`{
		"expected_risk_version":2,
		"kind":"RESIDUAL",
		"method_code":"QUAL-5X5",
		"method_version":"v1",
		"dimensions":{"likelihood":4,"impact":5},
		"appetite_statement_id":%q,
		"appetite_position":"BREACHED",
		"appetite_rationale":"Recovery exceeds tolerance.",
		"actor_id":"spoofed-reviewer"
	}`, appetiteResponse.Appetite.ID)
	assessment := httptest.NewRecorder()
	handler.ServeHTTP(assessment, httptest.NewRequest(http.MethodPost, "/api/v1/risks/"+created.ID+"/assessments", strings.NewReader(assessmentBody)))
	if assessment.Code != http.StatusCreated {
		t.Fatalf("assessment status=%d body=%s", assessment.Code, assessment.Body.String())
	}
	var assessmentResponse struct {
		Assessment risk.Assessment `json:"assessment"`
	}
	if err := json.Unmarshal(assessment.Body.Bytes(), &assessmentResponse); err != nil {
		t.Fatal(err)
	}
	if assessmentResponse.Assessment.AssessedBy != "risk-owner" {
		t.Fatalf("body actor redirected assessment: %#v", assessmentResponse.Assessment)
	}
}

func TestRiskHTTPRejectsCrossEntityReadAsNotFound(t *testing.T) {
	repository := risk.NewMemoryRepository()
	service := risk.NewService(repository)
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC) }
	value, err := service.Create(t.Context(), risk.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-01", Name: "Scoped risk",
		Statement: "A scoped risk exists.", Impact: "Material impact.", Scope: json.RawMessage(`{"unit":"A"}`),
		ActorID: "owner-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "owner-b", "entity-b"),
		Risk:     service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/risks/"+value.ID, nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "risk_not_found") {
		t.Fatalf("cross-entity response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRiskHTTPListUsesVerifiedEntityAndBoundedFilters(t *testing.T) {
	service := risk.NewService(risk.NewMemoryRepository())
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	for _, entity := range []string{"entity-a", "entity-b"} {
		if _, err := service.Create(t.Context(), risk.CreateInput{
			TenantID: "bank", LegalEntityID: entity, Code: "RISK-" + entity, Name: "Network resilience",
			Category: "Operational resilience", Statement: "Service interruption risk.", Impact: "Customer impact.",
			Scope: json.RawMessage(`{"service":"network"}`),
		}); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Minute)
	}
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "reader-a", "entity-a"),
		Risk:     service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/risks?search=network&limit=25", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", response.Code, response.Body.String())
	}
	var page risk.Page
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Risk.LegalEntityID != "entity-a" {
		t.Fatalf("list leaked another entity: %#v", page)
	}

	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/api/v1/risks?limit=500", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}

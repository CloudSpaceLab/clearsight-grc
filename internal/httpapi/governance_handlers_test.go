package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func governanceHandler() http.Handler {
	return New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Mode:       "memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank-demo", "maker", "bank-ng", "GRC_ADMIN"),
		Governance: governance.NewService(governance.NewMemoryRepository()),
	})
}

func TestGovernancePolicyMakerCheckerHTTP(t *testing.T) {
	handler := governanceHandler()
	create := []byte(`{"tenant_id":"bank-demo","code":"risk","name":"Risk routing","maker_id":"forged","definition":{"rules":[{"id":"r1","legal_entity_id":"bank-ng","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/governance/policies", bytes.NewReader(create)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var policy governance.RoutingPolicy
	if err := json.NewDecoder(response.Body).Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if policy.MakerID != "maker" {
		t.Fatalf("maker was not bound from verified identity: %#v", policy)
	}
	if policy.TenantID != "bank-demo" || policy.LegalEntityID != "bank-ng" {
		t.Fatalf("governance scope was not bound from verified identity: %#v", policy)
	}
	transition := func(action, actor string, version int64) *httptest.ResponseRecorder {
		body, _ := json.Marshal(governance.TransitionInput{TenantID: "bank-demo", ActorID: "forged", ExpectedVersion: version})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/governance/policies/"+policy.ID+"/"+action, bytes.NewReader(body))
		request.Header.Set("X-ClearSight-Demo-Principal", actor)
		value := httptest.NewRecorder()
		handler.ServeHTTP(value, request)
		return value
	}
	response = transition("submit", "maker", policy.Version)
	if response.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(&policy); err != nil {
		t.Fatal(err)
	}
	response = transition("approve", "maker", policy.Version)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected maker-checker failure, got %d", response.Code)
	}
	response = transition("approve", "checker", policy.Version)
	if response.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", response.Code, response.Body.String())
	}
}

func TestGovernanceInventoryUsesVerifiedEntityWithoutClientScope(t *testing.T) {
	handler := governanceHandler()
	create := []byte(`{"code":"risk","name":"Risk routing","definition":{"rules":[{"id":"r1","legal_entity_id":"bank-ng","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}}`)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/governance/policies", bytes.NewReader(create)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/governance/policies?limit=25", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list: %d %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []governance.RoutingPolicy `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].LegalEntityID != "bank-ng" {
		t.Fatalf("inventory escaped verified entity: %#v", body.Items)
	}
}

func TestGovernanceDelegationCandidatesUseVerifiedEntityResponsibilityAndSafeLabels(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "memory",
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "maker", "bank-ng", "GRC_ADMIN"),
		Governance: governance.NewService(governance.NewMemoryRepositoryWithDelegationCandidates([]governance.DelegationCandidateDirectoryEntry{
			{PrincipalID: "ada", DisplayName: "Ada Okafor", ContextLabel: "Risk assurance lead", TenantID: "bank-demo", LegalEntityID: "bank-ng", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: true},
			{PrincipalID: "foreign", DisplayName: "Foreign Person", TenantID: "bank-demo", LegalEntityID: "bank-gh", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: true},
		})),
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/governance/delegation-candidates?responsibility=REVIEWER&q=assurance&limit=50", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("candidate read: %d %s", response.Code, response.Body.String())
	}
	raw := response.Body.String()
	if !strings.Contains(raw, `"display_name":"Ada Okafor"`) || strings.Contains(raw, "Foreign Person") {
		t.Fatalf("candidate read escaped verified scope: %s", raw)
	}
	for _, internal := range []string{"tenant_id", "legal_entity_id", "responsibilities", "active"} {
		if strings.Contains(raw, internal) {
			t.Fatalf("candidate read exposed %s: %s", internal, raw)
		}
	}
}

func TestGovernanceDelegationCandidatesRejectInvalidSearchAndFailClosedWhenUnavailable(t *testing.T) {
	configured := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank-demo", "maker", "bank-ng", "GRC_ADMIN"),
		Governance: governance.NewService(governance.NewMemoryRepositoryWithDelegationCandidates(nil)),
	})
	response := httptest.NewRecorder()
	configured.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/governance/delegation-candidates?responsibility=ADMIN", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid responsibility: %d %s", response.Code, response.Body.String())
	}

	unavailable := governanceHandler()
	response = httptest.NewRecorder()
	unavailable.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/governance/delegation-candidates?responsibility=REVIEWER", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable candidate directory: %d %s", response.Code, response.Body.String())
	}
}

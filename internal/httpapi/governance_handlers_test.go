package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

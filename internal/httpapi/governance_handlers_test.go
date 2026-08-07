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
	create := []byte(`{"tenant_id":"bank-demo","code":"risk","name":"Risk routing","maker_id":"forged","definition":{"rules":[{"id":"r1","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}}`)
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

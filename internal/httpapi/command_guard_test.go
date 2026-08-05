package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type commandAuthorityStub struct{ principal string }

func (s commandAuthorityStub) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return authority.Resolution{Principal: authority.Principal{ID: s.principal}, RuleID: "route-1", PolicyVersion: "v1"}, nil
}
func (s commandAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (s commandAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (s commandAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestCommandBindsSignedActor(t *testing.T) {
	guard, err := commandauth.New(commandAuthorityStub{principal: "person-1"}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{CommandGuard: guard}}
	var received map[string]any
	handler := api.command("matter.decision.record", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/matters/matter-1/decisions", strings.NewReader(`{"tenant_id":"bank-demo","authority_principal_id":"forged","rationale":"Approved"}`))
	req.SetPathValue("id", "matter-1")
	now := time.Now().UTC()
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "person-1", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()
	handler(response, req)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	if received["authority_principal_id"] != "person-1" || received["actor_id"] != "person-1" {
		t.Fatalf("actor fields were not bound to verified identity: %#v", received)
	}
}

func TestCommandRejectsTenantMismatch(t *testing.T) {
	guard, _ := commandauth.New(commandAuthorityStub{principal: "person-1"}, commandauth.ModeEnforce, slog.Default())
	api := &API{deps: Dependencies{CommandGuard: guard}}
	handler := api.command("matter.create", func(http.ResponseWriter, *http.Request) { t.Fatal("handler should not run") })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/matters", strings.NewReader(`{"tenant_id":"other-bank","title":"Test"}`))
	now := time.Now().UTC()
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "person-1", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()
	handler(response, req)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "outside your signed-in bank scope") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}

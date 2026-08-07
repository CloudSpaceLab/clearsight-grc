package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMatterCommandUsesRouteLifecycleResponsibilityAndPriorityFloor(t *testing.T) {
	ctx := t.Context()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := continuityService.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 5,
		Title: "Material regulatory change", Summary: "Test route-bound authority.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	resolver := &capturingCommandAuthority{}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: continuityService, CommandGuard: guard}}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, ActorField: "authority_principal_id"}

	var received map[string]any
	handler := api.command("matter.decision.record", policy, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/decisions", strings.NewReader(`{"tenant_id":"bank","type":"POSITION","status":"PROPOSED","rationale":"Proposed position"}`))
	request.SetPathValue("id", matter.Matter.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "person-1", LegalEntityID: "*", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour),
	}))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
	if resolver.input.ObjectID != matter.Matter.ID || resolver.input.Responsibility != authority.ResponsibilityProposer || resolver.input.Materiality != 5 {
		t.Fatalf("route-bound authority was not applied: %#v", resolver.input)
	}
	if received["authority_principal_id"] != "person-1" {
		t.Fatalf("verified actor was not rebound after lifecycle policy resolution: %#v", received)
	}
}

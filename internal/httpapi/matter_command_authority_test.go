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
	ctx := continuity.WithTrustedSystemScope(t.Context())
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := continuityService.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 5,
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

func TestMatterTransitionServerPermitsTheCurrentActorForEachLifecycleRoute(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(t.Context())
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "authorizer-1", DisplayName: "CCO"}},
	}}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	handler := api.command("matter.transition", commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, test := range []struct {
		name      string
		actorID   string
		target    continuity.MatterStatus
		wantRoute authority.Responsibility
	}{
		{name: "ordinary transition", actorID: "owner-1", target: continuity.MatterInitialReview, wantRoute: authority.ResponsibilityOwner},
		{name: "governed cancellation", actorID: "authorizer-1", target: continuity.MatterCancelled, wantRoute: authority.ResponsibilityAuthorizer},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/transition", strings.NewReader(`{"expected_version":1,"to":"`+string(test.target)+`","rationale":"Current route confirmed."}`))
			request.SetPathValue("id", matter.Matter.ID)
			request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
				TenantID: "bank", PrincipalID: test.actorID, LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour),
			}))
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("current %s was rejected: %d %s", test.wantRoute, response.Code, response.Body.String())
			}
			if resolver.resolutions[test.wantRoute].Principal.ID != test.actorID {
				t.Fatalf("test route does not distinguish actor from responsibility: %#v", test)
			}
		})
	}
}

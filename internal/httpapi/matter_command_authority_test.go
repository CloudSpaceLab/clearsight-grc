package httpapi

import (
	"encoding/json"
	"fmt"
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

func TestMatterActionTransitionRequiresStoredCurrentOwnerWithinPerformerRoute(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(t.Context())
	repository := continuity.NewMemoryRepository()
	service := continuity.NewService(repository)
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "matter-owner", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(ctx, continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "performer-a", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID

	// Both people remain eligible on the current PERFORMER route. Assignment of
	// this exact Action determines which one may execute its status transition.
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityPerformer: {
			Principal:           authority.Principal{ID: "performer-a", DisplayName: "Annual return lead"},
			CandidatePrincipals: []authority.Principal{{ID: "performer-b", DisplayName: "Compliance operations analyst"}},
		},
	}}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	handler := api.command(
		"matter.action.transition",
		commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
		api.transitionMatterAction,
	)

	transition := func(actorID string, expectedVersion int64) *httptest.ResponseRecorder {
		body := strings.NewReader(`{"expected_version":` + fmt.Sprint(expectedVersion) + `,"to":"IN_PROGRESS","actor_id":"performer-a","rationale":"Work started."}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions/"+actionID+"/transition", body)
		request.SetPathValue("id", matter.Matter.ID)
		request.SetPathValue("action_id", actionID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", PrincipalID: actorID, LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour),
		}))
		response := httptest.NewRecorder()
		handler(response, request)
		return response
	}

	response := transition("performer-b", matter.Matter.Version)
	if response.Code != http.StatusForbidden {
		t.Fatalf("eligible but unassigned performer changed the Action: %d %s", response.Code, response.Body.String())
	}
	current, err := service.GetMatter(ctx, "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Matter.Version != matter.Matter.Version || current.Actions[0].Status != continuity.ActionPlanned {
		t.Fatalf("rejected transition changed authoritative state: %#v", current.Actions[0])
	}

	reassigned, err := service.AssignAction(ctx, continuity.AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: current.Matter.Version,
		OwnerPrincipalID: "performer-b", ActorID: "matter-owner", Rationale: "Move the work to the current operations analyst.",
	})
	if err != nil {
		t.Fatal(err)
	}
	response = transition("performer-b", reassigned.Matter.Version)
	if response.Code != http.StatusOK {
		t.Fatalf("newly assigned performer could not transition the Action: %d %s", response.Code, response.Body.String())
	}
	current, err = service.GetMatter(ctx, "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Actions[0].OwnerPrincipalID != "performer-b" || current.Actions[0].Status != continuity.ActionInProgress {
		t.Fatalf("assigned performer transition was not applied: %#v", current.Actions[0])
	}
	events, err := repository.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := events[len(events)-1].ActorID; got != "performer-b" {
		t.Fatalf("transition trusted the client actor field instead of verified identity: %q", got)
	}
}

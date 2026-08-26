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
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type programCommandAuthority struct {
	inputs []authority.ResolveInput
}

type fixedProgramAuthority struct {
	resolution authority.Resolution
}

func (s fixedProgramAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return s.resolution, nil
}
func (s fixedProgramAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (s fixedProgramAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (s fixedProgramAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func (s *programCommandAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.inputs = append(s.inputs, input)
	return authority.Resolution{
		Principal:           authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"},
		CandidatePrincipals: []authority.Principal{{ID: "deputy-cro", DisplayName: "Deputy Chief Risk Officer"}},
		RuleID:              "route-1", PolicyVersion: "v1",
	}, nil
}
func (s *programCommandAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (s *programCommandAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (s *programCommandAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func newAuthorityTestProgram(t *testing.T) (*continuity.Service, continuity.ProgramAggregate) {
	t.Helper()
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Privacy", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1",
		Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, program
}

func TestProgramMaterialDecisionsRequireStoredAuthorityWithinCurrentRoute(t *testing.T) {
	service, program := newAuthorityTestProgram(t)
	resolver := &programCommandAuthority{}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}

	for _, command := range []string{"program.transition", "program.applicability.decide"} {
		t.Run(command, func(t *testing.T) {
			called := false
			handler := api.command(command, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID, strings.NewReader(`{"expected_version":1}`))
			request.SetPathValue("id", program.Program.ID)
			request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
				TenantID: "bank", PrincipalID: "deputy-cro", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
			}))
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != http.StatusForbidden || called {
				t.Fatalf("eligible but unassigned candidate executed %s: %d %s", command, response.Code, response.Body.String())
			}
		})
	}
}

func TestProgramMaterialDecisionAllowsDelegateOnlyForStoredAuthority(t *testing.T) {
	service, program := newAuthorityTestProgram(t)
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"},
		CandidatePrincipals: []authority.Principal{
			{ID: "cro-1", DisplayName: "Chief Risk Officer"},
			{ID: "delegate-1", DisplayName: "Acting Chief Risk Officer"},
			{ID: "other-candidate", DisplayName: "Another eligible authorizer"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "cro-1", OriginPrincipalID: "cro-1"},
			{PrincipalID: "delegate-1", OriginPrincipalID: "cro-1"},
			{PrincipalID: "other-candidate", OriginPrincipalID: "other-candidate"},
		},
		RuleID: "route-1", PolicyVersion: "v1",
	}}
	guard, _ := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	handler := api.command("program.transition", commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	decide := func(actorID string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID, strings.NewReader(`{"expected_version":1}`))
		request.SetPathValue("id", program.Program.ID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", PrincipalID: actorID, LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
		}))
		response := httptest.NewRecorder()
		handler(response, request)
		return response
	}

	if delegated := decide("delegate-1"); delegated.Code != http.StatusNoContent {
		t.Fatalf("delegate acting for the stored authority was rejected: %d %s", delegated.Code, delegated.Body.String())
	}
	if unrelated := decide("other-candidate"); unrelated.Code != http.StatusForbidden {
		t.Fatalf("unrelated route candidate acted for stored authority: %d %s", unrelated.Code, unrelated.Body.String())
	}
}

func TestProgramMaterialCommandRebindsWildcardIdentityToRecordEntity(t *testing.T) {
	service, program := newAuthorityTestProgram(t)
	resolver := &programCommandAuthority{}
	guard, _ := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	handler := api.command("program.transition", commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID, strings.NewReader(`{"expected_version":1}`))
	request.SetPathValue("id", program.Program.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "cro-1", LegalEntityID: "*", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("wildcard oversight command returned %d: %s", response.Code, response.Body.String())
	}
	for _, input := range resolver.inputs {
		if input.LegalEntityID != "entity-a" {
			t.Fatalf("Program authority resolved outside the record entity: %#v", resolver.inputs)
		}
	}
}

func TestProgramMaterialCommandRejectsWildcardEntityOverride(t *testing.T) {
	service, program := newAuthorityTestProgram(t)
	resolver := &programCommandAuthority{}
	guard, _ := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	called := false
	handler := api.command("program.transition", commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID, strings.NewReader(`{"legal_entity_id":"entity-b","expected_version":1}`))
	request.SetPathValue("id", program.Program.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "cro-1", LegalEntityID: "*", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("wildcard actor redirected Program authority to another entity: %d %s", response.Code, response.Body.String())
	}
}

func TestProgramAuthorityHandoffTakesEffectForTheNextDecision(t *testing.T) {
	service, program := newAuthorityTestProgram(t)
	resolver := &programCommandAuthority{}
	guard, _ := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}

	handoff := api.command("program.approval-authority.assign", commandPolicy{
		ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, DecisionType: "program.transition",
	}, api.assignProgramApprovalAuthority)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/approval-authority", strings.NewReader(`{"expected_version":1,"candidate_id":"deputy-cro","rationale":"The deputy now holds this Program authority."}`))
	request.SetPathValue("id", program.Program.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "cro-1", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()
	handoff(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authority handoff returned %d: %s", response.Code, response.Body.String())
	}

	transition := api.command("program.transition", commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	decide := func(actorID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/transition", strings.NewReader(`{"expected_version":2}`))
		req.SetPathValue("id", program.Program.ID)
		req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{
			TenantID: "bank", PrincipalID: actorID, LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
		}))
		result := httptest.NewRecorder()
		transition(result, req)
		return result
	}
	if old := decide("cro-1"); old.Code != http.StatusForbidden {
		t.Fatalf("previous authority retained Program decision access: %d %s", old.Code, old.Body.String())
	}
	if current := decide("deputy-cro"); current.Code != http.StatusNoContent {
		t.Fatalf("new stored authority could not use the current route: %d %s", current.Code, current.Body.String())
	}
}

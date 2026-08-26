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

type lifecycleAuthorityCapture struct {
	inputs []authority.ResolveInput
}

func (s *lifecycleAuthorityCapture) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.inputs = append(s.inputs, input)
	principalID := "program-owner"
	switch input.Responsibility {
	case authority.ResponsibilityPerformer:
		principalID = "safeguard-owner"
	case authority.ResponsibilityReviewer:
		principalID = "evidence-reviewer"
	}
	return authority.Resolution{
		Principal: authority.Principal{ID: principalID, DisplayName: principalID},
		CandidatePrincipals: []authority.Principal{
			{ID: principalID, DisplayName: principalID},
			{ID: "replacement-owner", DisplayName: "Replacement owner"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}},
	}, nil
}

func (s *lifecycleAuthorityCapture) ResolveMany(ctx context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	result := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		resolution, err := s.Resolve(ctx, input)
		result[index] = authority.ResolveOutcome{Resolution: resolution, Err: err}
	}
	return result, nil
}

func (s *lifecycleAuthorityCapture) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s *lifecycleAuthorityCapture) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s *lifecycleAuthorityCapture) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestProgramResourceLifecycleCommandsAuthorizeExactObjectAndResponsibility(t *testing.T) {
	service, program := programWithLifecycleResources(t)
	safeguardID := program.ControlImplementations[0].ID
	evidenceID := program.EvidenceContracts[0].ID

	tests := []struct {
		name           string
		command        string
		policy         commandPolicy
		path           string
		pathValues     map[string]string
		body           string
		actorID        string
		wantObjectType string
		wantObjectID   string
		wantRole       authority.Responsibility
	}{
		{
			name: "revise safeguard remains a Program owner command", command: "program.safeguard.update",
			policy:     commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
			path:       "/api/v1/programs/" + program.Program.ID + "/control-implementations/" + safeguardID + "/details",
			pathValues: map[string]string{"id": program.Program.ID, "implementation_id": safeguardID},
			body:       `{"expected_version":3,"expected_implementation_version":1}`,
			actorID:    "program-owner", wantObjectType: "PROGRAM", wantObjectID: program.Program.ID, wantRole: authority.ResponsibilityOwner,
		},
		{
			name: "assign safeguard remains a Program owner command", command: "program.safeguard.assign",
			policy:     commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
			path:       "/api/v1/programs/" + program.Program.ID + "/control-implementations/" + safeguardID + "/assignment",
			pathValues: map[string]string{"id": program.Program.ID, "implementation_id": safeguardID},
			body:       `{"expected_version":3,"expected_implementation_version":1,"owner_principal_id":"replacement-owner"}`,
			actorID:    "program-owner", wantObjectType: "PROGRAM", wantObjectID: program.Program.ID, wantRole: authority.ResponsibilityOwner,
		},
		{
			name: "transition safeguard is a performer command on the safeguard", command: "program.safeguard.transition",
			policy:     commandPolicy{ObjectType: "CONTROL_IMPLEMENTATION", ObjectIDPath: "implementation_id", Responsibility: authority.ResponsibilityPerformer, Materiality: 3},
			path:       "/api/v1/programs/" + program.Program.ID + "/control-implementations/" + safeguardID + "/transition",
			pathValues: map[string]string{"id": program.Program.ID, "implementation_id": safeguardID},
			body:       `{"expected_version":3,"expected_implementation_version":1,"to":"IN_PROGRESS"}`,
			actorID:    "safeguard-owner", wantObjectType: "CONTROL_IMPLEMENTATION", wantObjectID: safeguardID, wantRole: authority.ResponsibilityPerformer,
		},
		{
			name: "transition evidence check is a reviewer command", command: "program.evidence.transition",
			policy:     commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
			path:       "/api/v1/programs/" + program.Program.ID + "/evidence-contracts/" + evidenceID + "/transition",
			pathValues: map[string]string{"id": program.Program.ID, "contract_id": evidenceID},
			body:       `{"expected_version":3,"expected_contract_version":1,"to":"ACTIVE"}`,
			actorID:    "evidence-reviewer", wantObjectType: "PROGRAM", wantObjectID: program.Program.ID, wantRole: authority.ResponsibilityReviewer,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := &lifecycleAuthorityCapture{}
			guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
			if err != nil {
				t.Fatal(err)
			}
			api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
			called := false
			handler := api.command(test.command, test.policy, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			for key, value := range test.pathValues {
				request.SetPathValue(key, value)
			}
			request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
				TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: test.actorID, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
			}))
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != http.StatusNoContent || !called {
				t.Fatalf("command returned %d: %s", response.Code, response.Body.String())
			}
			found := false
			for _, input := range resolver.inputs {
				if input.DecisionType != test.command || input.Responsibility != test.wantRole {
					continue
				}
				found = true
				if input.ObjectType != test.wantObjectType || input.ObjectID != test.wantObjectID {
					t.Fatalf("authority input = %#v, want %s/%s", input, test.wantObjectType, test.wantObjectID)
				}
			}
			if !found {
				t.Fatalf("authority call for %s was not captured: %#v", test.command, resolver.inputs)
			}
		})
	}
}

func TestProgramResourceLifecycleOperationsExposeCurrentAssignmentsAndTargets(t *testing.T) {
	_, program := programWithLifecycleResources(t)
	resolver := &lifecycleAuthorityCapture{}
	api := &API{deps: Dependencies{Authority: resolver}}
	find := func(t *testing.T, operations []RecordOperation, command, subresourceID string) RecordOperation {
		t.Helper()
		for _, operation := range operations {
			if operation.Command == command && operation.SubresourceID == subresourceID {
				return operation
			}
		}
		t.Fatalf("operation %s/%s not found", command, subresourceID)
		return RecordOperation{}
	}
	build := func(principalID string) []RecordOperation {
		actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: principalID}
		return api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, time.Now().UTC()).Operations
	}

	safeguardID := program.ControlImplementations[0].ID
	evidenceID := program.EvidenceContracts[0].ID
	ownerOperations := build("program-owner")
	for _, command := range []string{"program.safeguard.update", "program.safeguard.assign", "program.evidence.revise"} {
		subresourceID := safeguardID
		if command == "program.evidence.revise" {
			subresourceID = evidenceID
		}
		operation := find(t, ownerOperations, command, subresourceID)
		if !operation.CanAct || operation.AssignedTo == nil || operation.AssignedTo.ID != "program-owner" {
			t.Fatalf("owner operation %s = %#v", command, operation)
		}
	}
	safeguardTransition := find(t, ownerOperations, "program.safeguard.transition", safeguardID)
	if safeguardTransition.CanAct || safeguardTransition.AssignedTo == nil || safeguardTransition.AssignedTo.ID != "safeguard-owner" || len(safeguardTransition.AllowedTargets) == 0 {
		t.Fatalf("safeguard transition = %#v", safeguardTransition)
	}
	evidenceTransition := find(t, ownerOperations, "program.evidence.transition", evidenceID)
	if evidenceTransition.CanAct || evidenceTransition.AssignedTo == nil || evidenceTransition.AssignedTo.ID != "evidence-reviewer" || len(evidenceTransition.AllowedTargets) == 0 {
		t.Fatalf("evidence transition = %#v", evidenceTransition)
	}

	if operation := find(t, build("safeguard-owner"), "program.safeguard.transition", safeguardID); !operation.CanAct {
		t.Fatalf("safeguard owner cannot transition: %#v", operation)
	}
	if operation := find(t, build("evidence-reviewer"), "program.evidence.transition", evidenceID); !operation.CanAct {
		t.Fatalf("evidence reviewer cannot transition: %#v", operation)
	}
}

func programWithLifecycleResources(t *testing.T) (*continuity.Service, continuity.ProgramAggregate) {
	t.Helper()
	service := continuity.NewService(continuity.NewMemoryRepository())
	ctx := continuity.WithTrustedSystemScope(t.Context())
	now := time.Now().UTC()
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RESOURCE", Name: "Resource lifecycle", Type: "ASSURANCE",
		OwningFunction: "Risk", OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer", EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, continuity.AddControlObjectiveInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "OBJ", Name: "Objective", Outcome: "The intended outcome remains controlled.", Status: continuity.ObjectiveActive, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, continuity.AddControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ObjectiveID: program.ControlObjectives[0].ID, Name: "Safeguard", Description: "Operate the safeguard.", ImplementationType: "REVIEW",
		OwnerPrincipalID: "safeguard-owner", Scope: json.RawMessage(`{}`), Status: continuity.ImplementationPlanned, EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, continuity.AddEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ControlImplementationID: program.ControlImplementations[0].ID, Code: "CHECK", Name: "Evidence check", Claim: "Evidence remains current.",
		FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: continuity.EvidenceContractDraft, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, program
}

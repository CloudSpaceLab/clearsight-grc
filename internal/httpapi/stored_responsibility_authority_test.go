package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type unassignedRecoveryAuthority struct{}

func (unassignedRecoveryAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	if input.Responsibility == authority.ResponsibilityAuthorizer {
		return authority.Resolution{Principal: authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "cro-1", OriginPrincipalID: "cro-1"}}}, nil
	}
	return authority.Resolution{Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}, CandidatePrincipals: []authority.Principal{{ID: "owner-1", DisplayName: "Program Owner"}}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"}}}, nil
}
func (r unassignedRecoveryAuthority) ResolveMany(ctx context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	result := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		result[index].Resolution, result[index].Err = r.Resolve(ctx, input)
	}
	return result, nil
}
func (unassignedRecoveryAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (unassignedRecoveryAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (unassignedRecoveryAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestStoredProgramOwnerOrDelegateRequiredForOwnerCommands(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Privacy", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1",
		Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "owner-1", DisplayName: "Program Owner"},
			{ID: "owner-delegate", DisplayName: "Acting Program Owner"},
			{ID: "other-owner-candidate", DisplayName: "Another eligible owner"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"},
			{PrincipalID: "owner-delegate", OriginPrincipalID: "owner-1"},
			{PrincipalID: "other-owner-candidate", OriginPrincipalID: "other-owner-candidate"},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}

	commands := []string{"program.details.update", "program.requirement.add", "program.safeguard.define", "program.evidence.define"}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			check := func(actorID string) error {
				request := httptest.NewRequest("POST", "/api/v1/programs/"+program.Program.ID, nil)
				request.SetPathValue("id", program.Program.ID)
				request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID}))
				_, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", command, map[string]any{}, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2})
				return err
			}
			if err := check("owner-delegate"); err != nil {
				t.Fatalf("stored owner's delegate was rejected: %v", err)
			}
			if err := check("other-owner-candidate"); !errors.Is(err, commandauth.ErrNotAuthorized) {
				t.Fatalf("unassigned owner candidate was not rejected: %v", err)
			}
		})
	}
}

func TestStoredMatterOwnerOrDelegateRequiredForOwnerCommands(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterRegulatoryChange, Priority: 3,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "owner-1"}, {ID: "owner-delegate"}, {ID: "other-owner-candidate"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"},
			{PrincipalID: "owner-delegate", OriginPrincipalID: "owner-1"},
			{PrincipalID: "other-owner-candidate", OriginPrincipalID: "other-owner-candidate"},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}

	commands := []string{"matter.details.update", "matter.context.change", "matter.link", "matter.action.add"}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			check := func(actorID string) error {
				request := httptest.NewRequest("POST", "/api/v1/matters/"+matter.Matter.ID, nil)
				request.SetPathValue("id", matter.Matter.ID)
				request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID}))
				_, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", command, map[string]any{}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2})
				return err
			}
			if err := check("owner-delegate"); err != nil {
				t.Fatalf("stored Matter owner's delegate was rejected: %v", err)
			}
			if err := check("other-owner-candidate"); !errors.Is(err, commandauth.ErrNotAuthorized) {
				t.Fatalf("unassigned Matter owner candidate was not rejected: %v", err)
			}
		})
	}
}

func TestStoredActionPerformerDelegateCanTransitionButOtherCandidateCannot(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterRegulatoryChange, Priority: 3,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "performer-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal:           authority.Principal{ID: "performer-1"},
		CandidatePrincipals: []authority.Principal{{ID: "performer-1"}, {ID: "performer-delegate"}, {ID: "other-performer"}},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "performer-1", OriginPrincipalID: "performer-1"},
			{PrincipalID: "performer-delegate", OriginPrincipalID: "performer-1"},
			{PrincipalID: "other-performer", OriginPrincipalID: "other-performer"},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	check := func(actorID string) error {
		request := httptest.NewRequest("POST", "/api/v1/matters/"+matter.Matter.ID+"/actions/"+actionID+"/transition", nil)
		request.SetPathValue("id", matter.Matter.ID)
		request.SetPathValue("action_id", actionID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID}))
		_, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.action.transition", map[string]any{"to": "IN_PROGRESS"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2})
		return err
	}
	if err := check("performer-delegate"); err != nil {
		t.Fatalf("stored performer's delegate was rejected: %v", err)
	}
	if err := check("other-performer"); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("unassigned performer candidate was not rejected: %v", err)
	}
}

func TestBlankRecordOwnerCanOnlyBeRepairedByTheCurrentOwnerRoute(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LEGACY", Name: "Legacy Program", Type: "COMPLIANCE",
		OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC(), ActorID: "system",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Legacy issue", Summary: "Ownership was not migrated.", ActorID: "system",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "route-owner", DisplayName: "Current owner route"},
		CandidatePrincipals: []authority.Principal{
			{ID: "new-owner", DisplayName: "New owner"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "route-owner", OriginPrincipalID: "route-owner"},
			{PrincipalID: "new-owner", OriginPrincipalID: "new-owner"},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}

	tests := []struct {
		name       string
		recordID   string
		assign     string
		blocked    string
		ownerField string
		objectType string
	}{
		{name: "Program", recordID: program.Program.ID, assign: "program.assign", blocked: "program.details.update", ownerField: "owner_principal_id", objectType: "PROGRAM"},
		{name: "Matter", recordID: matter.Matter.ID, assign: "matter.assign", blocked: "matter.details.update", ownerField: "owner_principal_id", objectType: "MATTER"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestFor := func(actorID string) *http.Request {
				request := httptest.NewRequest("POST", "/api/v1/records/"+test.recordID, nil)
				request.SetPathValue("id", test.recordID)
				return request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID}))
			}
			base := commandPolicy{ObjectType: test.objectType, Responsibility: authority.ResponsibilityOwner, Materiality: 2}
			request := requestFor("route-owner")
			if _, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", test.assign, map[string]any{test.ownerField: "new-owner"}, base); err != nil {
				t.Fatalf("current owner route could not repair blank ownership: %v", err)
			}
			request = requestFor("outside-route")
			if _, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", test.assign, map[string]any{test.ownerField: "new-owner"}, base); !errors.Is(err, commandauth.ErrNotAuthorized) {
				t.Fatalf("actor outside owner route was not rejected: %v", err)
			}
			request = requestFor("route-owner")
			if _, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", test.blocked, map[string]any{}, base); !errors.Is(err, commandauth.ErrGuardUnavailable) {
				t.Fatalf("owner-bound command ran before ownership was repaired: %v", err)
			}
		})
	}
}

func TestBlankOwnerOperationsExposeOnlyAssignmentRepair(t *testing.T) {
	now := time.Now().UTC()
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal:           authority.Principal{ID: "route-owner", DisplayName: "Current owner route"},
		CandidatePrincipals: []authority.Principal{{ID: "new-owner", DisplayName: "New owner"}},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "route-owner", OriginPrincipalID: "route-owner"},
			{PrincipalID: "new-owner", OriginPrincipalID: "new-owner"},
		},
	}}
	api := &API{deps: Dependencies{Authority: resolver}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "route-owner"}
	find := func(operations []RecordOperation, command, subresource string) RecordOperation {
		t.Helper()
		for _, operation := range operations {
			if operation.Command == command && operation.SubresourceID == subresource {
				return operation
			}
		}
		t.Fatalf("operation %s/%s was not returned", command, subresource)
		return RecordOperation{}
	}

	program := continuity.ProgramAggregate{Program: continuity.Program{
		ID: "program-legacy", TenantID: "bank", LegalEntityID: "entity-a", Code: "LEGACY", Name: "Legacy Program",
		Status: continuity.ProgramDraft, Version: 1, CreatedAt: now, UpdatedAt: now,
	}}
	programOperations := api.buildProgramOperations(t.Context(), actor, program, now).Operations
	if assignment := find(programOperations, "program.assign", ""); !assignment.CanAct || assignment.AssignedTo != nil {
		t.Fatalf("blank Program owner assignment repair = %#v", assignment)
	}
	if details := find(programOperations, "program.details.update", ""); details.CanAct {
		t.Fatalf("blank Program owner allowed details update: %#v", details)
	}

	matter := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-legacy", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap,
		Status: continuity.MatterAssessment, Priority: 3, Title: "Legacy issue", Version: 1, CreatedAt: now, UpdatedAt: now,
	}, Actions: []continuity.Action{{
		ID: "action-legacy", TenantID: "bank", MatterID: "matter-legacy", Title: "Legacy action",
		Description: "Ownership was not migrated.", Status: continuity.ActionPlanned, CreatedAt: now, UpdatedAt: now, Version: 1,
	}}}
	matterOperations := api.buildMatterOperations(t.Context(), actor, matter, now).Operations
	if assignment := find(matterOperations, "matter.assign", ""); !assignment.CanAct || assignment.AssignedTo != nil {
		t.Fatalf("blank Matter owner assignment repair = %#v", assignment)
	}
	if details := find(matterOperations, "matter.details.update", ""); details.CanAct {
		t.Fatalf("blank Matter owner allowed details update: %#v", details)
	}
	if transition := find(matterOperations, "matter.action.transition", "action-legacy"); transition.CanAct || transition.AssignedTo != nil {
		t.Fatalf("blank Action owner allowed a lifecycle change: %#v", transition)
	}
}

func TestCurrentAuthorizerCanRecoverUnassignedMatterWithoutBecomingOwner(t *testing.T) {
	now := time.Now().UTC()
	matter := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-unassigned", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap,
		Status: continuity.MatterAssessment, Priority: 3, Title: "Restore unavailable source", Version: 2, CreatedAt: now, UpdatedAt: now,
	}}
	service := continuity.NewService(continuity.NewMemoryRepository())
	stored, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: matter.Matter.TenantID, LegalEntityID: matter.Matter.LegalEntityID, Type: matter.Matter.Type, Priority: matter.Matter.Priority,
		Title: matter.Matter.Title, Summary: "A source is unavailable.", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["cro-1"]}`), ActorID: "system",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: unassignedRecoveryAuthority{}}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "cro-1"}
	operations := api.buildMatterOperations(identity.WithActor(t.Context(), actor), actor, stored, now).Operations
	var assignment RecordOperation
	for _, operation := range operations {
		if operation.Command == "matter.assign" {
			assignment = operation
			break
		}
	}
	if !assignment.CanAct || assignment.Responsibility != string(authority.ResponsibilityAuthorizer) || len(assignment.Candidates) != 1 || assignment.Candidates[0].ID != "owner-1" {
		t.Fatalf("unassigned recovery operation = %#v", assignment)
	}
	request := httptest.NewRequest("POST", "/api/v1/matters/"+stored.Matter.ID+"/assignment", nil)
	request.SetPathValue("id", stored.Matter.ID)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	payload := map[string]any{"owner_principal_id": "owner-1"}
	policy, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.assign", payload, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3})
	if err != nil {
		t.Fatalf("authorizer could not recover assignment: %v", err)
	}
	if !policy.SpecializedAuthorization || policy.Responsibility != authority.ResponsibilityAuthorizer || payload["reassignment_basis"] != "UNASSIGNED_RECOVERY" {
		t.Fatalf("unassigned recovery was not independently bound: policy=%#v payload=%#v", policy, payload)
	}
}

func TestMatterOwnerCanReassignLegacyActionWithBlankPerformer(t *testing.T) {
	repository := continuity.NewMemoryRepository()
	service := continuity.NewService(repository)
	now := time.Now().UTC()
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Legacy issue", Summary: "An action owner was not migrated.", OwnerPrincipalID: "owner-1", ActorID: "system",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := continuity.Action{
		ID: "action-legacy", TenantID: "bank", MatterID: matter.Matter.ID, Title: "Legacy action",
		Description: "Assign the accountable performer.", Status: continuity.ActionPlanned, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	payload, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ApplyMatterEvent(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, matter.Matter.Version, continuity.Event{
		ID: "event-legacy-action", TenantID: "bank", AggregateType: "MATTER", AggregateID: matter.Matter.ID,
		AggregateVersion: matter.Matter.Version + 1, Type: continuity.EventActionAdded, Payload: payload, ActorID: "migration", OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner-1", DisplayName: "Matter owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "performer-1", DisplayName: "Action performer"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"},
			{PrincipalID: "performer-1", OriginPrincipalID: "performer-1"},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	request := httptest.NewRequest("POST", "/api/v1/matters/"+matter.Matter.ID+"/actions/"+action.ID+"/assignment", nil)
	request.SetPathValue("id", matter.Matter.ID)
	request.SetPathValue("action_id", action.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}))
	base := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}
	if _, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.action.assign", map[string]any{"owner_principal_id": "performer-1"}, base); err != nil {
		t.Fatalf("Matter owner could not repair blank Action ownership: %v", err)
	}
	transition := httptest.NewRequest("POST", "/api/v1/matters/"+matter.Matter.ID+"/actions/"+action.ID+"/transition", nil)
	transition.SetPathValue("id", matter.Matter.ID)
	transition.SetPathValue("action_id", action.ID)
	transition = transition.WithContext(request.Context())
	if _, err := api.lifecycleCommandPolicy(transition.Context(), transition, "bank", "matter.action.transition", map[string]any{"to": "IN_PROGRESS"}, base); !errors.Is(err, commandauth.ErrGuardUnavailable) {
		t.Fatalf("blank Action owner allowed a lifecycle change: %v", err)
	}
}

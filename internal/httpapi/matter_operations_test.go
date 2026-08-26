package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMatterOperationsExplainOwnershipAcrossRoles(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	activeActionID := matter.Actions[0].ID
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Publish filing timetable", Description: "Publish the approved timetable.", OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	completedActionID := matter.Actions[1].ID
	for _, target := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		matter, err = service.TransitionAction(continuity.WithTrustedSystemScope(t.Context()), continuity.TransitionActionInput{
			TenantID: "bank", MatterID: matter.Matter.ID, ActionID: completedActionID,
			ExpectedVersion: matter.Matter.Version, To: target, ActorID: "program-owner", Rationale: "Complete the timetable work.",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddVerificationContract(continuity.WithTrustedSystemScope(t.Context()), continuity.AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ActionID: completedActionID, ExpectedOutcome: "The approved timetable is available to every filing owner.",
		FailureResponse: "BLOCK_CLOSE", AuthorityPrincipalID: "auditor", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	contractID := matter.VerificationContracts[0].ID

	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "cro", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner: {
				Principal:           authority.Principal{ID: "program-owner", DisplayName: "Program Owner", Kind: "PERSON"},
				CandidatePrincipals: []authority.Principal{{ID: "program-owner", DisplayName: "Program Owner", Kind: "PERSON"}, {ID: "privacy-owner", DisplayName: "Privacy Owner", Kind: "PERSON"}},
			},
			authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "program-owner", DisplayName: "Program Owner", Kind: "PERSON"}},
			authority.ResponsibilityReviewer:  {Principal: authority.Principal{ID: "auditor", DisplayName: "Internal Auditor", Kind: "PERSON"}},
		}},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matters/"+matter.Matter.ID+"/operations?tenant_id=bank", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("operations returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		AuthorityAvailable bool `json:"authority_available"`
		Operations         []struct {
			Command        string                `json:"command"`
			SubresourceID  string                `json:"subresource_id"`
			Responsibility string                `json:"responsibility"`
			CanAct         bool                  `json:"can_act"`
			AssignedTo     *authority.Principal  `json:"assigned_to"`
			Candidates     []authority.Principal `json:"candidates"`
			Reason         string                `json:"reason"`
			AllowedTargets []string              `json:"allowed_targets"`
		} `json:"operations"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.AuthorityAvailable {
		t.Fatal("authority was reported unavailable")
	}
	operations := map[string]struct {
		command string
		name    string
		canAct  bool
		reason  string
	}{}
	actionAddCandidates := []authority.Principal{}
	matterTargets := map[string][]string{}
	for _, operation := range payload.Operations {
		key := operation.Command + ":" + operation.SubresourceID
		name := ""
		if operation.AssignedTo != nil {
			name = operation.AssignedTo.DisplayName
		}
		operations[key] = struct {
			command string
			name    string
			canAct  bool
			reason  string
		}{operation.Command, name, operation.CanAct, operation.Reason}
		if operation.Command == "matter.action.add" {
			actionAddCandidates = operation.Candidates
		}
		if operation.Command == "matter.transition" {
			matterTargets[operation.Responsibility] = operation.AllowedTargets
		}
	}
	owner := operations["matter.details.update:"]
	action := operations["matter.action.transition:"+activeActionID]
	outcome := operations["matter.outcome.record:"+contractID]
	outcomeDefinition := operations["matter.outcome.define:"]
	if owner.name != "Program Owner" || owner.canAct {
		t.Fatalf("owner operation is unexplained: %#v", owner)
	}
	if action.name != "Program Owner" || action.canAct {
		t.Fatalf("Action performer is unexplained: %#v", action)
	}
	if outcome.name != "Internal Auditor" || outcome.canAct {
		t.Fatalf("outcome reviewer is unexplained: %#v", outcome)
	}
	if outcomeDefinition.name != "Internal Auditor" || outcomeDefinition.canAct {
		t.Fatalf("outcome definition reviewer is unexplained: %#v", outcomeDefinition)
	}
	if !reflect.DeepEqual(matterTargets, map[string][]string{
		string(authority.ResponsibilityOwner):      {"TRIAGE"},
		string(authority.ResponsibilityAuthorizer): {"CANCELLED"},
	}) {
		t.Fatalf("Matter lifecycle targets = %#v", matterTargets)
	}
	if len(actionAddCandidates) != 1 || actionAddCandidates[0].DisplayName != "Program Owner" {
		t.Fatalf("Action creation did not return eligible performers: %#v", actionAddCandidates)
	}
}

func TestMatterOperationsBindWildcardViewerToRecordEntity(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterRegulatoryChange,
		Status: continuity.MatterAssessment, Priority: 4, Title: "Annual return", OwnerPrincipalID: "owner-1",
		CreatedAt: now, UpdatedAt: now, Version: 2,
	}}
	resolver := &capturingProgramAuthority{assignmentAuthorityStub: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}},
		authority.ResponsibilityReviewer:   {Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Internal Auditor"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "authorizer-1", DisplayName: "CCO"}},
	}}}
	api := &API{deps: Dependencies{Authority: resolver}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "owner-1", LegalEntityID: "*", Kind: "PERSON"}

	payload := api.buildMatterOperations(t.Context(), actor, aggregate, now)
	if len(payload.Operations) == 0 {
		t.Fatal("Matter operations were not returned")
	}
	for _, entity := range resolver.legalEntities {
		if entity != "entity-a" {
			t.Fatalf("Matter authority resolved outside the record entity: %#v", resolver.legalEntities)
		}
	}
}

func TestMatterOperationsHideRestrictedMatterFromUnlistedActor(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterAuthorityRequest, Priority: 5,
		Title: "Restricted request", Summary: "Protected response work.",
		Scope:            json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["program-owner"]}`),
		OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "cro", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matters/"+matter.Matter.ID+"/operations?tenant_id=bank", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("restricted operations returned %d: %s", response.Code, response.Body.String())
	}
}

func TestMatterOperationsResolveStoredParticipantOutsideCurrentRoute(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 3,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "stored-owner", ActorID: "stored-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{
		Continuity: service,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "new-owner", DisplayName: "New owner"}},
		}},
		Access: principalResolverStub{values: map[string]access.Resolution{
			"stored-owner": {TenantID: "bank", PrincipalID: "stored-owner", LegalEntityID: "bank-ng", DisplayName: "Program Owner", Kind: "PERSON"},
		}},
	}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "cro", LegalEntityID: "bank-ng", Kind: "PERSON"}
	payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, matter, matter.Matter.UpdatedAt)
	for _, operation := range payload.Operations {
		if operation.Command == "matter.details.update" {
			if operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Program Owner" {
				t.Fatalf("stored owner label was not resolved exactly: %#v", operation)
			}
			return
		}
	}
	t.Fatal("detail operation was not returned")
}

func TestMatterOperationsKeepStoredOwnerVisibleWhenAuthorityIsUnavailable(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 3,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "stored-owner", ActorID: "stored-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{
		Continuity: service,
		Access: principalResolverStub{values: map[string]access.Resolution{
			"stored-owner": {TenantID: "bank", PrincipalID: "stored-owner", LegalEntityID: "bank-ng", DisplayName: "Program Owner", Kind: "PERSON"},
		}},
	}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "cro", LegalEntityID: "bank-ng", Kind: "PERSON"}
	payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, matter, matter.Matter.UpdatedAt)
	if payload.AuthorityAvailable {
		t.Fatal("authority was reported available")
	}
	for _, operation := range payload.Operations {
		if operation.Command == "matter.details.update" {
			if operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Program Owner" || operation.CanAct {
				t.Fatalf("stored owner disappeared during authority failure: %#v", operation)
			}
			return
		}
	}
	t.Fatal("detail operation was not returned")
}

func TestMatterOperationsKeepCancelledResponsibilitiesReadableWithoutCommandsOrPrincipalIDs(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{
		Matter: continuity.Matter{
			ID: "matter-closed", TenantID: "bank", LegalEntityID: "bank-ng",
			Type: continuity.MatterRegulatoryChange, Status: continuity.MatterCancelled,
			Title: "Completed annual return", OwnerPrincipalID: "stored-owner", Version: 12,
			CreatedAt: now, UpdatedAt: now,
		},
		Actions: []continuity.Action{
			{ID: "action-implemented", MatterID: "matter-closed", Title: "File the return", OwnerPrincipalID: "stored-performer", Status: continuity.ActionImplemented},
			{ID: "action-cancelled", MatterID: "matter-closed", Title: "Prepare a duplicate schedule", OwnerPrincipalID: "stored-cancelled-performer", Status: continuity.ActionCancelled},
		},
	}
	api := &API{deps: Dependencies{Access: principalResolverStub{values: map[string]access.Resolution{
		"stored-owner":               {TenantID: "bank", PrincipalID: "stored-owner", LegalEntityID: "bank-ng", DisplayName: "Privacy Program Owner", Kind: "PERSON"},
		"stored-performer":           {TenantID: "bank", PrincipalID: "stored-performer", LegalEntityID: "bank-ng", DisplayName: "Annual Return Lead", Kind: "PERSON"},
		"stored-cancelled-performer": {TenantID: "bank", PrincipalID: "stored-cancelled-performer", LegalEntityID: "bank-ng", DisplayName: "Compliance Operations Analyst", Kind: "PERSON"},
	}}}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "auditor", LegalEntityID: "bank-ng", Kind: "PERSON"}

	payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Operations         []RecordOperation `json:"operations"`
		ResponsibleParties []struct {
			Scope          string `json:"scope"`
			SubresourceID  string `json:"subresource_id"`
			Responsibility string `json:"responsibility"`
			DisplayName    string `json:"display_name"`
		} `json:"responsible_parties"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Operations) != 0 {
		t.Fatalf("terminal issue exposed commands: %#v", response.Operations)
	}
	got := map[string]string{}
	for _, party := range response.ResponsibleParties {
		got[party.Scope+":"+party.SubresourceID+":"+party.Responsibility] = party.DisplayName
	}
	want := map[string]string{
		"RECORD::ACCOUNTABLE_OWNER":           "Privacy Program Owner",
		"ACTION:action-implemented:PERFORMER": "Annual Return Lead",
		"ACTION:action-cancelled:PERFORMER":   "Compliance Operations Analyst",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("terminal responsibilities = %#v, want %#v", got, want)
	}
	for _, principalID := range []string{"stored-owner", "stored-performer", "stored-cancelled-performer"} {
		if strings.Contains(string(encoded), principalID) {
			t.Fatalf("terminal responsibility response exposed principal ID %q: %s", principalID, encoded)
		}
	}
}

func TestMatterOperationsExposeOnlyGovernedReopenForClosedMatter(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-closed", TenantID: "bank", LegalEntityID: "bank-ng",
		Type: continuity.MatterRegulatoryChange, Status: continuity.MatterClosed,
		Title: "Completed annual return", OwnerPrincipalID: "stored-owner", Priority: 4, Version: 12,
		CreatedAt: now, UpdatedAt: now,
	}}
	api := &API{deps: Dependencies{Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "authorizer-1", DisplayName: "Chief Compliance Officer", Kind: "PERSON"}},
	}}}}

	for _, test := range []struct {
		name      string
		actorID   string
		canReopen bool
	}{
		{name: "current authorizer", actorID: "authorizer-1", canReopen: true},
		{name: "different actor", actorID: "auditor-1", canReopen: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			actor := identity.Actor{TenantID: "bank", PrincipalID: test.actorID, LegalEntityID: "bank-ng", Kind: "PERSON"}
			payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)
			if len(payload.Operations) != 1 {
				t.Fatalf("closed issue operations = %#v, want only the governed reopen command", payload.Operations)
			}
			operation := payload.Operations[0]
			if operation.Command != "matter.transition" || operation.Responsibility != string(authority.ResponsibilityAuthorizer) {
				t.Fatalf("closed issue operation = %#v", operation)
			}
			if !reflect.DeepEqual(operation.AllowedTargets, []string{string(continuity.MatterAssessment)}) {
				t.Fatalf("closed issue targets = %#v, want ASSESSMENT only", operation.AllowedTargets)
			}
			if operation.CanAct != test.canReopen {
				t.Fatalf("closed issue can_act = %v, want %v for %s", operation.CanAct, test.canReopen, test.actorID)
			}
			if operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Chief Compliance Officer" {
				t.Fatalf("closed issue responsibility label = %#v", operation.AssignedTo)
			}
		})
	}
}

func TestMatterOperationsExposeDecisionAndResponseLifecycleByResponsibility(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{
		Matter: continuity.Matter{
			ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterAuthorityRequest,
			Status: continuity.MatterDecisionRequired, Priority: 4, OwnerPrincipalID: "owner-1",
			CreatedAt: now, UpdatedAt: now, Version: 5,
		},
		Decisions:        []continuity.Decision{{ID: "decision-1", Type: "POSITION", Status: continuity.DecisionProposed, CreatedAt: now, UpdatedAt: now}},
		ResponsePackages: []continuity.ResponsePackage{{ID: "response-1", Purpose: "Respond to the regulator", Audience: "NDPC", Status: continuity.ResponseDraft, CreatedAt: now, UpdatedAt: now}},
	}
	api := &API{deps: Dependencies{Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}},
		authority.ResponsibilityProposer:   {Principal: authority.Principal{ID: "proposer-1", DisplayName: "Response Lead"}},
		authority.ResponsibilityReviewer:   {Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Compliance Reviewer"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "authorizer-1", DisplayName: "CCO"}},
		authority.ResponsibilityChallenger: {Principal: authority.Principal{ID: "challenger-1", DisplayName: "Independent Risk"}},
	}}}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "reviewer-1", LegalEntityID: "bank-ng", Kind: "PERSON"}
	payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)

	var decision, response *RecordOperation
	for index := range payload.Operations {
		operation := &payload.Operations[index]
		if operation.Command == "matter.decision.record" && operation.SubresourceID == "decision-1" && operation.CanAct {
			decision = operation
		}
		if operation.Command == "matter.response.transition" && operation.SubresourceID == "response-1" && operation.CanAct {
			response = operation
		}
	}
	if decision == nil || !reflect.DeepEqual(decision.AllowedTargets, []string{"IN_REVIEW", "RETURNED"}) {
		t.Fatalf("reviewer decision operation = %#v", decision)
	}
	if response == nil || !reflect.DeepEqual(response.AllowedTargets, []string{"IN_REVIEW"}) {
		t.Fatalf("reviewer response operation = %#v", response)
	}
}

func TestMatterOperationsRouteGovernedAndOrdinaryTransitionsToDistinctActors(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange,
		Status: continuity.MatterAssessment, Priority: 4, OwnerPrincipalID: "owner-1", CreatedAt: now, UpdatedAt: now, Version: 5,
	}}
	api := &API{deps: Dependencies{Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "authorizer-1", DisplayName: "CCO"}},
	}}}}

	assertTransitions := func(t *testing.T, principalID string, wantResponsibility authority.Responsibility, wantTargets []string) {
		t.Helper()
		actor := identity.Actor{TenantID: "bank", PrincipalID: principalID, LegalEntityID: "bank-ng", Kind: "PERSON"}
		payload := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)
		for _, operation := range payload.Operations {
			if operation.Command == "matter.transition" && operation.CanAct {
				if operation.Responsibility != string(wantResponsibility) || !reflect.DeepEqual(operation.AllowedTargets, wantTargets) {
					t.Fatalf("%s transition operation = %#v", principalID, operation)
				}
				return
			}
		}
		t.Fatalf("%s did not receive an actionable lifecycle operation: %#v", principalID, payload.Operations)
	}

	assertTransitions(t, "owner-1", authority.ResponsibilityOwner, []string{"ACTION_IN_PROGRESS", "RESPONSE_PREPARATION", "VERIFICATION"})
	assertTransitions(t, "authorizer-1", authority.ResponsibilityAuthorizer, []string{"DECISION_REQUIRED", "CANCELLED"})
}

type principalResolverStub struct {
	values map[string]access.Resolution
}

func (s principalResolverStub) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	return access.Resolution{}, access.ErrIdentityNotProvisioned
}

func (s principalResolverStub) ResolvePrincipal(_ context.Context, _, principalID, _ string) (access.Resolution, error) {
	value, ok := s.values[principalID]
	if !ok {
		return access.Resolution{}, access.ErrPrincipalUnavailable
	}
	return value, nil
}

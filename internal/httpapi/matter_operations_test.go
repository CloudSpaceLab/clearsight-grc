package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMatterOperationsExplainOwnershipAcrossRoles(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(t.Context(), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	activeActionID := matter.Actions[0].ID
	matter, err = service.AddAction(t.Context(), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Publish filing timetable", Description: "Publish the approved timetable.", OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	completedActionID := matter.Actions[1].ID
	for _, target := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		matter, err = service.TransitionAction(t.Context(), continuity.TransitionActionInput{
			TenantID: "bank", MatterID: matter.Matter.ID, ActionID: completedActionID,
			ExpectedVersion: matter.Matter.Version, To: target, ActorID: "program-owner", Rationale: "Complete the timetable work.",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddVerificationContract(t.Context(), continuity.AddVerificationContractInput{
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
	matterTargets := []string{}
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
			matterTargets = operation.AllowedTargets
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
	if !reflect.DeepEqual(matterTargets, []string{"TRIAGE", "CANCELLED"}) {
		t.Fatalf("Matter lifecycle targets = %#v", matterTargets)
	}
	if len(actionAddCandidates) != 1 || actionAddCandidates[0].DisplayName != "Program Owner" {
		t.Fatalf("Action creation did not return eligible performers: %#v", actionAddCandidates)
	}
}

func TestMatterOperationsHideRestrictedMatterFromUnlistedActor(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterAuthorityRequest, Priority: 5,
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
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 3,
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
	payload := api.buildMatterOperations(t.Context(), actor, matter, matter.Matter.UpdatedAt)
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
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 3,
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
	payload := api.buildMatterOperations(t.Context(), actor, matter, matter.Matter.UpdatedAt)
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

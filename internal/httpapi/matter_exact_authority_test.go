package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

type exactMatterAuthority struct {
	batchCalls  int
	scalarCalls int
	inputs      []authority.ResolveInput
	routes      map[string]authority.Resolution
}

func exactMatterRoute(input authority.ResolveInput) string {
	return input.ObjectType + ":" + input.ObjectID + ":" + string(input.Responsibility)
}

func (s *exactMatterAuthority) resolution(input authority.ResolveInput) (authority.Resolution, error) {
	value, ok := s.routes[exactMatterRoute(input)]
	if !ok {
		return authority.Resolution{}, authority.ErrNoRoute
	}
	return value, nil
}

func (s *exactMatterAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.scalarCalls++
	s.inputs = append(s.inputs, input)
	return s.resolution(input)
}

func (s *exactMatterAuthority) ResolveMany(_ context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	s.batchCalls++
	s.inputs = append(s.inputs, inputs...)
	result := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		result[index].Resolution, result[index].Err = s.resolution(input)
	}
	return result, nil
}

func (s *exactMatterAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s *exactMatterAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s *exactMatterAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func directResolution(principalID string) authority.Resolution {
	return authority.Resolution{
		Principal:        authority.Principal{ID: principalID, DisplayName: principalID},
		EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}},
	}
}

func delegatedResolution(originID, delegateID string) authority.Resolution {
	return authority.Resolution{
		Principal:           authority.Principal{ID: originID, DisplayName: originID},
		CandidatePrincipals: []authority.Principal{{ID: delegateID, DisplayName: delegateID}},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: originID, OriginPrincipalID: originID},
			{PrincipalID: delegateID, OriginPrincipalID: originID},
		},
	}
}

func TestMatterOperationsUseExactSubresourceAuthorityInOneBatch(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{
		Matter: continuity.Matter{
			ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterAuthorityRequest,
			Status: continuity.MatterActionsInProgress, Priority: 4, OwnerPrincipalID: "matter-owner", Version: 8,
		},
		Actions: []continuity.Action{
			{ID: "action-a", MatterID: "matter-1", OwnerPrincipalID: "performer-a", RequiredResponsibility: string(authority.ResponsibilityPerformer), Status: continuity.ActionPlanned},
			{ID: "action-b", MatterID: "matter-1", OwnerPrincipalID: "performer-b", RequiredResponsibility: string(authority.ResponsibilityPerformer), Status: continuity.ActionPlanned},
		},
		VerificationContracts: []continuity.VerificationContract{
			{ID: "contract-a", MatterID: "matter-1", AuthorityPrincipalID: "reviewer-a", ExpectedOutcome: "A remains effective.", Status: continuity.VerificationActive, CreatedAt: now.Add(-time.Hour)},
			{ID: "contract-b", MatterID: "matter-1", AuthorityPrincipalID: "reviewer-b", ExpectedOutcome: "B remains effective.", Status: continuity.VerificationActive, CreatedAt: now.Add(-time.Hour)},
		},
		ResponsePackages: []continuity.ResponsePackage{{ID: "response-a", MatterID: "matter-1", Purpose: "Reply to regulator", Status: continuity.ResponseDraft}},
		Decisions:        []continuity.Decision{{ID: "decision-a", MatterID: "matter-1", Type: "POSITION", Status: continuity.DecisionProposed}},
	}
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{}}
	for _, route := range []struct {
		objectType, objectID string
		responsibility       authority.Responsibility
	}{
		{"MATTER", "matter-1", authority.ResponsibilityOwner},
		{"MATTER", "matter-1", authority.ResponsibilityProposer},
		{"MATTER", "matter-1", authority.ResponsibilityReviewer},
		{"MATTER", "matter-1", authority.ResponsibilityAuthorizer},
		{"ACTION", "action-a", authority.ResponsibilityOwner},
		{"ACTION", "action-a", authority.ResponsibilityPerformer},
		{"ACTION", "action-b", authority.ResponsibilityOwner},
		{"ACTION", "action-b", authority.ResponsibilityPerformer},
		{"VERIFICATION_CONTRACT", "contract-a", authority.ResponsibilityReviewer},
		{"VERIFICATION_CONTRACT", "contract-b", authority.ResponsibilityReviewer},
		{"RESPONSE_PACKAGE", "response-a", authority.ResponsibilityReviewer},
		{"RESPONSE_PACKAGE", "response-a", authority.ResponsibilityProposer},
		{"DECISION", "decision-a", authority.ResponsibilityReviewer},
		{"DECISION", "decision-a", authority.ResponsibilityChallenger},
		{"DECISION", "decision-a", authority.ResponsibilityAuthorizer},
	} {
		resolver.routes[route.objectType+":"+route.objectID+":"+string(route.responsibility)] = directResolution("viewer")
	}
	// The viewer is a delegate only for the first exact Action and outcome
	// check. Broad Matter routes also include the viewer, proving those routes
	// cannot make a sibling subresource actionable.
	resolver.routes["MATTER:matter-1:ACCOUNTABLE_OWNER"] = delegatedResolution("matter-owner", "viewer")
	resolver.routes["MATTER:matter-1:REVIEWER"] = directResolution("viewer")
	resolver.routes["ACTION:action-a:ACCOUNTABLE_OWNER"] = delegatedResolution("matter-owner", "viewer")
	resolver.routes["ACTION:action-a:PERFORMER"] = delegatedResolution("performer-a", "viewer")
	resolver.routes["ACTION:action-b:ACCOUNTABLE_OWNER"] = directResolution("matter-owner")
	resolver.routes["ACTION:action-b:PERFORMER"] = directResolution("performer-b")
	resolver.routes["VERIFICATION_CONTRACT:contract-a:REVIEWER"] = delegatedResolution("reviewer-a", "viewer")
	resolver.routes["VERIFICATION_CONTRACT:contract-b:REVIEWER"] = directResolution("reviewer-b")

	api := &API{deps: Dependencies{Authority: resolver}}
	response := api.buildMatterOperations(t.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "viewer",
	}, aggregate, now)
	if resolver.batchCalls != 1 || resolver.scalarCalls != 0 {
		t.Fatalf("authority calls = batch %d scalar %d, want one batch and zero scalar", resolver.batchCalls, resolver.scalarCalls)
	}

	for _, operation := range response.Operations {
		found := false
		for _, input := range resolver.inputs {
			if input.DecisionType == operation.Command && input.Responsibility == authority.Responsibility(operation.Responsibility) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("operation has no matching primary authority input: %#v", operation)
		}
	}
	for _, expected := range []struct {
		command, objectType, objectID string
		responsibility                authority.Responsibility
	}{
		{"matter.action.update", "ACTION", "action-a", authority.ResponsibilityOwner},
		{"matter.action.assign", "ACTION", "action-a", authority.ResponsibilityOwner},
		{"matter.action.assign", "ACTION", "action-a", authority.ResponsibilityPerformer},
		{"matter.action.transition", "ACTION", "action-a", authority.ResponsibilityPerformer},
		{"matter.outcome.supersede", "VERIFICATION_CONTRACT", "contract-a", authority.ResponsibilityReviewer},
		{"matter.outcome.retire", "VERIFICATION_CONTRACT", "contract-a", authority.ResponsibilityReviewer},
		{"matter.outcome.record", "VERIFICATION_CONTRACT", "contract-a", authority.ResponsibilityReviewer},
		{"matter.response.transition", "RESPONSE_PACKAGE", "response-a", authority.ResponsibilityReviewer},
		{"matter.decision.record", "DECISION", "decision-a", authority.ResponsibilityReviewer},
	} {
		found := false
		for _, input := range resolver.inputs {
			if input.DecisionType == expected.command && input.ObjectType == expected.objectType && input.ObjectID == expected.objectID && input.Responsibility == expected.responsibility {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing exact authority input for %s %s/%s %s: %#v", expected.command, expected.objectType, expected.objectID, expected.responsibility, resolver.inputs)
		}
	}
	assertCanAct := func(command, subresourceID string, want bool) {
		t.Helper()
		for _, operation := range response.Operations {
			if operation.Command == command && operation.SubresourceID == subresourceID {
				if operation.CanAct != want {
					t.Errorf("%s/%s can_act = %v, want %v: %#v", command, subresourceID, operation.CanAct, want, operation)
				}
				return
			}
		}
		t.Errorf("missing operation %s/%s", command, subresourceID)
	}
	for _, command := range []string{"matter.action.update", "matter.action.assign", "matter.action.transition"} {
		assertCanAct(command, "action-a", true)
		assertCanAct(command, "action-b", false)
	}
	for _, command := range []string{"matter.outcome.supersede", "matter.outcome.retire", "matter.outcome.record"} {
		assertCanAct(command, "contract-a", true)
		assertCanAct(command, "contract-b", false)
	}
	assertCanAct("matter.response.transition", "response-a", true)
	assertCanAct("matter.decision.record", "decision-a", true)

	broadOnly := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"MATTER:matter-1:ACCOUNTABLE_OWNER": delegatedResolution("matter-owner", "viewer"),
		"MATTER:matter-1:PERFORMER":         directResolution("viewer"),
		"MATTER:matter-1:REVIEWER":          directResolution("viewer"),
		"MATTER:matter-1:PROPOSER":          directResolution("viewer"),
		"MATTER:matter-1:AUTHORIZER":        directResolution("viewer"),
	}}
	broadResponse := (&API{deps: Dependencies{Authority: broadOnly}}).buildMatterOperations(t.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "viewer",
	}, aggregate, now)
	for _, operation := range broadResponse.Operations {
		if operation.SubresourceID != "" && operation.CanAct {
			t.Errorf("broad Matter route enabled exact control %s/%s", operation.Command, operation.SubresourceID)
		}
	}
}

func TestMatterActionAuthorityDoesNotLeakAcrossSiblingActionsOrBroadMatterRoute(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(t.Context())
	service := continuity.NewService(continuity.NewMemoryRepository())
	aggregate, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 4,
		Title: "Two actions", Summary: "Keep each assignment route exact.", OwnerPrincipalID: "matter-owner", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ownerID := range []string{"performer-a", "performer-b"} {
		aggregate, err = service.AddAction(ctx, continuity.AddActionInput{
			TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
			Title: "Action " + ownerID, Description: "Exact work item.", OwnerPrincipalID: ownerID, ActorID: "matter-owner",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	actionA, actionB := aggregate.Actions[0], aggregate.Actions[1]
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"MATTER:" + aggregate.Matter.ID + ":ACCOUNTABLE_OWNER": delegatedResolution("matter-owner", "delegate-a"),
		"ACTION:" + actionA.ID + ":ACCOUNTABLE_OWNER":          delegatedResolution("matter-owner", "delegate-a"),
		"ACTION:" + actionB.ID + ":ACCOUNTABLE_OWNER":          directResolution("matter-owner"),
		"ACTION:" + actionA.ID + ":PERFORMER":                  directResolution("performer-a"),
		"ACTION:" + actionB.ID + ":PERFORMER":                  directResolution("performer-b"),
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}

	requestFor := func(actionID, actorID string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+aggregate.Matter.ID+"/actions/"+actionID, nil)
		request.SetPathValue("id", aggregate.Matter.ID)
		request.SetPathValue("action_id", actionID)
		return request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
		}))
	}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}

	bound, err := api.lifecycleCommandPolicy(requestFor(actionA.ID, "delegate-a").Context(), requestFor(actionA.ID, "delegate-a"), "bank", "matter.action.update", map[string]any{}, policy)
	if err != nil {
		t.Fatalf("delegate for exact action was rejected: %v", err)
	}
	if bound.ObjectType != "ACTION" || bound.ObjectIDPath != "action_id" || commandObjectID(requestFor(actionA.ID, "delegate-a"), map[string]any{}, bound) != actionA.ID {
		t.Fatalf("action command remained bound to the Matter: %#v", bound)
	}
	_, err = api.lifecycleCommandPolicy(requestFor(actionB.ID, "delegate-a").Context(), requestFor(actionB.ID, "delegate-a"), "bank", "matter.action.update", map[string]any{}, policy)
	if !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("delegate for sibling action was not rejected: %v", err)
	}
}

func TestMatterOutcomeAuthorityDoesNotLeakAcrossSiblingContractsOrBroadMatterRoute(t *testing.T) {
	service, aggregate := matterWithExactAuthorityResources(t)
	ctx := continuity.WithTrustedSystemScope(t.Context())
	var err error
	aggregate, err = service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		ExpectedOutcome: "The second response is complete.", FailureResponse: "BLOCK_CLOSE", AuthorityPrincipalID: "reviewer-b", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	contractA, contractB := aggregate.VerificationContracts[0], aggregate.VerificationContracts[1]
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"MATTER:" + aggregate.Matter.ID + ":REVIEWER":         delegatedResolution("reviewer-a", "delegate-a"),
		"VERIFICATION_CONTRACT:" + contractA.ID + ":REVIEWER": delegatedResolution("reviewer-a", "delegate-a"),
		"VERIFICATION_CONTRACT:" + contractB.ID + ":REVIEWER": directResolution("reviewer-b"),
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}

	request := exactMatterCommandRequest(aggregate.Matter.ID, "contract_id", contractA.ID, "delegate-a")
	bound, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.outcome.retire", map[string]any{}, policy)
	if err != nil {
		t.Fatalf("delegate for exact outcome check was rejected: %v", err)
	}
	if bound.ObjectType != "VERIFICATION_CONTRACT" || commandObjectID(request, map[string]any{"contract_id": contractA.ID}, bound) != contractA.ID {
		t.Fatalf("outcome command remained bound to the Matter: %#v", bound)
	}
	request = exactMatterCommandRequest(aggregate.Matter.ID, "contract_id", contractB.ID, "delegate-a")
	if _, err = api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.outcome.retire", map[string]any{}, policy); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("delegate for sibling outcome check was not rejected: %v", err)
	}
}

func TestCommandObjectIDReadsLifecycleObjectFromPayloadAndRejectsRouteMismatch(t *testing.T) {
	policy := commandPolicy{ObjectType: "VERIFICATION_CONTRACT", ObjectIDPath: "contract_id"}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/matter-1/verification-results", nil)
	request.SetPathValue("id", "matter-1")
	if got := commandObjectID(request, map[string]any{"contract_id": "contract-a"}, policy); got != "contract-a" {
		t.Fatalf("payload-bound command object = %q, want contract-a", got)
	}
	request.SetPathValue("contract_id", "contract-a")
	if _, err := lifecycleSubresourceID(request, map[string]any{"contract_id": "contract-b"}, "contract_id"); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("path/body mismatch was not rejected: %v", err)
	}
}

func TestInitialDecisionUsesMatterAuthorityThenFollowUpUsesCurrentDecision(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	ctx := continuity.WithTrustedSystemScope(t.Context())
	aggregate, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterException, Priority: 4,
		Title: "Exception decision", Summary: "Route the proposal then its review.", OwnerPrincipalID: "owner", ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"MATTER:" + aggregate.Matter.ID + ":PROPOSER": directResolution("proposer"),
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	request := matterLifecycleRequest(aggregate.Matter.ID, "proposer")
	initial, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.decision.record", map[string]any{"type": "EXCEPTION", "status": "PROPOSED"}, commandPolicy{ObjectType: "MATTER"})
	if err != nil {
		t.Fatal(err)
	}
	if initial.ObjectType != "MATTER" || initial.ObjectIDPath != "" {
		t.Fatalf("initial proposal authority = %#v", initial)
	}
	aggregate, err = service.AddDecision(ctx, continuity.AddDecisionInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		Type: "EXCEPTION", Status: continuity.DecisionProposed, Rationale: "Propose a bounded exception.", AuthorityPrincipalID: "proposer", Options: json.RawMessage(`[]`), Conditions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	currentID := aggregate.Decisions[len(aggregate.Decisions)-1].ID
	resolver.routes["DECISION:"+currentID+":REVIEWER"] = directResolution("reviewer")
	request = matterLifecycleRequest(aggregate.Matter.ID, "reviewer")
	payload := map[string]any{"type": "EXCEPTION", "status": "IN_REVIEW"}
	followUp, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.decision.record", payload, commandPolicy{ObjectType: "MATTER"})
	if err != nil {
		t.Fatal(err)
	}
	if followUp.ObjectType != "DECISION" || followUp.ObjectIDPath != "decision_id" || commandObjectID(request, payload, followUp) != currentID {
		t.Fatalf("follow-up decision authority = %#v", followUp)
	}
}

func TestMatterSubresourceCommandPoliciesUseExactAuthorityTuples(t *testing.T) {
	service, aggregate := matterWithExactAuthorityResources(t)
	action := aggregate.Actions[0]
	contract := aggregate.VerificationContracts[0]
	responsePackage := aggregate.ResponsePackages[0]
	decision := aggregate.Decisions[0]
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"ACTION:" + action.ID + ":ACCOUNTABLE_OWNER":           directResolution("matter-owner"),
		"ACTION:" + action.ID + ":PERFORMER":                   {Principal: authority.Principal{ID: "performer-a"}, CandidatePrincipals: []authority.Principal{{ID: "replacement-performer"}}},
		"VERIFICATION_CONTRACT:" + contract.ID + ":REVIEWER":   {Principal: authority.Principal{ID: "reviewer-a"}, CandidatePrincipals: []authority.Principal{{ID: "replacement-reviewer"}}},
		"RESPONSE_PACKAGE:" + responsePackage.ID + ":REVIEWER": directResolution("response-reviewer"),
		"DECISION:" + decision.ID + ":REVIEWER":                directResolution("decision-reviewer"),
		"MATTER:" + aggregate.Matter.ID + ":REVIEWER":          {Principal: authority.Principal{ID: "reviewer-a"}, CandidatePrincipals: []authority.Principal{{ID: "replacement-reviewer"}}},
		"MATTER:" + aggregate.Matter.ID + ":ACCOUNTABLE_OWNER": directResolution("matter-owner"),
		"MATTER:" + aggregate.Matter.ID + ":PERFORMER":         {Principal: authority.Principal{ID: "performer-a"}, CandidatePrincipals: []authority.Principal{{ID: "replacement-performer"}}},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	tests := []struct {
		name, actorID, command, routeField, routeID, objectType, objectID string
		responsibility                                                    authority.Responsibility
		payload                                                           map[string]any
	}{
		{"action update", "matter-owner", "matter.action.update", "action_id", action.ID, "ACTION", action.ID, authority.ResponsibilityOwner, map[string]any{}},
		{"action assign", "matter-owner", "matter.action.assign", "action_id", action.ID, "ACTION", action.ID, authority.ResponsibilityOwner, map[string]any{"owner_principal_id": "replacement-performer"}},
		{"action transition", "performer-a", "matter.action.transition", "action_id", action.ID, "ACTION", action.ID, authority.ResponsibilityPerformer, map[string]any{"to": "IN_PROGRESS"}},
		{"outcome supersede", "reviewer-a", "matter.outcome.supersede", "contract_id", contract.ID, "VERIFICATION_CONTRACT", contract.ID, authority.ResponsibilityReviewer, map[string]any{"reviewer_candidate_id": "replacement-reviewer"}},
		{"outcome retire", "reviewer-a", "matter.outcome.retire", "contract_id", contract.ID, "VERIFICATION_CONTRACT", contract.ID, authority.ResponsibilityReviewer, map[string]any{}},
		{"outcome record", "reviewer-a", "matter.outcome.record", "", "", "VERIFICATION_CONTRACT", contract.ID, authority.ResponsibilityReviewer, map[string]any{"contract_id": contract.ID, "result": "PASSED"}},
		{"response transition", "response-reviewer", "matter.response.transition", "response_id", responsePackage.ID, "RESPONSE_PACKAGE", responsePackage.ID, authority.ResponsibilityReviewer, map[string]any{"to": "IN_REVIEW"}},
		{"decision follow-up", "decision-reviewer", "matter.decision.record", "", "", "DECISION", decision.ID, authority.ResponsibilityReviewer, map[string]any{"type": decision.Type, "status": "IN_REVIEW"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver.inputs = nil
			request := exactMatterCommandRequest(aggregate.Matter.ID, test.routeField, test.routeID, test.actorID)
			policy, err := api.lifecycleCommandPolicy(request.Context(), request, "bank", test.command, test.payload, commandPolicy{
				ObjectType: "MATTER", Responsibility: test.responsibility, Materiality: 2,
			})
			if err != nil {
				t.Fatal(err)
			}
			if policy.ObjectType != test.objectType || policy.ObjectIDPath == "" || commandObjectID(request, test.payload, policy) != test.objectID || policy.Responsibility != test.responsibility {
				t.Fatalf("command policy = %#v, payload %#v", policy, test.payload)
			}
			for _, input := range resolver.inputs {
				if input.ObjectType != test.objectType || input.ObjectID != test.objectID {
					t.Fatalf("pre-authorization used a broader or sibling route: %#v", input)
				}
			}
		})
	}
}

func TestBroadMatterRouteCannotAuthorizeExactResponseOrDecision(t *testing.T) {
	service, aggregate := matterWithExactAuthorityResources(t)
	responsePackage := aggregate.ResponsePackages[0]
	decision := aggregate.Decisions[0]
	resolver := &exactMatterAuthority{routes: map[string]authority.Resolution{
		"MATTER:" + aggregate.Matter.ID + ":REVIEWER":          directResolution("broad-reviewer"),
		"RESPONSE_PACKAGE:" + responsePackage.ID + ":REVIEWER": directResolution("response-reviewer"),
		"DECISION:" + decision.ID + ":REVIEWER":                directResolution("decision-reviewer"),
	}}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	for _, test := range []struct {
		name, command, routeField, routeID, body string
		policy                                   commandPolicy
	}{
		{
			name: "response", command: "matter.response.transition", routeField: "response_id", routeID: responsePackage.ID,
			body: `{"to":"IN_REVIEW"}`, policy: commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilitySignatory, Materiality: 4},
		},
		{
			name: "decision", command: "matter.decision.record",
			body: `{"type":"` + decision.Type + `","status":"IN_REVIEW","rationale":"Review current proposal."}`, policy: commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := api.command(test.command, test.policy, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+aggregate.Matter.ID, strings.NewReader(test.body))
			request.SetPathValue("id", aggregate.Matter.ID)
			if test.routeField != "" {
				request.SetPathValue(test.routeField, test.routeID)
			}
			request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
				TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "broad-reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
			}))
			result := httptest.NewRecorder()
			handler(result, request)
			if result.Code != http.StatusForbidden || called {
				t.Fatalf("broad Matter route authorized exact %s: %d %s", test.name, result.Code, result.Body.String())
			}
		})
	}
}

func matterWithExactAuthorityResources(t *testing.T) (*continuity.Service, continuity.MatterAggregate) {
	t.Helper()
	ctx := continuity.WithTrustedSystemScope(t.Context())
	service := continuity.NewService(continuity.NewMemoryRepository())
	aggregate, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterAuthorityRequest, Priority: 4,
		Title: "Exact authority resources", Summary: "Keep every governed resource on its own route.", OwnerPrincipalID: "matter-owner", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.AddAction(ctx, continuity.AddActionInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		Title: "Prepare evidence", Description: "Prepare the evidence package.", OwnerPrincipalID: "performer-a", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		ExpectedOutcome: "The response is complete.", FailureResponse: "BLOCK_CLOSE", AuthorityPrincipalID: "reviewer-a", ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.AddResponsePackage(ctx, continuity.AddResponsePackageInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		Purpose: "Respond to regulator", Audience: "Regulator", Manifest: json.RawMessage(`[]`), ActorID: "matter-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.AddDecision(ctx, continuity.AddDecisionInput{
		TenantID: "bank", MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
		Type: "POSITION", Status: continuity.DecisionProposed, Rationale: "Propose the response position.", AuthorityPrincipalID: "proposer", Options: json.RawMessage(`[]`), Conditions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, aggregate
}

func exactMatterCommandRequest(matterID, routeField, routeID, actorID string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matterID, nil)
	request.SetPathValue("id", matterID)
	if routeField != "" {
		request.SetPathValue(routeField, routeID)
	}
	return request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
}

func matterLifecycleRequest(matterID, actorID string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matterID+"/decisions", nil)
	request.SetPathValue("id", matterID)
	return request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
}

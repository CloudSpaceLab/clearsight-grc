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
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestDecisionLifecycleResponsibilityMatrix(t *testing.T) {
	tests := []struct {
		status continuity.DecisionStatus
		want   authority.Responsibility
		mat    int
	}{
		{continuity.DecisionProposed, authority.ResponsibilityProposer, 2},
		{continuity.DecisionInReview, authority.ResponsibilityReviewer, 3},
		{continuity.DecisionReturned, authority.ResponsibilityReviewer, 3},
		{continuity.DecisionChallenged, authority.ResponsibilityChallenger, 3},
		{continuity.DecisionApproved, authority.ResponsibilityAuthorizer, 4},
		{continuity.DecisionConditionallyApproved, authority.ResponsibilityAuthorizer, 4},
		{continuity.DecisionRejected, authority.ResponsibilityAuthorizer, 4},
		{continuity.DecisionExpired, authority.ResponsibilityAuthorizer, 4},
		{continuity.DecisionSuperseded, authority.ResponsibilityAuthorizer, 4},
	}
	for _, test := range tests {
		got, err := continuity.DecisionLifecyclePolicy(test.status)
		if err != nil || authority.Responsibility(got.Responsibility) != test.want || got.Materiality != test.mat {
			t.Fatalf("%s: got responsibility=%s materiality=%d err=%v", test.status, got.Responsibility, got.Materiality, err)
		}
	}
}

func TestResponseLifecycleResponsibilityMatrix(t *testing.T) {
	tests := []struct {
		from continuity.ResponseStatus
		to   continuity.ResponseStatus
		want authority.Responsibility
		mat  int
	}{
		{continuity.ResponseDraft, continuity.ResponseInReview, authority.ResponsibilityReviewer, 3},
		{continuity.ResponseDraft, continuity.ResponseWithdrawn, authority.ResponsibilityProposer, 2},
		{continuity.ResponseInReview, continuity.ResponseApproved, authority.ResponsibilitySignatory, 4},
		{continuity.ResponseInReview, continuity.ResponseRejected, authority.ResponsibilityReviewer, 3},
		{continuity.ResponseInReview, continuity.ResponseDraft, authority.ResponsibilityReviewer, 3},
		{continuity.ResponseInReview, continuity.ResponseWithdrawn, authority.ResponsibilityProposer, 2},
		{continuity.ResponseApproved, continuity.ResponseTransmitted, authority.ResponsibilityTransmitter, 4},
		{continuity.ResponseApproved, continuity.ResponseWithdrawn, authority.ResponsibilitySignatory, 4},
		{continuity.ResponseTransmitted, continuity.ResponseAcknowledged, authority.ResponsibilityAcknowledger, 3},
		{continuity.ResponseRejected, continuity.ResponseDraft, authority.ResponsibilityProposer, 2},
	}
	for _, test := range tests {
		got, err := continuity.ResponseLifecyclePolicy(test.from, test.to)
		if err != nil || authority.Responsibility(got.Responsibility) != test.want || got.Materiality != test.mat {
			t.Fatalf("%s -> %s: got responsibility=%s materiality=%d err=%v", test.from, test.to, got.Responsibility, got.Materiality, err)
		}
	}
	if _, err := continuity.ResponseLifecyclePolicy(continuity.ResponseDraft, continuity.ResponseAcknowledged); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("expected invalid lifecycle transition, got %v", err)
	}
}

func TestResponsePreparationUsesProposerResponsibility(t *testing.T) {
	api := &API{}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}
	got, err := api.lifecycleCommandPolicy(context.Background(), lifecycleRequest(""), "bank", "matter.response.add", map[string]any{}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if got.Responsibility != authority.ResponsibilityProposer || got.Materiality != 2 {
		t.Fatalf("unexpected response preparation policy: %#v", got)
	}
}

func TestLifecyclePolicyLoadsCurrentDecisionStateBeforeAuthorization(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "reviewer", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterRegulatoryChange, Priority: 3, Title: "Decision lifecycle", Summary: "Test lifecycle authority.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.RecordDecisionLifecycle(ctx, continuity.AddDecisionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "POSITION", Status: continuity.DecisionProposed, Rationale: "Propose.", AuthorityPrincipalID: "proposer"})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	policy, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.decision.record", map[string]any{"type": "POSITION", "status": "IN_REVIEW"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Responsibility != authority.ResponsibilityReviewer || policy.Materiality != 3 || policy.ActorField != "authority_principal_id" {
		t.Fatalf("unexpected lifecycle policy: %#v", policy)
	}
}

func TestLifecyclePolicyUsesRouteMatterAndPriorityFloor(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "proposer", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterException, Priority: 5, Title: "Material exception", Summary: "High-impact exception.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	policy, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.decision.record", map[string]any{"type": "RISK_ACCEPTANCE", "status": "PROPOSED"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Responsibility != authority.ResponsibilityProposer || policy.Materiality != 5 {
		t.Fatalf("route-bound priority was not enforced: %#v", policy)
	}
}

func TestLifecyclePolicyRejectsBodyMatterThatConflictsWithRoute(t *testing.T) {
	api := &API{}
	_, err := api.lifecycleCommandPolicy(context.Background(), lifecycleRequest("matter-route"), "bank", "matter.action.add", map[string]any{"matter_id": "matter-body"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2})
	if !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("expected route/body identifier conflict, got %v", err)
	}
}

func TestLifecyclePolicyRejectsRestrictedMatterActionOwnerWithoutVisibility(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "allowed-owner", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterAuthorityRequest, Priority: 5,
		Title: "Restricted authority request", Summary: "Protected response work.",
		Scope:            json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["allowed-owner"]}`),
		OwnerPrincipalID: "allowed-owner", ActorID: "allowed-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "allowed-owner", DisplayName: "Allowed owner"}},
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "allowed-owner", DisplayName: "Allowed owner"}},
	}}}}
	base := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}
	if _, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.action.add", map[string]any{"owner_principal_id": "blocked-owner"}, base); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("expected restricted owner rejection, got %v", err)
	}
	policy, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.action.add", map[string]any{"owner_principal_id": "allowed-owner"}, base)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Materiality != 5 {
		t.Fatalf("expected Matter priority materiality floor, got %#v", policy)
	}
}

func TestLifecyclePolicyRejectsVisibleActionOwnerOutsideCurrentPerformerRoute(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "current-owner", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "current-owner", ActorID: "current-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{
		Continuity: service,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "current-owner", DisplayName: "Current owner"}},
			authority.ResponsibilityPerformer: {
				Principal:           authority.Principal{ID: "performer-1", DisplayName: "Operations lead"},
				CandidatePrincipals: []authority.Principal{{ID: "performer-2", DisplayName: "Operations analyst"}},
			},
		}},
	}}
	base := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}

	if _, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.action.add", map[string]any{"owner_principal_id": "current-owner"}, base); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("visible accountable owner was accepted without a current performer route: %v", err)
	}
	if _, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.action.add", map[string]any{"owner_principal_id": "performer-2"}, base); err != nil {
		t.Fatalf("current eligible performer was rejected: %v", err)
	}
}

func TestUnrelatedCommandDoesNotRequireContinuityService(t *testing.T) {
	api := &API{}
	policy := commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}
	got, err := api.lifecycleCommandPolicy(context.Background(), lifecycleRequest(""), "bank", "program.requirement.add", map[string]any{}, policy)
	if err != nil || got != policy {
		t.Fatalf("unrelated command was coupled to continuity service: got=%#v err=%v", got, err)
	}
}

func lifecycleRequest(matterID string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matterID, nil)
	if matterID != "" {
		r.SetPathValue("id", matterID)
	}
	return r
}

func TestMatterAssignmentLifecycleValidatesDistinctOwnerAndPerformerCandidates(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "current-owner", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterAuthorityRequest, Priority: 5,
		Title: "Restricted authority request", Summary: "Protected response work.",
		Scope:            json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["current-owner","owner-2","performer-2"]}`),
		OwnerPrincipalID: "current-owner", ActorID: "current-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(ctx, continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Prepare response", Description: "Assemble the requested records.", OwnerPrincipalID: "current-owner", ActorID: "current-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {
			Principal:           authority.Principal{ID: "current-owner", DisplayName: "Current owner"},
			CandidatePrincipals: []authority.Principal{{ID: "owner-2", DisplayName: "Privacy owner"}},
		},
		authority.ResponsibilityPerformer: {
			Principal:           authority.Principal{ID: "performer-1", DisplayName: "Current performer"},
			CandidatePrincipals: []authority.Principal{{ID: "performer-2", DisplayName: "Privacy operations analyst"}},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	base := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}

	if _, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.assign", map[string]any{"owner_principal_id": "owner-2"}, base); err != nil {
		t.Fatalf("eligible accountable owner was rejected: %v", err)
	}
	actionRequest := lifecycleRequest(matter.Matter.ID)
	actionRequest.SetPathValue("action_id", actionID)
	if _, err := api.lifecycleCommandPolicy(ctx, actionRequest, "bank", "matter.action.assign", map[string]any{"owner_principal_id": "performer-2"}, base); err != nil {
		t.Fatalf("eligible performer was rejected: %v", err)
	}
	if _, err := api.lifecycleCommandPolicy(ctx, actionRequest, "bank", "matter.action.assign", map[string]any{"owner_principal_id": "owner-2"}, base); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("accountable owner was accepted as an Action performer without a performer route: %v", err)
	}
	if _, err := api.lifecycleCommandPolicy(ctx, lifecycleRequest(matter.Matter.ID), "bank", "matter.assign", map[string]any{"owner_principal_id": "outside-scope"}, base); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("restricted invisible owner was accepted: %v", err)
	}
}

func TestActionTransitionUsesPerformerResponsibility(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "performer", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", OwnerPrincipalID: "owner", ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(ctx, continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "performer", ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := lifecycleRequest(matter.Matter.ID)
	request.SetPathValue("action_id", matter.Actions[0].ID)
	api := &API{deps: Dependencies{Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer", DisplayName: "Action performer"}},
	}}}}
	policy, err := api.lifecycleCommandPolicy(ctx, request, "bank", "matter.action.transition", map[string]any{"to": "IN_PROGRESS"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Responsibility != authority.ResponsibilityPerformer || policy.Materiality != 4 {
		t.Fatalf("Action transition did not preserve performer responsibility: %#v", policy)
	}
}

type assignmentAuthorityStub struct {
	resolutions map[authority.Responsibility]authority.Resolution
	err         error
}

func (s *assignmentAuthorityStub) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	if s.err != nil {
		return authority.Resolution{}, s.err
	}
	return s.resolutions[input.Responsibility], nil
}

func (s *assignmentAuthorityStub) ResolveMany(_ context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	outcomes := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		if s.err != nil {
			outcomes[index].Err = s.err
			continue
		}
		outcomes[index].Resolution = s.resolutions[input.Responsibility]
	}
	return outcomes, nil
}

func (s *assignmentAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s *assignmentAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s *assignmentAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

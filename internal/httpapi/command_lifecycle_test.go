package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
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
		got, mat, err := decisionLifecyclePolicy(test.status)
		if err != nil || got != test.want || mat != test.mat {
			t.Fatalf("%s: got responsibility=%s materiality=%d err=%v", test.status, got, mat, err)
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
		got, mat, err := responseLifecyclePolicy(test.from, test.to)
		if err != nil || got != test.want || mat != test.mat {
			t.Fatalf("%s -> %s: got responsibility=%s materiality=%d err=%v", test.from, test.to, got, mat, err)
		}
	}
	if _, _, err := responseLifecyclePolicy(continuity.ResponseDraft, continuity.ResponseAcknowledged); !errors.Is(err, continuity.ErrInvalidState) {
		t.Fatalf("expected invalid lifecycle transition, got %v", err)
	}
}

func TestResponsePreparationUsesProposerResponsibility(t *testing.T) {
	api := &API{}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}
	got, err := api.lifecycleCommandPolicy(context.Background(), "bank", "matter.response.add", map[string]any{}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if got.Responsibility != authority.ResponsibilityProposer || got.Materiality != 2 {
		t.Fatalf("unexpected response preparation policy: %#v", got)
	}
}

func TestLifecyclePolicyLoadsCurrentDecisionStateBeforeAuthorization(t *testing.T) {
	ctx := context.Background()
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "bank", Type: continuity.MatterRegulatoryChange, Priority: 3, Title: "Decision lifecycle", Summary: "Test lifecycle authority.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.RecordDecisionLifecycle(ctx, continuity.AddDecisionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "POSITION", Status: continuity.DecisionProposed, Rationale: "Propose.", AuthorityPrincipalID: "proposer"})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	policy, err := api.lifecycleCommandPolicy(ctx, "bank", "matter.decision.record", map[string]any{"matter_id": matter.Matter.ID, "type": "POSITION", "status": "IN_REVIEW"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Responsibility != authority.ResponsibilityReviewer || policy.Materiality != 3 || policy.ActorField != "authority_principal_id" {
		t.Fatalf("unexpected lifecycle policy: %#v", policy)
	}
}

func TestUnrelatedCommandDoesNotRequireContinuityService(t *testing.T) {
	api := &API{}
	policy := commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}
	got, err := api.lifecycleCommandPolicy(context.Background(), "bank", "program.requirement.add", map[string]any{}, policy)
	if err != nil || got != policy {
		t.Fatalf("unrelated command was coupled to continuity service: got=%#v err=%v", got, err)
	}
}

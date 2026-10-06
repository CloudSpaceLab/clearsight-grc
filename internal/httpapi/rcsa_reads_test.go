package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
)

func TestRCSACycleVisibilityUsesOwnerAndExactReviewerAuthority(t *testing.T) {
	reviewer := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}
	resolver := authority.NewResolver("test-v1", []authority.Rule{{
		ID: "review-rule", TenantID: "bank", LegalEntityID: "entity-a",
		ObjectType: "RCSA_CYCLE", ObjectID: "cycle-review",
		Responsibility: authority.ResponsibilityReviewer,
		DecisionType:   "rcsa.challenge.start", MinMateriality: 3,
		Principal: authority.Principal{ID: "reviewer-a", DisplayName: "Reviewer A", Kind: "PERSON"},
		Priority: 1,
	}})
	api := &API{deps: Dependencies{Authority: resolver}}
	items := []rcsa.CycleSummary{
		{Cycle: rcsa.Cycle{ID: "cycle-owner", LegalEntityID: "entity-a", FirstLineOwnerID: "reviewer-a", Status: rcsa.StatusAssessmentOpen}},
		{Cycle: rcsa.Cycle{ID: "cycle-review", LegalEntityID: "entity-a", FirstLineOwnerID: "owner-b", Status: rcsa.StatusAwaitingChallenge}},
		{Cycle: rcsa.Cycle{ID: "cycle-hidden", LegalEntityID: "entity-a", FirstLineOwnerID: "owner-c", Status: rcsa.StatusAwaitingChallenge}},
	}
	visible, complete := api.visibleRCSACycleSummaries(t.Context(), reviewer, items)
	if !complete || len(visible) != 2 || visible[0].Cycle.ID != "cycle-owner" || visible[1].Cycle.ID != "cycle-review" {
		t.Fatalf("visible=%#v complete=%v", visible, complete)
	}
	if !api.canReadRCSACycle(t.Context(), reviewer, items[1].Cycle) {
		t.Fatal("effective reviewer could not read the exact challenge cycle")
	}
	if api.canReadRCSACycle(t.Context(), reviewer, items[2].Cycle) {
		t.Fatal("reviewer authority leaked to an unrelated cycle")
	}
}

func TestRCSAReviewerReadDoesNotExposeFirstLineStage(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer"}
	for _, status := range []rcsa.Status{rcsa.StatusDraft, rcsa.StatusAssessmentOpen} {
		if _, ok := rcsaReviewerReadInput(actor, rcsa.Cycle{ID: "cycle", LegalEntityID: "entity-a", Status: status}); ok {
			t.Fatalf("status %s unexpectedly exposed to reviewer", status)
		}
	}
	input, ok := rcsaReviewerReadInput(actor, rcsa.Cycle{ID: "cycle", LegalEntityID: "entity-a", Status: rcsa.StatusAwaitingChallenge})
	if !ok || input.DecisionType != "rcsa.challenge.start" || input.Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("challenge start input=%#v ok=%v", input, ok)
	}
	input, ok = rcsaReviewerReadInput(actor, rcsa.Cycle{
		ID: "cycle", LegalEntityID: "entity-a", Status: rcsa.StatusAwaitingChallenge, ChallengeMatterID: "matter-1",
	})
	if !ok || input.DecisionType != "rcsa.challenge.complete" {
		t.Fatalf("challenge completion input=%#v ok=%v", input, ok)
	}
}

func TestRCSAHandoffUsesExistingEvidenceAndMatterTargets(t *testing.T) {
	cases := []struct {
		name       string
		cycle      rcsa.Cycle
		requestID  string
		stage      string
		label      string
		targetType string
		targetID   string
	}{
		{name: "draft", cycle: rcsa.Cycle{Status: rcsa.StatusDraft}, stage: "SETUP", label: "First-line assessment not started"},
		{name: "first line", cycle: rcsa.Cycle{Status: rcsa.StatusAssessmentOpen}, requestID: "request-1", stage: "FIRST_LINE", label: "Complete first-line assessment", targetType: "EVIDENCE_REQUEST", targetID: "request-1"},
		{name: "challenge ready", cycle: rcsa.Cycle{Status: rcsa.StatusAwaitingChallenge}, stage: "CHALLENGE", label: "Start independent challenge"},
		{name: "challenge work", cycle: rcsa.Cycle{Status: rcsa.StatusAwaitingChallenge, ChallengeMatterID: "matter-1"}, stage: "CHALLENGE", label: "Complete independent challenge", targetType: "MATTER", targetID: "matter-1"},
		{name: "complete", cycle: rcsa.Cycle{Status: rcsa.StatusCompleted, ChallengeMatterID: "matter-1"}, stage: "COMPLETE", label: "Completed", targetType: "MATTER", targetID: "matter-1"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := rcsaCycleHandoff(test.cycle, test.requestID)
			if got.Stage != test.stage || got.Label != test.label || got.TargetType != test.targetType || got.TargetID != test.targetID {
				t.Fatalf("handoff=%#v", got)
			}
		})
	}
}

func TestRCSAListAuthorityInfrastructureFailureIsPartial(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}
	api := &API{deps: Dependencies{Authority: failingRCSAAuthority{}}}
	visible, complete := api.visibleRCSACycleSummaries(t.Context(), actor, []rcsa.CycleSummary{{
		Cycle: rcsa.Cycle{ID: "cycle-review", LegalEntityID: "entity-a", FirstLineOwnerID: "owner", Status: rcsa.StatusAwaitingChallenge},
	}})
	if complete || len(visible) != 0 {
		t.Fatalf("visible=%#v complete=%v", visible, complete)
	}
}

type failingRCSAAuthority struct{}

func (failingRCSAAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return authority.Resolution{}, errors.New("authority unavailable")
}
func (failingRCSAAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, errors.New("authority unavailable")
}
func (failingRCSAAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, errors.New("authority unavailable")
}
func (failingRCSAAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, errors.New("authority unavailable")
}
func (failingRCSAAuthority) ResolveMany(context.Context, []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	return nil, errors.New("authority unavailable")
}

func TestRCSACycleVisibilityPreservesSourceOrder(t *testing.T) {
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}
	resolver := authority.NewResolver("test-v1", []authority.Rule{{
		ID: "review-first", TenantID: "bank", LegalEntityID: "entity-a",
		ObjectType: "RCSA_CYCLE", ObjectID: "cycle-review",
		Responsibility: authority.ResponsibilityReviewer,
		DecisionType:   "rcsa.challenge.start", MinMateriality: 3,
		Principal: authority.Principal{ID: "reviewer-a", Kind: "PERSON"},
		Priority: 1,
	}})
	api := &API{deps: Dependencies{Authority: resolver}}
	items := []rcsa.CycleSummary{
		{Cycle: rcsa.Cycle{ID: "cycle-review", LegalEntityID: "entity-a", FirstLineOwnerID: "owner", Status: rcsa.StatusAwaitingChallenge}},
		{Cycle: rcsa.Cycle{ID: "cycle-owner", LegalEntityID: "entity-a", FirstLineOwnerID: "reviewer-a", Status: rcsa.StatusAssessmentOpen}},
	}
	visible, complete := api.visibleRCSACycleSummaries(t.Context(), actor, items)
	if !complete || len(visible) != 2 || visible[0].Cycle.ID != "cycle-review" || visible[1].Cycle.ID != "cycle-owner" {
		t.Fatalf("visible=%#v complete=%v", visible, complete)
	}
}

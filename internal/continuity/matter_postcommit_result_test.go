package continuity

import (
	"encoding/json"
	"testing"
)

func TestChangeMatterContextReturnsCommittedFallbackWhenCurrentReadFails(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	base := NewMemoryRepository()
	setup := NewService(base)
	matter, err := setup.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", Scope: json.RawMessage(`{}`),
		KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`["final DPCO checklist"]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := &failReadAfterMatterApplyRepository{Repository: base}
	service := NewService(repo)
	updated, err := service.ChangeMatterContext(ctx, ChangeMatterContextInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Kind: MatterContextResolveMissing, Key: "final_dpco_checklist", Label: "final DPCO checklist",
		Value: json.RawMessage(`"Checklist v3"`), EvidenceReferences: json.RawMessage(`["artifact-v3"]`),
		ActorID: "owner-1", Rationale: "Record the approved checklist.",
	})
	if err != nil {
		t.Fatalf("committed context change was reported as failed: %v", err)
	}
	if updated.Matter.Version != matter.Matter.Version+1 || string(updated.Matter.MissingFacts) != `[]` {
		t.Fatalf("fallback does not describe the committed context change: %#v", updated)
	}
	var facts map[string]any
	if err := json.Unmarshal(updated.Matter.KnownFacts, &facts); err != nil {
		t.Fatal(err)
	}
	if facts["final_dpco_checklist"] != "Checklist v3" {
		t.Fatalf("fallback omitted committed information: %#v", facts)
	}
}

func TestAssignActionReturnsCommittedFallbackWhenCurrentReadFails(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	base := NewMemoryRepository()
	setup := NewService(base)
	matter, err := setup.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 3,
		Title: "Account review gap", Summary: "Resolve accounts without current ownership evidence.", Scope: json.RawMessage(`{}`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = setup.AddAction(ctx, AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Confirm account owners", Description: "Obtain current ownership confirmation.",
		OwnerPrincipalID: "performer-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := &failReadAfterMatterApplyRepository{Repository: base}
	service := NewService(repo)
	updated, err := service.AssignAction(ctx, AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: matter.Actions[0].ID,
		ExpectedVersion: matter.Matter.Version, OwnerPrincipalID: "performer-2",
		ActorID: "owner-1", Rationale: "Assign the current process owner.",
	})
	if err != nil {
		t.Fatalf("committed action assignment was reported as failed: %v", err)
	}
	if updated.Matter.Version != matter.Matter.Version+1 || len(updated.Actions) != 1 || updated.Actions[0].OwnerPrincipalID != "performer-2" || updated.Actions[0].Version != matter.Actions[0].Version+1 {
		t.Fatalf("fallback does not describe the committed action assignment: %#v", updated)
	}
}

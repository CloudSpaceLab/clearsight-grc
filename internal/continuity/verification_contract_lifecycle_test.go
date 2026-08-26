package continuity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSupersedeVerificationContractPreservesHistoryAndMovesClosureToReplacement(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 4,
		Title: "Access review gap", Summary: "The access review outcome must be confirmed.",
		Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "Every privileged account is approved.", Baseline: json.RawMessage(`{"unapproved":4}`),
		Scope: json.RawMessage(`{"population":"privileged accounts"}`), Threshold: json.RawMessage(`{"unapproved":0}`),
		ObservationPeriodMinutes: 0, AuthorityPrincipalID: "reviewer-old", FailureResponse: "BLOCK_CLOSE", ActorID: "reviewer-old",
	})
	if err != nil {
		t.Fatal(err)
	}
	prior := matter.VerificationContracts[0]
	now = now.Add(time.Minute)
	matter, err = service.RecordVerificationResult(ctx, RecordVerificationResultInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: prior.ID,
		Result: VerificationFailed, Observations: json.RawMessage(`{"unapproved":1}`), EvidenceReferences: json.RawMessage(`[]`),
		ReviewerPrincipalID: "reviewer-old", ReviewerAuthorityPrincipalID: "reviewer-old", Rationale: "One account remains unapproved.", ObservedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeSupersede := now
	now = now.Add(time.Minute)
	updated, err := service.SupersedeVerificationContract(ctx, SupersedeVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ContractID: prior.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "Every privileged account has current approval.", Baseline: json.RawMessage(`{"unapproved":1}`),
		Scope: json.RawMessage(`{"population":"privileged accounts"}`), Threshold: json.RawMessage(`{"unapproved":0}`),
		ObservationPeriodMinutes: 60, AuthorityPrincipalID: "reviewer-current", FailureResponse: "REOPEN",
		ActorID: "reviewer-current", Rationale: "Correct the observation period and reviewer before the next check.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.VerificationContracts) != 2 || len(updated.VerificationResults) != 1 {
		t.Fatalf("history was not preserved: contracts=%#v results=%#v", updated.VerificationContracts, updated.VerificationResults)
	}
	retired, replacement := updated.VerificationContracts[0], updated.VerificationContracts[1]
	if retired.ID != prior.ID || retired.Status != VerificationRetired || retired.Version != prior.Version+1 {
		t.Fatalf("prior contract was not retired: %#v", retired)
	}
	if replacement.Status != VerificationActive || replacement.SupersedesContractID != prior.ID || replacement.Version != 1 || replacement.AuthorityPrincipalID != "reviewer-current" {
		t.Fatalf("replacement contract lineage is incomplete: %#v", replacement)
	}
	if updated.VerificationResults[0].ContractID != prior.ID {
		t.Fatalf("historical result moved to replacement: %#v", updated.VerificationResults[0])
	}
	closure := strings.Join(updated.Closure.Reasons, " ")
	if !strings.Contains(closure, "1 outcome check(s) have no result") || strings.Contains(closure, "did not pass") {
		t.Fatalf("closure did not use only the active replacement: %#v", updated.Closure)
	}

	historical, err := service.MatterAt(ctx, "bank", matter.Matter.ID, beforeSupersede)
	if err != nil {
		t.Fatal(err)
	}
	if len(historical.VerificationContracts) != 1 || historical.VerificationContracts[0].Status != VerificationActive {
		t.Fatalf("point-in-time contract history was not reconstructable: %#v", historical.VerificationContracts)
	}
	events, err := repo.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if events[len(events)-1].Type != EventVerificationContractSuperseded || events[len(events)-1].ActorID != "reviewer-current" {
		t.Fatalf("supersession event = %#v", events[len(events)-1])
	}
}

func TestRetireVerificationContractIsOptimisticAndLeavesHistory(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 26, 19, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 3, Title: "Retirement proof", Summary: "Retire a mistaken outcome check.",
		Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ExpectedOutcome: "Wrong population",
		Baseline: json.RawMessage(`{}`), Scope: json.RawMessage(`{"population":"wrong"}`), Threshold: json.RawMessage(`{"exceptions":0}`),
		AuthorityPrincipalID: "reviewer", FailureResponse: "BLOCK_CLOSE", ActorID: "reviewer",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := matter.VerificationContracts[0]
	retired, err := service.RetireVerificationContract(ctx, RetireVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ContractID: contract.ID, ExpectedVersion: matter.Matter.Version,
		ActorID: "reviewer", Rationale: "The population was attached to the wrong issue.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(retired.VerificationContracts) != 1 || retired.VerificationContracts[0].Status != VerificationRetired || retired.VerificationContracts[0].Version != 2 {
		t.Fatalf("retired contract history = %#v", retired.VerificationContracts)
	}
	if strings.Contains(strings.Join(retired.Closure.Reasons, " "), "has no result") {
		t.Fatalf("retired contract still blocks closure as an active check: %#v", retired.Closure)
	}
	_, err = service.RetireVerificationContract(ctx, RetireVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ContractID: contract.ID, ExpectedVersion: retired.Matter.Version,
		ActorID: "reviewer", Rationale: "Duplicate retirement.",
	})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("duplicate retirement error = %v, want invalid state", err)
	}
	_, err = service.SupersedeVerificationContract(ctx, SupersedeVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ContractID: contract.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "Replacement", Scope: json.RawMessage(`{}`), Baseline: json.RawMessage(`{}`), Threshold: json.RawMessage(`{}`),
		AuthorityPrincipalID: "reviewer", FailureResponse: "BLOCK_CLOSE", ActorID: "reviewer", Rationale: "Stale edit.",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale supersession error = %v, want version conflict", err)
	}
}

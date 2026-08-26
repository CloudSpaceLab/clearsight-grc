package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestVerificationFailureResponsesRunBeforeVerificationWithoutDuplicateSideEffects(t *testing.T) {
	for _, test := range []struct {
		response      string
		wantStatus    MatterStatus
		wantVersion   int64
		wantMatterLen int
	}{
		{response: "REOPEN", wantStatus: MatterActionsInProgress, wantVersion: 5, wantMatterLen: 1},
		{response: "ESCALATE", wantStatus: MatterDecisionRequired, wantVersion: 6, wantMatterLen: 1},
		{response: "CREATE_MATTER", wantStatus: MatterActionsInProgress, wantVersion: 5, wantMatterLen: 2},
		{response: "BLOCK_CLOSE", wantStatus: MatterActionsInProgress, wantVersion: 5, wantMatterLen: 1},
	} {
		t.Run(test.response, func(t *testing.T) {
			service, repo, input := verificationLifecycleFixture(test.response)

			matter, err := service.RecordVerificationResult(WithTrustedSystemScope(context.Background()), input)
			if err != nil {
				t.Fatal(err)
			}
			if matter.Matter.Status != test.wantStatus || matter.Matter.Version != test.wantVersion {
				t.Fatalf("failure response %s produced status/version %s/%d", test.response, matter.Matter.Status, matter.Matter.Version)
			}
			if len(matter.VerificationResults) != 1 || matter.VerificationResults[0].Result != VerificationFailed {
				t.Fatalf("failure result history was not preserved: %#v", matter.VerificationResults)
			}

			_, retryErr := service.RecordVerificationResult(WithTrustedSystemScope(context.Background()), input)
			if !errors.Is(retryErr, ErrVersionConflict) {
				t.Fatalf("replayed command error = %v, want version conflict", retryErr)
			}
			matters, err := repo.ListMatters(WithTrustedSystemScope(context.Background()), "bank", "", 20)
			if err != nil || len(matters) != test.wantMatterLen {
				t.Fatalf("failure response %s produced %d matters after retry, want %d: %v", test.response, len(matters), test.wantMatterLen, err)
			}
		})
	}
}

func TestActiveOutcomeContractAcceptsNewResultAndClosureUsesLatest(t *testing.T) {
	service, _, input := verificationLifecycleFixture("REOPEN")
	ctx := WithTrustedSystemScope(context.Background())

	failed, err := service.RecordVerificationResult(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	requirements, _ := CompileMatterWork(failed, input.ObservedAt.Add(time.Minute))
	if len(requirements) != 1 || requirements[0].CommandName != "matter.outcome.record" {
		t.Fatalf("failed check did not remain actionable: %#v", requirements)
	}

	service.now = func() time.Time { return input.ObservedAt.Add(time.Minute) }
	passed, err := service.RecordVerificationResult(ctx, RecordVerificationResultInput{
		TenantID: "bank", MatterID: input.MatterID, ExpectedVersion: failed.Matter.Version,
		ContractID: input.ContractID, Result: VerificationPassed,
		Observations: json.RawMessage(`{"unresolved":0}`), EvidenceReferences: json.RawMessage(`["evidence-2"]`),
		ReviewerPrincipalID: input.ReviewerPrincipalID, Rationale: "Remediation was independently rechecked and passed.", ObservedAt: input.ObservedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(passed.VerificationResults) != 2 {
		t.Fatalf("result history length = %d, want 2", len(passed.VerificationResults))
	}
	if !passed.Closure.Ready {
		t.Fatalf("latest valid pass must control closure: %#v", passed.Closure.Reasons)
	}
	requirements, _ = CompileMatterWork(passed, input.ObservedAt.Add(2*time.Minute))
	if len(requirements) != 0 {
		t.Fatalf("latest pass must complete outcome work: %#v", requirements)
	}
}

func verificationLifecycleFixture(failureResponse string) (*Service, *MemoryRepository, RecordVerificationResultInput) {
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	implementedAt := now.Add(-time.Hour)
	repo := NewMemoryRepository()
	repo.matters["bank"] = map[string]MatterAggregate{
		"matter-1": {
			Matter:  Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-1", Type: MatterAuditFinding, Status: MatterActionsInProgress, Version: 4},
			Actions: []Action{{ID: "action-1", MatterID: "matter-1", Status: ActionImplemented, OwnerPrincipalID: "owner", ImplementedAt: &implementedAt}},
			VerificationContracts: []VerificationContract{{
				ID: "contract-1", MatterID: "matter-1", ActionID: "action-1", ExpectedOutcome: "All exceptions are resolved.",
				AuthorityPrincipalID: "reviewer", FailureResponse: failureResponse, Status: VerificationActive, CreatedAt: implementedAt,
			}},
		},
	}
	repo.matterEvents["bank"] = map[string][]Event{"matter-1": {}}
	service := NewService(repo)
	service.now = func() time.Time { return now }
	return service, repo, RecordVerificationResultInput{
		TenantID: "bank", MatterID: "matter-1", ExpectedVersion: 4, ContractID: "contract-1", Result: VerificationFailed,
		Observations: json.RawMessage(`{"unresolved":1}`), EvidenceReferences: json.RawMessage(`["evidence-1"]`),
		ReviewerPrincipalID: "reviewer", Rationale: "One exception remains unresolved.", ObservedAt: now,
	}
}

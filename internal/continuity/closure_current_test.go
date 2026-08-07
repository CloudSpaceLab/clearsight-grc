package continuity

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestClosureUsesCurrentDecisionWithinDecisionType(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	aggregate := MatterAggregate{
		Matter: Matter{ID: "matter-1", TenantID: "bank", Type: MatterRegulatoryChange, Status: MatterVerification},
		Decisions: []Decision{
			{ID: "decision-1", Type: "REGULATORY_POSITION", Status: DecisionApproved, SelectedOption: "NO_CHANGE_REQUIRED", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: "decision-2", Type: "REGULATORY_POSITION", Status: DecisionRejected, SelectedOption: "NO_CHANGE_REQUIRED", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		},
	}

	closure := assessClosureAt(aggregate, now)
	if closure.Ready {
		t.Fatal("later rejection must invalidate the earlier approved decision")
	}
	if !containsClosureReason(closure, "current regulatory position") {
		t.Fatalf("missing current-decision blocker: %#v", closure.Reasons)
	}
}

func TestExceptionClosureRejectsExpiredOrUnresolvedApproval(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Minute)
	aggregate := MatterAggregate{
		Matter: Matter{ID: "exception-1", TenantID: "bank", Type: MatterException, Status: MatterVerification},
		Decisions: []Decision{{
			ID: "decision-1", Type: "EXCEPTION_APPROVAL", Status: DecisionApproved,
			ExpiresAt: &expired, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
		}},
	}
	if closure := assessClosureAt(aggregate, now); closure.Ready {
		t.Fatal("expired exception approval must not satisfy closure")
	}

	aggregate.Decisions = []Decision{{
		ID: "decision-2", Type: "EXCEPTION_APPROVAL", Status: DecisionConditionallyApproved,
		Conditions: json.RawMessage(`[{
			"id":"condition-1","status":"PENDING"
		}]`), CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute),
	}}
	if closure := assessClosureAt(aggregate, now); closure.Ready {
		t.Fatal("unresolved conditional approval must not satisfy closure")
	}

	aggregate.Decisions[0].Conditions = json.RawMessage(`[{
		"id":"condition-1","status":"SATISFIED"
	}]`)
	if closure := assessClosureAt(aggregate, now); !closure.Ready {
		t.Fatalf("resolved conditional approval should satisfy exception closure: %#v", closure.Reasons)
	}
}

func TestAuthorityRequestUsesCurrentResponseChain(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	transmitted := now.Add(-2 * time.Hour)
	acknowledged := now.Add(-90 * time.Minute)
	aggregate := MatterAggregate{
		Matter: Matter{ID: "request-1", TenantID: "bank", Type: MatterAuthorityRequest, Status: MatterVerification},
		ResponsePackages: []ResponsePackage{
			{
				ID: "response-1", Purpose: "Regulatory response", Audience: "NDPC", Status: ResponseAcknowledged,
				TransmittedAt: &transmitted, AcknowledgedAt: &acknowledged,
				CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: acknowledged,
			},
			{
				ID: "response-2", Purpose: "Regulatory response", Audience: "NDPC", Status: ResponseWithdrawn,
				CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
			},
		},
	}

	closure := assessClosureAt(aggregate, now)
	if closure.Ready {
		t.Fatal("a withdrawn replacement must invalidate the historical acknowledged response")
	}
	if !containsClosureReason(closure, "current external response") {
		t.Fatalf("missing current-response blocker: %#v", closure.Reasons)
	}
}

func TestVerificationClosureRequiresChronologyAuthorityAndIndependence(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	implementedAt := now.Add(-2 * time.Hour)
	action := Action{
		ID: "action-1", OwnerPrincipalID: "owner", Status: ActionImplemented,
		ImplementedAt: &implementedAt, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: implementedAt,
	}
	contract := VerificationContract{
		ID: "contract-1", ActionID: action.ID, Status: VerificationActive,
		AuthorityPrincipalID: "reviewer", ObservationPeriodMinutes: 60,
		CreatedAt: now.Add(-3 * time.Hour),
	}

	premature := VerificationResult{
		ID: "result-1", ContractID: contract.ID, Result: VerificationPassed,
		ReviewerPrincipalID: "reviewer", ObservedAt: implementedAt.Add(30 * time.Minute), CreatedAt: implementedAt.Add(30 * time.Minute),
	}
	aggregate := MatterAggregate{
		Matter:  Matter{ID: "finding-1", TenantID: "bank", Type: MatterAuditFinding, Status: MatterVerification},
		Actions: []Action{action}, VerificationContracts: []VerificationContract{contract}, VerificationResults: []VerificationResult{premature},
	}
	if closure := assessClosureAt(aggregate, now); closure.Ready || !containsClosureReason(closure, "not yet valid") {
		t.Fatalf("premature PASS must not satisfy closure: %#v", closure.Reasons)
	}

	contract.AuthorityPrincipalID = ""
	aggregate.VerificationContracts[0] = contract
	aggregate.VerificationResults[0] = VerificationResult{
		ID: "result-2", ContractID: contract.ID, Result: VerificationPassed,
		ReviewerPrincipalID: "owner", ObservedAt: implementedAt.Add(90 * time.Minute), CreatedAt: implementedAt.Add(90 * time.Minute),
	}
	if closure := assessClosureAt(aggregate, now); closure.Ready {
		t.Fatal("action owner must not independently verify their own implementation")
	}

	contract.AuthorityPrincipalID = "reviewer"
	aggregate.VerificationContracts[0] = contract
	aggregate.VerificationResults[0] = VerificationResult{
		ID: "result-3", ContractID: contract.ID, Result: VerificationPassed,
		ReviewerPrincipalID: "reviewer", ObservedAt: implementedAt.Add(90 * time.Minute), CreatedAt: implementedAt.Add(90 * time.Minute),
	}
	if closure := assessClosureAt(aggregate, now); !closure.Ready {
		t.Fatalf("valid independent PASS after the observation window should allow closure: %#v", closure.Reasons)
	}
}

func TestRecordVerificationResultRejectsInvalidPassBeforePersistence(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	implementedAt := now.Add(-30 * time.Minute)
	repo := NewMemoryRepository()
	repo.matters["bank"] = map[string]MatterAggregate{
		"matter-1": {
			Matter:  Matter{ID: "matter-1", TenantID: "bank", Type: MatterAuditFinding, Status: MatterVerification, Version: 4},
			Actions: []Action{{ID: "action-1", OwnerPrincipalID: "owner", Status: ActionImplemented, ImplementedAt: &implementedAt}},
			VerificationContracts: []VerificationContract{{
				ID: "contract-1", ActionID: "action-1", Status: VerificationActive,
				ObservationPeriodMinutes: 60, CreatedAt: now.Add(-time.Hour),
			}},
		},
	}
	repo.matterEvents["bank"] = map[string][]Event{"matter-1": {}}
	service := NewService(repo)
	service.now = func() time.Time { return now }

	_, err := service.RecordVerificationResult(context.Background(), RecordVerificationResultInput{
		TenantID: "bank", MatterID: "matter-1", ExpectedVersion: 4, ContractID: "contract-1",
		Result: VerificationPassed, ReviewerPrincipalID: "owner", Rationale: "Checked.", ObservedAt: now,
	})
	if err == nil || !strings.Contains(err.Error(), "independent") {
		t.Fatalf("self-review should fail before persistence, got %v", err)
	}

	_, err = service.RecordVerificationResult(context.Background(), RecordVerificationResultInput{
		TenantID: "bank", MatterID: "matter-1", ExpectedVersion: 4, ContractID: "contract-1",
		Result: VerificationPassed, ReviewerPrincipalID: "reviewer", Rationale: "Checked.", ObservedAt: now,
	})
	if err == nil || !strings.Contains(err.Error(), "observation period") {
		t.Fatalf("premature review should fail before persistence, got %v", err)
	}
}

func containsClosureReason(closure ClosureAssessment, fragment string) bool {
	for _, reason := range closure.Reasons {
		if strings.Contains(strings.ToLower(reason), strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

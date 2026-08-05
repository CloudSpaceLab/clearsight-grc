package bankverticals

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestJourneyCompletionUsesCurrentRecords(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	if currentDecisionApproved([]continuity.Decision{
		{ID: "old", Status: continuity.DecisionApproved, UpdatedAt: now.Add(-time.Hour)},
		{ID: "new", Status: continuity.DecisionRejected, UpdatedAt: now},
	}) {
		t.Fatal("superseded approved decision satisfied the journey")
	}

	matter := continuity.MatterAggregate{ResponsePackages: []continuity.ResponsePackage{
		{ID: "old", Status: continuity.ResponseAcknowledged, UpdatedAt: now.Add(-time.Hour)},
		{ID: "new", Status: continuity.ResponseWithdrawn, UpdatedAt: now},
	}}
	if responseAtLeast(matter, continuity.ResponseApproved) {
		t.Fatal("withdrawn current response inherited an old acknowledgement")
	}
}

func TestVerificationRequiresActiveIndependentPassingResults(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	matter := continuity.MatterAggregate{
		Actions:               []continuity.Action{{ID: "action-1", OwnerPrincipalID: "owner", Status: continuity.ActionImplemented}},
		VerificationContracts: []continuity.VerificationContract{{ID: "contract-1", ActionID: "action-1", AuthorityPrincipalID: "reviewer", Status: continuity.VerificationActive}},
		VerificationResults:   []continuity.VerificationResult{{ID: "result-1", ContractID: "contract-1", Result: continuity.VerificationPassed, ReviewerPrincipalID: "owner", ObservedAt: now}},
	}
	if latestVerificationPassed(matter) {
		t.Fatal("remediation owner was treated as an independent reviewer")
	}
	matter.VerificationResults[0].ReviewerPrincipalID = "reviewer"
	if !latestVerificationPassed(matter) {
		t.Fatal("independent passing result did not satisfy the active contract")
	}
	matter.VerificationContracts[0].Status = continuity.VerificationRetired
	if latestVerificationPassed(matter) {
		t.Fatal("retired contract was treated as a current outcome check")
	}
}

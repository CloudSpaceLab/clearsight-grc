package continuity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRegulatoryClosureCannotMaskAdverseCurrentDecisionWithUnrelatedApproval(t *testing.T) {
	now := time.Date(2026, 8, 7, 16, 0, 0, 0, time.UTC)
	aggregate := MatterAggregate{
		Matter: Matter{Type: MatterRegulatoryChange},
		Decisions: []Decision{
			{ID: "approved", Type: "IMPLEMENTATION_APPROACH", Status: DecisionApproved, SelectedOption: "NO_CHANGE_REQUIRED", Conditions: json.RawMessage(`[]`), CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: "rejected", Type: "REGULATORY_POSITION", Status: DecisionRejected, SelectedOption: "REJECT", Conditions: json.RawMessage(`[]`), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		},
	}
	closure := assessClosureAt(aggregate, now)
	if closure.Ready {
		t.Fatal("an unrelated approval must not mask an adverse current regulatory decision")
	}
	if !containsClosureReason(closure.Reasons, "every current regulatory decision") {
		t.Fatalf("expected current-decision blocker, got %#v", closure.Reasons)
	}
}

func TestAuthorityRequestRequiresEveryCurrentResponseLineageAcknowledged(t *testing.T) {
	now := time.Date(2026, 8, 7, 16, 0, 0, 0, time.UTC)
	transmitted := now.Add(-2 * time.Hour)
	acknowledged := now.Add(-time.Hour)
	aggregate := MatterAggregate{
		Matter: Matter{Type: MatterAuthorityRequest},
		ResponsePackages: []ResponsePackage{
			{ID: "complete", Purpose: "Incident records", Audience: "Regulator", Status: ResponseAcknowledged, TransmittedAt: &transmitted, AcknowledgedAt: &acknowledged, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: acknowledged},
			{ID: "incomplete", Purpose: "Supplemental records", Audience: "Regulator", Status: ResponseDraft, CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)},
		},
	}
	closure := assessClosureAt(aggregate, now)
	if closure.Ready {
		t.Fatal("one acknowledged response must not mask another incomplete current response lineage")
	}
	if !containsClosureReason(closure.Reasons, "every current external response") {
		t.Fatalf("expected response-lineage blocker, got %#v", closure.Reasons)
	}
}

func containsClosureReason(values []string, part string) bool {
	part = strings.ToLower(part)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), part) {
			return true
		}
	}
	return false
}

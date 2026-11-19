package continuity

import (
	"testing"
	"time"
)

func TestRequirementReviewDecisionRequiresIndependentAssignedAuthority(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	p := Program{ID: "program", TenantID: "tenant", AuthorityPrincipalID: "reviewer"}
	req := Requirement{ID: "requirement", TenantID: "tenant", ProgramID: "program", Status: RequirementDraft}
	original := RequirementReview{ID: "review", TenantID: "tenant", ProgramID: "program", RequirementID: "requirement", ProposedBy: "owner", ProposedAt: now.Add(-time.Minute), Status: "PENDING", Version: 1}
	result, err := prepareRequirementReviewDecision(p, req, original, "reviewer", "Reviewed", true, now)
	if err != nil || result.Status != "APPROVED" || result.ReviewedBy != "reviewer" || result.Version != 2 {
		t.Fatalf("unexpected approval: %+v %v", result, err)
	}
	if original.Status != "PENDING" || original.ReviewedAt != nil {
		t.Fatal("input was mutated")
	}
	cases := []struct {
		name               string
		program            Program
		requirement        Requirement
		review             RequirementReview
		checker, rationale string
		at                 time.Time
	}{
		{"wrong actor", p, req, original, "other", "Reviewed", now},
		{"self approval", Program{ID: "program", TenantID: "tenant", AuthorityPrincipalID: "owner"}, req, original, "owner", "Reviewed", now},
		{"missing reason", p, req, original, "reviewer", " ", now},
		{"wrong tenant", p, Requirement{ID: "requirement", TenantID: "other", ProgramID: "program", Status: RequirementDraft}, original, "reviewer", "Reviewed", now},
		{"already effective", p, Requirement{ID: "requirement", TenantID: "tenant", ProgramID: "program", Status: RequirementApproved}, original, "reviewer", "Reviewed", now},
		{"before proposal", p, req, original, "reviewer", "Reviewed", now.Add(-time.Hour)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := prepareRequirementReviewDecision(tc.program, tc.requirement, tc.review, tc.checker, tc.rationale, true, tc.at); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	rejected, err := prepareRequirementReviewDecision(p, req, original, "reviewer", "Not ready", false, now)
	if err != nil || rejected.Status != "REJECTED" {
		t.Fatalf("unexpected rejection: %+v %v", rejected, err)
	}
	if _, err = prepareRequirementReviewDecision(p, req, rejected, "reviewer", "Again", true, now); err == nil {
		t.Fatal("decided review cannot be replayed")
	}
}

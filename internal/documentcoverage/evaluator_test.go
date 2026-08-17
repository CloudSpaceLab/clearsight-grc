package documentcoverage

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestEvaluateRejectsCrossJurisdictionKeywordOverlap(t *testing.T) {
	documentCandidate := Candidate{
		ID: "candidate-fed", Fingerprint: "fed", Eligible: true, Jurisdiction: "US",
		Statement: "The bank must maintain cybersecurity controls.", Action: "maintain", Object: "cybersecurity controls",
		Topics: []string{"bank", "cybersecurity", "controls"},
	}
	program := ProgramSnapshot{
		TenantID: "bank", ProgramID: "program-ndpa", Code: "NDPA-2023", Name: "Nigeria data protection",
		Type: "PRIVACY", Status: continuity.ProgramActive, Jurisdiction: "Nigeria",
		Requirements: []RequirementTarget{{
			ID: "req-1", Code: "NDPA-1", Statement: "Maintain cybersecurity controls", Status: continuity.RequirementApproved,
			Action: "maintain", Object: "cybersecurity controls", Topics: []string{"cybersecurity", "controls"},
			Applicability: continuity.ApplicabilityApplicable,
			Coverage:      continuity.RequirementCoverage{RequirementID: "req-1", Applicable: true, ControlImplemented: true, EvidenceSupported: true, Complete: true},
		}},
	}
	got := Evaluate([]Candidate{documentCandidate}, []ProgramSnapshot{program})
	if got.Metrics.EstimatedVerified.Numerator != 0 || got.Candidates[0].Classification != ClassificationGap {
		t.Fatalf("cross-jurisdiction text must not count: %#v", got)
	}
}

func TestEvaluateUsesCompleteChainOnly(t *testing.T) {
	documentCandidate, program := completeCoverageFixture()
	got := Evaluate([]Candidate{documentCandidate}, []ProgramSnapshot{program})
	if got.Metrics.EstimatedVerified.Numerator != 1 || got.Metrics.Verified.Numerator != 0 {
		t.Fatalf("strong unreviewed match should estimate but not verify: %#v", got.Metrics)
	}
	if got.Candidates[0].Classification != ClassificationPartialMatch {
		t.Fatalf("unreviewed match must remain partial: %#v", got.Candidates[0])
	}

	program.Requirements[0].Coverage.ControlImplemented = false
	program.Requirements[0].Coverage.EvidenceSupported = false
	program.Requirements[0].Coverage.Complete = false
	missingControl := Evaluate([]Candidate{documentCandidate}, []ProgramSnapshot{program})
	if missingControl.Metrics.EstimatedVerified.Numerator != 0 {
		t.Fatalf("missing control must not estimate verified coverage: %#v", missingControl.Metrics)
	}
}

func TestEvaluateZeroEligibleCandidatesHasNoPercentageDenominator(t *testing.T) {
	got := Evaluate([]Candidate{{ID: "context", Fingerprint: "context", Eligible: false, Statement: "Definitions."}}, nil)
	if got.Metrics.Verified.Denominator != 0 || got.Metrics.EstimatedVerified.Denominator != 0 {
		t.Fatalf("context must not create a denominator: %#v", got.Metrics)
	}
}

func completeCoverageFixture() (Candidate, ProgramSnapshot) {
	candidate := Candidate{
		ID: "candidate-ndpa", Fingerprint: "ndpa", Eligible: true, Jurisdiction: "Nigeria",
		Statement: "A data controller must retain processing records annually under section 41.", Modality: "MUST",
		Actor: "data controller", Action: "retain", Object: "processing records annually",
		Citations: []string{"section 41"}, Topics: []string{"controller", "processing", "records", "retain"}, Dates: []string{"annually"},
	}
	program := ProgramSnapshot{
		TenantID: "bank", ProgramID: "program-ndpa", Code: "NDPA-2023", Name: "Nigeria data protection",
		Type: "PRIVACY", Status: continuity.ProgramActive, Jurisdiction: "NG", Version: 2,
		Requirements: []RequirementTarget{{
			ID: "req-1", Code: "NDPA-41", Title: "Retain processing records",
			Statement: "The data controller must retain processing records annually under section 41.",
			Status:    continuity.RequirementApproved, Version: 3, Modality: "MUST", Actor: "data controller", Action: "retain",
			Object: "processing records annually", Citations: []string{"section 41"}, Dates: []string{"annually"},
			Topics: []string{"controller", "processing", "records", "retain"}, Applicability: continuity.ApplicabilityApplicable,
			Coverage: continuity.RequirementCoverage{RequirementID: "req-1", Applicable: true, ControlImplemented: true, EvidenceSupported: true, Complete: true},
		}},
	}
	return candidate, program
}

package documentcoverage

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestMatchCandidateRejectsCrossJurisdictionKeywordOverlap(t *testing.T) {
	candidate := Candidate{
		ID: "candidate-fed", Fingerprint: "fed", Eligible: true, Jurisdiction: "United States",
		Statement: "The bank must maintain cybersecurity controls.", Action: "maintain", Object: "cybersecurity controls",
		Topics: []string{"bank", "cybersecurity", "controls"},
	}
	program := ProgramSnapshot{
		TenantID: "bank", ProgramID: "program-ndpa", Code: "NDPA-2023", Name: "Nigeria data protection",
		Type: "PRIVACY", Status: continuity.ProgramActive, Jurisdiction: "Nigeria",
		Requirements: []RequirementTarget{{
			ID: "req-1", Code: "NDPA-1", Title: "Maintain cybersecurity controls",
			Statement: "The controller must maintain cybersecurity controls.", Status: continuity.RequirementApproved,
			Action: "maintain", Object: "cybersecurity controls", Topics: []string{"controller", "cybersecurity", "controls"},
			Applicability: continuity.ApplicabilityApplicable,
		}},
	}
	matches := MatchCandidate(candidate, []ProgramSnapshot{program})
	if len(matches) != 0 {
		t.Fatalf("cross-jurisdiction text must not match: %#v", matches)
	}
}

func TestMatchCandidateReturnsExplainableStrongMatch(t *testing.T) {
	candidate := Candidate{
		ID: "candidate-ndpa", Fingerprint: "ndpa", Eligible: true, Jurisdiction: "NG",
		Statement: "A data controller must retain processing records annually.", Modality: "MUST",
		Actor: "data controller", Action: "retain", Object: "processing records annually",
		Citations: []string{"section 41"}, Topics: []string{"controller", "processing", "records", "retain"}, Dates: []string{"annually"},
	}
	program := ProgramSnapshot{
		TenantID: "bank", ProgramID: "program-ndpa", Code: "NDPA-2023", Name: "Nigeria data protection",
		Type: "PRIVACY", Status: continuity.ProgramActive, Jurisdiction: "Nigeria",
		Requirements: []RequirementTarget{{
			ID: "req-1", Code: "NDPA-41", Title: "Retain processing records",
			Statement: "The data controller must retain processing records annually under section 41.",
			Status:    continuity.RequirementApproved, Modality: "MUST", Actor: "data controller", Action: "retain",
			Object: "processing records annually", Citations: []string{"section 41"}, Dates: []string{"annually"},
			Topics: []string{"controller", "processing", "records", "retain"}, Applicability: continuity.ApplicabilityApplicable,
		}},
	}
	matches := MatchCandidate(candidate, []ProgramSnapshot{program})
	if len(matches) != 1 || matches[0].Score < StrongMatchThreshold {
		t.Fatalf("expected one strong match, got %#v", matches)
	}
	if len(matches[0].Components) != 4 || matches[0].Rationale == "" {
		t.Fatalf("expected visible score explanation, got %#v", matches[0])
	}
}

func TestMatchCandidateThresholdBands(t *testing.T) {
	if matchBand(StrongMatchThreshold) != MatchStrong || matchBand(PossibleMatchThreshold) != MatchPossible || matchBand(PossibleMatchThreshold-.01) != MatchWeak {
		t.Fatalf("threshold bands changed")
	}
}

package documentcoverage

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
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

func TestMatchCandidateRecognizesOfficialCARWording(t *testing.T) {
	statement := "Data Controllers and Data Processors are to rely on Articles 4.1(5) and (7) of the NDPR to file CAR with the Commission."
	parsed := documentimport.ParseObligation(statement, "REQUIREMENT_CANDIDATE")
	candidate := Candidate{
		ID: "candidate-car", Fingerprint: parsed.Fingerprint, Eligible: true, Statement: statement,
		Modality: parsed.Modality, Actor: parsed.Actor, Action: parsed.Action, Object: parsed.Object,
		Citations: parsed.Citations, Dates: parsed.Dates, Topics: parsed.Topics,
		Jurisdiction: "Nigeria", ProgramType: "PRIVACY",
	}
	targetText := "The bank must maintain the records and independent review needed for its annual Compliance Audit Return. GAID 2025, Articles 10.7 and 10.8; filing before 31 March"
	target := documentimport.ParseObligation(targetText, "REQUIREMENT_CANDIDATE")
	program := ProgramSnapshot{
		TenantID: "bank", ProgramID: "program-ndpa", Code: "NDPA-2023", Name: "Nigeria data protection",
		Type: "PRIVACY", Status: continuity.ProgramActive, Jurisdiction: "Nigeria",
		Requirements: []RequirementTarget{{
			ID: "req-car", Code: "CAR-ANNUAL", Title: "Prepare the annual compliance audit return",
			Statement:    "The bank must maintain the records and independent review needed for its annual Compliance Audit Return.",
			SourceAnchor: "GAID 2025, Articles 10.7 and 10.8; filing before 31 March", Status: continuity.RequirementApproved,
			Modality: "MUST", Actor: "Bank", Action: "Maintain", Object: "Prepare the annual compliance audit return",
			Citations: target.Citations, Dates: target.Dates, Topics: target.Topics, Applicability: continuity.ApplicabilityApplicable,
		}},
	}
	matches := MatchCandidate(candidate, []ProgramSnapshot{program})
	if len(matches) != 1 || matches[0].Score < PossibleMatchThreshold {
		t.Fatalf("official CAR wording should produce an explainable candidate match: %#v", matches)
	}

	unrelated := candidate
	unrelated.ID = "candidate-penalty"
	unrelated.Statement = "A person may be ordered to pay the standard maximum penalty."
	unrelated.Action = "pay"
	unrelated.Object = "standard maximum penalty"
	unrelated.Topics = []string{"person", "penalty", "maximum", "pay"}
	unrelated.Citations = nil
	unrelated.Dates = nil
	if matches := MatchCandidate(unrelated, []ProgramSnapshot{program}); len(matches) != 0 {
		t.Fatalf("generic enforcement text must not map to the CAR requirement: %#v", matches)
	}
}

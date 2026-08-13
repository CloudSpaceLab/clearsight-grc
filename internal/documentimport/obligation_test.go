package documentimport

import (
	"slices"
	"testing"
)

func TestParseObligation(t *testing.T) {
	got := ParseObligation("A data controller must notify the Commission within 72 hours under section 41.", "REQUIREMENT_CANDIDATE")
	if !got.Eligible || got.Modality != "MUST" || got.Actor != "data controller" || got.Action != "notify" {
		t.Fatalf("unexpected obligation: %#v", got)
	}
	if !slices.Contains(got.Citations, "section 41") || !slices.Contains(got.Dates, "within 72 hours") {
		t.Fatalf("missing source structure: %#v", got)
	}
	if got.Fingerprint == "" || len(got.Topics) == 0 {
		t.Fatalf("expected normalized fingerprint and topics: %#v", got)
	}
}

func TestParseObligationDoesNotPromoteContextToObligation(t *testing.T) {
	definition := ParseObligation("Data controller means a person who determines processing purposes.", "AUTHORITY_REFERENCE")
	risk := ParseObligation("Failure may expose the organization to a penalty.", "RISK_SIGNAL")
	if definition.Eligible || risk.Eligible {
		t.Fatalf("context must not enter the coverage denominator: definition=%#v risk=%#v", definition, risk)
	}
}

func TestParseObligationFingerprintNormalizesCaseAndWhitespace(t *testing.T) {
	first := ParseObligation("A controller MUST retain records annually.", "REQUIREMENT_CANDIDATE")
	second := ParseObligation("  a   controller must retain records annually. ", "REQUIREMENT_CANDIDATE")
	if first.Fingerprint != second.Fingerprint {
		t.Fatalf("equivalent obligations must share a fingerprint: %q != %q", first.Fingerprint, second.Fingerprint)
	}
}

func TestAnalyzeBoundedAttachesObligationAndPage(t *testing.T) {
	result := AnalyzeBounded([]Section{{
		ID: "page-7", Page: 7,
		Text: "A data controller must notify the Commission within 72 hours under section 41.",
	}}, 10)
	if len(result.Proposals) != 1 {
		t.Fatalf("expected one proposal, got %#v", result.Proposals)
	}
	if result.Proposals[0].Obligation == nil || result.Proposals[0].Anchor.Page != 7 {
		t.Fatalf("expected structured page-backed obligation: %#v", result.Proposals[0])
	}
}

func TestAnalyzeBoundedDeduplicatesNormalizedObligation(t *testing.T) {
	result := AnalyzeBounded([]Section{
		{ID: "page-1", Page: 1, Text: "A controller must retain records annually."},
		{ID: "page-2", Page: 2, Text: " a  controller MUST retain records annually. "},
	}, 10)
	if result.Total != 1 || len(result.Proposals) != 1 {
		t.Fatalf("expected one normalized obligation, total=%d proposals=%#v", result.Total, result.Proposals)
	}
}

func TestAnalyzeBoundedReconstructsWrappedPDFObligations(t *testing.T) {
	result := AnalyzeBounded([]Section{{
		ID: "page-1", Page: 1,
		Text: "WHEREAS, the Nigeria Data Protection Act 2023 preserves the\nfiling of Compliance Audit Returns, which is an\nobligation for data controllers and data processors under the regulation;\n\n1. RELIANCE ON NDPR FOR FILING OF CAR\nData Controllers and Data Processors are required to file\nCompliance Audit Returns with the Commission annually.",
	}}, 10)
	if len(result.Proposals) != 2 {
		t.Fatalf("expected two complete source statements, got %#v", result.Proposals)
	}
	for _, proposal := range result.Proposals {
		if proposal.Anchor.Page != 1 || len(proposal.Statement) < 70 {
			t.Fatalf("layout line fragment escaped into analysis: %#v", proposal)
		}
	}
	if result.Proposals[1].Statement != "Data Controllers and Data Processors are required to file Compliance Audit Returns with the Commission annually." {
		t.Fatalf("wrapped requirement was not reconstructed: %q", result.Proposals[1].Statement)
	}
}

func TestAnalyzeBoundedKeepsEnumeratedDutiesSeparate(t *testing.T) {
	result := AnalyzeBounded([]Section{{
		ID: "page-2", Page: 2,
		Text: "a) Data Protection Compliance Organizations are to facilitate\nthe filing of CAR with the Commission.\nb) Data controllers shall retain evidence of the filing for five years.",
	}}, 10)
	if len(result.Proposals) != 2 {
		t.Fatalf("expected separate enumerated duties, got %#v", result.Proposals)
	}
	if result.Proposals[0].Anchor.Quote == result.Proposals[1].Anchor.Quote {
		t.Fatalf("enumerated duties were conflated: %#v", result.Proposals)
	}
}

func TestAnalyzeBoundedDoesNotSplitDecimalArticleCitations(t *testing.T) {
	result := AnalyzeBounded([]Section{{
		ID: "page-1", Page: 1,
		Text: "Data Controllers and Data Processors are to rely on Articles 4.1(5) and (7) of the NDPR to file CAR with the Commission.",
	}}, 10)
	if len(result.Proposals) != 1 || result.Proposals[0].Statement != "Data Controllers and Data Processors are to rely on Articles 4.1(5) and (7) of the NDPR to file CAR with the Commission." {
		t.Fatalf("decimal citation was split: %#v", result.Proposals)
	}
	if result.Proposals[0].Obligation == nil || result.Proposals[0].Obligation.Modality != "MUST" {
		t.Fatalf("prescriptive 'are to' wording was not structured: %#v", result.Proposals[0])
	}
}

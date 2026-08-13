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

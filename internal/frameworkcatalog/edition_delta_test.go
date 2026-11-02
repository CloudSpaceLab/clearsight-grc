package frameworkcatalog

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPreviewEditionClauseDeltaIsDeterministicAndReferenceOnly(t *testing.T) {
	from := validTestPublication()
	to := validTestPublication()
	to.EditionCode = "2027"
	to.SourceURL = "https://example.org/publications/security-2027"
	to.SourceSHA256 = strings.Repeat("b", 64)
	to.SourcePublishedOn = "2027-10-01"
	to.Clauses = []Clause{
		{Code: "C", ParentCode: "A", Title: "Third clause", SourceAnchor: "Part 2, C", Ordinal: 3},
		{Code: "A", Title: "Access governance", SourceAnchor: "Part 1, A", Ordinal: 1},
	}
	first, err := PreviewEditionClauseDelta(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if first.FromClauseCount != 2 || first.ToClauseCount != 2 || first.UnchangedCount != 0 {
		t.Fatalf("bad population counts: %+v", first)
	}
	want := []ClauseChange{
		{Code: "A", Kind: ClauseModified, ChangedFields: []string{"title"}},
		{Code: "B", Kind: ClauseRemoved},
		{Code: "C", Kind: ClauseAdded},
	}
	if !reflect.DeepEqual(first.ClauseChanges, want) {
		t.Fatalf("unexpected version changes: %#v", first.ClauseChanges)
	}
	if !reflect.DeepEqual(first.MetadataChanges, []string{"source_url", "source_sha256", "source_published_on"}) {
		t.Fatalf("missing changed publication provenance: %#v", first.MetadataChanges)
	}
	if from.Clauses[0].Code != "B" || to.Clauses[0].Code != "C" {
		t.Fatal("comparison mutated original releases")
	}
	from.Clauses[0], from.Clauses[1] = from.Clauses[1], from.Clauses[0]
	to.Clauses[0], to.Clauses[1] = to.Clauses[1], to.Clauses[0]
	second, err := PreviewEditionClauseDelta(from, to)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("input ordering changed the review preflight: %#v / %v", second, err)
	}
}

func TestPreviewEditionClauseDeltaNoContentChangeStillRequiresEditionReview(t *testing.T) {
	from := validTestPublication()
	to := validTestPublication()
	to.EditionCode = "2027"
	result, err := PreviewEditionClauseDelta(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnchangedCount != 2 || len(result.ClauseChanges) != 0 || len(result.MetadataChanges) != 0 {
		t.Fatalf("equivalent clause populations should not invent changes: %+v", result)
	}
	if result.FromEdition == result.ToEdition {
		t.Fatal("separate edition identity was lost")
	}
}

func TestPreviewEditionClauseDeltaIncludesHierarchyAndLicensedSummaryChanges(t *testing.T) {
	from := validTestPublication()
	from.LicenseBasis = LicenseSummariesPermitted
	from.LicenseReference = "Publication permission A"
	from.Clauses[0].Summary = "Periodic access reviews"
	to := from
	to.Clauses = append([]Clause(nil), from.Clauses...)
	to.EditionCode = "2027"
	to.LicenseReference = "Publication permission B"
	to.Clauses[0].ParentCode = ""
	to.Clauses[0].Summary = "Reviewed access checks"
	to.Clauses[0].Ordinal = 4
	delta, err := PreviewEditionClauseDelta(from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []ClauseChange{{Code: "B", Kind: ClauseModified, ChangedFields: []string{"parent_code", "summary", "ordinal"}}}
	if !reflect.DeepEqual(delta.ClauseChanges, want) ||
		!reflect.DeepEqual(delta.MetadataChanges, []string{"license_reference"}) || delta.UnchangedCount != 1 {
		t.Fatalf("rights or structural changes were lost: %+v", delta)
	}
}

func TestPreviewEditionClauseDeltaRejectsInvalidOrUnrelatedReleases(t *testing.T) {
	tests := map[string]func(*Publication, *Publication){
		"same edition":               func(_, to *Publication) { to.EditionCode = "2026" },
		"different standard":         func(_, to *Publication) { to.FrameworkCode = "PRIVACY" },
		"different issuer":           func(_, to *Publication) { to.PublisherCode = "OTHER" },
		"unlicensed source summary":  func(_, to *Publication) { to.Clauses[0].Summary = "Restricted prose" },
		"invalid original hierarchy": func(from, _ *Publication) { from.Clauses[0].ParentCode = "MISSING" },
		"duplicate target clauses":   func(_, to *Publication) { to.Clauses = append(to.Clauses, to.Clauses[0]) },
	}
	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			from, to := validTestPublication(), validTestPublication()
			to.EditionCode = "2027"
			modify(&from, &to)
			if _, err := PreviewEditionClauseDelta(from, to); !errors.Is(err, ErrInvalidPublication) {
				t.Fatalf("unsafe edition comparison accepted: %v", err)
			}
		})
	}
}

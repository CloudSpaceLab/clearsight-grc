package access

import (
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeLegalEntityDataBoundaryInputDefaultsToAggregateOnlySemantics(t *testing.T) {
	input, err := NormalizeLegalEntityDataBoundaryInput(ProposeLegalEntityDataBoundaryInput{
		TenantID: " bank ", LegalEntityID: " entity-a ", ResidencyRegion: " ng ",
		DetailTransferMode:        DetailTransferAggregateOnly,
		AllowedDestinationRegions: []string{"GH"}, ActorID: " admin ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if input.TenantID != "bank" || input.LegalEntityID != "entity-a" || input.ResidencyRegion != "NG" ||
		input.ActorID != "admin" || len(input.AllowedDestinationRegions) != 0 {
		t.Fatalf("normalized aggregate-only input=%#v", input)
	}
}

func TestNormalizeLegalEntityDataBoundaryInputSortsAndDeduplicatesAllowlist(t *testing.T) {
	input, err := NormalizeLegalEntityDataBoundaryInput(ProposeLegalEntityDataBoundaryInput{
		TenantID: "bank", LegalEntityID: "entity-a", ResidencyRegion: "NG",
		DetailTransferMode:        DetailTransferAllowlist,
		AllowedDestinationRegions: []string{" eu-west ", "GH", "EU-WEST", "NG"},
		ExpectedVersion:           2, ActorID: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"EU-WEST", "GH", "NG"}
	if !reflect.DeepEqual(input.AllowedDestinationRegions, want) {
		t.Fatalf("allowlist=%#v want=%#v", input.AllowedDestinationRegions, want)
	}
}

func TestNormalizeLegalEntityDataBoundaryInputRejectsInvalidPolicy(t *testing.T) {
	tests := []ProposeLegalEntityDataBoundaryInput{
		{TenantID: "bank", LegalEntityID: "entity-a", ResidencyRegion: "", DetailTransferMode: DetailTransferAggregateOnly, ActorID: "admin"},
		{TenantID: "bank", LegalEntityID: "entity-a", ResidencyRegion: "NG", DetailTransferMode: DetailTransferAllowlist, ActorID: "admin"},
		{TenantID: "bank", LegalEntityID: "entity-a", ResidencyRegion: "NG", DetailTransferMode: "OPEN", ActorID: "admin"},
		{TenantID: "bank", LegalEntityID: "entity-a", ResidencyRegion: "NG", DetailTransferMode: DetailTransferAggregateOnly, ExpectedVersion: -1, ActorID: "admin"},
	}
	for _, input := range tests {
		if _, err := NormalizeLegalEntityDataBoundaryInput(input); !errors.Is(err, ErrAdminInvalid) {
			t.Fatalf("input=%#v error=%v", input, err)
		}
	}
}

func TestLegalEntityDetailTransferAllowedFailsClosed(t *testing.T) {
	boundary := LegalEntityDataBoundary{
		Configured: true, DetailTransferMode: DetailTransferAllowlist,
		AllowedDestinationRegions: []string{"EU-WEST", "GH"},
	}
	if !LegalEntityDetailTransferAllowed(boundary, "gh") {
		t.Fatal("configured allowlisted destination was denied")
	}
	if LegalEntityDetailTransferAllowed(boundary, "NG") {
		t.Fatal("non-allowlisted destination was allowed")
	}
	boundary.Configured = false
	if LegalEntityDetailTransferAllowed(boundary, "GH") {
		t.Fatal("unconfigured boundary allowed detail transfer")
	}
	boundary.Configured = true
	boundary.DetailTransferMode = DetailTransferAggregateOnly
	if LegalEntityDetailTransferAllowed(boundary, "GH") {
		t.Fatal("aggregate-only boundary allowed detail transfer")
	}
}

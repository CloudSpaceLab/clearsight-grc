package frameworkcatalog

import (
	"errors"
	"strings"
	"testing"
)

func validTestPublication() Publication {
	return Publication{
		PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY", EditionCode: "2026",
		Title: "Reference control standard", Jurisdiction: "GLOBAL",
		SourceURL:         "https://example.org/publications/security-2026",
		SourceSHA256:      strings.Repeat("a", 64),
		SourcePublishedOn: "2026-10-01", EffectiveOn: "2027-01-01",
		LicenseBasis: LicenseCitationsOnly,
		Clauses: []Clause{
			{Code: "B", ParentCode: "A", Title: "Review access", SourceAnchor: "Part 1, B", Ordinal: 2},
			{Code: "A", Title: "Access management", SourceAnchor: "Part 1, A", Ordinal: 1},
		},
	}
}

func TestValidatePublicationOrderIndependentDigest(t *testing.T) {
	original := validTestPublication()
	fingerprint, err := ValidatePublication(original)
	if err != nil {
		t.Fatal(err)
	}
	if len(fingerprint) != 64 {
		t.Fatalf("bad digest: %s", fingerprint)
	}
	if original.Clauses[0].Code != "B" {
		t.Fatal("validation mutated publisher-owned clause ordering")
	}
	original.Clauses[0], original.Clauses[1] = original.Clauses[1], original.Clauses[0]
	reordered, err := ValidatePublication(original)
	if err != nil || reordered != fingerprint {
		t.Fatalf("reordered release changed digest: %s != %s, %v", reordered, fingerprint, err)
	}
	original.Clauses[0].Title = "Changed source clause title"
	changed, err := ValidatePublication(original)
	if err != nil || changed == fingerprint {
		t.Fatal("changed content retained fingerprint")
	}
}

func TestValidatePublicationRejectsUnlicensedContent(t *testing.T) {
	value := validTestPublication()
	value.Clauses[0].Summary = "Unlicensed normative text"
	if _, err := ValidatePublication(value); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("unlicensed summary accepted: %v", err)
	}
	value.LicenseBasis = LicenseSummariesPermitted
	if _, err := ValidatePublication(value); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("unattributed license accepted: %v", err)
	}
	value.LicenseReference = "License grant REF-2026-01"
	if _, err := ValidatePublication(value); err != nil {
		t.Fatalf("attributed permitted summary rejected: %v", err)
	}
}

func TestValidatePublicationRejectsBadIdentitySourceAndDates(t *testing.T) {
	cases := map[string]func(*Publication){
		"lowercase framework":         func(v *Publication) { v.FrameworkCode = "security" },
		"invalid issuer":              func(v *Publication) { v.PublisherCode = "AB CD" },
		"invalid source hash":         func(v *Publication) { v.SourceSHA256 = "not-a-hash" },
		"source url credentials":      func(v *Publication) { v.SourceURL = "https://owner:secret@example.org/source" },
		"source url scheme":           func(v *Publication) { v.SourceURL = "http://example.org/source" },
		"source url fragment":         func(v *Publication) { v.SourceURL = "https://example.org/source#draft" },
		"source url newline":          func(v *Publication) { v.SourceURL += "\n" },
		"source date":                 func(v *Publication) { v.SourcePublishedOn = "2026-02-30" },
		"effective date":              func(v *Publication) { v.EffectiveOn = "2026/01/01" },
		"future publisher text":       func(v *Publication) { v.Title = "" },
		"unsupported rights":          func(v *Publication) { v.LicenseBasis = "PUBLIC_DOMAIN_ASSUMED" },
		"license trailing whitespace": func(v *Publication) { v.LicenseReference = "permit " },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			value := validTestPublication()
			change(&value)
			if _, err := ValidatePublication(value); !errors.Is(err, ErrInvalidPublication) {
				t.Fatalf("invalid source accepted: %v", err)
			}
		})
	}
}

func TestValidatePublicationRejectsDuplicateMissingAndCyclicClauses(t *testing.T) {
	cases := map[string]func(*Publication){
		"duplicate":             func(v *Publication) { v.Clauses = append(v.Clauses, v.Clauses[0]) },
		"parent missing":        func(v *Publication) { v.Clauses[0].ParentCode = "MISSING" },
		"self parent":           func(v *Publication) { v.Clauses[0].ParentCode = "B" },
		"cycle":                 func(v *Publication) { v.Clauses[1].ParentCode = "B" },
		"blank source anchor":   func(v *Publication) { v.Clauses[0].SourceAnchor = "" },
		"invalid ordinal":       func(v *Publication) { v.Clauses[0].Ordinal = -1 },
		"duplicate code padded": func(v *Publication) { v.Clauses[1].Code = " A" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			value := validTestPublication()
			change(&value)
			if _, err := ValidatePublication(value); !errors.Is(err, ErrInvalidPublication) {
				t.Fatalf("invalid clause tree accepted: %v", err)
			}
		})
	}
}

func TestValidatePublicationDepthAndBounds(t *testing.T) {
	value := validTestPublication()
	value.Clauses = make([]Clause, maxClausesPerEdition+1)
	if _, err := ValidatePublication(value); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("unbounded input accepted: %v", err)
	}
	value.Clauses = []Clause{}
	if _, err := ValidatePublication(value); err != nil {
		t.Fatalf("metadata-only edition rejected: %v", err)
	}
}

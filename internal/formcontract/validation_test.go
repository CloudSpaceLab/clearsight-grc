package formcontract

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestNormalizeRejectsConstraintForWrongType(t *testing.T) {
	_, err := Normalize(Contract{
		Sections: []Section{{ID: "company", Title: "Company"}},
		Fields: []Field{{
			ID: "website", SectionID: "company", Label: "Website", Type: TypeURL,
			Constraints: Constraints{Maximum: floatPointer(10)},
		}},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
}

func TestNormalizeKeepsLegacyTypesReadable(t *testing.T) {
	got, err := Normalize(Contract{
		Sections: []Section{{ID: "review", Title: "Review"}},
		Fields: []Field{
			{ID: "note", SectionID: "review", Label: "Note", Type: Type("text")},
			{ID: "value", SectionID: "review", Label: "Value", Type: Type("number")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields[0].Type != TypeShortText || got.Fields[1].Type != TypeDecimal {
		t.Fatalf("unexpected aliases %#v", got.Fields)
	}
}

func TestNormalizeAppliesPresentationAndSectionDefaults(t *testing.T) {
	got, err := Normalize(Contract{Fields: []Field{{ID: "contact", Label: "Security contact", Type: TypeEmail}}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Presentation.DefaultMode != PresentationAutomatic || got.Presentation.AllowModeSwitch {
		t.Fatalf("unexpected presentation %#v", got.Presentation)
	}
	if len(got.Sections) != 1 || got.Sections[0].ID != DefaultSectionID || got.Fields[0].SectionID != DefaultSectionID {
		t.Fatalf("unexpected default section %#v fields %#v", got.Sections, got.Fields)
	}
}

func TestNormalizeAcceptsEachExplicitFieldType(t *testing.T) {
	types := []Type{
		TypeShortText, TypeLongText, TypeEmail, TypeTelephone, TypeURL,
		TypeInteger, TypeDecimal, TypePercentage, TypeCurrency, TypeDate,
		TypeYesNo, TypeSingleSelect, TypeMultiSelect, TypeCheckbox,
		TypeAttestation, TypeFile, TypePhoto, TypeSignature, TypeVendorDocument,
	}
	fields := make([]Field, 0, len(types))
	for index, fieldType := range types {
		field := Field{ID: fmt.Sprintf("field_%d", index), SectionID: "review", Label: fmt.Sprintf("Field %d", index), Type: fieldType}
		if fieldType == TypeSingleSelect || fieldType == TypeMultiSelect {
			field.Options = []string{"Option A", "Option B"}
		}
		if fieldType == TypeCurrency {
			field.Constraints.Currency = "NGN"
		}
		if fieldType == TypeAttestation {
			field.Attestation = "I confirm this response is complete."
		}
		fields = append(fields, field)
	}
	got, err := Normalize(Contract{Sections: []Section{{ID: "review", Title: "Review"}}, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != len(types) {
		t.Fatalf("expected %d fields, got %d", len(types), len(got.Fields))
	}
}

func TestNormalizeRejectsDuplicateAndExcessiveFields(t *testing.T) {
	_, err := Normalize(Contract{
		Sections: []Section{{ID: "review", Title: "Review"}},
		Fields: []Field{
			{ID: "answer", SectionID: "review", Label: "First", Type: TypeShortText},
			{ID: "answer", SectionID: "review", Label: "Second", Type: TypeShortText},
		},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected duplicate field rejection, got %v", err)
	}

	fields := make([]Field, MaxFields+1)
	for index := range fields {
		fields[index] = Field{ID: fmt.Sprintf("field_%d", index), SectionID: "review", Label: fmt.Sprintf("Field %d", index), Type: TypeShortText}
	}
	_, err = Normalize(Contract{Sections: []Section{{ID: "review", Title: "Review"}}, Fields: fields})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected field limit rejection, got %v", err)
	}
}

func TestNormalizeRejectsInvalidBoundedContractDefinitions(t *testing.T) {
	tooManySections := make([]Section, MaxSections+1)
	for index := range tooManySections {
		tooManySections[index] = Section{ID: fmt.Sprintf("section_%d", index), Title: fmt.Sprintf("Section %d", index)}
	}
	maximumSelections := 3
	maximumFiles := 11
	maximumFileBytes := int64(100<<20) + 1
	minimumFiles, fewerMaximumFiles := 3, 2
	maximumTotalFileBytes := int64(500<<20) + 1
	perFileBytes, smallerTotalBytes := int64(10<<20), int64(5<<20)
	minimumLength, maximumLength := 10, 2
	precision := 7
	notANumber := math.NaN()

	tests := []struct {
		name     string
		contract Contract
	}{
		{name: "unsupported presentation", contract: Contract{Presentation: Presentation{DefaultMode: "PAGED"}, Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeShortText}}}},
		{name: "too many sections", contract: Contract{Sections: tooManySections, Fields: []Field{{ID: "answer", SectionID: "section_0", Label: "Answer", Type: TypeShortText}}}},
		{name: "unknown section", contract: Contract{Sections: []Section{{ID: "known", Title: "Known"}}, Fields: []Field{{ID: "answer", SectionID: "missing", Label: "Answer", Type: TypeShortText}}}},
		{name: "reversed text limits", contract: Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeShortText, Constraints: Constraints{MinLength: &minimumLength, MaxLength: &maximumLength}}}}},
		{name: "non-finite numeric limit", contract: Contract{Fields: []Field{{ID: "amount", Label: "Amount", Type: TypeDecimal, Constraints: Constraints{Minimum: &notANumber}}}}},
		{name: "excessive decimal precision", contract: Contract{Fields: []Field{{ID: "amount", Label: "Amount", Type: TypeDecimal, Constraints: Constraints{DecimalPrecision: &precision}}}}},
		{name: "unsupported currency", contract: Contract{Fields: []Field{{ID: "amount", Label: "Amount", Type: TypeCurrency, Constraints: Constraints{Currency: "CAD"}}}}},
		{name: "invalid date range", contract: Contract{Fields: []Field{{ID: "date", Label: "Date", Type: TypeDate, Constraints: Constraints{MinDate: "2026-12-31", MaxDate: "2026-01-01"}}}}},
		{name: "excessive selections", contract: Contract{Fields: []Field{{ID: "regions", Label: "Regions", Type: TypeMultiSelect, Options: []string{"A", "B"}, Constraints: Constraints{MaxSelections: &maximumSelections}}}}},
		{name: "excessive file count", contract: Contract{Fields: []Field{{ID: "files", Label: "Files", Type: TypeFile, Constraints: Constraints{MaxFiles: &maximumFiles}}}}},
		{name: "reversed file count", contract: Contract{Fields: []Field{{ID: "files", Label: "Files", Type: TypeFile, Constraints: Constraints{MinFiles: &minimumFiles, MaxFiles: &fewerMaximumFiles}}}}},
		{name: "excessive file size", contract: Contract{Fields: []Field{{ID: "files", Label: "Files", Type: TypeFile, Constraints: Constraints{MaxFileBytes: &maximumFileBytes}}}}},
		{name: "excessive total file size", contract: Contract{Fields: []Field{{ID: "files", Label: "Files", Type: TypeFile, Constraints: Constraints{MaxTotalFileBytes: &maximumTotalFileBytes}}}}},
		{name: "total smaller than per file", contract: Contract{Fields: []Field{{ID: "files", Label: "Files", Type: TypeFile, Constraints: Constraints{MaxFileBytes: &perFileBytes, MaxTotalFileBytes: &smallerTotalBytes}}}}},
		{name: "file limits on text", contract: Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeShortText, Constraints: Constraints{MinFiles: &fewerMaximumFiles}}}}},
		{name: "invalid photo format", contract: Contract{Fields: []Field{{ID: "photo", Label: "Photo", Type: TypePhoto, AcceptedFormats: []string{"application/pdf"}}}}},
		{name: "invalid scoring option", contract: Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeSingleSelect, Options: []string{"Yes", "No"}, Scoring: &Scoring{Weight: 1, AnswerScores: map[string]int{"Maybe": 100}}}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Normalize(test.contract); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid contract, got %v", err)
			}
		})
	}
}

func TestNormalizeAppliesSmartDefaultsAndCanonicalValues(t *testing.T) {
	contract, err := Normalize(Contract{
		Presentation: Presentation{DefaultMode: " wizard ", AllowModeSwitch: true},
		Sections:     []Section{{ID: " company ", Title: " Company profile "}},
		Fields: []Field{
			{ID: " decision ", SectionID: "company", Label: " Decision ", Type: TypeYesNo},
			{ID: " percentage ", SectionID: "company", Label: " Percentage ", Type: TypePercentage},
			{ID: " amount ", SectionID: "company", Label: " Amount ", Type: TypeCurrency, Constraints: Constraints{Currency: " ngn "}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if contract.Presentation.DefaultMode != PresentationWizard || !contract.Presentation.AllowModeSwitch {
		t.Fatalf("unexpected presentation %#v", contract.Presentation)
	}
	if got := contract.Fields[0].Options; len(got) != 2 || got[0] != "Yes" || got[1] != "No" {
		t.Fatalf("unexpected yes/no options %#v", got)
	}
	if contract.Fields[1].Constraints.Minimum == nil || *contract.Fields[1].Constraints.Minimum != 0 || contract.Fields[1].Constraints.Maximum == nil || *contract.Fields[1].Constraints.Maximum != 100 {
		t.Fatalf("unexpected percentage limits %#v", contract.Fields[1].Constraints)
	}
	if contract.Fields[2].Constraints.Currency != "NGN" {
		t.Fatalf("unexpected currency %q", contract.Fields[2].Constraints.Currency)
	}
}

func floatPointer(value float64) *float64 { return &value }

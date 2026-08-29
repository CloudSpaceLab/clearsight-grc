package thirdparty

import (
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestComposeAssessmentScopeIncludesConditionDependenciesAndSections(t *testing.T) {
	contract := formcontract.Contract{
		Sections: []formcontract.Section{{ID: "identity", Title: "Identity"}, {ID: "documents", Title: "Documents"}},
		Fields: []formcontract.Field{
			{ID: "has_certificate", SectionID: "documents", Label: "Do you hold a certificate?", Type: formcontract.TypeYesNo},
			{ID: "certificate", SectionID: "documents", Label: "Certificate", Type: formcontract.TypeVendorDocument, Condition: &formcontract.VisibilityCondition{FieldID: "has_certificate", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
			{ID: "legal_name", SectionID: "identity", Label: "Legal name", Type: formcontract.TypeShortText},
		},
	}

	sections, fields, err := ComposeAssessmentScope(contract, AssessmentScopeFocused, []string{"certificate"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 1 || sections[0].ID != "documents" || len(fields) != 2 || fields[0].ID != "has_certificate" || fields[1].ID != "certificate" {
		t.Fatalf("focused scope did not retain ordered dependency closure: sections=%#v fields=%#v", sections, fields)
	}
}

func TestComposeAssessmentScopeRejectsEmptyUnknownAndDuplicateFocusedSelection(t *testing.T) {
	contract := formcontract.Contract{Sections: []formcontract.Section{{ID: "identity", Title: "Identity"}}, Fields: []formcontract.Field{{ID: "legal_name", SectionID: "identity", Label: "Legal name", Type: formcontract.TypeShortText}}}
	for _, selected := range [][]string{nil, []string{"missing"}, []string{"legal_name", "legal_name"}} {
		if _, _, err := ComposeAssessmentScope(contract, AssessmentScopeFocused, selected); !errors.Is(err, ErrInvalidAssessmentScope) {
			t.Fatalf("selection %#v error = %v", selected, err)
		}
	}
}

func TestComposeAssessmentScopeRetainsFullScopeCompatibility(t *testing.T) {
	contract := formcontract.Contract{Sections: []formcontract.Section{{ID: "identity", Title: "Identity"}}, Fields: []formcontract.Field{{ID: "legal_name", SectionID: "identity", Label: "Legal name", Type: formcontract.TypeShortText}}}
	sections, fields, err := ComposeAssessmentScope(contract, "", nil)
	if err != nil || len(sections) != 1 || len(fields) != 1 {
		t.Fatalf("full compatibility = sections=%#v fields=%#v err=%v", sections, fields, err)
	}
}

func TestNormalizeAssessmentRecordScopeDefaultsLegacyFullScope(t *testing.T) {
	assessment := Assessment{}
	if err := normalizeAssessmentRecordScope(&assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.ScopeKind != AssessmentScopeFull || assessment.ScopeVersion != 1 || assessment.SelectedFieldIDs == nil || len(assessment.SelectedFieldIDs) != 0 {
		t.Fatalf("legacy full-scope assessment was not normalized: %#v", assessment)
	}

	invalid := Assessment{ScopeKind: AssessmentScopeFocused, ScopeVersion: 1}
	if err := normalizeAssessmentRecordScope(&invalid); !errors.Is(err, ErrInvalidAssessmentScope) {
		t.Fatalf("expected empty focused scope to fail, got %v", err)
	}
}

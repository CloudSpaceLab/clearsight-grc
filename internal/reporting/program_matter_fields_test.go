package reporting

import (
	"errors"
	"testing"
)

func TestProgramDatasetRejectsAMatterScope(t *testing.T) {
	definition := ReportDefinition{
		ID: testDefinitionID, TenantID: testTenantID, LegalEntityID: testEntityA,
		Code: "PROGRAM-SCOPE", Name: "Program scope", Dataset: DatasetPrograms,
		ScopeKind: ScopeMatter, ScopeRef: testEntityB, Format: FormatCSV,
		Filter: emptyReportFilter(), Status: DefinitionDraft, CurrentVersion: 1,
		MakerID: testMakerID, Version: 1,
	}
	if err := validateDefinitionForCreate(definition); err == nil {
		t.Fatal("Program dataset accepted a Matter-only scope")
	}
}

func TestFullWorkDatasetRejectsAProgramScope(t *testing.T) {
	definition := ReportDefinition{
		ID: testDefinitionID, TenantID: testTenantID, LegalEntityID: testEntityA,
		Code: "WORK-SCOPE", Name: "Work scope", Dataset: DatasetMatters,
		ScopeKind: ScopeProgram, ScopeRef: testEntityB, Format: FormatCSV,
		Filter: emptyReportFilter(), Status: DefinitionDraft, CurrentVersion: 1,
		MakerID: testMakerID, Version: 1,
	}
	if err := validateDefinitionForCreate(definition); err == nil {
		t.Fatal("full Work dataset accepted a Program-only scope")
	}
}

func TestMatterDatasetRejectsAProgramScope(t *testing.T) {
	definition := ReportDefinition{
		ID: testDefinitionID, TenantID: testTenantID, LegalEntityID: testEntityA,
		Code: "MATTER-SCOPE", Name: "Matter scope", Dataset: DatasetMatterExceptions,
		ScopeKind: ScopeProgram, ScopeRef: testEntityB, Format: FormatCSV,
		Filter: emptyReportFilter(), Status: DefinitionDraft, CurrentVersion: 1,
		MakerID: testMakerID, Version: 1,
	}
	if err := validateDefinitionForCreate(definition); err == nil {
		t.Fatal("Matter dataset accepted a Program-only scope")
	}
}

func TestProgramReportFilterRejectsAMatterField(t *testing.T) {
	_, err := NormalizeReportFilterForDataset(DatasetPrograms, &ReportFilterExpression{
		Kind: "condition", Field: ReportFieldMatterType, Operator: "is", Value: "EXCEPTION",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Program filter accepted a Matter-only field: %v", err)
	}
}

func TestMatterReportFilterRejectsAProgramField(t *testing.T) {
	_, err := NormalizeReportFilterForDataset(DatasetMatterExceptions, &ReportFilterExpression{
		Kind: "condition", Field: ReportFieldOverallState, Operator: "is", Value: "AT_RISK",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Matter filter accepted a Program-only field: %v", err)
	}
}

func TestProgramReportFilterAcceptsItsClosedVocabulary(t *testing.T) {
	expression, err := NormalizeReportFilterForDataset(DatasetPrograms, &ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldOverallState, Operator: "is", Value: "at_risk"},
			{Kind: "condition", Field: ReportFieldHasOpenMatters, Operator: "is", Value: "true"},
			{Kind: "condition", Field: ReportFieldJurisdiction, Operator: "is", Value: "NG"},
		},
	})
	if err != nil {
		t.Fatalf("Program closed vocabulary was rejected: %v", err)
	}
	if expression == nil || expression.Children[0].Value != "AT_RISK" || expression.Children[1].Value != "true" {
		t.Fatalf("Program filter was not normalized: %#v", expression)
	}
}

func TestFullWorkReportFilterAcceptsMatterVocabulary(t *testing.T) {
	expression, err := NormalizeReportFilterForDataset(DatasetMatters, &ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldMatterType, Operator: "is", Value: "exception"},
			{Kind: "condition", Field: ReportFieldPriority, Operator: "is", Value: "3"},
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "closed"},
		},
	})
	if err != nil {
		t.Fatalf("full Work closed vocabulary was rejected: %v", err)
	}
	if expression == nil || expression.Children[0].Value != "EXCEPTION" || expression.Children[1].Value != "3" || expression.Children[2].Value != "CLOSED" {
		t.Fatalf("full Work filter was not normalized: %#v", expression)
	}
}

func TestMatterReportFilterAcceptsItsClosedVocabulary(t *testing.T) {
	expression, err := NormalizeReportFilterForDataset(DatasetMatterExceptions, &ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldMatterType, Operator: "is", Value: "exception"},
			{Kind: "condition", Field: ReportFieldPriority, Operator: "is", Value: "5"},
			{Kind: "condition", Field: ReportFieldDueCondition, Operator: "is", Value: "overdue"},
			{Kind: "condition", Field: ReportFieldLatestVerificationResult, Operator: "is", Value: "fail"},
		},
	})
	if err != nil {
		t.Fatalf("Matter closed vocabulary was rejected: %v", err)
	}
	if expression == nil || expression.Children[0].Value != "EXCEPTION" || expression.Children[1].Value != "5" || expression.Children[3].Value != "FAIL" {
		t.Fatalf("Matter filter was not normalized: %#v", expression)
	}
}

func TestMatterReportFilterAcceptsEveryRecordedMatterType(t *testing.T) {
	expression, err := NormalizeReportFilterForDataset(DatasetMatterExceptions, &ReportFilterExpression{
		Kind: "condition", Field: ReportFieldMatterType, Operator: "is", Value: "vendor_review",
	})
	if err != nil {
		t.Fatalf("recorded Matter type was rejected: %v", err)
	}
	if expression == nil || expression.Value != "VENDOR_REVIEW" {
		t.Fatalf("Matter type was not normalized: %#v", expression)
	}
}

func TestProgramAndMatterVocabularyFieldsArePublishedWithTheirDataset(t *testing.T) {
	wanted := map[ReportDataset][]ReportFilterField{
		DatasetPrograms: {
			ReportFieldOverallState,
			ReportFieldHasOpenMatters,
			ReportFieldJurisdiction,
		},
		DatasetMatterExceptions: {
			ReportFieldMatterType,
			ReportFieldPriority,
			ReportFieldDueCondition,
			ReportFieldLatestVerificationResult,
		},
	}
	published := make(map[ReportDataset]map[ReportFilterField]bool)
	for _, definition := range ReportFilterFieldVocabulary {
		if published[definition.Dataset] == nil {
			published[definition.Dataset] = make(map[ReportFilterField]bool)
		}
		published[definition.Dataset][definition.Field] = true
	}
	for dataset, fields := range wanted {
		for _, field := range fields {
			if !published[dataset][field] {
				t.Errorf("field %q is not published for %q", field, dataset)
			}
		}
	}
}

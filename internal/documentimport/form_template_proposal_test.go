package documentimport

import (
	"slices"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestProposeFormTemplateMapsDOCXControlsWithoutGuessingScoring(t *testing.T) {
	checked := true
	document := Document{
		ID: "doc-1", TenantID: "tenant-1", LegalEntityID: "entity-1", SHA256: strings64("a"),
		ExtractionStatus: ExtractionExtracted, ParserVersion: "DOCX_XML_STREAM_V3", AdapterVersion: docxElementAdapterVersion,
		Elements: []ExtractedElement{
			{Kind: ElementHeading, Text: "Vendor profile", Anchor: SourceAnchor{Paragraph: "paragraph-1"}},
			{Kind: ElementFormControl, Text: "Nigeria", Anchor: SourceAnchor{Paragraph: "paragraph-2"}, Control: &FormControl{Kind: "DROPDOWN", Label: "Country", Options: []string{"Nigeria", "Ghana"}}},
			{Kind: ElementFormControl, Text: "Yes", Anchor: SourceAnchor{Paragraph: "paragraph-3"}, Control: &FormControl{Kind: "CHECKBOX", Label: "Approved", Checked: &checked}},
		},
	}

	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Contract.ScoringMode != formcontract.ScoringNone || len(proposal.Contract.Fields) != 2 {
		t.Fatalf("unexpected proposed contract: %#v", proposal.Contract)
	}
	if proposal.Contract.Fields[0].Type != formcontract.TypeSingleSelect || !slices.Equal(proposal.Contract.Fields[0].Options, []string{"Nigeria", "Ghana"}) {
		t.Fatalf("DOCX dropdown options were not preserved: %#v", proposal.Contract.Fields[0])
	}
	if proposal.Contract.Fields[1].Type != formcontract.TypeYesNo {
		t.Fatalf("checkbox should map to yes/no: %#v", proposal.Contract.Fields[1])
	}
	for _, field := range proposal.Contract.Fields {
		if field.Required || field.Scoring != nil {
			t.Fatalf("proposal guessed requiredness or scoring: %#v", field)
		}
	}
	if !containsUnresolved(proposal.UnresolvedItems, "REQUIREDNESS_UNKNOWN") {
		t.Fatalf("requiredness ambiguity must be surfaced: %#v", proposal.UnresolvedItems)
	}
}

func TestProposeFormTemplateUsesStableFieldKeysAndSourceAnchors(t *testing.T) {
	document := Document{
		ID: "doc-1", SHA256: strings64("b"), ExtractionStatus: ExtractionExtracted,
		Elements: []ExtractedElement{{Kind: ElementFormControl, Anchor: SourceAnchor{Paragraph: "paragraph-7", Table: "table-2", Cell: "r1c3"}, Control: &FormControl{Kind: "TEXT", Label: "Legal name"}}},
	}

	first, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(first.FieldChanges) != 1 || first.FieldChanges[0].ID != second.FieldChanges[0].ID || first.Contract.Fields[0].ID != second.Contract.Fields[0].ID {
		t.Fatalf("field identity is not deterministic: first=%#v second=%#v", first, second)
	}
	anchor := first.FieldChanges[0].Anchor
	if anchor.Paragraph != "paragraph-7" || anchor.Table != "table-2" || anchor.Cell != "r1c3" {
		t.Fatalf("source anchor was not preserved: %#v", anchor)
	}
}

func TestProposeFormTemplateInfersTabularFieldTypesButNotChoices(t *testing.T) {
	document := Document{
		ID: "doc-2", SHA256: strings64("c"), ExtractionStatus: ExtractionExtracted,
		Tabular: &TabularMetadata{Format: TabularXLSX, ParserVersion: TabularParserVersion, Resources: []TabularResource{{Name: "Vendors", RowsTotal: 25, Fields: []TabularField{
			{Name: "Name", NativeType: "string"}, {Name: "Employees", NativeType: "integer"}, {Name: "Renewal Date", NativeType: "date", Nullable: true},
		}}},
	}

	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Contract.Fields) != 3 {
		t.Fatalf("unexpected fields: %#v", proposal.Contract.Fields)
	}
	want := []formcontract.Type{formcontract.TypeShortText, formcontract.TypeInteger, formcontract.TypeDate}
	for index, field := range proposal.Contract.Fields {
		if field.Type != want[index] || len(field.Options) != 0 || field.Scoring != nil {
			t.Fatalf("unsafe tabular inference at %d: %#v", index, field)
		}
	}
	if !containsUnresolved(proposal.UnresolvedItems, "OPTIONS_NOT_INFERRED") {
		t.Fatalf("lack of bounded option samples should remain explicit: %#v", proposal.UnresolvedItems)
	}
}

func TestProposeFormTemplateRejectsIncompleteOrUnusableSource(t *testing.T) {
	cases := []Document{
		{ID: "pending", ExtractionStatus: ExtractionPending},
		{ID: "partial", ExtractionStatus: ExtractionPartial, Elements: []ExtractedElement{{Kind: ElementFormControl, Control: &FormControl{Kind: "TEXT", Label: "Name"}}}},
		{ID: "empty", ExtractionStatus: ExtractionExtracted},
	}
	for _, document := range cases {
		if _, err := ProposeFormTemplate(document, DefaultProposalPolicy()); err == nil {
			t.Fatalf("expected source rejection for %#v", document)
		}
	}
}

func TestProposeFormTemplateHonorsFieldBudget(t *testing.T) {
	document := Document{ID: "doc", SHA256: strings64("d"), ExtractionStatus: ExtractionExtracted, Elements: []ExtractedElement{
		{Kind: ElementFormControl, Control: &FormControl{Kind: "TEXT", Label: "One"}},
		{Kind: ElementFormControl, Control: &FormControl{Kind: "TEXT", Label: "Two"}},
	}}
	policy := DefaultProposalPolicy()
	policy.MaxFields = 1
	proposal, err := ProposeFormTemplate(document, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.Contract.Fields) != 1 || !proposal.Truncated || !containsUnresolved(proposal.UnresolvedItems, "FIELD_LIMIT_REACHED") {
		t.Fatalf("field cap was not explicit: %#v", proposal)
	}
}

func strings64(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}

func containsUnresolved(items []ProposalUnresolvedItem, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

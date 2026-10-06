package reporting

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestMatterBoardBriefDefinitionRequiresMatterScopedPDF(t *testing.T) {
	base := ReportDefinition{
		ID:             "0199f2d0-0000-7000-8000-000000000001",
		TenantID:       "0199f2d0-0000-7000-8000-000000000002",
		LegalEntityID:  "0199f2d0-0000-7000-8000-000000000003",
		Code:           "MATTER-BOARD-BRIEF",
		Name:           "Matter board brief",
		Dataset:        DatasetMatterBoardBrief,
		ScopeKind:      ScopeMatter,
		ScopeRef:       "0199f2d0-0000-7000-8000-000000000004",
		Format:         FormatPDF,
		Filter:         &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{}},
		Status:         DefinitionDraft,
		CurrentVersion: 1,
		MakerID:        "0199f2d0-0000-7000-8000-000000000005",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Version:        1,
	}
	if err := validateDefinitionForCreate(base); err != nil {
		t.Fatalf("valid board brief definition: %v", err)
	}
	notMatter := base
	notMatter.ScopeKind = ScopeLegalEntity
	notMatter.ScopeRef = ""
	if !errors.Is(validateDefinitionForCreate(notMatter), ErrInvalid) {
		t.Fatal("board brief accepted legal-entity scope")
	}
	notPDF := base
	notPDF.Format = FormatXLSX
	if !errors.Is(validateDefinitionForCreate(notPDF), ErrInvalid) {
		t.Fatal("board brief accepted non-PDF format")
	}
	otherPDF := base
	otherPDF.Dataset = DatasetMatters
	if !errors.Is(validateDefinitionForCreate(otherPDF), ErrInvalid) {
		t.Fatal("PDF accepted for non-board-brief dataset")
	}
}

func TestMatterBoardBriefPDFIsDeterministicPaginatedAndExcludesProtectedData(t *testing.T) {
	run := ReportRun{
		ID:                "0199f2d0-0000-7000-8000-000000000010",
		DefinitionVersion: 3,
		ScopeKind:         ScopeMatter,
		ScopeRef:          "0199f2d0-0000-7000-8000-000000000011",
		AsOf:              time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC),
		Dataset:           DatasetMatterBoardBrief,
		Format:            FormatPDF,
	}
	actions := make([]any, 70)
	for index := range actions {
		actions[index] = map[string]any{"title": "Action item", "status": "IN_PROGRESS", "owner": "Control owner"}
	}
	row := ReportRow{
		ID: run.ScopeRef,
		Values: map[string]any{
			"reference": "MAT-82BF", "title": "Settlement exception", "status": "ASSESSMENT", "version": int64(7),
			"priority": 4, "summary": "Review current settlement controls.", "organization_scope": "BANK / OPERATIONS",
			"owner_name": "Control Owner", "actions": actions,
			"outcomes": []any{map[string]any{
				"expected_outcome": "All settlement controls are restored.",
				"status":           "ACTIVE",
			}},
			"forms": []any{map[string]any{"title": "Evidence request", "status": "OPEN", "response_state": "PROVISIONAL"}},
		},
	}
	first, err := renderMatterBoardBriefPDF(run, row)
	if err != nil {
		t.Fatal(err)
	}
	second, err := renderMatterBoardBriefPDF(run, row)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("board brief PDF is not deterministic")
	}
	if !bytes.HasPrefix(first, []byte("%PDF-1.4")) {
		t.Fatal("board brief is not a PDF")
	}
	if bytes.Count(first, []byte("/Type /Page ")) < 2 {
		t.Fatal("long board brief did not paginate")
	}
	if !bytes.Contains(first, []byte("Directory labels: resolved when this report is rendered")) {
		t.Fatal("board brief presents mutable directory labels without their render-time basis")
	}
	if !bytes.Contains(first, []byte("Expected Outcome: All settlement controls are restored.")) ||
		!bytes.Contains(first, []byte("Status: ACTIVE")) {
		t.Fatal("board brief omitted the active outcome contract when no result exists")
	}
	for _, protected := range [][]byte{
		[]byte("recipient@example.com"),
		[]byte("comment body"),
		[]byte("raw answer"),
	} {
		if bytes.Contains(first, protected) {
			t.Fatalf("board brief exposed protected content %q", protected)
		}
	}
}

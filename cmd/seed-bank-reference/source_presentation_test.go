//go:build postgres

package main

import (
	"strings"
	"testing"
)

func TestSourceDisplayTitleUsesRealFieldsInsteadOfRowReferences(t *testing.T) {
	cases := []struct {
		name string
		group sourceRecordGroup
		record sourceRecord
		want string
	}{
		{
			name: "branch KRI",
			group: sourceRecordGroup{Title: "Branch KRI — historical"},
			record: sourceRecord{Title: "Row 8", SourceRange: "Branch KRI!A8:BA8", Fields: []sourceRecordField{
				{Label: "Directorate", Value: "Example North"}, {Label: "Region", Value: "Example I"},
				{Label: "Branch", Value: "Sample Market Branch"},
			}}, want: "Sample Market Branch",
		},
		{
			name: "head office KRI",
			group: sourceRecordGroup{Title: "Head office KRI"},
			record: sourceRecord{Title: "Record 14", Fields: []sourceRecordField{
				{Label: "RISK OWNERS", Value: "Operations"},
				{Label: "RISK METRICS", Value: "Requests handled inside tolerance"},
				{Label: "PERIOD", Value: "August"},
			}}, want: "Requests handled inside tolerance",
		},
		{
			name: "IT exception",
			group: sourceRecordGroup{Title: "IT exceptions"},
			record: sourceRecord{Title: "Sheet1 Row 2", Fields: []sourceRecordField{
				{Label: "RISK ID", Value: "072"}, {Label: "RISK DESCRIPTION", Value: "Access review gaps"},
			}}, want: "Access review gaps",
		},
		{
			name: "BIA process",
			group: sourceRecordGroup{Title: "Business impact analysis"},
			record: sourceRecord{Title: "Line 41", Fields: []sourceRecordField{
				{Label: "Business Process", Value: "Service restoration"},
				{Label: "RTO", Value: "30 minutes"},
			}}, want: "Service restoration",
		},
		{
			name: "generic row prefix with a real title",
			group: sourceRecordGroup{Title: "IT exceptions"},
			record: sourceRecord{Title: "Row 12: Review access provisioning", Fields: []sourceRecordField{
				{Label: "RISK ID", Value: "075"},
			}}, want: "Review access provisioning",
		},
		{
			name: "operational loss",
			group: sourceRecordGroup{Title: "Operational loss register", SourceFile: "LOSS DATA BASE.xlsx"},
			record: sourceRecord{Title: "Row 17", Fields: []sourceRecordField{
				{Label: "Branch", Value: "Sample Branch"},
				{Label: "TRAN_PARTICULAR", Value: "Duplicate settlement posting"},
			}}, want: "Duplicate settlement posting",
		},
		{
			name: "source-defined title preserved",
			group: sourceRecordGroup{Title: "Branch KRI"},
			record: sourceRecord{Title: "Quarterly branch liquidity review", Fields: []sourceRecordField{
				{Label: "Branch", Value: "Sample Branch"},
			}}, want: "Quarterly branch liquidity review",
		},
		{
			name: "unreadable source",
			group: sourceRecordGroup{Title: "Historical returns"},
			record: sourceRecord{Title: "Row 51", Fields: []sourceRecordField{
				{Label: "Year", Value: "2025"}, {Label: "Amount", Value: "1000"},
			}}, want: "Historical returns",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourceRecordDisplayTitle(tc.group, tc.record); got != tc.want {
				t.Fatalf("title=%q want %q", got, tc.want)
			}
		})
	}
}

func TestSourceHeaderLookupNormalizesWhitespaceWithoutInventingValues(t *testing.T) {
	record := sourceRecord{Fields: []sourceRecordField{
		{Label: "APPLICATION/ \n SERVICES\u00a0AFFECTED", Value: "Sample digital gateway", SourceCell: "C10"},
		{Label: "RISK\t DESCRIPTION", Value: "Control gaps", SourceCell: "B10"},
		{Label: "MONTH", Value: "January"},
	}}
	if got := sourceFieldValue(record, "APPLICATION / SERVICES AFFECTED"); got != "Sample digital gateway" {
		t.Fatalf("linebreak header lost: %q", got)
	}
	if got := sourceFieldValue(record, "risk description"); got != "Control gaps" {
		t.Fatalf("tab header lost: %q", got)
	}
	if got := sourceFieldValue(record, "month number"); got != "" {
		t.Fatalf("unrelated header was invented: %q", got)
	}
	ambiguous := sourceRecord{Fields: []sourceRecordField{
		{Label: "Risk  Description", Value: "One"},
		{Label: "RISK\nDESCRIPTION", Value: "Two"},
	}}
	if got := sourceFieldValue(ambiguous, "Risk Description"); got != "" {
		t.Fatalf("ambiguous column choice is unsafe: %q", got)
	}
}

func TestSourceV2PreservesEveryColumnAndBlankSourceState(t *testing.T) {
	record := sourceRecord{Title: "Row 4", Fields: []sourceRecordField{
		{Label: "Risk finding", Value: "Access review\nrequires follow-up", SourceCell: "B4"},
		{Label: "Status", Value: "Open", SourceCell: "D4"},
		{Label: "Status", Value: "  ", SourceCell: "E4"},
		{Label: "", Value: "Additional context", SourceCell: "F4"},
	}}
	text := sourceRecordTextV2(record)
	for _, phrase := range []string{
		"Risk finding: Access review\n  requires follow-up",
		"Status (D4): Open",
		"Status (E4): Not recorded in source",
		"Source value (F4): Additional context",
	} {
		if !strings.Contains(text, phrase) {
			t.Fatalf("missing structured source field %q in %q", phrase, text)
		}
	}
	if got := sourceRecordText(record); got == text {
		t.Fatal("historical V1 submitted answers must stay unchanged")
	}
}

func TestSourcePresentationVersionRequiresNewFormAndResponseIdentity(t *testing.T) {
	original := sourceRecordGroup{Key: "ops-branch-kri", Title: "Branch KRI", SourceFile: "Branch KRI .xlsx", SourceSHA256: strings.Repeat("a", 64)}
	modern := original
	modern.PresentationVersion = 2
	codeOld, keyOld := sourceFormIdentity(original, 0)
	codeNew, keyNew := sourceFormIdentity(modern, 0)
	if codeOld == codeNew || keyOld == keyNew || !strings.HasPrefix(codeNew, "SOURCE-V2-") {
		t.Fatalf("revised form reused immutable V1 identity: old=%s new=%s", keyOld, keyNew)
	}
	if sourceRecordResponseKey(original, "branch-5") == sourceRecordResponseKey(modern, "branch-5") {
		t.Fatal("revised register response must not reuse historical distribution key")
	}
	record := sourceRecord{Title: "Row 5", Fields: []sourceRecordField{{Label: "Branch", Value: "Sample Branch"}}}
	if got := sourceRecordFormTitle(original, record); got != "Row 5" {
		t.Fatalf("legacy form contract unexpectedly changed: %s", got)
	}
	if got := sourceRecordFormTitle(modern, record); got != "Sample Branch" {
		t.Fatalf("revised form still has row label: %s", got)
	}
}

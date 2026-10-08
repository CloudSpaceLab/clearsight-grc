//go:build postgres

package main

import (
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestLegacySourceMatterTitleCorrectionRespectsOriginalTitleAndTrigger(t *testing.T) {
	group := sourceRecordGroup{
		Key: "it-risk-exceptions", Title: "IT risk exception register",
		SourceSHA256: strings.Repeat("a", 64),
	}
	record := sourceRecord{
		Key: "it-risk-exceptions-row-7", Title: "Row 7", SourceRange: "Sheet1!A7:R7",
		Fields: []sourceRecordField{
			{Label: "RISK ID", Value: "070"},
			{Label: "RISK DESCRIPTION", Value: "Privileged accounts remain active"},
		},
	}
	matter := continuity.Matter{
		Title: "Row 7", TriggerType: "SOURCE_REGISTER_IMPORT",
		TriggerKey: sourceRecordPackage + ":" + record.Key,
		Version:    3,
	}
	want := "Privileged accounts remain active"
	if title, ok := legacySourceMatterTitleCorrection(group, record, matter); !ok || title != want {
		t.Fatalf("source row title correction = %q, %v; want %q", title, ok, want)
	}

	for _, tc := range []struct {
		name   string
		change func(*sourceRecord, *continuity.Matter)
	}{
		{name: "operator edited title", change: func(_ *sourceRecord, m *continuity.Matter) { m.Title = "Operator reviewed the gap" }},
		{name: "different trigger", change: func(_ *sourceRecord, m *continuity.Matter) { m.TriggerKey = "other" }},
		{name: "not a source import", change: func(_ *sourceRecord, m *continuity.Matter) { m.TriggerType = "WORKFLOW_TRIGGER" }},
		{name: "no source coordinate", change: func(s *sourceRecord, _ *continuity.Matter) { s.SourceRange = "" }},
		{name: "missing fields", change: func(s *sourceRecord, _ *continuity.Matter) { s.Fields = nil }},
		{name: "descriptive original", change: func(s *sourceRecord, m *continuity.Matter) { s.Title, m.Title = "Privileged access risk", "Privileged access risk" }},
		{name: "already repaired", change: func(_ *sourceRecord, m *continuity.Matter) { m.Title = want }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := record
			original.Fields = append([]sourceRecordField(nil), record.Fields...)
			existing := matter
			tc.change(&original, &existing)
			if title, ok := legacySourceMatterTitleCorrection(group, original, existing); ok {
				t.Fatalf("unexpected repair %q", title)
			}
		})
	}
}

func TestLegacyTitleRepairNeverSubstitutesGroupNameForUnmappableRow(t *testing.T) {
	group := sourceRecordGroup{Key: "ops-history", Title: "Historical operational records"}
	record := sourceRecord{
		Key: "ops-history-row-8", Title: "Row 8", SourceRange: "Sheet1!A8:E8",
		Fields: []sourceRecordField{{Label: "Year", Value: "2025"}, {Label: "Amount", Value: "200"}},
	}
	matter := continuity.Matter{
		Title: "Row 8", TriggerType: "SOURCE_REGISTER_IMPORT",
		TriggerKey: sourceRecordPackage + ":" + record.Key,
	}
	if title, ok := legacySourceMatterTitleCorrection(group, record, matter); ok {
		t.Fatalf("source group title cannot replace a missing descriptive business title: %q", title)
	}
}

func TestLegacyTitleCorrectionUsesExistingSourcePrefixDescription(t *testing.T) {
	group := sourceRecordGroup{Key: "ops-branch-kri", Title: "Branch KRI"}
	record := sourceRecord{
		Key: "ops-branch-kri-row-3", Title: "Row 3: Branch operations review",
		SourceRange: "Branch KRI!A3:BA3",
		Fields: []sourceRecordField{{Label: "Branch", Value: "Test Branch"}},
	}
	matter := continuity.Matter{Title: record.Title, TriggerType: "SOURCE_REGISTER_IMPORT", TriggerKey: sourceRecordPackage + ":" + record.Key}
	if title, ok := legacySourceMatterTitleCorrection(group, record, matter); !ok || title != "Branch operations review" {
		t.Fatalf("prefixed row label remains: %q, %v", title, ok)
	}
}

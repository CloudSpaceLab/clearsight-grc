//go:build postgres

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func syntheticOpsLossGroup() sourceRecordGroup {
	return sourceRecordGroup{
		Key: "ops-loss-database", SourceFile: "LOSS DATA BASE.xlsx",
		SourceSHA256: strings.Repeat("a", 64), SourceSheet: "Sheet1",
	}
}

func syntheticOpsLossRecord(index int, month string) sourceRecord {
	return sourceRecord{
		Key: fmt.Sprintf("ops-loss-%d-%s", index, month),
		SourceRange: "Sheet1!A2:R2",
		Fields: []sourceRecordField{
			{Label: "ACCT_NAME", Value: "Synthetic operations loss"},
			{Label: "LOSS GROUP", Value: "CASH SHORTAGES"},
			{Label: "Month", Value: month},
			{Label: "Amount", Value: fmt.Sprintf("%d.05", index+101)},
			{Label: "TRAN_PARTICULAR", Value: fmt.Sprintf("Synthetic loss reference %d", index)},
			{Label: "Branch", Value: "Synthetic branch"},
			{Label: "Regional Bank", Value: "Synthetic regional bank"},
			{Label: "Directorate", Value: "Synthetic directorate"},
			{Label: "DATE OF OCCURRENCE", Value: "2024-06-01"},
			{Label: "DATE OF RECOGNITION", Value: "2025-01-31"},
			{Label: "ROOT CAUSE ANALYSIS", Value: "Pending review"},
			{Label: "DATE OF RECOVERY", Value: ""},
			{Label: "RECOVERY DATA", Value: "Full recovery of the amount"},
			{Label: "CURRENCY OF LOSS", Value: "Naira"},
		},
	}
}

func TestOpsLossMonthlyViewsDeduplicateToEightCanonicalEvents(t *testing.T) {
	group := syntheticOpsLossGroup()
	seen := map[string]sourceLossValue{}
	for index := 0; index < 8; index++ {
		for month := 1; month <= 12; month++ {
			record := syntheticOpsLossRecord(index, fmt.Sprintf("%02d", month))
			value, candidate, err := sourceLossProjection(group, record)
			if err != nil || !candidate {
				t.Fatalf("loss projection row %d month %d: candidate=%v err=%v", index, month, candidate, err)
			}
			if value.AmountMinor != int64(index+101)*100+5 || value.Currency != "NGN" ||
				value.EventType != "EXECUTION_DELIVERY_PROCESS_MANAGEMENT" {
				t.Fatalf("loss amount/type changed: %#v", value)
			}
			if !strings.Contains(value.Description, "Unverified recovery note (not posted)") {
				t.Fatal("undated recovery prose must not be treated as an accounting entry")
			}
			if previous, exists := seen[value.Code]; exists && previous.Identity != value.Identity {
				t.Fatal("loss identity collided")
			}
			seen[value.Code] = value
		}
	}
	if len(seen) != 8 {
		t.Fatalf("96 monthly views produced %d canonical Loss identities, expected 8", len(seen))
	}
}

func TestOpsLossDatesAreOccurrenceAndRecognitionNotMonthlyViewDates(t *testing.T) {
	group := syntheticOpsLossGroup()
	value, candidate, err := sourceLossProjection(group, syntheticOpsLossRecord(1, "Dec"))
	if err != nil || !candidate {
		t.Fatal(err)
	}
	if !value.OccurredAt.Equal(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)) ||
		!value.DiscoveredAt.Equal(time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("source event dates changed: occurred=%s recognized=%s", value.OccurredAt, value.DiscoveredAt)
	}
	if excelDate, dateErr := opsLossDate("45658"); dateErr != nil ||
		!excelDate.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Excel serial parsing changed: %s err=%v", excelDate, dateErr)
	}
	if writtenDate, dateErr := opsLossDate("21/09/2022"); dateErr != nil ||
		!writtenDate.Equal(time.Date(2022, 9, 21, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("mixed text date parsing changed: %s err=%v", writtenDate, dateErr)
	}
}

func TestOpsLossRejectsUnsupportedCurrenciesAndImpreciseAmounts(t *testing.T) {
	for _, value := range []string{"0", "-1", "1.001", "3.999", "NaN", "1,000"} {
		if minor, err := opsLossMinor(value); err == nil {
			t.Fatalf("unsafe amount %q accepted: %d", value, minor)
		}
	}
	if minor, err := opsLossMinor("1234567.89"); err != nil || minor != 123456789 {
		t.Fatalf("minor units %d err=%v", minor, err)
	}
	group := syntheticOpsLossGroup()
	record := syntheticOpsLossRecord(1, "Jan")
	record.Fields[len(record.Fields)-1].Value = "Unknown currency"
	if _, candidate, err := sourceLossProjection(group, record); !candidate || err == nil {
		t.Fatal("unrecognized currency cannot default to NGN")
	}
	other := group
	other.SourceFile = "HEAD OFFICE KRI MONTHLY.xlsx"
	if _, candidate, err := sourceLossProjection(other, record); candidate || err != nil {
		t.Fatal("KRI source must not be interpreted as financial Loss")
	}
}

func TestOpsLossIdentityDetectsChangedSourceDigestAndIgnoresViewMonth(t *testing.T) {
	group := syntheticOpsLossGroup()
	first, _, err := sourceLossProjection(group, syntheticOpsLossRecord(1, "Jan"))
	if err != nil {
		t.Fatal(err)
	}
	group.SourceSHA256 = strings.Repeat("b", 64)
	second, _, err := sourceLossProjection(group, syntheticOpsLossRecord(1, "Dec"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Code != second.Code || first.Identity != second.Identity ||
		first.SourceSHA == second.SourceSHA {
		t.Fatal("identity must be stable across monthly views but source version changes must be detectable")
	}
}

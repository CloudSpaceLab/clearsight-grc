package itgovernance

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestIndicatorSourceAcceptanceMapsBranchAndHeadOfficeWithoutLeakingValues(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	observedAt := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	page := sourceaccess.RecordPage{
		Records: []sourceaccess.Record{
			{
				"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
				"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
				"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7000"},
				"tx_count":     {Kind: sourceaccess.ScalarNumber, Text: "1250"},
			},
			{
				"location":     {Kind: sourceaccess.ScalarString, Text: "Head Office"},
				"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
				"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "99.875"},
				"tx_count":     {Kind: sourceaccess.ScalarNumber, Text: "840000"},
			},
			{
				"location":     {Kind: sourceaccess.ScalarString, Text: "Historical unknown"},
				"period":       {Kind: sourceaccess.ScalarString, Text: "November 2025"},
				"success_rate": {Kind: sourceaccess.ScalarString, Text: "N/A"},
			},
		},
		Receipt: acceptancePageReceipt(binding, view, observedAt, 3),
	}
	receipt, err := BuildIndicatorSourceAcceptance(IndicatorSourceAcceptanceInput{
		Binding: binding, View: view, Page: page,
		BranchRef: "CAC", HeadOfficeRef: "Head Office", GeneratedAt: observedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.RecordsRead != 3 || receipt.RecordsMapped != 2 || receipt.RecordsUnmapped != 1 || receipt.DistinctOrganizations != 2 {
		t.Fatalf("unexpected acceptance counts: %#v", receipt)
	}
	if !receipt.BranchWitnessRequired || !receipt.BranchWitnessObserved || !receipt.HeadOfficeWitnessRequired || !receipt.HeadOfficeWitnessObserved {
		t.Fatalf("source witnesses were not proved: %#v", receipt)
	}
	if receipt.MeasurementRoleCounts["success_rate"] != 2 || receipt.MeasurementRoleCounts["transaction_count"] != 2 {
		t.Fatalf("measurement role counts = %#v", receipt.MeasurementRoleCounts)
	}
	if receipt.PersistedIndicator.Proved {
		t.Fatal("live source acceptance must not imply a persisted Indicator proof")
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, protected := range []string{
		"CAC", "Head Office", "Historical unknown", "98.7000", "99.875", "1250", "840000",
		binding.BindingID, binding.SourceID, view.ViewID,
	} {
		if strings.Contains(string(encoded), protected) {
			t.Fatalf("redacted receipt leaked %q: %s", protected, encoded)
		}
	}
}

func TestIndicatorSourceAcceptanceRequiresNamedWitnessesWhenRequested(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	observedAt := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	page := sourceaccess.RecordPage{
		Records: []sourceaccess.Record{{
			"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
			"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7"},
		}},
		Receipt: acceptancePageReceipt(binding, view, observedAt, 1),
	}
	_, err := BuildIndicatorSourceAcceptance(IndicatorSourceAcceptanceInput{
		Binding: binding, View: view, Page: page,
		BranchRef: "CAC", HeadOfficeRef: "Head Office", GeneratedAt: observedAt,
	})
	if !errors.Is(err, ErrAcceptanceInvalid) || !strings.Contains(err.Error(), "head-office witness") {
		t.Fatalf("missing head-office witness error = %v", err)
	}
}

func TestIndicatorSourceAcceptanceRejectsReceiptOrSchemaDrift(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	observedAt := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	base := sourceaccess.RecordPage{
		Records: []sourceaccess.Record{{
			"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
			"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7"},
		}},
		Receipt: acceptancePageReceipt(binding, view, observedAt, 1),
	}
	for name, mutate := range map[string]func(*sourceaccess.RecordPage){
		"binding revision": func(page *sourceaccess.RecordPage) { page.Receipt.BindingVersion = "99" },
		"schema fingerprint": func(page *sourceaccess.RecordPage) { page.Receipt.SchemaFingerprint = strings.Repeat("b", 64) },
		"receipt count": func(page *sourceaccess.RecordPage) { page.Receipt.Count = 2 },
	} {
		t.Run(name, func(t *testing.T) {
			page := base
			mutate(&page)
			_, err := BuildIndicatorSourceAcceptance(IndicatorSourceAcceptanceInput{
				Binding: binding, View: view, Page: page, GeneratedAt: observedAt,
			})
			if !errors.Is(err, ErrAcceptanceInvalid) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestIndicatorSourceAcceptanceProvesPersistedNativeResultAgainstExactRevision(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	observedAt := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	pageReceipt := acceptancePageReceipt(binding, view, observedAt, 1)
	page := sourceaccess.RecordPage{
		Records: []sourceaccess.Record{{
			"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
			"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7"},
		}},
		Receipt: pageReceipt,
	}
	check := &monitoring.MonitoringCheck{
		ID: "check-private-1", TenantID: binding.TenantID, ProgramID: "program-1",
		InputKind: monitoring.InputSource, BindingID: binding.BindingID, BindingVersion: binding.Version,
		Status: monitoring.LifecycleActive, IsCurrent: true, Version: 4,
	}
	storedReceipt := pageReceipt
	storedReceipt.ObservedAt = observedAt.Add(-24 * time.Hour)
	storedReceipt.Count = 1
	storedReceiptBytes, err := json.Marshal(storedReceipt)
	if err != nil {
		t.Fatal(err)
	}
	result := &monitoring.MonitoringResult{
		ID: "result-private-1", TenantID: binding.TenantID, ProgramID: check.ProgramID,
		MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "source-operation-receipt", InputReferenceVersion: 1,
		SourceReceipt: storedReceiptBytes, EvaluatedAt: observedAt.Add(-23 * time.Hour),
		Evaluation: monitoring.Evaluation{
			Band: monitoring.RiskHigh, Coverage: 1,
			Measurement: &monitoring.NativeMeasurement{
				Field: "success_rate", Unit: monitoring.MeasurementPercent, Value: "98.7",
				Condition: monitoring.MeasurementConditionBreached,
			},
		},
	}
	receipt, err := BuildIndicatorSourceAcceptance(IndicatorSourceAcceptanceInput{
		Binding: binding, View: view, Page: page, Check: check, Result: result, GeneratedAt: observedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	proof := receipt.PersistedIndicator
	if !proof.Proved || !proof.NativeMeasurement || !proof.BindingRevisionMatched || !proof.ViewRevisionMatched || !proof.SchemaFingerprintMatched {
		t.Fatalf("persisted proof = %#v", proof)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, protected := range []string{check.ID, result.ID, "98.7"} {
		if strings.Contains(string(encoded), protected) {
			t.Fatalf("persisted acceptance leaked %q: %s", protected, encoded)
		}
	}
}

func TestIndicatorSourceAcceptanceRejectsPersistedResultRevisionMismatchOrMissingNativeMeasure(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	observedAt := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	pageReceipt := acceptancePageReceipt(binding, view, observedAt, 1)
	sourceReceiptBytes, _ := json.Marshal(pageReceipt)
	check := monitoring.MonitoringCheck{
		ID: "check-1", TenantID: binding.TenantID, InputKind: monitoring.InputSource,
		BindingID: binding.BindingID, BindingVersion: binding.Version,
		Status: monitoring.LifecycleActive, IsCurrent: true, Version: 2,
	}
	result := monitoring.MonitoringResult{
		ID: "result-1", TenantID: binding.TenantID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: monitoring.InputSource, SourceReceipt: sourceReceiptBytes, EvaluatedAt: observedAt,
		Evaluation: monitoring.Evaluation{Band: monitoring.RiskModerate, Coverage: 1, Measurement: &monitoring.NativeMeasurement{Field: "success_rate", Unit: monitoring.MeasurementPercent, Value: "98.7"}},
	}
	page := sourceaccess.RecordPage{
		Records: []sourceaccess.Record{{
			"location": {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period": {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
			"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7"},
		}},
		Receipt: pageReceipt,
	}
	for name, mutate := range map[string]func(*monitoring.MonitoringCheck, *monitoring.MonitoringResult){
		"binding version": func(check *monitoring.MonitoringCheck, _ *monitoring.MonitoringResult) { check.BindingVersion++ },
		"missing native measure": func(_ *monitoring.MonitoringCheck, result *monitoring.MonitoringResult) { result.Evaluation.Measurement = nil },
		"stored schema drift": func(_ *monitoring.MonitoringCheck, result *monitoring.MonitoringResult) {
			var receipt sourceaccess.OperationReceipt
			_ = json.Unmarshal(result.SourceReceipt, &receipt)
			receipt.SchemaFingerprint = strings.Repeat("c", 64)
			result.SourceReceipt, _ = json.Marshal(receipt)
		},
	} {
		t.Run(name, func(t *testing.T) {
			checkCopy, resultCopy := check, result
			mutate(&checkCopy, &resultCopy)
			_, err := BuildIndicatorSourceAcceptance(IndicatorSourceAcceptanceInput{
				Binding: binding, View: view, Page: page, Check: &checkCopy, Result: &resultCopy, GeneratedAt: observedAt,
			})
			if !errors.Is(err, ErrAcceptanceInvalid) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func acceptancePageReceipt(binding sourceaccess.BindingRevision, view sourceaccess.ViewRevision, observedAt time.Time, count int64) sourceaccess.OperationReceipt {
	return sourceaccess.OperationReceipt{
		SourceID: binding.SourceID,
		ConnectionID: view.ConnectionID,
		ConnectionVersion: "1",
		AdapterKind: sourceaccess.AdapterPostgres,
		AdapterVersion: "postgres-v1",
		ViewID: view.ViewID,
		ViewVersion: "1",
		BindingID: binding.BindingID,
		BindingVersion: "1",
		DefinitionFingerprint: strings.Repeat("d", 64),
		SchemaFingerprint: view.SchemaFingerprint,
		Operation: sourceaccess.OperationPage,
		ObservedAt: observedAt,
		Count: count,
		Bytes: 1024,
		Completeness: sourceaccess.CompletenessComplete,
	}
}

package itgovernance

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestMetricSeriesMappingMapsBranchAndHeadOfficeWithoutInference(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	contract, err := ParseMetricSeriesBinding(binding, view)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		organization string
		success      string
		transactions string
	}{
		{name: "branch return", organization: "CAC", success: "98.7000", transactions: "1250"},
		{name: "head office return", organization: "Head Office", success: "99.875", transactions: "840000"},
	} {
		t.Run(test.name, func(t *testing.T) {
			record, mapErr := contract.MapRecord(sourceaccess.Record{
				"location":     {Kind: sourceaccess.ScalarString, Text: test.organization},
				"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
				"success_rate": {Kind: sourceaccess.ScalarNumber, Text: test.success},
				"tx_count":     {Kind: sourceaccess.ScalarNumber, Text: test.transactions},
			})
			if mapErr != nil {
				t.Fatal(mapErr)
			}
			if record.OrganizationRef != test.organization || !record.ReportingPeriod.Equal(time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)) {
				t.Fatalf("scope/period=%#v", record)
			}
			success := record.Measurements["success_rate"]
			if success.Value != test.success || success.Unit != monitoring.MeasurementPercent || success.Condition != monitoring.MeasurementConditionUnknown {
				t.Fatalf("success measurement=%#v", success)
			}
			count := record.Measurements["transaction_count"]
			if count.Value != test.transactions || count.Unit != monitoring.MeasurementCount {
				t.Fatalf("count measurement=%#v", count)
			}
		})
	}
}

func TestMetricSeriesMappingRejectsUnreviewedOrDriftedSemantics(t *testing.T) {
	baseBinding, view := metricSeriesFixture(t)
	tests := []struct {
		name   string
		mutate func(*sourceaccess.BindingRevision, *sourceaccess.ViewRevision)
	}{
		{
			name: "missing organization mapping",
			mutate: func(binding *sourceaccess.BindingRevision, _ *sourceaccess.ViewRevision) {
				var mapping map[string]any
				_ = json.Unmarshal(binding.Mapping, &mapping)
				fields := mapping["fields"].(map[string]any)
				delete(fields, "organization_ref")
				binding.Mapping, _ = json.Marshal(mapping)
			},
		},
		{
			name: "missing success unit",
			mutate: func(binding *sourceaccess.BindingRevision, _ *sourceaccess.ViewRevision) {
				var mapping map[string]any
				_ = json.Unmarshal(binding.Mapping, &mapping)
				units := mapping["units"].(map[string]any)
				delete(units, "success_rate")
				binding.Mapping, _ = json.Marshal(mapping)
			},
		},
		{
			name: "mapped field outside selected fields",
			mutate: func(binding *sourceaccess.BindingRevision, _ *sourceaccess.ViewRevision) {
				binding.SelectedFields = []string{"location", "period", "tx_count"}
			},
		},
		{
			name: "schema drift removes mapped field",
			mutate: func(_ *sourceaccess.BindingRevision, view *sourceaccess.ViewRevision) {
				view.NativeSchema = view.NativeSchema[:3]
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binding := baseBinding
			binding.SelectedFields = append([]string(nil), baseBinding.SelectedFields...)
			binding.Mapping = append(json.RawMessage(nil), baseBinding.Mapping...)
			copyView := view
			copyView.NativeSchema = append([]sourceaccess.NativeField(nil), view.NativeSchema...)
			test.mutate(&binding, &copyView)
			if _, err := ParseMetricSeriesBinding(binding, copyView); !errors.Is(err, ErrMappingInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestMetricSeriesRecordRejectsNumericTypeDriftAndInvalidPeriod(t *testing.T) {
	binding, view := metricSeriesFixture(t)
	contract, err := ParseMetricSeriesBinding(binding, view)
	if err != nil {
		t.Fatal(err)
	}
	for name, record := range map[string]sourceaccess.Record{
		"numeric type drift": {
			"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period":       {Kind: sourceaccess.ScalarTime, Text: "2025-11-01T00:00:00Z"},
			"success_rate": {Kind: sourceaccess.ScalarString, Text: "98.7"},
		},
		"invalid period": {
			"location":     {Kind: sourceaccess.ScalarString, Text: "CAC"},
			"period":       {Kind: sourceaccess.ScalarTime, Text: "November 2025"},
			"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.7"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := contract.MapRecord(record); !errors.Is(err, ErrMappingInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func metricSeriesFixture(t *testing.T) (sourceaccess.BindingRevision, sourceaccess.ViewRevision) {
	t.Helper()
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	effective := now.Add(-time.Hour)
	view := sourceaccess.ViewRevision{
		RevisionID:        "view-revision-1",
		ViewID:            "channel-view",
		TenantID:          "bank",
		SourceID:          "channel-source",
		ConnectionID:      "connection-1",
		ConnectionVersion: 1,
		Code:              "CHANNEL-METRICS",
		Name:              "Channel metrics",
		Definition:        json.RawMessage(`{"resource":"channel_metrics"}`),
		OutputKind:        sourceaccess.OutputRecords,
		StableKeys:        []string{"location", "period"},
		NativeSchema: []sourceaccess.NativeField{
			{Name: "location", NativeType: "text"},
			{Name: "period", NativeType: "timestamptz"},
			{Name: "tx_count", NativeType: "numeric"},
			{Name: "success_rate", NativeType: "numeric"},
		},
		SchemaFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RevisionLifecycle: sourceaccess.RevisionLifecycle{
			Status: sourceaccess.RevisionActive, IsCurrent: true, EffectiveFrom: &effective,
			Version: 1, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now,
		},
	}
	binding := sourceaccess.BindingRevision{
		RevisionID:     "binding-revision-1",
		BindingID:      "channel-binding",
		TenantID:       "bank",
		SourceID:       "channel-source",
		ViewID:         view.ViewID,
		ViewVersion:    1,
		Code:           "CHANNEL-METRICS",
		Name:           "Channel performance metrics",
		Purpose:        "IT_GOVERNANCE_CHANNEL_PERFORMANCE",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationPage},
		SelectedFields: []string{"location", "period", "tx_count", "success_rate"},
		KeyFields:      []string{"location", "period"},
		Limits:         sourceaccess.DefaultResourceLimits(),
		Mapping: json.RawMessage(`{
			"schema":"clearsight.it-governance.binding.v1",
			"lens":"CHANNEL_PERFORMANCE",
			"shape":"METRIC_SERIES",
			"fields":{
				"organization_ref":"location",
				"reporting_period":"period",
				"transaction_count":"tx_count",
				"success_rate":"success_rate"
			},
			"types":{
				"organization_ref":"STRING",
				"reporting_period":"TIME",
				"transaction_count":"NUMBER",
				"success_rate":"NUMBER"
			},
			"units":{
				"transaction_count":"COUNT",
				"success_rate":"PERCENT"
			}
		}`),
		ParameterSchema:     json.RawMessage(`{}`),
		OutputSchema:        json.RawMessage(`{}`),
		Completeness:        sourceaccess.CompletenessRequireFull,
		SensitivityHandling: json.RawMessage(`{}`),
		RevisionLifecycle: sourceaccess.RevisionLifecycle{
			Status: sourceaccess.RevisionActive, IsCurrent: true, EffectiveFrom: &effective,
			Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
		},
	}
	return binding, view
}

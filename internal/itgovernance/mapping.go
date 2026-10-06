package itgovernance

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const (
	MetricSeriesSchema = "clearsight.it-governance.binding.v1"
	MetricSeriesShape  = "METRIC_SERIES"
	LensChannels       = "CHANNEL_PERFORMANCE"
)

var ErrMappingInvalid = errors.New("IT governance source mapping is invalid")

type metricSeriesMappingDocument struct {
	Schema string                             `json:"schema"`
	Lens   string                             `json:"lens"`
	Shape  string                             `json:"shape"`
	Fields map[string]string                  `json:"fields"`
	Types  map[string]sourceaccess.ScalarKind `json:"types"`
	Units  map[string]string                  `json:"units"`
}

type MetricSeriesField struct {
	Role        string
	SourceField string
	Measurement monitoring.MeasurementSpec
}

type MetricSeriesContract struct {
	Lens              string
	OrganizationField string
	PeriodField       string
	Fields            []MetricSeriesField
}

type MetricSeriesRecord struct {
	OrganizationRef string
	ReportingPeriod time.Time
	Measurements    map[string]monitoring.NativeMeasurement
}

func ParseMetricSeriesBinding(binding sourceaccess.BindingRevision, view sourceaccess.ViewRevision) (MetricSeriesContract, error) {
	if _, err := binding.Contract(view); err != nil {
		return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, err)
	}
	var document metricSeriesMappingDocument
	if err := json.Unmarshal(binding.Mapping, &document); err != nil {
		return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("mapping document is not valid JSON"))
	}
	document.Schema = strings.TrimSpace(document.Schema)
	document.Lens = strings.TrimSpace(document.Lens)
	document.Shape = strings.TrimSpace(document.Shape)
	if document.Schema != MetricSeriesSchema || document.Shape != MetricSeriesShape || document.Lens != LensChannels {
		return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("unsupported schema, lens or shape"))
	}
	if len(document.Fields) == 0 || len(document.Types) == 0 {
		return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("fields and explicit logical types are required"))
	}

	selected := make(map[string]struct{}, len(binding.SelectedFields))
	for _, field := range binding.SelectedFields {
		selected[field] = struct{}{}
	}
	available := make(map[string]struct{}, len(view.NativeSchema))
	for _, field := range view.NativeSchema {
		available[field.Name] = struct{}{}
	}
	resolve := func(role string, kind sourceaccess.ScalarKind) (string, error) {
		field := strings.TrimSpace(document.Fields[role])
		if field == "" {
			return "", errors.Join(ErrMappingInvalid, fmt.Errorf("%s mapping is required", role))
		}
		if document.Types[role] != kind {
			return "", errors.Join(ErrMappingInvalid, fmt.Errorf("%s requires explicit %s type", role, kind))
		}
		if _, ok := selected[field]; !ok {
			return "", errors.Join(ErrMappingInvalid, fmt.Errorf("%s field is outside the binding selection", role))
		}
		if _, ok := available[field]; !ok {
			return "", errors.Join(ErrMappingInvalid, fmt.Errorf("%s field is outside the inspected source schema", role))
		}
		return field, nil
	}

	organizationField, err := resolve("organization_ref", sourceaccess.ScalarString)
	if err != nil {
		return MetricSeriesContract{}, err
	}
	periodField, err := resolve("reporting_period", sourceaccess.ScalarTime)
	if err != nil {
		return MetricSeriesContract{}, err
	}

	roleSpecs := []struct {
		role         string
		unit         monitoring.MeasurementUnit
		durationUnit monitoring.MeasurementDurationUnit
		unitCode     string
	}{
		{role: "transaction_count", unit: monitoring.MeasurementCount, unitCode: "COUNT"},
		{role: "failed_transactions", unit: monitoring.MeasurementCount, unitCode: "COUNT"},
		{role: "success_rate", unit: monitoring.MeasurementPercent, unitCode: "PERCENT"},
		{role: "downtime_minutes", unit: monitoring.MeasurementDuration, durationUnit: monitoring.DurationMinutes, unitCode: "MINUTES"},
	}

	fields := make([]MetricSeriesField, 0, len(roleSpecs))
	for _, candidate := range roleSpecs {
		sourceField := strings.TrimSpace(document.Fields[candidate.role])
		if sourceField == "" {
			continue
		}
		if document.Types[candidate.role] != sourceaccess.ScalarNumber {
			return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s requires explicit NUMBER type", candidate.role))
		}
		if strings.TrimSpace(document.Units[candidate.role]) != candidate.unitCode {
			return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s requires explicit %s unit", candidate.role, candidate.unitCode))
		}
		if _, ok := selected[sourceField]; !ok {
			return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s field is outside the binding selection", candidate.role))
		}
		if _, ok := available[sourceField]; !ok {
			return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s field is outside the inspected source schema", candidate.role))
		}
		fields = append(fields, MetricSeriesField{
			Role:        candidate.role,
			SourceField: sourceField,
			Measurement: monitoring.MeasurementSpec{
				Field:        sourceField,
				Label:        metricRoleLabel(candidate.role),
				Unit:         candidate.unit,
				DurationUnit: candidate.durationUnit,
			},
		})
	}
	if len(fields) == 0 {
		return MetricSeriesContract{}, errors.Join(ErrMappingInvalid, fmt.Errorf("at least one supported native measurement is required"))
	}
	return MetricSeriesContract{
		Lens:              document.Lens,
		OrganizationField: organizationField,
		PeriodField:       periodField,
		Fields:            fields,
	}, nil
}

func (contract MetricSeriesContract) MapRecord(record sourceaccess.Record) (MetricSeriesRecord, error) {
	organization, ok := record[contract.OrganizationField]
	if !ok || organization.Kind != sourceaccess.ScalarString || strings.TrimSpace(organization.Text) == "" {
		return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("organization reference is missing or invalid"))
	}
	period, ok := record[contract.PeriodField]
	if !ok || period.Kind != sourceaccess.ScalarTime {
		return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("reporting period is missing or invalid"))
	}
	reportingPeriod, err := time.Parse(time.RFC3339, strings.TrimSpace(period.Text))
	if err != nil {
		return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("reporting period must be RFC3339"))
	}
	result := MetricSeriesRecord{
		OrganizationRef: strings.TrimSpace(organization.Text),
		ReportingPeriod: reportingPeriod.UTC(),
		Measurements:    make(map[string]monitoring.NativeMeasurement, len(contract.Fields)),
	}
	for _, field := range contract.Fields {
		scalar, exists := record[field.SourceField]
		if !exists || scalar.Kind == sourceaccess.ScalarNull {
			continue
		}
		if scalar.Kind != sourceaccess.ScalarNumber {
			return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s must remain numeric", field.Role))
		}
		measurement, buildErr := monitoring.NativeMeasurementFromExactValue(field.Measurement, scalar.Text)
		if buildErr != nil {
			return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("%s: %w", field.Role, buildErr))
		}
		result.Measurements[field.Role] = *measurement
	}
	if len(result.Measurements) == 0 {
		return MetricSeriesRecord{}, errors.Join(ErrMappingInvalid, fmt.Errorf("record has no mapped native measurements"))
	}
	return result, nil
}

func metricRoleLabel(role string) string {
	switch role {
	case "transaction_count":
		return "Transactions"
	case "failed_transactions":
		return "Failed transactions"
	case "success_rate":
		return "Success rate"
	case "downtime_minutes":
		return "Downtime"
	default:
		return role
	}
}

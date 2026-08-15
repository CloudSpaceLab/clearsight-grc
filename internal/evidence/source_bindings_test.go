package evidence

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type fakeBindingReader struct {
	revisions map[string]sourceaccess.BindingRevision
	preview   func(string, int64) (sourceaccess.RecordPage, error)
	lookup    func(string, int64, sourceaccess.LookupRequest) (sourceaccess.LookupResult, error)
}

func (f fakeBindingReader) Binding(_ context.Context, tenant, id string, version int64) (sourceaccess.BindingRevision, error) {
	value, ok := f.revisions[id+":"+strconv.FormatInt(version, 10)]
	if !ok || value.TenantID != tenant {
		return sourceaccess.BindingRevision{}, sourceaccess.ErrCatalogNotFound
	}
	return value, nil
}

func (f fakeBindingReader) PreviewBinding(_ context.Context, _ string, id string, version int64, _ sourceaccess.PageRequest) (sourceaccess.RecordPage, error) {
	return f.preview(id, version)
}

func (f fakeBindingReader) LookupBinding(_ context.Context, _ string, id string, version int64, request sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
	return f.lookup(id, version, request)
}

func bindingRevision(id, name string, operation sourceaccess.Operation, fields ...string) sourceaccess.BindingRevision {
	return sourceaccess.BindingRevision{
		BindingID: id, TenantID: "tenant-a", SourceID: "source-a", Name: name,
		Operations: []sourceaccess.Operation{operation}, SelectedFields: fields,
		RequiredFreshnessMinutes: 60,
		RevisionLifecycle:        sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, IsCurrent: true, Version: 3},
	}
}

func sourceReceipt(id string, operation sourceaccess.Operation, count int64, observedAt time.Time) sourceaccess.OperationReceipt {
	return sourceaccess.OperationReceipt{
		SourceID: "source-a", BindingID: id, BindingVersion: "3", Operation: operation,
		ObservedAt: observedAt, Count: count, Completeness: sourceaccess.CompletenessComplete,
	}
}

func TestPrepareRequestBindingsResolvesAllUsesWithoutCopyingConfiguration(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	reader := fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{
			"prefill:3":  bindingRevision("prefill", "Branch register", sourceaccess.OperationLookup, "branch_name"),
			"options:3":  bindingRevision("options", "Condition taxonomy", sourceaccess.OperationPage, "label"),
			"validate:3": bindingRevision("validate", "Staff directory", sourceaccess.OperationLookup, "employee_id"),
			"evidence:3": bindingRevision("evidence", "Risk register", sourceaccess.OperationLookup, "rating", "owner"),
		},
		preview: func(id string, _ int64) (sourceaccess.RecordPage, error) {
			return sourceaccess.RecordPage{
				Records: []sourceaccess.Record{{"label": sourceaccess.StringValue("Operational")}, {"label": sourceaccess.StringValue("Unavailable")}},
				Receipt: sourceReceipt(id, sourceaccess.OperationPage, 2, now),
			}, nil
		},
		lookup: func(id string, _ int64, _ sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			var records []sourceaccess.Record
			switch id {
			case "prefill":
				records = []sourceaccess.Record{{"branch_name": sourceaccess.StringValue("Enugu Main")}}
			case "evidence":
				records = []sourceaccess.Record{{"rating": sourceaccess.StringValue("High"), "owner": sourceaccess.StringValue("Operations")}}
			}
			return sourceaccess.LookupResult{Records: records, Receipt: sourceReceipt(id, sourceaccess.OperationLookup, int64(len(records)), now)}, nil
		},
	}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	service.ConfigureSourceBindings(reader)

	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "tenant-a", SubjectType: "BRANCH", SubjectID: "branch-42",
		Title: "Confirm branch state", Purpose: "Current resilience review", WhyYou: "You manage the branch",
		Sensitivity: "INTERNAL", AudienceType: "EXTERNAL",
		Recipient:        RecipientInput{Type: RecipientExternalAudience, Audience: "manager@example.com"},
		EstimatedMinutes: 2, Deadline: now.Add(time.Hour), KnownFacts: map[string]string{"employee": "E-17"},
		Fields: []Field{
			{ID: "branch", Label: "Branch", Type: "text", Required: true, Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill, ValueField: "branch_name", LookupValue: &LookupValueReference{Source: LookupValueSubjectID}}}},
			{ID: "condition", Label: "Condition", Type: "single_select", Required: true, Options: []string{"Fallback A", "Fallback B"}, Bindings: []FieldBindingReference{{BindingID: "options", BindingVersion: 3, Mode: BindingUseOptions, ValueField: "label"}}},
			{ID: "employee_id", Label: "Employee ID", Type: "text", Required: true, Bindings: []FieldBindingReference{{BindingID: "validate", BindingVersion: 3, Mode: BindingUseValidate, ValueField: "employee_id"}}},
			{ID: "note", Label: "Observation", Type: "long_text", Bindings: []FieldBindingReference{{BindingID: "evidence", BindingVersion: 3, Mode: BindingUseEvidence, LookupValue: &LookupValueReference{Source: LookupValueSubjectID}}}},
		},
		SourceBindings: []RequestBindingReference{{BindingID: "evidence", BindingVersion: 3, LookupValue: LookupValueReference{Source: LookupValueSubjectID}}},
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if got := request.Fields[1].Options; len(got) != 2 || got[0] != "Operational" || got[1] != "Unavailable" {
		t.Fatalf("source options not retained exactly: %#v", got)
	}
	prefill, ok := matchingResolution(request.Fields[0].SourceResolutions, request.Fields[0].Bindings[0])
	value, valueOK := resolvedScalar(prefill)
	if !ok || !valueOK || prefill.State != SourceResolutionCurrent || prefill.Receipt == nil || value.Text != "Enugu Main" || len(prefill.Records) != 0 {
		t.Fatalf("prefill provenance/value mismatch: %#v", prefill)
	}
	options := request.Fields[1].SourceResolutions[0]
	if options.State != SourceResolutionCurrent || options.Receipt == nil || len(options.Records) != 0 || options.Value != nil {
		t.Fatalf("options copied source rows: %#v", options)
	}
	fieldEvidence := request.Fields[3].SourceResolutions[0]
	if fieldEvidence.State != SourceResolutionCurrent || len(fieldEvidence.Records) != 1 || fieldEvidence.Receipt == nil {
		t.Fatalf("field evidence was not retained with receipt: %#v", fieldEvidence)
	}
	if len(request.SourceBindings) != 1 || request.SourceBindings[0].Resolution == nil || request.SourceBindings[0].Resolution.State != SourceResolutionCurrent || len(request.SourceBindings[0].Resolution.Records) != 1 {
		t.Fatalf("request evidence binding was not resolved: %#v", request.SourceBindings)
	}
}

func TestAnswerProvenanceSeparatesSourceRespondentCorrectionAndValidation(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	reader := fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{
			"validate:3": bindingRevision("validate", "Staff directory", sourceaccess.OperationLookup, "employee_id"),
		},
		preview: func(string, int64) (sourceaccess.RecordPage, error) { return sourceaccess.RecordPage{}, nil },
		lookup: func(id string, _ int64, request sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			var records []sourceaccess.Record
			if id == "validate" && len(request.Values) == 1 && request.Values[0].Text == "E-17" {
				records = []sourceaccess.Record{{"employee_id": sourceaccess.StringValue("E-17")}}
			}
			return sourceaccess.LookupResult{Records: records, Receipt: sourceReceipt(id, sourceaccess.OperationLookup, int64(len(records)), now)}, nil
		},
	}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	service.ConfigureSourceBindings(reader)
	prefillValue := sourceaccess.StringValue("Enugu Main")
	prefillReceipt := sourceReceipt("prefill", sourceaccess.OperationLookup, 1, now)
	resolution := SourceResolution{
		Mode: BindingUsePrefill, BindingID: "prefill", BindingVersion: 3, BindingName: "Branch register", SourceID: "source-a",
		State: SourceResolutionCurrent, Value: &prefillValue, Receipt: &prefillReceipt,
	}
	request := Request{TenantID: "tenant-a", Fields: []Field{
		{ID: "same", Type: "text", Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill}}, SourceResolutions: []SourceResolution{resolution}},
		{ID: "changed", Type: "text", Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill}}, SourceResolutions: []SourceResolution{resolution}},
		{ID: "employee", Type: "text", Bindings: []FieldBindingReference{{BindingID: "validate", BindingVersion: 3, Mode: BindingUseValidate, ValueField: "employee_id"}}},
		{ID: "note", Type: "text"},
	}}
	provenance := service.deriveAnswerProvenance(context.Background(), request, map[string]string{
		"same": "Enugu Main", "changed": "Nsukka", "employee": "E-17", "note": "Observed locally",
	})
	if provenance["same"].Origin != AnswerSourcePrefilled {
		t.Fatalf("same value origin = %s", provenance["same"].Origin)
	}
	if corrected := provenance["changed"]; corrected.Origin != AnswerRespondentCorrected || corrected.SourceValue == nil || corrected.SourceValue.Text != "Enugu Main" {
		t.Fatalf("corrected value provenance missing: %#v", corrected)
	}
	if provenance["note"].Origin != AnswerRespondentEntered {
		t.Fatalf("entered value origin = %s", provenance["note"].Origin)
	}
	validations := provenance["employee"].Validations
	if len(validations) != 1 || validations[0].State != SourceResolutionCurrent || validations[0].Value == nil || validations[0].Value.Text != "E-17" || len(validations[0].Records) != 0 {
		t.Fatalf("validation provenance mismatch: %#v", validations)
	}
}

func TestSourceFailureAndFreshnessNeverClaimCurrent(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	reader := fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{"options:3": bindingRevision("options", "Condition taxonomy", sourceaccess.OperationPage, "label")},
		preview: func(string, int64) (sourceaccess.RecordPage, error) {
			return sourceaccess.RecordPage{}, sourceaccess.ErrConnection
		},
		lookup: func(string, int64, sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			return sourceaccess.LookupResult{}, errors.New("unexpected lookup")
		},
	}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	service.ConfigureSourceBindings(reader)
	fields, _, err := service.prepareRequestBindings(context.Background(), CreateRequestInput{TenantID: "tenant-a", Fields: []Field{{ID: "condition", Label: "Condition", Type: "single_select", Options: []string{"Fallback A", "Fallback B"}, Bindings: []FieldBindingReference{{BindingID: "options", BindingVersion: 3, Mode: BindingUseOptions, ValueField: "label"}}}}})
	if err != nil {
		t.Fatalf("prepare bindings: %v", err)
	}
	if fields[0].Options[0] != "Fallback A" || fields[0].SourceResolutions[0].State != SourceResolutionUnavailable || fields[0].SourceResolutions[0].FailureCode != "SOURCE_UNAVAILABLE" {
		t.Fatalf("fallback/provenance mismatch: %#v", fields[0])
	}
	revision := bindingRevision("prefill", "Branch register", sourceaccess.OperationLookup, "branch_name")
	if got := resolutionState(revision, sourceReceipt("prefill", sourceaccess.OperationLookup, 1, time.Time{}), now); got == SourceResolutionCurrent {
		t.Fatal("zero observed_at was treated as current")
	}
	if got := resolutionState(revision, sourceReceipt("prefill", sourceaccess.OperationLookup, 1, now.Add(-61*time.Minute)), now); got != SourceResolutionStale {
		t.Fatalf("stale receipt state = %s", got)
	}
}

func TestInvalidSourcePrefillFallsBackInsteadOfCreatingAnUnsubmittableForm(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	reader := fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{
			"prefill:3": bindingRevision("prefill", "Branch register", sourceaccess.OperationLookup, "opened_on"),
		},
		preview: func(string, int64) (sourceaccess.RecordPage, error) { return sourceaccess.RecordPage{}, nil },
		lookup: func(id string, _ int64, _ sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			return sourceaccess.LookupResult{
				Records: []sourceaccess.Record{{"opened_on": sourceaccess.StringValue("not-a-date")}},
				Receipt: sourceReceipt(id, sourceaccess.OperationLookup, 1, now),
			}, nil
		},
	}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	service.ConfigureSourceBindings(reader)
	fields, _, err := service.prepareRequestBindings(context.Background(), CreateRequestInput{
		TenantID: "tenant-a", SubjectID: "branch-1",
		Fields: []Field{{ID: "opened_on", Label: "Opened on", Type: "date", Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill, ValueField: "opened_on", LookupValue: &LookupValueReference{Source: LookupValueSubjectID}}}}},
	})
	if err != nil {
		t.Fatalf("prepare bindings: %v", err)
	}
	resolution := fields[0].SourceResolutions[0]
	if resolution.State != SourceResolutionInvalid || resolution.FailureCode != "PREFILL_VALUE_INVALID" || resolution.Value != nil {
		t.Fatalf("invalid source prefill became respondent-visible: %#v", resolution)
	}
}

func TestRequestConfigurationRequiresAnActiveExactBinding(t *testing.T) {
	revision := bindingRevision("prefill", "Branch register", sourceaccess.OperationLookup, "branch_name")
	revision.Status = sourceaccess.RevisionPaused
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.ConfigureSourceBindings(fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{"prefill:3": revision},
		preview:   func(string, int64) (sourceaccess.RecordPage, error) { return sourceaccess.RecordPage{}, nil },
		lookup: func(string, int64, sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			return sourceaccess.LookupResult{}, nil
		},
	})
	_, _, err := service.prepareRequestBindings(context.Background(), CreateRequestInput{TenantID: "tenant-a", SubjectID: "branch-1", Fields: []Field{{ID: "branch", Label: "Branch", Type: "text", Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill, ValueField: "branch_name", LookupValue: &LookupValueReference{Source: LookupValueSubjectID}}}}}})
	if err == nil {
		t.Fatal("paused binding was accepted as form configuration")
	}
}

func TestRespondentRequestHidesEvidenceValidationSelectorsAndRows(t *testing.T) {
	value := sourceaccess.StringValue("Enugu Main")
	receipt := sourceReceipt("prefill", sourceaccess.OperationLookup, 1, time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC))
	request := Request{
		SourceBindings: []RequestBindingReference{{BindingID: "request-evidence", BindingVersion: 4, LookupValue: LookupValueReference{Source: LookupValueKnownFact, Key: "account"}, Resolution: &SourceResolution{Mode: BindingUseEvidence, BindingID: "request-evidence", BindingVersion: 4, State: SourceResolutionCurrent, Records: []sourceaccess.Record{{"risk": sourceaccess.StringValue("High")}}}}},
		Fields: []Field{{
			ID: "branch", Bindings: []FieldBindingReference{
				{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill, ValueField: "branch_name", LookupValue: &LookupValueReference{Source: LookupValueSubjectID}},
				{BindingID: "validate", BindingVersion: 3, Mode: BindingUseValidate, ValueField: "employee_id"},
				{BindingID: "evidence", BindingVersion: 3, Mode: BindingUseEvidence, LookupValue: &LookupValueReference{Source: LookupValueSubjectID}},
			},
			SourceResolutions: []SourceResolution{
				{Mode: BindingUsePrefill, BindingID: "prefill", BindingVersion: 3, BindingName: "Branch register", State: SourceResolutionCurrent, Value: &value, Receipt: &receipt, Records: []sourceaccess.Record{{"secret": sourceaccess.StringValue("not-visible")}}},
				{Mode: BindingUseValidate, BindingID: "validate", BindingVersion: 3, State: SourceResolutionNotFound, FailureCode: "VALUE_MISMATCH"},
				{Mode: BindingUseEvidence, BindingID: "evidence", BindingVersion: 3, State: SourceResolutionCurrent, Records: []sourceaccess.Record{{"risk": sourceaccess.StringValue("High")}}},
			},
		}},
	}
	visible := RespondentRequest(request)
	if len(visible.SourceBindings) != 0 || len(visible.Fields) != 1 || len(visible.Fields[0].Bindings) != 1 || visible.Fields[0].Bindings[0].Mode != BindingUsePrefill {
		t.Fatalf("hidden binding rules leaked: %#v", visible)
	}
	binding := visible.Fields[0].Bindings[0]
	if binding.ValueField != "" || binding.LookupValue != nil {
		t.Fatalf("source schema/lookup selector leaked: %#v", binding)
	}
	if len(visible.Fields[0].SourceResolutions) != 1 {
		t.Fatalf("hidden resolutions leaked: %#v", visible.Fields[0].SourceResolutions)
	}
	resolution := visible.Fields[0].SourceResolutions[0]
	if resolution.Value == nil || resolution.Value.Text != "Enugu Main" || len(resolution.Records) != 0 || resolution.FailureCode != "" {
		t.Fatalf("respondent provenance was not minimized: %#v", resolution)
	}
	if len(request.SourceBindings) != 1 || len(request.Fields[0].SourceResolutions) != 3 {
		t.Fatal("redaction mutated authoritative request")
	}
}

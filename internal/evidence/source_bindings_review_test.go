package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestAnswerProvenanceRecordsExplicitlyClearedPrefill(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	sourceValue := sourceaccess.StringValue("Enugu Main")
	receipt := sourceReceipt("prefill", sourceaccess.OperationLookup, 1, now)
	request := Request{TenantID: "tenant-a", Fields: []Field{{
		ID: "branch", Type: "text",
		Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill}},
		SourceResolutions: []SourceResolution{{
			Mode: BindingUsePrefill, BindingID: "prefill", BindingVersion: 3,
			State: SourceResolutionCurrent, Value: &sourceValue, Receipt: &receipt,
		}},
	}}}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	provenance := service.deriveAnswerProvenance(context.Background(), request, formcontract.TextAnswers(map[string]string{"branch": ""}))
	cleared, ok := provenance["branch"]
	if !ok || cleared.Origin != AnswerRespondentCorrected || cleared.SourceValue == nil || cleared.SourceValue.Text != "Enugu Main" || cleared.SourceReceipt == nil {
		t.Fatalf("cleared prefill provenance = %#v", cleared)
	}
}

func TestSourceValidationUsesFieldScalarKind(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	seen := map[string]sourceaccess.ScalarKind{}
	reader := fakeBindingReader{
		revisions: map[string]sourceaccess.BindingRevision{
			"amount:3": bindingRevision("amount", "Amount register", sourceaccess.OperationLookup, "amount"),
			"date:3":   bindingRevision("date", "Effective-date register", sourceaccess.OperationLookup, "effective_on"),
		},
		preview: func(string, int64) (sourceaccess.RecordPage, error) { return sourceaccess.RecordPage{}, nil },
		lookup: func(id string, _ int64, request sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
			if len(request.Values) != 1 {
				t.Fatalf("%s lookup values = %#v", id, request.Values)
			}
			seen[id] = request.Values[0].Kind
			field := "amount"
			if id == "date" {
				field = "effective_on"
			}
			return sourceaccess.LookupResult{
				Records: []sourceaccess.Record{{field: request.Values[0]}},
				Receipt: sourceReceipt(id, sourceaccess.OperationLookup, 1, now),
			}, nil
		},
	}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	service.ConfigureSourceBindings(reader)
	request := Request{TenantID: "tenant-a", Fields: []Field{
		{ID: "amount", Type: "number", Bindings: []FieldBindingReference{{BindingID: "amount", BindingVersion: 3, Mode: BindingUseValidate, ValueField: "amount"}}},
		{ID: "effective_on", Type: "date", Bindings: []FieldBindingReference{{BindingID: "date", BindingVersion: 3, Mode: BindingUseValidate, ValueField: "effective_on"}}},
	}}
	provenance := service.deriveAnswerProvenance(context.Background(), request, formcontract.TextAnswers(map[string]string{"amount": "42.50", "effective_on": "2026-08-15"}))
	if seen["amount"] != sourceaccess.ScalarNumber || seen["date"] != sourceaccess.ScalarTime {
		t.Fatalf("validation scalar kinds = %#v", seen)
	}
	for _, fieldID := range []string{"amount", "effective_on"} {
		validations := provenance[fieldID].Validations
		if len(validations) != 1 || validations[0].State != SourceResolutionCurrent {
			t.Fatalf("%s validations = %#v", fieldID, validations)
		}
	}
}

func TestRespondentRequestMinimizesTechnicalReceipt(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	value := sourceaccess.StringValue("Enugu Main")
	receipt := sourceReceipt("prefill", sourceaccess.OperationLookup, 1, now)
	receipt.ConnectionID = "connection-private"
	receipt.ConnectionVersion = "7"
	receipt.ViewID = "view-private"
	receipt.ViewVersion = "9"
	receipt.AdapterKind = sourceaccess.AdapterPostgres
	receipt.AdapterVersion = "postgres-v1"
	receipt.DefinitionFingerprint = "definition-private"
	receipt.SchemaFingerprint = "schema-private"
	receipt.RetryIdentity = "retry-private"
	request := Request{Fields: []Field{{
		ID: "branch", Type: "text",
		Bindings: []FieldBindingReference{{BindingID: "prefill", BindingVersion: 3, Mode: BindingUsePrefill}},
		SourceResolutions: []SourceResolution{{
			Mode: BindingUsePrefill, BindingID: "prefill", BindingVersion: 3,
			State: SourceResolutionCurrent, Value: &value, Receipt: &receipt,
		}},
	}}}
	visible := RespondentRequest(request).Fields[0].SourceResolutions[0].Receipt
	if visible == nil || visible.SourceID != "source-a" || visible.BindingID != "prefill" || !visible.ObservedAt.Equal(now) {
		t.Fatalf("visible receipt lost provenance: %#v", visible)
	}
	if visible.ConnectionID != "" || visible.ViewID != "" || visible.AdapterKind != "" || visible.DefinitionFingerprint != "" || visible.SchemaFingerprint != "" || visible.RetryIdentity != "" {
		t.Fatalf("respondent receipt leaked technical metadata: %#v", visible)
	}
}

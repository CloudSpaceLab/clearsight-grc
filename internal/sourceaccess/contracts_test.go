package sourceaccess

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReusableBindingCarriesNoConnectionOrQueryCopy(t *testing.T) {
	connection := NewPostgresConnection("core-read", "core-banking", "v1", "secret://core/read")
	if err := connection.Validate(); err != nil {
		t.Fatal(err)
	}
	view, err := NewPostgresView(
		"active-accounts",
		connection.ID,
		"v3",
		"SELECT account_id,status,balance FROM active_accounts",
		"account_id",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := view.Validate(connection); err != nil {
		t.Fatal(err)
	}
	binding := Binding{
		ID:             "account-identity",
		ViewID:         view.ID,
		Version:        "v2",
		Purpose:        "account-governance",
		Operations:     []Operation{OperationAggregate, OperationLookup, OperationPage},
		SelectedFields: []string{"account_id", "status", "balance"},
		KeyFields:      []string{"account_id"},
		Limits:         DefaultResourceLimits(),
	}
	if err := binding.Validate(view); err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"SELECT account_id", "secret://", connection.ID} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("binding copied source configuration %q: %s", forbidden, encoded)
		}
	}
	if !binding.Allows(OperationAggregate) || !binding.Allows(OperationLookup) || !binding.Allows(OperationPage) {
		t.Fatalf("binding did not preserve its reusable operations: %#v", binding.Operations)
	}

	viewFingerprint, err := ViewFingerprint(view)
	if err != nil || viewFingerprint == "" {
		t.Fatalf("view fingerprint: %q %v", viewFingerprint, err)
	}
	bindingFingerprint, err := BindingFingerprint(view, binding)
	if err != nil || bindingFingerprint == "" || bindingFingerprint == viewFingerprint {
		t.Fatalf("binding fingerprint: view=%q binding=%q err=%v", viewFingerprint, bindingFingerprint, err)
	}
	reordered := binding
	reordered.Operations = []Operation{OperationPage, OperationLookup, OperationAggregate}
	reorderedFingerprint, err := BindingFingerprint(view, reordered)
	if err != nil {
		t.Fatal(err)
	}
	if reorderedFingerprint != bindingFingerprint {
		t.Fatal("operation set ordering changed the binding fingerprint")
	}

	changed := binding
	changed.SelectedFields = append([]string(nil), binding.SelectedFields...)
	changed.SelectedFields[2] = "available_balance"
	changedFingerprint, err := BindingFingerprint(view, changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFingerprint == bindingFingerprint {
		t.Fatal("binding mapping change did not change the fingerprint")
	}
}

func TestConnectionViewAndBindingRejectCrossResourceOrAmbiguousDefinitions(t *testing.T) {
	connection := NewPostgresConnection("core-read", "core-banking", "v1", "secret://core/read")
	view, err := NewPostgresView("accounts", connection.ID, "v1", "SELECT id,status FROM accounts", "id")
	if err != nil {
		t.Fatal(err)
	}
	binding := Binding{
		ID:             "accounts-lookup",
		ViewID:         view.ID,
		Version:        "v1",
		Purpose:        "form-validation",
		Operations:     []Operation{OperationLookup},
		SelectedFields: []string{"id", "status"},
		KeyFields:      []string{"id"},
		Limits:         DefaultResourceLimits(),
	}
	if err := binding.Validate(view); err != nil {
		t.Fatal(err)
	}

	otherConnection := connection
	otherConnection.ID = "other-read"
	if err := view.Validate(otherConnection); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("cross-connection view should fail, got %v", err)
	}
	otherView := view
	otherView.ID = "other-view"
	if err := binding.Validate(otherView); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("cross-view binding should fail, got %v", err)
	}

	duplicateOperation := binding
	duplicateOperation.Operations = []Operation{OperationLookup, OperationLookup}
	if err := duplicateOperation.Validate(view); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("duplicate operation should fail, got %v", err)
	}
	duplicateField := binding
	duplicateField.SelectedFields = []string{"id", "id"}
	if err := duplicateField.Validate(view); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("duplicate selected field should fail, got %v", err)
	}
	unselectedKey := binding
	unselectedKey.SelectedFields = []string{"status"}
	if err := unselectedKey.Validate(view); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("unselected key should fail, got %v", err)
	}
	nonStableKey := binding
	nonStableKey.KeyFields = []string{"status"}
	if err := nonStableKey.Validate(view); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("non-stable key should fail, got %v", err)
	}
}

func TestResourceLimitsHaveHardNonRaiseableCeilings(t *testing.T) {
	if _, err := (ResourceLimits{PageRows: -1}).Normalized(); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("negative resource budget should fail, got %v", err)
	}

	_, err := (ResourceLimits{
		PageRows:      HardMaxPageRows + 1,
		ResponseBytes: HardMaxResponseBytes + 1,
		LookupValues:  HardMaxLookupValues + 1,
		Timeout:       HardMaxOperationTimeout + time.Second,
	}).Normalized()
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("raised resource budget should fail, got %v", err)
	}

	limits, err := (ResourceLimits{}).Normalized()
	if err != nil {
		t.Fatal(err)
	}
	defaults := DefaultResourceLimits()
	if limits != defaults {
		t.Fatalf("zero limits=%+v want defaults=%+v", limits, defaults)
	}
}

func TestScalarInputsAreBoundedAndCanonical(t *testing.T) {
	accepted := []Scalar{
		{Kind: ScalarString, Text: "81111111-1111-7111-8111-111111111111"},
		{Kind: ScalarNumber, Text: "9007199254740994.125"},
		{Kind: ScalarNumber, Text: "-1.5e+12"},
		{Kind: ScalarBool, Text: "false"},
		{Kind: ScalarTime, Text: "2026-08-14T09:50:58Z"},
		{Kind: ScalarTime, Text: "2026-08-14"},
	}
	for _, value := range accepted {
		if err := value.ValidateInput(); err != nil {
			t.Fatalf("canonical scalar %#v rejected: %v", value, err)
		}
	}

	rejected := []Scalar{
		{Kind: ScalarNull},
		{Kind: ScalarString, Text: ""},
		{Kind: ScalarString, Text: "line\nbreak"},
		{Kind: ScalarNumber, Text: "01"},
		{Kind: ScalarNumber, Text: "1; SELECT 1"},
		{Kind: ScalarBool, Text: "TRUE"},
		{Kind: ScalarTime, Text: "14/08/2026"},
	}
	for _, value := range rejected {
		if err := value.ValidateInput(); !errors.Is(err, ErrDefinitionInvalid) {
			t.Fatalf("non-canonical scalar %#v should fail, got %v", value, err)
		}
	}
}

func TestFieldNamesRemainNativeButBounded(t *testing.T) {
	for _, value := range []string{"account_id", "État", "状态", "_source1", "account.id", "account-id", "Account ID", `account"id`} {
		if !ValidFieldName(value) {
			t.Fatalf("native field name %q should be accepted", value)
		}
	}
	for _, value := range []string{"", "line\nbreak", "nul\x00field", strings.Repeat("x", HardMaxIdentifierBytes+1)} {
		if ValidFieldName(value) {
			t.Fatalf("unsafe field name %q should be rejected", value)
		}
	}
}

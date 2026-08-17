package sourceaccess

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAggregateOnlyViewDoesNotRequireStableKeys(t *testing.T) {
	connection := NewPostgresConnection("ledger-read", "general-ledger", "v1", "secret://ledger/read")
	view, err := NewPostgresView("daily-balances", connection.ID, "v1", "SELECT currency,total_balance FROM daily_balances")
	if err != nil {
		t.Fatal(err)
	}
	if err := view.Validate(connection); err != nil {
		t.Fatalf("aggregate-only view without stable keys should be valid: %v", err)
	}
	aggregate := Binding{
		ID:             "daily-balance-aggregate",
		ViewID:         view.ID,
		Version:        "v1",
		Purpose:        "daily-balance-assurance",
		Operations:     []Operation{OperationAggregate},
		SelectedFields: []string{"currency", "total_balance"},
		Limits:         DefaultResourceLimits(),
	}
	if err := aggregate.Validate(view); err != nil {
		t.Fatalf("aggregate binding should not require a stable key: %v", err)
	}
	page := aggregate
	page.ID = "daily-balance-page"
	page.Operations = []Operation{OperationPage}
	if err := page.Validate(view); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("page binding without a stable key should fail, got %v", err)
	}
}

func TestLookupScalarHasASeparateRequestSizeCeiling(t *testing.T) {
	within := Scalar{Kind: ScalarString, Text: strings.Repeat("x", hardMaxLookupScalarBytes)}
	if err := within.ValidateInput(); err != nil {
		t.Fatalf("bounded lookup scalar rejected: %v", err)
	}
	over := Scalar{Kind: ScalarString, Text: strings.Repeat("x", hardMaxLookupScalarBytes+1)}
	if err := over.ValidateInput(); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("oversized lookup scalar should fail, got %v", err)
	}
}

func TestFingerprintsNormalizeEquivalentTransientDefinitions(t *testing.T) {
	viewA := View{
		ID: "accounts", ConnectionID: "core-read", Version: "v1", OutputKind: OutputRecords,
		Definition: json.RawMessage(`{ "query": "SELECT id FROM accounts", "options": {"timeout": 1, "mode": "read"} }`),
		StableKeys: nil,
	}
	viewB := viewA
	viewB.Definition = json.RawMessage(`{"options":{"mode":"read","timeout":1},"query":"SELECT id FROM accounts"}`)
	viewB.StableKeys = []string{}
	fingerprintA, err := ViewFingerprint(viewA)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintB, err := ViewFingerprint(viewB)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprintA != fingerprintB {
		t.Fatalf("equivalent view definitions produced different fingerprints: %s != %s", fingerprintA, fingerprintB)
	}

	bindingA := Binding{
		ID: "account-aggregate", ViewID: viewA.ID, Version: "v1", Purpose: "assurance",
		Operations: []Operation{OperationAggregate}, SelectedFields: []string{"id"}, KeyFields: nil,
		Limits: ResourceLimits{},
	}
	bindingB := bindingA
	bindingB.KeyFields = []string{}
	bindingB.Limits = DefaultResourceLimits()
	bindingFingerprintA, err := BindingFingerprint(viewA, bindingA)
	if err != nil {
		t.Fatal(err)
	}
	bindingFingerprintB, err := BindingFingerprint(viewB, bindingB)
	if err != nil {
		t.Fatal(err)
	}
	if bindingFingerprintA != bindingFingerprintB {
		t.Fatalf("equivalent binding defaults produced different fingerprints: %s != %s", bindingFingerprintA, bindingFingerprintB)
	}
}

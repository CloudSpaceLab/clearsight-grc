package sourceaccess

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestReferenceConnectionCannotCarryCredentialsOrExecutionCapabilities(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	base := catalogConnectionRevision(now)
	base.AdapterKind = AdapterReference
	base.AdapterVersion = ReferenceAdapterVersion
	base.Code = ReferenceConnectionCode
	base.Name = ReferenceConnectionName
	base.Definition = json.RawMessage(`{"endpoint":"https://example.invalid/source"}`)
	base.SecretRef = ""
	base.DeclaredCapabilities = nil
	base.VerifiedCapabilities = nil
	if _, err := normalizeConnectionRevision(base); err != nil {
		t.Fatalf("valid reference connection rejected: %v", err)
	}

	withSecret := base
	withSecret.SecretRef = "secret://reference"
	if _, err := normalizeConnectionRevision(withSecret); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("reference secret should fail, got %v", err)
	}
	withCapability := base
	withCapability.DeclaredCapabilities = []Capability{CapabilityLookup}
	if _, err := normalizeConnectionRevision(withCapability); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("reference capability should fail, got %v", err)
	}
}

func TestReferenceConnectionCanonicalizesEndpoint(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	value := catalogConnectionRevision(now)
	value.AdapterKind = AdapterKind("  REFERENCE  ")
	value.AdapterVersion = ReferenceAdapterVersion
	value.Code = ReferenceConnectionCode
	value.Name = ReferenceConnectionName
	value.SecretRef = ""
	value.Definition = json.RawMessage(`{"endpoint":"  https://example.invalid/source  "}`)
	value.DeclaredCapabilities = nil
	value.VerifiedCapabilities = nil

	normalized, err := normalizeConnectionRevision(value)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.AdapterKind != AdapterReference {
		t.Fatalf("adapter kind=%q want=%q", normalized.AdapterKind, AdapterReference)
	}
	if string(normalized.Definition) != `{"endpoint":"https://example.invalid/source"}` {
		t.Fatalf("reference definition was not canonicalized: %s", normalized.Definition)
	}

	value.Definition = json.RawMessage(`{"endpoint":"https://example.invalid/source\nvalue"}`)
	if _, err := normalizeConnectionRevision(value); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("reference endpoint with control characters should fail, got %v", err)
	}
}

func TestCatalogJSONObjectsRejectNullAndExpandedOversizeValues(t *testing.T) {
	if _, err := normalizeJSONObject(json.RawMessage(`null`), HardMaxDefinitionBytes, "definition"); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("JSON null should fail, got %v", err)
	}
}

func TestCurrentViewRequiresAnInspectedSchema(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	view := catalogViewRevision(now)
	view.NativeSchema = nil
	if _, err := normalizeViewRevision(view); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("current view without native schema should fail, got %v", err)
	}
}

func TestEmptyCatalogConnectionDefinitionCompilesToNoRuntimeConfiguration(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	connection.Definition = json.RawMessage(`{}`)
	contract, err := connection.Contract()
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.Definition) != 0 {
		t.Fatalf("empty catalog configuration should compile to an omitted runtime definition: %s", contract.Definition)
	}
}

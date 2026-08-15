package sourceaccess

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
)

func (c Connection) Validate() error {
	if c.TenantID != "" {
		if err := validateOpaqueID(c.TenantID, "tenant id"); err != nil {
			return err
		}
	}
	if err := validateOpaqueID(c.ID, "connection id"); err != nil {
		return err
	}
	if err := validateOpaqueID(c.SourceID, "source id"); err != nil {
		return err
	}
	if strings.TrimSpace(c.Version) == "" || c.Version != strings.TrimSpace(c.Version) || len(c.Version) > HardMaxIdentifierBytes || containsControl(c.Version) {
		return fmt.Errorf("%w: bounded connection version is required", ErrDefinitionInvalid)
	}
	adapterKind := string(c.AdapterKind)
	if strings.TrimSpace(adapterKind) == "" || adapterKind != strings.TrimSpace(adapterKind) || len(adapterKind) > hardMaxAdapterVersionLen || containsControl(adapterKind) || strings.TrimSpace(c.AdapterVersion) == "" || c.AdapterVersion != strings.TrimSpace(c.AdapterVersion) || len(c.AdapterVersion) > hardMaxAdapterVersionLen || containsControl(c.AdapterVersion) {
		return fmt.Errorf("%w: adapter kind and bounded version are required", ErrDefinitionInvalid)
	}
	if c.SecretRef != "" && (c.SecretRef != strings.TrimSpace(c.SecretRef) || len(c.SecretRef) > HardMaxIdentifierBytes || containsControl(c.SecretRef)) {
		return fmt.Errorf("%w: secret reference must be bounded when present", ErrDefinitionInvalid)
	}
	if len(c.Definition) > HardMaxDefinitionBytes {
		return ErrLimitExceeded
	}
	if len(c.Definition) > 0 && !json.Valid(c.Definition) {
		return fmt.Errorf("%w: connection definition must be valid JSON", ErrDefinitionInvalid)
	}
	return nil
}

func (v View) Validate(connection Connection) error {
	if err := connection.Validate(); err != nil {
		return err
	}
	if err := validateOpaqueID(v.ID, "view id"); err != nil {
		return err
	}
	if err := validateOpaqueID(v.ConnectionID, "connection id"); err != nil {
		return err
	}
	if v.ConnectionID != connection.ID {
		return fmt.Errorf("%w: view connection does not match the opened connection", ErrDefinitionInvalid)
	}
	if strings.TrimSpace(v.Version) == "" || v.Version != strings.TrimSpace(v.Version) || len(v.Version) > HardMaxIdentifierBytes || containsControl(v.Version) {
		return fmt.Errorf("%w: bounded view version is required", ErrDefinitionInvalid)
	}
	if v.OutputKind != OutputRecords {
		return fmt.Errorf("%w: unsupported view output kind", ErrDefinitionInvalid)
	}
	if len(v.Definition) == 0 || len(v.Definition) > HardMaxDefinitionBytes || !json.Valid(v.Definition) {
		return fmt.Errorf("%w: bounded adapter view definition is required", ErrDefinitionInvalid)
	}
	if len(v.StableKeys) > HardMaxStableKeyFields {
		return fmt.Errorf("%w: no more than %d stable keys are allowed", ErrDefinitionInvalid, HardMaxStableKeyFields)
	}
	if err := validateFields(v.StableKeys, HardMaxStableKeyFields); err != nil {
		return err
	}
	if len(v.NativeSchema) > HardMaxSchemaFields {
		return ErrLimitExceeded
	}
	if (len(v.NativeSchema) == 0) != (v.SchemaFingerprint == "") {
		return fmt.Errorf("%w: native schema and schema fingerprint must be supplied together", ErrDefinitionInvalid)
	}
	if v.SchemaFingerprint != "" && !isLowerHex(v.SchemaFingerprint, 64) {
		return fmt.Errorf("%w: schema fingerprint must be a lowercase SHA-256 value", ErrDefinitionInvalid)
	}
	if len(v.NativeSchema) > 0 {
		seen := make(map[string]struct{}, len(v.NativeSchema))
		for _, field := range v.NativeSchema {
			if !ValidFieldName(field.Name) || strings.TrimSpace(field.NativeType) == "" || field.NativeType != strings.TrimSpace(field.NativeType) || len(field.NativeType) > HardMaxIdentifierBytes || containsControl(field.NativeType) {
				return fmt.Errorf("%w: source schema contains an invalid field", ErrDefinitionInvalid)
			}
			if _, exists := seen[field.Name]; exists {
				return fmt.Errorf("%w: source schema contains duplicate fields", ErrDefinitionInvalid)
			}
			seen[field.Name] = struct{}{}
		}
		for _, key := range v.StableKeys {
			if _, exists := seen[key]; !exists {
				return fmt.Errorf("%w: stable keys must exist in the inspected native schema", ErrDefinitionInvalid)
			}
		}
	}
	return nil
}

func (b Binding) Validate(view View) error {
	if err := validateOpaqueID(b.ID, "binding id"); err != nil {
		return err
	}
	if err := validateOpaqueID(b.ViewID, "view id"); err != nil {
		return err
	}
	if b.ViewID != view.ID {
		return fmt.Errorf("%w: binding view does not match the requested view", ErrDefinitionInvalid)
	}
	if strings.TrimSpace(b.Version) == "" || b.Version != strings.TrimSpace(b.Version) || len(b.Version) > HardMaxIdentifierBytes || containsControl(b.Version) {
		return fmt.Errorf("%w: bounded binding version is required", ErrDefinitionInvalid)
	}
	if strings.TrimSpace(b.Purpose) == "" || b.Purpose != strings.TrimSpace(b.Purpose) || len(b.Purpose) > HardMaxIdentifierBytes || containsControl(b.Purpose) {
		return fmt.Errorf("%w: bounded binding purpose is required", ErrDefinitionInvalid)
	}
	if len(b.Operations) == 0 || len(b.Operations) > 5 {
		return fmt.Errorf("%w: at least one bounded operation is required", ErrDefinitionInvalid)
	}
	seenOperations := make(map[Operation]struct{}, len(b.Operations))
	for _, operation := range b.Operations {
		if !validOperation(operation) {
			return fmt.Errorf("%w: unsupported binding operation", ErrDefinitionInvalid)
		}
		if _, exists := seenOperations[operation]; exists {
			return fmt.Errorf("%w: duplicate binding operation", ErrDefinitionInvalid)
		}
		seenOperations[operation] = struct{}{}
	}
	if len(b.SelectedFields) == 0 || len(b.SelectedFields) > HardMaxSelectedFields {
		return fmt.Errorf("%w: one to %d selected fields are required", ErrDefinitionInvalid, HardMaxSelectedFields)
	}
	if err := validateFields(b.SelectedFields, HardMaxSelectedFields); err != nil {
		return err
	}
	if len(b.KeyFields) > HardMaxStableKeyFields {
		return fmt.Errorf("%w: too many key fields", ErrDefinitionInvalid)
	}
	if err := validateFields(b.KeyFields, HardMaxStableKeyFields); err != nil {
		return err
	}
	if b.Allows(OperationPage) || b.Allows(OperationLookup) {
		if len(b.KeyFields) == 0 {
			return fmt.Errorf("%w: page and lookup bindings require a key field", ErrDefinitionInvalid)
		}
		for _, key := range b.KeyFields {
			if !containsString(view.StableKeys, key) || !containsString(b.SelectedFields, key) {
				return fmt.Errorf("%w: key fields must be selected stable view keys", ErrDefinitionInvalid)
			}
		}
	}
	_, err := b.Limits.Normalized()
	return err
}

func (b Binding) Allows(operation Operation) bool {
	for _, allowed := range b.Operations {
		if allowed == operation {
			return true
		}
	}
	return false
}

func (b Binding) NormalizedLimits() (ResourceLimits, error) {
	return b.Limits.Normalized()
}

func (s Scalar) ValidateInput() error {
	if s.Kind == ScalarNull || !validScalarKind(s.Kind) || len(s.Text) == 0 || len(s.Text) > hardMaxLookupScalarBytes || strings.IndexByte(s.Text, 0) >= 0 || containsControl(s.Text) {
		return fmt.Errorf("%w: invalid lookup or cursor value", ErrDefinitionInvalid)
	}
	switch s.Kind {
	case ScalarNumber:
		if s.Text != strings.TrimSpace(s.Text) || !validNumberText(s.Text) {
			return fmt.Errorf("%w: invalid canonical number", ErrDefinitionInvalid)
		}
	case ScalarBool:
		if s.Text != "true" && s.Text != "false" {
			return fmt.Errorf("%w: invalid canonical boolean", ErrDefinitionInvalid)
		}
	case ScalarTime:
		if !validTimeText(s.Text) {
			return fmt.Errorf("%w: invalid canonical time", ErrDefinitionInvalid)
		}
	}
	return nil
}

func validNumberText(value string) bool {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var number json.Number
	if err := decoder.Decode(&number); err != nil {
		return false
	}
	var trailing any
	return decoder.Decode(&trailing) == io.EOF
}

func validTimeText(value string) bool {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil && !parsed.IsZero() {
			return true
		}
	}
	return false
}

func validateOpaqueID(value, label string) error {
	trimmed := strings.TrimSpace(value)
	if value != trimmed || trimmed == "" || len(value) > HardMaxIdentifierBytes || containsControl(value) {
		return fmt.Errorf("%w: bounded %s is required", ErrDefinitionInvalid, label)
	}
	return nil
}

func validateFields(values []string, maximum int) error {
	if len(values) > maximum {
		return ErrLimitExceeded
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !ValidFieldName(value) {
			return fmt.Errorf("%w: field names must be bounded native identifiers", ErrDefinitionInvalid)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate field name", ErrDefinitionInvalid)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func ValidFieldName(value string) bool {
	return value != "" && len(value) <= HardMaxIdentifierBytes && !containsControl(value)
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func validOperation(value Operation) bool {
	switch value {
	case OperationInspect, OperationPage, OperationLookup, OperationAggregate, OperationChanges:
		return true
	default:
		return false
	}
}

func validScalarKind(value ScalarKind) bool {
	switch value {
	case ScalarString, ScalarNumber, ScalarBool, ScalarTime:
		return true
	default:
		return false
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

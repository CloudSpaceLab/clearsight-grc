package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// LogicalType is the bounded type system used by deterministic profiling and
// condition evaluation. Unknown fields may still participate in EXISTS/MISSING
// checks but cannot be coerced into richer predicates.
type LogicalType string

const (
	TypeUnknown LogicalType = "UNKNOWN"
	TypeString  LogicalType = "STRING"
	TypeNumber  LogicalType = "NUMBER"
	TypeBool    LogicalType = "BOOL"
	TypeTime    LogicalType = "TIME"
)

type NativeField struct {
	Name       string `json:"name"`
	NativeType string `json:"native_type"`
	Nullable   bool   `json:"nullable"`
}

type Field struct {
	Name     string      `json:"name"`
	Type     LogicalType `json:"type"`
	Nullable bool        `json:"nullable"`
}

type Schema struct {
	Fields []Field `json:"fields"`
}

func NormalizeSchema(fields []NativeField) (Schema, error) {
	result := Schema{Fields: make([]Field, 0, len(fields))}
	for _, field := range fields {
		result.Fields = append(result.Fields, Field{
			Name:     strings.TrimSpace(field.Name),
			Type:     NormalizeLogicalType(field.NativeType),
			Nullable: field.Nullable,
		})
	}
	if err := result.Validate(); err != nil {
		return Schema{}, err
	}
	return result, nil
}

func NormalizeLogicalType(native string) LogicalType {
	value := strings.ToLower(strings.TrimSpace(native))
	if index := strings.IndexByte(value, '('); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	value = strings.Join(strings.Fields(value), " ")

	switch value {
	case "bool", "boolean":
		return TypeBool
	case "smallint", "int2", "integer", "int", "int4", "bigint", "int8",
		"numeric", "decimal", "real", "float4", "double precision", "float8", "float", "number":
		return TypeNumber
	case "date", "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz", "datetime":
		return TypeTime
	case "text", "varchar", "character varying", "char", "character", "citext", "uuid":
		return TypeString
	default:
		return TypeUnknown
	}
}

func (s Schema) Validate() error {
	seen := make(map[string]struct{}, len(s.Fields))
	for _, field := range s.Fields {
		if strings.TrimSpace(field.Name) == "" {
			return fmt.Errorf("schema field name is required")
		}
		if !validLogicalType(field.Type) {
			return fmt.Errorf("field %q has unsupported logical type %q", field.Name, field.Type)
		}
		if _, exists := seen[field.Name]; exists {
			return fmt.Errorf("schema contains duplicate field %q", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return nil
}

func (s Schema) Field(name string) (Field, bool) {
	for _, field := range s.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

// Fingerprint is stable across source column ordering. Reordering a SELECT
// projection should not invalidate a rule when field names/types/nullability are
// otherwise unchanged.
func (s Schema) Fingerprint() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	fields := append([]Field(nil), s.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	hash := sha256.New()
	for _, field := range fields {
		_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%t\n", field.Name, field.Type, field.Nullable)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validLogicalType(value LogicalType) bool {
	switch value {
	case TypeUnknown, TypeString, TypeNumber, TypeBool, TypeTime:
		return true
	default:
		return false
	}
}

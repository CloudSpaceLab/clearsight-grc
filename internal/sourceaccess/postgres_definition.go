package sourceaccess

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func decodePostgresView(view View) (PostgresViewDefinition, error) {
	decoder := json.NewDecoder(bytes.NewReader(view.Definition))
	decoder.DisallowUnknownFields()
	var definition PostgresViewDefinition
	if err := decoder.Decode(&definition); err != nil {
		return PostgresViewDefinition{}, fmt.Errorf("%w: PostgreSQL view definition is invalid", ErrDefinitionInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return PostgresViewDefinition{}, fmt.Errorf("%w: PostgreSQL view definition has trailing data", ErrDefinitionInvalid)
	}
	query, err := normalizePostgresQuery(definition.Query)
	if err != nil {
		return PostgresViewDefinition{}, err
	}
	definition.Query = query
	return definition, nil
}

func normalizePostgresQuery(query string) (string, error) {
	if strings.IndexByte(query, 0) >= 0 {
		return "", fmt.Errorf("%w: query contains NUL", ErrDefinitionInvalid)
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("%w: query is required", ErrDefinitionInvalid)
	}
	if len(query) > HardMaxDefinitionBytes {
		return "", ErrLimitExceeded
	}
	if strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if query == "" || strings.Contains(query, ";") {
		return "", fmt.Errorf("%w: query must contain one SELECT/WITH statement", ErrDefinitionInvalid)
	}
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return "", fmt.Errorf("%w: query is required", ErrDefinitionInvalid)
	}
	switch strings.ToUpper(fields[0]) {
	case "SELECT", "WITH":
		return query, nil
	default:
		return "", fmt.Errorf("%w: query must start with SELECT or WITH", ErrDefinitionInvalid)
	}
}

func (s *PostgresSession) inspectSchemaTx(ctx context.Context, tx pgx.Tx, definition PostgresViewDefinition) ([]NativeField, string, error) {
	rows, err := tx.Query(ctx, `SELECT * FROM (`+definition.Query+`) AS clearsight_source_view LIMIT 0`)
	if err != nil {
		return nil, "", postgresDatabaseError(ctx, ErrExecution)
	}
	descriptions := rows.FieldDescriptions()
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, "", postgresDatabaseError(ctx, ErrExecution)
	}
	if len(descriptions) > HardMaxSchemaFields {
		return nil, "", ErrLimitExceeded
	}
	fields := make([]NativeField, 0, len(descriptions))
	for _, description := range descriptions {
		nativeType := "oid:" + strconv.FormatUint(uint64(description.DataTypeOID), 10)
		if dataType, ok := tx.Conn().TypeMap().TypeForOID(description.DataTypeOID); ok {
			nativeType = dataType.Name
		}
		fields = append(fields, NativeField{Name: description.Name, NativeType: nativeType, Nullable: true})
	}
	fingerprint, err := nativeSchemaFingerprint(fields)
	if err != nil {
		return nil, "", err
	}
	return fields, fingerprint, nil
}

func selectedFieldKinds(schema []NativeField, selected []string) (map[string]ScalarKind, error) {
	available := make(map[string]NativeField, len(schema))
	for _, field := range schema {
		available[field.Name] = field
	}
	result := make(map[string]ScalarKind, len(selected))
	for _, name := range selected {
		field, exists := available[name]
		if !exists {
			return nil, fmt.Errorf("%w: selected field is not projected by the view", ErrDefinitionInvalid)
		}
		result[name] = postgresScalarKind(field.NativeType)
	}
	return result, nil
}

func postgresScalarKind(native string) ScalarKind {
	value := strings.ToLower(strings.TrimSpace(native))
	switch value {
	case "bool", "boolean":
		return ScalarBool
	case "smallint", "int2", "integer", "int", "int4", "bigint", "int8", "numeric", "decimal", "real", "float4", "double precision", "float8", "float":
		return ScalarNumber
	case "date", "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz", "datetime":
		return ScalarTime
	default:
		return ScalarString
	}
}

func selectedProjection(fields []string) string {
	selected := make([]string, 0, len(fields))
	for _, field := range fields {
		selected = append(selected, qualifiedField(field))
	}
	return strings.Join(selected, ",")
}

func qualifiedField(field string) string {
	return "clearsight_source_view." + pgx.Identifier{field}.Sanitize()
}

func qualifiedSelectedField(field string) string {
	return "clearsight_selected." + pgx.Identifier{field}.Sanitize()
}

func nativeSchemaFingerprint(fields []NativeField) (string, error) {
	seen := make(map[string]struct{}, len(fields))
	ordered := append([]NativeField(nil), fields...)
	for _, field := range ordered {
		if !ValidFieldName(field.Name) || field.NativeType != strings.TrimSpace(field.NativeType) || field.NativeType == "" || len(field.NativeType) > HardMaxIdentifierBytes || containsControl(field.NativeType) {
			return "", fmt.Errorf("%w: source schema contains an invalid field", ErrDefinitionInvalid)
		}
		if _, exists := seen[field.Name]; exists {
			return "", fmt.Errorf("%w: source schema contains duplicate fields", ErrDefinitionInvalid)
		}
		seen[field.Name] = struct{}{}
	}
	// Field order is intentionally preserved here because adapter-native
	// projection order can affect page/export consumers. Assurance separately
	// owns its order-independent logical schema fingerprint.
	hash := sha256.New()
	for _, field := range ordered {
		_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%t\n", field.Name, field.NativeType, field.Nullable)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *PostgresSession) receipt(view View, binding Binding, operation Operation, count, byteCount int64, completeness Completeness, definitionFingerprint, schemaFingerprint string) OperationReceipt {
	receipt := OperationReceipt{
		SourceID:              s.connection.SourceID,
		ConnectionID:          s.connection.ID,
		ConnectionVersion:     s.connection.Version,
		AdapterKind:           s.connection.AdapterKind,
		AdapterVersion:        s.connection.AdapterVersion,
		ViewID:                view.ID,
		ViewVersion:           view.Version,
		DefinitionFingerprint: definitionFingerprint,
		SchemaFingerprint:     schemaFingerprint,
		Operation:             operation,
		ObservedAt:            s.now().UTC(),
		Count:                 count,
		Bytes:                 byteCount,
		Completeness:          completeness,
	}
	if binding.ID != "" {
		receipt.BindingID = binding.ID
		receipt.BindingVersion = binding.Version
	}
	return receipt
}

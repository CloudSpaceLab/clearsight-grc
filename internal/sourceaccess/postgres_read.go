package sourceaccess

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (s *PostgresSession) Inspect(ctx context.Context, view View) (SchemaResult, error) {
	if err := s.ready(view); err != nil {
		return SchemaResult{}, err
	}
	definition, err := decodePostgresView(view)
	if err != nil {
		return SchemaResult{}, err
	}
	operationCtx, cancel := s.operationContext(ctx, s.options.StatementTimeout)
	defer cancel()
	tx, err := s.beginReadOnly(operationCtx)
	if err != nil {
		return SchemaResult{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	defer s.rollback(tx)
	fields, schemaFingerprint, err := s.inspectSchemaTx(operationCtx, tx, definition)
	if err != nil {
		return SchemaResult{}, err
	}
	fingerprint, err := ViewFingerprint(view)
	if err != nil {
		return SchemaResult{}, err
	}
	return SchemaResult{
		Fields:  fields,
		Receipt: s.receipt(view, Binding{}, OperationInspect, int64(len(fields)), 0, CompletenessComplete, fingerprint, schemaFingerprint),
	}, nil
}

func (s *PostgresSession) ReadPage(ctx context.Context, view View, binding Binding, request PageRequest) (RecordPage, error) {
	definition, limits, err := s.validateBoundOperation(view, binding, OperationPage)
	if err != nil {
		return RecordPage{}, err
	}
	if len(binding.KeyFields) != 1 {
		return RecordPage{}, fmt.Errorf("%w: PostgreSQL page requires one stable key", ErrCapabilityUnavailable)
	}
	limit := request.Limit
	if limit < 0 {
		return RecordPage{}, ErrLimitExceeded
	}
	if limit == 0 {
		limit = limits.PageRows
	}
	if limit > limits.PageRows {
		return RecordPage{}, ErrLimitExceeded
	}
	if request.After != nil {
		if err := request.After.ValidateInput(); err != nil {
			return RecordPage{}, err
		}
	}

	operationCtx, cancel := s.operationContext(ctx, limits.Timeout)
	defer cancel()
	tx, err := s.beginReadOnly(operationCtx)
	if err != nil {
		return RecordPage{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	defer s.rollback(tx)
	fields, schemaFingerprint, err := s.inspectSchemaTx(operationCtx, tx, definition)
	if err != nil {
		return RecordPage{}, err
	}
	if err := validateExpectedSchema(view, schemaFingerprint); err != nil {
		return RecordPage{}, err
	}
	fieldKinds, err := selectedFieldKinds(fields, binding.SelectedFields)
	if err != nil {
		return RecordPage{}, err
	}
	keyName := binding.KeyFields[0]
	keyField, err := nativeFieldByName(fields, keyName)
	if err != nil {
		return RecordPage{}, err
	}
	keyKind := fieldKinds[keyName]

	var args []any
	where := ""
	if request.After != nil {
		if request.After.Kind != keyKind {
			return RecordPage{}, fmt.Errorf("%w: page cursor type does not match the stable key", ErrDefinitionInvalid)
		}
		argument, err := postgresScalarArgument(*request.After, keyField.NativeType)
		if err != nil {
			return RecordPage{}, err
		}
		args = append(args, argument)
		where = " WHERE " + qualifiedField(keyName) + " > $1"
	}

	inner := "SELECT " + selectedProjection(binding.SelectedFields) +
		" FROM (" + definition.Query + ") AS clearsight_source_view" + where +
		" ORDER BY " + qualifiedField(keyName) +
		" LIMIT " + strconv.Itoa(limit+1)
	args = append(args, limits.ResponseBytes)
	query := boundedPagePayloadQuery(inner, keyName, limit, len(args))
	rows, err := tx.Query(operationCtx, query, args...)
	if err != nil {
		return RecordPage{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	records, byteCount, lookahead, err := readJSONPageRecords(operationCtx, rows, binding.SelectedFields, fieldKinds, keyKind, limits.ResponseBytes)
	if err != nil {
		return RecordPage{}, err
	}
	if len(records) > limit {
		return RecordPage{}, ErrExecution
	}
	if err := validateStableRecords(records, keyName, keyKind); err != nil {
		return RecordPage{}, err
	}

	more := lookahead != nil
	if more {
		if len(records) != limit || len(records) == 0 {
			return RecordPage{}, ErrExecution
		}
		last, exists := records[len(records)-1][keyName]
		if !exists || last.Kind == ScalarNull || scalarIdentity(last) == scalarIdentity(*lookahead) {
			return RecordPage{}, ErrExecution
		}
	}

	var next *Scalar
	completeness := CompletenessComplete
	if more {
		cursor := records[len(records)-1][keyName]
		cursorCopy := cursor
		next = &cursorCopy
		completeness = CompletenessPartial
	}
	fingerprint, err := BindingFingerprint(view, binding)
	if err != nil {
		return RecordPage{}, err
	}
	return RecordPage{
		Records:    records,
		NextCursor: next,
		Receipt:    s.receipt(view, binding, OperationPage, int64(len(records)), byteCount, completeness, fingerprint, schemaFingerprint),
	}, nil
}

func (s *PostgresSession) Lookup(ctx context.Context, view View, binding Binding, request LookupRequest) (LookupResult, error) {
	definition, limits, err := s.validateBoundOperation(view, binding, OperationLookup)
	if err != nil {
		return LookupResult{}, err
	}
	if len(binding.KeyFields) != 1 {
		return LookupResult{}, fmt.Errorf("%w: PostgreSQL lookup requires one stable key", ErrCapabilityUnavailable)
	}
	if len(request.Values) == 0 || len(request.Values) > limits.LookupValues {
		return LookupResult{}, ErrLimitExceeded
	}
	seen := make(map[string]struct{}, len(request.Values))
	var kind ScalarKind
	for index, value := range request.Values {
		if err := value.ValidateInput(); err != nil {
			return LookupResult{}, err
		}
		if index == 0 {
			kind = value.Kind
		} else if value.Kind != kind {
			return LookupResult{}, fmt.Errorf("%w: lookup values must use one scalar type", ErrDefinitionInvalid)
		}
		identity := scalarIdentity(value)
		if _, exists := seen[identity]; exists {
			return LookupResult{}, fmt.Errorf("%w: duplicate lookup value", ErrDefinitionInvalid)
		}
		seen[identity] = struct{}{}
	}

	operationCtx, cancel := s.operationContext(ctx, limits.Timeout)
	defer cancel()
	tx, err := s.beginReadOnly(operationCtx)
	if err != nil {
		return LookupResult{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	defer s.rollback(tx)
	fields, schemaFingerprint, err := s.inspectSchemaTx(operationCtx, tx, definition)
	if err != nil {
		return LookupResult{}, err
	}
	if err := validateExpectedSchema(view, schemaFingerprint); err != nil {
		return LookupResult{}, err
	}
	fieldKinds, err := selectedFieldKinds(fields, binding.SelectedFields)
	if err != nil {
		return LookupResult{}, err
	}
	keyName := binding.KeyFields[0]
	keyField, err := nativeFieldByName(fields, keyName)
	if err != nil {
		return LookupResult{}, err
	}
	keyKind := fieldKinds[keyName]
	if kind != keyKind {
		return LookupResult{}, fmt.Errorf("%w: lookup value type does not match the stable key", ErrDefinitionInvalid)
	}

	args := make([]any, 0, len(request.Values)+1)
	placeholders := make([]string, 0, len(request.Values))
	for index, value := range request.Values {
		argument, err := postgresScalarArgument(value, keyField.NativeType)
		if err != nil {
			return LookupResult{}, err
		}
		args = append(args, argument)
		placeholders = append(placeholders, "$"+strconv.Itoa(index+1))
	}
	inner := "SELECT " + selectedProjection(binding.SelectedFields) +
		" FROM (" + definition.Query + ") AS clearsight_source_view" +
		" WHERE " + qualifiedField(keyName) + " IN (" + strings.Join(placeholders, ",") + ")" +
		" ORDER BY " + qualifiedField(keyName) +
		" LIMIT " + strconv.Itoa(len(request.Values)+1)
	args = append(args, limits.ResponseBytes)
	query := boundedPayloadQuery(inner, keyName, len(args))
	rows, err := tx.Query(operationCtx, query, args...)
	if err != nil {
		return LookupResult{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	records, byteCount, err := readJSONRecords(operationCtx, rows, binding.SelectedFields, fieldKinds, limits.ResponseBytes)
	if err != nil {
		return LookupResult{}, err
	}
	if len(records) > len(request.Values) {
		return LookupResult{}, ErrExecution
	}
	if err := validateStableRecords(records, keyName, keyKind); err != nil {
		return LookupResult{}, err
	}
	fingerprint, err := BindingFingerprint(view, binding)
	if err != nil {
		return LookupResult{}, err
	}
	return LookupResult{
		Records: records,
		Receipt: s.receipt(view, binding, OperationLookup, int64(len(records)), byteCount, CompletenessComplete, fingerprint, schemaFingerprint),
	}, nil
}

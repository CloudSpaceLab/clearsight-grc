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
		return RecordPage{}, fmt.Errorf("%w: PostgreSQL page currently requires one stable key", ErrCapabilityUnavailable)
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
	var args []any
	where := ""
	if request.After != nil {
		if err := request.After.ValidateInput(); err != nil {
			return RecordPage{}, err
		}
		argument, err := postgresScalarArgument(*request.After)
		if err != nil {
			return RecordPage{}, err
		}
		args = append(args, argument)
		where = " WHERE " + qualifiedField(binding.KeyFields[0]) + " > $1"
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
	fieldKinds, err := selectedFieldKinds(fields, binding.SelectedFields)
	if err != nil {
		return RecordPage{}, err
	}
	keyKind := fieldKinds[binding.KeyFields[0]]
	if request.After != nil && request.After.Kind != keyKind {
		return RecordPage{}, fmt.Errorf("%w: page cursor type does not match the stable key", ErrDefinitionInvalid)
	}
	inner := "SELECT " + selectedProjection(binding.SelectedFields) +
		" FROM (" + definition.Query + ") AS clearsight_source_view" + where +
		" ORDER BY " + qualifiedField(binding.KeyFields[0]) +
		" LIMIT " + strconv.Itoa(limit+1)
	args = append(args, limits.ResponseBytes)
	query := boundedPayloadQuery(inner, binding.KeyFields[0], len(args))
	rows, err := tx.Query(operationCtx, query, args...)
	if err != nil {
		return RecordPage{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	records, byteCount, err := readJSONRecords(operationCtx, rows, binding.SelectedFields, fieldKinds, limits.ResponseBytes)
	if err != nil {
		return RecordPage{}, err
	}
	if err := validateStableRecords(records, binding.KeyFields[0], keyKind); err != nil {
		return RecordPage{}, err
	}
	more := len(records) > limit
	if more {
		records = records[:limit]
	}
	var next *Scalar
	completeness := CompletenessComplete
	if more && len(records) > 0 {
		cursor, exists := records[len(records)-1][binding.KeyFields[0]]
		if !exists || cursor.Kind == ScalarNull {
			return RecordPage{}, ErrExecution
		}
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
		return LookupResult{}, fmt.Errorf("%w: PostgreSQL lookup currently requires one stable key", ErrCapabilityUnavailable)
	}
	if len(request.Values) == 0 || len(request.Values) > limits.LookupValues {
		return LookupResult{}, ErrLimitExceeded
	}
	args := make([]any, 0, len(request.Values))
	placeholders := make([]string, 0, len(request.Values))
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
		identity := string(value.Kind) + "\x1f" + value.Text
		if _, exists := seen[identity]; exists {
			return LookupResult{}, fmt.Errorf("%w: duplicate lookup value", ErrDefinitionInvalid)
		}
		seen[identity] = struct{}{}
		argument, err := postgresScalarArgument(value)
		if err != nil {
			return LookupResult{}, err
		}
		args = append(args, argument)
		placeholders = append(placeholders, "$"+strconv.Itoa(index+1))
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
	fieldKinds, err := selectedFieldKinds(fields, binding.SelectedFields)
	if err != nil {
		return LookupResult{}, err
	}
	keyKind := fieldKinds[binding.KeyFields[0]]
	if kind != keyKind {
		return LookupResult{}, fmt.Errorf("%w: lookup value type does not match the stable key", ErrDefinitionInvalid)
	}
	inner := "SELECT " + selectedProjection(binding.SelectedFields) +
		" FROM (" + definition.Query + ") AS clearsight_source_view" +
		" WHERE " + qualifiedField(binding.KeyFields[0]) + " IN (" + strings.Join(placeholders, ",") + ")" +
		" ORDER BY " + qualifiedField(binding.KeyFields[0]) +
		" LIMIT " + strconv.Itoa(len(request.Values)+1)
	args = append(args, limits.ResponseBytes)
	query := boundedPayloadQuery(inner, binding.KeyFields[0], len(args))
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
	if err := validateStableRecords(records, binding.KeyFields[0], keyKind); err != nil {
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

package sourceaccess

import (
	"context"
	"fmt"
	"strings"
)

type PostgresPredicate struct {
	MatchSQL   string
	UnknownSQL string
	Args       []any
}

func (p PostgresPredicate) Validate() error {
	if strings.TrimSpace(p.MatchSQL) == "" || strings.TrimSpace(p.UnknownSQL) == "" || len(p.MatchSQL) > HardMaxDefinitionBytes || len(p.UnknownSQL) > HardMaxDefinitionBytes || len(p.Args) > hardMaxPredicateArgs {
		return fmt.Errorf("%w: bounded PostgreSQL predicate is required", ErrDefinitionInvalid)
	}
	if strings.Contains(p.MatchSQL, ";") || strings.Contains(p.UnknownSQL, ";") || strings.IndexByte(p.MatchSQL, 0) >= 0 || strings.IndexByte(p.UnknownSQL, 0) >= 0 {
		return fmt.Errorf("%w: PostgreSQL predicate contains an unsupported delimiter", ErrDefinitionInvalid)
	}
	return nil
}

type SchemaGuard func([]NativeField) error

type PostgresPredicateEvaluator interface {
	EvaluatePredicate(context.Context, View, Binding, PostgresPredicate, SchemaGuard) (AggregateResult, error)
}

func (s *PostgresSession) EvaluatePredicate(ctx context.Context, view View, binding Binding, predicate PostgresPredicate, guard SchemaGuard) (AggregateResult, error) {
	definition, limits, err := s.validateBoundOperation(view, binding, OperationAggregate)
	if err != nil {
		return AggregateResult{}, err
	}
	if err := predicate.Validate(); err != nil {
		return AggregateResult{}, err
	}
	operationCtx, cancel := s.operationContext(ctx, limits.Timeout)
	defer cancel()
	tx, err := s.beginReadOnly(operationCtx)
	if err != nil {
		return AggregateResult{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	defer s.rollback(tx)
	fields, schemaFingerprint, err := s.inspectSchemaTx(operationCtx, tx, definition)
	if err != nil {
		return AggregateResult{}, err
	}
	selectedFields, err := selectedNativeFields(fields, binding.SelectedFields)
	if err != nil {
		return AggregateResult{}, err
	}
	if guard != nil {
		if err := guard(append([]NativeField(nil), fields...)); err != nil {
			return AggregateResult{}, err
		}
	}
	query := `SELECT count(*)::bigint, ` +
		`count(*) FILTER (WHERE ` + predicate.MatchSQL + `)::bigint, ` +
		`count(*) FILTER (WHERE ` + predicate.UnknownSQL + `)::bigint, ` +
		`count(*) FILTER (WHERE (` + predicate.MatchSQL + `) AND (` + predicate.UnknownSQL + `))::bigint ` +
		`FROM (` + definition.Query + `) AS clearsight_population`
	var total, matched, unknown, overlap int64
	if err := tx.QueryRow(operationCtx, query, predicate.Args...).Scan(&total, &matched, &unknown, &overlap); err != nil {
		return AggregateResult{}, postgresDatabaseError(operationCtx, ErrExecution)
	}
	if total < 0 || matched < 0 || unknown < 0 || overlap != 0 || matched+unknown > total {
		return AggregateResult{}, ErrExecution
	}
	fingerprint, err := BindingFingerprint(view, binding)
	if err != nil {
		return AggregateResult{}, err
	}
	return AggregateResult{
		Fields:       selectedFields,
		TotalCount:   total,
		MatchCount:   matched,
		UnknownCount: unknown,
		ClearCount:   total - matched - unknown,
		Receipt:      s.receipt(view, binding, OperationAggregate, total, 0, CompletenessComplete, fingerprint, schemaFingerprint),
	}, nil
}

func (s *PostgresSession) validateBoundOperation(view View, binding Binding, operation Operation) (PostgresViewDefinition, ResourceLimits, error) {
	if err := s.ready(view); err != nil {
		return PostgresViewDefinition{}, ResourceLimits{}, err
	}
	if err := binding.Validate(view); err != nil {
		return PostgresViewDefinition{}, ResourceLimits{}, err
	}
	if !binding.Allows(operation) {
		return PostgresViewDefinition{}, ResourceLimits{}, ErrCapabilityUnavailable
	}
	definition, err := decodePostgresView(view)
	if err != nil {
		return PostgresViewDefinition{}, ResourceLimits{}, err
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return PostgresViewDefinition{}, ResourceLimits{}, err
	}
	return definition, limits, nil
}

func (s *PostgresSession) ready(view View) error {
	if s == nil || s.pool == nil {
		return ErrConnection
	}
	if err := s.connection.Validate(); err != nil {
		return err
	}
	return view.Validate(s.connection)
}

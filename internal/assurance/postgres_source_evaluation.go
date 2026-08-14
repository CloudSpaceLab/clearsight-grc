package assurance

import (
	"context"
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func (e *PostgresSourceExecutor) Evaluate(ctx context.Context, population PopulationDefinition, condition *CompiledCondition) (EvaluationReceipt, error) {
	population, view, err := e.legacyView(population)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	if condition == nil {
		return EvaluationReceipt{}, fmt.Errorf("%w: compiled condition is required", ErrPopulationInvalid)
	}
	binding := sourceaccess.Binding{
		ID:             population.ID + "-assurance",
		ViewID:         view.ID,
		Version:        populationFingerprint(population),
		Purpose:        "continuous-assurance",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate},
		SelectedFields: condition.requiredSchemaFields(population.SubjectKey),
		KeyFields:      []string{population.SubjectKey},
	}
	receipt, err := e.evaluateBinding(ctx, view, binding, condition)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	receipt.PopulationID = population.ID
	receipt.PopulationFingerprint = populationFingerprint(population)
	return receipt, nil
}

// EvaluateBinding consumes the same reusable Binding that can also serve a
// bounded lookup/page reader. Assurance retains condition compilation and
// tri-state semantics; sourceaccess retains connection and transaction safety.
func (e *PostgresSourceExecutor) EvaluateBinding(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, condition *CompiledCondition) (EvaluationReceipt, error) {
	return e.evaluateBinding(ctx, view, binding, condition)
}

func (e *PostgresSourceExecutor) evaluateBinding(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, condition *CompiledCondition) (EvaluationReceipt, error) {
	if e == nil || e.session == nil {
		return EvaluationReceipt{}, ErrSourceConnection
	}
	if condition == nil {
		return EvaluationReceipt{}, fmt.Errorf("%w: compiled condition is required", ErrPopulationInvalid)
	}
	subjectKey, err := validateAssuranceBinding(view, binding)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	criticalFields := condition.requiredSchemaFields(subjectKey)
	for _, name := range criticalFields {
		if !containsField(binding.SelectedFields, name) {
			return EvaluationReceipt{}, fmt.Errorf("%w: condition field %q is not selected by the binding", ErrPopulationInvalid, name)
		}
	}
	predicate, err := condition.PostgresPredicate()
	if err != nil {
		return EvaluationReceipt{}, err
	}
	expectedRequiredSchema, err := condition.RequiredSchemaFingerprint(subjectKey)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	definitionFingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return EvaluationReceipt{}, mapSourceAccessError(err)
	}

	var completeSchemaFingerprint string
	evaluator := e.session.(sourceaccess.PostgresPredicateEvaluator)
	result, err := evaluator.EvaluatePredicate(ctx, view, binding, sourceaccess.PostgresPredicate{
		MatchSQL: predicate.MatchSQL, UnknownSQL: predicate.UnknownSQL, Args: append([]any(nil), predicate.Args...),
	}, func(fields []sourceaccess.NativeField) error {
		schema, err := logicalSchemaForFields(fields, binding.SelectedFields)
		if err != nil {
			return err
		}
		currentRequiredSchema, err := schemaFingerprintForFields(schema, criticalFields)
		if err != nil {
			return err
		}
		if currentRequiredSchema != expectedRequiredSchema {
			return ErrSourceSchemaChanged
		}
		completeSchemaFingerprint, err = schema.Fingerprint()
		if err != nil {
			return fmt.Errorf("%w: source schema is invalid", ErrPopulationInvalid)
		}
		return nil
	})
	if err != nil {
		return EvaluationReceipt{}, mapSourceAccessError(err)
	}
	if completeSchemaFingerprint == "" {
		return EvaluationReceipt{}, ErrSourceExecution
	}
	return EvaluationReceipt{
		SourceID:              e.sourceID,
		PopulationID:          binding.ID,
		PopulationFingerprint: definitionFingerprint,
		SchemaFingerprint:     completeSchemaFingerprint,
		TotalCount:            result.TotalCount,
		MatchCount:            result.MatchCount,
		UnknownCount:          result.UnknownCount,
		ClearCount:            result.ClearCount,
		EvaluatedAt:           result.Receipt.ObservedAt,
		Complete:              result.Receipt.Completeness == sourceaccess.CompletenessComplete,
	}, nil
}

func (e *PostgresSourceExecutor) legacyView(population PopulationDefinition) (PopulationDefinition, sourceaccess.View, error) {
	if e == nil || e.session == nil {
		return PopulationDefinition{}, sourceaccess.View{}, ErrSourceConnection
	}
	population, err := normalizePopulationDefinition(population)
	if err != nil {
		return PopulationDefinition{}, sourceaccess.View{}, err
	}
	view, err := sourceaccess.NewPostgresView(population.ID, e.session.Connection().ID, "legacy-v1", population.Query, population.SubjectKey)
	if err != nil {
		return PopulationDefinition{}, sourceaccess.View{}, mapSourceAccessError(err)
	}
	return population, view, nil
}

func validateAssuranceBinding(view sourceaccess.View, binding sourceaccess.Binding) (string, error) {
	if err := binding.Validate(view); err != nil {
		return "", mapSourceAccessError(err)
	}
	if !binding.Allows(sourceaccess.OperationAggregate) || len(binding.KeyFields) != 1 {
		return "", fmt.Errorf("%w: assurance binding requires aggregate access and one stable subject key", ErrPopulationInvalid)
	}
	subjectKey := binding.KeyFields[0]
	if !containsField(view.StableKeys, subjectKey) || !containsField(binding.SelectedFields, subjectKey) {
		return "", fmt.Errorf("%w: assurance subject key must be a selected stable view key", ErrPopulationInvalid)
	}
	return subjectKey, nil
}

func logicalSchemaForFields(fields []sourceaccess.NativeField, selected []string) (Schema, error) {
	if len(selected) == 0 {
		return logicalSchema(fields)
	}
	available := make(map[string]sourceaccess.NativeField, len(fields))
	for _, field := range fields {
		available[field.Name] = field
	}
	filtered := make([]sourceaccess.NativeField, 0, len(selected))
	for _, name := range selected {
		field, exists := available[name]
		if !exists {
			return Schema{}, fmt.Errorf("%w: selected binding field %q is not projected by the source view", ErrPopulationInvalid, name)
		}
		filtered = append(filtered, field)
	}
	return logicalSchema(filtered)
}

func logicalSchema(fields []sourceaccess.NativeField) (Schema, error) {
	native := make([]NativeField, 0, len(fields))
	for _, field := range fields {
		native = append(native, NativeField{Name: field.Name, NativeType: field.NativeType, Nullable: field.Nullable})
	}
	schema, err := NormalizeSchema(native)
	if err != nil {
		return Schema{}, fmt.Errorf("%w: source schema is ambiguous or unsupported", ErrPopulationInvalid)
	}
	return schema, nil
}

func containsField(fields []string, target string) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}

func normalizedPostgresSourceOptions(value PostgresSourceOptions) PostgresSourceOptions {
	return sourceaccess.NormalizePostgresOptions(value)
}

func validateExplicitPostgresSourceDSN(value string) error {
	return mapSourceAccessError(sourceaccess.ValidatePostgresDSN(value))
}

func mapSourceAccessError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	switch {
	case errors.Is(err, sourceaccess.ErrCredentials):
		return ErrSourceCredentials
	case errors.Is(err, sourceaccess.ErrConnection):
		return ErrSourceConnection
	case errors.Is(err, sourceaccess.ErrPrivileges):
		return ErrSourcePrivileges
	case errors.Is(err, sourceaccess.ErrExecution):
		return ErrSourceExecution
	case errors.Is(err, sourceaccess.ErrUnsupportedValue):
		return fmt.Errorf("%w: %w", ErrSourceExecution, sourceaccess.ErrUnsupportedValue)
	case errors.Is(err, sourceaccess.ErrLimitExceeded):
		return fmt.Errorf("%w: %w", ErrPopulationInvalid, sourceaccess.ErrLimitExceeded)
	case errors.Is(err, sourceaccess.ErrDefinitionInvalid):
		return fmt.Errorf("%w: %w", ErrPopulationInvalid, sourceaccess.ErrDefinitionInvalid)
	case errors.Is(err, sourceaccess.ErrCapabilityUnavailable):
		return fmt.Errorf("%w: %w", ErrPopulationInvalid, sourceaccess.ErrCapabilityUnavailable)
	default:
		return err
	}
}

package assurance

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const hardMaxPostgresSourceConns int32 = sourceaccess.HardMaxPostgresSourceConns

type PostgresSourceOptions = sourceaccess.PostgresOptions

func DefaultPostgresSourceOptions() PostgresSourceOptions {
	return sourceaccess.DefaultPostgresOptions()
}

// PostgresSourceExecutor evaluates assurance conditions through a PostgreSQL
// sourceaccess session. Sessions opened by the compatibility constructor are
// owned by the executor; injected sessions remain caller-owned.
type PostgresSourceExecutor struct {
	sourceID    string
	session     sourceaccess.Session
	ownsSession bool
}

func OpenPostgresSourceExecutor(ctx context.Context, sourceID, secretRef string, resolver SourceSecretResolver, options PostgresSourceOptions) (*PostgresSourceExecutor, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" || strings.TrimSpace(secretRef) == "" || resolver == nil {
		return nil, fmt.Errorf("%w: source_id, secret_ref and resolver are required", ErrPopulationInvalid)
	}
	connection := sourceaccess.NewPostgresConnection(sourceID, sourceID, "legacy-v1", secretRef)
	session, err := sourceaccess.NewPostgresAdapter(options).Open(ctx, connection, resolver)
	if err != nil {
		return nil, mapSourceAccessError(err)
	}
	executor, err := NewPostgresSourceExecutorWithSession(sourceID, session)
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	executor.ownsSession = true
	return executor, nil
}

func NewPostgresSourceExecutorWithSession(sourceID string, session sourceaccess.Session) (*PostgresSourceExecutor, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" || session == nil {
		return nil, fmt.Errorf("%w: source_id and source session are required", ErrPopulationInvalid)
	}
	connection := session.Connection()
	if err := connection.Validate(); err != nil {
		return nil, mapSourceAccessError(err)
	}
	if connection.SourceID != sourceID || connection.AdapterKind != sourceaccess.AdapterPostgres || connection.AdapterVersion != sourceaccess.PostgresAdapterVersion {
		return nil, fmt.Errorf("%w: source session does not match the PostgreSQL assurance adapter", ErrPopulationInvalid)
	}
	if !session.Capabilities().Has(sourceaccess.CapabilityInspect) || !session.Capabilities().Has(sourceaccess.CapabilityAggregate) {
		return nil, fmt.Errorf("%w: source session lacks assurance capabilities", ErrPopulationInvalid)
	}
	if _, ok := session.(sourceaccess.SchemaReader); !ok {
		return nil, fmt.Errorf("%w: source session cannot inspect schema", ErrPopulationInvalid)
	}
	if _, ok := session.(sourceaccess.PostgresPredicateEvaluator); !ok {
		return nil, fmt.Errorf("%w: source session cannot evaluate PostgreSQL predicates", ErrPopulationInvalid)
	}
	return &PostgresSourceExecutor{sourceID: sourceID, session: session}, nil
}

func (e *PostgresSourceExecutor) Close() {
	if e != nil && e.ownsSession && e.session != nil {
		_ = e.session.Close()
	}
}

func (e *PostgresSourceExecutor) InspectSchema(ctx context.Context, population PopulationDefinition) (SourceSchema, error) {
	population, view, err := e.legacyView(population)
	if err != nil {
		return SourceSchema{}, err
	}
	return e.inspect(ctx, view, population.ID, populationFingerprint(population), population.SubjectKey, nil)
}

// InspectBinding returns the assurance schema exposed by a reusable binding.
func (e *PostgresSourceExecutor) InspectBinding(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding) (SourceSchema, error) {
	if e == nil || e.session == nil {
		return SourceSchema{}, ErrSourceConnection
	}
	subjectKey, err := validateAssuranceBinding(view, binding)
	if err != nil {
		return SourceSchema{}, err
	}
	fingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return SourceSchema{}, mapSourceAccessError(err)
	}
	return e.inspect(ctx, view, binding.ID, fingerprint, subjectKey, binding.SelectedFields)
}

func (e *PostgresSourceExecutor) inspect(ctx context.Context, view sourceaccess.View, populationID, populationDefinitionFingerprint, subjectKey string, selectedFields []string) (SourceSchema, error) {
	if e == nil || e.session == nil {
		return SourceSchema{}, ErrSourceConnection
	}
	reader, ok := e.session.(sourceaccess.SchemaReader)
	if !ok {
		return SourceSchema{}, ErrSourceConnection
	}
	result, err := reader.Inspect(ctx, view)
	if err != nil {
		return SourceSchema{}, mapSourceAccessError(err)
	}
	schema, err := logicalSchemaForFields(result.Fields, selectedFields)
	if err != nil {
		return SourceSchema{}, err
	}
	if _, exists := schema.Field(subjectKey); !exists {
		return SourceSchema{}, fmt.Errorf("%w: subject_key is not projected by the source view", ErrPopulationInvalid)
	}
	for _, name := range selectedFields {
		if _, exists := schema.Field(name); !exists {
			return SourceSchema{}, fmt.Errorf("%w: selected binding field %q is not projected by the source view", ErrPopulationInvalid, name)
		}
	}
	fingerprint, err := schema.Fingerprint()
	if err != nil {
		return SourceSchema{}, fmt.Errorf("%w: source schema is invalid", ErrPopulationInvalid)
	}
	return SourceSchema{
		SourceID:              e.sourceID,
		PopulationID:          populationID,
		PopulationFingerprint: populationDefinitionFingerprint,
		SchemaFingerprint:     fingerprint,
		Schema:                schema,
		InspectedAt:           result.Receipt.ObservedAt,
	}, nil
}

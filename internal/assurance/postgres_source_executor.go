package assurance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const hardMaxPostgresSourceConns int32 = 4

type PostgresSourceOptions struct {
	MaxConns          int32
	ConnectTimeout    time.Duration
	StatementTimeout  time.Duration
	LockTimeout       time.Duration
	IdleTxTimeout     time.Duration
	PingTimeout       time.Duration
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func DefaultPostgresSourceOptions() PostgresSourceOptions {
	return PostgresSourceOptions{
		MaxConns:          2,
		ConnectTimeout:    5 * time.Second,
		StatementTimeout:  5 * time.Second,
		LockTimeout:       500 * time.Millisecond,
		IdleTxTimeout:     10 * time.Second,
		PingTimeout:       2 * time.Second,
		MaxConnLifetime:   15 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}
}

type PostgresSourceExecutor struct {
	sourceID string
	pool     *pgxpool.Pool
	options  PostgresSourceOptions
	now      func() time.Time
}

func OpenPostgresSourceExecutor(ctx context.Context, sourceID, secretRef string, resolver SourceSecretResolver, options PostgresSourceOptions) (*PostgresSourceExecutor, error) {
	if strings.TrimSpace(sourceID) == "" || strings.TrimSpace(secretRef) == "" || resolver == nil {
		return nil, fmt.Errorf("%w: source_id, secret_ref and resolver are required", ErrPopulationInvalid)
	}
	options = normalizedPostgresSourceOptions(options)
	connectionString, err := resolver.Resolve(ctx, secretRef)
	if err != nil || strings.TrimSpace(connectionString) == "" {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, ErrSourceCredentials
	}

	poolConfig, err := pgxpool.ParseConfig(connectionString)
	connectionString = ""
	if err != nil {
		return nil, ErrSourceConnection
	}
	poolConfig.MinConns = 0
	poolConfig.MinIdleConns = 0
	poolConfig.MaxConns = options.MaxConns
	poolConfig.MaxConnLifetime = options.MaxConnLifetime
	poolConfig.MaxConnIdleTime = options.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = options.HealthCheckPeriod
	poolConfig.PingTimeout = options.PingTimeout
	poolConfig.ConnConfig.ConnectTimeout = options.ConnectTimeout
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "clearsight-assurance-source"
	poolConfig.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = postgresDuration(options.StatementTimeout)
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = postgresDuration(options.LockTimeout)
	poolConfig.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = postgresDuration(options.IdleTxTimeout)
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, sourceDatabaseError(ctx, ErrSourceConnection)
	}
	pingCtx, cancel := context.WithTimeout(ctx, options.ConnectTimeout+options.PingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, sourceDatabaseError(pingCtx, ErrSourceConnection)
	}
	return &PostgresSourceExecutor{sourceID: strings.TrimSpace(sourceID), pool: pool, options: options, now: time.Now}, nil
}

func (e *PostgresSourceExecutor) Close() {
	if e != nil && e.pool != nil {
		e.pool.Close()
	}
}

func (e *PostgresSourceExecutor) InspectSchema(ctx context.Context, population PopulationDefinition) (SourceSchema, error) {
	if e == nil || e.pool == nil {
		return SourceSchema{}, ErrSourceConnection
	}
	population, err := normalizePopulationDefinition(population)
	if err != nil {
		return SourceSchema{}, err
	}
	operationCtx, cancel := e.operationContext(ctx)
	defer cancel()
	tx, err := e.beginReadOnly(operationCtx)
	if err != nil {
		return SourceSchema{}, sourceDatabaseError(operationCtx, ErrSourceExecution)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	return e.inspectSchemaTx(operationCtx, tx, population)
}

func (e *PostgresSourceExecutor) Evaluate(ctx context.Context, population PopulationDefinition, condition *CompiledCondition) (EvaluationReceipt, error) {
	if e == nil || e.pool == nil {
		return EvaluationReceipt{}, ErrSourceConnection
	}
	population, err := normalizePopulationDefinition(population)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	if condition == nil {
		return EvaluationReceipt{}, fmt.Errorf("%w: compiled condition is required", ErrPopulationInvalid)
	}
	predicate, err := condition.PostgresPredicate()
	if err != nil {
		return EvaluationReceipt{}, err
	}
	expectedSchema, err := condition.SchemaFingerprint()
	if err != nil {
		return EvaluationReceipt{}, err
	}

	operationCtx, cancel := e.operationContext(ctx)
	defer cancel()
	tx, err := e.beginReadOnly(operationCtx)
	if err != nil {
		return EvaluationReceipt{}, sourceDatabaseError(operationCtx, ErrSourceExecution)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	currentSchema, err := e.inspectSchemaTx(operationCtx, tx, population)
	if err != nil {
		return EvaluationReceipt{}, err
	}
	if currentSchema.SchemaFingerprint != expectedSchema {
		return EvaluationReceipt{}, ErrSourceSchemaChanged
	}

	query := `SELECT count(*)::bigint, ` +
		`count(*) FILTER (WHERE ` + predicate.MatchSQL + `)::bigint, ` +
		`count(*) FILTER (WHERE ` + predicate.UnknownSQL + `)::bigint, ` +
		`count(*) FILTER (WHERE (` + predicate.MatchSQL + `) AND (` + predicate.UnknownSQL + `))::bigint ` +
		`FROM (` + population.Query + `) AS clearsight_population`
	var total, matched, unknown, overlap int64
	if err := tx.QueryRow(operationCtx, query, predicate.Args...).Scan(&total, &matched, &unknown, &overlap); err != nil {
		return EvaluationReceipt{}, sourceDatabaseError(operationCtx, ErrSourceExecution)
	}
	if total < 0 || matched < 0 || unknown < 0 || overlap != 0 || matched+unknown > total {
		return EvaluationReceipt{}, ErrSourceExecution
	}
	return EvaluationReceipt{
		SourceID:              e.sourceID,
		PopulationID:          population.ID,
		PopulationFingerprint: populationFingerprint(population),
		SchemaFingerprint:     currentSchema.SchemaFingerprint,
		TotalCount:            total,
		MatchCount:            matched,
		UnknownCount:          unknown,
		ClearCount:            total - matched - unknown,
		EvaluatedAt:           e.now().UTC(),
		Complete:              true,
	}, nil
}

func (e *PostgresSourceExecutor) beginReadOnly(ctx context.Context) (pgx.Tx, error) {
	return e.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
}

func (e *PostgresSourceExecutor) inspectSchemaTx(ctx context.Context, tx pgx.Tx, population PopulationDefinition) (SourceSchema, error) {
	rows, err := tx.Query(ctx, `SELECT * FROM (`+population.Query+`) AS clearsight_population LIMIT 0`)
	if err != nil {
		return SourceSchema{}, sourceDatabaseError(ctx, ErrSourceExecution)
	}
	descriptions := rows.FieldDescriptions()
	rows.Close()
	if err := rows.Err(); err != nil {
		return SourceSchema{}, sourceDatabaseError(ctx, ErrSourceExecution)
	}
	native := make([]NativeField, 0, len(descriptions))
	for _, description := range descriptions {
		nativeType := "oid:" + strconv.FormatUint(uint64(description.DataTypeOID), 10)
		if dataType, ok := tx.Conn().TypeMap().TypeForOID(description.DataTypeOID); ok {
			nativeType = dataType.Name
		}
		native = append(native, NativeField{Name: description.Name, NativeType: nativeType, Nullable: true})
	}
	schema, err := NormalizeSchema(native)
	if err != nil {
		return SourceSchema{}, fmt.Errorf("%w: source schema is ambiguous or unsupported", ErrPopulationInvalid)
	}
	if _, exists := schema.Field(population.SubjectKey); !exists {
		return SourceSchema{}, fmt.Errorf("%w: subject_key is not projected by the population", ErrPopulationInvalid)
	}
	fingerprint, err := schema.Fingerprint()
	if err != nil {
		return SourceSchema{}, fmt.Errorf("%w: source schema is invalid", ErrPopulationInvalid)
	}
	return SourceSchema{
		SourceID:              e.sourceID,
		PopulationID:          population.ID,
		PopulationFingerprint: populationFingerprint(population),
		SchemaFingerprint:     fingerprint,
		Schema:                schema,
		InspectedAt:           e.now().UTC(),
	}, nil
}

func (e *PostgresSourceExecutor) operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, e.options.ConnectTimeout+e.options.StatementTimeout+e.options.PingTimeout)
}

func normalizedPostgresSourceOptions(value PostgresSourceOptions) PostgresSourceOptions {
	defaults := DefaultPostgresSourceOptions()
	if value.MaxConns <= 0 {
		value.MaxConns = defaults.MaxConns
	}
	if value.MaxConns > hardMaxPostgresSourceConns {
		value.MaxConns = hardMaxPostgresSourceConns
	}
	value.ConnectTimeout = boundedDuration(value.ConnectTimeout, defaults.ConnectTimeout, 15*time.Second)
	value.StatementTimeout = boundedDuration(value.StatementTimeout, defaults.StatementTimeout, 30*time.Second)
	value.LockTimeout = boundedDuration(value.LockTimeout, defaults.LockTimeout, 5*time.Second)
	value.IdleTxTimeout = boundedDuration(value.IdleTxTimeout, defaults.IdleTxTimeout, 30*time.Second)
	value.PingTimeout = boundedDuration(value.PingTimeout, defaults.PingTimeout, 5*time.Second)
	value.MaxConnLifetime = boundedDuration(value.MaxConnLifetime, defaults.MaxConnLifetime, time.Hour)
	value.MaxConnIdleTime = boundedDuration(value.MaxConnIdleTime, defaults.MaxConnIdleTime, 15*time.Minute)
	value.HealthCheckPeriod = boundedDuration(value.HealthCheckPeriod, defaults.HealthCheckPeriod, 2*time.Minute)
	return value
}

func boundedDuration(value, fallback, maximum time.Duration) time.Duration {
	if value <= 0 {
		value = fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func postgresDuration(value time.Duration) string {
	milliseconds := value.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	return strconv.FormatInt(milliseconds, 10) + "ms"
}

func sourceDatabaseError(ctx context.Context, fallback error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if fallback == nil {
		return ErrSourceExecution
	}
	return fallback
}

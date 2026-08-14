package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	HardMaxPostgresSourceConns int32 = 4
	PostgresAdapterVersion           = "postgres-v1"
	hardMaxPredicateArgs             = 256
)

type PostgresOptions struct {
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

func DefaultPostgresOptions() PostgresOptions {
	return PostgresOptions{
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

func NormalizePostgresOptions(value PostgresOptions) PostgresOptions {
	defaults := DefaultPostgresOptions()
	if value.MaxConns <= 0 {
		value.MaxConns = defaults.MaxConns
	}
	if value.MaxConns > HardMaxPostgresSourceConns {
		value.MaxConns = HardMaxPostgresSourceConns
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

type PostgresAdapter struct {
	options PostgresOptions
}

func NewPostgresAdapter(options PostgresOptions) PostgresAdapter {
	return PostgresAdapter{options: NormalizePostgresOptions(options)}
}

type PostgresViewDefinition struct {
	Query string `json:"query"`
}

func NewPostgresConnection(id, sourceID, version, secretRef string) Connection {
	return Connection{
		ID:             strings.TrimSpace(id),
		SourceID:       strings.TrimSpace(sourceID),
		Version:        strings.TrimSpace(version),
		AdapterKind:    AdapterPostgres,
		AdapterVersion: PostgresAdapterVersion,
		SecretRef:      strings.TrimSpace(secretRef),
	}
}

func NewPostgresView(id, connectionID, version, query string, stableKeys ...string) (View, error) {
	query, err := normalizePostgresQuery(query)
	if err != nil {
		return View{}, err
	}
	definition, err := json.Marshal(PostgresViewDefinition{Query: query})
	if err != nil {
		return View{}, fmt.Errorf("%w: PostgreSQL view definition cannot be encoded", ErrDefinitionInvalid)
	}
	view := View{
		ID:           strings.TrimSpace(id),
		ConnectionID: strings.TrimSpace(connectionID),
		Version:      strings.TrimSpace(version),
		OutputKind:   OutputRecords,
		Definition:   definition,
		StableKeys:   append([]string(nil), stableKeys...),
	}
	return view, nil
}

func (a PostgresAdapter) Open(ctx context.Context, connection Connection, resolver SecretResolver) (Session, error) {
	if err := connection.Validate(); err != nil {
		return nil, err
	}
	if connection.AdapterKind != AdapterPostgres || connection.AdapterVersion != PostgresAdapterVersion || strings.TrimSpace(connection.SecretRef) == "" || resolver == nil {
		return nil, fmt.Errorf("%w: matching PostgreSQL adapter version, secret reference and resolver are required", ErrDefinitionInvalid)
	}
	if len(connection.Definition) != 0 {
		return nil, fmt.Errorf("%w: PostgreSQL connection definition is not supported in this version", ErrDefinitionInvalid)
	}
	options := NormalizePostgresOptions(a.options)
	connectionString, err := resolver.Resolve(ctx, connection.SecretRef)
	if err != nil || strings.TrimSpace(connectionString) == "" {
		connectionString = ""
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, ErrCredentials
	}
	connectionString = strings.TrimSpace(connectionString)
	if err := ValidatePostgresDSN(connectionString); err != nil {
		connectionString = ""
		return nil, ErrConnection
	}

	poolConfig, err := pgxpool.ParseConfig(connectionString)
	connectionString = ""
	if err != nil {
		return nil, ErrConnection
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
	delete(poolConfig.ConnConfig.RuntimeParams, "options")
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "clearsight-sourceaccess"
	poolConfig.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = postgresDuration(options.StatementTimeout)
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = postgresDuration(options.LockTimeout)
	poolConfig.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = postgresDuration(options.IdleTxTimeout)
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, postgresDatabaseError(ctx, ErrConnection)
	}
	pingCtx, cancel := context.WithTimeout(ctx, options.ConnectTimeout+options.PingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, postgresDatabaseError(pingCtx, ErrConnection)
	}
	if err := verifyPostgresPrincipal(pingCtx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresSession{connection: connection, pool: pool, options: options, now: time.Now}, nil
}

type PostgresSession struct {
	connection Connection
	pool       *pgxpool.Pool
	options    PostgresOptions
	now        func() time.Time
}

func (s *PostgresSession) Connection() Connection {
	if s == nil {
		return Connection{}
	}
	return s.connection
}

func (s *PostgresSession) Capabilities() CapabilitySet {
	return NewCapabilitySet(CapabilityInspect, CapabilityPage, CapabilityLookup, CapabilityAggregate)
}

func (s *PostgresSession) Close() error {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *PostgresSession) beginReadOnly(ctx context.Context) (pgx.Tx, error) {
	return s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
}

func (s *PostgresSession) rollback(tx pgx.Tx) {
	if s == nil || tx == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.options.PingTimeout)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (s *PostgresSession) operationContext(ctx context.Context, requested time.Duration) (context.Context, context.CancelFunc) {
	timeout := requested
	if timeout <= 0 || timeout > s.options.StatementTimeout {
		timeout = s.options.StatementTimeout
	}
	if timeout == s.options.StatementTimeout {
		timeout += s.options.PingTimeout
	}
	return context.WithTimeout(ctx, timeout)
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

func ValidatePostgresDSN(value string) error {
	parsed, err := neturl.Parse(strings.TrimSpace(value))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Hostname() == "" || parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" || strings.Trim(parsed.Path, "/") == "" {
		return ErrConnection
	}
	return nil
}

func verifyPostgresPrincipal(ctx context.Context, pool *pgxpool.Pool) error {
	var superuser, createRole, createDB, replication, bypassRLS, ownsRelation bool
	if err := pool.QueryRow(ctx, `
		SELECT r.rolsuper,
		       r.rolcreaterole,
		       r.rolcreatedb,
		       r.rolreplication,
		       r.rolbypassrls,
		       EXISTS (
		           SELECT 1
		             FROM pg_class c
		             JOIN pg_namespace n ON n.oid = c.relnamespace
		            WHERE c.relowner = r.oid
		              AND c.relkind IN ('r','p','v','m','f')
		              AND n.nspname <> 'information_schema'
		              AND n.nspname NOT LIKE 'pg_%'
		       )
		  FROM pg_roles r
		 WHERE r.rolname = current_user`).Scan(&superuser, &createRole, &createDB, &replication, &bypassRLS, &ownsRelation); err != nil {
		return postgresDatabaseError(ctx, ErrConnection)
	}
	if superuser || createRole || createDB || replication || bypassRLS || ownsRelation {
		return ErrPrivileges
	}
	return nil
}

func postgresDatabaseError(ctx context.Context, fallback error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if fallback == nil {
		return ErrExecution
	}
	return fallback
}

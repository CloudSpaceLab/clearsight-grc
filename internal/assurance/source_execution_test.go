package assurance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type testSecretResolver struct {
	value string
	err   error
}

func (r testSecretResolver) Resolve(context.Context, string) (string, error) {
	return r.value, r.err
}

func TestPopulationDefinitionQueryHygieneAndFingerprint(t *testing.T) {
	accepted := []string{
		"SELECT id, status FROM accounts",
		"  WITH current_accounts AS (SELECT id FROM accounts) SELECT id FROM current_accounts  ",
		"SELECT id FROM accounts;",
	}
	for _, query := range accepted {
		population, err := normalizePopulationDefinition(PopulationDefinition{ID: "accounts", Query: query, SubjectKey: "id"})
		if err != nil {
			t.Fatalf("accept %q: %v", query, err)
		}
		if strings.HasSuffix(population.Query, ";") {
			t.Fatalf("trailing semicolon was not normalized: %q", population.Query)
		}
		if populationFingerprint(population) == "" {
			t.Fatal("population fingerprint is empty")
		}
	}

	rejected := []string{
		"",
		"DELETE FROM accounts",
		"UPDATE accounts SET status='ACTIVE'",
		"SELECT id FROM accounts; DELETE FROM accounts",
		"/* comment */ SELECT id FROM accounts",
		"SELECT id FROM accounts\x00",
		strings.Repeat("X", hardMaxPopulationQueryBytes+1),
	}
	for _, query := range rejected {
		if _, err := normalizePopulationDefinition(PopulationDefinition{ID: "accounts", Query: query, SubjectKey: "id"}); !errors.Is(err, ErrPopulationInvalid) {
			t.Fatalf("query %q should be rejected, got %v", query, err)
		}
	}
}

func TestCompiledConditionSchemaFingerprintMatchesSchema(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "id", Type: TypeString, Nullable: true}, {Name: "score", Type: TypeNumber, Nullable: true}}}
	compiled, err := CompileCondition(schema, Condition{Op: OpGT, Field: "score", Value: NumberLiteral(3)}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := compiled.SchemaFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	want, err := schema.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("compiled schema fingerprint=%s want=%s", got, want)
	}
}

func TestPostgresSourceOptionsHaveNonRaiseableBounds(t *testing.T) {
	options := normalizedPostgresSourceOptions(PostgresSourceOptions{
		MaxConns:          100,
		ConnectTimeout:    time.Hour,
		StatementTimeout:  time.Hour,
		LockTimeout:       time.Hour,
		IdleTxTimeout:     time.Hour,
		PingTimeout:       time.Hour,
		MaxConnLifetime:   24 * time.Hour,
		MaxConnIdleTime:   24 * time.Hour,
		HealthCheckPeriod: 24 * time.Hour,
	})
	if options.MaxConns != hardMaxPostgresSourceConns {
		t.Fatalf("max conns=%d want=%d", options.MaxConns, hardMaxPostgresSourceConns)
	}
	if options.ConnectTimeout > 15*time.Second || options.StatementTimeout > 30*time.Second || options.LockTimeout > 5*time.Second || options.IdleTxTimeout > 30*time.Second || options.PingTimeout > 5*time.Second {
		t.Fatalf("runtime timeout ceiling not enforced: %+v", options)
	}
	if options.MaxConnLifetime > time.Hour || options.MaxConnIdleTime > 15*time.Minute || options.HealthCheckPeriod > 2*time.Minute {
		t.Fatalf("pool lifetime ceiling not enforced: %+v", options)
	}
}

func TestPostgresSourceOpenSanitizesCredentialAndConnectionErrors(t *testing.T) {
	secretMarker := "SUPER_SECRET_MARKER"
	_, err := OpenPostgresSourceExecutor(context.Background(), "source-1", "opaque-ref", testSecretResolver{err: errors.New(secretMarker)}, PostgresSourceOptions{})
	if !errors.Is(err, ErrSourceCredentials) || strings.Contains(err.Error(), secretMarker) {
		t.Fatalf("resolver error was not sanitized: %v", err)
	}

	_, err = OpenPostgresSourceExecutor(context.Background(), "source-1", "opaque-ref", testSecretResolver{value: "postgres://user:" + secretMarker + "@[%"}, PostgresSourceOptions{})
	if !errors.Is(err, ErrSourceConnection) || strings.Contains(err.Error(), secretMarker) {
		t.Fatalf("connection parse error was not sanitized: %v", err)
	}
}

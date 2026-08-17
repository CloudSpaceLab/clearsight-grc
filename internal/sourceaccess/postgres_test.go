package sourceaccess

import (
	"context"
	"encoding/json"
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

func TestPostgresOptionsHaveNonRaiseableBounds(t *testing.T) {
	options := NormalizePostgresOptions(PostgresOptions{
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
	if options.MaxConns != HardMaxPostgresSourceConns {
		t.Fatalf("max conns=%d want=%d", options.MaxConns, HardMaxPostgresSourceConns)
	}
	if options.ConnectTimeout > 15*time.Second || options.StatementTimeout > 30*time.Second || options.LockTimeout > 5*time.Second || options.IdleTxTimeout > 30*time.Second || options.PingTimeout > 5*time.Second {
		t.Fatalf("runtime timeout ceiling not enforced: %+v", options)
	}
	if options.MaxConnLifetime > time.Hour || options.MaxConnIdleTime > 15*time.Minute || options.HealthCheckPeriod > 2*time.Minute {
		t.Fatalf("pool lifetime ceiling not enforced: %+v", options)
	}
}

func TestPostgresViewQueryHygiene(t *testing.T) {
	accepted := []string{
		"SELECT id,status FROM accounts",
		" WITH active AS (SELECT id,status FROM accounts) SELECT id,status FROM active ",
		"SELECT id FROM accounts;",
	}
	for _, query := range accepted {
		view, err := NewPostgresView("accounts", "core-read", "v1", query, "id")
		if err != nil {
			t.Fatalf("accept %q: %v", query, err)
		}
		definition, err := decodePostgresView(view)
		if err != nil {
			t.Fatalf("decode %q: %v", query, err)
		}
		if strings.HasSuffix(definition.Query, ";") {
			t.Fatalf("trailing semicolon was not normalized: %q", definition.Query)
		}
	}

	rejected := []string{
		"",
		"DELETE FROM accounts",
		"UPDATE accounts SET status='ACTIVE'",
		"SELECT id FROM accounts; DELETE FROM accounts",
		"/* comment */ SELECT id FROM accounts",
		"SELECT id FROM accounts\x00",
		strings.Repeat("X", HardMaxDefinitionBytes+1),
	}
	for _, query := range rejected {
		if _, err := NewPostgresView("accounts", "core-read", "v1", query, "id"); !errors.Is(err, ErrDefinitionInvalid) && !errors.Is(err, ErrLimitExceeded) {
			t.Fatalf("query %q should be rejected, got %v", query, err)
		}
	}
}

func TestPostgresViewDefinitionRejectsUnknownAndTrailingFields(t *testing.T) {
	connection := NewPostgresConnection("core-read", "core", "v1", "secret-ref")
	for _, definition := range []string{
		`{"query":"SELECT id FROM accounts","other":true}`,
		`{"query":"SELECT id FROM accounts"} {"query":"SELECT id FROM other"}`,
		`{"query":"SELECT id FROM accounts"} trailing`,
	} {
		view := View{ID: "accounts", ConnectionID: connection.ID, Version: "v1", OutputKind: OutputRecords, Definition: json.RawMessage(definition), StableKeys: []string{"id"}}
		if _, err := decodePostgresView(view); !errors.Is(err, ErrDefinitionInvalid) {
			t.Fatalf("definition %q should fail, got %v", definition, err)
		}
	}
}

func TestPostgresAdapterRejectsMismatchedOrIgnoredConnectionConfiguration(t *testing.T) {
	adapter := NewPostgresAdapter(PostgresOptions{})
	connection := NewPostgresConnection("core-read", "core", "v1", "secret-ref")
	connection.AdapterVersion = "postgres-v0"
	if _, err := adapter.Open(context.Background(), connection, testSecretResolver{value: "postgres://reader@db/risk"}); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("mismatched adapter version should fail, got %v", err)
	}

	connection = NewPostgresConnection("core-read", "core", "v1", "secret-ref")
	connection.Definition = json.RawMessage(`{"silently_ignored":true}`)
	if _, err := adapter.Open(context.Background(), connection, testSecretResolver{value: "postgres://reader@db/risk"}); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("ignored connection definition should fail, got %v", err)
	}
}

func TestPostgresOpenSanitizesCredentialAndConnectionErrors(t *testing.T) {
	adapter := NewPostgresAdapter(PostgresOptions{})
	connection := NewPostgresConnection("core-read", "core", "v1", "opaque-ref")
	secretMarker := "SUPER_SECRET_MARKER"
	_, err := adapter.Open(context.Background(), connection, testSecretResolver{err: errors.New(secretMarker)})
	if !errors.Is(err, ErrCredentials) || strings.Contains(err.Error(), secretMarker) {
		t.Fatalf("resolver error was not sanitized: %v", err)
	}

	_, err = adapter.Open(context.Background(), connection, testSecretResolver{value: "postgres://user:" + secretMarker + "@[%"})
	if !errors.Is(err, ErrConnection) || strings.Contains(err.Error(), secretMarker) {
		t.Fatalf("connection parse error was not sanitized: %v", err)
	}
}

func TestExplicitPostgresDSNRequiresACompleteURL(t *testing.T) {
	accepted := []string{
		"postgres://reader@db.internal:5432/risk?sslmode=require",
		"postgresql://reader:secret@127.0.0.1:5432/risk?sslmode=disable",
	}
	for _, value := range accepted {
		if err := ValidatePostgresDSN(value); err != nil {
			t.Fatalf("explicit source DSN %q rejected: %v", value, err)
		}
	}

	rejected := []string{
		"host=db.internal user=reader dbname=risk",
		"postgres:///risk",
		"postgres://db.internal/risk",
		"postgres://reader@db.internal",
		"mysql://reader@db.internal/risk",
	}
	for _, value := range rejected {
		if err := ValidatePostgresDSN(value); !errors.Is(err, ErrConnection) {
			t.Fatalf("incomplete source DSN %q should be rejected, got %v", value, err)
		}
	}
}

func TestPostgresPredicateIsBoundedButKeepsArgumentsSeparate(t *testing.T) {
	predicate := PostgresPredicate{MatchSQL: `"status" = $1`, UnknownSQL: "FALSE", Args: []any{"ACTIVE"}}
	if err := predicate.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []PostgresPredicate{
		{},
		{MatchSQL: "TRUE; SELECT 1", UnknownSQL: "FALSE"},
		{MatchSQL: "TRUE", UnknownSQL: "FALSE\x00"},
		{MatchSQL: "TRUE", UnknownSQL: "FALSE", Args: make([]any, hardMaxPredicateArgs+1)},
	} {
		if err := invalid.Validate(); !errors.Is(err, ErrDefinitionInvalid) {
			t.Fatalf("predicate %#v should fail, got %v", invalid, err)
		}
	}
}

func TestPostgresScalarDecodingPreservesExactNumbersAndCanonicalizesTime(t *testing.T) {
	number, err := scalarFromJSON(json.Number("9007199254740994.125"), ScalarNumber)
	if err != nil {
		t.Fatal(err)
	}
	if number.Kind != ScalarNumber || number.Text != "9007199254740994.125" {
		t.Fatalf("exact number lost: %#v", number)
	}

	withoutZone, err := scalarFromJSON("2026-08-14T09:50:58.123456", ScalarTime)
	if err != nil {
		t.Fatal(err)
	}
	if withoutZone.Text != "2026-08-14T09:50:58.123456Z" {
		t.Fatalf("timestamp was not canonicalized in UTC: %#v", withoutZone)
	}
	if err := withoutZone.ValidateInput(); err != nil {
		t.Fatalf("canonicalized timestamp cannot be reused as a cursor: %v", err)
	}

	if _, err := scalarFromJSON(json.Number("1"), ScalarString); !errors.Is(err, ErrUnsupportedValue) {
		t.Fatalf("schema/value mismatch should fail, got %v", err)
	}
}

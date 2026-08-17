package sourceaccess

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPostgresOpenPreservesResolverCancellation(t *testing.T) {
	adapter := NewPostgresAdapter(PostgresOptions{})
	connection := NewPostgresConnection("core-read", "core", "v1", "secret-ref")
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		_, err := adapter.Open(context.Background(), connection, testSecretResolver{err: want})
		if !errors.Is(err, want) {
			t.Fatalf("resolver cancellation %v was collapsed into %v", want, err)
		}
	}
}

func TestBoundedPayloadQueryAppliesLimitBeforeReturningPayload(t *testing.T) {
	query := boundedPayloadQuery(`SELECT "Account ID","Payload" FROM source`, "Account ID", 3)
	for _, expected := range []string{
		`sum(octet_length(payload)) OVER`,
		`CASE WHEN cumulative_bytes <= $3 THEN payload ELSE '' END`,
		`cumulative_bytes > $3 AS limit_exceeded`,
		` AS sort_key FROM clearsight_selected`,
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("bounded payload query missing %q: %s", expected, query)
		}
	}
}

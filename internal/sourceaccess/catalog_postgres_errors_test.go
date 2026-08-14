//go:build postgres

package sourceaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestCatalogDatabaseErrorsPreserveCancellation(t *testing.T) {
	for _, err := range []error{context.Canceled, context.DeadlineExceeded} {
		if got := catalogReadError(err); !errors.Is(got, err) {
			t.Fatalf("catalogReadError(%v)=%v", err, got)
		}
		if got := catalogWriteError(err); !errors.Is(got, err) {
			t.Fatalf("catalogWriteError(%v)=%v", err, got)
		}
	}
}

func TestCatalogReadRejectsMalformedIdentifiersWithoutStorageFailure(t *testing.T) {
	for _, code := range []string{"22P02", "22023"} {
		err := &pgconn.PgError{Code: code}
		if got := catalogReadError(err); !errors.Is(got, ErrCatalogInvalid) {
			t.Fatalf("catalogReadError(%s)=%v", code, got)
		}
	}
}

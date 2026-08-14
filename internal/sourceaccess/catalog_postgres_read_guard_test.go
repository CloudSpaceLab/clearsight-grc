//go:build postgres

package sourceaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCatalogExactReadsDistinguishStorageFromInvalidVersion(t *testing.T) {
	ctx := context.Background()
	missingStorage := &PostgresCatalogRepository{}
	for name, read := range map[string]func() error{
		"connection": func() error { _, err := missingStorage.ConnectionRevision(ctx, "tenant", "connection", 1); return err },
		"view":       func() error { _, err := missingStorage.ViewRevision(ctx, "tenant", "view", 1); return err },
		"binding":    func() error { _, err := missingStorage.BindingRevision(ctx, "tenant", "binding", 1); return err },
	} {
		t.Run(name+" storage", func(t *testing.T) {
			if err := read(); !errors.Is(err, ErrCatalogStorage) {
				t.Fatalf("missing repository storage should remain distinguishable, got %v", err)
			}
		})
	}

	configured := &PostgresCatalogRepository{pool: &pgxpool.Pool{}}
	for name, read := range map[string]func() error{
		"connection": func() error { _, err := configured.ConnectionRevision(ctx, "tenant", "connection", 0); return err },
		"view":       func() error { _, err := configured.ViewRevision(ctx, "tenant", "view", 0); return err },
		"binding":    func() error { _, err := configured.BindingRevision(ctx, "tenant", "binding", 0); return err },
	} {
		t.Run(name+" version", func(t *testing.T) {
			if err := read(); !errors.Is(err, ErrCatalogInvalid) {
				t.Fatalf("invalid revision version should be rejected before storage access, got %v", err)
			}
		})
	}
}

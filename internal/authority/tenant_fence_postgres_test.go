//go:build postgres

package authority

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fenceTestRow struct {
	version int64
	err     error
}

func (r fenceTestRow) Scan(dst ...any) error {
	if r.err != nil {
		return r.err
	}
	*dst[0].(*int64) = r.version
	return nil
}

type fenceTestTx struct {
	pgx.Tx
	query string
	args  []any
	row   pgx.Row
}

func (t *fenceTestTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	t.query, t.args = query, args
	return t.row
}

func TestTenantAuthorityFenceUsesVerifiedUUIDAndRequiredLock(t *testing.T) {
	tenant := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	for _, tc := range []struct {
		name, sql string
		lock      func(context.Context, pgx.Tx, pgtype.UUID) (int64, error)
	}{
		{"command", "FOR SHARE", LockTenantAuthorityForCommand},
		{"mutation", "UPDATE tenant_authority_epochs", LockTenantAuthorityForMutation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &fenceTestTx{row: fenceTestRow{version: 3}}
			version, err := tc.lock(context.Background(), tx, tenant)
			if err != nil || version != 3 {
				t.Fatalf("version=%d err=%v", version, err)
			}
			if !strings.Contains(tx.query, tc.sql) || len(tx.args) != 1 || tx.args[0] != tenant {
				t.Fatalf("query=%q args=%v", tx.query, tx.args)
			}
		})
	}
}

func TestTenantAuthorityFenceFailsClosed(t *testing.T) {
	tenant := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	for _, lock := range []func(context.Context, pgx.Tx, pgtype.UUID) (int64, error){LockTenantAuthorityForCommand, LockTenantAuthorityForMutation} {
		tx := &fenceTestTx{row: fenceTestRow{err: pgx.ErrNoRows}}
		if _, err := lock(context.Background(), tx, tenant); !errors.Is(err, ErrAuthorityFenceUnavailable) {
			t.Fatalf("missing epoch: %v", err)
		}
		for _, invalid := range []pgtype.UUID{{}, {Valid: true}} {
			tx.query = ""
			if _, err := lock(context.Background(), tx, invalid); !errors.Is(err, ErrAuthorityFenceUnavailable) || tx.query != "" {
				t.Fatalf("invalid tenant: query=%q err=%v", tx.query, err)
			}
		}
	}
}

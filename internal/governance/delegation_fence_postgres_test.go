//go:build postgres

package governance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type delegationFenceRow struct{ scan func(...any) error }

func (r delegationFenceRow) Scan(dest ...any) error { return r.scan(dest...) }

type delegationFenceTx struct {
	pgx.Tx
	calls     []string
	args      [][]any
	ambiguous bool
}

func (t *delegationFenceTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	t.calls = append(t.calls, sql)
	t.args = append(t.args, args)
	if strings.Contains(sql, "FROM delegations d JOIN") {
		return delegationFenceRow{func(...any) error { return errors.New("stop after read") }}
	}
	return delegationFenceRow{func(dest ...any) error {
		if t.ambiguous {
			*dest[0].(*pgtype.UUID) = pgtype.UUID{}
			return nil
		}
		*dest[0].(*pgtype.UUID) = pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		return nil
	}}
}

func TestDelegationCreateRejectsAmbiguousTenant(t *testing.T) {
	tx := &delegationFenceTx{ambiguous: true}
	if _, err := createDelegationScoped(context.Background(), tx, Delegation{TenantID: "bank"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected invalid tenant to fail closed: %v", err)
	}
	if len(tx.calls) != 1 || !strings.Contains(tx.calls[0], "count(*)=1") {
		t.Fatalf("tenant resolution must reject multiple matches: %v", tx.calls)
	}
}
func (t *delegationFenceTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.calls = append(t.calls, sql)
	t.args = append(t.args, args)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestDelegationCreateReadsBackUsingTheResolvedUUID(t *testing.T) {
	tx := &delegationFenceTx{}
	_, err := createDelegationInTx(context.Background(), tx, Delegation{TenantID: "mutable-slug", LegalEntityID: "entity", ID: "delegation"})
	if err == nil || err.Error() != "stop after read" {
		t.Fatalf("expected read sentinel, got %v", err)
	}
	if len(tx.calls) != 3 {
		t.Fatalf("unexpected operations: %v", tx.calls)
	}
	tenant := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	if tx.args[1][1] != tenant || tx.args[2][0] != tenant {
		t.Fatalf("insert and read must use same tenant UUID: %#v", tx.args)
	}
	if strings.Contains(tx.calls[2], "slug=$1") || !strings.Contains(tx.calls[2], "d.tenant_id=$1") {
		t.Fatalf("read must not re-resolve mutable slug: %s", tx.calls[2])
	}
}

func TestDelegationCreateDoesNotDependOnTenantEpoch(t *testing.T) {
	tx := &delegationFenceTx{}
	if _, err := createDelegationScoped(context.Background(), tx, Delegation{TenantID: "bank"}); err != nil {
		t.Fatal(err)
	}
	if len(tx.calls) != 2 || !strings.Contains(tx.calls[1], "INSERT INTO delegations") {
		t.Fatalf("unexpected operation order: %v", tx.calls)
	}
}

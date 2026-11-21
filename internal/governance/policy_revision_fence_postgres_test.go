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

type policyRevisionFenceRow struct{ scan func(...any) error }

func (r policyRevisionFenceRow) Scan(dest ...any) error { return r.scan(dest...) }

type policyRevisionFenceTx struct {
	pgx.Tx
	calls     []string
	ambiguous bool
}

func (t *policyRevisionFenceTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	t.calls = append(t.calls, sql)
	switch {
	case strings.Contains(sql, "SELECT legal_entity_id"):
		return policyRevisionFenceRow{func(dest ...any) error { *dest[0].(*string) = "entity"; return nil }}
	default:
		return policyRevisionFenceRow{func(dest ...any) error {
			if t.ambiguous {
				*dest[0].(*pgtype.UUID) = pgtype.UUID{}
				return nil
			}
			*dest[0].(*pgtype.UUID) = pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
			return nil
		}}
	}
}

func TestPolicyRevisionRejectsAmbiguousTenantBeforePolicyRead(t *testing.T) {
	tx := &policyRevisionFenceTx{ambiguous: true}
	if _, err := preparePolicyRevisionTx(context.Background(), tx, "bank", "policy"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected invalid tenant to fail closed: %v", err)
	}
	if len(tx.calls) != 1 {
		t.Fatalf("unexpected access after failed tenant lookup: %v", tx.calls)
	}
}
func (t *policyRevisionFenceTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.calls = append(t.calls, sql)
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func TestPolicyRevisionUsesScopedAdvisoryLockWithoutTenantEpoch(t *testing.T) {
	tx := &policyRevisionFenceTx{}
	tenant, err := preparePolicyRevisionTx(context.Background(), tx, "bank", "policy")
	if err != nil || !tenant.Valid || tenant.Bytes != ([16]byte{1}) {
		t.Fatal(err)
	}
	if len(tx.calls) != 3 || !strings.Contains(tx.calls[0], "FROM tenants") || !strings.Contains(tx.calls[1], "FROM routing_policies") || !strings.Contains(tx.calls[2], "pg_advisory_xact_lock") {
		t.Fatalf("unexpected operation order: %v", tx.calls)
	}
}

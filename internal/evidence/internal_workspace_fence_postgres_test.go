//go:build postgres

package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type internalFenceRow struct{ scan func(...any) error }

func (r internalFenceRow) Scan(dest ...any) error { return r.scan(dest...) }

type internalFenceTx struct {
	pgx.Tx
	calls     []string
	args      [][]any
	missing   bool
	ambiguous bool
}

func (t *internalFenceTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	t.calls = append(t.calls, sql)
	t.args = append(t.args, args)
	if strings.Contains(sql, "tenant_authority_epochs") {
		return internalFenceRow{func(dest ...any) error {
			if t.missing {
				return pgx.ErrNoRows
			}
			*dest[0].(*int64) = 7
			return nil
		}}
	}
	return internalFenceRow{func(dest ...any) error {
		if t.ambiguous {
			*dest[0].(*pgtype.UUID) = pgtype.UUID{}
			return nil
		}
		*dest[0].(*pgtype.UUID) = pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		return nil
	}}
}
func TestInternalWorkspaceFencePinsVerifiedTenantBeforeCommand(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{TenantID: "bank-slug", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	tx := &internalFenceTx{}
	pinned, epoch, err := lockInternalWorkspaceTenant(identity.WithActor(context.Background(), actor), tx, now)
	if err != nil || epoch != 7 || pinned.TenantID != "01000000-0000-0000-0000-000000000000" || pinned.PrincipalID != actor.PrincipalID || pinned.LegalEntityID != actor.LegalEntityID {
		t.Fatalf("pinned=%+v epoch=%d err=%v", pinned, epoch, err)
	}
	if len(tx.calls) != 2 || !strings.Contains(tx.calls[0], "count(*)=1") || !strings.Contains(tx.calls[1], "FOR SHARE") || tx.args[0][0] != actor.TenantID {
		t.Fatalf("wrong lock order: %#v %#v", tx.calls, tx.args)
	}
}
func TestInternalWorkspaceFenceFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	if _, _, err := lockInternalWorkspaceTenant(context.Background(), &internalFenceTx{}, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("missing verified actor: %v", err)
	}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	for _, tx := range []*internalFenceTx{{missing: true}, {ambiguous: true}} {
		if _, _, err := lockInternalWorkspaceTenant(identity.WithActor(context.Background(), actor), tx, now); !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("should fail closed: %v", err)
		}
	}
	actor.LegalEntityID = "*"
	tx := &internalFenceTx{}
	if _, _, err := lockInternalWorkspaceTenant(identity.WithActor(context.Background(), actor), tx, now); !errors.Is(err, ErrWorkspaceUnavailable) || len(tx.calls) != 0 {
		t.Fatalf("wildcard scope: %v, calls=%v", err, tx.calls)
	}
}

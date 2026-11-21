package authority

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrAuthorityFenceUnavailable means the verified tenant or its epoch is missing.
var ErrAuthorityFenceUnavailable = errors.New("tenant authority fence unavailable")

// LockTenantAuthorityForCommand holds a shared fence until the transaction ends.
// The caller must pass the verified tenant UUID, before acquiring subject locks.
// This primitive is not a complete authority guarantee until all writers and
// database backstops participate in the same protocol.
func LockTenantAuthorityForCommand(ctx context.Context, tx pgx.Tx, tenant pgtype.UUID) (int64, error) {
	return tenantAuthorityEpoch(ctx, tx, tenant, `SELECT version FROM tenant_authority_epochs WHERE tenant_id=$1 FOR SHARE`)
}

// LockTenantAuthorityForMutation holds the exclusive fence and advances the
// epoch in the same transaction. Rollback also rolls the advance back.
func LockTenantAuthorityForMutation(ctx context.Context, tx pgx.Tx, tenant pgtype.UUID) (int64, error) {
	return tenantAuthorityEpoch(ctx, tx, tenant, `UPDATE tenant_authority_epochs SET version=version+1 WHERE tenant_id=$1 RETURNING version`)
}

func tenantAuthorityEpoch(ctx context.Context, tx pgx.Tx, tenant pgtype.UUID, query string) (int64, error) {
	if tx == nil || !tenant.Valid || tenant.Bytes == ([16]byte{}) {
		return 0, ErrAuthorityFenceUnavailable
	}
	var version int64
	err := tx.QueryRow(ctx, query, tenant).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAuthorityFenceUnavailable
	}
	if err != nil {
		return 0, fmt.Errorf("lock tenant authority: %w", err)
	}
	return version, nil
}

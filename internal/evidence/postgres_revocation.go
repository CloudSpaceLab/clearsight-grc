//go:build postgres

package evidence

import (
	"context"
	"time"
)

func (r *PostgresRepository) RevokeInvitation(ctx context.Context, tenant, id string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE capture_invitations SET revoked_at=COALESCE(revoked_at,$3) WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`, id, tenant, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, tenant, id string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE capture_sessions SET revoked_at=COALESCE(revoked_at,$3) WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`, id, tenant, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

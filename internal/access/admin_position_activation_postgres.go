//go:build postgres

package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const OrganizationPositionActivationWorkClass = "organization-position-activation"

type OrganizationPositionActivationMaintainer struct {
	pool *pgxpool.Pool
}

func NewOrganizationPositionActivationMaintainer(pool *pgxpool.Pool) *OrganizationPositionActivationMaintainer {
	return &OrganizationPositionActivationMaintainer{pool: pool}
}

func (m *OrganizationPositionActivationMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.pool == nil {
		return 0, errors.New("organization position activation store is unavailable")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	now = now.UTC()
	rows, err := m.pool.Query(ctx, `
		SELECT tenant_id::text,legal_entity_id::text,id::text
		FROM organization_position_revisions
		WHERE status='SCHEDULED' AND effective_from IS NOT NULL AND effective_from<=$1
		ORDER BY effective_from,id
		LIMIT $2`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("list due organization position activations: %w", err)
	}
	type dueRevision struct {
		tenantID string
		entityID string
		id       string
	}
	items := make([]dueRevision, 0, limit)
	for rows.Next() {
		var item dueRevision
		if err := rows.Scan(&item.tenantID, &item.entityID, &item.id); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	if err := closeRows(rows); err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		changed, err := m.activateOne(ctx, now, item.tenantID, item.entityID, item.id)
		if err != nil {
			return processed, err
		}
		if changed {
			processed++
		}
	}
	return processed, nil
}

func (m *OrganizationPositionActivationMaintainer) activateOne(ctx context.Context, now time.Time, tenantID, entityID, revisionID string) (bool, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	revision, err := organizationPositionRevision(ctx, tx, tenantID, entityID, revisionID, true)
	if err != nil {
		if errors.Is(err, ErrAdminNotFound) {
			return false, nil
		}
		return false, err
	}
	if revision.Status != "SCHEDULED" || revision.EffectiveFrom == nil || revision.EffectiveFrom.After(now) {
		return false, nil
	}

	if err := applyOrganizationPositionRevision(ctx, tx, tenantID, entityID, revision); err != nil {
		code, terminal := organizationPositionActivationFailure(err)
		if !terminal {
			return false, err
		}
		tag, updateErr := tx.Exec(ctx, `
			UPDATE organization_position_revisions
			SET status='FAILED',activation_attempts=activation_attempts+1,
			    activation_failed_at=$4,activation_error_code=$5
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='SCHEDULED'`,
			tenantID, entityID, revision.ID, now, code)
		if updateErr != nil {
			return false, updateErr
		}
		if tag.RowsAffected() != 1 {
			return false, ErrAdminConflict
		}
		if revision.CheckerID != "" {
			if err := recordAdminDecision(ctx, tx, tenantID, revision.CheckerID, "ORGANIZATION_POSITION_ACTIVATION_FAILED", "ORGANIZATION_POSITION_REVISION", revision.ID); err != nil {
				return false, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	tag, err := tx.Exec(ctx, `
		UPDATE organization_position_revisions
		SET status='APPLIED',applied_at=$4,activation_attempts=activation_attempts+1,
		    activation_failed_at=NULL,activation_error_code=''
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='SCHEDULED'`,
		tenantID, entityID, revision.ID, now)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() != 1 {
		return false, ErrAdminConflict
	}
	if revision.CheckerID == "" {
		return false, ErrAdminConflict
	}
	if err := recordAdminDecision(ctx, tx, tenantID, revision.CheckerID, organizationPositionAppliedEventType(revision.Operation), "ORGANIZATION_POSITION", revision.PositionID); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func organizationPositionActivationFailure(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrAdminConflict):
		return "STALE_OR_CONFLICTING_STATE", true
	case errors.Is(err, ErrAdminInvalid), errors.Is(err, ErrAdminNotFound):
		return "INVALID_CURRENT_STATE", true
	default:
		return "", false
	}
}

//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT prc.id::text,t.slug,prc.program_id::text,prc.principal_id::text,
		       prc.program_version,prc.projection_version,prc.accepted_at
		FROM program_review_checkpoints prc
		JOIN tenants t ON t.id=prc.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND prc.program_id=$2::uuid
		  AND prc.principal_id=$3::uuid
		ORDER BY prc.accepted_at DESC,prc.id DESC
		LIMIT 1`, tenant, programID, principalID)
	value, err := scanProgramReviewCheckpoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *PostgresRepository) RecordProgramReview(ctx context.Context, checkpoint ProgramReviewCheckpoint, event Event) (ProgramReviewCheckpoint, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	defer tx.Rollback(ctx)

	var programVersion, projectedProgramVersion, projectionVersion int64
	err = tx.QueryRow(ctx, `
		SELECT p.version,
		       COALESCE(pss.program_version,0),
		       COALESCE(pss.projection_version,0)
		FROM programs p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT program_version,projection_version
			FROM program_state_snapshots
			WHERE tenant_id=p.tenant_id AND program_id=p.id
			ORDER BY generated_at DESC,projection_version DESC
			LIMIT 1
		) pss ON true
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid
		FOR UPDATE OF p`, checkpoint.TenantID, checkpoint.ProgramID).Scan(&programVersion, &projectedProgramVersion, &projectionVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	if programVersion != checkpoint.ProgramVersion || projectedProgramVersion != checkpoint.ProgramVersion || projectionVersion != checkpoint.ProjectionVersion {
		return ProgramReviewCheckpoint{}, ErrVersionConflict
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO program_review_checkpoints(
			id,tenant_id,program_id,principal_id,program_version,projection_version,accepted_at
		) VALUES(
			$1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7
		)
		ON CONFLICT(tenant_id,program_id,principal_id,program_version,projection_version) DO NOTHING
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),program_id::text,principal_id::text,program_version,projection_version,accepted_at`,
		checkpoint.ID, checkpoint.TenantID, checkpoint.ProgramID, checkpoint.PrincipalID, checkpoint.ProgramVersion, checkpoint.ProjectionVersion, checkpoint.AcceptedAt)
	inserted, scanErr := scanProgramReviewCheckpoint(row)
	if scanErr == nil {
		if err := insertContinuityEvent(ctx, tx, event); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		if err := insertOutbox(ctx, tx, event); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		return inserted, nil
	}
	if !errors.Is(scanErr, pgx.ErrNoRows) {
		if isForeignKeyViolation(scanErr) {
			return ProgramReviewCheckpoint{}, ErrNotFound
		}
		return ProgramReviewCheckpoint{}, scanErr
	}

	row = tx.QueryRow(ctx, `
		SELECT prc.id::text,t.slug,prc.program_id::text,prc.principal_id::text,
		       prc.program_version,prc.projection_version,prc.accepted_at
		FROM program_review_checkpoints prc
		JOIN tenants t ON t.id=prc.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND prc.program_id=$2::uuid
		  AND prc.principal_id=$3::uuid
		  AND prc.program_version=$4
		  AND prc.projection_version=$5`,
		checkpoint.TenantID, checkpoint.ProgramID, checkpoint.PrincipalID, checkpoint.ProgramVersion, checkpoint.ProjectionVersion)
	existing, err := scanProgramReviewCheckpoint(row)
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	return existing, nil
}

func (r *PostgresRepository) ProgramStateVersion(ctx context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error) {
	var state ProgramStateSnapshot
	var dimensions, reasons json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT pss.id::text,t.slug,pss.program_id::text,pss.overall_state,pss.dimensions,pss.reasons,
		       pss.open_matter_count,pss.trigger_type,pss.trigger_id,pss.generated_at,pss.program_version,pss.projection_version
		FROM program_state_snapshots pss
		JOIN tenants t ON t.id=pss.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND pss.program_id=$2::uuid
		  AND pss.projection_version=$3`, tenant, programID, projectionVersion).Scan(
		&state.ID, &state.TenantID, &state.ProgramID, &state.Overall, &dimensions, &reasons,
		&state.OpenMatterCount, &state.TriggerType, &state.TriggerID, &state.GeneratedAt, &state.ProgramVersion, &state.ProjectionVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dimensions, &state.Dimensions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(reasons, &state.Reasons); err != nil {
		return nil, err
	}
	return &state, nil
}

type programReviewScanner interface {
	Scan(...any) error
}

func scanProgramReviewCheckpoint(row programReviewScanner) (ProgramReviewCheckpoint, error) {
	var value ProgramReviewCheckpoint
	err := row.Scan(&value.ID, &value.TenantID, &value.ProgramID, &value.PrincipalID, &value.ProgramVersion, &value.ProjectionVersion, &value.AcceptedAt)
	return value, err
}

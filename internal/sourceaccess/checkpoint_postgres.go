//go:build postgres

package sourceaccess

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCheckpointRepository struct {
	pool    *pgxpool.Pool
	catalog *PostgresCatalogRepository
}

func NewPostgresCheckpointRepository(pool *pgxpool.Pool) *PostgresCheckpointRepository {
	return &PostgresCheckpointRepository{pool: pool, catalog: NewPostgresCatalogRepository(pool)}
}

func (r *PostgresCheckpointRepository) EnsureBindingCheckpoint(ctx context.Context, tenantID, sourceID, bindingID string, bindingVersion int64, now time.Time) (BindingCheckpoint, error) {
	if r == nil || r.pool == nil || r.catalog == nil || bindingVersion < 1 {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	binding, err := r.catalog.BindingRevision(ctx, tenantID, bindingID, bindingVersion)
	if err != nil {
		return BindingCheckpoint{}, err
	}
	if binding.SourceID != sourceID || !statefulBindingRevision(binding) {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO source_binding_checkpoints(
			tenant_id,source_id,binding_id,binding_version,next_attempt_at,created_at,updated_at
		) VALUES (
			(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,$4,$5,$5,$5
		)
		ON CONFLICT (tenant_id,binding_id,binding_version) DO NOTHING`, tenantID, sourceID, bindingID, bindingVersion, now)
	if err != nil {
		return BindingCheckpoint{}, catalogWriteError(err)
	}
	return r.BindingCheckpoint(ctx, tenantID, bindingID, bindingVersion)
}

func (r *PostgresCheckpointRepository) BindingCheckpoint(ctx context.Context, tenantID, bindingID string, bindingVersion int64) (BindingCheckpoint, error) {
	if r == nil || r.pool == nil || bindingVersion < 1 {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	return scanBindingCheckpoint(r.pool.QueryRow(ctx, `
		SELECT cp.tenant_id::text,cp.source_id::text,cp.binding_id::text,cp.binding_version,
		       cp.position_kind,cp.position_value,cp.attempts,cp.locked_by,cp.lease_until,
		       cp.next_attempt_at,cp.last_error_code,cp.failed_at,cp.created_at,cp.updated_at
		  FROM source_binding_checkpoints cp
		  JOIN tenants t ON t.id=cp.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND cp.binding_id=$2::uuid AND cp.binding_version=$3`, tenantID, bindingID, bindingVersion))
}

func (r *PostgresCheckpointRepository) ClaimBindingCheckpoints(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]BindingCheckpoint, error) {
	if r == nil || r.pool == nil {
		return nil, ErrCatalogStorage
	}
	rows, err := r.pool.Query(ctx, `
		WITH due AS (
			SELECT cp.tenant_id,cp.binding_id,cp.binding_version
			  FROM source_binding_checkpoints cp
			  JOIN source_bindings sb
			    ON sb.tenant_id=cp.tenant_id AND sb.source_id=cp.source_id
			   AND sb.binding_id=cp.binding_id AND sb.version=cp.binding_version
			 WHERE cp.failed_at IS NULL
			   AND cp.next_attempt_at <= $1
			   AND (cp.lease_until IS NULL OR cp.lease_until < $1)
			   AND sb.status='ACTIVE' AND sb.is_current
			   AND (sb.operations ? 'PAGE' OR sb.operations ? 'CHANGES')
			 ORDER BY cp.next_attempt_at,cp.binding_id,cp.binding_version
			 LIMIT $2
			 FOR UPDATE OF cp SKIP LOCKED
		), claimed AS (
			UPDATE source_binding_checkpoints cp
			   SET attempts=attempts+1,locked_by=$3,lease_until=$1+$4::interval,updated_at=$1
			  FROM due
			 WHERE cp.tenant_id=due.tenant_id AND cp.binding_id=due.binding_id AND cp.binding_version=due.binding_version
			 RETURNING cp.*
		)
		SELECT tenant_id::text,source_id::text,binding_id::text,binding_version,
		       position_kind,position_value,attempts,locked_by,lease_until,
		       next_attempt_at,last_error_code,failed_at,created_at,updated_at
		  FROM claimed
		 ORDER BY next_attempt_at,binding_id,binding_version`, now, catalogListLimit(limit), worker, lease.String())
	if err != nil {
		return nil, catalogReadError(err)
	}
	defer rows.Close()
	values := make([]BindingCheckpoint, 0)
	for rows.Next() {
		value, scanErr := scanBindingCheckpoint(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, catalogReadError(err)
	}
	return values, nil
}

func (r *PostgresCheckpointRepository) AdvanceBindingCheckpoint(ctx context.Context, claimed BindingCheckpoint, position CheckpointPosition, at, next time.Time) (BindingCheckpoint, error) {
	if r == nil || r.pool == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	if err := validateCheckpointPosition(position); err != nil {
		return BindingCheckpoint{}, err
	}
	value, err := scanBindingCheckpoint(r.pool.QueryRow(ctx, `
		UPDATE source_binding_checkpoints
		   SET position_kind=$5,position_value=$6,attempts=0,locked_by='',lease_until=NULL,
		       next_attempt_at=$7,last_error_code='',failed_at=NULL,updated_at=$8
		 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		   AND binding_id=$2::uuid AND binding_version=$3
		   AND locked_by=$4 AND locked_by<>'' AND lease_until >= $8
		 RETURNING tenant_id::text,source_id::text,binding_id::text,binding_version,
		           position_kind,position_value,attempts,locked_by,lease_until,
		           next_attempt_at,last_error_code,failed_at,created_at,updated_at`,
		claimed.TenantID, claimed.BindingID, claimed.BindingVersion, claimed.LockedBy,
		position.Kind, position.Value, next, at))
	if errors.Is(err, ErrCatalogNotFound) {
		return BindingCheckpoint{}, ErrCheckpointClaimLost
	}
	return value, err
}

func (r *PostgresCheckpointRepository) FailBindingCheckpoint(ctx context.Context, claimed BindingCheckpoint, maxAttempts int, errorCode string, at, next time.Time) (bool, error) {
	if r == nil || r.pool == nil {
		return false, ErrCatalogStorage
	}
	var terminal bool
	err := r.pool.QueryRow(ctx, `
		UPDATE source_binding_checkpoints
		   SET locked_by='',lease_until=NULL,last_error_code=$5,
		       next_attempt_at=CASE WHEN attempts >= $6 THEN next_attempt_at ELSE $7 END,
		       failed_at=CASE WHEN attempts >= $6 THEN $8 ELSE NULL END,
		       updated_at=$8
		 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		   AND binding_id=$2::uuid AND binding_version=$3
		   AND locked_by=$4 AND locked_by<>'' AND lease_until >= $8
		 RETURNING failed_at IS NOT NULL`, claimed.TenantID, claimed.BindingID, claimed.BindingVersion,
		claimed.LockedBy, errorCode, maxAttempts, next, at).Scan(&terminal)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrCheckpointClaimLost
	}
	if err != nil {
		return false, catalogWriteError(err)
	}
	return terminal, nil
}

type checkpointScanner interface {
	Scan(...any) error
}

func scanBindingCheckpoint(row checkpointScanner) (BindingCheckpoint, error) {
	var value BindingCheckpoint
	var positionKind string
	var leaseUntil, failedAt sql.NullTime
	if err := row.Scan(
		&value.TenantID,
		&value.SourceID,
		&value.BindingID,
		&value.BindingVersion,
		&positionKind,
		&value.Position.Value,
		&value.Attempts,
		&value.LockedBy,
		&leaseUntil,
		&value.NextAttemptAt,
		&value.LastErrorCode,
		&failedAt,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return BindingCheckpoint{}, catalogReadError(err)
	}
	value.Position.Kind = CheckpointPositionKind(positionKind)
	if leaseUntil.Valid {
		lease := leaseUntil.Time
		value.LeaseUntil = &lease
	}
	if failedAt.Valid {
		failed := failedAt.Time
		value.FailedAt = &failed
	}
	return value, nil
}

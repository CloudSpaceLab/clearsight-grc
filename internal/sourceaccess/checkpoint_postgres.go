//go:build postgres

package sourceaccess

import (
	"context"
	"errors"
	"time"

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
			tenant_id,source_id,binding_id,binding_version,generation,created_at,updated_at
		) VALUES (
			(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,$4,0,$5,$5
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
		       cp.position_kind,cp.position_value,cp.generation,cp.created_at,cp.updated_at
		  FROM source_binding_checkpoints cp
		  JOIN tenants t ON t.id=cp.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND cp.binding_id=$2::uuid AND cp.binding_version=$3`, tenantID, bindingID, bindingVersion))
}

func (r *PostgresCheckpointRepository) AdvanceBindingCheckpoint(ctx context.Context, expected BindingCheckpoint, position CheckpointPosition, at time.Time) (BindingCheckpoint, error) {
	if r == nil || r.pool == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	if err := validateCheckpointPosition(position); err != nil {
		return BindingCheckpoint{}, err
	}
	value, err := scanBindingCheckpoint(r.pool.QueryRow(ctx, `
		UPDATE source_binding_checkpoints
		   SET position_kind=$5,position_value=$6,generation=generation+1,updated_at=$7
		 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		   AND binding_id=$2::uuid AND binding_version=$3
		   AND generation=$4
		   AND position_kind=$8 AND position_value=$9
		 RETURNING tenant_id::text,source_id::text,binding_id::text,binding_version,
		           position_kind,position_value,generation,created_at,updated_at`,
		expected.TenantID, expected.BindingID, expected.BindingVersion, expected.Generation,
		position.Kind, position.Value, at, expected.Position.Kind, expected.Position.Value))
	if errors.Is(err, ErrCatalogNotFound) {
		return BindingCheckpoint{}, ErrCheckpointConflict
	}
	if err != nil {
		return BindingCheckpoint{}, err
	}
	if value.SourceID != expected.SourceID {
		return BindingCheckpoint{}, ErrCheckpointConflict
	}
	return value, nil
}

type checkpointScanner interface {
	Scan(...any) error
}

func scanBindingCheckpoint(row checkpointScanner) (BindingCheckpoint, error) {
	var value BindingCheckpoint
	var positionKind string
	if err := row.Scan(
		&value.TenantID,
		&value.SourceID,
		&value.BindingID,
		&value.BindingVersion,
		&positionKind,
		&value.Position.Value,
		&value.Generation,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return BindingCheckpoint{}, catalogReadError(err)
	}
	value.Position.Kind = CheckpointPositionKind(positionKind)
	return value, nil
}

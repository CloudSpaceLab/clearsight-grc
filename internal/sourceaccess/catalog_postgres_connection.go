//go:build postgres

package sourceaccess

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresCatalogRepository) CreateConnectionRevision(ctx context.Context, value ConnectionRevision) (ConnectionRevision, error) {
	value, err := normalizeConnectionRevision(value)
	if err != nil {
		return ConnectionRevision{}, err
	}
	if r == nil || r.pool == nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	declared, err := json.Marshal(value.DeclaredCapabilities)
	if err != nil {
		return ConnectionRevision{}, ErrCatalogInvalid
	}
	verified, err := json.Marshal(value.VerifiedCapabilities)
	if err != nil {
		return ConnectionRevision{}, ErrCatalogInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConnectionRevision{}, catalogWriteError(err)
	}
	defer tx.Rollback(ctx)
	if err := requireCatalogSource(ctx, tx, value.TenantID, value.SourceID); err != nil {
		return ConnectionRevision{}, err
	}
	created, err := scanConnectionRevision(tx.QueryRow(ctx, `
		INSERT INTO source_connections AS sc(
			revision_id,connection_id,tenant_id,source_id,code,name,adapter_kind,adapter_version,
			secret_ref,definition,declared_capabilities,verified_capabilities,owner_principal_id,
			status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3),$4::uuid,$5,$6,$7,$8,
			$9,$10::jsonb,$11::jsonb,$12::jsonb,NULLIF($13,'')::uuid,
			$14,$15,$16,$17,$18,NULLIF($19,'')::uuid,$20,$21
		)
		RETURNING `+connectionRevisionColumns,
		value.RevisionID,
		value.ConnectionID,
		value.TenantID,
		value.SourceID,
		value.Code,
		value.Name,
		value.AdapterKind,
		value.AdapterVersion,
		value.SecretRef,
		string(value.Definition),
		string(declared),
		string(verified),
		value.OwnerPrincipalID,
		value.Status,
		value.IsCurrent,
		value.EffectiveFrom,
		value.EffectiveUntil,
		value.Version,
		value.CreatedBy,
		value.CreatedAt,
		value.UpdatedAt,
	))
	if err != nil {
		return ConnectionRevision{}, catalogWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ConnectionRevision{}, catalogWriteError(err)
	}
	return created, nil
}

func (r *PostgresCatalogRepository) ConnectionRevision(ctx context.Context, tenantID, connectionID string, version int64) (ConnectionRevision, error) {
	if r == nil || r.pool == nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	if version < 1 {
		return ConnectionRevision{}, ErrCatalogInvalid
	}
	return scanConnectionRevision(r.pool.QueryRow(ctx, `
		SELECT `+connectionRevisionColumns+`
		  FROM source_connections sc
		  JOIN tenants t ON t.id=sc.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sc.connection_id=$2::uuid
		   AND sc.version=$3`, tenantID, connectionID, version))
}

func (r *PostgresCatalogRepository) CurrentConnection(ctx context.Context, tenantID, connectionID string) (ConnectionRevision, error) {
	if r == nil || r.pool == nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	return scanConnectionRevision(r.pool.QueryRow(ctx, `
		SELECT `+connectionRevisionColumns+`
		  FROM source_connections sc
		  JOIN tenants t ON t.id=sc.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sc.connection_id=$2::uuid
		   AND sc.is_current`, tenantID, connectionID))
}

func (r *PostgresCatalogRepository) ListCurrentConnections(ctx context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
	if r == nil || r.pool == nil {
		return nil, ErrCatalogStorage
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+connectionRevisionColumns+`
		  FROM source_connections sc
		  JOIN tenants t ON t.id=sc.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sc.source_id=$2::uuid
		   AND sc.is_current
		 ORDER BY sc.code,sc.connection_id
		 LIMIT $3`, tenantID, sourceID, catalogListLimit(limit))
	if err != nil {
		return nil, catalogReadError(err)
	}
	defer rows.Close()
	values := make([]ConnectionRevision, 0)
	for rows.Next() {
		value, scanErr := scanConnectionRevision(rows)
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

func requireCatalogSource(ctx context.Context, tx pgx.Tx, tenantID, sourceID string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			  FROM evidence_sources es
			  JOIN tenants t ON t.id=es.tenant_id
			 WHERE (t.id::text=$1 OR t.slug=$1)
			   AND es.id=$2::uuid
		)`, tenantID, sourceID).Scan(&exists); err != nil {
		return catalogWriteError(err)
	}
	if !exists {
		return ErrCatalogNotFound
	}
	return nil
}

//go:build postgres

package sourceaccess

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresCatalogRepository) CreateViewRevision(ctx context.Context, value ViewRevision) (ViewRevision, error) {
	value, err := normalizeViewRevision(value)
	if err != nil {
		return ViewRevision{}, err
	}
	if r == nil || r.pool == nil {
		return ViewRevision{}, ErrCatalogStorage
	}
	stableKeys, err := json.Marshal(value.StableKeys)
	if err != nil {
		return ViewRevision{}, ErrCatalogInvalid
	}
	nativeSchema, err := json.Marshal(value.NativeSchema)
	if err != nil {
		return ViewRevision{}, ErrCatalogInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ViewRevision{}, catalogWriteError(err)
	}
	defer tx.Rollback(ctx)
	parent, err := connectionRevisionForChild(ctx, tx, value.TenantID, value.SourceID, value.ConnectionID, value.ConnectionVersion)
	if err != nil {
		return ViewRevision{}, err
	}
	if value.IsCurrent && !parent.IsCurrent {
		return ViewRevision{}, ErrCatalogNotFound
	}
	if parent.AdapterKind == AdapterReference {
		return ViewRevision{}, ErrCatalogInvalid
	}
	value, err = validateViewAgainstConnection(value, parent)
	if err != nil {
		return ViewRevision{}, err
	}
	created, err := scanViewRevision(tx.QueryRow(ctx, `
		INSERT INTO source_views AS sv(
			revision_id,view_id,tenant_id,source_id,connection_id,connection_version,
			code,name,definition,output_kind,stable_keys,native_schema,schema_fingerprint,
			status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3),$4::uuid,$5::uuid,$6,
			$7,$8,$9::jsonb,$10,$11::jsonb,$12::jsonb,$13,
			$14,$15,$16,$17,$18,NULLIF($19,'')::uuid,$20,$21
		)
		RETURNING `+viewRevisionColumns,
		value.RevisionID,
		value.ViewID,
		value.TenantID,
		value.SourceID,
		value.ConnectionID,
		value.ConnectionVersion,
		value.Code,
		value.Name,
		string(value.Definition),
		value.OutputKind,
		string(stableKeys),
		string(nativeSchema),
		value.SchemaFingerprint,
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
		return ViewRevision{}, catalogWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ViewRevision{}, catalogWriteError(err)
	}
	return created, nil
}

func (r *PostgresCatalogRepository) ViewRevision(ctx context.Context, tenantID, viewID string, version int64) (ViewRevision, error) {
	if r == nil || r.pool == nil || version < 1 {
		return ViewRevision{}, ErrCatalogInvalid
	}
	return scanViewRevision(r.pool.QueryRow(ctx, `
		SELECT `+viewRevisionColumns+`
		  FROM source_views sv
		  JOIN tenants t ON t.id=sv.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sv.view_id=$2::uuid
		   AND sv.version=$3`, tenantID, viewID, version))
}

func (r *PostgresCatalogRepository) CurrentView(ctx context.Context, tenantID, viewID string) (ViewRevision, error) {
	if r == nil || r.pool == nil {
		return ViewRevision{}, ErrCatalogStorage
	}
	return scanViewRevision(r.pool.QueryRow(ctx, `
		SELECT `+viewRevisionColumns+`
		  FROM source_views sv
		  JOIN tenants t ON t.id=sv.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sv.view_id=$2::uuid
		   AND sv.is_current`, tenantID, viewID))
}

func (r *PostgresCatalogRepository) ListCurrentViews(ctx context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
	if r == nil || r.pool == nil {
		return nil, ErrCatalogStorage
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+viewRevisionColumns+`
		  FROM source_views sv
		  JOIN tenants t ON t.id=sv.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sv.connection_id=$2::uuid
		   AND sv.is_current
		 ORDER BY sv.code,sv.view_id
		 LIMIT $3`, tenantID, connectionID, catalogListLimit(limit))
	if err != nil {
		return nil, catalogReadError(err)
	}
	defer rows.Close()
	values := make([]ViewRevision, 0)
	for rows.Next() {
		value, scanErr := scanViewRevision(rows)
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

func connectionRevisionForChild(ctx context.Context, tx pgx.Tx, tenantID, sourceID, connectionID string, version int64) (ConnectionRevision, error) {
	return scanConnectionRevision(tx.QueryRow(ctx, `
		SELECT `+connectionRevisionColumns+`
		  FROM source_connections sc
		  JOIN tenants t ON t.id=sc.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sc.source_id=$2::uuid
		   AND sc.connection_id=$3::uuid
		   AND sc.version=$4`, tenantID, sourceID, connectionID, version))
}

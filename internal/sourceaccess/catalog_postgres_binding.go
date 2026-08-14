//go:build postgres

package sourceaccess

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresCatalogRepository) CreateBindingRevision(ctx context.Context, value BindingRevision) (BindingRevision, error) {
	value, err := normalizeBindingRevision(value)
	if err != nil {
		return BindingRevision{}, err
	}
	if r == nil || r.pool == nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	operations, err := json.Marshal(value.Operations)
	if err != nil {
		return BindingRevision{}, ErrCatalogInvalid
	}
	selectedFields, err := json.Marshal(value.SelectedFields)
	if err != nil {
		return BindingRevision{}, ErrCatalogInvalid
	}
	keyFields, err := json.Marshal(value.KeyFields)
	if err != nil {
		return BindingRevision{}, ErrCatalogInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BindingRevision{}, catalogWriteError(err)
	}
	defer tx.Rollback(ctx)
	parent, err := viewRevisionForChild(ctx, tx, value.TenantID, value.SourceID, value.ViewID, value.ViewVersion)
	if err != nil {
		return BindingRevision{}, err
	}
	if value.IsCurrent && !parent.IsCurrent {
		return BindingRevision{}, ErrCatalogNotFound
	}
	value, err = validateBindingAgainstView(value, parent)
	if err != nil {
		return BindingRevision{}, err
	}
	created, err := scanBindingRevision(tx.QueryRow(ctx, `
		INSERT INTO source_bindings AS sb(
			revision_id,binding_id,tenant_id,source_id,view_id,view_version,
			code,name,purpose,operations,selected_fields,key_fields,
			page_rows,response_bytes,lookup_values,timeout_ms,
			mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,sensitivity_handling,
			status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3),$4::uuid,$5::uuid,$6,
			$7,$8,$9,$10::jsonb,$11::jsonb,$12::jsonb,
			$13,$14,$15,$16,
			$17::jsonb,$18::jsonb,$19::jsonb,$20,$21,$22::jsonb,
			$23,$24,$25,$26,$27,NULLIF($28,'')::uuid,$29,$30
		)
		RETURNING `+bindingRevisionColumns,
		value.RevisionID,
		value.BindingID,
		value.TenantID,
		value.SourceID,
		value.ViewID,
		value.ViewVersion,
		value.Code,
		value.Name,
		value.Purpose,
		string(operations),
		string(selectedFields),
		string(keyFields),
		value.Limits.PageRows,
		value.Limits.ResponseBytes,
		value.Limits.LookupValues,
		value.Limits.Timeout.Milliseconds(),
		string(value.Mapping),
		string(value.ParameterSchema),
		string(value.OutputSchema),
		value.RequiredFreshnessMinutes,
		value.Completeness,
		string(value.SensitivityHandling),
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
		return BindingRevision{}, catalogWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return BindingRevision{}, catalogWriteError(err)
	}
	return created, nil
}

func (r *PostgresCatalogRepository) BindingRevision(ctx context.Context, tenantID, bindingID string, version int64) (BindingRevision, error) {
	if r == nil || r.pool == nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	if version < 1 {
		return BindingRevision{}, ErrCatalogInvalid
	}
	return scanBindingRevision(r.pool.QueryRow(ctx, `
		SELECT `+bindingRevisionColumns+`
		  FROM source_bindings sb
		  JOIN tenants t ON t.id=sb.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sb.binding_id=$2::uuid
		   AND sb.version=$3`, tenantID, bindingID, version))
}

func (r *PostgresCatalogRepository) CurrentBinding(ctx context.Context, tenantID, bindingID string) (BindingRevision, error) {
	if r == nil || r.pool == nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	return scanBindingRevision(r.pool.QueryRow(ctx, `
		SELECT `+bindingRevisionColumns+`
		  FROM source_bindings sb
		  JOIN tenants t ON t.id=sb.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sb.binding_id=$2::uuid
		   AND sb.is_current`, tenantID, bindingID))
}

func (r *PostgresCatalogRepository) ListCurrentBindings(ctx context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
	if r == nil || r.pool == nil {
		return nil, ErrCatalogStorage
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+bindingRevisionColumns+`
		  FROM source_bindings sb
		  JOIN tenants t ON t.id=sb.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sb.view_id=$2::uuid
		   AND sb.is_current
		 ORDER BY sb.code,sb.binding_id
		 LIMIT $3`, tenantID, viewID, catalogListLimit(limit))
	if err != nil {
		return nil, catalogReadError(err)
	}
	defer rows.Close()
	values := make([]BindingRevision, 0)
	for rows.Next() {
		value, scanErr := scanBindingRevision(rows)
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

func viewRevisionForChild(ctx context.Context, tx pgx.Tx, tenantID, sourceID, viewID string, version int64) (ViewRevision, error) {
	return scanViewRevision(tx.QueryRow(ctx, `
		SELECT `+viewRevisionColumns+`
		  FROM source_views sv
		  JOIN tenants t ON t.id=sv.tenant_id
		 WHERE (t.id::text=$1 OR t.slug=$1)
		   AND sv.source_id=$2::uuid
		   AND sv.view_id=$3::uuid
		   AND sv.version=$4
		 FOR SHARE`, tenantID, sourceID, viewID, version))
}

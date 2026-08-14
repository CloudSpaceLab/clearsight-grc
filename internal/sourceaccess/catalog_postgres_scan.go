//go:build postgres

package sourceaccess

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCatalogRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCatalogRepository(pool *pgxpool.Pool) *PostgresCatalogRepository {
	return &PostgresCatalogRepository{pool: pool}
}

const connectionRevisionColumns = `
	sc.revision_id::text,
	sc.connection_id::text,
	sc.tenant_id::text,
	sc.source_id::text,
	sc.code,
	sc.name,
	sc.adapter_kind,
	sc.adapter_version,
	sc.secret_ref,
	sc.definition,
	sc.declared_capabilities,
	sc.verified_capabilities,
	COALESCE(sc.owner_principal_id::text,''),
	sc.status,
	sc.is_current,
	sc.effective_from,
	sc.effective_until,
	sc.version,
	COALESCE(sc.created_by::text,''),
	sc.created_at,
	sc.updated_at`

const viewRevisionColumns = `
	sv.revision_id::text,
	sv.view_id::text,
	sv.tenant_id::text,
	sv.source_id::text,
	sv.connection_id::text,
	sv.connection_version,
	sv.code,
	sv.name,
	sv.definition,
	sv.output_kind,
	sv.stable_keys,
	sv.native_schema,
	sv.schema_fingerprint,
	sv.status,
	sv.is_current,
	sv.effective_from,
	sv.effective_until,
	sv.version,
	COALESCE(sv.created_by::text,''),
	sv.created_at,
	sv.updated_at`

const bindingRevisionColumns = `
	sb.revision_id::text,
	sb.binding_id::text,
	sb.tenant_id::text,
	sb.source_id::text,
	sb.view_id::text,
	sb.view_version,
	sb.code,
	sb.name,
	sb.purpose,
	sb.operations,
	sb.selected_fields,
	sb.key_fields,
	sb.page_rows,
	sb.response_bytes,
	sb.lookup_values,
	sb.timeout_ms,
	sb.mapping,
	sb.parameter_schema,
	sb.output_schema,
	sb.required_freshness_minutes,
	sb.completeness,
	sb.sensitivity_handling,
	sb.status,
	sb.is_current,
	sb.effective_from,
	sb.effective_until,
	sb.version,
	COALESCE(sb.created_by::text,''),
	sb.created_at,
	sb.updated_at`

type catalogScanner interface {
	Scan(...any) error
}

func scanConnectionRevision(row catalogScanner) (ConnectionRevision, error) {
	var value ConnectionRevision
	var definition, declared, verified []byte
	var effectiveFrom, effectiveUntil sql.NullTime
	if err := row.Scan(
		&value.RevisionID,
		&value.ConnectionID,
		&value.TenantID,
		&value.SourceID,
		&value.Code,
		&value.Name,
		&value.AdapterKind,
		&value.AdapterVersion,
		&value.SecretRef,
		&definition,
		&declared,
		&verified,
		&value.OwnerPrincipalID,
		&value.Status,
		&value.IsCurrent,
		&effectiveFrom,
		&effectiveUntil,
		&value.Version,
		&value.CreatedBy,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return ConnectionRevision{}, catalogReadError(err)
	}
	value.Definition = append(json.RawMessage(nil), definition...)
	if err := json.Unmarshal(declared, &value.DeclaredCapabilities); err != nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	if err := json.Unmarshal(verified, &value.VerifiedCapabilities); err != nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	setLifecycleTimes(&value.RevisionLifecycle, effectiveFrom, effectiveUntil)
	normalized, err := normalizeConnectionRevision(value)
	if err != nil {
		return ConnectionRevision{}, ErrCatalogStorage
	}
	return normalized, nil
}

func scanViewRevision(row catalogScanner) (ViewRevision, error) {
	var value ViewRevision
	var definition, stableKeys, nativeSchema []byte
	var effectiveFrom, effectiveUntil sql.NullTime
	if err := row.Scan(
		&value.RevisionID,
		&value.ViewID,
		&value.TenantID,
		&value.SourceID,
		&value.ConnectionID,
		&value.ConnectionVersion,
		&value.Code,
		&value.Name,
		&definition,
		&value.OutputKind,
		&stableKeys,
		&nativeSchema,
		&value.SchemaFingerprint,
		&value.Status,
		&value.IsCurrent,
		&effectiveFrom,
		&effectiveUntil,
		&value.Version,
		&value.CreatedBy,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return ViewRevision{}, catalogReadError(err)
	}
	value.Definition = append(json.RawMessage(nil), definition...)
	if err := json.Unmarshal(stableKeys, &value.StableKeys); err != nil {
		return ViewRevision{}, ErrCatalogStorage
	}
	if err := json.Unmarshal(nativeSchema, &value.NativeSchema); err != nil {
		return ViewRevision{}, ErrCatalogStorage
	}
	setLifecycleTimes(&value.RevisionLifecycle, effectiveFrom, effectiveUntil)
	normalized, err := normalizeViewRevision(value)
	if err != nil {
		return ViewRevision{}, ErrCatalogStorage
	}
	return normalized, nil
}

func scanBindingRevision(row catalogScanner) (BindingRevision, error) {
	var value BindingRevision
	var operations, selectedFields, keyFields []byte
	var mapping, parameterSchema, outputSchema, sensitivityHandling []byte
	var timeoutMS int64
	var effectiveFrom, effectiveUntil sql.NullTime
	if err := row.Scan(
		&value.RevisionID,
		&value.BindingID,
		&value.TenantID,
		&value.SourceID,
		&value.ViewID,
		&value.ViewVersion,
		&value.Code,
		&value.Name,
		&value.Purpose,
		&operations,
		&selectedFields,
		&keyFields,
		&value.Limits.PageRows,
		&value.Limits.ResponseBytes,
		&value.Limits.LookupValues,
		&timeoutMS,
		&mapping,
		&parameterSchema,
		&outputSchema,
		&value.RequiredFreshnessMinutes,
		&value.Completeness,
		&sensitivityHandling,
		&value.Status,
		&value.IsCurrent,
		&effectiveFrom,
		&effectiveUntil,
		&value.Version,
		&value.CreatedBy,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return BindingRevision{}, catalogReadError(err)
	}
	if timeoutMS < 1 {
		return BindingRevision{}, ErrCatalogStorage
	}
	value.Limits.Timeout = time.Duration(timeoutMS) * time.Millisecond
	if err := json.Unmarshal(operations, &value.Operations); err != nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	if err := json.Unmarshal(selectedFields, &value.SelectedFields); err != nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	if err := json.Unmarshal(keyFields, &value.KeyFields); err != nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	value.Mapping = append(json.RawMessage(nil), mapping...)
	value.ParameterSchema = append(json.RawMessage(nil), parameterSchema...)
	value.OutputSchema = append(json.RawMessage(nil), outputSchema...)
	value.SensitivityHandling = append(json.RawMessage(nil), sensitivityHandling...)
	setLifecycleTimes(&value.RevisionLifecycle, effectiveFrom, effectiveUntil)
	normalized, err := normalizeBindingRevision(value)
	if err != nil {
		return BindingRevision{}, ErrCatalogStorage
	}
	return normalized, nil
}

func setLifecycleTimes(value *RevisionLifecycle, effectiveFrom, effectiveUntil sql.NullTime) {
	if effectiveFrom.Valid {
		value.EffectiveFrom = cloneTime(&effectiveFrom.Time)
	}
	if effectiveUntil.Valid {
		value.EffectiveUntil = cloneTime(&effectiveUntil.Time)
	}
}

func catalogReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCatalogNotFound
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && (databaseError.Code == "22P02" || databaseError.Code == "22023") {
		return ErrCatalogInvalid
	}
	return ErrCatalogStorage
}

func catalogWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCatalogNotFound
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "23505":
			return ErrCatalogConflict
		case "23503":
			return ErrCatalogNotFound
		case "23514", "22P02", "22023":
			return ErrCatalogInvalid
		}
	}
	return ErrCatalogStorage
}

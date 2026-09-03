//go:build postgres

package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresExportRepository struct{ pool *pgxpool.Pool }

func NewPostgresExportRepository(pool *pgxpool.Pool) *PostgresExportRepository {
	return &PostgresExportRepository{pool: pool}
}

func (r *PostgresExportRepository) CreateExport(ctx context.Context, receipt ExportReceipt) (ExportReceipt, error) {
	if r == nil || r.pool == nil || receipt.TenantID == "" || receipt.RequestedBy == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	filter, err := json.Marshal(receipt.Filter)
	if err != nil {
		return ExportReceipt{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ExportReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		WITH scope AS (
			SELECT t.id tenant_id,
			       CASE WHEN $2='' THEN NULL ELSE (
				SELECT le.id FROM legal_entities le
				WHERE le.tenant_id=t.id AND (le.id::text=$2 OR le.code=$2)
				  AND le.valid_until IS NULL
				ORDER BY le.valid_from DESC LIMIT 1
			) END legal_entity_id
			FROM tenants t WHERE t.id::text=$1 OR t.slug=$1 LIMIT 1
		)
		INSERT INTO audit_export_receipts(
			tenant_id,legal_entity_id,requested_by_ref,format,filter,as_of,status,expires_at
		)
		SELECT tenant_id,legal_entity_id,$3,$4,$5::jsonb,$6,'GENERATING',$7
		FROM scope WHERE $2='' OR legal_entity_id IS NOT NULL
		RETURNING id::text,created_at`,
		receipt.TenantID, receipt.LegalEntityID, receipt.RequestedBy, receipt.Format, filter, receipt.AsOf, receipt.ExpiresAt,
	)
	if err := row.Scan(&receipt.ID, &receipt.CreatedAt); errors.Is(err, pgx.ErrNoRows) {
		return ExportReceipt{}, ErrExportInvalid
	} else if err != nil {
		return ExportReceipt{}, err
	}
	receipt.Status = "GENERATING"
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		SELECT tenant_id,'AUDIT_EXPORT',id,'AUDIT_EXPORT_REQUESTED',
		       jsonb_strip_nulls(jsonb_build_object(
		         'actor_id',requested_by_ref,
		         'legal_entity_id',legal_entity_id::text,
		         'format',format,
		         'as_of',as_of
		       )),created_at,created_at
		FROM audit_export_receipts WHERE id=$1::uuid`, receipt.ID); err != nil {
		return ExportReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExportReceipt{}, err
	}
	return receipt, nil
}

func (r *PostgresExportRepository) CompleteExport(ctx context.Context, receipt ExportReceipt) (ExportReceipt, error) {
	if r == nil || r.pool == nil || receipt.TenantID == "" || receipt.ID == "" || receipt.CompletedAt == nil {
		return ExportReceipt{}, ErrExportInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ExportReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		UPDATE audit_export_receipts ae
		SET status='READY',row_count=$3,data_object_key=$4,data_sha256=$5,
		    manifest_object_key=$6,manifest_sha256=$7,completed_at=$8,failure_code=NULL
		FROM tenants t
		WHERE ae.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND ae.id::text=$2 AND ae.status='GENERATING'
		RETURNING ae.created_at,ae.expires_at`,
		receipt.TenantID, receipt.ID, receipt.RowCount, receipt.DataObjectKey, receipt.DataSHA256,
		receipt.ManifestKey, receipt.ManifestSHA256, *receipt.CompletedAt,
	)
	if err := row.Scan(&receipt.CreatedAt, &receipt.ExpiresAt); errors.Is(err, pgx.ErrNoRows) {
		return ExportReceipt{}, ErrNotFound
	} else if err != nil {
		return ExportReceipt{}, err
	}
	receipt.Status = ExportStatusReady
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		SELECT tenant_id,'AUDIT_EXPORT',id,'AUDIT_EXPORT_COMPLETED',
		       jsonb_strip_nulls(jsonb_build_object(
		         'actor_id',requested_by_ref,
		         'legal_entity_id',legal_entity_id::text,
		         'format',format,
		         'row_count',row_count,
		         'data_sha256',data_sha256
		       )),completed_at,completed_at
		FROM audit_export_receipts WHERE id=$1::uuid`, receipt.ID); err != nil {
		return ExportReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExportReceipt{}, err
	}
	return receipt, nil
}

func (r *PostgresExportRepository) FailExport(ctx context.Context, tenantID, exportID, failureCode string) error {
	if r == nil || r.pool == nil || tenantID == "" || exportID == "" || failureCode == "" {
		return ErrExportInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var completedAt any
	err = tx.QueryRow(ctx, `
		UPDATE audit_export_receipts ae
		SET status='FAILED',failure_code=$3,completed_at=clock_timestamp()
		FROM tenants t
		WHERE ae.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND ae.id::text=$2 AND ae.status='GENERATING'
		RETURNING ae.completed_at`, tenantID, exportID, failureCode).Scan(&completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		SELECT tenant_id,'AUDIT_EXPORT',id,'AUDIT_EXPORT_FAILED',
		       jsonb_strip_nulls(jsonb_build_object(
		         'actor_id',requested_by_ref,
		         'legal_entity_id',legal_entity_id::text,
		         'format',format,
		         'failure_code',failure_code
		       )),completed_at,completed_at
		FROM audit_export_receipts WHERE id=$1::uuid`, exportID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresExportRepository) GetExport(ctx context.Context, tenantID, exportID string) (ExportReceipt, error) {
	if r == nil || r.pool == nil || tenantID == "" || exportID == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	row := r.pool.QueryRow(ctx, `
		SELECT ae.id::text,t.slug,COALESCE(ae.legal_entity_id::text,''),ae.requested_by_ref,ae.format,
		       ae.filter,ae.as_of,ae.status,ae.row_count,COALESCE(ae.data_object_key,''),
		       COALESCE(ae.data_sha256,''),COALESCE(ae.manifest_object_key,''),COALESCE(ae.manifest_sha256,''),
		       COALESCE(ae.failure_code,''),ae.created_at,ae.completed_at,ae.expires_at
		FROM audit_export_receipts ae JOIN tenants t ON t.id=ae.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ae.id::text=$2`, tenantID, exportID)
	return scanExportReceipt(row)
}

func scanExportReceipt(row pgx.Row) (ExportReceipt, error) {
	var receipt ExportReceipt
	var filter []byte
	if err := row.Scan(
		&receipt.ID, &receipt.TenantID, &receipt.LegalEntityID, &receipt.RequestedBy, &receipt.Format,
		&filter, &receipt.AsOf, &receipt.Status, &receipt.RowCount, &receipt.DataObjectKey,
		&receipt.DataSHA256, &receipt.ManifestKey, &receipt.ManifestSHA256, &receipt.FailureCode,
		&receipt.CreatedAt, &receipt.CompletedAt, &receipt.ExpiresAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return ExportReceipt{}, ErrNotFound
	} else if err != nil {
		return ExportReceipt{}, err
	}
	if err := json.Unmarshal(filter, &receipt.Filter); err != nil {
		return ExportReceipt{}, fmt.Errorf("decode audit export filter: %w", err)
	}
	return receipt, nil
}

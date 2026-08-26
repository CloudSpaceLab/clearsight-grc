//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetVendorForBrandDiscovery(ctx context.Context, tenantID, vendorID string) (Vendor, error) {
	value, err := scanVendor(r.pool.QueryRow(ctx, `
		SELECT `+vendorProjection+`
		FROM third_parties p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2`, tenantID, vendorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	if err != nil {
		return Vendor{}, fmt.Errorf("get vendor for brand discovery: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) ClaimVendorBrandJobs(ctx context.Context, workerID string, now time.Time, lease time.Duration, maxAttempts, limit int) ([]VendorBrandJob, error) {
	if workerID == "" || lease <= 0 || maxAttempts < 1 || maxAttempts > 20 {
		return nil, ErrInvalid
	}
	limit = boundedVendorBrandJobLimit(limit)
	now = now.UTC()
	expires := now.Add(lease)
	if _, err := r.pool.Exec(ctx, `
		WITH exhausted AS (
			SELECT id FROM third_party_vendor_brand_jobs
			WHERE job_type='DISCOVER_ICON' AND attempts >= $2
			  AND (state='READY' OR (state='LEASED' AND lease_expires_at<=$1))
			ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $3
		)
		UPDATE third_party_vendor_brand_jobs job
		SET state='FAILED',lease_token=NULL,lease_expires_at=NULL,last_failure_code=$4,updated_at=$1,version=version+1
		FROM exhausted WHERE job.id=exhausted.id`, now, maxAttempts, limit, VendorBrandFailureAttemptsExhausted); err != nil {
		return nil, fmt.Errorf("terminalize exhausted vendor brand jobs: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM third_party_vendor_brand_jobs
			WHERE job_type='DISCOVER_ICON' AND available_at<=$1 AND attempts<$4
			  AND (state='READY' OR (state='LEASED' AND lease_expires_at<=$1))
			ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $2
		), claimed AS (
			UPDATE third_party_vendor_brand_jobs job
			SET state='LEASED',attempts=job.attempts+1,lease_token=uuidv7(),lease_expires_at=$3,updated_at=$1,version=version+1
			FROM candidates WHERE job.id=candidates.id RETURNING job.*
		)
		SELECT `+vendorBrandJobProjection+` FROM claimed j JOIN tenants t ON t.id=j.tenant_id ORDER BY j.available_at,j.id`, now, limit, expires, maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("claim vendor brand jobs: %w", err)
	}
	defer rows.Close()
	values := make([]VendorBrandJob, 0, limit)
	for rows.Next() {
		value, scanErr := scanVendorBrandJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) CompleteVendorBrandJob(ctx context.Context, claim VendorBrandJob, asset VendorBrandAsset, at time.Time) (VendorBrandAsset, error) {
	at = at.UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("begin vendor brand completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, claim.TenantID)
	if err != nil {
		return VendorBrandAsset{}, err
	}
	current, err := scanVendorBrandJob(tx.QueryRow(ctx, `
		SELECT `+vendorBrandJobProjection+`
		FROM third_party_vendor_brand_jobs j JOIN tenants t ON t.id=j.tenant_id
		WHERE j.id::text=$1 AND j.tenant_id=$2::uuid AND j.vendor_id::text=$3
		  AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>$5
		FOR UPDATE OF j`, claim.ID, tenantID, claim.VendorID, claim.LeaseToken, at))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, ErrVendorBrandJobLeaseLost
	}
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("lock vendor brand job: %w", err)
	}
	var vendorVersion int64
	var websiteDomain WebsiteDomain
	err = tx.QueryRow(ctx, `
		SELECT version,COALESCE(website_domain,'') FROM third_parties
		WHERE tenant_id=$1::uuid AND id::text=$2 FOR UPDATE`, tenantID, claim.VendorID).Scan(&vendorVersion, &websiteDomain)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, ErrVendorBrandJobStale
	}
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("lock vendor for brand completion: %w", err)
	}
	if vendorVersion != current.VendorVersion || websiteDomain != current.WebsiteDomain {
		return VendorBrandAsset{}, ErrVendorBrandJobStale
	}
	if _, err := tx.Exec(ctx, `
		UPDATE third_party_vendor_brand_assets
		SET state='SUPERSEDED',updated_at=$4,version=version+1
		WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND source_kind='DISCOVERED' AND state='CURRENT' AND id<>$3::uuid`,
		tenantID, claim.VendorID, asset.ID, at); err != nil {
		return VendorBrandAsset{}, fmt.Errorf("supersede vendor brand asset: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_vendor_brand_assets(
			id,tenant_id,vendor_id,source_kind,state,source_domain,artifact_key,source_digest,media_type,
			pixel_width,pixel_height,byte_size,retrieved_at,next_refresh_at,approved_by_principal_id,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DISCOVERED','CURRENT',$4,$5,$6,'image/png',$7,$8,$9,$10,$11,NULL,$10,$10,1)`,
		asset.ID, tenantID, claim.VendorID, asset.SourceDomain, asset.ArtifactKey, asset.SourceDigest,
		asset.PixelWidth, asset.PixelHeight, asset.ByteSize, at, asset.NextRefreshAt)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("store vendor brand asset: %w", err)
	}
	if err := appendVendorBrandDiscoveredEvent(ctx, tx, tenantID, claim, asset, at); err != nil {
		return VendorBrandAsset{}, err
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE third_party_vendor_brand_jobs
		SET state='COMPLETED',lease_token=NULL,lease_expires_at=NULL,last_failure_code='',updated_at=$5,version=version+1
		WHERE id::text=$1 AND tenant_id=$2::uuid AND vendor_id::text=$3 AND state='LEASED' AND lease_token::text=$4`,
		claim.ID, tenantID, claim.VendorID, claim.LeaseToken, at)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("complete vendor brand job: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return VendorBrandAsset{}, ErrVendorBrandJobLeaseLost
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorBrandAsset{}, fmt.Errorf("commit vendor brand completion: %w", err)
	}
	return asset, nil
}

func appendVendorBrandDiscoveredEvent(ctx context.Context, tx pgx.Tx, tenantID string, claim VendorBrandJob, asset VendorBrandAsset, at time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'VENDOR_BRAND',$2::uuid,$3,NULL,$4,
			jsonb_build_object('asset_id',$5::text,'asset_version',$6::bigint,'vendor_version',$7::bigint,'artifact_key',$8::text,'source_digest',$9::text,'media_type','image/png','pixel_width',$10::integer,'pixel_height',$11::integer,'byte_size',$12::bigint),$13)`,
		tenantID, claim.VendorID, claim.Version, VendorBrandDiscoveredEvent, asset.ID, asset.Version, claim.VendorVersion,
		asset.ArtifactKey, asset.SourceDigest, asset.PixelWidth, asset.PixelHeight, asset.ByteSize, at)
	if err != nil {
		return fmt.Errorf("append vendor brand event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'VENDOR_BRAND',$2::uuid,$3,
			jsonb_build_object('asset_id',$4::text,'asset_version',$5::bigint,'vendor_id',$6::text,'vendor_version',$7::bigint,'artifact_key',$8::text,'source_digest',$9::text,'media_type','image/png','pixel_width',$10::integer,'pixel_height',$11::integer,'byte_size',$12::bigint),$13,$13)`,
		tenantID, claim.VendorID, VendorBrandDiscoveredEvent, asset.ID, asset.Version, claim.VendorID, claim.VendorVersion,
		asset.ArtifactKey, asset.SourceDigest, asset.PixelWidth, asset.PixelHeight, asset.ByteSize, at)
	if err != nil {
		return fmt.Errorf("append vendor brand outbox event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CancelVendorBrandJob(ctx context.Context, claim VendorBrandJob, code string, at time.Time) error {
	if !validVendorBrandFailureCode(code) {
		return ErrInvalid
	}
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE third_party_vendor_brand_jobs j
		SET state='CANCELLED',website_domain='',lease_token=NULL,lease_expires_at=NULL,last_failure_code=$5,updated_at=$6,version=version+1
		FROM tenants t
		WHERE j.id::text=$1 AND j.tenant_id=t.id AND (t.id::text=$2 OR t.slug=$2) AND j.vendor_id::text=$3 AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>$6`,
		claim.ID, claim.TenantID, claim.VendorID, claim.LeaseToken, code, at.UTC())
	if err != nil {
		return fmt.Errorf("cancel stale vendor brand job: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return ErrVendorBrandJobLeaseLost
	}
	return nil
}

func (r *PostgresRepository) FailVendorBrandJob(ctx context.Context, claim VendorBrandJob, maxAttempts int, code string, at, availableAt time.Time) (VendorBrandJob, error) {
	if maxAttempts < 1 || maxAttempts > 20 || !validVendorBrandFailureCode(code) || availableAt.Before(at) {
		return VendorBrandJob{}, ErrInvalid
	}
	value, err := scanVendorBrandJob(r.pool.QueryRow(ctx, `
		UPDATE third_party_vendor_brand_jobs j
		SET state=CASE WHEN attempts >= $6 THEN 'FAILED' ELSE 'READY' END,
			available_at=CASE WHEN attempts >= $6 THEN available_at ELSE $7 END,
			lease_token=NULL,lease_expires_at=NULL,last_failure_code=$8,updated_at=$9,version=version+1
		FROM tenants t
		WHERE j.id::text=$1 AND j.tenant_id=t.id AND (t.id::text=$2 OR t.slug=$2) AND j.vendor_id::text=$3
		  AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>$9 AND j.job_type=$5
		RETURNING `+vendorBrandJobProjection,
		claim.ID, claim.TenantID, claim.VendorID, claim.LeaseToken, claim.JobType, maxAttempts, availableAt.UTC(), code, at.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandJob{}, ErrVendorBrandJobLeaseLost
	}
	if err != nil {
		return VendorBrandJob{}, fmt.Errorf("release vendor brand job: %w", err)
	}
	return value, nil
}

var _ VendorBrandWorkerRepository = (*PostgresRepository)(nil)

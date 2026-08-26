//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
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

func (r *PostgresRepository) ClaimVendorBrandJobs(ctx context.Context, workerID string, _ time.Time, lease time.Duration, maxAttempts, limit int) ([]VendorBrandJob, error) {
	if workerID == "" || lease <= 0 || maxAttempts < 1 || maxAttempts > 20 {
		return nil, ErrInvalid
	}
	limit = boundedVendorBrandJobLimit(limit)
	if _, err := r.pool.Exec(ctx, `
		WITH due AS (
			SELECT job.id
			FROM third_party_vendor_brand_assets asset
			JOIN third_party_vendor_brand_jobs job ON job.tenant_id=asset.tenant_id AND job.vendor_id=asset.vendor_id
			JOIN third_parties vendor ON vendor.tenant_id=asset.tenant_id AND vendor.id=asset.vendor_id
			WHERE asset.source_kind='DISCOVERED' AND asset.state='CURRENT' AND asset.next_refresh_at<=clock_timestamp()
			  AND asset.source_domain=vendor.website_domain AND vendor.website_domain IS NOT NULL
			  AND job.job_type='DISCOVER_ICON' AND job.state='COMPLETED'
			ORDER BY asset.next_refresh_at,asset.tenant_id,asset.vendor_id
			FOR UPDATE OF asset,job SKIP LOCKED LIMIT $1
		)
		UPDATE third_party_vendor_brand_jobs job
		SET state='READY',vendor_version=vendor.version,website_domain=vendor.website_domain,
			attempts=0,available_at=clock_timestamp(),lease_token=NULL,lease_expires_at=NULL,last_failure_code='',updated_at=clock_timestamp(),version=job.version+1
		FROM third_parties vendor,due
		WHERE job.id=due.id AND job.tenant_id=vendor.tenant_id AND job.vendor_id=vendor.id`, limit); err != nil {
		return nil, fmt.Errorf("requeue due vendor brand refreshes: %w", err)
	}
	if _, err := r.pool.Exec(ctx, `
		WITH exhausted AS (
			SELECT id FROM third_party_vendor_brand_jobs
			WHERE job_type='DISCOVER_ICON' AND attempts >= $1
			  AND (state='READY' OR (state='LEASED' AND lease_expires_at<=clock_timestamp()))
			ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $2
		)
		UPDATE third_party_vendor_brand_jobs job
		SET state='FAILED',lease_token=NULL,lease_expires_at=NULL,last_failure_code=$3,updated_at=clock_timestamp(),version=version+1
		FROM exhausted WHERE job.id=exhausted.id`, maxAttempts, limit, VendorBrandFailureAttemptsExhausted); err != nil {
		return nil, fmt.Errorf("terminalize exhausted vendor brand jobs: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM third_party_vendor_brand_jobs
			WHERE job_type='DISCOVER_ICON' AND available_at<=clock_timestamp() AND attempts<$3
			  AND (state='READY' OR (state='LEASED' AND lease_expires_at<=clock_timestamp()))
			ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $1
		), claimed AS (
			UPDATE third_party_vendor_brand_jobs job
			SET state='LEASED',attempts=job.attempts+1,lease_token=uuidv7(),
				lease_expires_at=clock_timestamp()+($2::double precision * interval '1 second'),updated_at=clock_timestamp(),version=version+1
			FROM candidates WHERE job.id=candidates.id RETURNING job.*
		)
		SELECT `+vendorBrandJobProjection+` FROM claimed j JOIN tenants t ON t.id=j.tenant_id ORDER BY j.available_at,j.id`, limit, lease.Seconds(), maxAttempts)
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

func (r *PostgresRepository) CompleteVendorBrandJob(ctx context.Context, claim VendorBrandJob, asset VendorBrandAsset, _ time.Time) (VendorBrandAsset, error) {
	if !validVendorBrandAssetCompletion(claim, asset) {
		return VendorBrandAsset{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("begin vendor brand completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, claim.TenantID)
	if err != nil {
		return VendorBrandAsset{}, err
	}
	// Vendor identity commands lock third_parties before their discovery job.
	// Completion must keep the same order so the two transactions cannot wait
	// on each other's row locks.
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
	current, err := scanVendorBrandJob(tx.QueryRow(ctx, `
		SELECT `+vendorBrandJobProjection+`
		FROM third_party_vendor_brand_jobs j JOIN tenants t ON t.id=j.tenant_id
		WHERE j.id::text=$1 AND j.tenant_id=$2::uuid AND j.vendor_id::text=$3
		  AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>clock_timestamp()
		FOR UPDATE OF j`, claim.ID, tenantID, claim.VendorID, claim.LeaseToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, ErrVendorBrandJobLeaseLost
	}
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("lock vendor brand job: %w", err)
	}
	if vendorVersion != current.VendorVersion || websiteDomain != current.WebsiteDomain {
		return VendorBrandAsset{}, ErrVendorBrandJobStale
	}
	var completionAt time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&completionAt); err != nil {
		return VendorBrandAsset{}, fmt.Errorf("read vendor brand completion time: %w", err)
	}
	completionAt = completionAt.UTC()
	nextRefresh := completionAt.Add(defaultVendorBrandRefresh)
	if asset.NextRefreshAt != nil {
		nextRefresh = completionAt.Add(asset.NextRefreshAt.Sub(*asset.RetrievedAt))
	}
	asset.RetrievedAt = &completionAt
	asset.NextRefreshAt = &nextRefresh
	asset.CreatedAt = completionAt
	asset.UpdatedAt = completionAt
	if _, err := tx.Exec(ctx, `
		UPDATE third_party_vendor_brand_assets
		SET state='SUPERSEDED',updated_at=$4,version=version+1
		WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND source_kind='DISCOVERED' AND state='CURRENT' AND id<>$3::uuid`,
		tenantID, claim.VendorID, asset.ID, completionAt); err != nil {
		return VendorBrandAsset{}, fmt.Errorf("supersede vendor brand asset: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_vendor_brand_assets(
			id,tenant_id,vendor_id,source_kind,state,source_domain,artifact_key,source_digest,media_type,
			pixel_width,pixel_height,byte_size,retrieved_at,next_refresh_at,approved_by_principal_id,created_at,updated_at,version,asset_token
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DISCOVERED','CURRENT',$4,$5,$6,'image/png',$7,$8,$9,$10,$11,NULL,$10,$10,1,$12)`,
		asset.ID, tenantID, claim.VendorID, asset.SourceDomain, asset.ArtifactKey, asset.SourceDigest,
		asset.PixelWidth, asset.PixelHeight, asset.ByteSize, completionAt, asset.NextRefreshAt, asset.AssetToken)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("store vendor brand asset: %w", err)
	}
	eventVersion, err := nextVendorBrandEventVersionPG(ctx, tx, tenantID, claim.VendorID)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("allocate vendor brand event version: %w", err)
	}
	if err := appendVendorBrandDiscoveredEvent(ctx, tx, tenantID, claim, asset, completionAt, eventVersion); err != nil {
		return VendorBrandAsset{}, err
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE third_party_vendor_brand_jobs
		SET state='COMPLETED',lease_token=NULL,lease_expires_at=NULL,last_failure_code='',updated_at=clock_timestamp(),version=version+1
		WHERE id::text=$1 AND tenant_id=$2::uuid AND vendor_id::text=$3 AND state='LEASED'
		  AND lease_token::text=$4 AND lease_expires_at>clock_timestamp()`,
		claim.ID, tenantID, claim.VendorID, claim.LeaseToken)
	if err != nil {
		return VendorBrandAsset{}, fmt.Errorf("complete vendor brand job: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return VendorBrandAsset{}, ErrVendorBrandJobLeaseLost
	}
	if err := tx.Commit(ctx); err != nil {
		commitErr := fmt.Errorf("commit vendor brand completion: %w", err)
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.vendorBrandCompletionRecorded(probeCtx, claim, asset)
		if confirmed {
			return asset, nil
		}
		if probeErr != nil {
			return VendorBrandAsset{}, errors.Join(commitErr, fmt.Errorf("probe vendor brand completion: %w", probeErr))
		}
		return VendorBrandAsset{}, commitErr
	}
	return asset, nil
}

func (r *PostgresRepository) vendorBrandCompletionRecorded(ctx context.Context, claim VendorBrandJob, asset VendorBrandAsset) (bool, error) {
	var recorded bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM third_party_vendor_brand_assets brand
			JOIN tenants tenant ON tenant.id=brand.tenant_id
			JOIN third_party_vendor_brand_jobs job ON job.tenant_id=brand.tenant_id AND job.vendor_id=brand.vendor_id AND job.id::text=$1
			JOIN third_party_events event ON event.tenant_id=brand.tenant_id
			  AND event.aggregate_type='VENDOR_BRAND' AND event.aggregate_id=brand.vendor_id
			  AND event.event_type=$12 AND event.payload->>'asset_id'=brand.id::text
			JOIN outbox_events outbox ON outbox.tenant_id=brand.tenant_id
			  AND outbox.aggregate_type='VENDOR_BRAND' AND outbox.aggregate_id=brand.vendor_id
			  AND outbox.event_type=$12 AND outbox.payload->>'asset_id'=brand.id::text
			WHERE (tenant.id::text=$2 OR tenant.slug=$2) AND brand.vendor_id::text=$3
			  AND brand.id::text=$6 AND brand.source_kind='DISCOVERED'
			  AND brand.artifact_key=$7 AND brand.source_digest=$8 AND brand.source_domain=$5
			  AND brand.media_type='image/png' AND brand.pixel_width=$9 AND brand.pixel_height=$10 AND brand.byte_size=$11
			  AND event.payload->>'vendor_version'=$4::text AND event.payload->>'artifact_key'=$7 AND event.payload->>'source_digest'=$8
			  AND outbox.payload->>'vendor_version'=$4::text AND outbox.payload->>'artifact_key'=$7 AND outbox.payload->>'source_digest'=$8
		)`, claim.ID, claim.TenantID, claim.VendorID, claim.VendorVersion, claim.WebsiteDomain,
		asset.ID, asset.ArtifactKey, asset.SourceDigest, asset.PixelWidth, asset.PixelHeight, asset.ByteSize, VendorBrandDiscoveredEvent).Scan(&recorded)
	return recorded, err
}

func (r *PostgresRepository) VendorBrandQueueHealth(ctx context.Context) (workflowruntime.QueueHealth, error) {
	var health workflowruntime.QueueHealth
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE state IN ('READY','LEASED')),
			count(*) FILTER (WHERE state='FAILED'),
			COALESCE(max(attempts) FILTER (WHERE state IN ('READY','LEASED','FAILED')),0),
			min(available_at) FILTER (WHERE state IN ('READY','LEASED'))
		FROM third_party_vendor_brand_jobs
		WHERE job_type='DISCOVER_ICON'`).Scan(&health.Pending, &health.Terminal, &health.HighestAttempts, &health.OldestPending)
	if err != nil {
		return workflowruntime.QueueHealth{}, fmt.Errorf("read vendor brand queue health: %w", err)
	}
	return health, nil
}

func appendVendorBrandDiscoveredEvent(ctx context.Context, tx pgx.Tx, tenantID string, claim VendorBrandJob, asset VendorBrandAsset, at time.Time, eventVersion int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'VENDOR_BRAND',$2::uuid,$3,NULL,$4,
			jsonb_build_object('asset_id',$5::text,'asset_version',$6::bigint,'vendor_version',$7::bigint,'artifact_key',$8::text,'source_digest',$9::text,'media_type','image/png','pixel_width',$10::integer,'pixel_height',$11::integer,'byte_size',$12::bigint),$13)`,
		tenantID, claim.VendorID, eventVersion, VendorBrandDiscoveredEvent, asset.ID, asset.Version, claim.VendorVersion,
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

func (r *PostgresRepository) CancelVendorBrandJob(ctx context.Context, claim VendorBrandJob, code string, _ time.Time) error {
	if !validVendorBrandFailureCode(code) {
		return ErrInvalid
	}
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE third_party_vendor_brand_jobs j
		SET state='CANCELLED',website_domain='',lease_token=NULL,lease_expires_at=NULL,last_failure_code=$5,updated_at=clock_timestamp(),version=version+1
		FROM tenants t
		WHERE j.id::text=$1 AND j.tenant_id=t.id AND (t.id::text=$2 OR t.slug=$2) AND j.vendor_id::text=$3 AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>clock_timestamp()`,
		claim.ID, claim.TenantID, claim.VendorID, claim.LeaseToken, code)
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
			lease_token=NULL,lease_expires_at=NULL,last_failure_code=$8,updated_at=clock_timestamp(),version=version+1
		FROM tenants t
		WHERE j.id::text=$1 AND j.tenant_id=t.id AND (t.id::text=$2 OR t.slug=$2) AND j.vendor_id::text=$3
		  AND j.state='LEASED' AND j.lease_token::text=$4 AND j.lease_expires_at>clock_timestamp() AND j.job_type=$5
		RETURNING `+vendorBrandJobProjection,
		claim.ID, claim.TenantID, claim.VendorID, claim.LeaseToken, claim.JobType, maxAttempts, availableAt.UTC(), code))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandJob{}, ErrVendorBrandJobLeaseLost
	}
	if err != nil {
		return VendorBrandJob{}, fmt.Errorf("release vendor brand job: %w", err)
	}
	return value, nil
}

var _ VendorBrandWorkerRepository = (*PostgresRepository)(nil)

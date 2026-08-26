//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const vendorProjection = `
	p.id::text,t.slug,p.legal_name,p.trading_name,p.registration_ref,p.jurisdiction,p.source_id,p.external_ref,
	COALESCE(p.website_domain,''),p.status,p.created_at,p.updated_at,p.version`

const vendorBrandJobProjection = `
	j.id::text,t.slug,j.vendor_id::text,j.vendor_version,j.job_type,j.website_domain,j.state,j.attempts,j.available_at,
	COALESCE(j.lease_token::text,''),j.lease_expires_at,j.last_failure_code,j.created_at,j.updated_at,j.version`

const vendorBrandAssetProjection = `
	a.id::text,t.slug,a.vendor_id::text,a.source_kind,a.state,a.source_domain,a.artifact_key,a.source_digest,a.media_type,
	a.pixel_width,a.pixel_height,a.byte_size,a.retrieved_at,a.next_refresh_at,COALESCE(a.approved_by_principal_id::text,''),
	a.created_at,a.updated_at,a.version`

func (r *PostgresRepository) GetVendor(ctx context.Context, scope Scope, vendorID string) (Vendor, error) {
	value, err := scanVendor(r.pool.QueryRow(ctx, `
		SELECT `+vendorProjection+`
		FROM third_parties p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$3
		  AND EXISTS (
			SELECT 1 FROM third_party_relationships relationship
			WHERE relationship.tenant_id=p.tenant_id AND relationship.vendor_id=p.id AND relationship.legal_entity_id::text=$2
		  )`, scope.TenantID, scope.LegalEntityID, vendorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	if err != nil {
		return Vendor{}, fmt.Errorf("get vendor identity: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) UpdateVendorIdentity(ctx context.Context, record UpdateVendorIdentityRecord) (Vendor, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Vendor{}, fmt.Errorf("begin vendor identity update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return Vendor{}, err
	}
	current, err := scanVendor(tx.QueryRow(ctx, `
		SELECT `+vendorProjection+`
		FROM third_parties p JOIN tenants t ON t.id=p.tenant_id
		WHERE p.tenant_id=$1::uuid AND p.id::text=$3
		  AND EXISTS (
			SELECT 1 FROM third_party_relationships relationship
			WHERE relationship.tenant_id=p.tenant_id AND relationship.vendor_id=p.id AND relationship.legal_entity_id::text=$2
		  )
		FOR UPDATE OF p`, tenantID, record.LegalEntityID, record.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrNotFound
	}
	if err != nil {
		return Vendor{}, fmt.Errorf("lock vendor identity: %w", err)
	}
	if current.Version != record.ExpectedVersion {
		return Vendor{}, ErrVersionConflict
	}
	updated := record.Vendor
	updated.ID, updated.TenantID = current.ID, current.TenantID
	updated.SourceID, updated.ExternalRef = current.SourceID, current.ExternalRef
	updated.Status, updated.CreatedAt = current.Status, current.CreatedAt
	err = tx.QueryRow(ctx, `
		UPDATE third_parties
		SET legal_name=$3,trading_name=$4,registration_ref=$5,jurisdiction=$6,website_domain=NULLIF($7,''),updated_at=$8,version=version+1
		WHERE tenant_id=$1::uuid AND id=$2::uuid AND version=$9
		RETURNING version`, tenantID, current.ID, updated.LegalName, updated.TradingName, updated.RegistrationRef,
		updated.Jurisdiction, updated.WebsiteDomain, updated.UpdatedAt, record.ExpectedVersion).Scan(&updated.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vendor{}, ErrVersionConflict
	}
	if err != nil {
		return Vendor{}, fmt.Errorf("update vendor identity: %w", err)
	}
	if record.BrandJob != nil {
		job := *record.BrandJob
		job.TenantID, job.VendorID, job.VendorVersion = updated.TenantID, updated.ID, updated.Version
		if err := storeVendorBrandJob(ctx, tx, tenantID, job); err != nil {
			return Vendor{}, err
		}
	}
	if err := appendVendorIdentityEvent(ctx, tx, tenantID, updated, record.ActorID, VendorIdentityUpdatedEvent); err != nil {
		return Vendor{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Vendor{}, fmt.Errorf("commit vendor identity update: %w", err)
	}
	return updated, nil
}

func (r *PostgresRepository) GetVendorBrandJob(ctx context.Context, scope Scope, vendorID string) (VendorBrandJob, error) {
	value, err := scanVendorBrandJob(r.pool.QueryRow(ctx, `
		SELECT `+vendorBrandJobProjection+`
		FROM third_party_vendor_brand_jobs j JOIN tenants t ON t.id=j.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND j.vendor_id::text=$3
		  AND EXISTS (
			SELECT 1 FROM third_party_relationships relationship
			WHERE relationship.tenant_id=j.tenant_id AND relationship.vendor_id=j.vendor_id AND relationship.legal_entity_id::text=$2
		  )`, scope.TenantID, scope.LegalEntityID, vendorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandJob{}, ErrNotFound
	}
	if err != nil {
		return VendorBrandJob{}, fmt.Errorf("get vendor brand job: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) ListVendorBrandAssets(ctx context.Context, scope Scope, vendorID string) ([]VendorBrandAsset, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+vendorBrandAssetProjection+`
		FROM third_party_vendor_brand_assets a JOIN tenants t ON t.id=a.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.vendor_id::text=$3
		  AND EXISTS (
			SELECT 1 FROM third_party_relationships relationship
			WHERE relationship.tenant_id=a.tenant_id AND relationship.vendor_id=a.vendor_id AND relationship.legal_entity_id::text=$2
		  )
		ORDER BY a.updated_at DESC,a.id DESC`, scope.TenantID, scope.LegalEntityID, vendorID)
	if err != nil {
		return nil, fmt.Errorf("list vendor brand assets: %w", err)
	}
	defer rows.Close()
	values := []VendorBrandAsset{}
	for rows.Next() {
		value, scanErr := scanVendorBrandAsset(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		if _, err := r.GetVendor(ctx, scope, vendorID); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func storeVendorBrandJob(ctx context.Context, tx pgx.Tx, tenantID string, job VendorBrandJob) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_vendor_brand_jobs(
			id,tenant_id,vendor_id,vendor_version,job_type,website_domain,state,attempts,available_at,
			lease_token,lease_expires_at,last_failure_code,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,0,$8,NULL,NULL,'',$8,$8,1)
		ON CONFLICT (tenant_id,vendor_id) DO UPDATE SET
			vendor_version=EXCLUDED.vendor_version,website_domain=EXCLUDED.website_domain,state=EXCLUDED.state,
			attempts=0,available_at=EXCLUDED.available_at,lease_token=NULL,lease_expires_at=NULL,last_failure_code='',
			updated_at=EXCLUDED.updated_at,version=third_party_vendor_brand_jobs.version+1`,
		job.ID, tenantID, job.VendorID, job.VendorVersion, job.JobType, job.WebsiteDomain, job.State, job.AvailableAt)
	if err != nil {
		return fmt.Errorf("schedule vendor brand discovery: %w", err)
	}
	return nil
}

func appendVendorIdentityEvent(ctx context.Context, tx pgx.Tx, tenantID string, vendor Vendor, actorID, eventType string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'VENDOR',$2::uuid,$3,$4::uuid,$5,
			jsonb_build_object(
				'legal_name',$6::text,'trading_name',$7::text,'registration_ref',$8::text,
				'jurisdiction',$9::text,'website_domain',$10::text,'status',$11::text
			),$12)`,
		tenantID, vendor.ID, vendor.Version, actorID, eventType, vendor.LegalName, vendor.TradingName,
		vendor.RegistrationRef, vendor.Jurisdiction, vendor.WebsiteDomain, vendor.Status, vendor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor identity event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'VENDOR',$2::uuid,$3,
			jsonb_build_object(
				'version',$4::bigint,'legal_name',$5::text,'trading_name',$6::text,'registration_ref',$7::text,
				'jurisdiction',$8::text,'website_domain',$9::text,'status',$10::text
			),$11,$11)`,
		tenantID, vendor.ID, eventType, vendor.Version, vendor.LegalName, vendor.TradingName,
		vendor.RegistrationRef, vendor.Jurisdiction, vendor.WebsiteDomain, vendor.Status, vendor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor identity outbox event: %w", err)
	}
	return nil
}

func scanVendor(row rowScanner) (Vendor, error) {
	var value Vendor
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalName, &value.TradingName, &value.RegistrationRef, &value.Jurisdiction,
		&value.SourceID, &value.ExternalRef, &value.WebsiteDomain, &value.Status, &value.CreatedAt, &value.UpdatedAt, &value.Version,
	)
	return value, err
}

func scanVendorBrandJob(row rowScanner) (VendorBrandJob, error) {
	var value VendorBrandJob
	err := row.Scan(
		&value.ID, &value.TenantID, &value.VendorID, &value.VendorVersion, &value.JobType, &value.WebsiteDomain,
		&value.State, &value.Attempts, &value.AvailableAt, &value.LeaseToken, &value.LeaseExpiresAt,
		&value.LastFailureCode, &value.CreatedAt, &value.UpdatedAt, &value.Version,
	)
	return value, err
}

func scanVendorBrandAsset(row rowScanner) (VendorBrandAsset, error) {
	var value VendorBrandAsset
	err := row.Scan(
		&value.ID, &value.TenantID, &value.VendorID, &value.SourceKind, &value.State, &value.SourceDomain,
		&value.ArtifactKey, &value.SourceDigest, &value.MediaType, &value.PixelWidth, &value.PixelHeight, &value.ByteSize,
		&value.RetrievedAt, &value.NextRefreshAt, &value.ApprovedByPrincipalID, &value.CreatedAt, &value.UpdatedAt, &value.Version,
	)
	return value, err
}

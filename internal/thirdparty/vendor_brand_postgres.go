//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresRepository) GetVendorBrandProjection(ctx context.Context, scope Scope, vendorID string) (VendorBrandProjection, error) {
	values, err := r.GetVendorBrandProjections(ctx, scope, []string{vendorID})
	if err != nil {
		return VendorBrandProjection{}, err
	}
	value, ok := values[vendorID]
	if !ok {
		return VendorBrandProjection{}, ErrNotFound
	}
	return value, nil
}

func (r *PostgresRepository) GetVendorBrandProjections(ctx context.Context, scope Scope, vendorIDs []string) (map[string]VendorBrandProjection, error) {
	if len(vendorIDs) > 100 {
		return nil, ErrInvalid
	}
	values := make(map[string]VendorBrandProjection, len(vendorIDs))
	if len(vendorIDs) == 0 {
		return values, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text,
			COALESCE((SELECT to_jsonb(a) FROM third_party_vendor_brand_assets a WHERE a.tenant_id=p.tenant_id AND a.vendor_id=p.id AND a.source_kind='APPROVED_OVERRIDE' AND a.state='CURRENT' ORDER BY a.updated_at DESC,a.id DESC LIMIT 1),'null'::jsonb),
			COALESCE((SELECT to_jsonb(a) FROM third_party_vendor_brand_assets a WHERE a.tenant_id=p.tenant_id AND a.vendor_id=p.id AND a.source_kind='DISCOVERED' AND a.state='CURRENT' AND a.source_domain=COALESCE(p.website_domain,'') ORDER BY a.updated_at DESC,a.id DESC LIMIT 1),'null'::jsonb),
			COALESCE((SELECT j.state FROM third_party_vendor_brand_jobs j WHERE j.tenant_id=p.tenant_id AND j.vendor_id=p.id),''),
			COALESCE((SELECT max(e.aggregate_version) FROM third_party_events e WHERE e.tenant_id=p.tenant_id AND e.aggregate_type='VENDOR_BRAND' AND e.aggregate_id=p.id),0)
		FROM third_parties p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=ANY($3::uuid[])
		  AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=p.tenant_id AND rel.vendor_id=p.id AND rel.legal_entity_id::text=$2)`, scope.TenantID, scope.LegalEntityID, vendorIDs)
	if err != nil {
		return nil, fmt.Errorf("get current vendor brand projections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var value VendorBrandProjection
		var approvedJSON, discoveredJSON []byte
		if err := rows.Scan(&value.VendorID, &approvedJSON, &discoveredJSON, &value.JobState, &value.EventVersion); err != nil {
			return nil, err
		}
		value.CurrentApproved, err = decodeVendorBrandAssetJSON(approvedJSON)
		if err != nil {
			return nil, err
		}
		value.CurrentDiscovered, err = decodeVendorBrandAssetJSON(discoveredJSON)
		if err != nil {
			return nil, err
		}
		values[value.VendorID] = value
	}
	return values, rows.Err()
}

func decodeVendorBrandAssetJSON(raw []byte) (*VendorBrandAsset, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value VendorBrandAsset
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	var token struct {
		AssetToken string `json:"asset_token"`
	}
	if err := json.Unmarshal(raw, &token); err != nil {
		return nil, err
	}
	value.AssetToken = token.AssetToken
	return &value, nil
}

const vendorProjection = `
	p.id::text,t.slug,p.legal_name,p.trading_name,p.registration_ref,p.jurisdiction,p.source_id,p.external_ref,
	COALESCE(p.website_domain,''),p.status,p.created_at,p.updated_at,p.version`

const vendorBrandJobProjection = `
	j.id::text,t.slug,j.vendor_id::text,j.vendor_version,j.job_type,j.website_domain,j.state,j.attempts,j.available_at,
	COALESCE(j.lease_token::text,''),j.lease_expires_at,j.last_failure_code,j.created_at,j.updated_at,j.version`

const vendorBrandAssetProjection = `
	a.id::text,t.slug,a.vendor_id::text,a.source_kind,a.state,a.source_domain,a.artifact_key,a.source_digest,a.media_type,
	a.pixel_width,a.pixel_height,a.byte_size,a.retrieved_at,a.next_refresh_at,COALESCE(a.approved_by_principal_id::text,''),
	a.created_at,a.updated_at,a.version,a.asset_token`

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
		commitErr := fmt.Errorf("commit vendor identity update: %w", err)
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.vendorIdentityMutationRecorded(probeCtx, updated, record.ActorID)
		if confirmed {
			return updated, nil
		}
		if probeErr != nil {
			return Vendor{}, errors.Join(commitErr, probeErr)
		}
		return Vendor{}, commitErr
	}
	return updated, nil
}

func (r *PostgresRepository) vendorIdentityMutationRecorded(ctx context.Context, vendor Vendor, actorID string) (bool, error) {
	var confirmed bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM third_parties p JOIN tenants t ON t.id=p.tenant_id
		JOIN third_party_events e ON e.tenant_id=p.tenant_id AND e.aggregate_type='VENDOR' AND e.aggregate_id=p.id AND e.aggregate_version=p.version AND e.event_type=$11 AND e.actor_principal_id::text=$12
		JOIN outbox_events o ON o.tenant_id=p.tenant_id AND o.aggregate_type='VENDOR' AND o.aggregate_id=p.id AND o.event_type=$11 AND o.payload->>'version'=$10::text
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2 AND p.legal_name=$3 AND COALESCE(p.trading_name,'')=$4 AND COALESCE(p.registration_ref,'')=$5
		  AND COALESCE(p.jurisdiction,'')=$6 AND COALESCE(p.website_domain,'')=$7 AND p.status=$8 AND p.updated_at=$9 AND p.version=$10
		  AND e.payload->>'legal_name'=$3 AND e.payload->>'trading_name'=$4 AND e.payload->>'registration_ref'=$5 AND e.payload->>'jurisdiction'=$6 AND e.payload->>'website_domain'=$7 AND e.payload->>'status'=$8
		  AND o.payload->>'legal_name'=$3 AND o.payload->>'trading_name'=$4 AND o.payload->>'registration_ref'=$5 AND o.payload->>'jurisdiction'=$6 AND o.payload->>'website_domain'=$7 AND o.payload->>'status'=$8
	)`, vendor.TenantID, vendor.ID, vendor.LegalName, vendor.TradingName, vendor.RegistrationRef, vendor.Jurisdiction, vendor.WebsiteDomain, vendor.Status, vendor.UpdatedAt, vendor.Version, VendorIdentityUpdatedEvent, actorID).Scan(&confirmed)
	return confirmed, err
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
		ORDER BY a.updated_at DESC,a.id DESC LIMIT 1000`, scope.TenantID, scope.LegalEntityID, vendorID)
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

func (r *PostgresRepository) GetVendorBrandAsset(ctx context.Context, scope Scope, vendorID, token string) (VendorBrandAsset, error) {
	value, err := scanVendorBrandAsset(r.pool.QueryRow(ctx, `SELECT `+vendorBrandAssetProjection+` FROM third_party_vendor_brand_assets a JOIN tenants t ON t.id=a.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND a.vendor_id::text=$3 AND a.asset_token=$4 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=a.tenant_id AND rel.vendor_id=a.vendor_id AND rel.legal_entity_id::text=$2)`, scope.TenantID, scope.LegalEntityID, vendorID, token))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, ErrNotFound
	}
	return value, err
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
		&value.RetrievedAt, &value.NextRefreshAt, &value.ApprovedByPrincipalID, &value.CreatedAt, &value.UpdatedAt, &value.Version, &value.AssetToken,
	)
	return value, err
}

func (r *PostgresRepository) CurrentVendorBrandVersion(ctx context.Context, scope Scope, vendorID string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(max(e.aggregate_version),0) FROM third_party_events e JOIN tenants t ON t.id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND e.aggregate_type='VENDOR_BRAND' AND e.aggregate_id::text=$3 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=e.tenant_id AND rel.vendor_id=e.aggregate_id AND rel.legal_entity_id::text=$2)`, scope.TenantID, scope.LegalEntityID, vendorID).Scan(&version)
	return version, err
}

func (r *PostgresRepository) CanonicalVendorBrandTenantID(ctx context.Context, scope Scope, vendorID string) (string, error) {
	var tenantID string
	err := r.pool.QueryRow(ctx, `SELECT t.id::text FROM tenants t WHERE (t.id::text=$1 OR t.slug=$1) AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=t.id AND rel.vendor_id::text=$3 AND rel.legal_entity_id::text=$2)`, scope.TenantID, scope.LegalEntityID, vendorID).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return tenantID, err
}

func (r *PostgresRepository) VendorBrandCommandReceipt(ctx context.Context, scope Scope, vendorID, idempotencyKey string) (VendorBrandReceipt, error) {
	var receipt VendorBrandReceipt
	receipt.TenantID, receipt.VendorID, receipt.IdempotencyKey = scope.TenantID, vendorID, idempotencyKey
	err := r.pool.QueryRow(ctx, `SELECT receipt.command_type,receipt.expected_brand_version,receipt.result_brand_version FROM third_party_vendor_brand_command_receipts receipt JOIN tenants t ON t.id=receipt.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND receipt.vendor_id::text=$3 AND receipt.idempotency_key=$4 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=receipt.tenant_id AND rel.vendor_id=receipt.vendor_id AND rel.legal_entity_id::text=$2)`, scope.TenantID, scope.LegalEntityID, vendorID, idempotencyKey).Scan(&receipt.Command, &receipt.ExpectedVersion, &receipt.ResultVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandReceipt{}, ErrNotFound
	}
	return receipt, err
}

func (r *PostgresRepository) ReserveApprovedVendorBrand(ctx context.Context, record VendorBrandMutationRecord) error {
	if receipt, receiptErr := r.VendorBrandCommandReceipt(ctx, record.Scope, record.VendorID, record.IdempotencyKey); receiptErr == nil {
		_, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandApproveCommand, record.ExpectedVersion)
		return replayErr
	} else if !errors.Is(receiptErr, ErrNotFound) {
		return receiptErr
	}
	command, err := r.pool.Exec(ctx, `INSERT INTO third_party_vendor_brand_upload_reservations(tenant_id,vendor_id,idempotency_key,expected_brand_version,artifact_key,source_digest,state,created_at,updated_at)
	SELECT p.tenant_id,p.id,$4,$5,$6,$7,'RESERVED',$8,$8 FROM third_parties p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$3 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=p.tenant_id AND rel.vendor_id=p.id AND rel.legal_entity_id::text=$2) AND NOT EXISTS(SELECT 1 FROM third_party_vendor_brand_command_receipts receipt WHERE receipt.tenant_id=p.tenant_id AND receipt.vendor_id=p.id AND receipt.idempotency_key=$4)
	ON CONFLICT(tenant_id,vendor_id,idempotency_key) DO NOTHING`, record.TenantID, record.LegalEntityID, record.VendorID, record.IdempotencyKey, record.ExpectedVersion, record.Asset.ArtifactKey, record.Asset.SourceDigest, record.OccurredAt)
	if err != nil {
		return fmt.Errorf("reserve vendor brand upload: %w", err)
	}
	if command.RowsAffected() == 0 {
		if receipt, receiptErr := r.VendorBrandCommandReceipt(ctx, record.Scope, record.VendorID, record.IdempotencyKey); receiptErr == nil {
			_, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandApproveCommand, record.ExpectedVersion)
			return replayErr
		} else if !errors.Is(receiptErr, ErrNotFound) {
			return receiptErr
		}
		var matches bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_vendor_brand_upload_reservations q JOIN tenants t ON t.id=q.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND q.vendor_id::text=$2 AND q.idempotency_key=$3 AND q.expected_brand_version=$4 AND q.artifact_key=$5 AND q.source_digest=$6)`, record.TenantID, record.VendorID, record.IdempotencyKey, record.ExpectedVersion, record.Asset.ArtifactKey, record.Asset.SourceDigest).Scan(&matches)
		if err != nil {
			return err
		}
		if !matches {
			return ErrVersionConflict
		}
	}
	return nil
}

func (r *PostgresRepository) PutApprovedVendorBrand(ctx context.Context, record VendorBrandMutationRecord) (VendorBrandAsset, int64, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	var receipt VendorBrandReceipt
	receipt.TenantID, receipt.VendorID, receipt.IdempotencyKey = record.TenantID, record.VendorID, record.IdempotencyKey
	receiptErr := tx.QueryRow(ctx, `SELECT command_type,expected_brand_version,result_brand_version FROM third_party_vendor_brand_command_receipts WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3`, tenantID, record.VendorID, record.IdempotencyKey).Scan(&receipt.Command, &receipt.ExpectedVersion, &receipt.ResultVersion)
	if receiptErr == nil {
		version, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandApproveCommand, record.ExpectedVersion)
		return record.Asset, version, replayErr
	}
	if !errors.Is(receiptErr, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, receiptErr
	}
	var lockedVendor string
	err = tx.QueryRow(ctx, `SELECT p.id::text FROM third_parties p WHERE p.tenant_id=$1::uuid AND p.id::text=$3 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=p.tenant_id AND rel.vendor_id=p.id AND rel.legal_entity_id::text=$2) FOR UPDATE`, tenantID, record.LegalEntityID, record.VendorID).Scan(&lockedVendor)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, ErrNotFound
	}
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	receipt = VendorBrandReceipt{TenantID: record.TenantID, VendorID: record.VendorID, IdempotencyKey: record.IdempotencyKey}
	receiptErr = tx.QueryRow(ctx, `SELECT command_type,expected_brand_version,result_brand_version FROM third_party_vendor_brand_command_receipts WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3`, tenantID, record.VendorID, record.IdempotencyKey).Scan(&receipt.Command, &receipt.ExpectedVersion, &receipt.ResultVersion)
	if receiptErr == nil {
		version, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandApproveCommand, record.ExpectedVersion)
		return record.Asset, version, replayErr
	}
	if !errors.Is(receiptErr, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, receiptErr
	}
	version, err := nextVendorBrandEventVersionPG(ctx, tx, tenantID, record.VendorID)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if version-1 != record.ExpectedVersion {
		return VendorBrandAsset{}, version - 1, ErrBrandVersionConflict
	}
	var reserved string
	err = tx.QueryRow(ctx, `SELECT idempotency_key FROM third_party_vendor_brand_upload_reservations WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3 AND artifact_key=$4 AND source_digest=$5 AND state='RESERVED' FOR UPDATE`, tenantID, record.VendorID, record.IdempotencyKey, record.Asset.ArtifactKey, record.Asset.SourceDigest).Scan(&reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, ErrInvalid
	}
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE third_party_vendor_brand_assets SET state='SUPERSEDED',updated_at=$3,version=version+1 WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND source_kind='APPROVED_OVERRIDE' AND state='CURRENT'`, tenantID, record.VendorID, record.OccurredAt)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	a := record.Asset
	_, err = tx.Exec(ctx, `INSERT INTO third_party_vendor_brand_assets(id,tenant_id,vendor_id,source_kind,state,source_domain,artifact_key,source_digest,media_type,pixel_width,pixel_height,byte_size,retrieved_at,next_refresh_at,approved_by_principal_id,created_at,updated_at,version,asset_token) VALUES($1::uuid,$2::uuid,$3::uuid,'APPROVED_OVERRIDE','CURRENT','',$4,$5,'image/png',$6,$7,$8,$9,NULL,$10::uuid,$9,$9,1,$11)`, a.ID, tenantID, record.VendorID, a.ArtifactKey, a.SourceDigest, a.PixelWidth, a.PixelHeight, a.ByteSize, record.OccurredAt, record.ActorID, a.AssetToken)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if err = appendVendorBrandMutationEvent(ctx, tx, tenantID, record, VendorBrandApprovedEvent, version); err != nil {
		return VendorBrandAsset{}, 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_vendor_brand_command_receipts(tenant_id,vendor_id,idempotency_key,command_type,expected_brand_version,result_brand_version,created_at) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7)`, tenantID, record.VendorID, record.IdempotencyKey, VendorBrandApproveCommand, record.ExpectedVersion, version, record.OccurredAt)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	command, err := tx.Exec(ctx, `UPDATE third_party_vendor_brand_upload_reservations SET state='COMMITTED',lease_token=NULL,lease_expires_at=NULL,updated_at=$4 WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3 AND state='RESERVED'`, tenantID, record.VendorID, record.IdempotencyKey, record.OccurredAt)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if command.RowsAffected() != 1 {
		return VendorBrandAsset{}, 0, ErrVersionConflict
	}
	if err = tx.Commit(ctx); err != nil {
		commitErr := err
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.vendorBrandMutationRecorded(probeCtx, record, VendorBrandApprovedEvent, version, true)
		if confirmed {
			return a, version, nil
		}
		if probeErr != nil {
			return VendorBrandAsset{}, 0, errors.Join(commitErr, probeErr)
		}
		return VendorBrandAsset{}, 0, commitErr
	}
	return a, version, nil
}

func (r *PostgresRepository) RemoveApprovedVendorBrand(ctx context.Context, record VendorBrandMutationRecord) (VendorBrandAsset, int64, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	var receipt VendorBrandReceipt
	receipt.TenantID, receipt.VendorID, receipt.IdempotencyKey = record.TenantID, record.VendorID, record.IdempotencyKey
	receiptErr := tx.QueryRow(ctx, `SELECT command_type,expected_brand_version,result_brand_version FROM third_party_vendor_brand_command_receipts WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3`, tenantID, record.VendorID, record.IdempotencyKey).Scan(&receipt.Command, &receipt.ExpectedVersion, &receipt.ResultVersion)
	if receiptErr == nil {
		version, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandRemoveCommand, record.ExpectedVersion)
		return receipt.Asset, version, replayErr
	}
	if !errors.Is(receiptErr, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, receiptErr
	}
	var lockedVendor string
	err = tx.QueryRow(ctx, `SELECT p.id::text FROM third_parties p WHERE p.tenant_id=$1::uuid AND p.id::text=$3 AND EXISTS(SELECT 1 FROM third_party_relationships rel WHERE rel.tenant_id=p.tenant_id AND rel.vendor_id=p.id AND rel.legal_entity_id::text=$2) FOR UPDATE`, tenantID, record.LegalEntityID, record.VendorID).Scan(&lockedVendor)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, ErrNotFound
	}
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	receipt = VendorBrandReceipt{TenantID: record.TenantID, VendorID: record.VendorID, IdempotencyKey: record.IdempotencyKey}
	receiptErr = tx.QueryRow(ctx, `SELECT command_type,expected_brand_version,result_brand_version FROM third_party_vendor_brand_command_receipts WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND idempotency_key=$3`, tenantID, record.VendorID, record.IdempotencyKey).Scan(&receipt.Command, &receipt.ExpectedVersion, &receipt.ResultVersion)
	if receiptErr == nil {
		version, replayErr := vendorBrandReceiptVersion(receipt, VendorBrandRemoveCommand, record.ExpectedVersion)
		return receipt.Asset, version, replayErr
	}
	if !errors.Is(receiptErr, pgx.ErrNoRows) {
		return VendorBrandAsset{}, 0, receiptErr
	}
	version, err := nextVendorBrandEventVersionPG(ctx, tx, tenantID, record.VendorID)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if version-1 != record.ExpectedVersion {
		return VendorBrandAsset{}, version - 1, ErrBrandVersionConflict
	}
	removed, err := scanVendorBrandAsset(tx.QueryRow(ctx, `SELECT `+vendorBrandAssetProjection+` FROM third_party_vendor_brand_assets a JOIN tenants t ON t.id=a.tenant_id WHERE a.tenant_id=$1::uuid AND a.vendor_id=$2::uuid AND a.source_kind='APPROVED_OVERRIDE' AND a.state='CURRENT' ORDER BY a.updated_at DESC,a.id DESC LIMIT 1 FOR UPDATE OF a`, tenantID, record.VendorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorBrandAsset{}, version - 1, ErrVendorBrandOverrideNotFound
	}
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	command, err := tx.Exec(ctx, `UPDATE third_party_vendor_brand_assets SET state='SUPERSEDED',updated_at=$4,version=version+1 WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid AND id=$3::uuid AND version=$5 AND state='CURRENT'`, tenantID, record.VendorID, removed.ID, record.OccurredAt, removed.Version)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if command.RowsAffected() != 1 {
		return VendorBrandAsset{}, 0, ErrVersionConflict
	}
	removed.State, removed.UpdatedAt, removed.Version = VendorBrandAssetSuperseded, record.OccurredAt, removed.Version+1
	record.Asset = removed
	if err = appendVendorBrandMutationEvent(ctx, tx, tenantID, record, VendorBrandRemovedEvent, version); err != nil {
		return VendorBrandAsset{}, 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_vendor_brand_command_receipts(tenant_id,vendor_id,idempotency_key,command_type,expected_brand_version,result_brand_version,created_at) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7)`, tenantID, record.VendorID, record.IdempotencyKey, VendorBrandRemoveCommand, record.ExpectedVersion, version, record.OccurredAt)
	if err != nil {
		return VendorBrandAsset{}, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		commitErr := err
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.vendorBrandMutationRecorded(probeCtx, record, VendorBrandRemovedEvent, version, false)
		if confirmed {
			return removed, version, nil
		}
		if probeErr != nil {
			return VendorBrandAsset{}, 0, errors.Join(commitErr, probeErr)
		}
		return VendorBrandAsset{}, 0, commitErr
	}
	return removed, version, nil
}

func nextVendorBrandEventVersionPG(ctx context.Context, tx pgx.Tx, tenantID, vendorID string) (int64, error) {
	var version int64
	err := tx.QueryRow(ctx, `SELECT COALESCE(max(aggregate_version),0)+1 FROM third_party_events WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR_BRAND' AND aggregate_id=$2::uuid`, tenantID, vendorID).Scan(&version)
	return version, err
}
func appendVendorBrandMutationEvent(ctx context.Context, tx pgx.Tx, tenantID string, record VendorBrandMutationRecord, event string, version int64) error {
	_, err := tx.Exec(ctx, `INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at) VALUES($1::uuid,'VENDOR_BRAND',$2::uuid,$3,$4::uuid,$5,jsonb_build_object('asset_id',$6::text,'asset_version',$7::bigint,'brand_version',$3::bigint),$8)`, tenantID, record.VendorID, version, record.ActorID, event, record.Asset.ID, record.Asset.Version, record.OccurredAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'VENDOR_BRAND',$2::uuid,$3,jsonb_build_object('asset_id',$4::text,'asset_version',$5::bigint,'brand_version',$6::bigint),$7,$7)`, tenantID, record.VendorID, event, record.Asset.ID, record.Asset.Version, version, record.OccurredAt)
	return err
}

func (r *PostgresRepository) vendorBrandMutationRecorded(ctx context.Context, record VendorBrandMutationRecord, event string, version int64, requiresAsset bool) (bool, error) {
	command := VendorBrandRemoveCommand
	if event == VendorBrandApprovedEvent {
		command = VendorBrandApproveCommand
	}
	var confirmed bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants t JOIN third_party_vendor_brand_command_receipts receipt ON receipt.tenant_id=t.id JOIN third_party_events e ON e.tenant_id=receipt.tenant_id AND e.aggregate_type='VENDOR_BRAND' AND e.aggregate_id=receipt.vendor_id AND e.aggregate_version=receipt.result_brand_version AND e.event_type=$5 JOIN outbox_events o ON o.tenant_id=receipt.tenant_id AND o.aggregate_type='VENDOR_BRAND' AND o.aggregate_id=receipt.vendor_id AND o.event_type=$5 AND o.payload->>'brand_version'=$6::text WHERE (t.id::text=$1 OR t.slug=$1) AND receipt.vendor_id::text=$2 AND receipt.idempotency_key=$3 AND receipt.command_type=$12 AND receipt.expected_brand_version=$4 AND receipt.result_brand_version=$6 AND e.payload->>'asset_id'=$8 AND e.payload->>'asset_version'=$11::text AND o.payload->>'asset_id'=$8 AND o.payload->>'asset_version'=$11::text AND (NOT $7 OR (EXISTS(SELECT 1 FROM third_party_vendor_brand_assets a WHERE a.tenant_id=receipt.tenant_id AND a.vendor_id=receipt.vendor_id AND a.id::text=$8 AND a.artifact_key=$9 AND a.source_digest=$10 AND a.media_type='image/png') AND EXISTS(SELECT 1 FROM third_party_vendor_brand_upload_reservations q WHERE q.tenant_id=receipt.tenant_id AND q.vendor_id=receipt.vendor_id AND q.idempotency_key=receipt.idempotency_key AND q.state='COMMITTED' AND q.artifact_key=$9 AND q.source_digest=$10))))`, record.TenantID, record.VendorID, record.IdempotencyKey, record.ExpectedVersion, event, version, requiresAsset, record.Asset.ID, record.Asset.ArtifactKey, record.Asset.SourceDigest, record.Asset.Version, command).Scan(&confirmed)
	return confirmed, err
}

func (r *PostgresRepository) ClaimExpiredVendorBrandReservations(ctx context.Context, now, cutoff time.Time, lease time.Duration, limit int) ([]VendorBrandUploadReservation, error) {
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `WITH candidates AS (SELECT tenant_id,vendor_id,idempotency_key FROM third_party_vendor_brand_upload_reservations WHERE (state='RESERVED' AND updated_at<=$1) OR (state='CLEANING' AND lease_expires_at<=$2) ORDER BY updated_at,tenant_id,vendor_id LIMIT $3 FOR UPDATE SKIP LOCKED), claimed AS (UPDATE third_party_vendor_brand_upload_reservations q SET state='CLEANING',lease_token=uuidv7(),lease_expires_at=$2+($4 * interval '1 second'),updated_at=$2 FROM candidates c WHERE q.tenant_id=c.tenant_id AND q.vendor_id=c.vendor_id AND q.idempotency_key=c.idempotency_key RETURNING q.*) SELECT c.tenant_id::text,c.vendor_id::text,c.idempotency_key,c.artifact_key,c.source_digest,c.state,c.expected_brand_version,c.created_at,c.updated_at,c.lease_token::text,c.lease_expires_at FROM claimed c`, cutoff, now, limit, lease.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []VendorBrandUploadReservation{}
	for rows.Next() {
		var x VendorBrandUploadReservation
		if err := rows.Scan(&x.TenantID, &x.VendorID, &x.IdempotencyKey, &x.ArtifactKey, &x.SourceDigest, &x.State, &x.ExpectedVersion, &x.CreatedAt, &x.UpdatedAt, &x.LeaseToken, &x.LeaseExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (r *PostgresRepository) VendorBrandArtifactReference(ctx context.Context, item VendorBrandUploadReservation) (VendorBrandArtifactReference, error) {
	var committed, protected bool
	expectedAssetID := vendorBrandReservationAssetID(item.TenantID, item.VendorID, item.IdempotencyKey)
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_vendor_brand_assets a JOIN tenants t ON t.id=a.tenant_id JOIN third_party_vendor_brand_command_receipts receipt ON receipt.tenant_id=a.tenant_id AND receipt.vendor_id=a.vendor_id AND receipt.idempotency_key=$4 WHERE (t.id::text=$1 OR t.slug=$1) AND a.vendor_id::text=$3 AND a.id::text=$5 AND a.artifact_key=$2 AND a.source_digest=$6 AND receipt.command_type=$7 AND receipt.expected_brand_version=$8), EXISTS(SELECT 1 FROM third_party_vendor_brand_assets a JOIN tenants t ON t.id=a.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND a.artifact_key=$2 UNION ALL SELECT 1 FROM third_party_vendor_brand_upload_reservations q JOIN tenants t ON t.id=q.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND q.artifact_key=$2 AND (q.vendor_id::text<>$3 OR q.idempotency_key<>$4) AND q.state IN ('RESERVED','CLEANING','COMMITTED'))`, item.TenantID, item.ArtifactKey, item.VendorID, item.IdempotencyKey, expectedAssetID, item.SourceDigest, VendorBrandApproveCommand, item.ExpectedVersion).Scan(&committed, &protected)
	if err != nil {
		return "", err
	}
	if committed {
		return VendorBrandArtifactCommitted, nil
	}
	if protected {
		return VendorBrandArtifactProtected, nil
	}
	return VendorBrandArtifactUnreferenced, nil
}
func (r *PostgresRepository) CompleteVendorBrandReservationCleanup(ctx context.Context, item VendorBrandUploadReservation, reference VendorBrandArtifactReference, at time.Time) error {
	var command pgconn.CommandTag
	var err error
	if reference == VendorBrandArtifactCommitted {
		command, err = r.pool.Exec(ctx, `UPDATE third_party_vendor_brand_upload_reservations q SET state='COMMITTED',lease_token=NULL,lease_expires_at=NULL,updated_at=$5 FROM tenants t WHERE q.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND q.vendor_id::text=$2 AND q.idempotency_key=$3 AND q.state='CLEANING' AND q.lease_token::text=$4`, item.TenantID, item.VendorID, item.IdempotencyKey, item.LeaseToken, at)
	} else {
		command, err = r.pool.Exec(ctx, `DELETE FROM third_party_vendor_brand_upload_reservations q USING tenants t WHERE q.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND q.vendor_id::text=$2 AND q.idempotency_key=$3 AND q.state='CLEANING' AND q.lease_token::text=$4`, item.TenantID, item.VendorID, item.IdempotencyKey, item.LeaseToken)
	}
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	return nil
}

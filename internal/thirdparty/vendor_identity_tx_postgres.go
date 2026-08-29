//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func updateVendorIdentityTx(ctx context.Context, tx pgx.Tx, tenantID string, record UpdateVendorIdentityRecord) (Vendor, error) {
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
		SET legal_name=$3,trading_name=$4,registration_ref=$5,jurisdiction=$6,registered_address=NULLIF($7,''),website_domain=NULLIF($8,''),updated_at=$9,version=version+1
		WHERE tenant_id=$1::uuid AND id=$2::uuid AND version=$10
		RETURNING version`, tenantID, current.ID, updated.LegalName, updated.TradingName, updated.RegistrationRef,
		updated.Jurisdiction, updated.RegisteredAddress, updated.WebsiteDomain, updated.UpdatedAt, record.ExpectedVersion).Scan(&updated.Version)
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
	return updated, nil
}

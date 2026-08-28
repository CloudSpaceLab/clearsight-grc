//go:build postgres

package evidence

import (
	"context"
	"time"
)

func (store *PostgresCommunicationStore) MarkProfileRollback(ctx context.Context, tenantID, legalEntityID string, version, sourceVersion int64) (CommunicationProfile, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 || sourceVersion < 1 {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationProfile{}, err
	}
	defer tx.Rollback(ctx)
	canonicalTenantID, canonicalLegalEntityID, err := resolveCommunicationScope(ctx, tx, tenantID, legalEntityID)
	if err != nil {
		return CommunicationProfile{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "profile", canonicalTenantID, canonicalLegalEntityID, "", ""); err != nil {
		return CommunicationProfile{}, err
	}
	var sourceExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM form_communication_profiles
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$3
		)`, canonicalTenantID, canonicalLegalEntityID, sourceVersion).Scan(&sourceExists); err != nil {
		return CommunicationProfile{}, err
	}
	if !sourceExists {
		return CommunicationProfile{}, ErrCommunicationNotFound
	}
	row := tx.QueryRow(ctx, `
		UPDATE form_communication_profiles
		SET rollback_origin_version=$4,updated_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$3
		RETURNING id::text,tenant_id::text,legal_entity_id::text,version,default_locale,bank_name,support_contact,
		          COALESCE(brand_asset_id::text,''),status,effective_from,effective_until,maker_id::text,
		          COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		canonicalTenantID, canonicalLegalEntityID, version, sourceVersion)
	updated, err := scanCommunicationProfile(row)
	if err != nil {
		return CommunicationProfile{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(
		ctx,
		tx,
		updated.TenantID,
		updated.ID,
		"FORM_COMMUNICATION_PROFILE",
		"FORM_COMMUNICATION_PROFILE_ROLLBACK_MARKED",
		updated.MakerID,
		updated.LegalEntityID,
		"",
		"",
		updated.Version,
		updated.Status,
		time.Now().UTC(),
	); err != nil {
		return CommunicationProfile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationProfile{}, err
	}
	return updated, nil
}

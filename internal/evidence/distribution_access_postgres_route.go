//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (store *PostgresDistributionStore) GetRequest(ctx context.Context, tenantID, requestID string) (Request, error) {
	if store == nil || store.repo == nil {
		return Request{}, ErrNotFound
	}
	request, err := store.repo.GetRequest(ctx, tenantID, requestID)
	if err != nil {
		return Request{}, err
	}
	return hydrateRequestRecipient(ctx, store.repo, request)
}

func (store *PostgresDistributionStore) CreateAccessRoutes(ctx context.Context, routes []AccessRoute) error {
	if store == nil || store.repo == nil || store.repo.pool == nil || len(routes) == 0 {
		return ErrDistributionAccessUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	defer tx.Rollback(ctx)
	for _, route := range routes {
		if !validPersistedPostgresAccessRoute(route) {
			return ErrDistributionAccessUnavailable
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO capture_access_routes(
				id,tenant_id,legal_entity_id,distribution_id,recipient_id,access_policy,selector_hash,
				audience_hint,expires_at,max_redemptions,redemptions,created_by,created_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12::uuid,$13)`,
			route.ID, route.TenantID, route.LegalEntityID, route.DistributionID, route.RecipientID,
			route.Policy, route.SelectorHash, route.AudienceHint, route.ExpiresAt, route.MaxRedemptions,
			route.Redemptions, route.CreatedBy, route.CreatedAt); err != nil {
			return ErrDistributionAccessUnavailable
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func (store *PostgresDistributionStore) AccessRouteBySelectorHash(ctx context.Context, selectorHash []byte) (AccessRoute, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || len(selectorHash) != 32 {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	route, err := scanAccessRoute(store.repo.pool.QueryRow(ctx, accessRouteSelect+` WHERE ar.selector_hash=$1`, selectorHash))
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return route, nil
}

func (store *PostgresDistributionStore) AccessRouteByID(ctx context.Context, tenantID, legalEntityID, distributionID, routeID string) (AccessRoute, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	route, err := scanAccessRoute(store.repo.pool.QueryRow(ctx, accessRouteSelect+`
		WHERE ar.id=$1::uuid AND ar.tenant_id=$2::uuid AND ar.legal_entity_id=$3::uuid AND ar.distribution_id=$4::uuid`,
		routeID, tenantID, legalEntityID, distributionID))
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return route, nil
}

func (store *PostgresDistributionStore) ProtectedRecipientForAccess(ctx context.Context, route AccessRoute, recipientID string) (DistributionRecipient, protectedRecipientAddress, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
	}
	var recipient DistributionRecipient
	var protected protectedRecipientAddress
	err := store.repo.pool.QueryRow(ctx, `
		SELECT r.id::text,r.distribution_id::text,r.tenant_id::text,r.legal_entity_id::text,r.role,r.recipient_type,
		       COALESCE(r.principal_id::text,''),COALESCE(r.request_id::text,''),r.audience_hint,r.contact_label,r.state,r.version,r.created_at,r.updated_at,
		       r.address_hash,r.address_ciphertext,r.address_key_id
		FROM capture_distribution_recipients r
		WHERE r.id=$1::uuid AND r.tenant_id=$2::uuid AND r.legal_entity_id=$3::uuid AND r.distribution_id=$4::uuid`,
		recipientID, route.TenantID, route.LegalEntityID, route.DistributionID).Scan(
		&recipient.ID, &recipient.DistributionID, &recipient.TenantID, &recipient.LegalEntityID,
		&recipient.Role, &recipient.Type, &recipient.PrincipalID, &recipient.RequestID,
		&recipient.AudienceHint, &recipient.ContactLabel, &recipient.State, &recipient.Version,
		&recipient.CreatedAt, &recipient.UpdatedAt, &protected.Hash, &protected.Ciphertext, &protected.KeyID,
	)
	if errors.Is(err, pgx.ErrNoRows) || err != nil || len(eligibleAccessRecipients(route, []DistributionRecipient{recipient})) != 1 || len(protected.Hash) != 32 || len(protected.Ciphertext) == 0 || protected.KeyID == "" {
		return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
	}
	return recipient, protected, nil
}

func validPersistedPostgresAccessRoute(route AccessRoute) bool {
	return route.ID != "" && len(route.SelectorHash) == 32 && route.Redemptions == 0 && accessRouteOpen(route, route.CreatedAt)
}

var _ DistributionAccessStore = (*PostgresDistributionStore)(nil)

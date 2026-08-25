//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateRelationship(ctx context.Context, record CreateRecord) (Aggregate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Aggregate{}, fmt.Errorf("begin vendor relationship create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tenantID, err := resolveTenant(ctx, tx, record.Vendor.TenantID)
	if err != nil {
		return Aggregate{}, err
	}
	vendorID := record.Vendor.ID
	if record.Vendor.SourceID != "" && record.Vendor.ExternalRef != "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO third_parties(id,tenant_id,legal_name,trading_name,registration_ref,jurisdiction,source_id,external_ref,status,created_at,updated_at,version)
			VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$10,1)
			ON CONFLICT (tenant_id,source_id,external_ref) WHERE source_id<>'' AND external_ref<>''
			DO UPDATE SET source_id=EXCLUDED.source_id
			RETURNING id::text`, record.Vendor.ID, tenantID, record.Vendor.LegalName, record.Vendor.TradingName, record.Vendor.RegistrationRef, record.Vendor.Jurisdiction, record.Vendor.SourceID, record.Vendor.ExternalRef, record.Vendor.Status, record.Vendor.CreatedAt).Scan(&vendorID)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO third_parties(id,tenant_id,legal_name,trading_name,registration_ref,jurisdiction,source_id,external_ref,status,created_at,updated_at,version)
			VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,'','',$7,$8,$8,1)`, record.Vendor.ID, tenantID, record.Vendor.LegalName, record.Vendor.TradingName, record.Vendor.RegistrationRef, record.Vendor.Jurisdiction, record.Vendor.Status, record.Vendor.CreatedAt)
	}
	if err != nil {
		return Aggregate{}, fmt.Errorf("store vendor: %w", err)
	}
	storedVendor := Vendor{TenantID: record.Vendor.TenantID}
	err = tx.QueryRow(ctx, `
		SELECT id::text,legal_name,trading_name,registration_ref,jurisdiction,source_id,external_ref,status,created_at,updated_at,version
		FROM third_parties WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, vendorID).Scan(
		&storedVendor.ID, &storedVendor.LegalName, &storedVendor.TradingName, &storedVendor.RegistrationRef, &storedVendor.Jurisdiction,
		&storedVendor.SourceID, &storedVendor.ExternalRef, &storedVendor.Status, &storedVendor.CreatedAt, &storedVendor.UpdatedAt, &storedVendor.Version,
	)
	if err != nil {
		return Aggregate{}, fmt.Errorf("load stored vendor: %w", err)
	}
	record.Vendor = storedVendor
	record.Relationship.VendorID = vendorID
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,effective_from,renewal_at,source_id,external_ref,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,(SELECT id FROM legal_entities WHERE tenant_id=$2::uuid AND id::text=$3),$4::uuid,$5,$6::uuid,$7,$8,$9,$10,$11,$12,$13,$14,$14,1)`,
		record.Relationship.ID, tenantID, record.Relationship.LegalEntityID, vendorID, record.Relationship.ServiceName, record.Relationship.BusinessOwnerPrincipalID,
		record.Relationship.Criticality, record.Relationship.PrivacyRole, record.Relationship.Status, record.Relationship.EffectiveFrom, record.Relationship.RenewalAt,
		record.Relationship.SourceID, record.Relationship.ExternalRef, record.Relationship.CreatedAt)
	if err != nil {
		return Aggregate{}, fmt.Errorf("store vendor relationship: %w", err)
	}
	if err := appendRelationshipEvent(ctx, tx, tenantID, record.Relationship, record.Relationship.BusinessOwnerPrincipalID, "VendorRelationshipCreated"); err != nil {
		return Aggregate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Aggregate{}, fmt.Errorf("commit vendor relationship create: %w", err)
	}
	return Aggregate{Vendor: record.Vendor, Relationship: record.Relationship}, nil
}

func (r *PostgresRepository) UpdateRelationship(ctx context.Context, record UpdateRecord) (Aggregate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Aggregate{}, fmt.Errorf("begin vendor relationship update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return Aggregate{}, err
	}
	var vendorID string
	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT r.vendor_id::text,r.version
		FROM third_party_relationships r
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id::text=$2 AND r.id::text=$3
		FOR UPDATE`, tenantID, record.LegalEntityID, record.ID).Scan(&vendorID, &currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrNotFound
	}
	if err != nil {
		return Aggregate{}, fmt.Errorf("lock vendor relationship: %w", err)
	}
	if currentVersion != record.ExpectedVersion {
		return Aggregate{}, ErrVersionConflict
	}
	var relationshipVersion int64
	err = tx.QueryRow(ctx, `
		UPDATE third_party_relationships
		SET service_name=$4,criticality=$5,privacy_role=$6,effective_from=$7,renewal_at=$8,updated_at=$9,version=version+1
		WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND id::text=$3 AND version=$10
		RETURNING version`, tenantID, record.LegalEntityID, record.ID, record.Relationship.ServiceName, record.Relationship.Criticality,
		record.Relationship.PrivacyRole, record.Relationship.EffectiveFrom, record.Relationship.RenewalAt, record.Relationship.UpdatedAt, record.ExpectedVersion).Scan(&relationshipVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrVersionConflict
	}
	if err != nil {
		return Aggregate{}, fmt.Errorf("update vendor relationship: %w", err)
	}
	record.Relationship.ID, record.Relationship.TenantID = record.ID, record.TenantID
	record.Relationship.LegalEntityID, record.Relationship.VendorID = record.LegalEntityID, vendorID
	record.Relationship.Version = relationshipVersion
	if err := appendRelationshipEvent(ctx, tx, tenantID, record.Relationship, record.ActorID, "VendorRelationshipUpdated"); err != nil {
		return Aggregate{}, err
	}
	stored, err := scanAggregate(tx.QueryRow(ctx, relationshipSelect+`
		WHERE t.id=$1::uuid AND r.legal_entity_id::text=$2 AND r.id::text=$3`, tenantID, record.LegalEntityID, record.ID))
	if err != nil {
		return Aggregate{}, fmt.Errorf("load updated vendor relationship: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Aggregate{}, fmt.Errorf("commit vendor relationship update: %w", err)
	}
	return stored, nil
}

func (r *PostgresRepository) GetRelationship(ctx context.Context, scope Scope, relationshipID string) (Aggregate, error) {
	row := r.pool.QueryRow(ctx, relationshipSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id::text=$2 AND r.id::text=$3`, scope.TenantID, scope.LegalEntityID, relationshipID)
	value, err := scanAggregate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrNotFound
	}
	if err != nil {
		return Aggregate{}, fmt.Errorf("get vendor relationship: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) ListRelationships(ctx context.Context, filter ListFilter) (RelationshipPage, error) {
	args := []any{filter.TenantID, filter.LegalEntityID, strings.TrimSpace(filter.Search)}
	whereCursor := ""
	if filter.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(filter.Cursor)
		if err != nil {
			return RelationshipPage{}, ErrInvalid
		}
		args = append(args, cursorTime, cursorID)
		whereCursor = " AND (r.updated_at,r.id) < ($4,$5::uuid)"
	}
	args = append(args, filter.Limit+1)
	query := relationshipSelect + `
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id::text=$2
		  AND ($3='' OR p.legal_name ILIKE '%'||$3||'%' OR p.trading_name ILIKE '%'||$3||'%' OR r.service_name ILIKE '%'||$3||'%')` + whereCursor + `
		ORDER BY r.updated_at DESC,r.id DESC LIMIT $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return RelationshipPage{}, fmt.Errorf("list vendor relationships: %w", err)
	}
	defer rows.Close()
	items := make([]Aggregate, 0, filter.Limit+1)
	for rows.Next() {
		value, scanErr := scanAggregate(rows)
		if scanErr != nil {
			return RelationshipPage{}, scanErr
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return RelationshipPage{}, err
	}
	page := RelationshipPage{Items: items}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		last := page.Items[len(page.Items)-1].Relationship
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

const relationshipSelect = `
	SELECT p.id::text,t.slug,p.legal_name,p.trading_name,p.registration_ref,p.jurisdiction,p.source_id,p.external_ref,p.status,p.created_at,p.updated_at,p.version,
	       r.id::text,t.slug,r.legal_entity_id::text,r.vendor_id::text,r.service_name,r.business_owner_principal_id::text,r.criticality,r.privacy_role,r.status,
	       r.effective_from,r.renewal_at,r.source_id,r.external_ref,r.created_at,r.updated_at,r.version
	FROM third_party_relationships r
	JOIN third_parties p ON p.tenant_id=r.tenant_id AND p.id=r.vendor_id
	JOIN tenants t ON t.id=r.tenant_id`

type rowScanner interface{ Scan(...any) error }

func scanAggregate(row rowScanner) (Aggregate, error) {
	var value Aggregate
	err := row.Scan(
		&value.Vendor.ID, &value.Vendor.TenantID, &value.Vendor.LegalName, &value.Vendor.TradingName, &value.Vendor.RegistrationRef, &value.Vendor.Jurisdiction,
		&value.Vendor.SourceID, &value.Vendor.ExternalRef, &value.Vendor.Status, &value.Vendor.CreatedAt, &value.Vendor.UpdatedAt, &value.Vendor.Version,
		&value.Relationship.ID, &value.Relationship.TenantID, &value.Relationship.LegalEntityID, &value.Relationship.VendorID, &value.Relationship.ServiceName,
		&value.Relationship.BusinessOwnerPrincipalID, &value.Relationship.Criticality, &value.Relationship.PrivacyRole, &value.Relationship.Status,
		&value.Relationship.EffectiveFrom, &value.Relationship.RenewalAt, &value.Relationship.SourceID, &value.Relationship.ExternalRef,
		&value.Relationship.CreatedAt, &value.Relationship.UpdatedAt, &value.Relationship.Version,
	)
	return value, err
}

func resolveTenant(ctx context.Context, tx pgx.Tx, tenant string) (string, error) {
	var tenantID string
	err := tx.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1`, tenant).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve tenant: %w", err)
	}
	return tenantID, nil
}

func appendRelationshipEvent(ctx context.Context, tx pgx.Tx, tenantID string, relationship Relationship, actorID, eventType string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'VENDOR_RELATIONSHIP',$2::uuid,$3,$4::uuid,$5,jsonb_build_object('vendor_id',$6::text,'status',$7::text,'criticality',$8::text),$9)`,
		tenantID, relationship.ID, relationship.Version, actorID, eventType, relationship.VendorID, relationship.Status, relationship.Criticality, relationship.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor relationship event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'VENDOR_RELATIONSHIP',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'status',$5::text),$6,$6)`,
		tenantID, relationship.ID, eventType, relationship.Version, relationship.Status, relationship.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor relationship outbox event: %w", err)
	}
	return nil
}

var _ Repository = (*PostgresRepository)(nil)

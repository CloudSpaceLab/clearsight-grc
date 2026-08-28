//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type PostgresCommunicationBrandStore struct {
	repo *PostgresRepository
}

func NewPostgresCommunicationBrandStore(repo *PostgresRepository) *PostgresCommunicationBrandStore {
	return &PostgresCommunicationBrandStore{repo: repo}
}

func (store *PostgresCommunicationBrandStore) CreateCommunicationBrandAsset(ctx context.Context, value BrandAsset) (BrandAsset, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	row := store.repo.pool.QueryRow(ctx, `
		INSERT INTO form_brand_assets(
			id,tenant_id,legal_entity_id,artifact_key,digest_hex,media_type,width,height,size_bytes,alt_text,created_by,created_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11::uuid,$12
		)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,artifact_key,digest_hex,media_type,width,height,size_bytes,alt_text,created_by::text,created_at`,
		value.ID, value.TenantID, value.LegalEntityID, value.ArtifactKey, value.DigestHex, value.MediaType,
		value.Width, value.Height, value.SizeBytes, value.AltText, value.CreatedBy, value.CreatedAt.UTC())
	return scanCommunicationBrandAsset(row)
}

func (store *PostgresCommunicationBrandStore) GetCommunicationBrandAsset(ctx context.Context, tenantID, legalEntityID, assetID string) (BrandAsset, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	row := store.repo.pool.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,artifact_key,digest_hex,media_type,width,height,size_bytes,alt_text,created_by::text,created_at
		FROM form_brand_assets
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`, tenantID, legalEntityID, assetID)
	value, err := scanCommunicationBrandAsset(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return BrandAsset{}, ErrCommunicationNotFound
	}
	return value, err
}

func (store *PostgresCommunicationBrandStore) ListCommunicationBrandAssets(ctx context.Context, tenantID, legalEntityID string) ([]BrandAsset, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return nil, ErrCommunicationInvalid
	}
	rows, err := store.repo.pool.Query(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,artifact_key,digest_hex,media_type,width,height,size_bytes,alt_text,created_by::text,created_at
		FROM form_brand_assets
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		ORDER BY created_at DESC,id DESC`, tenantID, legalEntityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]BrandAsset, 0)
	for rows.Next() {
		value, err := scanCommunicationBrandAsset(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

type communicationBrandRow interface {
	Scan(...any) error
}

func scanCommunicationBrandAsset(row communicationBrandRow) (BrandAsset, error) {
	var value BrandAsset
	if err := row.Scan(
		&value.ID,
		&value.TenantID,
		&value.LegalEntityID,
		&value.ArtifactKey,
		&value.DigestHex,
		&value.MediaType,
		&value.Width,
		&value.Height,
		&value.SizeBytes,
		&value.AltText,
		&value.CreatedBy,
		&value.CreatedAt,
	); err != nil {
		return BrandAsset{}, err
	}
	value.CreatedAt = value.CreatedAt.UTC()
	return value, nil
}

var _ communicationBrandStore = (*PostgresCommunicationBrandStore)(nil)

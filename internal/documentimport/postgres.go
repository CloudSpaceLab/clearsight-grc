//go:build postgres

package documentimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, value Document) (Document, error) {
	limitations, _ := json.Marshal(value.Limitations)
	sections, _ := json.Marshal(value.Sections)
	proposals, _ := json.Marshal(value.Proposals)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO document_imports(
			id,tenant_id,legal_entity_id,file_name,media_type,purpose,source_type,size_bytes,sha256,storage_key,
			artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,limitations,sections,proposals,
			created_by,created_at,updated_at,version)
		SELECT $2::uuid,t.id,
		       CASE WHEN $3='' THEN NULL ELSE (SELECT le.id FROM legal_entities le WHERE le.tenant_id=t.id AND le.id=$3::uuid) END,
		       $4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::jsonb,$17::jsonb,$18::jsonb,$19::uuid,$20,$21,1
		FROM tenants t WHERE t.id::text=$1 OR t.slug=$1
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),file_name,media_type,purpose,source_type,
		          size_bytes,sha256,storage_key,artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,
		          limitations,sections,proposals,created_by::text,created_at,updated_at,version`,
		value.TenantID, value.ID, value.LegalEntityID, value.FileName, value.MediaType, value.Purpose, value.SourceType,
		value.SizeBytes, value.SHA256, value.StorageKey, value.ArtifactStatus, value.ExtractionStatus, value.ExtractionMethod,
		value.AnalysisStatus, value.AnalysisMethod, limitations, sections, proposals, value.CreatedBy, value.CreatedAt, value.UpdatedAt,
	)
	created, err := scanDocument(row)
	if err != nil {
		return Document{}, fmt.Errorf("create document import: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenant string, limit int) ([]Document, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT di.id::text,t.slug,COALESCE(di.legal_entity_id::text,''),di.file_name,di.media_type,di.purpose,di.source_type,
		       di.size_bytes,di.sha256,di.storage_key,di.artifact_status,di.extraction_status,di.extraction_method,
		       di.analysis_status,di.analysis_method,di.limitations,di.sections,di.proposals,di.created_by::text,
		       di.created_at,di.updated_at,di.version
		FROM document_imports di JOIN tenants t ON t.id=di.tenant_id
		WHERE t.id::text=$1 OR t.slug=$1
		ORDER BY di.created_at DESC,di.id DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, fmt.Errorf("list document imports: %w", err)
	}
	defer rows.Close()
	values := make([]Document, 0, limit)
	for rows.Next() {
		value, scanErr := scanDocument(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan document import: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document imports: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) Get(ctx context.Context, tenant, id string) (Document, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT di.id::text,t.slug,COALESCE(di.legal_entity_id::text,''),di.file_name,di.media_type,di.purpose,di.source_type,
		       di.size_bytes,di.sha256,di.storage_key,di.artifact_status,di.extraction_status,di.extraction_method,
		       di.analysis_status,di.analysis_method,di.limitations,di.sections,di.proposals,di.created_by::text,
		       di.created_at,di.updated_at,di.version
		FROM document_imports di JOIN tenants t ON t.id=di.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid`, tenant, id)
	value, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("load document import: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) SaveReview(ctx context.Context, value Document, expected int64) (Document, error) {
	proposals, _ := json.Marshal(value.Proposals)
	row := r.pool.QueryRow(ctx, `
		UPDATE document_imports SET proposals=$4::jsonb,updated_at=$5,version=version+1
		WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$3
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),file_name,media_type,purpose,source_type,
		          size_bytes,sha256,storage_key,artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,
		          limitations,sections,proposals,created_by::text,created_at,updated_at,version`,
		value.TenantID, value.ID, expected, proposals, value.UpdatedAt)
	updated, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, value.TenantID, value.ID); errors.Is(getErr, ErrNotFound) {
			return Document{}, ErrNotFound
		}
		return Document{}, ErrVersionConflict
	}
	if err != nil {
		return Document{}, fmt.Errorf("save proposal review: %w", err)
	}
	return updated, nil
}

type rowScanner interface{ Scan(...any) error }

func scanDocument(row rowScanner) (Document, error) {
	var value Document
	var limitations, sections, proposals []byte
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.FileName, &value.MediaType, &value.Purpose, &value.SourceType,
		&value.SizeBytes, &value.SHA256, &value.StorageKey, &value.ArtifactStatus, &value.ExtractionStatus, &value.ExtractionMethod,
		&value.AnalysisStatus, &value.AnalysisMethod, &limitations, &sections, &proposals, &value.CreatedBy,
		&value.CreatedAt, &value.UpdatedAt, &value.Version,
	)
	if err != nil {
		return Document{}, err
	}
	if err := json.Unmarshal(limitations, &value.Limitations); err != nil {
		return Document{}, err
	}
	if err := json.Unmarshal(sections, &value.Sections); err != nil {
		return Document{}, err
	}
	if err := json.Unmarshal(proposals, &value.Proposals); err != nil {
		return Document{}, err
	}
	return value, nil
}

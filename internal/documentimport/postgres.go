//go:build postgres

package documentimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, value Document) (Document, error) {
	return insertDocument(ctx, r.pool, value)
}

func (r *PostgresRepository) CreatePending(ctx context.Context, value Document) (Document, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)
	created, err := insertDocument(ctx, tx, value)
	if err != nil {
		return Document{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES(uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DOCUMENT_IMPORT',$2::uuid,$3,'{}'::jsonb,$4,$4,$4)`,
		value.TenantID, value.ID, EventDocumentProcessingRequested, value.CreatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("queue document processing: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return created, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenant string, limit int) ([]DocumentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT di.id::text,t.slug,COALESCE(di.legal_entity_id::text,''),di.file_name,di.media_type,di.purpose,di.source_type,
		       di.size_bytes,di.sha256,di.artifact_status,di.extraction_status,di.analysis_status,
		       di.sections_total,di.sections_omitted,di.proposals_total,di.proposals_omitted,
		       COALESCE((SELECT count(*)::int FROM jsonb_array_elements(di.proposals) p WHERE p->>'status'='PENDING_REVIEW'),0),
		       GREATEST(jsonb_array_length(di.proposals)-COALESCE((SELECT count(*)::int FROM jsonb_array_elements(di.proposals) p WHERE p->>'status'='PENDING_REVIEW'),0),0),
		       di.content_truncated,di.processed_at,di.created_at,di.updated_at,di.version
		FROM document_imports di JOIN tenants t ON t.id=di.tenant_id
		WHERE t.id::text=$1 OR t.slug=$1
		ORDER BY di.created_at DESC,di.id DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, fmt.Errorf("list document imports: %w", err)
	}
	defer rows.Close()
	values := make([]DocumentSummary, 0, limit)
	for rows.Next() {
		value, scanErr := scanSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan document import summary: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document imports: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) Get(ctx context.Context, tenant, id string) (Document, error) {
	row := r.pool.QueryRow(ctx, documentSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid`, tenant, id)
	value, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("load document import: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) ReviewProposal(ctx context.Context, input ReviewInput, now time.Time) (Document, error) {
	handoff := newAcceptedProposalHandoff(input, "", "", now)
	handoffJSON := []byte("{}")
	if input.Status == ProposalAccepted {
		encoded, err := json.Marshal(handoff)
		if err != nil {
			return Document{}, fmt.Errorf("encode proposal handoff: %w", err)
		}
		handoffJSON = encoded
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		WITH target AS (
			SELECT di.id,(p.ordinality-1)::int AS proposal_index,p.value AS proposal
			FROM document_imports di
			JOIN tenants t ON t.id=di.tenant_id
			CROSS JOIN LATERAL jsonb_array_elements(di.proposals) WITH ORDINALITY AS p(value,ordinality)
			WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid AND di.version=$3
			  AND p.value->>'id'=$4 AND p.value->>'status'='PENDING_REVIEW'
		), changed AS (
			UPDATE document_imports di
			SET proposals=jsonb_set(
				di.proposals,
				ARRAY[target.proposal_index::text],
				target.proposal || jsonb_build_object(
					'status',$5::text,
					'reviewed_by',$6::text,
					'reviewed_at',$7::timestamptz,
					'review_note',$8::text
				) || CASE WHEN $5::text='ACCEPTED' THEN jsonb_build_object(
					'handoff', ($9::jsonb || jsonb_build_object(
						'draft_title', target.proposal->>'title',
						'draft_statement', target.proposal->>'statement'
					))
				) ELSE '{}'::jsonb END,
				false
			),updated_at=$7::timestamptz,version=di.version+1
			FROM target WHERE di.id=target.id
			RETURNING di.*
		)
		SELECT c.id::text,t.slug,COALESCE(c.legal_entity_id::text,''),c.file_name,c.media_type,c.purpose,c.source_type,
		       c.size_bytes,c.sha256,c.storage_key,c.artifact_status,c.extraction_status,c.extraction_method,c.analysis_status,c.analysis_method,
		       c.limitations,c.sections,c.proposals,c.tabular_metadata,c.sections_total,c.sections_omitted,c.proposals_total,c.proposals_omitted,
		       c.content_truncated,c.processed_at,c.created_by::text,c.created_at,c.updated_at,c.version
		FROM changed c JOIN tenants t ON t.id=c.tenant_id`,
		input.TenantID, input.DocumentID, input.ExpectedVersion, input.ProposalID, input.Status, input.ReviewerID, now, input.Note, handoffJSON)
	updated, scanErr := scanDocument(row)
	if scanErr == nil {
		if input.Status == ProposalAccepted {
			payload, err := json.Marshal(map[string]string{
				"document_id": input.DocumentID,
				"proposal_id": input.ProposalID,
				"handoff_id":  handoff.ID,
			})
			if err != nil {
				return Document{}, fmt.Errorf("encode proposal handoff event: %w", err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
				VALUES(uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DOCUMENT_IMPORT',$2::uuid,$3,$4::jsonb,$5,$5,$5)`,
				input.TenantID, input.DocumentID, EventDocumentProposalAccepted, payload, now)
			if err != nil {
				return Document{}, fmt.Errorf("queue document proposal handoff: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return Document{}, err
		}
		return updated, nil
	}
	if !errors.Is(scanErr, pgx.ErrNoRows) {
		return Document{}, fmt.Errorf("review document proposal: %w", scanErr)
	}
	current, getErr := r.Get(ctx, input.TenantID, input.DocumentID)
	if getErr != nil {
		return Document{}, getErr
	}
	if current.Version != input.ExpectedVersion {
		for _, proposal := range current.Proposals {
			if proposal.ID == input.ProposalID && proposal.Status == input.Status && proposal.ReviewedBy == input.ReviewerID {
				return current, nil
			}
		}
		return Document{}, ErrVersionConflict
	}
	for _, proposal := range current.Proposals {
		if proposal.ID != input.ProposalID {
			continue
		}
		if proposal.Status == input.Status && proposal.ReviewedBy == input.ReviewerID {
			return current, nil
		}
		if proposal.Status != ProposalPending {
			return Document{}, ErrInvalidReview
		}
	}
	return Document{}, ErrNotFound
}

func (r *PostgresRepository) SaveProcessing(ctx context.Context, value Document, expected int64) (Document, error) {
	limitations, _ := json.Marshal(value.Limitations)
	sections, _ := json.Marshal(value.Sections)
	proposals, _ := json.Marshal(value.Proposals)
	tabular := marshalTabularMetadata(value.Tabular)
	row := r.pool.QueryRow(ctx, `
		UPDATE document_imports SET
			extraction_status=$4,extraction_method=$5,analysis_status=$6,analysis_method=$7,
			limitations=$8::jsonb,sections=$9::jsonb,proposals=$10::jsonb,tabular_metadata=$11::jsonb,
			sections_total=$12,sections_omitted=$13,proposals_total=$14,proposals_omitted=$15,
			content_truncated=$16,processed_at=$17,updated_at=$18,version=version+1
		WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$3
		  AND (extraction_status='PENDING' OR analysis_status='PENDING')
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),file_name,media_type,purpose,source_type,
		          size_bytes,sha256,storage_key,artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,
		          limitations,sections,proposals,tabular_metadata,sections_total,sections_omitted,proposals_total,proposals_omitted,
		          content_truncated,processed_at,created_by::text,created_at,updated_at,version`,
		value.TenantID, value.ID, expected, value.ExtractionStatus, value.ExtractionMethod, value.AnalysisStatus, value.AnalysisMethod,
		limitations, sections, proposals, tabular, value.SectionsTotal, value.SectionsOmitted, value.ProposalsTotal, value.ProposalsOmitted,
		value.ContentTruncated, value.ProcessedAt, value.UpdatedAt)
	updated, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, value.TenantID, value.ID); errors.Is(getErr, ErrNotFound) {
			return Document{}, ErrNotFound
		}
		return Document{}, ErrVersionConflict
	}
	if err != nil {
		return Document{}, fmt.Errorf("save document processing: %w", err)
	}
	return updated, nil
}

type documentQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertDocument(ctx context.Context, queryer documentQueryer, value Document) (Document, error) {
	limitations, _ := json.Marshal(value.Limitations)
	sections, _ := json.Marshal(value.Sections)
	proposals, _ := json.Marshal(value.Proposals)
	tabular := marshalTabularMetadata(value.Tabular)
	row := queryer.QueryRow(ctx, `
		INSERT INTO document_imports(
			id,tenant_id,legal_entity_id,file_name,media_type,purpose,source_type,size_bytes,sha256,storage_key,
			artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,limitations,sections,proposals,tabular_metadata,
			sections_total,sections_omitted,proposals_total,proposals_omitted,content_truncated,processed_at,
			created_by,created_at,updated_at,version)
		SELECT $2::uuid,t.id,
		       CASE WHEN $3='' THEN NULL ELSE (SELECT le.id FROM legal_entities le WHERE le.tenant_id=t.id AND le.id=$3::uuid) END,
		       $4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::jsonb,$17::jsonb,$18::jsonb,$19::jsonb,
		       $20,$21,$22,$23,$24,$25,$26::uuid,$27,$28,1
		FROM tenants t WHERE t.id::text=$1 OR t.slug=$1
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),file_name,media_type,purpose,source_type,
		          size_bytes,sha256,storage_key,artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,
		          limitations,sections,proposals,tabular_metadata,sections_total,sections_omitted,proposals_total,proposals_omitted,
		          content_truncated,processed_at,created_by::text,created_at,updated_at,version`,
		value.TenantID, value.ID, value.LegalEntityID, value.FileName, value.MediaType, value.Purpose, value.SourceType,
		value.SizeBytes, value.SHA256, value.StorageKey, value.ArtifactStatus, value.ExtractionStatus, value.ExtractionMethod,
		value.AnalysisStatus, value.AnalysisMethod, limitations, sections, proposals, tabular,
		value.SectionsTotal, value.SectionsOmitted, value.ProposalsTotal, value.ProposalsOmitted, value.ContentTruncated, value.ProcessedAt,
		value.CreatedBy, value.CreatedAt, value.UpdatedAt)
	created, err := scanDocument(row)
	if err != nil {
		return Document{}, fmt.Errorf("create document import: %w", err)
	}
	return created, nil
}

const documentSelect = `
	SELECT di.id::text,t.slug,COALESCE(di.legal_entity_id::text,''),di.file_name,di.media_type,di.purpose,di.source_type,
	       di.size_bytes,di.sha256,di.storage_key,di.artifact_status,di.extraction_status,di.extraction_method,
	       di.analysis_status,di.analysis_method,di.limitations,di.sections,di.proposals,di.tabular_metadata,
	       di.sections_total,di.sections_omitted,di.proposals_total,di.proposals_omitted,di.content_truncated,di.processed_at,
	       di.created_by::text,di.created_at,di.updated_at,di.version
	FROM document_imports di JOIN tenants t ON t.id=di.tenant_id`

type rowScanner interface{ Scan(...any) error }

func scanDocument(row rowScanner) (Document, error) {
	var value Document
	var limitations, sections, proposals, tabular []byte
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.FileName, &value.MediaType, &value.Purpose, &value.SourceType,
		&value.SizeBytes, &value.SHA256, &value.StorageKey, &value.ArtifactStatus, &value.ExtractionStatus, &value.ExtractionMethod,
		&value.AnalysisStatus, &value.AnalysisMethod, &limitations, &sections, &proposals, &tabular,
		&value.SectionsTotal, &value.SectionsOmitted, &value.ProposalsTotal, &value.ProposalsOmitted, &value.ContentTruncated, &value.ProcessedAt,
		&value.CreatedBy, &value.CreatedAt, &value.UpdatedAt, &value.Version,
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
	if len(tabular) > 0 && string(tabular) != "{}" {
		var metadata TabularMetadata
		if err := json.Unmarshal(tabular, &metadata); err != nil {
			return Document{}, err
		}
		if metadata.ParserVersion != "" {
			value.Tabular = &metadata
		}
	}
	return value, nil
}

func marshalTabularMetadata(value *TabularMetadata) []byte {
	if value == nil {
		return []byte("{}")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return encoded
}

func scanSummary(row rowScanner) (DocumentSummary, error) {
	var value DocumentSummary
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.FileName, &value.MediaType, &value.Purpose, &value.SourceType,
		&value.SizeBytes, &value.SHA256, &value.ArtifactStatus, &value.ExtractionStatus, &value.AnalysisStatus,
		&value.SectionsTotal, &value.SectionsOmitted, &value.ProposalsTotal, &value.ProposalsOmitted,
		&value.PendingProposalCount, &value.ReviewedProposalCount, &value.ContentTruncated, &value.ProcessedAt,
		&value.CreatedAt, &value.UpdatedAt, &value.Version,
	)
	return value, err
}

var _ Repository = (*PostgresRepository)(nil)
var _ QueuedRepository = (*PostgresRepository)(nil)

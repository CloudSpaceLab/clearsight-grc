//go:build postgres

package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const EventFormProposalGenerationRequested = "FORM_TEMPLATE_PROPOSAL_GENERATION_REQUESTED"

type PostgresFormProposalStore struct {
	pool *pgxpool.Pool
}

func NewPostgresFormProposalStore(pool *pgxpool.Pool) *PostgresFormProposalStore {
	return &PostgresFormProposalStore{pool: pool}
}

func (s *PostgresFormProposalStore) QueuesGeneration() bool { return true }

func (s *PostgresFormProposalStore) Create(ctx context.Context, value FormTemplateProposal) (FormTemplateProposal, error) {
	if err := validateNewFormProposal(value); err != nil {
		return FormTemplateProposal{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	created, err := insertFormProposal(ctx, tx, value)
	if errors.Is(err, pgx.ErrNoRows) {
		created, err = selectFormProposalBySource(ctx, tx, value)
		if err != nil {
			return FormTemplateProposal{}, mapPostgresError(err)
		}
		if err := tx.Commit(ctx); err != nil {
			return FormTemplateProposal{}, mapPostgresError(err)
		}
		return created, nil
	}
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	payload, err := json.Marshal(map[string]any{
		"proposal_id":             created.ID,
		"legal_entity_id":         created.LegalEntityID,
		"source_document_id":      created.SourceDocumentID,
		"source_document_version": created.SourceDocumentVersion,
	})
	if err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(
			id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES(
			uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),
			'FORM_TEMPLATE_PROPOSAL',$2::uuid,$3,$4::jsonb,$5,$5,$5
		)`, created.TenantID, created.ID, EventFormProposalGenerationRequested, payload, created.CreatedAt)
	if err != nil {
		return FormTemplateProposal{}, fmt.Errorf("queue form proposal generation: %w", mapPostgresError(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	return created, nil
}

func (s *PostgresFormProposalStore) Get(ctx context.Context, tenantID, legalEntityID, proposalID string) (FormTemplateProposal, error) {
	value, err := scanFormProposal(s.pool.QueryRow(ctx, `
		SELECT `+formProposalProjection+`
		FROM form_template_proposals p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid`,
		strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(proposalID)))
	return value, mapPostgresError(err)
}

func (s *PostgresFormProposalStore) CompleteGeneration(ctx context.Context, value FormTemplateProposal, expectedVersion int64) (FormTemplateProposal, error) {
	contract, changes, unresolved, provenance, err := encodeFormProposalPayload(value)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	updated, err := scanFormProposal(s.pool.QueryRow(ctx, `
		UPDATE form_template_proposals p SET
			status='REVIEW_REQUIRED',proposed_contract=$9::jsonb,field_changes=$10::jsonb,
			unresolved_items=$11::jsonb,provenance=$12::jsonb,failure_code='',failure_message='',
			updated_at=$13,version=p.version+1
		FROM tenants t
		WHERE p.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1)
		  AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid AND p.version=$4 AND p.status='GENERATING'
		  AND p.source_kind=$5 AND p.source_document_id IS NOT DISTINCT FROM NULLIF($6,'')::uuid
		  AND p.source_document_version IS NOT DISTINCT FROM NULLIF($7,0)
		  AND p.source_sha256=$8
		RETURNING `+formProposalProjectionReturning,
		value.TenantID, value.LegalEntityID, value.ID, expectedVersion, value.SourceKind,
		value.SourceDocumentID, value.SourceDocumentVersion, value.SourceSHA256,
		contract, changes, unresolved, provenance, value.UpdatedAt.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return FormTemplateProposal{}, s.classifyMutationMiss(ctx, value.TenantID, value.LegalEntityID, value.ID, expectedVersion, FormProposalGenerating)
	}
	return updated, mapPostgresError(err)
}

func (s *PostgresFormProposalStore) FailGeneration(ctx context.Context, tenantID, legalEntityID, proposalID string, expectedVersion int64, code, message string, at time.Time) (FormTemplateProposal, error) {
	updated, err := scanFormProposal(s.pool.QueryRow(ctx, `
		UPDATE form_template_proposals p SET
			status='FAILED',failure_code=$5,failure_message=$6,updated_at=$7,version=p.version+1
		FROM tenants t
		WHERE p.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1)
		  AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid AND p.version=$4 AND p.status='GENERATING'
		RETURNING `+formProposalProjectionReturning,
		tenantID, legalEntityID, proposalID, expectedVersion, boundedProposalText(code, 128), boundedProposalText(message, 2000), at.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return FormTemplateProposal{}, s.classifyMutationMiss(ctx, tenantID, legalEntityID, proposalID, expectedVersion, FormProposalGenerating)
	}
	return updated, mapPostgresError(err)
}

func (s *PostgresFormProposalStore) Review(ctx context.Context, mutation FormProposalReviewMutation) (FormTemplateProposal, error) {
	if mutation.Status != FormProposalAccepted && mutation.Status != FormProposalRejected {
		return FormTemplateProposal{}, ErrInvalid
	}
	resultID := strings.TrimSpace(mutation.ResultTemplateID)
	resultVersion := mutation.ResultTemplateVersion
	if mutation.Status == FormProposalAccepted {
		if resultID == "" || resultVersion < 1 {
			return FormTemplateProposal{}, ErrInvalid
		}
	} else {
		resultID = ""
		resultVersion = 0
	}
	updated, err := scanFormProposal(s.pool.QueryRow(ctx, `
		UPDATE form_template_proposals p SET
			status=$5,reviewed_by=$6::uuid,reviewed_at=$7,
			result_template_id=NULLIF($8,'')::uuid,result_template_version=NULLIF($9,0),
			updated_at=$7,version=p.version+1
		FROM tenants t
		WHERE p.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1)
		  AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid AND p.version=$4 AND p.status='REVIEW_REQUIRED'
		RETURNING `+formProposalProjectionReturning,
		mutation.TenantID, mutation.LegalEntityID, mutation.ProposalID, mutation.ExpectedVersion,
		mutation.Status, mutation.ReviewerID, mutation.At.UTC(), resultID, resultVersion))
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	current, getErr := s.Get(ctx, mutation.TenantID, mutation.LegalEntityID, mutation.ProposalID)
	if getErr != nil {
		return FormTemplateProposal{}, getErr
	}
	if current.Status == mutation.Status && current.ReviewedBy == mutation.ReviewerID && current.ResultTemplateID == resultID && current.ResultTemplateVersion == resultVersion {
		return current, nil
	}
	if current.Version != mutation.ExpectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	return FormTemplateProposal{}, ErrFormProposalState
}

func (s *PostgresFormProposalStore) classifyMutationMiss(ctx context.Context, tenantID, legalEntityID, proposalID string, expectedVersion int64, expectedStatus FormProposalStatus) error {
	current, err := s.Get(ctx, tenantID, legalEntityID, proposalID)
	if err != nil {
		return err
	}
	if current.Version != expectedVersion {
		return ErrConflict
	}
	if current.Status != expectedStatus {
		return ErrFormProposalState
	}
	return ErrFormProposalSourceChanged
}

func insertFormProposal(ctx context.Context, tx pgx.Tx, value FormTemplateProposal) (FormTemplateProposal, error) {
	created, err := scanFormProposal(tx.QueryRow(ctx, `
		INSERT INTO form_template_proposals AS p(
			id,tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,source_sha256,
			base_template_id,base_template_version,status,created_by,created_at,updated_at,version
		) SELECT
			$2::uuid,t.id,$3::uuid,$4,NULLIF($5,'')::uuid,NULLIF($6,0),$7,
			NULLIF($8,'')::uuid,NULLIF($9,0),$10,$11::uuid,$12,$13,1
		FROM tenants t WHERE t.id::text=$1 OR t.slug=$1
		ON CONFLICT DO NOTHING
		RETURNING `+formProposalProjectionReturning,
		value.TenantID, value.ID, value.LegalEntityID, value.SourceKind, value.SourceDocumentID,
		value.SourceDocumentVersion, value.SourceSHA256, value.BaseTemplateID, value.BaseTemplateVersion,
		value.Status, value.CreatedBy, value.CreatedAt.UTC(), value.UpdatedAt.UTC()))
	return created, err
}

func selectFormProposalBySource(ctx context.Context, tx pgx.Tx, value FormTemplateProposal) (FormTemplateProposal, error) {
	return scanFormProposal(tx.QueryRow(ctx, `
		SELECT `+formProposalProjection+`
		FROM form_template_proposals p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid AND p.source_kind=$3
		  AND p.source_document_id IS NOT DISTINCT FROM NULLIF($4,'')::uuid
		  AND p.source_document_version IS NOT DISTINCT FROM NULLIF($5,0)
		  AND p.source_sha256=$6
		  AND p.base_template_id IS NOT DISTINCT FROM NULLIF($7,'')::uuid
		  AND p.base_template_version IS NOT DISTINCT FROM NULLIF($8,0)`,
		value.TenantID, value.LegalEntityID, value.SourceKind, value.SourceDocumentID,
		value.SourceDocumentVersion, value.SourceSHA256, value.BaseTemplateID, value.BaseTemplateVersion))
}

func encodeFormProposalPayload(value FormTemplateProposal) ([]byte, []byte, []byte, []byte, error) {
	contract, err := json.Marshal(value.ProposedContract)
	if err != nil {
		return nil, nil, nil, nil, errors.Join(ErrInvalid, err)
	}
	changes, err := json.Marshal(value.FieldChanges)
	if err != nil {
		return nil, nil, nil, nil, errors.Join(ErrInvalid, err)
	}
	unresolved, err := json.Marshal(value.UnresolvedItems)
	if err != nil {
		return nil, nil, nil, nil, errors.Join(ErrInvalid, err)
	}
	provenance, err := json.Marshal(value.Provenance)
	if err != nil {
		return nil, nil, nil, nil, errors.Join(ErrInvalid, err)
	}
	return contract, changes, unresolved, provenance, nil
}

const formProposalProjection = `
	p.id::text,t.slug,p.legal_entity_id::text,p.source_kind,
	COALESCE(p.source_document_id::text,''),COALESCE(p.source_document_version,0),p.source_sha256,
	COALESCE(p.base_template_id::text,''),COALESCE(p.base_template_version,0),p.status,
	p.proposed_contract,p.field_changes,p.unresolved_items,p.provenance,p.failure_code,p.failure_message,
	p.created_by::text,COALESCE(p.reviewed_by::text,''),COALESCE(p.result_template_id::text,''),COALESCE(p.result_template_version,0),
	p.created_at,p.updated_at,p.reviewed_at,p.version`

const formProposalProjectionReturning = `
	p.id::text,(SELECT slug FROM tenants WHERE id=p.tenant_id),p.legal_entity_id::text,p.source_kind,
	COALESCE(p.source_document_id::text,''),COALESCE(p.source_document_version,0),p.source_sha256,
	COALESCE(p.base_template_id::text,''),COALESCE(p.base_template_version,0),p.status,
	p.proposed_contract,p.field_changes,p.unresolved_items,p.provenance,p.failure_code,p.failure_message,
	p.created_by::text,COALESCE(p.reviewed_by::text,''),COALESCE(p.result_template_id::text,''),COALESCE(p.result_template_version,0),
	p.created_at,p.updated_at,p.reviewed_at,p.version`

type formProposalScanner interface {
	Scan(...any) error
}

func scanFormProposal(row formProposalScanner) (FormTemplateProposal, error) {
	var value FormTemplateProposal
	var contract, changes, unresolved, provenance []byte
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.SourceKind,
		&value.SourceDocumentID, &value.SourceDocumentVersion, &value.SourceSHA256,
		&value.BaseTemplateID, &value.BaseTemplateVersion, &value.Status,
		&contract, &changes, &unresolved, &provenance, &value.FailureCode, &value.FailureMessage,
		&value.CreatedBy, &value.ReviewedBy, &value.ResultTemplateID, &value.ResultTemplateVersion,
		&value.CreatedAt, &value.UpdatedAt, &value.ReviewedAt, &value.Version,
	); err != nil {
		return FormTemplateProposal{}, err
	}
	if err := json.Unmarshal(contract, &value.ProposedContract); err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	if err := json.Unmarshal(changes, &value.FieldChanges); err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	if err := json.Unmarshal(unresolved, &value.UnresolvedItems); err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	if err := json.Unmarshal(provenance, &value.Provenance); err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	return value, nil
}

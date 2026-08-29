//go:build postgres

package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

type formProposalAtomicAcceptor interface {
	AcceptWithDraft(context.Context, FormProposalReviewMutation, FormTemplate) (FormTemplateProposal, error)
}

// AcceptWithDraft composes the material PostgreSQL transition: the exact source
// snapshot is revalidated while the proposal row is locked, the ordinary form
// revision and its event/outbox are inserted, and only then is the proposal
// marked accepted with the exact selected change IDs. Any failure rolls the
// whole transaction back.
func (s *PostgresFormProposalStore) AcceptWithDraft(ctx context.Context, mutation FormProposalReviewMutation, draft FormTemplate) (FormTemplateProposal, error) {
	if mutation.Status != FormProposalAccepted || strings.TrimSpace(mutation.ReviewerID) == "" {
		return FormTemplateProposal{}, ErrInvalid
	}
	changeIDs := normalizeProposalChangeIDs(mutation.ChangeIDs)
	if len(changeIDs) == 0 || len(changeIDs) > 500 {
		return FormTemplateProposal{}, ErrFormProposalSelection
	}
	if draft.Status != LifecycleDraft || draft.Version < 1 || draft.ID == "" {
		return FormTemplateProposal{}, ErrInvalid
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	current, err := scanFormProposal(tx.QueryRow(ctx, `
		SELECT `+formProposalProjection+`
		FROM form_template_proposals p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid
		FOR UPDATE OF p`, mutation.TenantID, mutation.LegalEntityID, mutation.ProposalID))
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	if current.Status == FormProposalAccepted && current.ReviewedBy == mutation.ReviewerID && slices.Equal(current.AcceptedChangeIDs, changeIDs) {
		return current, nil
	}
	if current.Version != mutation.ExpectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if current.Status != FormProposalReviewRequired {
		return FormTemplateProposal{}, ErrFormProposalState
	}
	if current.TenantID != draft.TenantID || current.LegalEntityID != draft.LegalEntityID {
		return FormTemplateProposal{}, ErrFormProposalSourceChanged
	}
	if sourceSHA, required := proposalAcceptanceSourceSHA256(current); required {
		if sourceSHA == "" {
			return FormTemplateProposal{}, ErrFormProposalSourceChanged
		}
		var sourceValid bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM document_imports di JOIN tenants t ON t.id=di.tenant_id
				WHERE (t.id::text=$1 OR t.slug=$1)
				  AND di.legal_entity_id=$2::uuid AND di.id=$3::uuid
				  AND di.version=$4 AND di.sha256=$5
			)`, current.TenantID, current.LegalEntityID, current.SourceDocumentID, current.SourceDocumentVersion, sourceSHA).Scan(&sourceValid)
		if err != nil {
			return FormTemplateProposal{}, mapPostgresError(err)
		}
		if !sourceValid {
			return FormTemplateProposal{}, ErrFormProposalSourceChanged
		}
	}

	created, err := insertFormRevision(ctx, tx, draft)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	formEvent, err := newMonitoringEvent(created.TenantID, AggregateMonitoringForm, created.ID, created.Version, EventMonitoringFormCreated, created, created.CreatedBy, created.CreatedAt)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, formEvent); err != nil {
		return FormTemplateProposal{}, err
	}

	acceptedJSON, err := json.Marshal(changeIDs)
	if err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	updated, err := scanFormProposal(tx.QueryRow(ctx, `
		UPDATE form_template_proposals p SET
			status='ACCEPTED',reviewed_by=$5::uuid,reviewed_at=$6,accepted_change_ids=$7::jsonb,
			result_template_id=$8::uuid,result_template_version=$9,updated_at=$6,version=p.version+1
		FROM tenants t
		WHERE p.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1)
		  AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid AND p.version=$4 AND p.status='REVIEW_REQUIRED'
		RETURNING `+formProposalProjectionReturning,
		mutation.TenantID, mutation.LegalEntityID, mutation.ProposalID, mutation.ExpectedVersion,
		mutation.ReviewerID, mutation.At.UTC(), acceptedJSON, created.ID, created.Version))
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}

	payload, err := json.Marshal(map[string]any{
		"proposal_id":             updated.ID,
		"legal_entity_id":         updated.LegalEntityID,
		"accepted_change_ids":     updated.AcceptedChangeIDs,
		"result_template_id":      updated.ResultTemplateID,
		"result_template_version": updated.ResultTemplateVersion,
	})
	if err != nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(
			id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES(
			uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),
			'FORM_TEMPLATE_PROPOSAL',$2::uuid,$3,$4::jsonb,$5,$5,$5
		)`, updated.TenantID, updated.ID, EventFormProposalAccepted, payload, mutation.At.UTC()); err != nil {
		return FormTemplateProposal{}, fmt.Errorf("queue form proposal acceptance: %w", mapPostgresError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	return updated, nil
}

var _ formProposalAtomicAcceptor = (*PostgresFormProposalStore)(nil)

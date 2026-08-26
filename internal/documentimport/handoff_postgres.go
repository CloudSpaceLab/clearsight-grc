//go:build postgres

package documentimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ReviewProposalHandoff(ctx context.Context, input HandoffReviewInput, now time.Time) (Document, error) {
	return r.mutateProposalHandoff(ctx, input.TenantID, input.DocumentID, input.ProposalID, input.ExpectedDocumentVersion, input.ExpectedHandoffVersion, now, func(handoff *ProposalHandoff) error {
		if handoff.Status != HandoffAwaitingReview {
			return ErrInvalidHandoff
		}
		if handoff.IntakePrincipalID == input.ActorID {
			return ErrHandoffSegregation
		}
		handoff.ReviewerPrincipalID = input.ActorID
		handoff.ReviewNote = input.Note
		handoff.Route = nil
		switch input.Action {
		case HandoffReviewReturn:
			handoff.Status = HandoffReturned
		case HandoffReviewReject:
			handoff.Status = HandoffRejected
		case HandoffReviewSubmit:
			handoff.Status = HandoffAwaitingAuthorization
			handoff.DraftTitle = input.Title
			handoff.DraftStatement = input.Statement
			handoff.TargetType = input.TargetType
			handoff.TargetProgramID = input.TargetProgramID
			handoff.TargetProgramVersion = input.TargetProgramVersion
		default:
			return ErrInvalidHandoff
		}
		return nil
	})
}

func (r *PostgresRepository) AuthorizeProposalHandoff(ctx context.Context, input HandoffAuthorizationInput, now time.Time) (Document, error) {
	return r.mutateProposalHandoff(ctx, input.TenantID, input.DocumentID, input.ProposalID, input.ExpectedDocumentVersion, input.ExpectedHandoffVersion, now, func(handoff *ProposalHandoff) error {
		if handoff.Status != HandoffAwaitingAuthorization {
			return ErrInvalidHandoff
		}
		if handoff.IntakePrincipalID == input.ActorID || handoff.ReviewerPrincipalID == input.ActorID {
			return ErrHandoffSegregation
		}
		handoff.AuthorizerPrincipalID = input.ActorID
		handoff.AuthorizationNote = input.Note
		handoff.Route = nil
		switch input.Action {
		case HandoffAuthorizeReturn:
			handoff.Status = HandoffReturned
		case HandoffAuthorizeReject:
			handoff.Status = HandoffRejected
		case HandoffAuthorizeApprove:
			handoff.Status = HandoffApproved
			handoff.ResultObjectType = input.ResultObjectType
			handoff.ResultObjectID = input.ResultObjectID
		default:
			return ErrInvalidHandoff
		}
		return nil
	})
}

func (r *PostgresRepository) mutateProposalHandoff(
	ctx context.Context,
	tenant, documentID, proposalID string,
	expectedDocumentVersion, expectedHandoffVersion int64,
	now time.Time,
	mutate func(*ProposalHandoff) error,
) (Document, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, documentSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid FOR UPDATE OF di`, strings.TrimSpace(tenant), strings.TrimSpace(documentID))
	current, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("load proposal handoff: %w", err)
	}
	if current.Version != expectedDocumentVersion {
		return Document{}, ErrVersionConflict
	}

	found := false
	for index := range current.Proposals {
		proposal := &current.Proposals[index]
		if proposal.ID != strings.TrimSpace(proposalID) {
			continue
		}
		found = true
		if proposal.Status != ProposalAccepted || proposal.Handoff == nil || proposal.Handoff.Version != expectedHandoffVersion {
			return Document{}, ErrInvalidHandoff
		}
		if err := mutate(proposal.Handoff); err != nil {
			return Document{}, err
		}
		proposal.Handoff.Version++
		proposal.Handoff.UpdatedAt = now.UTC()
		break
	}
	if !found {
		return Document{}, ErrNotFound
	}

	proposals, err := json.Marshal(current.Proposals)
	if err != nil {
		return Document{}, fmt.Errorf("encode proposal handoff: %w", err)
	}
	command, err := tx.Exec(ctx, `
		UPDATE document_imports
		SET proposals=$3::jsonb,updated_at=$4::timestamptz,version=version+1
		WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$5`,
		tenant, documentID, proposals, now.UTC(), expectedDocumentVersion)
	if err != nil {
		return Document{}, fmt.Errorf("save proposal handoff: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Document{}, ErrVersionConflict
	}

	payload, err := json.Marshal(map[string]string{
		"document_id": documentID,
		"proposal_id": proposalID,
		"handoff_id":  proposalHandoffID(documentID, proposalID),
	})
	if err != nil {
		return Document{}, fmt.Errorf("encode proposal handoff event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES(uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DOCUMENT_IMPORT',$2::uuid,$3,$4::jsonb,$5,$5,$5)`,
		tenant, documentID, EventDocumentProposalHandoffChanged, payload, now.UTC())
	if err != nil {
		return Document{}, fmt.Errorf("queue proposal handoff transition: %w", err)
	}

	updated, err := scanDocument(tx.QueryRow(ctx, documentSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid`, tenant, documentID))
	if err != nil {
		return Document{}, fmt.Errorf("reload proposal handoff: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return updated, nil
}

var _ ProposalHandoffRepository = (*PostgresRepository)(nil)

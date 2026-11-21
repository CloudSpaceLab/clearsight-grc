//go:build postgres

package apiquestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
)

// Approval records an independent review only; association activation and
// runtime source execution remain separate and are not performed here.
func (s *PostgresStore) ReadBindingReviewCandidate(ctx context.Context, input BindingReviewInput) (BindingReviewCandidate, error) {
	actor, err := s.bindingReviewActor(ctx, input, false)
	if err != nil {
		return BindingReviewCandidate{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return BindingReviewCandidate{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	ctx = sourceaccess.WithCatalogProofTransaction(ctx, tx)
	candidate, _, _, err := s.loadBindingCandidate(ctx, tx, actor, input)
	if err != nil {
		return BindingReviewCandidate{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return BindingReviewCandidate{}, ErrUnavailable
	}
	return candidate, nil
}

func (s *PostgresStore) ApproveQuestionBinding(ctx context.Context, input BindingReviewInput) (Receipt, error) {
	actor, err := s.bindingReviewActor(ctx, input, true)
	if err != nil {
		return Receipt{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	ctx = sourceaccess.WithCatalogProofTransaction(ctx, tx)
	candidate, tenant, decision, err := s.loadBindingCandidate(ctx, tx, actor, input)
	if err != nil {
		return Receipt{}, err
	}
	canonical := input
	canonical.Key = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	digest := sha256.Sum256(encoded)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "question-binding-approval:"+tenant+":"+actor.LegalEntityID+":"+actor.PrincipalID+":"+input.Key)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var existing Receipt
	var prior []byte
	err = tx.QueryRow(ctx, `SELECT id::text,proposal_version,request_digest FROM form_question_binding_approvals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND reviewer_id=$3::uuid AND idempotency_key=$4`, tenant, actor.LegalEntityID, actor.PrincipalID, input.Key).Scan(&existing.ID, &existing.Version, &prior)
	if err == nil {
		if !bytes.Equal(prior, digest[:]) {
			return Receipt{}, ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return Receipt{}, ErrUnavailable
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, ErrUnavailable
	}
	var approved bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM form_question_binding_approvals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND proposal_id=$3::uuid)`, tenant, actor.LegalEntityID, input.ProposalID).Scan(&approved); err != nil {
		return Receipt{}, ErrUnavailable
	}
	if approved {
		return Receipt{}, ErrConflict
	}
	rowID, err := id.NewUUIDv7()
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `INSERT INTO form_question_binding_approvals(id,tenant_id,legal_entity_id,proposal_id,proposal_version,reviewer_id,rationale,idempotency_key,request_digest,authority_rule_id,authority_policy_version,reviewed_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8,$9,$10,$11,$12)`, rowID, tenant, actor.LegalEntityID, input.ProposalID, input.ProposalVersion, actor.PrincipalID, input.Rationale, input.Key, digest[:], decision.Resolution.RuleID, decision.Resolution.PolicyVersion, now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	payload, err := json.Marshal(map[string]any{"proposal_id": input.ProposalID, "proposal_version": input.ProposalVersion, "form_id": candidate.FormID, "form_version": candidate.FormVersion, "field_id": candidate.FieldID, "binding_id": candidate.BindingID, "binding_version": candidate.BindingVersion})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_QUESTION_BINDING_APPROVAL',$2::uuid,'FORM_QUESTION_BINDING_APPROVED',$3::jsonb,$4,$4,$4)`, tenant, rowID, string(payload), now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Receipt{}, ErrUnavailable
	}
	return Receipt{ID: rowID, Version: input.ProposalVersion}, nil
}

func (s *PostgresStore) bindingReviewActor(ctx context.Context, input BindingReviewInput, write bool) (identity.Actor, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || s == nil || s.Pool == nil || s.Guard == nil || s.BindingAccess == nil {
		return identity.Actor{}, ErrUnavailable
	}
	if strings.TrimSpace(input.FormID) == "" || input.FormVersion < 1 || strings.TrimSpace(input.ProposalID) == "" || input.ProposalVersion < 1 {
		return identity.Actor{}, ErrInvalid
	}
	if write && (!validIdempotencyKey(input.Key) || strings.TrimSpace(input.Rationale) == "" || len(input.Rationale) > 1000 || !utf8.ValidString(input.Rationale) || strings.ContainsRune(input.Rationale, 0)) {
		return identity.Actor{}, ErrInvalid
	}
	return actor, nil
}

func (s *PostgresStore) loadBindingCandidate(ctx context.Context, tx pgx.Tx, actor identity.Actor, input BindingReviewInput) (BindingReviewCandidate, string, commandauth.Decision, error) {
	var candidate BindingReviewCandidate
	var tenant, proposalMaker, draftMaker, draftID, endpoint string
	var draftVersion int64
	err := tx.QueryRow(ctx, `SELECT f.tenant_id::text,p.actor_id::text,d.actor_id::text,d.id::text,d.version,d.endpoint,p.id::text,p.version,p.form_id::text,p.form_version,p.field_id,p.binding_id::text,p.binding_version,d.result_type FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id JOIN form_question_binding_proposals p ON p.tenant_id=f.tenant_id AND p.legal_entity_id=f.legal_entity_id AND p.form_revision_id=f.revision_id AND p.form_id=f.id AND p.form_version=f.version JOIN api_question_draft_revisions d ON d.id=p.draft_id AND d.version=p.draft_version AND d.tenant_id=p.tenant_id AND d.legal_entity_id=p.legal_entity_id JOIN source_bindings b ON b.binding_id=p.binding_id AND b.tenant_id=p.tenant_id AND b.source_id=p.source_id AND b.version=p.binding_version WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id::text=$2 AND f.id::text=$3 AND f.version=$4 AND f.status='DRAFT' AND p.id::text=$5 AND p.version=$6 AND p.status='PROPOSED' AND d.mode='API' AND b.status='ACTIVE' AND b.is_current FOR UPDATE OF f FOR SHARE OF b`, actor.TenantID, actor.LegalEntityID, input.FormID, input.FormVersion, input.ProposalID, input.ProposalVersion).Scan(&tenant, &proposalMaker, &draftMaker, &draftID, &draftVersion, &endpoint, &candidate.ProposalID, &candidate.ProposalVersion, &candidate.FormID, &candidate.FormVersion, &candidate.FieldID, &candidate.BindingID, &candidate.BindingVersion, &candidate.ResultType)
	if err != nil || actor.PrincipalID == proposalMaker || actor.PrincipalID == draftMaker {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	var latestForm, latestDraft, latestProposal int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`, tenant, actor.LegalEntityID, input.FormID).Scan(&latestForm); err != nil || latestForm != input.FormVersion {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM api_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND form_id=$3::uuid AND form_version=$4 AND field_id=$5`, tenant, actor.LegalEntityID, input.FormID, input.FormVersion, candidate.FieldID).Scan(&latestDraft); err != nil || latestDraft != draftVersion {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM form_question_binding_proposals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND draft_id=$3::uuid`, tenant, actor.LegalEntityID, draftID).Scan(&latestProposal); err != nil || latestProposal != input.ProposalVersion {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_TEMPLATE", ObjectID: input.FormID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.template.transition", Materiality: 3})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	if err = s.BindingAccess.AuthorizeQuestionBinding(ctx, actor.TenantID, actor.LegalEntityID, candidate.FormID, candidate.FormVersion, candidate.FieldID, candidate.BindingID, candidate.BindingVersion, candidate.ResultType, endpoint); err != nil {
		return BindingReviewCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	return candidate, tenant, decision, nil
}

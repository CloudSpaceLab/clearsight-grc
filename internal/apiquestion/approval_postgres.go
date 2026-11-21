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
	"github.com/jackc/pgx/v5"
)

// Adoption approval and source execution remain separate. This checker's
// decision only attests to the exact manual-choice proposal.
func (s *PostgresStore) ReadAdoptionQuestionCandidate(ctx context.Context, input AdoptionQuestionApproval) (AdoptionQuestionCandidate, error) {
	actor, err := s.approvalActor(ctx, input, false)
	if err != nil {
		return AdoptionQuestionCandidate{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AdoptionQuestionCandidate{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	candidate, _, _, err := s.loadApprovalCandidate(ctx, tx, actor, input)
	if err != nil {
		return AdoptionQuestionCandidate{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdoptionQuestionCandidate{}, ErrUnavailable
	}
	return candidate, nil
}

func (s *PostgresStore) ApproveAdoptionQuestion(ctx context.Context, input AdoptionQuestionApproval) (Receipt, error) {
	actor, err := s.approvalActor(ctx, input, true)
	if err != nil {
		return Receipt{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	candidate, tenant, decision, err := s.loadApprovalCandidate(ctx, tx, actor, input)
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
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "adoption-question-approval:"+tenant+":"+actor.LegalEntityID+":"+actor.PrincipalID+":"+input.Key)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var existing Receipt
	var previousDigest []byte
	err = tx.QueryRow(ctx, `SELECT id::text,draft_version,request_digest FROM program_adoption_question_approvals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND reviewer_id=$3::uuid AND idempotency_key=$4`, tenant, actor.LegalEntityID, actor.PrincipalID, input.Key).Scan(&existing.ID, &existing.Version, &previousDigest)
	if err == nil {
		if !bytes.Equal(previousDigest, digest[:]) {
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
	var already bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM program_adoption_question_approvals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND draft_id=$3::uuid)`, tenant, actor.LegalEntityID, input.DraftID).Scan(&already); err != nil {
		return Receipt{}, ErrUnavailable
	}
	if already {
		return Receipt{}, ErrConflict
	}
	rowID, err := id.NewUUIDv7()
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `INSERT INTO program_adoption_question_approvals(id,tenant_id,legal_entity_id,draft_id,draft_version,program_id,adoption_id,adoption_version,reviewer_id,rationale,idempotency_key,request_digest,authority_rule_id,authority_policy_version,reviewed_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7::uuid,$8,$9::uuid,$10,$11,$12,$13,$14,$15)`, rowID, tenant, actor.LegalEntityID, input.DraftID, input.DraftVersion, input.ProgramID, input.AdoptionID, input.AdoptionVersion, actor.PrincipalID, input.Rationale, input.Key, digest[:], decision.Resolution.RuleID, decision.Resolution.PolicyVersion, now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	payload, err := json.Marshal(map[string]any{"program_id": input.ProgramID, "adoption_id": input.AdoptionID, "draft_id": input.DraftID, "draft_version": input.DraftVersion, "form_id": candidate.FormID, "form_version": candidate.FormVersion, "field_id": candidate.FieldID, "mode": "MANUAL"})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'ADOPTION_QUESTION_APPROVAL',$2::uuid,'ADOPTION_QUESTION_APPROVED',$3::jsonb,$4,$4,$4)`, tenant, rowID, string(payload), now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Receipt{}, ErrUnavailable
	}
	return Receipt{ID: rowID, Version: input.DraftVersion}, nil
}

func (s *PostgresStore) approvalActor(ctx context.Context, input AdoptionQuestionApproval, write bool) (identity.Actor, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || s == nil || s.Pool == nil || s.Guard == nil || s.FormAccess == nil {
		return identity.Actor{}, ErrUnavailable
	}
	if strings.TrimSpace(input.ProgramID) == "" || strings.TrimSpace(input.AdoptionID) == "" || strings.TrimSpace(input.DraftID) == "" || input.ProgramVersion < 1 || input.AdoptionVersion < 1 || input.DraftVersion < 1 {
		return identity.Actor{}, ErrInvalid
	}
	if write && (!validIdempotencyKey(input.Key) || strings.TrimSpace(input.Rationale) == "" || len(input.Rationale) > 1000 || !utf8.ValidString(input.Rationale) || strings.ContainsRune(input.Rationale, 0)) {
		return identity.Actor{}, ErrInvalid
	}
	return actor, nil
}

func (s *PostgresStore) loadApprovalCandidate(ctx context.Context, tx pgx.Tx, actor identity.Actor, input AdoptionQuestionApproval) (AdoptionQuestionCandidate, string, commandauth.Decision, error) {
	var candidate AdoptionQuestionCandidate
	var tenant, proposalMaker, draftMaker, revision string
	err := tx.QueryRow(ctx, `SELECT p.tenant_id::text,a.proposed_by::text,d.actor_id::text,d.id::text,d.version,d.form_id::text,d.form_version,d.field_id,d.mode,d.rationale,d.form_revision_id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id JOIN program_framework_adoptions a ON a.tenant_id=p.tenant_id AND a.legal_entity_id=p.legal_entity_id AND a.program_id=p.id JOIN program_adoption_question_draft_revisions d ON d.tenant_id=a.tenant_id AND d.legal_entity_id=a.legal_entity_id AND d.program_id=a.program_id AND d.adoption_id=a.id AND d.adoption_version=a.version WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.id::text=$3 AND p.version=$4 AND p.status<>'RETIRED' AND a.id::text=$5 AND a.version=$6 AND a.status='PENDING' AND d.id::text=$7 AND d.version=$8 FOR UPDATE OF p,a`, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.ProgramVersion, input.AdoptionID, input.AdoptionVersion, input.DraftID, input.DraftVersion).Scan(&tenant, &proposalMaker, &draftMaker, &candidate.DraftID, &candidate.DraftVersion, &candidate.FormID, &candidate.FormVersion, &candidate.FieldID, &candidate.Mode, &candidate.Rationale, &revision)
	if err != nil || actor.PrincipalID == proposalMaker || actor.PrincipalID == draftMaker {
		return AdoptionQuestionCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	var latest int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM program_adoption_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND adoption_id=$3::uuid AND form_id=$4::uuid AND form_version=$5 AND field_id=$6`, tenant, actor.LegalEntityID, input.AdoptionID, candidate.FormID, candidate.FormVersion, candidate.FieldID).Scan(&latest)
	if err != nil || latest != candidate.DraftVersion {
		return AdoptionQuestionCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	var found string
	err = tx.QueryRow(ctx, `SELECT f.revision_id::text FROM monitoring_form_templates f WHERE f.revision_id=$1::uuid AND f.tenant_id=$2::uuid AND f.legal_entity_id=$3::uuid AND f.id=$4::uuid AND f.version=$5 AND f.status='ACTIVE' AND f.is_current AND (f.program_id IS NULL OR f.program_id::text=$6) FOR SHARE OF f`, revision, tenant, actor.LegalEntityID, candidate.FormID, candidate.FormVersion, input.ProgramID).Scan(&found)
	if err != nil {
		return AdoptionQuestionCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	if err = s.FormAccess.AuthorizeAdoptionForm(ctx, actor.TenantID, actor.LegalEntityID, input.ProgramID, candidate.FormID, candidate.FormVersion); err != nil {
		return AdoptionQuestionCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: input.ProgramID, Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "program.framework.approve", Materiality: 4})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return AdoptionQuestionCandidate{}, "", commandauth.Decision{}, ErrUnavailable
	}
	return candidate, tenant, decision, nil
}

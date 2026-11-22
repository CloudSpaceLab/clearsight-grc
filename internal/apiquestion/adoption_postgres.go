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

// ReadAdoptionManualDraft returns the latest proposal only under the same
// scoped visibility and current authority checks as creation.
func (s *PostgresStore) ReadAdoptionManualDraft(ctx context.Context, input AdoptionManualDraft) (AdoptionQuestionDraft, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) || s == nil || s.Pool == nil || s.Guard == nil || s.FormAccess == nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	if strings.TrimSpace(input.ProgramID) == "" || strings.TrimSpace(input.AdoptionID) == "" || strings.TrimSpace(input.FormID) == "" || input.ProgramVersion < 1 || input.AdoptionVersion < 1 || input.FormVersion < 1 || len(input.FieldID) < 1 || len(input.FieldID) > 80 || strings.TrimSpace(input.FieldID) != input.FieldID {
		return AdoptionQuestionDraft{}, ErrInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	var tenant string
	err = tx.QueryRow(ctx, `SELECT p.tenant_id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id JOIN program_framework_adoptions a ON a.tenant_id=p.tenant_id AND a.legal_entity_id=p.legal_entity_id AND a.program_id=p.id WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.id::text=$3 AND p.version=$4 AND p.status<>'RETIRED' AND a.id::text=$5 AND a.version=$6 AND a.status='PENDING' FOR SHARE OF p,a`, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.ProgramVersion, input.AdoptionID, input.AdoptionVersion).Scan(&tenant)
	if err != nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	var revision string
	err = tx.QueryRow(ctx, `SELECT f.revision_id::text FROM monitoring_form_templates f CROSS JOIN LATERAL jsonb_array_elements(f.fields) q(value) WHERE f.tenant_id=$1::uuid AND f.legal_entity_id=$2::uuid AND f.id::text=$3 AND f.version=$4 AND f.status='ACTIVE' AND f.is_current AND (f.program_id IS NULL OR f.program_id::text=$5) AND q.value->>'id'=$6 FOR SHARE OF f`, tenant, actor.LegalEntityID, input.FormID, input.FormVersion, input.ProgramID, input.FieldID).Scan(&revision)
	if err != nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	if err = s.FormAccess.AuthorizeAdoptionForm(ctx, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.FormID, input.FormVersion); err != nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: input.ProgramID, Responsibility: authority.ResponsibilityOwner, DecisionType: "program.framework.propose", Materiality: 3})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	var draft AdoptionQuestionDraft
	var sessionCurrent bool
	err = tx.QueryRow(ctx, `SELECT version,mode,rationale,clock_timestamp() < $6::timestamptz FROM program_adoption_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND adoption_id=$3::uuid AND form_revision_id=$4::uuid AND field_id=$5 ORDER BY version DESC LIMIT 1`, tenant, actor.LegalEntityID, input.AdoptionID, revision, input.FieldID, actor.ExpiresAt).Scan(&draft.Version, &draft.Mode, &draft.Rationale, &sessionCurrent)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdoptionQuestionDraft{}, ErrNotFound
	}
	if err != nil || !sessionCurrent {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return AdoptionQuestionDraft{}, ErrUnavailable
	}
	return draft, nil
}

// WriteAdoptionManualDraft records a proposed explicit association; it never
// changes the source form, activates an override, or changes an assessment.
func (s *PostgresStore) WriteAdoptionManualDraft(ctx context.Context, input AdoptionManualDraft) (Receipt, error) {
	now := time.Now().UTC()
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) || s == nil || s.Pool == nil || s.Guard == nil || s.FormAccess == nil {
		return Receipt{}, ErrUnavailable
	}
	if strings.TrimSpace(input.ProgramID) == "" || strings.TrimSpace(input.AdoptionID) == "" || strings.TrimSpace(input.FormID) == "" || len(input.FieldID) < 1 || len(input.FieldID) > 80 || strings.TrimSpace(input.FieldID) != input.FieldID || input.ProgramVersion < 1 || input.AdoptionVersion < 1 || input.FormVersion < 1 || input.ExpectedVersion < 0 || !validIdempotencyKey(input.Key) || len(input.Rationale) > 1000 || !utf8.ValidString(input.Rationale) || strings.ContainsRune(input.Rationale, 0) {
		return Receipt{}, ErrInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	var tenant, edition string
	err = tx.QueryRow(ctx, `SELECT p.tenant_id::text,a.edition_id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id JOIN program_framework_adoptions a ON a.tenant_id=p.tenant_id AND a.legal_entity_id=p.legal_entity_id AND a.program_id=p.id WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.id::text=$3 AND p.version=$4 AND p.status<>'RETIRED' AND a.id::text=$5 AND a.version=$6 AND a.status='PENDING' FOR UPDATE OF p,a`, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.ProgramVersion, input.AdoptionID, input.AdoptionVersion).Scan(&tenant, &edition)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var formRevision string
	err = tx.QueryRow(ctx, `SELECT f.revision_id::text FROM monitoring_form_templates f CROSS JOIN LATERAL jsonb_array_elements(f.fields) q(value) WHERE f.tenant_id=$1::uuid AND f.legal_entity_id=$2::uuid AND f.id::text=$3 AND f.version=$4 AND f.status='ACTIVE' AND f.is_current AND (f.program_id IS NULL OR f.program_id::text=$5) AND q.value->>'id'=$6 FOR SHARE OF f`, tenant, actor.LegalEntityID, input.FormID, input.FormVersion, input.ProgramID, input.FieldID).Scan(&formRevision)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = s.FormAccess.AuthorizeAdoptionForm(ctx, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.FormID, input.FormVersion); err != nil {
		return Receipt{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: input.ProgramID, Responsibility: authority.ResponsibilityOwner, DecisionType: "program.framework.propose", Materiality: 3})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return Receipt{}, ErrUnavailable
	}
	canonical := input
	canonical.Key = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	digest := sha256.Sum256(encoded)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "adoption-question:"+tenant+":"+actor.LegalEntityID+":"+actor.PrincipalID+":"+input.Key)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var existing Receipt
	var previousDigest []byte
	var sessionCurrent bool
	err = tx.QueryRow(ctx, `SELECT id::text,version,request_digest,clock_timestamp() < $5::timestamptz FROM program_adoption_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND actor_id=$3::uuid AND idempotency_key=$4`, tenant, actor.LegalEntityID, actor.PrincipalID, input.Key, actor.ExpiresAt).Scan(&existing.ID, &existing.Version, &previousDigest, &sessionCurrent)
	if err == nil {
		if !sessionCurrent {
			return Receipt{}, ErrUnavailable
		}
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
	var current int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM program_adoption_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND adoption_id=$3::uuid AND form_id=$4::uuid AND form_version=$5 AND field_id=$6`, tenant, actor.LegalEntityID, input.AdoptionID, input.FormID, input.FormVersion, input.FieldID).Scan(&current)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if current != input.ExpectedVersion {
		return Receipt{}, ErrConflict
	}
	rowID, err := id.NewUUIDv7()
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	err = tx.QueryRow(ctx, `WITH write_time AS MATERIALIZED (SELECT clock_timestamp() AS at) INSERT INTO program_adoption_question_draft_revisions(id,tenant_id,legal_entity_id,program_id,program_version,adoption_id,adoption_version,edition_id,form_revision_id,form_id,form_version,field_id,version,mode,rationale,actor_id,idempotency_key,request_digest,authority_rule_id,authority_policy_version,created_at) SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9::uuid,$10::uuid,$11,$12,$13,'MANUAL',$14,$15::uuid,$16,$17,$18,$19,write_time.at FROM write_time WHERE write_time.at < $20::timestamptz RETURNING created_at`, rowID, tenant, actor.LegalEntityID, input.ProgramID, input.ProgramVersion, input.AdoptionID, input.AdoptionVersion, edition, formRevision, input.FormID, input.FormVersion, input.FieldID, current+1, input.Rationale, actor.PrincipalID, input.Key, digest[:], decision.Resolution.RuleID, decision.Resolution.PolicyVersion, actor.ExpiresAt).Scan(&now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	payload, err := json.Marshal(map[string]any{"program_id": input.ProgramID, "adoption_id": input.AdoptionID, "adoption_version": input.AdoptionVersion, "form_id": input.FormID, "form_version": input.FormVersion, "field_id": input.FieldID, "version": current + 1, "mode": "MANUAL"})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'ADOPTION_QUESTION_DRAFT',$2::uuid,'ADOPTION_QUESTION_DRAFT_REVISED',$3::jsonb,$4,$4,$4)`, tenant, rowID, string(payload), now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Receipt{}, ErrUnavailable
	}
	return Receipt{ID: rowID, Version: current + 1}, nil
}

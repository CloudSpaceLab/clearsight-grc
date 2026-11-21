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

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
)

// ProposeQuestionBinding records an exact association proposal only. It does
// not approve it, alter form fields, activate a binding, or create an observation.
func (s *PostgresStore) ProposeQuestionBinding(ctx context.Context, input BindingProposal) (Receipt, error) {
	now := time.Now().UTC()
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) || s == nil || s.Pool == nil || s.Guard == nil || s.BindingAccess == nil {
		return Receipt{}, ErrUnavailable
	}
	if strings.TrimSpace(input.FormID) == "" || input.FormVersion < 1 || strings.TrimSpace(input.FieldID) != input.FieldID || len(input.FieldID) < 1 || len(input.FieldID) > 80 || strings.TrimSpace(input.DraftID) == "" || input.DraftVersion < 1 || strings.TrimSpace(input.BindingID) == "" || input.BindingVersion < 1 || input.ExpectedVersion < 0 || !validIdempotencyKey(input.Key) {
		return Receipt{}, ErrInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	ctx = sourceaccess.WithCatalogProofTransaction(ctx, tx)
	var tenant, formRevision, resultType, endpoint string
	err = tx.QueryRow(ctx, `SELECT f.tenant_id::text,f.revision_id::text,d.result_type,d.endpoint FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id JOIN api_question_draft_revisions d ON d.tenant_id=f.tenant_id AND d.legal_entity_id=f.legal_entity_id AND d.form_revision_id=f.revision_id AND d.form_id=f.id AND d.form_version=f.version WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id::text=$2 AND f.id::text=$3 AND f.version=$4 AND f.status='DRAFT' AND d.field_id=$5 AND d.id::text=$6 AND d.version=$7 AND d.mode='API' FOR UPDATE OF f`, actor.TenantID, actor.LegalEntityID, input.FormID, input.FormVersion, input.FieldID, input.DraftID, input.DraftVersion).Scan(&tenant, &formRevision, &resultType, &endpoint)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var latestForm, latestDraft int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`, tenant, actor.LegalEntityID, input.FormID).Scan(&latestForm); err != nil || latestForm != input.FormVersion {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM api_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND form_id=$3::uuid AND form_version=$4 AND field_id=$5`, tenant, actor.LegalEntityID, input.FormID, input.FormVersion, input.FieldID).Scan(&latestDraft); err != nil || latestDraft != input.DraftVersion {
		return Receipt{}, ErrUnavailable
	}
	var source string
	err = tx.QueryRow(ctx, `SELECT source_id::text FROM source_bindings WHERE tenant_id=$1::uuid AND binding_id::text=$2 AND version=$3 AND status='ACTIVE' AND is_current FOR SHARE`, tenant, input.BindingID, input.BindingVersion).Scan(&source)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_TEMPLATE", ObjectID: input.FormID, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return Receipt{}, ErrUnavailable
	}
	if err = s.BindingAccess.AuthorizeQuestionBinding(ctx, actor.TenantID, actor.LegalEntityID, input.FormID, input.FormVersion, input.FieldID, input.BindingID, input.BindingVersion, resultType, endpoint); err != nil {
		return Receipt{}, ErrUnavailable
	}
	canonical := input
	canonical.Key = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	digest := sha256.Sum256(encoded)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "question-binding:"+tenant+":"+actor.LegalEntityID+":"+actor.PrincipalID+":"+input.Key)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var existing Receipt
	var previous []byte
	err = tx.QueryRow(ctx, `SELECT id::text,version,request_digest FROM form_question_binding_proposals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND actor_id=$3::uuid AND idempotency_key=$4`, tenant, actor.LegalEntityID, actor.PrincipalID, input.Key).Scan(&existing.ID, &existing.Version, &previous)
	if err == nil {
		if !bytes.Equal(previous, digest[:]) {
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
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM form_question_binding_proposals WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND draft_id=$3::uuid`, tenant, actor.LegalEntityID, input.DraftID).Scan(&current); err != nil {
		return Receipt{}, ErrUnavailable
	}
	if current != input.ExpectedVersion {
		return Receipt{}, ErrConflict
	}
	rowID, err := id.NewUUIDv7()
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO form_question_binding_proposals(id,tenant_id,legal_entity_id,draft_id,draft_version,form_revision_id,form_id,form_version,field_id,binding_id,source_id,binding_version,version,status,actor_id,idempotency_key,request_digest,authority_rule_id,authority_policy_version,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7::uuid,$8,$9,$10::uuid,$11::uuid,$12,$13,'PROPOSED',$14::uuid,$15,$16,$17,$18,$19)`, rowID, tenant, actor.LegalEntityID, input.DraftID, input.DraftVersion, formRevision, input.FormID, input.FormVersion, input.FieldID, input.BindingID, source, input.BindingVersion, current+1, actor.PrincipalID, input.Key, digest[:], decision.Resolution.RuleID, decision.Resolution.PolicyVersion, now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	payload, err := json.Marshal(map[string]any{"draft_id": input.DraftID, "draft_version": input.DraftVersion, "form_id": input.FormID, "form_version": input.FormVersion, "field_id": input.FieldID, "binding_id": input.BindingID, "binding_version": input.BindingVersion, "version": current + 1})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_QUESTION_BINDING_PROPOSAL',$2::uuid,'FORM_QUESTION_BINDING_PROPOSED',$3::jsonb,$4,$4,$4)`, tenant, rowID, string(payload), now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Receipt{}, ErrUnavailable
	}
	return Receipt{ID: rowID, Version: current + 1}, nil
}

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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Authority interface {
	Authorize(context.Context, commandauth.Request) (commandauth.Decision, error)
}

// AdoptionFormAccess validates use of an exact form revision by the Program.
type AdoptionFormAccess interface {
	AuthorizeAdoptionForm(context.Context, string, string, string, string, int64) error
}

// QuestionBindingAccess must independently verify the exact scoped binding,
// operation, adapter and output contract before any proposal is recorded.
type QuestionBindingAccess interface {
	AuthorizeQuestionBinding(context.Context, string, string, string, int64, string, string, int64, string, string) error
}

type PostgresStore struct {
	Pool          *pgxpool.Pool
	Guard         Authority
	FormAccess    AdoptionFormAccess
	BindingAccess QuestionBindingAccess
}

// ConfigureAuthority is called once before the server accepts requests.
func (s *PostgresStore) ConfigureAuthority(guard *commandauth.Guard) { s.Guard = guard }

// ReadDraft returns connection details only to the current, authorized form owner.
func (s *PostgresStore) ReadDraft(ctx context.Context, form string, formVersion int64, field string) (Draft, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) || s == nil || s.Pool == nil || s.Guard == nil || strings.TrimSpace(form) == "" || len(field) < 1 || len(field) > 80 || formVersion < 1 {
		return Draft{}, ErrUnavailable
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Draft{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	var tenant string
	err = tx.QueryRow(ctx, `SELECT f.tenant_id::text FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id CROSS JOIN LATERAL jsonb_array_elements(f.fields) q(value) WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id::text=$2 AND f.id::text=$3 AND f.version=$4 AND f.status='DRAFT' AND q.value->>'id'=$5 FOR SHARE OF f`, actor.TenantID, actor.LegalEntityID, form, formVersion, field).Scan(&tenant)
	if err != nil {
		return Draft{}, ErrUnavailable
	}
	var latest int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`, tenant, actor.LegalEntityID, form).Scan(&latest); err != nil || latest != formVersion {
		return Draft{}, ErrUnavailable
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_TEMPLATE", ObjectID: form, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return Draft{}, ErrUnavailable
	}
	var draft Draft
	err = tx.QueryRow(ctx, `SELECT version,mode,endpoint,result_type FROM api_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND form_id=$3::uuid AND form_version=$4 AND field_id=$5 ORDER BY version DESC LIMIT 1`, tenant, actor.LegalEntityID, form, formVersion, field).Scan(&draft.Version, &draft.Mode, &draft.Endpoint, &draft.ResultType)
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrNotFound
	}
	if err != nil {
		return Draft{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Draft{}, ErrUnavailable
	}
	return draft, nil
}

// WriteDraft writes only a protected draft. It does not configure a runtime
// binding, execute a source, or alter the form field delivered to respondents.
func (s *PostgresStore) WriteDraft(ctx context.Context, form string, formVersion int64, field string, expected int64, input DraftInput) (Receipt, error) {
	now := time.Now().UTC()
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) || s == nil || s.Pool == nil || s.Guard == nil || strings.TrimSpace(form) == "" || len(field) < 1 || len(field) > 80 || formVersion < 1 || expected < 0 || !validIdempotencyKey(input.Key) {
		return Receipt{}, ErrUnavailable
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	var tenant, revision, fieldType string
	err = tx.QueryRow(ctx, `SELECT f.tenant_id::text, f.revision_id::text, q.value->>'type'
		FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
		CROSS JOIN LATERAL jsonb_array_elements(f.fields) q(value)
		WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id::text=$2 AND f.id::text=$3
		AND f.version=$4 AND f.status='DRAFT' AND q.value->>'id'=$5 FOR UPDATE OF f`, actor.TenantID, actor.LegalEntityID, form, formVersion, field).Scan(&tenant, &revision, &fieldType)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var latest int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`, tenant, actor.LegalEntityID, form).Scan(&latest); err != nil || latest != formVersion {
		return Receipt{}, ErrUnavailable
	}
	if err = validateDraft(input, fieldType); err != nil {
		return Receipt{}, err
	}
	decision, err := s.Guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_TEMPLATE", ObjectID: form, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		return Receipt{}, ErrUnavailable
	}
	request, err := json.Marshal(struct {
		Form        string     `json:"form"`
		FormVersion int64      `json:"form_version"`
		Field       string     `json:"field"`
		Expected    int64      `json:"expected"`
		Input       DraftInput `json:"input"`
	}{form, formVersion, field, expected, input})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	digest := sha256.Sum256(request)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tenant+":"+actor.LegalEntityID+":"+actor.PrincipalID+":"+input.Key)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	var existing Receipt
	var priorDigest []byte
	err = tx.QueryRow(ctx, `SELECT id::text,version,request_digest FROM api_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND actor_id=$3::uuid AND idempotency_key=$4`, tenant, actor.LegalEntityID, actor.PrincipalID, input.Key).Scan(&existing.ID, &existing.Version, &priorDigest)
	if err == nil {
		if !bytes.Equal(priorDigest, digest[:]) {
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
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM api_question_draft_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND form_id=$3::uuid AND form_version=$4 AND field_id=$5`, tenant, actor.LegalEntityID, form, formVersion, field).Scan(&current); err != nil {
		return Receipt{}, ErrUnavailable
	}
	if current != expected {
		return Receipt{}, ErrConflict
	}
	rowID, err := id.NewUUIDv7()
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	err = tx.QueryRow(ctx, `WITH write_time AS MATERIALIZED (SELECT clock_timestamp() AS at) INSERT INTO api_question_draft_revisions(id,tenant_id,legal_entity_id,form_revision_id,form_id,form_version,field_id,version,mode,endpoint,result_type,actor_id,authority_rule_id,authority_policy_version,created_at,idempotency_key,request_digest) SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11,$12::uuid,$13,$14,write_time.at,$15,$16 FROM write_time WHERE write_time.at < $17::timestamptz RETURNING created_at`, rowID, tenant, actor.LegalEntityID, revision, form, formVersion, field, current+1, input.Mode, input.Endpoint, input.ResultType, actor.PrincipalID, decision.Resolution.RuleID, decision.Resolution.PolicyVersion, input.Key, digest[:], actor.ExpiresAt).Scan(&now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	// Event payloads contain no endpoint or credential material.
	payload, err := json.Marshal(map[string]any{"form_id": form, "form_version": formVersion, "field_id": field, "version": current + 1, "mode": input.Mode})
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'API_QUESTION_DRAFT',$2::uuid,'API_QUESTION_DRAFT_REVISED',$3::jsonb,$4,$4,$4)`, tenant, rowID, string(payload), now)
	if err != nil {
		return Receipt{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Receipt{}, ErrUnavailable
	}
	return Receipt{ID: rowID, Version: current + 1}, nil
}

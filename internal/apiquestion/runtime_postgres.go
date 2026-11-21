//go:build postgres

package apiquestion

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

type CaptureRequestAccess interface {
	AuthorizeCaptureRequest(context.Context, string) error
	VerifyApprovedQuestionBinding(context.Context, string, string, string) error
}

type DistributionPinAccess interface {
	AuthorizeDistributionPin(context.Context, string, string) error
	VerifyApprovedQuestionBinding(context.Context, string, string, string) error
}

// PinApprovedManual is for the existing distribution-creation transaction.
// The callback must verify its current authority and cannot be selected by
// request data. This method is not wired to any distribution command.
func (s *PostgresStore) PinApprovedManual(ctx context.Context, tx pgx.Tx, access DistributionPinAccess, distribution, field, approvalID string) (string, error) {
	if s == nil || tx == nil || access == nil || strings.TrimSpace(distribution) == "" || strings.TrimSpace(approvalID) == "" || strings.TrimSpace(field) != field || len(field) < 1 || len(field) > 80 {
		return "", ErrUnavailable
	}
	ctx = authority.WithPostgresTransaction(ctx, tx)
	if err := access.AuthorizeDistributionPin(ctx, distribution, approvalID); err != nil {
		return "", ErrUnavailable
	}
	if err := access.VerifyApprovedQuestionBinding(ctx, distribution, field, approvalID); err != nil {
		return "", ErrUnavailable
	}
	var tenant, entity string
	err := tx.QueryRow(ctx, `SELECT dist.tenant_id::text,dist.legal_entity_id::text FROM capture_form_distributions dist
		JOIN program_adoption_question_approvals approval ON approval.tenant_id=dist.tenant_id AND approval.legal_entity_id=dist.legal_entity_id
		JOIN program_adoption_question_draft_revisions draft ON draft.id=approval.draft_id AND draft.tenant_id=approval.tenant_id AND draft.legal_entity_id=approval.legal_entity_id AND draft.version=approval.draft_version
		JOIN program_framework_adoptions adoption ON adoption.id=draft.adoption_id AND adoption.tenant_id=draft.tenant_id AND adoption.legal_entity_id=draft.legal_entity_id AND adoption.program_id=draft.program_id AND adoption.edition_id=draft.edition_id
		JOIN monitoring_form_templates form ON form.revision_id=draft.form_revision_id AND form.tenant_id=draft.tenant_id AND form.legal_entity_id=draft.legal_entity_id AND form.id=draft.form_id AND form.version=draft.form_version
		WHERE dist.id::text=$1 AND approval.id::text=$2 AND draft.field_id=$3 AND draft.mode='MANUAL'
		AND dist.status='DRAFT' AND dist.subject_type='PROGRAM' AND dist.subject_id=draft.program_id
		AND dist.form_template_id=draft.form_id AND dist.form_template_version=draft.form_version AND form.status='ACTIVE' AND form.is_current
		AND adoption.status='ACTIVE' AND adoption.version=draft.adoption_version+1 AND approval.adoption_id=adoption.id AND approval.adoption_version=draft.adoption_version
		AND approval.reviewer_id<>draft.actor_id AND approval.reviewer_id<>adoption.proposed_by
		AND NOT EXISTS (SELECT 1 FROM program_adoption_question_draft_revisions newer WHERE newer.tenant_id=draft.tenant_id AND newer.legal_entity_id=draft.legal_entity_id AND newer.adoption_id=draft.adoption_id AND newer.form_id=draft.form_id AND newer.form_version=draft.form_version AND newer.field_id=draft.field_id AND newer.version>draft.version)
		FOR SHARE OF dist,adoption,form`, distribution, approvalID, field).Scan(&tenant, &entity)
	if err != nil {
		return "", ErrUnavailable
	}
	pin, err := id.NewUUIDv7()
	if err != nil {
		return "", ErrUnavailable
	}
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `INSERT INTO capture_distribution_manual_question_pins(id,tenant_id,legal_entity_id,distribution_id,field_id,approval_id,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7)`, pin, tenant, entity, distribution, field, approvalID, now)
	if err != nil {
		return "", ErrUnavailable
	}
	payload, err := json.Marshal(map[string]any{"distribution_id": distribution, "field_id": field, "approval_id": approvalID, "mode": "MANUAL"})
	if err != nil {
		return "", ErrUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'DISTRIBUTION_MANUAL_QUESTION_PIN',$2::uuid,'DISTRIBUTION_MANUAL_QUESTION_PINNED',$3::jsonb,$4,$4,$4)`, tenant, pin, string(payload), now)
	if err != nil {
		return "", ErrUnavailable
	}
	return pin, nil
}

// ResolvePinnedManual is not wired to capture. The caller must pin the choice
// during distribution creation, authorize the request in this transaction,
// and use the result before the same transaction ends.
func (s *PostgresStore) ResolvePinnedManual(ctx context.Context, tx pgx.Tx, access CaptureRequestAccess, request, field string) (bool, error) {
	if s == nil || tx == nil || access == nil || strings.TrimSpace(request) == "" || strings.TrimSpace(field) != field || len(field) < 1 || len(field) > 80 {
		return false, ErrUnavailable
	}
	ctx = authority.WithPostgresTransaction(ctx, tx)
	if err := access.AuthorizeCaptureRequest(ctx, request); err != nil {
		return false, ErrUnavailable
	}
	var found, approvalID string
	err := tx.QueryRow(ctx, `SELECT pin.id::text,approval.id::text
		FROM capture_requests r
		JOIN capture_form_distributions dist ON dist.id=r.distribution_id AND dist.tenant_id=r.tenant_id AND dist.legal_entity_id=r.legal_entity_id
		JOIN capture_distribution_manual_question_pins pin ON pin.distribution_id=dist.id AND pin.tenant_id=dist.tenant_id AND pin.legal_entity_id=dist.legal_entity_id
		JOIN program_adoption_question_approvals approval ON approval.id=pin.approval_id AND approval.tenant_id=pin.tenant_id AND approval.legal_entity_id=pin.legal_entity_id
		JOIN program_adoption_question_draft_revisions draft ON draft.id=approval.draft_id AND draft.tenant_id=approval.tenant_id AND draft.legal_entity_id=approval.legal_entity_id AND draft.version=approval.draft_version
		JOIN program_framework_adoptions adoption ON adoption.id=draft.adoption_id AND adoption.tenant_id=draft.tenant_id AND adoption.legal_entity_id=draft.legal_entity_id AND adoption.program_id=draft.program_id AND adoption.edition_id=draft.edition_id
		WHERE r.id::text=$1 AND pin.field_id=$2 AND draft.field_id=pin.field_id AND draft.mode='MANUAL'
		AND r.subject_type='PROGRAM' AND dist.subject_type='PROGRAM' AND r.subject_id=dist.subject_id::text AND draft.program_id=dist.subject_id
		AND r.form_template_id=dist.form_template_id AND r.form_template_version=dist.form_template_version AND draft.form_id=dist.form_template_id AND draft.form_version=dist.form_template_version
		AND r.status IN ('READY','IN_PROGRESS') AND dist.status='OPEN' AND r.deadline>clock_timestamp() AND dist.deadline>clock_timestamp()
		AND adoption.status='ACTIVE' AND adoption.version=draft.adoption_version+1 AND approval.adoption_id=adoption.id AND approval.adoption_version=draft.adoption_version
		AND approval.reviewer_id<>draft.actor_id AND approval.reviewer_id<>adoption.proposed_by
		AND EXISTS (SELECT 1 FROM jsonb_array_elements(r.fields) q WHERE q->>'id'=pin.field_id)
		AND NOT EXISTS (SELECT 1 FROM program_adoption_question_draft_revisions newer WHERE newer.tenant_id=draft.tenant_id AND newer.legal_entity_id=draft.legal_entity_id AND newer.adoption_id=draft.adoption_id AND newer.form_id=draft.form_id AND newer.form_version=draft.form_version AND newer.field_id=draft.field_id AND newer.version>draft.version)
		FOR SHARE OF r,dist,adoption`, request, field).Scan(&found, &approvalID)
	if err != nil || found == "" {
		return false, ErrUnavailable
	}
	if err = access.VerifyApprovedQuestionBinding(ctx, request, field, approvalID); err != nil {
		return false, ErrUnavailable
	}
	return true, nil
}

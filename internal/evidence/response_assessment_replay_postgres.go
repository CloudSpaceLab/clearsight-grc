//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

type assessmentReplayRequest struct {
	Key             string
	ExpectedVersion int64
	Decisions       []FieldAssessmentInput
}

func assessmentReplayPayload(expected int64, decisions []FieldAssessmentInput) ([]byte, error) {
	return json.Marshal(struct {
		ExpectedVersion int64                  `json:"expected_version"`
		Decisions       []FieldAssessmentInput `json:"decisions"`
	}{expected, decisions})
}

// lookupAssessmentReplay is unregistered. It reads only the requested historical
// receipt under the current route, field roles, subject scope and session.
func lookupAssessmentReplay(ctx context.Context, tx pgx.Tx, actor identity.Actor, material assessmentMaterial, request assessmentReplayRequest) (assessmentSnapshot, error) {
	verified, ok := identity.FromContext(ctx)
	if tx == nil || !ok || verified.Valid(time.Now()) != nil || verified.PrincipalID != actor.PrincipalID || verified.TenantID != actor.TenantID || verified.LegalEntityID != actor.LegalEntityID || !verified.ExpiresAt.Equal(actor.ExpiresAt) || !verified.IssuedAt.Equal(actor.IssuedAt) || !strings.EqualFold(strings.TrimSpace(verified.Kind), "PERSON") || !responseReviewRequestKeyPattern.MatchString(request.Key) || request.ExpectedVersion < 0 || len(request.Decisions) == 0 || len(request.Decisions) > formcontract.MaxFields || material.Summary.ID != material.Revision.ID || material.Summary.TenantID == "" || material.Summary.LegalEntityID == "" || actor.ExpiresAt.IsZero() {
		return assessmentSnapshot{}, ErrAssessmentForbidden
	}
	session, err := federation.LockVerifiedOIDCSession(ctx, tx, actor)
	if err != nil {
		if errors.Is(err, federation.ErrOIDCSessionProofUnavailable) {
			return assessmentSnapshot{}, ErrAssessmentForbidden
		}
		return assessmentSnapshot{}, err
	}
	fieldIDs := make([]string, 0, len(request.Decisions))
	seen := map[string]bool{}
	for _, d := range request.Decisions {
		if strings.TrimSpace(d.FieldID) == "" || seen[d.FieldID] {
			return assessmentSnapshot{}, ErrAssessmentForbidden
		}
		seen[d.FieldID] = true
		fieldIDs = append(fieldIDs, d.FieldID)
	}
	route, err := authority.PostgresCurrentRouteRequest(authority.ResolveInput{TenantID: material.Summary.TenantID, LegalEntityID: material.Summary.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: material.Revision.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
	if err != nil {
		return assessmentSnapshot{}, ErrAssessmentForbidden
	}
	payload, err := assessmentReplayPayload(request.ExpectedVersion, request.Decisions)
	if err != nil {
		return assessmentSnapshot{}, err
	}
	var issued *time.Time
	if !actor.IssuedAt.IsZero() {
		issued = &actor.IssuedAt
	}
	statement := authority.PostgresCurrentRouteCTEs() + ` SELECT DISTINCT prior.version,prior.state,prior.required_count,prior.reviewed_required_count,prior.reviewed_count,prior.score_result,prior.decisions,(prior.request_payload=$3::jsonb AND prior.version=$4::bigint+1) ` + responseReviewEligibilitySQL(`JOIN capture_response_assessments prior ON prior.tenant_id=r.tenant_id AND prior.legal_entity_id=r.legal_entity_id AND prior.response_revision_id=r.id AND prior.request_reviewer_id=actor.id AND prior.request_key=$2::uuid`) + `
 AND r.revision=$15::bigint AND $5::text='' AND $7::text='' AND EXISTS(SELECT 1 FROM web_sessions staff_session WHERE staff_session.token=$17 AND staff_session.data=$18::bytea AND staff_session.expiry>statement_timestamp() AND staff_session.expiry>clock_timestamp()) AND NOT EXISTS (SELECT 1 FROM unnest($16::text[]) required(field_id) WHERE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(req.fields)='array' THEN req.fields ELSE '[]'::jsonb END) field WHERE field->>'id'=required.field_id AND (SELECT count(*) FROM jsonb_array_elements(CASE WHEN jsonb_typeof(req.fields)='array' THEN req.fields ELSE '[]'::jsonb END) other_field WHERE other_field->>'id'=required.field_id)=1 AND COALESCE(btrim(field->'assessment'->>'reviewer_role'),'')<>'' AND field->'assessment'->>'mode' IN ('MANUAL','AUTOMATIC_REVIEW') AND EXISTS (SELECT 1 FROM effective role_origin WHERE role_origin.request_index=0 AND role_origin.principal_id=actor.id AND (upper(role_origin.seed_role_code)=upper(field->'assessment'->>'reviewer_role') OR (role_origin.principal_id=role_origin.origin_id AND role_origin.seed_role_code='' AND upper(role_origin.seed_role)=upper(field->'assessment'->>'reviewer_role'))))))`
	var v assessmentSnapshot
	var score, decisions []byte
	var matches bool
	err = tx.QueryRow(ctx, statement, string(route), request.Key, string(payload), request.ExpectedVersion, "", actor.PrincipalID, "", material.Revision.ID, material.Summary.TenantID, material.Summary.LegalEntityID, actor.TenantID, actor.LegalEntityID, actor.ExpiresAt, issued, material.Revision.Revision, fieldIDs, session.TokenHash, session.Data).Scan(&v.Version, &v.Result.State, &v.Result.RequiredCount, &v.Result.ReviewedRequiredCount, &v.Result.ReviewedCount, &score, &decisions, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return assessmentSnapshot{}, ErrAssessmentForbidden
	}
	if err != nil {
		return assessmentSnapshot{}, err
	}
	if !matches {
		return assessmentSnapshot{}, ErrAssessmentConflict
	}
	v.Result.Version = v.Version
	if err = json.Unmarshal(score, &v.Result.Score); err != nil {
		return assessmentSnapshot{}, err
	}
	if err = json.Unmarshal(decisions, &v.Decisions); err != nil {
		return assessmentSnapshot{}, err
	}
	return v, nil
}

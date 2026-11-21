//go:build postgres

package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgtype"
)

// listPendingProgramReviews evaluates both subject visibility and the reviewer
// route within the bounded source query, before its keyset limit. It never
// treats the browser or a previously loaded list as an authorization boundary.
func (s *PostgresDistributionStore) listPendingProgramReviews(ctx context.Context, actor identity.Actor, query ProgramReviewQuery) (ProgramReviewPage, error) {
	position, err := parseProgramReviewCursor(query.Cursor, actor, query.LegalEntityID)
	if err != nil {
		return ProgramReviewPage{}, err
	}
	var entity, principal, afterID pgtype.UUID
	if entity.Scan(query.LegalEntityID) != nil || !entity.Valid || principal.Scan(actor.PrincipalID) != nil || !principal.Valid {
		return ProgramReviewPage{}, ErrDistributionInvalid
	}
	if position.ID != "" && (afterID.Scan(position.ID) != nil || !afterID.Valid) {
		return ProgramReviewPage{}, ErrDistributionInvalid
	}
	var afterTime *time.Time
	if !position.CompletedAt.IsZero() {
		afterTime = &position.CompletedAt
	}
	evaluation := "GREATEST($4::timestamptz,statement_timestamp())"
	subject := strings.ReplaceAll(strings.ReplaceAll(completedResponseVisibilitySQL(3, 4), "$4", evaluation), "$3", "$3::text")
	route := strings.ReplaceAll(authority.PostgresReadRouteSQL("r", "id", "FORM_RESPONSE", "FORMS.RESPONSE.REVIEW", "REVIEWER", 3, 4), "$4", evaluation)
	rows, err := s.repo.pool.Query(ctx, `
		SELECT `+completedResponseRevisionProjection()+`, d.form_template_id::text,d.form_template_version,d.title,d.subject_type,d.subject_id::text,COALESCE(submission.submitted_by::text,'')
		FROM capture_response_revisions r
		JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		`+completedResponseRequestJoinsSQL()+`
		WHERE r.tenant_id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid ELSE NULL END FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND r.legal_entity_id=$2 AND $5::timestamptz>`+evaluation+`
		  AND d.subject_type='PROGRAM' AND r.is_current AND r.state IN ('FINAL','PROVISIONAL') AND d.status NOT IN ('REVOKED','SUPERSEDED')
		  AND submission.id IS NOT NULL AND req.id IS NOT NULL AND COALESCE(req.origin_type,'') NOT IN ('THIRD_PARTY_WORK','THIRD_PARTY_ASSESSMENT')
		  AND `+documentRevisionScopeSQL()+`
		  AND COALESCE(submission.submitted_by::text,'')<>$3::text
		  AND (COALESCE(req.recipient_type,'')<>'INTERNAL_PRINCIPAL' OR (req.recipient_principal_id IS NOT NULL AND req.recipient_principal_id::text<>$3::text))
		  AND COALESCE(r.submitted_recipient_principal_id::text,'')<>$3::text
		  AND (r.submitted_recipient_id IS NULL OR EXISTS(SELECT 1 FROM capture_distribution_recipients sr WHERE sr.id=r.submitted_recipient_id AND sr.tenant_id=r.tenant_id AND sr.legal_entity_id=r.legal_entity_id AND sr.distribution_id=r.distribution_id AND sr.request_id=submission.request_id AND sr.role='TO'))
		  AND NOT EXISTS(SELECT 1 FROM demo_form_distribution_archives archive WHERE archive.distribution_id=d.id AND archive.tenant_id=d.tenant_id AND archive.legal_entity_id=d.legal_entity_id AND archive.restored_at IS NULL)
		  AND NOT EXISTS(SELECT 1 FROM capture_response_reviews reviewed WHERE reviewed.tenant_id=r.tenant_id AND reviewed.legal_entity_id=r.legal_entity_id AND reviewed.response_revision_id=r.id)
		  AND (`+subject+`) AND (`+route+`)
		  AND ($6::timestamptz IS NULL OR (r.created_at,r.id)<($6,$7::uuid))
		ORDER BY r.created_at DESC,r.id DESC LIMIT $8`, actor.TenantID, entity, actor.PrincipalID, time.Now().UTC(), actor.ExpiresAt, afterTime, afterID, query.Limit+1)
	if err != nil {
		return ProgramReviewPage{}, fmt.Errorf("list program reviews: %w", err)
	}
	defer rows.Close()
	values := make([]CompletedResponseSummary, 0, query.Limit+1)
	for rows.Next() {
		var formID, title, subjectType, subjectID, submittedBy string
		var formVersion int64
		revision, err := scanPostgresResponseRevisionWithExtra(rows, &formID, &formVersion, &title, &subjectType, &subjectID, &submittedBy)
		if err != nil {
			return ProgramReviewPage{}, err
		}
		values = append(values, CompletedResponseSummary{ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID, DistributionID: revision.DistributionID, FormTemplateID: formID, FormTemplateVersion: formVersion, Title: title, SubjectType: subjectType, SubjectID: subjectID, Revision: revision.Revision, Current: revision.Current, State: revision.State, CompletedAt: revision.CreatedAt.UTC(), SubmittedBy: submittedBy, SubmissionRecipient: responseSubmissionRecipient(revision)})
	}
	if err := rows.Err(); err != nil {
		return ProgramReviewPage{}, err
	}
	page := ProgramReviewPage{Items: values, Coverage: "PROGRAM_RESPONSE_REVIEWS"}
	if len(values) > query.Limit {
		page.Items = values[:query.Limit]
		page.NextCursor = makeProgramReviewCursor(actor, query.LegalEntityID, page.Items[len(page.Items)-1])
	}
	return page, nil
}

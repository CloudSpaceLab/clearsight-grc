//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *PostgresDistributionStore) ReadCompletedResponseAnswers(ctx context.Context, tenant, entity, principal, submissionID string) (CompletedResponseAnswers, error) {
	return readCompletedResponseAnswers(ctx, s.repo, tenant, entity, submissionID, func(Request, Submission) error {
		if principal == "" {
			return ErrNotFound
		}
		var allowed bool
		err := s.repo.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 `+documentSubmissionJoinsSQL()+`
 WHERE (t.id::text=$1 OR t.slug=$1) AND req.legal_entity_id=$2::uuid AND submission.id=$5::uuid
 AND `+documentRevisionScopeSQL()+` AND (`+documentReadAuthoritySQL()+`))`, tenant, entity, principal, time.Now().UTC(), submissionID).Scan(&allowed)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrNotFound
		}
		return nil
	})
}

// ListDocuments resolves the immutable submitted occurrence first. Artifact's
// first submission is deliberately not used as membership for later revisions.
func (s *PostgresDistributionStore) ListDocuments(ctx context.Context, q DocumentQuery) (DocumentPage, error) {
	cursor, err := normalizeDocumentQuery(&q)
	if err != nil {
		return DocumentPage{}, err
	}
	for _, value := range []string{q.LegalEntityID, q.PrincipalID, q.FormTemplateID, q.RelationshipID, q.ResponseRevisionID, q.SubmissionID, q.ArtifactID} {
		if value != "" {
			var parsed pgtype.UUID
			if parsed.Scan(value) != nil {
				return DocumentPage{}, ErrDistributionInvalid
			}
		}
	}
	rows, err := s.repo.pool.Query(ctx, documentInventorySQL(), q.TenantID, q.LegalEntityID, q.PrincipalID, time.Now().UTC(), q.FormTemplateID, q.RelationshipID, q.ResponseRevisionID, q.CurrentOnly, string(q.FileKind), q.Query, cursor.SubmittedAt, cursor.ID, q.SubmissionID, q.FieldID, q.ArtifactID, q.Limit+1)
	if err != nil {
		return DocumentPage{}, err
	}
	defer rows.Close()
	values := []DocumentOccurrence{}
	for rows.Next() {
		var raw []byte
		var artifactRequest string
		if err := rows.Scan(&raw, &artifactRequest); err != nil {
			return DocumentPage{}, err
		}
		var v DocumentOccurrence
		if err := json.Unmarshal(raw, &v); err != nil {
			return DocumentPage{}, err
		}
		v.ArtifactRequestID = artifactRequest
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return DocumentPage{}, err
	}
	return documentPage(values, q.Limit), nil
}

func documentInventorySQL() string {
	return `WITH occurrences AS (
 SELECT COALESCE(r.id::text,'legacy')||':'||submission.id::text||':'||(field->>'id')||':'||artifact.id::text AS id,
 artifact.id::text AS artifact_id,req.id::text AS request_id,submission.id::text AS submission_id,field->>'id' AS field_id,
 r.id::text AS response_revision_id,d.id::text AS distribution_id,
 COALESCE(assessment.relationship_id,work.relationship_id,CASE WHEN d.subject_type='VENDOR_RELATIONSHIP' THEN d.subject_id END)::text AS relationship_id,
 assessment.id::text AS assessment_id,work.id::text AS work_request_id,
 req.form_template_id::text AS form_template_id,COALESCE(req.form_template_version,0) AS form_template_version,
 req.title AS form_title,field->>'label' AS field_label,artifact.file_name,artifact.media_type,
 ` + documentKindSQL("artifact.media_type") + ` AS file_kind,
 artifact.size_bytes,artifact.sha256,artifact.status AS artifact_status,artifact.created_at AS uploaded_at,artifact.created_by::text AS uploaded_by,
 submission.submitted_at,submission.submitted_by::text AS submitted_by,
 COALESCE(review.expires_on::text,submission.answers->(field->>'id')->'document'->>'expires_on','') AS expires_on,
 ` + documentCurrentSQL() + ` AS current,
 CASE WHEN review.id IS NOT NULL THEN jsonb_build_object('id',review.id::text,'status',review.status,'reviewed_by',review.validated_by_principal_id::text,'reviewed_at',review.validated_at,'source','VENDOR_ASSESSMENT') END AS review,
 artifact.request_id::text AS artifact_request_id
 ` + documentSubmissionJoinsSQL() + `
 JOIN LATERAL jsonb_array_elements(req.fields) field ON field->>'type' IN ('file','photo','vendor_document')
 JOIN LATERAL (
 SELECT DISTINCT value AS artifact_id FROM jsonb_array_elements_text(
 CASE WHEN field->>'type'='vendor_document' THEN CASE WHEN submission.answers->(field->>'id')->'document'->>'artifact_id' IS NOT NULL THEN jsonb_build_array(submission.answers->(field->>'id')->'document'->>'artifact_id') ELSE '[]'::jsonb END
 WHEN jsonb_typeof(submission.answers->(field->>'id')->'artifact_ids')='array' THEN submission.answers->(field->>'id')->'artifact_ids' ELSE '[]'::jsonb END)
 ) answer_artifact ON true
 JOIN capture_artifacts artifact ON artifact.id=CASE WHEN answer_artifact.artifact_id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN answer_artifact.artifact_id::uuid END AND artifact.tenant_id=req.tenant_id
 JOIN capture_requests artifact_request ON artifact_request.id=artifact.request_id AND artifact_request.tenant_id=req.tenant_id AND artifact_request.legal_entity_id=req.legal_entity_id
 AND (artifact.request_id=req.id OR (r.id IS NOT NULL AND artifact_request.distribution_id=r.distribution_id))
 LEFT JOIN third_party_documents review ON review.tenant_id=req.tenant_id AND review.legal_entity_id=req.legal_entity_id AND review.assessment_id=assessment.id AND review.request_id=req.id AND review.artifact_id=artifact.id AND artifact.submission_id=submission.id
 WHERE (t.id::text=$1 OR t.slug=$1) AND req.legal_entity_id=$2::uuid
 AND ` + documentRevisionScopeSQL() + `
 AND ($5='' OR req.form_template_id=NULLIF($5,'')::uuid) AND ($7='' OR r.id=NULLIF($7,'')::uuid)
 AND ($13='' OR submission.id=NULLIF($13,'')::uuid) AND ($14='' OR field->>'id'=$14) AND ($15='' OR artifact.id=NULLIF($15,'')::uuid)
 AND (` + documentReadAuthoritySQL() + `)
 ) SELECT to_jsonb(o)-'artifact_request_id',artifact_request_id FROM occurrences o
 WHERE ($6='' OR relationship_id=$6) AND (NOT $8::boolean OR current) AND ($9='' OR file_kind=$9)
 AND ($10='' OR strpos(lower(file_name),lower($10))>0)
 AND ($12='' OR (submitted_at,id)<($11::timestamptz,$12))
 ORDER BY submitted_at DESC,id DESC LIMIT $16`
}

// Shared submission scope deliberately contains no file/answer joins: scalar
// answers require the same workflow permission even when the file list is empty.
func documentSubmissionJoinsSQL() string {
	return `FROM capture_submissions submission
 JOIN tenants t ON t.id=submission.tenant_id
 JOIN capture_requests req ON req.id=submission.request_id AND req.tenant_id=submission.tenant_id
 LEFT JOIN capture_response_revisions r ON r.submission_id=submission.id AND r.tenant_id=req.tenant_id AND r.legal_entity_id=req.legal_entity_id
 LEFT JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=req.tenant_id AND d.legal_entity_id=req.legal_entity_id
 LEFT JOIN third_party_assessment_request_links assessment_link ON assessment_link.request_id=req.id AND assessment_link.tenant_id=req.tenant_id AND assessment_link.legal_entity_id=req.legal_entity_id AND req.origin_type=assessment_link.origin_type AND req.origin_id=assessment_link.origin_id::text AND req.origin_version=assessment_link.origin_sequence
 LEFT JOIN third_party_assessments assessment ON assessment.id=assessment_link.assessment_id AND assessment.tenant_id=req.tenant_id AND assessment.legal_entity_id=req.legal_entity_id AND req.subject_type='VENDOR_RELATIONSHIP' AND req.subject_id=assessment.relationship_id::text AND req.form_template_id=assessment.form_template_id AND req.form_template_version=assessment.form_template_version
 LEFT JOIN third_party_work_capture_links work_link ON work_link.request_id=req.id AND work_link.tenant_id=req.tenant_id AND work_link.legal_entity_id=req.legal_entity_id AND req.origin_type=work_link.origin_type AND req.origin_id=work_link.origin_id::text AND req.origin_version=work_link.origin_version
 LEFT JOIN third_party_work_requests work ON work.id=work_link.work_request_id AND work.tenant_id=req.tenant_id AND work.legal_entity_id=req.legal_entity_id AND req.subject_type='VENDOR_RELATIONSHIP' AND req.subject_id=work.relationship_id::text AND req.form_template_id=work.form_template_id AND req.form_template_version=work.form_template_version
 LEFT JOIN third_party_relationships relationship ON relationship.id=COALESCE(assessment.relationship_id,work.relationship_id) AND relationship.tenant_id=req.tenant_id AND relationship.legal_entity_id=req.legal_entity_id`
}

func documentRevisionScopeSQL() string {
	return `(r.id IS NULL OR (req.distribution_id=r.distribution_id AND req.subject_type=d.subject_type AND req.subject_id=d.subject_id::text AND req.form_template_id=d.form_template_id AND req.form_template_version=d.form_template_version))`
}

func documentReadAuthoritySQL() string {
	assessmentRoute := authority.PostgresReadRouteSQL("assessment", "id", "THIRD_PARTY_ASSESSMENT", "THIRDPARTY.ASSESSMENT.REVIEW", "REVIEWER", 3, 4)
	workReviewer := authority.PostgresReadRouteSQL("work", "relationship_id", "VENDOR_RELATIONSHIP", "THIRDPARTY.WORK.REVIEW", "REVIEWER", 3, 4)
	workOwner := authority.PostgresReadRouteSQL("work", "relationship_id", "VENDOR_RELATIONSHIP", "THIRDPARTY.WORK.SEND", "OWNER", 3, 4)
	subjectVisibility := strings.ReplaceAll(completedResponseSubjectVisibilitySQL("$3"), "ELSE true", "ELSE false")
	return `CASE
 WHEN req.origin_type='THIRD_PARTY_ASSESSMENT' THEN assessment.id IS NOT NULL AND (assessment.started_by_principal_id::text=$3 OR relationship.business_owner_principal_id::text=$3 OR ` + assessmentRoute + `)
 WHEN req.origin_type='THIRD_PARTY_WORK' THEN work.id IS NOT NULL
 AND CASE work.target_type
 WHEN 'PROGRAM' THEN EXISTS(SELECT 1 FROM programs target WHERE target.id=work.target_id AND target.tenant_id=req.tenant_id AND target.legal_entity_id=req.legal_entity_id AND ` + recipientSubjectVisibilityPredicate("target", "$3") + `)
 WHEN 'MATTER' THEN EXISTS(SELECT 1 FROM matters target WHERE target.id=work.target_id AND target.tenant_id=req.tenant_id AND target.legal_entity_id=req.legal_entity_id AND ` + recipientSubjectVisibilityPredicate("target", "$3") + `)
 ELSE false END
 AND (work.owner_principal_id::text=$3 OR work.reviewer_principal_id::text=$3 OR relationship.business_owner_principal_id::text=$3 OR ` + workReviewer + ` OR ` + workOwner + `)
 ELSE r.id IS NOT NULL AND (` + subjectVisibility + `) END`
}

// Capture currency follows submitted field replacement, including an omitted
// answer for a requested field. Pending captures cannot retire prior evidence.
// Separate workflow distributions therefore do not use their local is_current
// flags to classify the merged work/assessment response.
func documentCurrentSQL() string {
	return `CASE WHEN req.origin_type IN ('THIRD_PARTY_ASSESSMENT','THIRD_PARTY_WORK') THEN NOT EXISTS (
 SELECT 1 FROM capture_requests newer
 JOIN capture_submissions newer_submission ON newer_submission.request_id=newer.id AND newer_submission.tenant_id=newer.tenant_id
 LEFT JOIN capture_response_revisions newer_revision ON newer_revision.submission_id=newer_submission.id AND newer_revision.tenant_id=newer.tenant_id AND newer_revision.legal_entity_id=newer.legal_entity_id
 LEFT JOIN capture_form_distributions newer_distribution ON newer_distribution.id=newer_revision.distribution_id AND newer_distribution.tenant_id=newer.tenant_id AND newer_distribution.legal_entity_id=newer.legal_entity_id
 WHERE newer.tenant_id=req.tenant_id AND newer.legal_entity_id=req.legal_entity_id
 AND newer.origin_type=req.origin_type AND newer.origin_id=req.origin_id
 AND newer.subject_type=req.subject_type AND newer.subject_id=req.subject_id
 AND newer.form_template_id=req.form_template_id AND newer.form_template_version=req.form_template_version
 AND (newer_revision.id IS NULL OR (newer.distribution_id=newer_revision.distribution_id AND newer.subject_type=newer_distribution.subject_type AND newer.subject_id=newer_distribution.subject_id::text AND newer.form_template_id=newer_distribution.form_template_id AND newer.form_template_version=newer_distribution.form_template_version))
 AND (newer.origin_version,COALESCE(newer_revision.revision,0),newer_submission.submitted_at,newer_submission.id) > (req.origin_version,COALESCE(r.revision,0),submission.submitted_at,submission.id)
 AND EXISTS (SELECT 1 FROM jsonb_array_elements(newer.fields) newer_field WHERE newer_field->>'id'=field->>'id')
 AND CASE req.origin_type
 WHEN 'THIRD_PARTY_WORK' THEN EXISTS (SELECT 1 FROM third_party_work_capture_links link WHERE link.tenant_id=newer.tenant_id AND link.legal_entity_id=newer.legal_entity_id AND link.work_request_id=work.id AND link.request_id=newer.id AND link.origin_type=newer.origin_type AND link.origin_id::text=newer.origin_id AND link.origin_version=newer.origin_version)
 WHEN 'THIRD_PARTY_ASSESSMENT' THEN EXISTS (SELECT 1 FROM third_party_assessment_request_links link WHERE link.tenant_id=newer.tenant_id AND link.legal_entity_id=newer.legal_entity_id AND link.assessment_id=assessment.id AND link.request_id=newer.id AND link.origin_type=newer.origin_type AND link.origin_id::text=newer.origin_id AND link.origin_sequence=newer.origin_version)
 ELSE false END
 ) ELSE COALESCE(r.is_current,false) END`
}

func documentKindSQL(column string) string {
	return `CASE WHEN ` + column + ` !~* '^[[:space:]]*[a-z0-9!#$&^_.+-]+/[a-z0-9!#$&^_.+-]+[[:space:]]*(;[[:space:]]*[a-z0-9!#$&^_.+-]+=("[^"]*"|[a-z0-9!#$&^_.+-]+)[[:space:]]*)*$' THEN 'OTHER' ELSE CASE lower(btrim(split_part(` + column + `,';',1)))
 WHEN 'application/pdf' THEN 'PDF'
 WHEN 'image/png' THEN 'IMAGE' WHEN 'image/jpeg' THEN 'IMAGE' WHEN 'image/gif' THEN 'IMAGE' WHEN 'image/webp' THEN 'IMAGE' WHEN 'image/tiff' THEN 'IMAGE' WHEN 'image/bmp' THEN 'IMAGE'
 WHEN 'application/msword' THEN 'WORD' WHEN 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' THEN 'WORD'
 WHEN 'application/vnd.ms-excel' THEN 'SPREADSHEET' WHEN 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' THEN 'SPREADSHEET' WHEN 'text/csv' THEN 'SPREADSHEET'
 ELSE 'OTHER' END END`
}

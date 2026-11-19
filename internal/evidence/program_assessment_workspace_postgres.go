//go:build postgres

package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const programAssessmentReadScope = ` (t.id::text=$1 OR t.slug=$1) AND er.legal_entity_id=$2::uuid AND er.subject_type='PROGRAM' AND er.subject_id=$3
 AND EXISTS(SELECT 1 FROM programs p WHERE p.id::text=er.subject_id AND p.tenant_id=er.tenant_id AND p.legal_entity_id=er.legal_entity_id
 AND (p.owner_principal_id::text=$4 OR p.authority_principal_id::text=$4 OR (er.recipient_principal_id::text=$4 AND er.recipient_state='ASSIGNED')))`

func (r *PostgresRepository) ProgramAssessmentWorkspace(ctx context.Context, actor identity.Actor, program string) (ProgramAssessmentWorkspace, error) {
	result := ProgramAssessmentWorkspace{Items: []Request{}, Activities: []AssessmentActivity{}}
	args := []any{actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID}
	rows, err := r.pool.Query(ctx, requestSelect+` WHERE `+programAssessmentReadScope+` ORDER BY er.updated_at DESC,er.id DESC LIMIT 101`, args...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		v, e := scanRequest(rows)
		if e != nil {
			rows.Close()
			return result, e
		}
		result.Items = append(result.Items, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(result.Items) > 100 {
		result.Limited = true
		result.Items = result.Items[:100]
	}
	if err = r.populateAssessmentRevisionPointers(ctx, result.Items); err != nil {
		return result, err
	}
	err = r.pool.QueryRow(ctx, `WITH scoped AS (SELECT er.* FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id WHERE `+programAssessmentReadScope+`),
 current_responses AS (
 SELECT req.id,req.tenant_id,sub.answers,review.id AS review_id,review.groups
 FROM scoped req
 JOIN capture_submissions sub ON sub.id=(`+assessmentLatestSubmission+`)
 `+assessmentCurrentReviewJoin+`)
 SELECT count(*) FILTER(WHERE er.form_template_id IS NOT NULL),
 count(*) FILTER(WHERE er.form_template_id IS NOT NULL AND er.status='SUBMITTED' AND response.id IS NOT NULL AND response.review_id IS NULL),
 (SELECT count(*) FROM current_responses current_response
 JOIN LATERAL jsonb_each(current_response.answers) answer ON true
 JOIN LATERAL jsonb_array_elements_text(CASE WHEN jsonb_typeof(answer.value->'artifact_ids')='array' THEN answer.value->'artifact_ids' ELSE '[]'::jsonb END) aid ON true
 JOIN capture_artifacts artifact ON artifact.id::text=aid.value AND artifact.tenant_id=current_response.tenant_id AND artifact.request_id=current_response.id
 WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(current_response.groups) decision
 LEFT JOIN LATERAL (SELECT q.value->>'outcome' AS outcome FROM jsonb_array_elements(decision->'questions') q WHERE q.value->>'field_id'=answer.key) feedback ON true
 WHERE (decision->'artifact_ids') ? aid.value AND COALESCE(feedback.outcome,decision->>'outcome')='REVIEWED'))
 FROM scoped er LEFT JOIN current_responses response ON response.id=er.id`, args...).Scan(&result.AssessmentCount, &result.ResponsesToReview, &result.DocumentsToReview)
	if err != nil {
		return result, err
	}
	events, err := r.pool.Query(ctx, `WITH scoped AS (SELECT er.* FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id WHERE `+programAssessmentReadScope+`), events AS (
 SELECT er.id::text||':created' AS id,er.id::text AS request_id,er.title,'ASSESSMENT_CREATED'::text AS event,er.created_by::text AS actor,er.created_at AS at FROM scoped er WHERE er.origin_type='FRAMEWORK_ASSESSMENT'
 UNION ALL SELECT submission.id::text,er.id::text,er.title,CASE WHEN submission.assessment_revision>1 THEN 'RESPONSE_RESUBMITTED' ELSE 'RESPONSE_SUBMITTED' END,submission.submitted_by::text,submission.submitted_at FROM scoped er JOIN capture_submissions submission ON submission.tenant_id=er.tenant_id AND submission.request_id=er.id
 UNION ALL SELECT review.id::text,er.id::text,er.title,review.status,review.reviewed_by::text,review.created_at FROM scoped er JOIN capture_submission_reviews review ON review.tenant_id=er.tenant_id AND review.request_id=er.id)
 SELECT id,request_id,title,event,actor,COALESCE((SELECT person.display_name FROM principals person JOIN tenants tenant ON tenant.id=person.tenant_id WHERE (tenant.id::text=$1 OR tenant.slug=$1) AND person.id::text=events.actor),''),to_char(at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM events ORDER BY at DESC,id DESC LIMIT 301`, args...)
	if err != nil {
		return result, err
	}
	defer events.Close()
	for events.Next() {
		var v AssessmentActivity
		if err = events.Scan(&v.ID, &v.RequestID, &v.Title, &v.Event, &v.Actor, &v.ActorName, &v.At); err != nil {
			return result, err
		}
		result.Activities = append(result.Activities, v)
	}
	if len(result.Activities) > 300 {
		result.ActivitiesLimited = true
		result.Activities = result.Activities[:300]
	}
	if err := events.Err(); err != nil {
		return result, err
	}
	events.Close()
	result.RequirementReviews, err = r.requirementReviewCoverage(ctx, args)
	return result, err
}

// requirementReviewCoverage uses the approved requirement's pinned question group
// and the latest reviewed collection period. It is not a compliance determination.
func (r *PostgresRepository) requirementReviewCoverage(ctx context.Context, args []any) ([]RequirementReviewCoverage, error) {
	rows, err := r.pool.Query(ctx, `
 SELECT requirement.id::text, requirement.title, COALESCE(section.value->>'title','Unmapped requirements'), COALESCE(decision.outcome,'UNASSESSED')
 FROM programs p JOIN tenants t ON t.id=p.tenant_id
 JOIN program_requirements requirement ON requirement.program_id=p.id AND requirement.tenant_id=p.tenant_id
 LEFT JOIN program_framework_requirement_questions link ON link.requirement_id=requirement.id AND link.program_id=p.id AND link.tenant_id=p.tenant_id AND link.legal_entity_id=p.legal_entity_id
 LEFT JOIN program_framework_adoptions adoption ON adoption.id=link.adoption_id AND adoption.status='ACTIVE' AND adoption.effective_from<=now() AND (adoption.effective_until IS NULL OR adoption.effective_until>now())
 LEFT JOIN monitoring_form_templates form ON form.id=link.form_template_id AND form.version=link.form_template_version AND form.tenant_id=p.tenant_id AND form.legal_entity_id=p.legal_entity_id
 LEFT JOIN LATERAL (SELECT value FROM jsonb_array_elements(form.sections) WHERE value->>'id'=link.group_id LIMIT 1) section ON true
 LEFT JOIN LATERAL (
 SELECT g.value->>'outcome' AS outcome
 FROM capture_requests req JOIN capture_submission_reviews review ON review.tenant_id=req.tenant_id AND review.legal_entity_id=req.legal_entity_id AND review.request_id=req.id
 CROSS JOIN LATERAL jsonb_array_elements(review.groups) g
 WHERE req.tenant_id=p.tenant_id AND req.legal_entity_id=p.legal_entity_id AND req.subject_type='PROGRAM'
 AND req.subject_id=p.id::text AND req.origin_type='FRAMEWORK_ASSESSMENT' AND req.known_facts->>'adoption_id'=adoption.id::text
 AND req.form_template_id=link.form_template_id AND req.form_template_version=link.form_template_version
 AND g.value->>'group_id'=link.group_id AND review.submission_id=(`+assessmentLatestSubmission+`)
 ORDER BY req.collection_period_end DESC NULLS LAST,review.created_at DESC,review.version DESC,review.id DESC LIMIT 1
 ) decision ON true
 WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid AND p.id::text=$3
 AND $4 IN (p.owner_principal_id::text,p.authority_principal_id::text)
 AND requirement.status='APPROVED' AND requirement.effective_from<=now() AND (requirement.effective_until IS NULL OR requirement.effective_until>now())
 ORDER BY 3,2,1`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RequirementReviewCoverage
	for rows.Next() {
		var item RequirementReviewCoverage
		if err = rows.Scan(&item.ID, &item.Title, &item.Area, &item.Outcome); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) populateAssessmentRevisionPointers(ctx context.Context, requests []Request) error {
	indices := make(map[string]int, len(requests))
	var ids []string
	for i, request := range requests {
		if request.Status == RequestSubmitted {
			ids = append(ids, request.ID)
			indices[request.ID] = i
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.pool.Query(ctx, `SELECT req.id::text,sub.id::text,sub.assessment_revision
 FROM capture_requests req JOIN capture_submissions sub ON sub.id=(`+assessmentLatestSubmission+`)
 WHERE req.id::text=ANY($1::text[])`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var requestID, submissionID string
		var revision int64
		if err := rows.Scan(&requestID, &submissionID, &revision); err != nil {
			return err
		}
		request := &requests[indices[requestID]]
		request.LatestAssessmentSubmissionID, request.LatestAssessmentRevision = submissionID, revision
	}
	return rows.Err()
}

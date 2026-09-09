//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// Every population, including counts, is limited by current workflow permission
// before grouping or pagination. Pending requests use the same exact links as
// submitted responses, without requiring a submission to exist.
// Workflow currency uses the existing submitted field replacement predicate:
// only retirement of every field retires the response; partial concerns remain
// relevant with explicit PARTIALLY_REPLACED freshness.
func vendorFormsScopedSQL(allowUnscanned bool) string {
	access := documentReadAuthoritySQLAt(3, 4)
	access = strings.ReplaceAll(access, "ELSE r.id IS NOT NULL AND (", "ELSE (")
	access = "(" + access + ") OR (" + completedResponseReviewerSQL(3, 4) + ")"
	access = regexp.MustCompile(`\br\.(tenant_id|legal_entity_id)\b`).ReplaceAllString(access, "req.$1")
	return `WITH scoped AS MATERIALIZED (
 SELECT req.*,d.status AS distribution_status,recipient.state AS delivery_state,recipient.audience_hint,
 r.id AS response_id,r.created_at AS submitted_at,r.score_result AS automatic_score,
 CASE WHEN bank.response_revision_id IS NOT NULL THEN bank.score_result WHEN COALESCE((r.score_result->>'assessment_review_count')::int,0)>0 THEN NULL ELSE r.score_result END AS effective_score,bank.score_result AS assessed_score,
 COALESCE(bank.state,CASE WHEN COALESCE((r.score_result->>'assessment_review_count')::int,0)>0 THEN 'AWAITING_REVIEW' ELSE 'NOT_REQUIRED' END) AS assessment_state,
 COALESCE(bank.required_count,(r.score_result->>'assessment_required_count')::int,0) AS required_reviews,
 COALESCE(bank.reviewed_required_count,0) AS completed_reviews,
	CASE WHEN d.status IN ('REVOKED','SUPERSEDED') THEN d.status WHEN r.id IS NOT NULL THEN 'SUBMITTED' WHEN req.status='SUBMITTED' THEN 'SUBMITTED' WHEN req.status='CANCELLED' THEN 'CANCELLED' WHEN ` + collectionNoVendorActionSQL("req", "$4", allowUnscanned) + ` THEN 'NO_VENDOR_ACTION' WHEN req.status='IN_PROGRESS' THEN 'IN_PROGRESS' WHEN req.status='DRAFT' THEN 'REQUEST_READY' WHEN req.status='EXPIRED' THEN req.status ELSE 'AWAITING_RESPONSE' END AS response_state,
 COALESCE(d.status NOT IN ('REVOKED','SUPERSEDED'),true) AND (req.origin_type NOT IN ('THIRD_PARTY_ASSESSMENT','THIRD_PARTY_WORK') OR submission.id IS NULL OR currency.total=0 OR currency.remaining>0) AS current,
 CASE WHEN d.status IN ('REVOKED','SUPERSEDED') THEN 'HISTORICAL'
 WHEN req.origin_type NOT IN ('THIRD_PARTY_ASSESSMENT','THIRD_PARTY_WORK') OR submission.id IS NULL OR currency.total=0 OR currency.remaining=currency.total THEN 'CURRENT'
 WHEN currency.remaining=0 THEN 'HISTORICAL' ELSE 'PARTIALLY_REPLACED' END AS response_currency
 FROM capture_requests req JOIN tenants t ON t.id=req.tenant_id
 LEFT JOIN capture_form_distributions d ON d.id=req.distribution_id AND d.tenant_id=req.tenant_id AND d.legal_entity_id=req.legal_entity_id
 LEFT JOIN LATERAL (SELECT sub.* FROM capture_submissions sub WHERE sub.tenant_id=req.tenant_id AND sub.request_id=req.id ORDER BY sub.submitted_at DESC,sub.id DESC LIMIT 1) submission ON true
 LEFT JOIN capture_response_revisions r ON r.submission_id=submission.id AND r.tenant_id=req.tenant_id AND r.legal_entity_id=req.legal_entity_id AND r.is_current
 ` + documentWorkflowJoinsSQL() + `
 LEFT JOIN LATERAL (SELECT count(*) AS total,count(*) FILTER(WHERE ` + documentCurrentSQL() + `) AS remaining FROM jsonb_array_elements(req.fields) field) currency ON true
 LEFT JOIN LATERAL (SELECT a.* FROM capture_response_assessments a WHERE a.tenant_id=req.tenant_id AND a.legal_entity_id=req.legal_entity_id AND a.response_revision_id=r.id ORDER BY a.version DESC LIMIT 1) bank ON true
 LEFT JOIN capture_distribution_recipients recipient ON recipient.request_id=req.id AND recipient.tenant_id=req.tenant_id AND recipient.legal_entity_id=req.legal_entity_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND req.legal_entity_id=$2::uuid AND req.subject_type='VENDOR_RELATIONSHIP' AND req.subject_id=ANY($5::text[]) AND req.form_template_id IS NOT NULL
 AND ($6='' OR req.form_template_id::text=$6) AND ` + documentRevisionScopeSQL() + ` AND (` + access + `)
 ), filtered AS (
 SELECT * FROM scoped WHERE CASE $7
 WHEN 'AWAITING_VENDOR' THEN response_state IN ('REQUEST_READY','AWAITING_RESPONSE','IN_PROGRESS','EXPIRED')
 WHEN 'AWAITING_REVIEW' THEN assessment_state IN ('AWAITING_REVIEW','IN_REVIEW')
 WHEN 'WITH_RISKS' THEN current AND response_state='SUBMITTED' AND effective_score->>'state'='FINAL' AND effective_score->>'band' IN ('MODERATE','HIGH','CRITICAL')
 WHEN 'HIGH_RISK' THEN current AND response_state='SUBMITTED' AND effective_score->>'state'='FINAL' AND effective_score->>'band' IN ('HIGH','CRITICAL')
 WHEN 'OVERDUE' THEN response_state IN ('REQUEST_READY','AWAITING_RESPONSE','IN_PROGRESS','EXPIRED') AND deadline<$4
 WHEN 'NOT_ASSESSED' THEN COALESCE(effective_score->>'state','')<>'FINAL'
 ELSE true END
 ) `
}

func (s *PostgresDistributionStore) ListVendorForms(ctx context.Context, q VendorFormsQuery) (VendorFormsPage, error) {
	if err := normalizeVendorFormsQuery(&q); err != nil {
		return VendorFormsPage{}, err
	}
	now := time.Now().UTC()
	cursor := decodeVendorFormsCursor(q.Cursor)
	rows, err := s.repo.pool.Query(ctx, vendorFormsScopedSQL(s.repo.demoUnscannedAllowed)+`
 SELECT to_jsonb(f)||jsonb_build_object('origin',jsonb_build_object('type',f.origin_type,'id',f.origin_id,'version',f.origin_version)),
 jsonb_build_object('request_id',f.id::text,'relationship_id',f.subject_id,'distribution_id',f.distribution_id::text,'response_id',f.response_id::text,'form_template_id',f.form_template_id::text,'form_template_version',f.form_template_version,'title',f.title,'purpose',f.purpose,'origin_type',f.origin_type,'origin_id',f.origin_id,'response_state',f.response_state,'recipient_hint',f.audience_hint,'delivery_state',f.delivery_state,'deadline',f.deadline,'updated_at',f.updated_at,'submitted_at',f.submitted_at,'score',f.automatic_score,'assessed_score',f.assessed_score,'assessment_state',f.assessment_state,'required_reviews',f.required_reviews,'completed_reviews',f.completed_reviews,'current',f.current,'response_currency',f.response_currency),
 COALESCE(submission.answers,edits.answers,draft.answers,'{}'::jsonb),
 submission.id IS NOT NULL OR edits.answers IS NOT NULL OR draft.id IS NOT NULL OR f.status IN ('READY','DRAFT')
 FROM (SELECT * FROM filtered WHERE ($8='' OR (updated_at,id::text)<($9,$8)) ORDER BY updated_at DESC,id DESC LIMIT $10) f
 LEFT JOIN LATERAL (SELECT sub.id,sub.answers FROM capture_submissions sub WHERE sub.tenant_id=f.tenant_id AND sub.request_id=f.id ORDER BY sub.submitted_at DESC,sub.id DESC LIMIT 1) submission ON true
 LEFT JOIN LATERAL (SELECT dr.id,dr.answers FROM capture_response_drafts dr WHERE dr.tenant_id=f.tenant_id AND dr.request_id=f.id ORDER BY dr.updated_at DESC,dr.id DESC LIMIT 1) draft ON true
 LEFT JOIN LATERAL (SELECT jsonb_object_agg(e.field_id,e.value) AS answers FROM (
 SELECT DISTINCT ON (patch->>'field_id') patch->>'field_id' AS field_id,patch->'value' AS value FROM capture_response_workspace_edits WHERE tenant_id=f.tenant_id AND legal_entity_id=f.legal_entity_id AND distribution_id=f.distribution_id ORDER BY patch->>'field_id',result_version DESC,id DESC
 ) e) edits ON true ORDER BY f.updated_at DESC,f.id DESC`, q.TenantID, q.LegalEntityID, q.PrincipalID, now, q.RelationshipIDs, q.FormTemplateID, q.Filter, cursor.ID, cursor.UpdatedAt, q.Limit+1)
	if err != nil {
		return VendorFormsPage{}, vendorFormsError(err)
	}
	defer rows.Close()
	values := []VendorFormRow{}
	type progressInput struct {
		request Request
		row     VendorFormRow
		answers map[string]formcontract.AnswerValue
		known   bool
	}
	inputs := make([]progressInput, 0, q.Limit+1)
	for rows.Next() {
		var reqJSON, rowJSON, answerJSON []byte
		var known bool
		if err := rows.Scan(&reqJSON, &rowJSON, &answerJSON, &known); err != nil {
			return VendorFormsPage{}, err
		}
		var req Request
		var row VendorFormRow
		var answers map[string]formcontract.AnswerValue
		if err := json.Unmarshal(reqJSON, &req); err != nil {
			return VendorFormsPage{}, err
		}
		if err := json.Unmarshal(rowJSON, &row); err != nil {
			return VendorFormsPage{}, err
		}
		if err := json.Unmarshal(answerJSON, &answers); err != nil {
			return VendorFormsPage{}, err
		}
		inputs = append(inputs, progressInput{req, row, answers, known})
	}
	if err := rows.Err(); err != nil {
		return VendorFormsPage{}, err
	}
	// Release the bounded page cursor before exact artifact reads so a
	// one-connection pool cannot deadlock while refreshing collection status.
	rows.Close()
	for _, input := range inputs {
		req, row, answers, known := input.request, input.row, input.answers, input.known
		req = RefreshCollectionResolutions(ctx, req, s.repo.GetArtifact, now)
		req, err = s.repo.RefreshCollectionRequestReviews(ctx, req)
		if err != nil {
			return VendorFormsPage{}, err
		}
		progress := vendorFormRow(req, answers, known, nil, now)
		row.RequiredCount = progress.RequiredCount
		row.AnsweredRequired = progress.AnsweredRequired
		row.HeldRequired = progress.HeldRequired
		row.MissingFields = progress.MissingFields
		if row.ResponseState == "IN_PROGRESS" || row.ResponseState == "AWAITING_RESPONSE" || row.ResponseState == "NO_VENDOR_ACTION" || row.ResponseState == "REQUEST_READY" || row.ResponseState == "EXPIRED" {
			row.ResponseState = progress.ResponseState
		}
		values = append(values, row)
	}
	return vendorFormsPage(values, q.Limit, now), nil
}
func (s *PostgresDistributionStore) VendorFormSummaries(ctx context.Context, q VendorFormsQuery) ([]VendorFormSummary, error) {
	if err := normalizeVendorFormsQuery(&q); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rows, err := s.repo.pool.Query(ctx, vendorFormsScopedSQL(s.repo.demoUnscannedAllowed)+` SELECT subject_id,
 count(*) FILTER(WHERE response_state IN ('REQUEST_READY','AWAITING_RESPONSE','IN_PROGRESS','EXPIRED')),
 count(*) FILTER(WHERE response_state IN ('REQUEST_READY','AWAITING_RESPONSE','IN_PROGRESS','EXPIRED') AND deadline<$4),
 count(*) FILTER(WHERE response_state='SUBMITTED'),
 count(*) FILTER(WHERE response_currency='PARTIALLY_REPLACED'),
 count(*) FILTER(WHERE response_state='SUBMITTED' AND assessment_state IN ('AWAITING_REVIEW','IN_REVIEW')),
 count(*) FILTER(WHERE response_state='SUBMITTED' AND COALESCE(effective_score->>'state','')<>'FINAL'),
 count(*) FILTER(WHERE response_state='SUBMITTED' AND effective_score->>'state'='FINAL'),
 COALESCE(max(CASE WHEN effective_score->>'state'='FINAL' THEN CASE effective_score->>'band' WHEN 'CRITICAL' THEN 4 WHEN 'HIGH' THEN 3 WHEN 'MODERATE' THEN 2 WHEN 'LOW' THEN 1 ELSE 0 END ELSE 0 END),0)
 FROM filtered WHERE current GROUP BY subject_id`, q.TenantID, q.LegalEntityID, q.PrincipalID, now, q.RelationshipIDs, q.FormTemplateID, q.Filter)
	if err != nil {
		return nil, vendorFormsError(err)
	}
	defer rows.Close()
	byID := map[string]VendorFormSummary{}
	for rows.Next() {
		var v VendorFormSummary
		var band int
		if err := rows.Scan(&v.RelationshipID, &v.OutstandingForms, &v.OverdueForms, &v.SubmittedForms, &v.PartiallyReplacedForms, &v.AwaitingReview, &v.UnassessedForms, &v.AssessedForms, &band); err != nil {
			return nil, err
		}
		v.ObservedAt = now
		v.HighestConcern = []formcontract.ConcernBand{"", formcontract.ConcernLow, formcontract.ConcernModerate, formcontract.ConcernHigh, formcontract.ConcernCritical}[band]
		byID[v.RelationshipID] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	values := []VendorFormSummary{}
	for _, id := range q.RelationshipIDs {
		v, ok := byID[id]
		if !ok {
			v = VendorFormSummary{RelationshipID: id, ObservedAt: now}
		}
		values = append(values, v)
	}
	return values, nil
}

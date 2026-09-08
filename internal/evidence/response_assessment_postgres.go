//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/jackc/pgx/v5"
	"time"
)

type assessmentQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgreSQL execution checks the assessment while locking the response in the
// policy repository's transaction; acquiring another transaction here would
// deadlock that repository on its own row lock.
func (s *PostgresDistributionStore) WithAssessedResponseForExecution(ctx context.Context, tenant, response string, version int64, apply func() error) error {
	return apply()
}

func readPostgresAssessmentSnapshot(ctx context.Context, q assessmentQueryer, tenant, response string) (assessmentSnapshot, error) {
	var v assessmentSnapshot
	var decisions, score []byte
	err := q.QueryRow(ctx, `SELECT version,state,required_count,reviewed_required_count,reviewed_count,score_result,decisions FROM capture_response_assessments WHERE tenant_id=$1::uuid AND response_revision_id=$2::uuid ORDER BY version DESC LIMIT 1`, tenant, response).Scan(&v.Version, &v.Result.State, &v.Result.RequiredCount, &v.Result.ReviewedRequiredCount, &v.Result.ReviewedCount, &score, &decisions)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	v.Result.Version = v.Version
	if err = json.Unmarshal(score, &v.Result.Score); err != nil {
		return v, err
	}
	err = json.Unmarshal(decisions, &v.Decisions)
	return v, err
}
func (s *PostgresDistributionStore) ReadResponseAssessment(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer) (assessmentMaterial, error) {
	return readResponseAssessmentMaterial(ctx, s.repo.pool, tenant, entity, principal, response, authorize)
}
func readResponseAssessmentMaterial(ctx context.Context, q assessmentQueryer, tenant, entity, principal, response string, authorize assessmentReadAuthorizer) (assessmentMaterial, error) {
	var formID, title, subjectType, subjectID string
	var formVersion int64
	var ordinaryRead, distributionCurrent bool
	revision, err := scanPostgresResponseRevisionWithExtra(q.QueryRow(ctx, `SELECT `+responseRevisionProjection+`,d.form_template_id::text,d.form_template_version,d.title,d.subject_type,d.subject_id::text,(`+completedResponseVisibilitySQL(4, 5)+`),d.status NOT IN ('REVOKED','SUPERSEDED') FROM capture_response_revisions r JOIN tenants t ON t.id=r.tenant_id JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id `+completedResponseRequestJoinsSQL()+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid AND ((`+completedResponseVisibilitySQL(4, 5)+`) OR (`+vendorAssessmentCandidateSQL()+`))`, tenant, entity, response, principal, time.Now().UTC()), &formID, &formVersion, &title, &subjectType, &subjectID, &ordinaryRead, &distributionCurrent)
	if errors.Is(err, pgx.ErrNoRows) {
		return assessmentMaterial{}, ErrNotFound
	}
	if err != nil {
		return assessmentMaterial{}, err
	}
	if revision.State != ResponseRevisionFinal && revision.State != ResponseRevisionProvisional {
		return assessmentMaterial{}, ErrNotFound
	}
	revision.Current = revision.Current && distributionCurrent
	summary := CompletedResponseSummary{ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID, DistributionID: revision.DistributionID, FormTemplateID: formID, FormTemplateVersion: formVersion, Title: title, SubjectType: subjectType, SubjectID: subjectID, Revision: revision.Revision, Current: revision.Current, State: revision.State, Score: revision.Score, CompletedAt: revision.CreatedAt.UTC()}
	var submission Submission
	var answers, provenance []byte
	err = q.QueryRow(ctx, `SELECT id::text,tenant_id::text,request_id::text,COALESCE(session_id::text,''),COALESCE(submitted_by::text,''),channel,answers,answer_provenance,submitted_at FROM capture_submissions WHERE tenant_id=$1::uuid AND id=$2::uuid`, summary.TenantID, revision.SubmissionID).Scan(&submission.ID, &submission.TenantID, &submission.RequestID, &submission.SessionID, &submission.SubmittedBy, &submission.Channel, &answers, &provenance, &submission.SubmittedAt)
	if err != nil {
		return assessmentMaterial{}, err
	}
	if err = json.Unmarshal(answers, &submission.Answers); err != nil {
		return assessmentMaterial{}, err
	}
	if err = json.Unmarshal(provenance, &submission.AnswerProvenance); err != nil {
		return assessmentMaterial{}, err
	}
	request, err := scanRequest(q.QueryRow(ctx, requestSelect+` WHERE er.id=$1::uuid AND er.tenant_id=$2::uuid`, submission.RequestID, summary.TenantID))
	if err != nil {
		return assessmentMaterial{}, err
	}
	if request.LegalEntityID != summary.LegalEntityID || request.FormTemplateID != summary.FormTemplateID || request.FormTemplateVersion != summary.FormTemplateVersion || request.SubjectType != summary.SubjectType || request.SubjectID != summary.SubjectID {
		return assessmentMaterial{}, ErrNotFound
	}
	m := assessmentMaterial{Summary: summary, Revision: revision, Request: request, Submission: submission}
	if !ordinaryRead && (authorize == nil || !authorize(ctx, m)) {
		return assessmentMaterial{}, ErrNotFound
	}
	m.Snapshot, err = readPostgresAssessmentSnapshot(ctx, q, summary.TenantID, response)
	return m, err
}
func (s *PostgresDistributionStore) WriteResponseAssessment(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer, apply func(context.Context, assessmentMaterial) (assessmentSnapshot, []FieldAssessmentDecision, error)) (assessmentMaterial, error) {
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return assessmentMaterial{}, err
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	var current bool
	err = tx.QueryRow(ctx, `SELECT r.is_current AND d.status NOT IN ('REVOKED','SUPERSEDED') FROM capture_response_revisions r JOIN tenants t ON t.id=r.tenant_id JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id `+completedResponseRequestJoinsSQL()+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid AND ((`+completedResponseVisibilitySQL(4, 5)+`) OR (`+vendorAssessmentCandidateSQL()+`)) FOR UPDATE OF r,d`, tenant, entity, response, principal, time.Now().UTC()).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return assessmentMaterial{}, ErrNotFound
	}
	if err != nil {
		return assessmentMaterial{}, err
	}
	if !current {
		return assessmentMaterial{}, ErrAssessmentConflict
	}
	m, err := readResponseAssessmentMaterial(ctx, tx, tenant, entity, principal, response, authorize)
	if err != nil {
		return assessmentMaterial{}, err
	}
	next, decisions, err := apply(ctx, m)
	if err != nil {
		return assessmentMaterial{}, err
	}
	m.Snapshot = next
	score, err := json.Marshal(next.Result.Score)
	if err != nil {
		return assessmentMaterial{}, err
	}
	decisionJSON, err := json.Marshal(next.Decisions)
	if err != nil {
		return assessmentMaterial{}, err
	}
	now := s.now().UTC()
	_, err = tx.Exec(ctx, `INSERT INTO capture_response_assessments(tenant_id,legal_entity_id,response_revision_id,version,state,required_count,reviewed_required_count,reviewed_count,score_result,decisions,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11)`, m.Summary.TenantID, m.Summary.LegalEntityID, response, next.Version, next.Result.State, next.Result.RequiredCount, next.Result.ReviewedRequiredCount, next.Result.ReviewedCount, string(score), string(decisionJSON), now)
	if err != nil {
		return assessmentMaterial{}, err
	}
	for _, d := range decisions {
		payload, err := json.Marshal(d)
		if err != nil {
			return assessmentMaterial{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO capture_field_assessments(id,tenant_id,legal_entity_id,response_revision_id,assessment_version,form_template_id,form_template_version,field_id,field_checksum,reviewer_id,authority_route,supersedes_id,decision,assessed_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8,$9,$10::uuid,$11,NULLIF($12,'')::uuid,$13::jsonb,$14)`, d.ID, m.Summary.TenantID, m.Summary.LegalEntityID, response, next.Version, d.FormTemplateID, d.FormTemplateVersion, d.FieldID, d.FieldChecksum, d.ReviewerID, d.AuthorityRoute, d.SupersedesID, string(payload), d.AssessedAt)
		if err != nil {
			return assessmentMaterial{}, err
		}
	}
	event := assessmentEvent(m, principal, now)
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return assessmentMaterial{}, err
	}
	// Audit event keys include response and assessment versions while the outbox
	// uses one stable event type for inbox deduplication and policy dispatch.
	eventType := fmt.Sprintf("FORM_RESPONSE_ASSESSED_%s_%d", response, next.Version)
	_, err = tx.Exec(ctx, `INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at) SELECT $1::uuid,$2::uuid,d.id,d.version,$4,$5::jsonb,$6::uuid,$7 FROM capture_form_distributions d WHERE d.id=$3::uuid AND d.tenant_id=$1::uuid`, m.Summary.TenantID, m.Summary.LegalEntityID, m.Revision.DistributionID, eventType, string(payload), principal, now)
	if err != nil {
		return assessmentMaterial{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_RESPONSE_ASSESSMENT',$2::uuid,'FORM_RESPONSE_ASSESSED',$3::jsonb,$4,$4,$4)`, m.Summary.TenantID, m.Revision.ID, string(payload), now)
	if err != nil {
		return assessmentMaterial{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return assessmentMaterial{}, err
	}
	return m, nil
}
func (s *PostgresDistributionStore) ReadAssessmentSummary(ctx context.Context, tenant, response string) (*ResponseAssessmentSummary, error) {
	v, err := readPostgresAssessmentSnapshot(ctx, s.repo.pool, tenant, response)
	if err != nil {
		return nil, err
	}
	if v.Version == 0 {
		return nil, nil
	}
	return &v.Result, nil
}

func (s *PostgresDistributionStore) ReadAssessmentSummaries(ctx context.Context, tenant string, responses []string) (map[string]*ResponseAssessmentSummary, error) {
	if len(responses) > 100 {
		return nil, ErrAssessmentInvalid
	}
	result := map[string]*ResponseAssessmentSummary{}
	if len(responses) == 0 {
		return result, nil
	}
	rows, err := s.repo.pool.Query(ctx, `SELECT DISTINCT ON(a.response_revision_id) a.response_revision_id::text,a.version,a.state,a.required_count,a.reviewed_required_count,a.reviewed_count,a.score_result FROM capture_response_assessments a JOIN tenants t ON t.id=a.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND a.response_revision_id=ANY($2::uuid[]) ORDER BY a.response_revision_id,a.version DESC`, tenant, responses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var response string
		var v ResponseAssessmentSummary
		var score []byte
		if err := rows.Scan(&response, &v.Version, &v.State, &v.RequiredCount, &v.ReviewedRequiredCount, &v.ReviewedCount, &score); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(score, &v.Score); err != nil {
			return nil, err
		}
		result[response] = &v
	}
	return result, rows.Err()
}

func vendorAssessmentCandidateSQL() string {
	return `submission.id IS NOT NULL AND req.id IS NOT NULL AND d.subject_type='VENDOR_RELATIONSHIP' AND COALESCE(req.origin_type,'') NOT IN ('THIRD_PARTY_WORK','THIRD_PARTY_ASSESSMENT') AND EXISTS (SELECT 1 FROM third_party_relationships vr WHERE vr.tenant_id=r.tenant_id AND vr.legal_entity_id=r.legal_entity_id AND vr.id=d.subject_id)`
}

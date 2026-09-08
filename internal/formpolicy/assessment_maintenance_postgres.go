//go:build postgres

package formpolicy

import (
	"context"
	"time"
)

func (repo *PostgresRepository) seedAssessedReconciliation(ctx context.Context, now time.Time, limit int) (int, error) {
	tag, err := repo.pool.Exec(ctx, `INSERT INTO form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,job_type,response_revision_id,due_at,state,created_at,updated_at,result_basis,assessment_version,result_occurred_at)
 SELECT candidate.tenant_id,candidate.legal_entity_id,'RECONCILE',candidate.response_revision_id,$1,'READY',$1,$1,'BANK_ASSESSED',candidate.version,candidate.created_at
 FROM (
 SELECT DISTINCT a.tenant_id,a.legal_entity_id,a.response_revision_id,a.version,a.created_at
 FROM capture_response_assessments a
 JOIN capture_response_revisions r ON r.id=a.response_revision_id AND r.tenant_id=a.tenant_id AND r.legal_entity_id=a.legal_entity_id AND r.is_current
 JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
 JOIN form_response_policy_definitions p ON p.tenant_id=a.tenant_id AND p.legal_entity_id=a.legal_entity_id AND p.form_template_id=d.form_template_id AND p.form_template_version=d.form_template_version
 WHERE a.state='ASSESSED' AND a.score_result->>'state'='FINAL' AND a.reviewed_required_count=a.required_count
 AND NOT EXISTS(SELECT 1 FROM capture_response_assessments newer WHERE newer.tenant_id=a.tenant_id AND newer.response_revision_id=a.response_revision_id AND newer.version>a.version)
 AND p.status='ACTIVE' AND p.eligibility->>'result_basis'='BANK_ASSESSED' AND p.activated_at<=a.created_at
 AND (p.effective_from IS NULL OR p.effective_from<=a.created_at) AND (p.effective_until IS NULL OR p.effective_until>a.created_at)
 AND NOT EXISTS(SELECT 1 FROM form_response_policy_executions e WHERE e.tenant_id=a.tenant_id AND e.legal_entity_id=a.legal_entity_id AND e.policy_id=p.id AND e.policy_version=p.version AND e.response_revision_id=a.response_revision_id AND e.assessment_version=a.version)
 AND NOT EXISTS(SELECT 1 FROM form_response_policy_maintenance_jobs job WHERE job.tenant_id=a.tenant_id AND job.legal_entity_id=a.legal_entity_id AND job.response_revision_id=a.response_revision_id AND job.assessment_version=a.version AND job.job_type='RECONCILE')
 ORDER BY a.created_at,a.response_revision_id LIMIT $2
 ) candidate ON CONFLICT(tenant_id,legal_entity_id,response_revision_id,assessment_version) WHERE job_type='RECONCILE' DO NOTHING`, now.UTC(), limit)
	if err != nil {
		return 0, normalizePostgresError(err)
	}
	return int(tag.RowsAffected()), nil
}

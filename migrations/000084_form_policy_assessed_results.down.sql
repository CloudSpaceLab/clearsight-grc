BEGIN;
DROP INDEX form_policy_response_issue_membership_idx;
DROP INDEX form_policy_automation_uses_idx;
DROP INDEX form_policy_failure_history_idx;
DROP INDEX form_policy_execution_history_idx;
DROP INDEX form_response_policy_reconcile_job_uq;
CREATE UNIQUE INDEX form_response_policy_reconcile_job_uq ON form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,response_revision_id) WHERE job_type='RECONCILE';
ALTER TABLE form_response_policy_maintenance_jobs DROP COLUMN result_basis,DROP COLUMN assessment_version,DROP COLUMN result_occurred_at;
ALTER TABLE form_response_policy_execution_failures DROP COLUMN result_basis,DROP COLUMN assessment_version;
DROP INDEX automation_form_policy_choices_idx;
DROP INDEX form_policy_same_response_episode_idx;
ALTER TABLE form_response_policy_executions DROP CONSTRAINT form_policy_result_revision_unique;
-- Downgrade fails transactionally if assessment history requires the expanded identity.
-- No material execution history is deleted to force a downgrade.
ALTER TABLE form_response_policy_executions ADD UNIQUE(tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id);
ALTER TABLE form_response_policy_executions DROP CONSTRAINT form_policy_assessed_receipt_basis_check,
 DROP COLUMN assessment_version,DROP COLUMN result_basis;
COMMIT;

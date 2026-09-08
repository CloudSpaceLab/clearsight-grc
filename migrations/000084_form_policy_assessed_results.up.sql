BEGIN;
ALTER TABLE form_response_policy_executions
 ADD COLUMN result_basis text NOT NULL DEFAULT 'AUTOMATIC' CHECK(result_basis IN ('AUTOMATIC','BANK_ASSESSED')),
 ADD COLUMN assessment_version bigint NOT NULL DEFAULT 0 CHECK(assessment_version>=0),
 ADD CONSTRAINT form_policy_assessed_receipt_basis_check CHECK((result_basis='AUTOMATIC' AND assessment_version=0) OR (result_basis='BANK_ASSESSED' AND assessment_version>0));
DO $$ DECLARE existing_name text; BEGIN
 SELECT c.conname INTO STRICT existing_name FROM pg_constraint c
 WHERE c.conrelid='form_response_policy_executions'::regclass AND c.contype='u' AND cardinality(c.conkey)=5;
 EXECUTE format('ALTER TABLE form_response_policy_executions DROP CONSTRAINT %I',existing_name);
END $$;
ALTER TABLE form_response_policy_executions ADD CONSTRAINT form_policy_result_revision_unique
 UNIQUE(tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id,assessment_version);
CREATE INDEX form_policy_same_response_episode_idx ON form_response_policy_adverse_episodes(tenant_id,legal_entity_id,subject_type,subject_id,last_response_revision_id) WHERE state='OPEN';
CREATE INDEX automation_form_policy_choices_idx ON automation_policies(tenant_id,(eligibility->>'form_template_id'),(eligibility->>'form_template_version'),code,version DESC) WHERE action_class='FORM_RESPONSE_CREATE_MATTER';
ALTER TABLE form_response_policy_execution_failures ADD COLUMN result_basis text NOT NULL DEFAULT 'AUTOMATIC' CHECK(result_basis IN ('AUTOMATIC','BANK_ASSESSED')),ADD COLUMN assessment_version bigint NOT NULL DEFAULT 0 CHECK(assessment_version>=0);
ALTER TABLE form_response_policy_maintenance_jobs ADD COLUMN result_basis text NOT NULL DEFAULT 'AUTOMATIC' CHECK(result_basis IN ('AUTOMATIC','BANK_ASSESSED')),ADD COLUMN assessment_version bigint NOT NULL DEFAULT 0 CHECK(assessment_version>=0),ADD COLUMN result_occurred_at timestamptz;
DROP INDEX form_response_policy_reconcile_job_uq;
CREATE UNIQUE INDEX form_response_policy_reconcile_job_uq ON form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,response_revision_id,assessment_version) WHERE job_type='RECONCILE';
CREATE INDEX form_policy_execution_history_idx ON form_response_policy_executions(tenant_id,legal_entity_id,policy_id,created_at DESC,id DESC);
CREATE INDEX form_policy_failure_history_idx ON form_response_policy_execution_failures(tenant_id,legal_entity_id,policy_id,created_at DESC,id DESC);
CREATE INDEX form_policy_automation_uses_idx ON form_response_policy_definitions(tenant_id,legal_entity_id,automation_policy_id,automation_policy_version);
CREATE INDEX form_policy_response_issue_membership_idx ON form_response_policy_executions(tenant_id,legal_entity_id,matter_id,response_revision_id,result_basis) WHERE state IN ('APPLIED','REUSED');
COMMIT;

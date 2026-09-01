BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM form_response_policy_maintenance_jobs)
       OR EXISTS (SELECT 1 FROM form_response_policy_execution_failures)
       OR EXISTS (SELECT 1 FROM form_response_policy_compensations) THEN
        RAISE EXCEPTION 'cannot roll back form-response policy maintenance while recovery history exists';
    END IF;
END;
$$;

DROP TABLE IF EXISTS form_response_policy_compensations;
DROP TABLE IF EXISTS form_response_policy_execution_failures;
DROP TABLE IF EXISTS form_response_policy_maintenance_jobs;
ALTER TABLE form_response_policy_executions
    DROP CONSTRAINT IF EXISTS form_response_policy_executions_id_scope_key;
DROP INDEX IF EXISTS form_response_policy_effective_form_idx;

COMMIT;

ALTER TABLE verification_results
    DROP CONSTRAINT IF EXISTS verification_results_reviewer_authority_tenant_fk,
    DROP COLUMN IF EXISTS reviewer_authority_principal_id;

ALTER TABLE matter_actions
    DROP COLUMN IF EXISTS required_responsibility;

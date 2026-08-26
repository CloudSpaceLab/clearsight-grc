BEGIN;

ALTER TABLE matter_actions
    ADD COLUMN required_responsibility text NOT NULL DEFAULT 'PERFORMER'
    CHECK (required_responsibility IN (
        'PERFORMER','ACCOUNTABLE_OWNER','PROPOSER','REVIEWER','INDEPENDENT_CHALLENGER',
        'AUTHORIZER','SIGNATORY','TRANSMITTER','ACKNOWLEDGEMENT_RECORDER','ESCALATION_OWNER'
    ));

ALTER TABLE verification_results
    ADD COLUMN reviewer_authority_principal_id uuid;

UPDATE verification_results vr
SET reviewer_authority_principal_id = vc.authority_principal_id
FROM verification_contracts vc
WHERE vc.tenant_id = vr.tenant_id
  AND vc.matter_id = vr.matter_id
  AND vc.id = vr.contract_id
  AND vc.authority_principal_id IS NOT NULL;

ALTER TABLE verification_results
    ADD CONSTRAINT verification_results_reviewer_authority_tenant_fk
    FOREIGN KEY (reviewer_authority_principal_id,tenant_id) REFERENCES principals(id,tenant_id);

COMMIT;

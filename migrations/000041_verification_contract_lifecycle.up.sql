BEGIN;

ALTER TABLE verification_contracts
    ADD COLUMN supersedes_contract_id uuid;

ALTER TABLE verification_contracts
    ADD CONSTRAINT verification_contracts_supersedes_scope_fk
    FOREIGN KEY (supersedes_contract_id,tenant_id,matter_id)
    REFERENCES verification_contracts(id,tenant_id,matter_id);

CREATE UNIQUE INDEX verification_contracts_one_replacement_idx
    ON verification_contracts(tenant_id,matter_id,supersedes_contract_id)
    WHERE supersedes_contract_id IS NOT NULL;

COMMIT;

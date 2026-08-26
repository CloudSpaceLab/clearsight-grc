BEGIN;

DROP INDEX IF EXISTS verification_contracts_one_replacement_idx;

ALTER TABLE verification_contracts
    DROP CONSTRAINT IF EXISTS verification_contracts_supersedes_scope_fk,
    DROP COLUMN IF EXISTS supersedes_contract_id;

COMMIT;

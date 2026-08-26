ALTER TABLE evidence_contracts
    DROP CONSTRAINT IF EXISTS evidence_contracts_configured_by_tenant_fk,
    DROP COLUMN IF EXISTS configured_by;

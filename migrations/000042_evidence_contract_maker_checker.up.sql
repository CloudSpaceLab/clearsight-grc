ALTER TABLE evidence_contracts
    ADD COLUMN configured_by uuid,
    ADD CONSTRAINT evidence_contracts_configured_by_tenant_fk
        FOREIGN KEY (configured_by, tenant_id) REFERENCES principals(id, tenant_id);

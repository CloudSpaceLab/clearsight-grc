BEGIN;

ALTER TABLE programs
    ADD COLUMN organization_scope_id uuid;

ALTER TABLE programs
    ADD CONSTRAINT programs_organization_scope_fk
    FOREIGN KEY (tenant_id,legal_entity_id,organization_scope_id)
    REFERENCES organization_scopes(tenant_id,legal_entity_id,id);

CREATE INDEX programs_organization_scope_updated_idx
    ON programs(tenant_id,legal_entity_id,organization_scope_id,status,updated_at DESC,id DESC)
    WHERE organization_scope_id IS NOT NULL;

COMMIT;

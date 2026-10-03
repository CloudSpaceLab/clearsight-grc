BEGIN;

ALTER TABLE risks
    ADD COLUMN organization_scope_id uuid;

ALTER TABLE risks
    ADD CONSTRAINT risks_organization_scope_fk
    FOREIGN KEY (tenant_id,legal_entity_id,organization_scope_id)
    REFERENCES organization_scopes(tenant_id,legal_entity_id,id);

CREATE INDEX risks_organization_scope_updated_idx
    ON risks(tenant_id,legal_entity_id,organization_scope_id,status,updated_at DESC,id DESC)
    WHERE organization_scope_id IS NOT NULL;

COMMIT;

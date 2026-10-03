BEGIN;

ALTER TABLE matters
    ADD COLUMN organization_scope_id uuid;

ALTER TABLE matters
    ADD CONSTRAINT matters_organization_scope_fk
    FOREIGN KEY (tenant_id,legal_entity_id,organization_scope_id)
    REFERENCES organization_scopes(tenant_id,legal_entity_id,id);

CREATE INDEX matters_organization_scope_updated_idx
    ON matters(tenant_id,legal_entity_id,organization_scope_id,updated_at DESC,id DESC)
    WHERE organization_scope_id IS NOT NULL;

COMMIT;

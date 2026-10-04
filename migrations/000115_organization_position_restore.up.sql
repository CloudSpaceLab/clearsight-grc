BEGIN;

ALTER TABLE organization_position_revisions
    ADD COLUMN restored_from_revision_id uuid;

ALTER TABLE organization_position_revisions
    ADD CONSTRAINT organization_position_revisions_restore_source_fk
    FOREIGN KEY (restored_from_revision_id)
    REFERENCES organization_position_revisions(id);

CREATE INDEX organization_position_revisions_restore_source_idx
    ON organization_position_revisions(tenant_id,legal_entity_id,restored_from_revision_id)
    WHERE restored_from_revision_id IS NOT NULL;

COMMIT;

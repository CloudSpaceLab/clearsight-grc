BEGIN;

CREATE UNIQUE INDEX capture_response_revisions_rcsa_scope_idx
    ON capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id);

ALTER TABLE rcsa_cycles
    ADD COLUMN first_line_response_revision_id uuid,
    ADD CONSTRAINT rcsa_cycles_first_line_response_scope_fk
        FOREIGN KEY(first_line_response_revision_id,tenant_id,legal_entity_id,first_line_distribution_id)
        REFERENCES capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id);

COMMIT;

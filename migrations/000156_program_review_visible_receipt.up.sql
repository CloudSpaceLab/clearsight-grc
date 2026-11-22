BEGIN;
ALTER TABLE program_review_checkpoints ADD COLUMN visible_projection_version bigint
    CHECK (visible_projection_version IS NULL OR visible_projection_version > 0);
ALTER TABLE program_review_checkpoints ADD COLUMN visible_open_matter_count bigint;
ALTER TABLE program_review_checkpoints ADD CONSTRAINT program_review_visible_receipt_complete
    CHECK ((visible_projection_version IS NULL AND visible_open_matter_count IS NULL) OR
           (visible_projection_version IS NOT NULL AND visible_open_matter_count IS NOT NULL AND visible_open_matter_count >= 0));
ALTER TABLE program_review_checkpoints DROP CONSTRAINT program_review_checkpoints_tenant_id_program_id_principal_i_key;
ALTER TABLE program_review_checkpoints ADD CONSTRAINT program_review_checkpoint_visible_version_key
    UNIQUE (tenant_id, program_id, principal_id, program_version, projection_version, visible_projection_version);
COMMIT;

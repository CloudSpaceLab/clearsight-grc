BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM program_review_checkpoints WHERE visible_projection_version IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove recorded Program review visibility receipts';
    END IF;
END $$;
ALTER TABLE program_review_checkpoints DROP CONSTRAINT program_review_checkpoint_visible_version_key;
ALTER TABLE program_review_checkpoints ADD CONSTRAINT program_review_checkpoints_tenant_id_program_id_principal_i_key
    UNIQUE (tenant_id, program_id, principal_id, program_version, projection_version);
ALTER TABLE program_review_checkpoints DROP CONSTRAINT program_review_visible_receipt_complete;
ALTER TABLE program_review_checkpoints DROP COLUMN visible_open_matter_count;
ALTER TABLE program_review_checkpoints DROP COLUMN visible_projection_version;
COMMIT;

BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM program_review_checkpoints WHERE authority_source_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove recorded Program review authority receipts';
    END IF;
END $$;
DROP INDEX program_review_authorized_visible_version_key;
ALTER TABLE program_review_checkpoints ADD CONSTRAINT program_review_checkpoint_visible_version_key
    UNIQUE (tenant_id,program_id,principal_id,program_version,projection_version,visible_projection_version);
ALTER TABLE program_review_checkpoints DROP CONSTRAINT program_review_authority_receipt_complete,
    DROP COLUMN authority_policy_version, DROP COLUMN authority_rule_id,
    DROP COLUMN authority_source_id, DROP COLUMN authority_source_type;
COMMIT;

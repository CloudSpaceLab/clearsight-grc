BEGIN;
ALTER TABLE program_review_checkpoints
    ADD COLUMN authority_source_type text,
    ADD COLUMN authority_source_id uuid,
    ADD COLUMN authority_rule_id text,
    ADD COLUMN authority_policy_version text,
    ADD CONSTRAINT program_review_authority_receipt_complete CHECK (
        (authority_source_type IS NULL AND authority_source_id IS NULL AND authority_rule_id IS NULL AND authority_policy_version IS NULL)
        OR (authority_source_type IS NOT NULL AND authority_source_type IN ('ASSIGNMENT','POLICY') AND authority_source_id IS NOT NULL AND authority_rule_id IS NOT NULL AND btrim(authority_rule_id) <> '' AND authority_policy_version IS NOT NULL AND btrim(authority_policy_version) <> '' AND visible_projection_version IS NOT NULL AND visible_open_matter_count IS NOT NULL)
    );
ALTER TABLE program_review_checkpoints DROP CONSTRAINT program_review_checkpoint_visible_version_key;
CREATE UNIQUE INDEX program_review_authorized_visible_version_key
    ON program_review_checkpoints(tenant_id,program_id,principal_id,program_version,projection_version,visible_projection_version)
    WHERE authority_source_id IS NOT NULL;
COMMIT;

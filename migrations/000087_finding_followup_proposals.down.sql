BEGIN;

-- Preserve assessment-specific review/draft history. Downgrade is safe only
-- before any optional follow-up proposal has been created.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM form_template_proposals WHERE finding_assessment_id<>'') THEN
        RAISE EXCEPTION 'Follow-up proposal history exists; retain migration 000087 and roll back the application only to a compatible revision.';
    END IF;
END $$;
DROP INDEX form_template_proposals_source_revision_uq;
ALTER TABLE form_template_proposals DROP COLUMN finding_assessment_id;
CREATE UNIQUE INDEX form_template_proposals_source_revision_uq
    ON form_template_proposals(
        tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,
        source_sha256,base_template_id,base_template_version
    ) NULLS NOT DISTINCT;

COMMIT;

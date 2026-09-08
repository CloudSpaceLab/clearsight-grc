BEGIN;
-- Refuse rollback when it would require deleting independently reviewed forms.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM form_template_proposals GROUP BY tenant_id,legal_entity_id,source_kind,
        source_document_id,source_document_version,source_sha256,base_template_id,base_template_version HAVING count(*) > 1)
    THEN RAISE EXCEPTION 'Scoped form proposals exist; retain this migration to preserve their history'; END IF;
END $$;
DROP INDEX form_template_proposals_source_revision_uq;
CREATE UNIQUE INDEX form_template_proposals_source_revision_uq
    ON form_template_proposals(tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,
        source_sha256,base_template_id,base_template_version) NULLS NOT DISTINCT;
ALTER TABLE form_template_proposals DROP CONSTRAINT form_proposal_generation_scope_ck,
    DROP COLUMN generator_version, DROP COLUMN assessment_group_id;
COMMIT;

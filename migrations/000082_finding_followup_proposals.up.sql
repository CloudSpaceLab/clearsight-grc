BEGIN;
ALTER TABLE form_template_proposals
    ADD COLUMN generator_version text NOT NULL DEFAULT '',
    ADD COLUMN assessment_group_id text NOT NULL DEFAULT '',
    ADD CONSTRAINT form_proposal_generation_scope_ck CHECK (length(generator_version) <= 100 AND length(assessment_group_id) <= 100);
DROP INDEX form_template_proposals_source_revision_uq;
CREATE UNIQUE INDEX form_template_proposals_source_revision_uq
    ON form_template_proposals(tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,
        source_sha256,base_template_id,base_template_version,generator_version,assessment_group_id) NULLS NOT DISTINCT;
COMMIT;

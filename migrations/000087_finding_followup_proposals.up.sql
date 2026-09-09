BEGIN;

ALTER TABLE form_template_proposals
    ADD COLUMN finding_assessment_id text NOT NULL DEFAULT '',
    ADD CONSTRAINT form_template_proposals_finding_assessment_ck CHECK (
        char_length(finding_assessment_id) <= 128
        AND (finding_assessment_id='' OR (source_kind='DOCUMENT' AND base_template_id IS NULL))
    );

DROP INDEX form_template_proposals_source_revision_uq;
CREATE UNIQUE INDEX form_template_proposals_source_revision_uq
    ON form_template_proposals(
        tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,
        source_sha256,base_template_id,base_template_version,finding_assessment_id
    ) NULLS NOT DISTINCT;

COMMIT;

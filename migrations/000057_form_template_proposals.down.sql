BEGIN;

DROP TABLE IF EXISTS form_template_proposals;

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_tenant_entity_revision_key;

ALTER TABLE document_imports
    DROP CONSTRAINT IF EXISTS document_imports_id_tenant_entity_key,
    DROP CONSTRAINT IF EXISTS document_imports_extraction_details_ck,
    DROP COLUMN IF EXISTS extraction_details;

COMMIT;

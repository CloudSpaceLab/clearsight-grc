BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM monitoring_form_templates
        WHERE program_id IS NULL AND legal_entity_id IS NOT NULL
    ) OR EXISTS (SELECT 1 FROM form_saved_views) THEN
        RAISE EXCEPTION 'cannot roll back legal-entity Forms after legal-entity-only templates or saved views exist';
    END IF;
END;
$$;

DROP TABLE form_saved_views;

DROP INDEX monitoring_form_templates_library_search_idx;
DROP INDEX monitoring_form_templates_library_idx;
DROP INDEX monitoring_form_templates_unscoped_current_code_idx;
DROP INDEX monitoring_form_templates_entity_current_code_idx;

DROP TRIGGER monitoring_form_template_scoring_mode_default_trigger ON monitoring_form_templates;
DROP FUNCTION monitoring_form_template_scoring_mode_default();

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT monitoring_form_templates_starter_pair_ck,
    DROP CONSTRAINT monitoring_form_templates_scoring_mode_ck,
    DROP CONSTRAINT monitoring_form_templates_metadata_ck,
    DROP CONSTRAINT monitoring_form_templates_legal_entity_tenant_fk,
    DROP CONSTRAINT monitoring_form_templates_owner_tenant_fk,
    DROP CONSTRAINT monitoring_form_templates_entity_scope_ck,
    DROP COLUMN starter_catalog_version,
    DROP COLUMN starter_catalog_code,
    DROP COLUMN next_review_at,
    DROP COLUMN scoring_mode,
    DROP COLUMN sensitivity,
    DROP COLUMN industry,
    DROP COLUMN jurisdiction,
    DROP COLUMN tags,
    DROP COLUMN approved_uses,
    DROP COLUMN responsible_team,
    DROP COLUMN owner_principal_id,
    ADD CONSTRAINT monitoring_form_templates_scope_pair_ck
        CHECK ((legal_entity_id IS NULL AND program_id IS NULL) OR (legal_entity_id IS NOT NULL AND program_id IS NOT NULL));

CREATE UNIQUE INDEX monitoring_form_templates_legacy_current_code_idx
    ON monitoring_form_templates(tenant_id,code)
    WHERE is_current AND program_id IS NULL;

COMMIT;

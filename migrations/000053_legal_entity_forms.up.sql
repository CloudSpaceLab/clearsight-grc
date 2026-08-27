BEGIN;

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT monitoring_form_templates_scope_pair_ck,
    ADD COLUMN owner_principal_id uuid,
    ADD COLUMN responsible_team text NOT NULL DEFAULT '',
    ADD COLUMN approved_uses text[] NOT NULL DEFAULT '{}',
    ADD COLUMN tags text[] NOT NULL DEFAULT '{}',
    ADD COLUMN jurisdiction text NOT NULL DEFAULT '',
    ADD COLUMN industry text NOT NULL DEFAULT '',
    ADD COLUMN sensitivity text NOT NULL DEFAULT 'INTERNAL',
    ADD COLUMN scoring_mode text,
    ADD COLUMN next_review_at timestamptz,
    ADD COLUMN starter_catalog_code text,
    ADD COLUMN starter_catalog_version bigint,
    ADD CONSTRAINT monitoring_form_templates_entity_scope_ck
        CHECK (program_id IS NULL OR legal_entity_id IS NOT NULL),
    ADD CONSTRAINT monitoring_form_templates_owner_tenant_fk
        FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT monitoring_form_templates_legal_entity_tenant_fk
        FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    ADD CONSTRAINT monitoring_form_templates_metadata_ck CHECK (
        char_length(responsible_team) <= 200
        AND cardinality(approved_uses) <= 32 AND octet_length(array_to_string(approved_uses,E'\x1f')) <= 4096
        AND cardinality(tags) <= 64 AND octet_length(array_to_string(tags,E'\x1f')) <= 4096
        AND char_length(jurisdiction) <= 120 AND char_length(industry) <= 120
        AND sensitivity IN ('INTERNAL','CONFIDENTIAL','RESTRICTED')
    ),
    ADD CONSTRAINT monitoring_form_templates_scoring_mode_ck
        CHECK (scoring_mode IN ('NONE','RISK','COMPLIANCE')),
    ADD CONSTRAINT monitoring_form_templates_starter_pair_ck CHECK (
        (starter_catalog_code IS NULL AND starter_catalog_version IS NULL)
        OR (
            starter_catalog_code=btrim(starter_catalog_code)
            AND char_length(starter_catalog_code) BETWEEN 1 AND 128
            AND starter_catalog_version > 0
        )
    );

UPDATE monitoring_form_templates f
SET scoring_mode = CASE
    WHEN EXISTS (SELECT 1 FROM jsonb_array_elements(f.fields) field WHERE field ? 'scoring') THEN 'RISK'
    ELSE 'NONE'
END;

CREATE FUNCTION monitoring_form_template_scoring_mode_default()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.scoring_mode IS NULL THEN
        NEW.scoring_mode := CASE
            WHEN EXISTS (SELECT 1 FROM jsonb_array_elements(NEW.fields) field WHERE field ? 'scoring') THEN 'RISK'
            ELSE 'NONE'
        END;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER monitoring_form_template_scoring_mode_default_trigger
BEFORE INSERT OR UPDATE OF fields,scoring_mode ON monitoring_form_templates
FOR EACH ROW EXECUTE FUNCTION monitoring_form_template_scoring_mode_default();

ALTER TABLE monitoring_form_templates ALTER COLUMN scoring_mode SET NOT NULL;

DROP INDEX monitoring_form_templates_legacy_current_code_idx;
CREATE UNIQUE INDEX monitoring_form_templates_entity_current_code_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,code)
    WHERE is_current AND program_id IS NULL AND legal_entity_id IS NOT NULL;
CREATE UNIQUE INDEX monitoring_form_templates_unscoped_current_code_idx
    ON monitoring_form_templates(tenant_id,code)
    WHERE is_current AND program_id IS NULL AND legal_entity_id IS NULL;
CREATE INDEX monitoring_form_templates_library_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,updated_at DESC,id DESC,version DESC)
    WHERE legal_entity_id IS NOT NULL;
CREATE INDEX monitoring_form_templates_library_search_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,lower(name),updated_at DESC,id DESC)
    WHERE legal_entity_id IS NOT NULL;

CREATE TABLE form_saved_views (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 120),
    filter jsonb NOT NULL CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE (id,tenant_id,legal_entity_id,principal_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (principal_id,tenant_id) REFERENCES principals(id,tenant_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX form_saved_views_principal_name_uq
    ON form_saved_views(tenant_id,legal_entity_id,principal_id,lower(name));
CREATE INDEX form_saved_views_principal_updated_idx
    ON form_saved_views(tenant_id,legal_entity_id,principal_id,updated_at DESC,id DESC);

COMMIT;

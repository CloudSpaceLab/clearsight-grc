BEGIN;

DROP INDEX IF EXISTS monitoring_outbox_event_uq;
DROP TABLE IF EXISTS monitoring_events;

ALTER TABLE monitoring_checks DROP CONSTRAINT IF EXISTS monitoring_checks_form_program_fk;
DROP INDEX IF EXISTS monitoring_form_templates_program_idx;
DROP INDEX IF EXISTS monitoring_form_templates_legacy_current_code_idx;
DROP INDEX IF EXISTS monitoring_form_templates_current_code_idx;
CREATE UNIQUE INDEX monitoring_form_templates_current_code_idx
    ON monitoring_form_templates(tenant_id,code) WHERE is_current;
ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_scope_version_key,
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_program_entity_fk,
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_scope_pair_ck,
    DROP COLUMN IF EXISTS program_id,
    DROP COLUMN IF EXISTS legal_entity_id;
ALTER TABLE programs DROP CONSTRAINT IF EXISTS programs_id_tenant_entity_key;

COMMIT;

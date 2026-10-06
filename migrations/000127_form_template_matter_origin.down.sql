BEGIN;

DROP TRIGGER IF EXISTS monitoring_form_origin_immutable_trigger ON monitoring_form_templates;
DROP FUNCTION IF EXISTS enforce_monitoring_form_origin_immutable();

DROP INDEX IF EXISTS monitoring_form_templates_origin_idx;

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_origin_matter_fk,
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_origin_pair_ck,
    DROP COLUMN IF EXISTS origin_id,
    DROP COLUMN IF EXISTS origin_type;

COMMIT;

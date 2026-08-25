BEGIN;

DROP TABLE IF EXISTS capture_response_drafts;

DROP INDEX IF EXISTS capture_requests_origin_idx;
ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_origin_check,
    DROP CONSTRAINT IF EXISTS capture_requests_sections_check,
    DROP CONSTRAINT IF EXISTS capture_requests_presentation_check,
    DROP CONSTRAINT IF EXISTS capture_requests_fields_check,
    DROP COLUMN IF EXISTS origin_version,
    DROP COLUMN IF EXISTS origin_id,
    DROP COLUMN IF EXISTS origin_type,
    DROP COLUMN IF EXISTS sections,
    DROP COLUMN IF EXISTS presentation,
    ADD CONSTRAINT capture_requests_fields_check CHECK (jsonb_typeof(fields)='array' AND jsonb_array_length(fields) BETWEEN 1 AND 50);

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_sections_check,
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_presentation_check,
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_fields_check,
    DROP COLUMN IF EXISTS sections,
    DROP COLUMN IF EXISTS presentation,
    ADD CONSTRAINT monitoring_form_templates_fields_check CHECK (jsonb_typeof(fields)='array' AND jsonb_array_length(fields) BETWEEN 1 AND 50);

COMMIT;

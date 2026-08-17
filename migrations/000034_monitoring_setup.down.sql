BEGIN;

DROP INDEX IF EXISTS capture_requests_form_template_idx;
ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_form_template_fk,
    DROP CONSTRAINT IF EXISTS capture_requests_collection_period_pair,
    DROP CONSTRAINT IF EXISTS capture_requests_form_template_pair,
    DROP COLUMN IF EXISTS collection_period_end,
    DROP COLUMN IF EXISTS collection_period_start,
    DROP COLUMN IF EXISTS form_template_version,
    DROP COLUMN IF EXISTS form_template_id;

DROP TABLE IF EXISTS monitoring_results;
DROP TABLE IF EXISTS monitoring_checks;
DROP TABLE IF EXISTS monitoring_form_templates;

COMMIT;

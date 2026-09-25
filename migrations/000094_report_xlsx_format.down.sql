BEGIN;

ALTER TABLE report_definitions DROP CONSTRAINT IF EXISTS report_definitions_format_check;
ALTER TABLE report_definitions ADD CONSTRAINT report_definitions_format_check
    CHECK (format IN ('CSV','NDJSON'));

ALTER TABLE report_definition_revisions DROP CONSTRAINT IF EXISTS report_definition_revisions_format_check;
ALTER TABLE report_definition_revisions ADD CONSTRAINT report_definition_revisions_format_check
    CHECK (format IN ('CSV','NDJSON'));

ALTER TABLE report_runs DROP CONSTRAINT IF EXISTS report_runs_format_check;
ALTER TABLE report_runs ADD CONSTRAINT report_runs_format_check
    CHECK (format IN ('CSV','NDJSON'));

COMMIT;

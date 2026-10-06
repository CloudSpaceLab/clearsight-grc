BEGIN;

ALTER TABLE report_definitions DROP CONSTRAINT IF EXISTS report_definitions_dataset_check;
ALTER TABLE report_definitions ADD CONSTRAINT report_definitions_dataset_check
    CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTERS','MATTER_EXCEPTIONS','MATTER_BOARD_BRIEF','VENDORS'));
ALTER TABLE report_definition_revisions DROP CONSTRAINT IF EXISTS report_definition_revisions_dataset_check;
ALTER TABLE report_definition_revisions ADD CONSTRAINT report_definition_revisions_dataset_check
    CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTERS','MATTER_EXCEPTIONS','MATTER_BOARD_BRIEF','VENDORS'));
ALTER TABLE report_runs DROP CONSTRAINT IF EXISTS report_runs_dataset_check;
ALTER TABLE report_runs ADD CONSTRAINT report_runs_dataset_check
    CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTERS','MATTER_EXCEPTIONS','MATTER_BOARD_BRIEF','VENDORS'));

ALTER TABLE report_definitions DROP CONSTRAINT IF EXISTS report_definitions_format_check;
ALTER TABLE report_definitions ADD CONSTRAINT report_definitions_format_check CHECK (format IN ('CSV','NDJSON','XLSX','PDF'));
ALTER TABLE report_definition_revisions DROP CONSTRAINT IF EXISTS report_definition_revisions_format_check;
ALTER TABLE report_definition_revisions ADD CONSTRAINT report_definition_revisions_format_check CHECK (format IN ('CSV','NDJSON','XLSX','PDF'));
ALTER TABLE report_runs DROP CONSTRAINT IF EXISTS report_runs_format_check;
ALTER TABLE report_runs ADD CONSTRAINT report_runs_format_check CHECK (format IN ('CSV','NDJSON','XLSX','PDF'));

COMMIT;

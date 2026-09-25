BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM report_definitions WHERE dataset='VENDORS')
       OR EXISTS (SELECT 1 FROM report_definition_revisions WHERE dataset='VENDORS')
       OR EXISTS (SELECT 1 FROM report_runs WHERE dataset='VENDORS') THEN
        RAISE EXCEPTION 'vendor report history exists; refusing to remove VENDORS dataset support';
    END IF;
END
$$;

ALTER TABLE report_definitions
    DROP CONSTRAINT IF EXISTS report_definitions_dataset_check;
ALTER TABLE report_definitions
    ADD CONSTRAINT report_definitions_dataset_check
    CHECK (dataset IN (
        'PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTER_EXCEPTIONS'));

ALTER TABLE report_definition_revisions
    DROP CONSTRAINT IF EXISTS report_definition_revisions_dataset_check;
ALTER TABLE report_definition_revisions
    ADD CONSTRAINT report_definition_revisions_dataset_check
    CHECK (dataset IN (
        'PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTER_EXCEPTIONS'));

ALTER TABLE report_runs
    DROP CONSTRAINT IF EXISTS report_runs_dataset_check;
ALTER TABLE report_runs
    ADD CONSTRAINT report_runs_dataset_check
    CHECK (dataset IN (
        'PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTER_EXCEPTIONS'));

COMMIT;

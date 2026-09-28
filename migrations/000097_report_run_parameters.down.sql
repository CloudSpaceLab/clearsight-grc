BEGIN;

ALTER TABLE report_runs DROP COLUMN parameters;

COMMIT;

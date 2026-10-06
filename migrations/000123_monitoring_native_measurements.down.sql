BEGIN;

ALTER TABLE monitoring_checks
    DROP CONSTRAINT IF EXISTS monitoring_checks_measurement_check,
    DROP COLUMN measurement;

COMMIT;

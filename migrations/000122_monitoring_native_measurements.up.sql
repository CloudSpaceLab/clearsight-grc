BEGIN;

ALTER TABLE monitoring_checks
    ADD COLUMN measurement jsonb,
    ADD CONSTRAINT monitoring_checks_measurement_check
        CHECK (measurement IS NULL OR jsonb_typeof(measurement)='object');

COMMIT;

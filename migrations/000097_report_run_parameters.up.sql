BEGIN;

ALTER TABLE report_runs
    ADD COLUMN parameters jsonb NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(parameters)='object' AND octet_length(parameters::text) <= 2048);

COMMIT;

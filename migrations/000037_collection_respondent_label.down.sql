BEGIN;

ALTER TABLE monitoring_collection_cycles
    DROP COLUMN IF EXISTS latest_respondent_label;

COMMIT;

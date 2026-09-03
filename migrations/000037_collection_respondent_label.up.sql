BEGIN;

ALTER TABLE monitoring_collection_cycles
    ADD COLUMN latest_respondent_label text NOT NULL DEFAULT ''
        CHECK (latest_respondent_label=btrim(latest_respondent_label) AND char_length(latest_respondent_label) <= 256);

COMMIT;

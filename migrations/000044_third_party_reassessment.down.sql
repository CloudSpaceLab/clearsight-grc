BEGIN;

DROP INDEX IF EXISTS third_party_assessments_one_active_episode_idx;
ALTER TABLE third_party_assessments DROP COLUMN source_trigger;
ALTER TABLE third_party_assessments
    DROP CONSTRAINT third_party_assessments_review_kind_check;
ALTER TABLE third_party_assessments
    ADD CONSTRAINT third_party_assessments_review_kind_check
    CHECK (review_kind IN ('ONBOARDING'));

COMMIT;

BEGIN;

ALTER TABLE third_party_assessments
    DROP CONSTRAINT third_party_assessments_source_trigger_kind_check;
ALTER TABLE third_party_assessments
    ADD CONSTRAINT third_party_assessments_source_trigger_kind_check
    CHECK ((review_kind='ONBOARDING' AND source_trigger='INITIAL')
        OR (review_kind IN ('PERIODIC','TRIGGERED') AND source_trigger<>'INITIAL'));

COMMIT;

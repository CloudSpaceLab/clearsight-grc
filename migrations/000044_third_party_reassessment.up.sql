BEGIN;

ALTER TABLE third_party_assessments
    DROP CONSTRAINT third_party_assessments_review_kind_check;
ALTER TABLE third_party_assessments
    ADD CONSTRAINT third_party_assessments_review_kind_check
    CHECK (review_kind IN ('ONBOARDING','PERIODIC','TRIGGERED'));
ALTER TABLE third_party_assessments
    ADD COLUMN source_trigger text NOT NULL DEFAULT 'INITIAL'
    CHECK (source_trigger=btrim(source_trigger) AND source_trigger<>'' AND char_length(source_trigger) <= 128);
ALTER TABLE third_party_assessments
    ADD CONSTRAINT third_party_assessments_source_trigger_kind_check
    CHECK ((review_kind='ONBOARDING' AND source_trigger='INITIAL')
        OR (review_kind IN ('PERIODIC','TRIGGERED') AND source_trigger<>'INITIAL'));

CREATE UNIQUE INDEX third_party_assessments_one_active_episode_idx
    ON third_party_assessments(tenant_id,legal_entity_id,relationship_id)
    WHERE status NOT IN ('COMPLETED','CANCELLED');

COMMIT;

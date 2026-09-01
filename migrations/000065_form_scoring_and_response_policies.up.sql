BEGIN;

ALTER TABLE monitoring_form_templates
    ADD COLUMN score_profile jsonb,
    ADD CONSTRAINT monitoring_form_templates_score_profile_ck CHECK (
        score_profile IS NULL OR (
            jsonb_typeof(score_profile)='object'
            AND octet_length(score_profile::text) <= 262144
        )
    );

ALTER TABLE capture_requests
    ADD COLUMN scoring_mode text NOT NULL DEFAULT 'NONE',
    ADD COLUMN score_profile jsonb,
    ADD CONSTRAINT capture_requests_scoring_mode_ck CHECK (scoring_mode IN ('NONE','RISK','COMPLIANCE')),
    ADD CONSTRAINT capture_requests_score_profile_ck CHECK (
        score_profile IS NULL OR (
            jsonb_typeof(score_profile)='object'
            AND octet_length(score_profile::text) <= 262144
        )
    );

ALTER TABLE capture_response_revisions
    ADD COLUMN score_mode text,
    ADD COLUMN score_direction text,
    ADD COLUMN raw_score numeric(7,4),
    ADD COLUMN adverse_score numeric(7,4),
    ADD COLUMN concern_band text,
    ADD COLUMN score_state text NOT NULL DEFAULT 'NOT_CONFIGURED',
    ADD COLUMN score_result jsonb,
    ADD COLUMN score_profile_checksum text NOT NULL DEFAULT '',
    ADD COLUMN score_calculated_at timestamptz,
    ADD CONSTRAINT capture_response_revisions_score_mode_ck CHECK (score_mode IS NULL OR score_mode IN ('NONE','RISK','COMPLIANCE')),
    ADD CONSTRAINT capture_response_revisions_score_direction_ck CHECK (score_direction IS NULL OR score_direction IN ('HIGH_IS_POOR','LOW_IS_POOR')),
    ADD CONSTRAINT capture_response_revisions_raw_score_ck CHECK (raw_score IS NULL OR raw_score BETWEEN 0 AND 100),
    ADD CONSTRAINT capture_response_revisions_adverse_score_ck CHECK (adverse_score IS NULL OR adverse_score BETWEEN 0 AND 100),
    ADD CONSTRAINT capture_response_revisions_concern_band_ck CHECK (concern_band IS NULL OR concern_band IN ('LOW','MODERATE','HIGH','CRITICAL')),
    ADD CONSTRAINT capture_response_revisions_score_state_ck CHECK (score_state IN ('NOT_CONFIGURED','FINAL','PROVISIONAL','FAILED')),
    ADD CONSTRAINT capture_response_revisions_score_result_ck CHECK (
        score_result IS NULL OR (
            jsonb_typeof(score_result)='object'
            AND octet_length(score_result::text) <= 1048576
        )
    ),
    ADD CONSTRAINT capture_response_revisions_score_checksum_ck CHECK (
        char_length(score_profile_checksum) <= 128
    );

CREATE INDEX capture_response_revisions_current_adverse_idx
    ON capture_response_revisions(tenant_id,legal_entity_id,adverse_score DESC,created_at DESC,id DESC)
    WHERE is_current AND score_state IN ('FINAL','PROVISIONAL') AND adverse_score IS NOT NULL;

CREATE INDEX capture_response_revisions_current_raw_idx
    ON capture_response_revisions(tenant_id,legal_entity_id,raw_score DESC,created_at DESC,id DESC)
    WHERE is_current AND score_state IN ('FINAL','PROVISIONAL') AND raw_score IS NOT NULL;

COMMIT;

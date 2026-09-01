BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM monitoring_form_templates WHERE score_profile IS NOT NULL)
       OR EXISTS (SELECT 1 FROM capture_requests WHERE score_profile IS NOT NULL OR scoring_mode <> 'NONE')
       OR EXISTS (SELECT 1 FROM capture_response_revisions WHERE score_state <> 'NOT_CONFIGURED')
       OR EXISTS (SELECT 1 FROM form_response_policy_definitions)
       OR EXISTS (SELECT 1 FROM form_response_policy_simulations)
       OR EXISTS (SELECT 1 FROM form_response_policy_executions)
       OR EXISTS (SELECT 1 FROM form_response_policy_adverse_episodes)
       OR EXISTS (SELECT 1 FROM form_response_policy_events)
       OR EXISTS (SELECT 1 FROM outbox_events WHERE aggregate_type='FORM_RESPONSE_POLICY') THEN
        RAISE EXCEPTION 'cannot roll back form scoring after score profiles or calculated response scores exist';
    END IF;
END;
$$;

DROP INDEX IF EXISTS form_response_policy_outbox_uq;
DROP TABLE IF EXISTS form_response_policy_events;
DROP TABLE IF EXISTS form_response_policy_adverse_episodes;
DROP TABLE IF EXISTS form_response_policy_executions;
ALTER TABLE form_response_policy_definitions DROP CONSTRAINT IF EXISTS form_response_policy_approved_simulation_fk;
DROP TABLE IF EXISTS form_response_policy_simulations;
DROP TABLE IF EXISTS form_response_policy_definitions;

ALTER TABLE matters DROP CONSTRAINT IF EXISTS matters_id_tenant_entity_key;
ALTER TABLE capture_response_revisions DROP CONSTRAINT IF EXISTS capture_response_revisions_id_tenant_entity_key;

DROP INDEX IF EXISTS capture_response_revisions_current_raw_idx;
DROP INDEX IF EXISTS capture_response_revisions_current_adverse_idx;

ALTER TABLE capture_response_revisions
    DROP CONSTRAINT IF EXISTS capture_response_revisions_score_checksum_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_score_result_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_score_state_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_concern_band_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_adverse_score_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_raw_score_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_score_direction_ck,
    DROP CONSTRAINT IF EXISTS capture_response_revisions_score_mode_ck,
    DROP COLUMN IF EXISTS score_calculated_at,
    DROP COLUMN IF EXISTS score_profile_checksum,
    DROP COLUMN IF EXISTS score_result,
    DROP COLUMN IF EXISTS score_state,
    DROP COLUMN IF EXISTS concern_band,
    DROP COLUMN IF EXISTS adverse_score,
    DROP COLUMN IF EXISTS raw_score,
    DROP COLUMN IF EXISTS score_direction,
    DROP COLUMN IF EXISTS score_mode;

ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_score_profile_ck,
    DROP CONSTRAINT IF EXISTS capture_requests_scoring_mode_ck,
    DROP COLUMN IF EXISTS score_profile,
    DROP COLUMN IF EXISTS scoring_mode;

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT IF EXISTS monitoring_form_templates_score_profile_ck,
    DROP COLUMN IF EXISTS score_profile;

COMMIT;

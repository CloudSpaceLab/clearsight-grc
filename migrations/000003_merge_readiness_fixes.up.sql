BEGIN;

DROP INDEX IF EXISTS org_positions_active_code_idx;
CREATE UNIQUE INDEX org_positions_active_code_idx
    ON org_positions(tenant_id, legal_entity_id, code) NULLS NOT DISTINCT
    WHERE valid_until IS NULL;

ALTER TABLE compliance_signals
    ADD CONSTRAINT compliance_signal_dedupe_key_not_blank CHECK (btrim(dedupe_key) <> '');

ALTER TABLE user_onboarding_state
    ADD CONSTRAINT onboarding_current_step_nonnegative CHECK (current_step >= 0);

COMMIT;

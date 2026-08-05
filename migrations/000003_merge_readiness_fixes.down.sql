BEGIN;

ALTER TABLE user_onboarding_state DROP CONSTRAINT IF EXISTS onboarding_current_step_nonnegative;
ALTER TABLE compliance_signals DROP CONSTRAINT IF EXISTS compliance_signal_dedupe_key_not_blank;
DROP INDEX IF EXISTS org_positions_active_code_idx;
CREATE UNIQUE INDEX org_positions_active_code_idx
    ON org_positions(tenant_id, legal_entity_id, code)
    WHERE valid_until IS NULL;

COMMIT;

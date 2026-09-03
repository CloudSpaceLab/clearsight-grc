BEGIN;

DROP INDEX IF EXISTS ai_gateway_receipts_baseline_idx;
ALTER TABLE ai_gateway_decision_receipts
    DROP CONSTRAINT IF EXISTS ai_gateway_receipts_baseline_proposed_check,
    DROP CONSTRAINT IF EXISTS ai_gateway_receipts_baseline_decision_check,
    DROP CONSTRAINT IF EXISTS ai_gateway_receipts_baseline_rollout_check,
    DROP CONSTRAINT IF EXISTS ai_gateway_receipts_baseline_version_check,
    DROP CONSTRAINT IF EXISTS ai_gateway_receipts_baseline_policy_tenant_fk,
    DROP COLUMN IF EXISTS baseline_reason_codes,
    DROP COLUMN IF EXISTS baseline_proposed_action,
    DROP COLUMN IF EXISTS baseline_decision_action,
    DROP COLUMN IF EXISTS baseline_rollout_mode,
    DROP COLUMN IF EXISTS baseline_policy_version,
    DROP COLUMN IF EXISTS baseline_policy_code,
    DROP COLUMN IF EXISTS baseline_policy_id;

COMMIT;
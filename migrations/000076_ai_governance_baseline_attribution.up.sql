BEGIN;

ALTER TABLE ai_gateway_decision_receipts
    ADD COLUMN baseline_policy_id uuid,
    ADD COLUMN baseline_policy_code text NOT NULL DEFAULT '',
    ADD COLUMN baseline_policy_version bigint,
    ADD COLUMN baseline_rollout_mode text NOT NULL DEFAULT '',
    ADD COLUMN baseline_decision_action text NOT NULL DEFAULT '',
    ADD COLUMN baseline_proposed_action text NOT NULL DEFAULT '',
    ADD COLUMN baseline_reason_codes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(baseline_reason_codes)='array'),
    ADD CONSTRAINT ai_gateway_receipts_baseline_policy_tenant_fk
        FOREIGN KEY (baseline_policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    ADD CONSTRAINT ai_gateway_receipts_baseline_version_check
        CHECK((baseline_policy_id IS NULL AND baseline_policy_version IS NULL AND baseline_policy_code='') OR
              (baseline_policy_id IS NOT NULL AND baseline_policy_version > 0 AND baseline_policy_code<>'')),
    ADD CONSTRAINT ai_gateway_receipts_baseline_rollout_check
        CHECK(baseline_rollout_mode IN ('','SHADOW','ENFORCE')),
    ADD CONSTRAINT ai_gateway_receipts_baseline_decision_check
        CHECK(baseline_decision_action IN ('','ALLOW','DENY','MODIFY','ROUTE','REQUIRE_APPROVAL','SHADOW')),
    ADD CONSTRAINT ai_gateway_receipts_baseline_proposed_check
        CHECK(baseline_proposed_action IN ('','ALLOW','DENY','MODIFY','ROUTE','REQUIRE_APPROVAL','SHADOW'));

CREATE INDEX ai_gateway_receipts_baseline_idx
    ON ai_gateway_decision_receipts(tenant_id,baseline_policy_id,observed_at DESC)
    WHERE baseline_policy_id IS NOT NULL;

COMMIT;

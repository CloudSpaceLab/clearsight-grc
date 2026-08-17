BEGIN;
CREATE TABLE ai_gateway_decision_receipts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    receipt_id text NOT NULL,
    request_id text NOT NULL,
    workload_id text NOT NULL,
    policy_id uuid,
    policy_code text NOT NULL,
    policy_version bigint NOT NULL CHECK(policy_version > 0),
    decision_action text NOT NULL CHECK(decision_action IN ('ALLOW','DENY','MODIFY','ROUTE','REQUIRE_APPROVAL','SHADOW')),
    proposed_action text NOT NULL DEFAULT '',
    reason_codes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(reason_codes)='array'),
    obligations jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(obligations)='array'),
    model_alias text NOT NULL DEFAULT '',
    route_id text NOT NULL DEFAULT '',
    outcome text NOT NULL,
    error_code text NOT NULL DEFAULT '',
    observed_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    fingerprint text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id,receipt_id),
    CONSTRAINT ai_gateway_receipts_policy_tenant_fk FOREIGN KEY (policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    CHECK(expires_at > observed_at)
);
CREATE INDEX ai_gateway_receipts_tenant_observed_idx ON ai_gateway_decision_receipts(tenant_id,observed_at DESC,id DESC);
CREATE INDEX ai_gateway_receipts_episode_idx ON ai_gateway_decision_receipts(tenant_id,workload_id,policy_id,decision_action,observed_at DESC);

CREATE TABLE ai_execution_grants (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    workload_id text NOT NULL,
    matter_id uuid NOT NULL,
    decision_id uuid NOT NULL,
    action_hash text NOT NULL CHECK(length(action_hash)=64),
    approved_by uuid NOT NULL,
    state text NOT NULL DEFAULT 'ACTIVE' CHECK(state IN ('ACTIVE','USED','EXPIRED','REVOKED')),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    token_sha256 bytea NOT NULL CHECK(octet_length(token_sha256)=32),
    record_version bigint NOT NULL DEFAULT 1 CHECK(record_version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT ai_execution_grants_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT ai_execution_grants_decision_tenant_matter_fk FOREIGN KEY (decision_id,tenant_id,matter_id) REFERENCES matter_decisions(id,tenant_id,matter_id),
    CONSTRAINT ai_execution_grants_approver_tenant_fk FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE UNIQUE INDEX ai_execution_grants_token_idx ON ai_execution_grants(token_sha256);
CREATE INDEX ai_execution_grants_active_idx ON ai_execution_grants(tenant_id,workload_id,action_hash,expires_at) WHERE state='ACTIVE';
COMMIT;

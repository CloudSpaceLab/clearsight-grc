BEGIN;

ALTER TABLE automation_policies
    DROP CONSTRAINT IF EXISTS automation_policies_status_check;
ALTER TABLE automation_policies
    ADD CONSTRAINT automation_policies_status_check CHECK(status IN ('DRAFT','PENDING_APPROVAL','APPROVED','ACTIVE','SUSPENDED','EXPIRED','RETIRED')),
    ADD COLUMN IF NOT EXISTS ai_definition jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS rollout_mode text NOT NULL DEFAULT 'SHADOW' CHECK(rollout_mode IN ('SHADOW','ENFORCE')),
    ADD COLUMN IF NOT EXISTS maker_id uuid,
    ADD COLUMN IF NOT EXISTS checker_id uuid,
    ADD COLUMN IF NOT EXISTS checksum text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS submitted_at timestamptz,
    ADD COLUMN IF NOT EXISTS approved_at timestamptz,
    ADD COLUMN IF NOT EXISTS activated_at timestamptz,
    ADD COLUMN IF NOT EXISTS suspended_at timestamptz,
    ADD COLUMN IF NOT EXISTS retired_at timestamptz,
    ADD COLUMN IF NOT EXISTS record_version bigint NOT NULL DEFAULT 1 CHECK(record_version > 0);

ALTER TABLE automation_policies
    ADD CONSTRAINT automation_policies_maker_tenant_fk FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT automation_policies_checker_tenant_fk FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id);

ALTER TABLE automation_policies
    ADD CONSTRAINT automation_policies_id_tenant_unique UNIQUE(id,tenant_id);

CREATE INDEX IF NOT EXISTS automation_policies_ai_active_idx
    ON automation_policies(tenant_id, code, version DESC)
    WHERE status='ACTIVE';

CREATE TABLE ai_workloads (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    workload_id text NOT NULL,
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    name text NOT NULL,
    purpose text NOT NULL,
    environment text NOT NULL,
    owner_principal_id uuid NOT NULL,
    service_principal_id uuid,
    allowed_models jsonb NOT NULL CHECK(jsonb_typeof(allowed_models)='array'),
    requests_per_minute bigint NOT NULL CHECK(requests_per_minute > 0),
    tokens_per_minute bigint NOT NULL CHECK(tokens_per_minute > 0),
    cost_microusd_per_minute bigint NOT NULL CHECK(cost_microusd_per_minute > 0),
    max_concurrent bigint NOT NULL CHECK(max_concurrent > 0),
    verified_metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(verified_metadata)='object'),
    approved_resources jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(approved_resources)='array'),
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK(policy_version > 0),
    key_sha256 bytea NOT NULL CHECK(octet_length(key_sha256)=32),
    state text NOT NULL DEFAULT 'DRAFT' CHECK(state IN ('DRAFT','PENDING_APPROVAL','APPROVED','ACTIVE','SUSPENDED','RETIRED')),
    maker_id uuid,
    checker_id uuid,
    effective_from timestamptz,
    effective_until timestamptz,
    submitted_at timestamptz,
    approved_at timestamptz,
    activated_at timestamptz,
    suspended_at timestamptz,
    retired_at timestamptz,
    checksum text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK(version > 0),
    record_version bigint NOT NULL DEFAULT 1 CHECK(record_version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, workload_id, version),
    UNIQUE(tenant_id, code, version),
    CHECK(effective_until IS NULL OR effective_from IS NULL OR effective_until > effective_from),
    CONSTRAINT ai_workloads_policy_tenant_fk FOREIGN KEY (policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    CONSTRAINT ai_workloads_owner_tenant_fk FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT ai_workloads_service_tenant_fk FOREIGN KEY (service_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT ai_workloads_maker_tenant_fk FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT ai_workloads_checker_tenant_fk FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE UNIQUE INDEX ai_workloads_active_identity_idx ON ai_workloads(tenant_id,workload_id) WHERE state='ACTIVE';
CREATE UNIQUE INDEX ai_workloads_active_credential_idx ON ai_workloads(key_sha256) WHERE state='ACTIVE';
CREATE INDEX ai_workloads_tenant_state_idx ON ai_workloads(tenant_id,state,code,version DESC);

COMMIT;

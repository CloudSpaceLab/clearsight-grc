BEGIN;

CREATE TABLE ai_gateway_config_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    environment text NOT NULL CHECK(environment IN ('DEVELOPMENT','TEST','PRODUCTION')),
    definition jsonb NOT NULL CHECK(jsonb_typeof(definition)='object'),
    status text NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','PENDING_APPROVAL','APPROVED','ACTIVE','SUSPENDED','RETIRED')),
    maker_id uuid NOT NULL,
    checker_id uuid,
    change_reason text NOT NULL CHECK(length(change_reason) BETWEEN 1 AND 1000),
    checksum text NOT NULL CHECK(length(checksum)=64),
    submitted_at timestamptz,
    approved_at timestamptz,
    activated_at timestamptz,
    suspended_at timestamptz,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL CHECK(version > 0),
    record_version bigint NOT NULL DEFAULT 1 CHECK(record_version > 0),
    UNIQUE(tenant_id,environment,version),
    CONSTRAINT ai_gateway_config_maker_tenant_fk FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT ai_gateway_config_checker_tenant_fk FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE UNIQUE INDEX ai_gateway_config_active_idx
    ON ai_gateway_config_revisions(tenant_id,environment)
    WHERE status='ACTIVE';
CREATE INDEX ai_gateway_config_history_idx
    ON ai_gateway_config_revisions(tenant_id,environment,version DESC);

COMMIT;

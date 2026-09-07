BEGIN;

CREATE TABLE ai_gateway_emergency_controls (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    environment text NOT NULL CHECK(environment IN ('DEVELOPMENT','TEST','PRODUCTION')),
    frozen boolean NOT NULL,
    reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 1000),
    actor_id uuid NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    record_version bigint NOT NULL DEFAULT 1 CHECK(record_version > 0),
    UNIQUE(tenant_id,environment),
    CONSTRAINT ai_gateway_emergency_actor_tenant_fk FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

COMMIT;
